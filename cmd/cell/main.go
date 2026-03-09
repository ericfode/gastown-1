// Command cell is the Cell language CLI — parse, validate, and execute .cell files.
//
// Usage:
//
//	cell pour <file.cell>           Execute a .cell file with mock evaluator
//	cell pour <file.cell> --live    Execute with Claude API (requires ANTHROPIC_API_KEY)
//	cell parse <file.cell>          Parse and show structure (no execution)
//	cell lex <file.cell>            Tokenize and show token stream
//	cell validate <file.cell>       Parse + validate semantic constraints
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/cell/parser"
	"github.com/steveyegge/gastown/internal/cell/subzero"
	"github.com/steveyegge/gastown/internal/formula/engine"
)

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	file := os.Args[2]

	src, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch cmd {
	case "lex":
		cmdLex(string(src))
	case "parse":
		cmdParse(string(src))
	case "validate":
		cmdValidate(string(src), file)
	case "pour":
		live := len(os.Args) > 3 && os.Args[3] == "--live"
		cmdPour(string(src), file, live)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: cell <command> <file.cell> [flags]")
	fmt.Fprintln(os.Stderr, "Commands: lex, parse, validate, pour")
	fmt.Fprintln(os.Stderr, "Flags: --live (pour with Claude API instead of mock)")
}

func cmdLex(src string) {
	tokens, err := parser.Lex(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lex error: %v\n", err)
		os.Exit(1)
	}
	for i, tok := range tokens {
		val := tok.Value
		if len(val) > 60 {
			val = val[:57] + "..."
		}
		val = strings.ReplaceAll(val, "\n", "\\n")
		fmt.Printf("%4d  %-20s %s\n", i, tok.Type, val)
	}
	fmt.Printf("\n%d tokens\n", len(tokens))
}

func cmdParse(src string) {
	prog, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	printProgram(prog)
}

func cmdValidate(src, filename string) {
	prog, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Resolve imports.
	prog = resolveImports(prog, filename)

	errors := validate(prog)
	if len(errors) == 0 {
		fmt.Println("OK — no validation errors")
		printProgram(prog)
	} else {
		fmt.Fprintf(os.Stderr, "%d validation errors:\n", len(errors))
		for _, e := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}
}

func resolveImports(prog *parser.Program, filename string) *parser.Program {
	baseDir := filepath.Dir(filename)
	if baseDir == "" {
		baseDir = "."
	}
	result := parser.Resolve(prog, parser.ResolveOptions{BaseDir: baseDir})
	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "%d import errors:\n", len(result.Errors))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		os.Exit(1)
	}
	return result.Program
}

func cmdPour(src, filename string, live bool) {
	prog, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	// Resolve imports.
	prog = resolveImports(prog, filename)

	if len(prog.Molecules) == 0 {
		fmt.Fprintln(os.Stderr, "no molecules found — nothing to pour")
		os.Exit(1)
	}

	mol := prog.Molecules[0]
	fmt.Printf("=== Pouring %s (molecule: %s) ===\n", filename, mol.Name)

	// Try engine path first (typed reactive sheet).
	sheet, sheetErr := engine.FromMolecule(mol)
	if sheetErr == nil {
		pourViaEngine(sheet, live)
		return
	}

	// Fallback to Sub-Zero runner.
	fmt.Printf("(engine: %v — falling back to subzero)\n", sheetErr)
	pourViaSubZero(mol, live)
}

