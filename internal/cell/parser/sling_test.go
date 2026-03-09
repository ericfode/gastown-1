package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// slingLevel describes one test level in the Cell sling progression.
type slingLevel struct {
	Level       int
	Name        string
	Description string
	Files       []string // relative to docs/examples/
	Checks      func(t *testing.T, prog *Program, valErrs []*ValidationError)
}

var slingLevels = []slingLevel{
	{
		Level:       1,
		Name:        "Trivial sling",
		Description: "hello.cell — 2 cells, no oracles",
		Files:       []string{"hello.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L1: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			if mol.Name != "hello" {
				t.Errorf("L1: expected molecule name 'hello', got %q", mol.Name)
			}
			if len(mol.Cells) != 2 {
				t.Errorf("L1: expected 2 cells, got %d", len(mol.Cells))
			}
			if len(mol.Inputs) != 1 {
				t.Errorf("L1: expected 1 input, got %d", len(mol.Inputs))
			}
		},
	},
	{
		Level:       2,
		Name:        "Simple with oracle",
		Description: "rule-of-five.cell (recipe with graph ops) and security-audit.cell (oracle blocks)",
		Files:       []string{"rule-of-five.cell", "security-audit.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			// Both files are recipes — check that they have at least one
			if prog.Recipes == nil || len(prog.Recipes) == 0 {
				t.Errorf("L2: expected at least 1 recipe, got 0")
			}
		},
	},
	{
		Level:       3,
		Name:        "Distilled cell",
		Description: "boot-triage-distilled.cell — distill> block exercises SOFT_BLOCK mode",
		Files:       []string{"boot-triage-distilled.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L3: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			// Check that the distilled cell was found
			foundDistilled := false
			for _, c := range mol.Cells {
				if c.Name == "decide" && c.Type.Name == "distilled" {
					foundDistilled = true
				}
			}
			if !foundDistilled {
				t.Errorf("L3: expected distilled cell 'decide', not found")
			}
		},
	},
	{
		Level:       4,
		Name:        "Parallel dependencies",
		Description: "deacon-patrol.cell — 26 cells, parallel groups, fan-out/fan-in",
		Files:       []string{"deacon-patrol.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L4: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			if len(mol.Cells) < 20 {
				t.Errorf("L4: expected 20+ cells, got %d", len(mol.Cells))
			}
			// Check DAG is acyclic
			for _, ve := range valErrs {
				if strings.Contains(ve.Message, "cycle") {
					t.Errorf("L4: unexpected cycle in deacon-patrol DAG: %s", ve.Message)
				}
			}
		},
	},
	{
		Level:       5,
		Name:        "Sub-molecule invocation",
		Description: "idea-to-plan.cell — mol() calls exercise sub-molecule grammar",
		Files:       []string{"idea-to-plan.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L5: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			// Check for mol() type cells
			foundMol := false
			for _, c := range mol.Cells {
				if c.Type.MolRef != "" {
					foundMol = true
					break
				}
			}
			if !foundMol {
				t.Errorf("L5: expected at least one mol() cell, none found")
			}
		},
	},
	{
		Level:       6,
		Name:        "Map/reduce cells",
		Description: "code-review.cell (convoy pattern) — map cells, prompt fragments",
		Files:       []string{"code-review.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L6: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			if len(mol.Cells) < 5 {
				t.Errorf("L6: expected many cells in code-review, got %d", len(mol.Cells))
			}
			// Check for prompt fragments
			if len(prog.Fragments) == 0 && len(mol.Fragments) == 0 {
				t.Errorf("L6: expected prompt fragments in code-review")
			}
		},
	},
	{
		Level:       7,
		Name:        "Meta cell (metacircular)",
		Description: "cell-migration.cell — meta # ... meta #/ emit Cell source",
		Files:       []string{"cell-migration.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L7: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			foundMeta := false
			for _, c := range mol.Cells {
				if c.IsMeta {
					foundMeta = true
					break
				}
			}
			if !foundMeta {
				t.Errorf("L7: expected at least one meta cell, none found")
			}
		},
	},
	{
		Level:       8,
		Name:        "The bootstrap (metacircular)",
		Description: "cell-reader.cell and cell-zero.cell parse themselves",
		Files:       []string{"cell-reader.cell", "cell-zero.cell"},
		Checks: func(t *testing.T, prog *Program, valErrs []*ValidationError) {
			if len(prog.Molecules) != 1 {
				t.Errorf("L8: expected 1 molecule, got %d", len(prog.Molecules))
				return
			}
			mol := prog.Molecules[0]
			if len(mol.Cells) < 3 {
				t.Errorf("L8: expected 3+ cells, got %d", len(mol.Cells))
			}
		},
	},
}

