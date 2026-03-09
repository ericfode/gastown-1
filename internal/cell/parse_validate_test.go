package cell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseAllTestdata parses every .cell file in testdata/ and validates them.
// This ensures 100% of spec examples and survey translations are valid Cell.
func TestParseAllTestdata(t *testing.T) {
	testdataDir := filepath.Join("testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	var cellFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".cell") {
			cellFiles = append(cellFiles, e.Name())
		}
	}

	if len(cellFiles) == 0 {
		t.Fatal("no .cell files found in testdata/")
	}

	t.Logf("Found %d .cell files to validate", len(cellFiles))

	parseOK := 0
	validateOK := 0

	for _, name := range cellFiles {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(testdataDir, name)
			f, err := ParseFile(path)
			if err != nil {
				t.Fatalf("PARSE FAIL: %v", err)
			}
			parseOK++

			// Structural invariants
			if len(f.Cells) == 0 && len(f.Recipes) == 0 {
				t.Fatal("file parsed but contains no cells or recipes")
			}

			// Check for duplicate names
			seen := make(map[string]bool)
			for _, c := range f.Cells {
				if seen[c.Name] {
					t.Errorf("duplicate cell name: %s", c.Name)
				}
				seen[c.Name] = true
			}
			for _, r := range f.Recipes {
				if seen[r.Name] {
					t.Errorf("duplicate recipe name: %s", r.Name)
				}
				seen[r.Name] = true
			}

			// Run full validation
			errs := Validate(f)

			// Filter to only real errors (not prompt ref warnings for param.* refs)
			var realErrs []Error
			for _, e := range errs {
				// Skip prompt ref warnings for param.* references (they're external inputs)
				if strings.Contains(e.Message, "param.") && strings.Contains(e.Message, "not in refs") {
					continue
				}
				realErrs = append(realErrs, e)
			}

			if len(realErrs) > 0 {
				for _, e := range realErrs {
					t.Errorf("VALIDATE: %s", e)
				}
			} else {
				validateOK++
			}

			// Log summary for passing files
			t.Logf("cells=%d recipes=%d", len(f.Cells), len(f.Recipes))
		})
	}

	t.Logf("Summary: %d/%d parsed, %d/%d validated", parseOK, len(cellFiles), validateOK, len(cellFiles))
}

// TestSpecExampleCoverage verifies we have test files for all major spec sections.
func TestSpecExampleCoverage(t *testing.T) {
	required := []struct {
		file    string
		section string
	}{
		{"spec-14-shiny.cell", "Section 14: Workflow (shiny)"},
		{"spec-15-code-review.cell", "Section 15: Convoy (code-review)"},
		{"spec-16-self-evolving.cell", "Section 16: Metacircular Evolution"},
		{"spec-17-compliance-gate.cell", "Section 17: Complex Oracle"},
		{"spec-18-dog-reaper.cell", "Section 18: Operational Molecule"},
		{"spec-19-composition.cell", "Section 19: Composition"},
		{"spec-20-idea-to-plan.cell", "Section 20: Sub-Molecule Invocation"},
		{"spec-21-patrol.cell", "Section 21: Wire Modifiers"},
		{"spec-04-prompt-sections.cell", "Section 4: Prompt Sections"},
		{"spec-06-combinators-map.cell", "Section 6: Combinators (map)"},
		{"spec-06-combinators-reduce.cell", "Section 6: Combinators (reduce)"},
		{"spec-09-oracles.cell", "Section 9: Oracles"},
		{"spec-10-recipes.cell", "Section 10: Graph Operations"},
	}

	for _, req := range required {
		path := filepath.Join("testdata", req.file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing test file for %s: %s", req.section, req.file)
		}
	}
}

// TestSurveyCoverage verifies we have test files for all DIRECT survey formulas.
func TestSurveyCoverage(t *testing.T) {
	directFormulas := []struct {
		file    string
		formula string
	}{
		{"spec-survey-polecat-work.cell", "mol-polecat-work"},
		{"spec-survey-town-shutdown.cell", "mol-town-shutdown"},
		{"spec-survey-dog-doctor.cell", "mol-dog-doctor"},
		{"spec-survey-session-gc.cell", "mol-session-gc"},
		{"spec-survey-phantom-db.cell", "mol-dog-phantom-db"},
		{"spec-survey-boot-triage.cell", "mol-boot-triage"},
		{"spec-survey-convoy-cleanup.cell", "mol-convoy-cleanup"},
		{"spec-survey-sync-workspace.cell", "mol-sync-workspace"},
		{"spec-survey-polecat-lease.cell", "mol-polecat-lease"},
		{"spec-survey-polecat-review-pr.cell", "mol-polecat-review-pr"},
		{"spec-survey-conflict-resolve.cell", "mol-polecat-conflict-resolve"},
		{"spec-survey-algebraic-survey.cell", "mol-algebraic-survey"},
		{"spec-survey-towers-of-hanoi.cell", "towers-of-hanoi"},
	}

	for _, df := range directFormulas {
		path := filepath.Join("testdata", df.file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing test file for DIRECT formula %s: %s", df.formula, df.file)
		}
	}
}

// TestAllCellTypesExercised verifies that the test corpus exercises all valid cell types.
func TestAllCellTypesExercised(t *testing.T) {
	testdataDir := filepath.Join("testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	typeSeen := make(map[string]bool)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cell") {
			continue
		}
		path := filepath.Join(testdataDir, e.Name())
		f, err := ParseFile(path)
		if err != nil {
			continue
		}
		for _, c := range f.Cells {
			if c.Type != "" {
				typeSeen[c.Type] = true
			}
		}
	}

	for typ := range ValidCellTypes {
		if !typeSeen[typ] {
			t.Errorf("cell type %q not exercised by any testdata file", typ)
		}
	}

	t.Logf("Cell types exercised: %v", typeSeen)
}

// TestLeafCellsAreReady verifies that cells with no outgoing refs (leaf nodes)
// have all their dependencies satisfied.
func TestLeafCellsAreReady(t *testing.T) {
	testdataDir := filepath.Join("testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".cell") {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			path := filepath.Join(testdataDir, e.Name())
			f, err := ParseFile(path)
			if err != nil {
				t.Skipf("parse error: %v", err)
			}

			cellNames := make(map[string]bool)
			for _, c := range f.Cells {
				cellNames[c.Name] = true
			}

			// Check all refs resolve to existing cells
			for _, c := range f.Cells {
				for _, ref := range c.Refs {
					if !cellNames[ref] {
						t.Errorf("cell %q refs non-existent cell %q", c.Name, ref)
					}
				}
			}

			// Check no cycles
			if err := checkCellCycles(f.Cells); err != nil {
				t.Errorf("cycle detected: %s", err.Message)
			}
		})
	}
}
