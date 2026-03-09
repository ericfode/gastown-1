package subzero

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// TestHarnessAllCellFiles attempts to parse and run every .cell file in docs/examples/.
func TestHarnessAllCellFiles(t *testing.T) {
	examplesDir := "../../../docs/examples"
	files, err := filepath.Glob(filepath.Join(examplesDir, "*.cell"))
	if err != nil || len(files) == 0 {
		t.Skip("no .cell files found")
	}

	type result struct {
		name   string
		status string // "pass", "parse-fail", "run-fail", "skip"
		cells  int
		detail string
	}

	var results []result

	for _, path := range files {
		name := filepath.Base(path)
		src, err := os.ReadFile(path)
		if err != nil {
			results = append(results, result{name, "skip", 0, "read error"})
			continue
		}

		prog, err := parser.Parse(string(src))
		if err != nil {
			results = append(results, result{name, "parse-fail", 0, firstLine(err.Error())})
			continue
		}
		if len(prog.Molecules) == 0 {
			results = append(results, result{name, "skip", 0, "no molecules"})
			continue
		}

		mol := prog.Molecules[0]

		runner := &Runner{
			Executor: &DispatchExecutor{
				LLM:    &MockExecutor{},
				Script: &ScriptExecutor{TimeoutSec: 5},
			},
			Params:   defaultParams(name),
			MaxCells: 50,
		}

		out, err := runner.Run(context.Background(), mol)
		if err != nil {
			if strings.Contains(err.Error(), "BLOCKED") {
				results = append(results, result{name, "skip", 0, firstLine(err.Error())})
			} else {
				results = append(results, result{name, "run-fail", 0, firstLine(err.Error())})
			}
			continue
		}

		results = append(results, result{name, "pass", len(out), ""})
	}

	// Write summary to file and t.Log
	f, _ := os.Create("/tmp/cell-subzero-harness.txt")
	defer f.Close()

	var passed, parseFail, runFail, skipped int
	for _, r := range results {
		var line string
		switch r.status {
		case "pass":
			passed++
			line = fmt.Sprintf("  PASS  %-35s %d cells", r.name, r.cells)
		case "parse-fail":
			parseFail++
			line = fmt.Sprintf("  PARSE %-35s %s", r.name, r.detail)
		case "run-fail":
			runFail++
			line = fmt.Sprintf("  FAIL  %-35s %s", r.name, r.detail)
		case "skip":
			skipped++
			line = fmt.Sprintf("  SKIP  %-35s %s", r.name, r.detail)
		}
		fmt.Fprintln(f, line)
	}
	summary := fmt.Sprintf("\n=== HARNESS: %d pass, %d parse-fail, %d run-fail, %d skip (of %d) ===",
		passed, parseFail, runFail, skipped, len(files))
	fmt.Fprintln(f, summary)
	t.Log(summary)

	// Fail the test if nothing passes
	if passed == 0 {
		t.Fatal("no .cell files passed the harness")
	}
}

func defaultParams(filename string) map[string]string {
	return map[string]string{
		"name":                  "TestUser",
		"feature":               "test-feature",
		"topic":                 "test-topic",
		"assignee":              "test-agent",
		"target_molecule":       "test-molecule",
		"pr_number":             "42",
		"repo":                  "test/repo",
		"branch":                "main",
		"source":                "test source code",
		"target":                "test target",
		"formula":               "test-formula",
		"role":                  "crew",
		"agent":                 "test-agent",
		"bead_id":               "gt-test",
		"min_runs":              "10",
		"consistency_threshold": "0.95",
	}
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx > 0 {
		return s[:idx]
	}
	return s
}
