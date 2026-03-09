package engine

import (
	"context"
	"os"
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

func TestFromMolecule_Hello(t *testing.T) {
	src, err := os.ReadFile("../../../docs/examples/hello.cell")
	if err != nil {
		t.Skip("hello.cell not found:", err)
	}
	prog, err := parser.Parse(string(src))
	if err != nil {
		t.Fatal("parse:", err)
	}
	if len(prog.Molecules) == 0 {
		t.Skip("no molecules")
	}
	sheet, err := FromMolecule(prog.Molecules[0])
	if err != nil {
		t.Fatal("from molecule:", err)
	}
	if len(sheet.Cells()) == 0 {
		t.Fatal("expected at least one cell")
	}
	t.Logf("Sheet %q: %d cells", sheet.Name, len(sheet.Cells()))
	for _, name := range sheet.Cells() {
		c, _ := sheet.Cell(name)
		t.Logf("  %s (type=%s, refs=%d)", name, c.Type, len(c.Refs))
	}
}

func TestFromMolecule_RunHello(t *testing.T) {
	src, err := os.ReadFile("../../../docs/examples/hello.cell")
	if err != nil {
		t.Skip("hello.cell not found:", err)
	}
	prog, err := parser.Parse(string(src))
	if err != nil {
		t.Fatal("parse:", err)
	}
	if len(prog.Molecules) == 0 {
		t.Skip("no molecules")
	}
	sheet, err := FromMolecule(prog.Molecules[0])
	if err != nil {
		t.Fatal("from molecule:", err)
	}

	eval := &MockEvaluator{Responses: map[string]string{}}
	evaluated, err := Run(context.Background(), sheet, eval)
	if err != nil {
		t.Fatal("run:", err)
	}
	t.Logf("Evaluated %d cells: %v", len(evaluated), evaluated)
	if !sheet.AllFresh() {
		for _, name := range sheet.Cells() {
			e, _ := sheet.State(name)
			if e.State != StateFresh {
				t.Logf("  %s: %s", name, e.State)
			}
		}
		t.Fatal("not all cells fresh")
	}
}

func TestFromMolecule_BootTriageDistilled(t *testing.T) {
	src, err := os.ReadFile("../../../docs/examples/boot-triage-distilled.cell")
	if err != nil {
		t.Skip("boot-triage-distilled.cell not found:", err)
	}
	prog, err := parser.Parse(string(src))
	if err != nil {
		t.Fatal("parse:", err)
	}
	if len(prog.Molecules) == 0 {
		t.Skip("no molecules")
	}
	sheet, err := FromMolecule(prog.Molecules[0])
	if err != nil {
		t.Fatal("from molecule:", err)
	}
	t.Logf("Sheet %q: %d cells", sheet.Name, len(sheet.Cells()))
	for _, name := range sheet.Cells() {
		c, _ := sheet.Cell(name)
		t.Logf("  %s (type=%s, deps=%v)", name, c.Type, c.Deps())
	}

	// Run with mock evaluator.
	eval := &MockEvaluator{Responses: map[string]string{
		"observe": "deacon_alive=false",
		"decide":  `{"action":"START","reason":"Dead session"}`,
		"act":     "Started deacon.",
		"cleanup": "Cleanup complete.",
		"exit":    "Boot triage complete: START",
	}}
	evaluated, err := Run(context.Background(), sheet, eval)
	if err != nil {
		t.Fatal("run:", err)
	}
	if len(evaluated) != 5 {
		t.Fatalf("expected 5 cells evaluated, got %d: %v", len(evaluated), evaluated)
	}
}
