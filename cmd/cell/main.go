// Command cell is the Cell language CLI — parse, validate, and execute .cell files.
//
// Usage:
//
//	cell pour <file.cell>           Execute a .cell file with mock evaluator
//	cell pour <file.cell> --live    Execute with Claude API (requires ANTHROPIC_API_KEY)
//	cell parse <file.cell>          Parse and show structure (no execution)
//	cell lex <file.cell>            Tokenize and show token stream
//	cell validate <file.cell>       Parse + validate semantic constraints
//	cell dag <file.cell>            Output DOT graph of cell DAG
//	cell dag <file.cell> --json     Output JSON graph of cell DAG
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
	case "dag":
		asJSON := len(os.Args) > 3 && os.Args[3] == "--json"
		cmdDag(string(src), file, asJSON)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: cell <command> <file.cell> [flags]")
	fmt.Fprintln(os.Stderr, "Commands: lex, parse, validate, pour, dag")
	fmt.Fprintln(os.Stderr, "Flags: --live (pour), --json (dag)")
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

func cmdDag(src, filename string, asJSON bool) {
	prog, err := parser.Parse(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	prog = resolveImports(prog, filename)

	if asJSON {
		dagJSON(prog)
	} else {
		dagDOT(prog)
	}
}

type dagNode struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Kind string `json:"kind"` // "cell", "map", "reduce"
}

type dagEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"` // wire label, oracle gate
}

type dagGraph struct {
	Molecule string    `json:"molecule"`
	Nodes    []dagNode `json:"nodes"`
	Edges    []dagEdge `json:"edges"`
}

func dagJSON(prog *parser.Program) {
	var graphs []dagGraph
	for _, mol := range prog.Molecules {
		g := buildDAG(mol)
		graphs = append(graphs, g)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(graphs)
}

func dagDOT(prog *parser.Program) {
	for _, mol := range prog.Molecules {
		g := buildDAG(mol)
		fmt.Printf("digraph %q {\n", g.Molecule)
		fmt.Println("  rankdir=LR;")
		fmt.Println("  node [shape=box, style=rounded];")

		// Node styling by type
		for _, n := range g.Nodes {
			shape := "box"
			color := "black"
			switch n.Type {
			case "llm":
				color = "blue"
			case "script":
				color = "green"
				shape = "rectangle"
			case "decision":
				color = "orange"
				shape = "diamond"
			case "human":
				color = "purple"
			case "text":
				color = "gray"
			}
			style := "rounded"
			if n.Kind == "map" {
				style = "rounded,bold"
			} else if n.Kind == "reduce" {
				style = "rounded,dashed"
			}
			fmt.Printf("  %q [shape=%s, color=%s, style=%q, label=%q];\n",
				n.Name, shape, color, style, n.Name+"\n("+n.Type+")")
		}

		// Edges
		for _, e := range g.Edges {
			label := ""
			if e.Label != "" {
				label = fmt.Sprintf(" [label=%q]", e.Label)
			}
			fmt.Printf("  %q -> %q%s;\n", e.From, e.To, label)
		}
		fmt.Println("}")
	}
}

func buildDAG(mol *parser.Molecule) dagGraph {
	g := dagGraph{Molecule: mol.Name}

	// Nodes
	for _, c := range mol.Cells {
		g.Nodes = append(g.Nodes, dagNode{Name: c.Name, Type: c.Type.Name, Kind: "cell"})
	}
	for _, mc := range mol.MapCells {
		g.Nodes = append(g.Nodes, dagNode{Name: mc.Name, Type: mc.Type.Name, Kind: "map"})
	}
	for _, rc := range mol.ReduceCells {
		g.Nodes = append(g.Nodes, dagNode{Name: rc.Name, Type: rc.Type.Name, Kind: "reduce"})
	}

	// Edges from refs
	for _, c := range mol.Cells {
		for _, ref := range c.Refs {
			g.Edges = append(g.Edges, dagEdge{From: ref.Name, To: c.Name})
		}
	}
	for _, mc := range mol.MapCells {
		g.Edges = append(g.Edges, dagEdge{From: mc.OverRef, To: mc.Name, Label: "map over"})
		if mc.Body != nil {
			for _, ref := range mc.Body.Refs {
				g.Edges = append(g.Edges, dagEdge{From: ref.Name, To: mc.Name})
			}
		}
	}
	for _, rc := range mol.ReduceCells {
		if rc.OverRef != "" {
			g.Edges = append(g.Edges, dagEdge{From: rc.OverRef, To: rc.Name, Label: "reduce over"})
		} else if rc.TimesN > 0 {
			g.Edges = append(g.Edges, dagEdge{From: rc.Name, To: rc.Name, Label: fmt.Sprintf("×%d", rc.TimesN)})
		}
		if rc.Body != nil {
			for _, ref := range rc.Body.Refs {
				g.Edges = append(g.Edges, dagEdge{From: ref.Name, To: rc.Name})
			}
		}
	}

	// Edges from wires
	for _, w := range mol.Wires {
		label := ""
		if w.OracleGate != "" {
			label = "?" + w.OracleGate
		}
		g.Edges = append(g.Edges, dagEdge{From: w.From, To: w.To, Label: label})
	}

	return g
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
