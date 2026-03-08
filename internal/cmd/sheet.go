package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/formula/reactive"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	sheetJSON bool
)

var sheetCmd = &cobra.Command{
	Use:     "sheet",
	GroupID: GroupWork,
	Short:   "Gas City reactive spreadsheet engine",
	RunE:    requireSubcommand,
	Long: `Gas City reactive spreadsheet — typed cells with staleness propagation.

A sheet is a DAG of cells connected by {{ref}} template references.
Each cell has a prompt template, a state (empty/stale/computing/fresh/failed),
and a computed value. When an upstream cell changes, downstream cells go stale.

Commands:
  status    Show cell states and values for a sheet
  ready     Show cells ready for evaluation
  fill      Fill a cell's prompt template with upstream values
  eval      Set a cell's value (simulates LLM completion)
  stale     Propagate staleness from a changed cell
  dag       Show the dependency DAG

Sheet definitions are YAML files:
  name: my-analysis
  cells:
    - name: extract
      type: inventory
      prompt: "Read the codebase and list all types."
    - name: synthesize
      type: synthesis
      prompt: "Given {{extract}}, what algebra?"
      refs: [extract]

Examples:
  gt sheet status formula.yaml          # Show all cell states
  gt sheet ready formula.yaml           # Show cells ready to eval
  gt sheet fill formula.yaml synthesize # Fill prompt template
  gt sheet eval formula.yaml extract "Found 5 types"
  gt sheet stale formula.yaml extract   # Mark downstream stale
  gt sheet dag formula.yaml             # Show dependency graph`,
}

