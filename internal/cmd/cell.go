package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/cell/parser"
	"github.com/steveyegge/gastown/internal/style"
)

var (
	cellJSON bool
)

var cellCmd = &cobra.Command{
	Use:     "cell",
	GroupID: GroupWork,
	Short:   "Cell language tools — parse, validate, and pretty-print .cell files",
	RunE:    requireSubcommand,
	Long: `Cell language tools for parsing and validating .cell source files.

Cell is a context-free DSL for describing reactive computation graphs
where cells are agents, wires are typed data flows, and oracles gate quality.

Commands:
  parse     Parse a .cell file and report syntax errors
  validate  Parse and run semantic validation (cycles, refs, etc.)
  fmt       Parse and pretty-print a .cell file (formatter)
  ast       Dump the parsed AST as JSON

Examples:
  gt cell parse hello.cell           # Check syntax
  gt cell validate pipeline.cell     # Full validation
  gt cell fmt pipeline.cell          # Pretty-print
  gt cell ast pipeline.cell --json   # Dump AST`,
}

var cellParseCmd = &cobra.Command{
	Use:   "parse <file>",
	Short: "Parse a .cell file and report syntax errors",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellParse,
}

var cellValidateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Parse and validate a .cell file (syntax + semantics)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellValidate,
}

var cellFmtCmd = &cobra.Command{
	Use:   "fmt <file>",
	Short: "Parse and pretty-print a .cell file",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellFmt,
}

var cellASTCmd = &cobra.Command{
	Use:   "ast <file>",
	Short: "Dump the parsed AST",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellAST,
}

func init() {
	cellCmd.PersistentFlags().BoolVar(&cellJSON, "json", false, "Output as JSON")
	cellCmd.AddCommand(cellParseCmd, cellValidateCmd, cellFmtCmd, cellASTCmd)
	rootCmd.AddCommand(cellCmd)
}

func readCellFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", path, err)
	}
	return string(data), nil
}

func runCellParse(_ *cobra.Command, args []string) error {
	source, err := readCellFile(args[0])
	if err != nil {
		return err
	}

	prog, parseErr := parser.Parse(source)
	if parseErr != nil {
		if cellJSON {
			result := map[string]any{
				"file":   args[0],
				"valid":  false,
				"errors": []string{parseErr.Error()},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", style.ErrorPrefix, parseErr.Error())
		return fmt.Errorf("parse failed")
	}

	if cellJSON {
		result := map[string]any{
			"file":      args[0],
			"valid":     true,
			"molecules": len(prog.Molecules),
			"recipes":   len(prog.Recipes),
			"fragments": len(prog.Fragments),
			"oracles":   len(prog.Oracles),
			"inputs":    len(prog.Inputs),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	fmt.Printf("%s %s — parsed successfully\n", style.SuccessPrefix, args[0])
	printParseSummary(prog)
	return nil
}

func runCellValidate(_ *cobra.Command, args []string) error {
	source, err := readCellFile(args[0])
	if err != nil {
		return err
	}

	prog, parseErr := parser.Parse(source)
	if parseErr != nil {
		if cellJSON {
			result := map[string]any{
				"file":   args[0],
				"valid":  false,
				"errors": []string{parseErr.Error()},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", style.ErrorPrefix, parseErr.Error())
		return fmt.Errorf("parse failed")
	}

	validationErrors := parser.Validate(prog)

	if cellJSON {
		var errStrs []string
		var warnStrs []string
		for _, ve := range validationErrors {
			if ve.Severity == "error" {
				errStrs = append(errStrs, ve.Error())
			} else {
				warnStrs = append(warnStrs, ve.Error())
			}
		}
		result := map[string]any{
			"file":     args[0],
			"valid":    len(errStrs) == 0,
			"errors":   errStrs,
			"warnings": warnStrs,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	hasErrors := false
	for _, ve := range validationErrors {
		if ve.Severity == "error" {
			fmt.Fprintf(os.Stderr, "%s %s\n", style.ErrorPrefix, ve.Error())
			hasErrors = true
		} else {
			fmt.Fprintf(os.Stderr, "%s %s\n", style.WarningPrefix, ve.Error())
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	fmt.Printf("%s %s — valid\n", style.SuccessPrefix, args[0])
	printParseSummary(prog)
	return nil
}

func runCellFmt(_ *cobra.Command, args []string) error {
	source, err := readCellFile(args[0])
	if err != nil {
		return err
	}

	prog, parseErr := parser.Parse(source)
	if parseErr != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", style.ErrorPrefix, parseErr.Error())
		return fmt.Errorf("parse failed")
	}

	fmt.Print(parser.PrettyPrint(prog))
	return nil
}

func runCellAST(_ *cobra.Command, args []string) error {
	source, err := readCellFile(args[0])
	if err != nil {
		return err
	}

	prog, parseErr := parser.Parse(source)
	if parseErr != nil {
		if cellJSON {
			result := map[string]any{
				"file":   args[0],
				"valid":  false,
				"errors": []string{parseErr.Error()},
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}
		fmt.Fprintf(os.Stderr, "%s %s\n", style.ErrorPrefix, parseErr.Error())
		return fmt.Errorf("parse failed")
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(prog)
}

func printParseSummary(prog *parser.Program) {
	var parts []string
	if len(prog.Molecules) > 0 {
		parts = append(parts, fmt.Sprintf("%d molecule(s)", len(prog.Molecules)))
	}
	for _, mol := range prog.Molecules {
		cellCount := len(mol.Cells) + len(mol.MapCells) + len(mol.ReduceCells)
		if cellCount > 0 {
			parts = append(parts, fmt.Sprintf("  %s: %d cell(s), %d wire(s)", mol.Name, cellCount, len(mol.Wires)))
		}
	}
	if len(prog.Recipes) > 0 {
		parts = append(parts, fmt.Sprintf("%d recipe(s)", len(prog.Recipes)))
	}
	if len(prog.Fragments) > 0 {
		parts = append(parts, fmt.Sprintf("%d fragment(s)", len(prog.Fragments)))
	}
	if len(parts) > 0 {
		fmt.Println(strings.Join(parts, "\n"))
	}
}