// TestSlingProgression runs the Cell sling test progression through all 8 levels.
func TestSlingProgression(t *testing.T) {
	examplesDir := filepath.Join("..", "..", "..", "docs", "examples")

	type fileResult struct {
		Filename   string
		Tokens     int
		Cells      int
		MapCells   int
		Recipes    int
		Fragments  int
		Wires      int
		LexErr     string
		ParseErr   string
		ValErrors  int
		ValWarns   int
		ValDetails []string
	}

	type levelResult struct {
		Level       int
		Name        string
		Description string
		Files       []fileResult
		Passed      bool
		CheckErr    string
	}

	var results []levelResult

	for _, level := range slingLevels {
		t.Run(fmt.Sprintf("L%d_%s", level.Level, strings.ReplaceAll(level.Name, " ", "_")), func(t *testing.T) {
			lr := levelResult{
				Level:       level.Level,
				Name:        level.Name,
				Description: level.Description,
				Passed:      true,
			}

			for _, filename := range level.Files {
				fr := fileResult{Filename: filename}
				path := filepath.Join(examplesDir, filename)

				data, err := os.ReadFile(path)
				if err != nil {
					fr.LexErr = fmt.Sprintf("read error: %v", err)
					lr.Files = append(lr.Files, fr)
					lr.Passed = false
					t.Errorf("cannot read %s: %v", filename, err)
					continue
				}

				// Phase 1: Lex
				tokens, lexErr := Lex(string(data))
				if lexErr != nil {
					fr.LexErr = lexErr.Error()
					lr.Files = append(lr.Files, fr)
					lr.Passed = false
					t.Errorf("%s: lex error: %v", filename, lexErr)
					continue
				}
				fr.Tokens = len(tokens)

				// Phase 2: Parse
				prog, parseErr := Parse(string(data))
				if parseErr != nil {
					fr.ParseErr = parseErr.Error()
					// Still try to gather what we can from partial parse
					if prog != nil {
						for _, mol := range prog.Molecules {
							fr.Cells += len(mol.Cells)
							fr.MapCells += len(mol.MapCells)
							fr.Wires += len(mol.Wires)
						}
						fr.Recipes = len(prog.Recipes)
						fr.Fragments = len(prog.Fragments)
					}
					lr.Files = append(lr.Files, fr)
					lr.Passed = false
					t.Errorf("%s: parse error: %v", filename, parseErr)
					continue
				}

				// Gather stats
				for _, mol := range prog.Molecules {
					fr.Cells += len(mol.Cells)
					fr.MapCells += len(mol.MapCells)
					fr.Wires += len(mol.Wires)
					fr.Fragments += len(mol.Fragments)
				}
				fr.Recipes = len(prog.Recipes)
				fr.Fragments += len(prog.Fragments)

				// Phase 3: Validate
				valErrs := Validate(prog)
				for _, ve := range valErrs {
					if ve.Severity == "error" {
						fr.ValErrors++
					} else {
						fr.ValWarns++
					}
					fr.ValDetails = append(fr.ValDetails, ve.Error())
				}

				// Phase 4: Pretty-print round-trip (should not panic)
				output := PrettyPrint(prog)
				if len(output) == 0 {
					t.Errorf("%s: pretty-print produced empty output", filename)
					lr.Passed = false
				}

				// Run level-specific checks
				if level.Checks != nil {
					level.Checks(t, prog, valErrs)
				}

				lr.Files = append(lr.Files, fr)
			}

			if t.Failed() {
				lr.Passed = false
			}
			results = append(results, lr)
		})
	}

	// Print summary table
	t.Log("\n=== CELL SLING TEST RESULTS ===")
	t.Log(fmt.Sprintf("%-6s %-30s %-8s %s", "Level", "Name", "Status", "Details"))
	t.Log(strings.Repeat("-", 80))
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		for _, f := range r.Files {
			details := fmt.Sprintf("%s: %d tokens, %d cells, %d map_cells, %d wires",
				f.Filename, f.Tokens, f.Cells, f.MapCells, f.Wires)
			if f.LexErr != "" {
				details += " | LEX_ERR: " + f.LexErr
			}
			if f.ParseErr != "" {
				details += " | PARSE_ERR: " + truncate(f.ParseErr, 80)
			}
			if f.ValErrors > 0 {
				details += fmt.Sprintf(" | %d val_errors", f.ValErrors)
			}
			if f.ValWarns > 0 {
				details += fmt.Sprintf(" | %d val_warnings", f.ValWarns)
			}
			t.Log(fmt.Sprintf("L%-5d %-30s %-8s %s", r.Level, r.Name, status, details))
		}
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// TestSlingAllExamples runs lex+parse on every .cell file in docs/examples/ as a bulk smoke test.
func TestSlingAllExamples(t *testing.T) {
	examplesDir := filepath.Join("..", "..", "..", "docs", "examples")
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Skipf("no examples directory: %v", err)
	}

	var passed, failed int
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".cell") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(examplesDir, entry.Name()))
			if err != nil {
				t.Fatalf("read error: %v", err)
			}

			_, lexErr := Lex(string(data))
			if lexErr != nil {
				t.Errorf("lex error: %v", lexErr)
				failed++
				return
			}

			prog, parseErr := Parse(string(data))
			if parseErr != nil {
				t.Errorf("parse error: %v", parseErr)
				failed++
				return
			}

			// Validate
			valErrs := Validate(prog)
			for _, ve := range valErrs {
				if ve.Severity == "error" {
					t.Logf("validation error: %s", ve.Error())
				}
			}

			// Pretty-print round trip
			output := PrettyPrint(prog)
			if len(output) == 0 {
				t.Error("pretty-print produced empty output")
				failed++
				return
			}
			passed++
		})
	}
}