func init() {
	sheetCmd.PersistentFlags().BoolVar(&sheetJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(sheetCmd)

	sheetCmd.AddCommand(sheetStatusCmd)
	sheetCmd.AddCommand(sheetReadyCmd)
	sheetCmd.AddCommand(sheetFillCmd)
	sheetCmd.AddCommand(sheetEvalCmd)
	sheetCmd.AddCommand(sheetStaleCmd)
	sheetCmd.AddCommand(sheetDagCmd)
}

// loadSheet reads a YAML sheet definition and returns a Sheet.
func loadSheet(path string) (*reactive.Sheet, error) {
	// Try the path as-is first (relative to CWD or absolute).
	if _, err := os.Stat(path); err != nil && !filepath.IsAbs(path) {
		// Fall back to workspace root.
		if root, err := workspace.FindFromCwd(); err == nil {
			candidate := filepath.Join(root, path)
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sheet: %w", err)
	}
	return reactive.ParseYAML(data)
}

// ── gt sheet status ─────────────────────────────────────────

var sheetStatusCmd = &cobra.Command{
	Use:   "status <sheet.yaml>",
	Short: "Show cell states and values",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetStatus,
}

type cellStatusJSON struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	State   string `json:"state"`
	Value   string `json:"value,omitempty"`
	Version int    `json:"version,omitempty"`
	Ready   bool   `json:"ready"`
	Refs    int    `json:"refs"`
}

func runSheetStatus(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	cells := sheet.Cells()
	if sheetJSON {
		var out []cellStatusJSON
		for _, c := range cells {
			st, _ := sheet.State(c.Name)
			v := sheet.GetValue(c.Name)
			entry := cellStatusJSON{
				Name:  c.Name,
				Type:  c.Type.String(),
				State: st.String(),
				Ready: sheet.Ready(c.Name),
				Refs:  len(c.Refs),
			}
			if v != nil {
				entry.Value = v.Content
				entry.Version = v.Version
			}
			out = append(out, entry)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// Human-readable output.
	fmt.Printf("%s: %s\n\n", style.Bold.Render("Sheet"), sheet.Name)

	stateColors := map[string]string{
		"fresh":     "\033[32m", // green
		"stale":     "\033[33m", // amber
		"computing": "\033[34m", // blue
		"empty":     "\033[90m", // gray
		"failed":    "\033[31m", // red
	}
	reset := "\033[0m"

	for _, c := range cells {
		st, _ := sheet.State(c.Name)
		color := stateColors[st.String()]
		ready := ""
		if sheet.Ready(c.Name) {
			ready = " ◉"
		}

		fmt.Printf("  %s%-10s%s %-12s %s%s\n",
			color, st.String(), reset,
			c.Name, c.Type.String(), ready)

		if v := sheet.GetValue(c.Name); v != nil {
			preview := v.Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Printf("             → v%d: %s\n", v.Version, preview)
		}
	}

	nReady := len(sheet.ReadySet())
	fmt.Printf("\n%s/%d cells ready", style.Bold.Render(fmt.Sprintf("%d", nReady)), len(cells))
	if sheet.AllFresh() {
		fmt.Print(" (all fresh ✓)")
	}
	fmt.Println()
	return nil
}

// ── gt sheet ready ──────────────────────────────────────────

var sheetReadyCmd = &cobra.Command{
	Use:   "ready <sheet.yaml>",
	Short: "Show cells ready for evaluation",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetReady,
}

func runSheetReady(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	ready := sheet.ReadySet()
	if sheetJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(ready)
	}

	if len(ready) == 0 {
		fmt.Println("No cells ready for evaluation.")
		return nil
	}
	fmt.Printf("%s:\n", style.Bold.Render("Ready cells"))
	for _, name := range ready {
		fmt.Printf("  ◉ %s\n", name)
	}
	return nil
}

// ── gt sheet fill ───────────────────────────────────────────

var sheetFillCmd = &cobra.Command{
	Use:   "fill <sheet.yaml> <cell>",
	Short: "Fill a cell's prompt template with upstream values",
	Args:  cobra.ExactArgs(2),
	RunE:  runSheetFill,
}

func runSheetFill(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	prompt, err := sheet.FillPrompt(args[1])
	if err != nil {
		return err
	}

	if sheetJSON {
		out := map[string]string{"cell": args[1], "prompt": prompt}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println(prompt)
	return nil
}

// ── gt sheet eval ───────────────────────────────────────────

var sheetEvalCmd = &cobra.Command{
	Use:   "eval <sheet.yaml> <cell> <value>",
	Short: "Set a cell's value (evaluate)",
	Long: `Evaluate a cell by providing its computed value.

This simulates an LLM completing the cell's prompt. The cell transitions
to fresh and downstream dependents are marked stale.

Examples:
  gt sheet eval formula.yaml extract "Found 5 types: Graph, DAG, Cell"
  gt sheet eval formula.yaml synthesize "This forms a typed reactive calculus"`,
	Args: cobra.ExactArgs(3),
	RunE: runSheetEval,
}

func runSheetEval(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	cellName := args[1]
	content := args[2]

	if err := sheet.Evaluate(cellName, content); err != nil {
		return err
	}

	if sheetJSON {
		result := map[string]interface{}{
			"cell":    cellName,
			"state":   "fresh",
			"version": sheet.GetValue(cellName).Version,
			"stale":   sheet.ReadySet(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	v := sheet.GetValue(cellName)
	fmt.Printf("✓ %s → fresh (v%d)\n", cellName, v.Version)

	// Show newly stale cells.
	for _, c := range sheet.Cells() {
		if c.Name == cellName {
			continue
		}
		st, _ := sheet.State(c.Name)
		if st == reactive.StateStale {
			fmt.Printf("  ⚠ %s → stale\n", c.Name)
		}
	}
	return nil
}

// ── gt sheet stale ──────────────────────────────────────────

var sheetStaleCmd = &cobra.Command{
	Use:   "stale <sheet.yaml> <cell>",
	Short: "Propagate staleness from a changed cell",
	Args:  cobra.ExactArgs(2),
	RunE:  runSheetStale,
}

func runSheetStale(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	sheet.PropagateStale(args[1])

	if sheetJSON {
		var stale []string
		for _, c := range sheet.Cells() {
			st, _ := sheet.State(c.Name)
			if st == reactive.StateStale {
				stale = append(stale, c.Name)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(stale)
	}

	fmt.Printf("Staleness propagated from %s:\n", args[1])
	for _, c := range sheet.Cells() {
		st, _ := sheet.State(c.Name)
		if st == reactive.StateStale {
			fmt.Printf("  ⚠ %s → stale\n", c.Name)
		}
	}
	return nil
}

// ── gt sheet dag ────────────────────────────────────────────

var sheetDagCmd = &cobra.Command{
	Use:   "dag <sheet.yaml>",
	Short: "Show the dependency DAG",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetDag,
}

type dagEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func runSheetDag(cmd *cobra.Command, args []string) error {
	sheet, err := loadSheet(args[0])
	if err != nil {
		return err
	}

	cells := sheet.Cells()

	if sheetJSON {
		var edges []dagEdge
		for _, c := range cells {
			for _, r := range c.Refs {
				edges = append(edges, dagEdge{From: r.Cell, To: c.Name})
			}
		}
		out := map[string]interface{}{
			"name":  sheet.Name,
			"cells": len(cells),
			"edges": edges,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	// ASCII DAG.
	fmt.Printf("%s: %s (%d cells)\n\n", style.Bold.Render("DAG"), sheet.Name, len(cells))

	for _, c := range cells {
		st, _ := sheet.State(c.Name)
		deps := make([]string, 0, len(c.Refs))
		for _, r := range c.Refs {
			deps = append(deps, r.Cell)
		}

		if len(deps) == 0 {
			fmt.Printf("  [%s] %s (%s)\n", st, c.Name, c.Type)
		} else {
			fmt.Printf("  [%s] %s (%s) ← %s\n", st, c.Name, c.Type,
				strings.Join(deps, ", "))
		}
	}
	return nil
}
