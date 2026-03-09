package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pourResult is the structured output for one file.
type pourResult struct {
	Filename string
	Tokens   int
	Mols     int
	Cells    int
	Recipes  int
	Valid    bool
	Errors   []string
}

func (r pourResult) OneLine() string {
	status := "OK"
	if !r.Valid {
		status = "FAIL"
	}
	errStr := ""
	if len(r.Errors) > 0 {
		errStr = " | " + strings.Join(r.Errors, "; ")
	}
	return fmt.Sprintf("%s | tokens=%d mols=%d cells=%d recipes=%d | %s%s",
		r.Filename, r.Tokens, r.Mols, r.Cells, r.Recipes, status, errStr)
}

func pourFile(path string) pourResult {
	filename := filepath.Base(path)
	result := pourResult{Filename: filename}

	src, err := os.ReadFile(path)
	if err != nil {
		result.Errors = append(result.Errors, "read: "+err.Error())
		return result
	}

	// Lex: count tokens.
	tokens, lexErr := Lex(string(src))
	if lexErr != nil {
		result.Errors = append(result.Errors, "lex: "+lexErr.Error())
		return result
	}
	result.Tokens = len(tokens)

	// Parse.
	prog, err := Parse(string(src))
	if err != nil {
		result.Errors = append(result.Errors, "parse: "+err.Error())
		return result
	}

	// Count structures.
	result.Mols = len(prog.Molecules)
	result.Recipes = len(prog.Recipes)
	for _, mol := range prog.Molecules {
		result.Cells += len(mol.Cells) + len(mol.MapCells) + len(mol.ReduceCells)
	}

	// Validate: check semantic constraints.
	var validationErrors []string

	for _, mol := range prog.Molecules {
		if mol.Name == "" {
			validationErrors = append(validationErrors, "molecule with empty name")
		}
		cellNames := make(map[string]bool)
		for _, c := range mol.Cells {
			if cellNames[c.Name] {
				validationErrors = append(validationErrors, fmt.Sprintf("duplicate cell %q in %s", c.Name, mol.Name))
			}
			cellNames[c.Name] = true
		}
		for _, mc := range mol.MapCells {
			if cellNames[mc.Name] {
				validationErrors = append(validationErrors, fmt.Sprintf("duplicate cell %q in %s", mc.Name, mol.Name))
			}
			cellNames[mc.Name] = true
		}
		for _, rc := range mol.ReduceCells {
			if cellNames[rc.Name] {
				validationErrors = append(validationErrors, fmt.Sprintf("duplicate cell %q in %s", rc.Name, mol.Name))
			}
			cellNames[rc.Name] = true
		}

		// Check refs point to existing cells or params.
		for _, c := range mol.Cells {
			for _, ref := range c.Refs {
				refBase := strings.Split(ref.Name, ".")[0]
				if !cellNames[refBase] && !strings.HasPrefix(refBase, "param") {
					validationErrors = append(validationErrors,
						fmt.Sprintf("cell %q refs unknown %q in %s", c.Name, ref.Name, mol.Name))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		result.Errors = append(result.Errors, validationErrors...)
	}
	result.Valid = len(result.Errors) == 0

	return result
}

// Batch 1: simple files
var batch1Files = []string{
	"hello.cell", "rule-of-five.cell", "survey.cell", "security-audit.cell",
	"shiny.cell", "shiny-secure.cell", "shiny-enterprise.cell",
	"dog-backup.cell", "dog-phantom-db.cell", "town-shutdown.cell",
}

// Batch 2: medium files
var batch2Files = []string{
	"boot-triage.cell", "boot-triage-distilled.cell", "dog-doctor.cell",
	"dog-reaper.cell", "gastown-boot.cell", "session-gc.cell",
	"convoy-cleanup.cell", "convoy-feed.cell", "dep-propagate.cell",
	"digest-generate.cell", "polecat-lease.cell",
}

// Batch 3: complex files
var batch3Files = []string{
	"code-review.cell", "design.cell", "prd-review.cell", "plan-review.cell",
	"polecat-code-review.cell", "idea-to-plan.cell", "shutdown-dance.cell",
	"deacon-patrol.cell", "cell-migration.cell", "cell-reader.cell",
	"cell-zero.cell", "towers-of-hanoi.cell",
}

func TestPourBatch1(t *testing.T) {
	runBatch(t, "Batch 1 (simple)", batch1Files)
}

func TestPourBatch2(t *testing.T) {
	runBatch(t, "Batch 2 (medium)", batch2Files)
}

func TestPourBatch3(t *testing.T) {
	runBatch(t, "Batch 3 (complex)", batch3Files)
}

func TestPourAllBatches(t *testing.T) {
	all := append(append(batch1Files, batch2Files...), batch3Files...)
	runBatch(t, "All batches", all)
}

func runBatch(t *testing.T, label string, files []string) {
	t.Helper()
	examplesDir := "../../../docs/examples"

	var results []pourResult
	failures := 0

	for _, f := range files {
		path := filepath.Join(examplesDir, f)
		r := pourFile(path)
		results = append(results, r)
		if !r.Valid {
			failures++
		}
	}

	t.Logf("\n=== %s: %d files, %d failures ===", label, len(files), failures)
	for _, r := range results {
		t.Logf("  %s", r.OneLine())
	}

	// Summary stats.
	totalTokens, totalCells, totalMols := 0, 0, 0
	for _, r := range results {
		totalTokens += r.Tokens
		totalCells += r.Cells
		totalMols += r.Mols
	}
	t.Logf("  --- totals: %d tokens, %d molecules, %d cells", totalTokens, totalMols, totalCells)

	if failures > 0 {
		for _, r := range results {
			if !r.Valid {
				t.Errorf("FAIL: %s: %s", r.Filename, strings.Join(r.Errors, "; "))
			}
		}
	}
}