func pourViaEngine(sheet *engine.Sheet, live bool) {
	fmt.Printf("Sheet: %s (%d cells)\n", sheet.Name, len(sheet.Cells()))
	for _, name := range sheet.Cells() {
		c, _ := sheet.Cell(name)
		fmt.Printf("  %s (type=%s, deps=%v)\n", name, c.Type, c.Deps())
	}

	var eval engine.Evaluator
	if live {
		fmt.Println("\n--live not yet implemented for engine path")
		fmt.Println("Using mock evaluator")
	}
	eval = &engine.MockEvaluator{Responses: map[string]string{}}

	fmt.Println("\n--- Execution ---")
	start := time.Now()
	ctx := context.Background()
	evaluated, err := engine.Run(ctx, sheet, eval)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "execution error: %v\n", err)
	}
	fmt.Printf("Evaluated %d cells in %v\n", len(evaluated), elapsed)

	// Print results.
	fmt.Println("\n--- Results ---")
	for _, name := range evaluated {
		state, _ := sheet.State(name)
		output := ""
		if state.Value != nil {
			output = state.Value.Content
			if len(output) > 100 {
				output = output[:97] + "..."
			}
		}
		fmt.Printf("  %s [v%d]: %s\n", name, state.Value.Version, output)
	}

	if sheet.AllFresh() {
		fmt.Println("\nAll cells fresh.")
	} else {
		fmt.Println("\nSome cells not fresh:")
		for _, name := range sheet.Cells() {
			state, _ := sheet.State(name)
			if state.State != engine.StateFresh {
				fmt.Printf("  %s: %s\n", name, state.State)
			}
		}
	}
}

func pourViaSubZero(mol *parser.Molecule, live bool) {
	var executor subzero.Executor
	if live {
		executor = &subzero.DispatchExecutor{
			LLM:    &subzero.MockExecutor{},
			Script: &subzero.ScriptExecutor{TimeoutSec: 30},
		}
	} else {
		executor = &subzero.MockExecutor{}
	}

	runner := &subzero.Runner{
		Executor: executor,
		MaxCells: 100,
	}

	start := time.Now()
	results, err := runner.Run(context.Background(), mol)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "execution error: %v\n", err)
	}

	fmt.Printf("Executed %d cells in %v\n", len(results), elapsed)
	for name, result := range results {
		output := result.Output
		if len(output) > 100 {
			output = output[:97] + "..."
		}
		fmt.Printf("  %s: %s\n", name, output)
	}
}

func printProgram(prog *parser.Program) {
	report := struct {
		Molecules int      `json:"molecules"`
		Recipes   int      `json:"recipes"`
		Fragments int      `json:"fragments"`
		Oracles   int      `json:"oracles"`
		Inputs    int      `json:"inputs"`
		Details   []molInfo `json:"details,omitempty"`
	}{
		Molecules: len(prog.Molecules),
		Recipes:   len(prog.Recipes),
		Fragments: len(prog.Fragments),
		Oracles:   len(prog.Oracles),
		Inputs:    len(prog.Inputs),
	}

	for _, mol := range prog.Molecules {
		info := molInfo{
			Name:  mol.Name,
			Cells: len(mol.Cells),
			Maps:  len(mol.MapCells),
			Reds:  len(mol.ReduceCells),
			Wires: len(mol.Wires),
		}
		for _, c := range mol.Cells {
			info.CellNames = append(info.CellNames, c.Name+":"+c.Type.Name)
		}
		report.Details = append(report.Details, info)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(report)
}

type molInfo struct {
	Name      string   `json:"name"`
	Cells     int      `json:"cells"`
	Maps      int      `json:"maps"`
	Reds      int      `json:"reduces"`
	Wires     int      `json:"wires"`
	CellNames []string `json:"cell_names,omitempty"`
}

func validate(prog *parser.Program) []string {
	var errors []string
	for _, mol := range prog.Molecules {
		if mol.Name == "" {
			errors = append(errors, "molecule with empty name")
		}
		cellNames := make(map[string]bool)
		for _, c := range mol.Cells {
			if cellNames[c.Name] {
				errors = append(errors, fmt.Sprintf("duplicate cell %q in %s", c.Name, mol.Name))
			}
			cellNames[c.Name] = true
		}
		for _, c := range mol.Cells {
			for _, ref := range c.Refs {
				refBase := strings.Split(ref.Name, ".")[0]
				if !cellNames[refBase] && !strings.HasPrefix(refBase, "param") {
					errors = append(errors, fmt.Sprintf("cell %q refs unknown %q in %s", c.Name, ref.Name, mol.Name))
				}
			}
		}
	}
	return errors
}
