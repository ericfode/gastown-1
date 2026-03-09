package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/cell"
)

func init() {
	cellCmd.GroupID = GroupWork
	rootCmd.AddCommand(cellCmd)

	cellCmd.AddCommand(cellParseCmd)
	cellCmd.AddCommand(cellValidateCmd)
	cellCmd.AddCommand(cellFmtCmd)
}

var cellCmd = &cobra.Command{
	Use:   "cell",
	Short: "Cell language tools — parse, validate, and format .cell files",
	Long: `Cell is a DSL for defining reactive bead computation graphs.

A .cell file declares cells (computation nodes), their dependencies (refs),
oracles (output validators), and recipes (named operation sequences).

Commands:
  parse      Parse a .cell file and pretty-print the AST
  validate   Check a .cell file for errors (syntax, DAG cycles, bad refs)
  fmt        Format a .cell file to canonical style`,
	RunE: requireSubcommand,
}

var cellParseCmd = &cobra.Command{
	Use:   "parse <file.cell>",
	Short: "Parse a .cell file and pretty-print",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellParse,
}

var cellValidateCmd = &cobra.Command{
	Use:   "validate <file.cell>",
	Short: "Validate a .cell file for errors",
	Long: `Checks a .cell file for:
  - Syntax errors
  - Duplicate cell/recipe names
  - Unknown cell types
  - Unresolved ref and oracle references
  - Dependency cycles in the cell DAG
  - Prompt template reference consistency
  - Recipe operation validity`,
	Args: cobra.ExactArgs(1),
	RunE: runCellValidate,
}

var cellFmtCmd = &cobra.Command{
	Use:   "fmt <file.cell>",
	Short: "Format a .cell file to canonical style",
	Args:  cobra.ExactArgs(1),
	RunE:  runCellFmt,
}

func runCellParse(cmd *cobra.Command, args []string) error {
	path := args[0]
	f, err := cell.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	fmt.Print(cell.PrettyPrint(f))
	return nil
}

func runCellValidate(cmd *cobra.Command, args []string) error {
	path := args[0]
	f, err := cell.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ %s\n", err)
		os.Exit(1)
	}

	errs := cell.Validate(f)
	if len(errs) == 0 {
		fmt.Fprintf(os.Stderr, "✓ %s: valid (%d cells, %d recipes)\n", path, len(f.Cells), len(f.Recipes))
		return nil
	}

	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "✗ %s\n", e)
	}
	fmt.Fprintf(os.Stderr, "\n%d error(s) found\n", len(errs))
	os.Exit(1)
	return nil
}

func runCellFmt(cmd *cobra.Command, args []string) error {
	path := args[0]
	f, err := cell.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	formatted := cell.PrettyPrint(f)

	if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
		return fmt.Errorf("writing formatted output: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ formatted %s\n", path)
	return nil
}
