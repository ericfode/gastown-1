package engine

import (
	"context"
	"strings"
	"testing"
)

// TestAlgebraicSurvey_AsSheet re-expresses the algebraic survey formula
// (convoy formula with 6 parallel legs + synthesis) as a reactive sheet.
// This demonstrates that the engine handles real formula patterns:
// - Parallel independent cells (the survey legs)
// - Synthesis cell that depends on all legs
// - Staleness propagation when a leg is recomputed
func TestAlgebraicSurvey_AsSheet(t *testing.T) {
	cells := []*Cell{
		// Input cell: the codebase to survey
		{Name: "codebase", Type: TypeText, Prompt: "The Gas Town codebase"},

		// 6 parallel survey legs — each examines the codebase from a different angle.
		{Name: "type-landscape", Type: TypeInventory,
			Prompt: "Survey the type landscape of: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},
		{Name: "state-machines", Type: TypeDiagram,
			Prompt: "Map the state machines in: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},
		{Name: "invariants", Type: TypeLaws,
			Prompt: "Extract invariants from: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},
		{Name: "boundaries", Type: TypeBoundaries,
			Prompt: "Map system boundaries in: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},
		{Name: "data-flow", Type: TypeDiagram,
			Prompt: "Trace data flow through: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},
		{Name: "composition", Type: TypeLaws,
			Prompt: "Identify composition patterns in: {{codebase}}",
			Refs:   []Ref{{Cell: "codebase"}}},

		// Synthesis: combines all 6 legs into a final analysis.
		{Name: "synthesis", Type: TypeSynthesis,
			Prompt: "Synthesize the algebraic structure from:\n" +
				"Types: {{type-landscape}}\n" +
				"Machines: {{state-machines}}\n" +
				"Invariants: {{invariants}}\n" +
				"Boundaries: {{boundaries}}\n" +
				"Data flow: {{data-flow}}\n" +
				"Composition: {{composition}}",
			Refs: []Ref{
				{Cell: "type-landscape"}, {Cell: "state-machines"},
				{Cell: "invariants"}, {Cell: "boundaries"},
				{Cell: "data-flow"}, {Cell: "composition"},
			}},
	}

	sheet, err := NewSheet("algebraic-survey", cells)
	if err != nil {
		t.Fatal(err)
	}

	// All 6 legs should be ready in parallel (after codebase).
	ready := sheet.ReadyCells()
	if len(ready) != 1 || ready[0] != "codebase" {
		t.Fatalf("initially only codebase should be ready, got: %v", ready)
	}

	// Run the full sheet.
	eval := &MockEvaluator{Responses: map[string]string{
		"codebase":       "Gas Town: beads, formulas, polecats, rigs...",
		"type-landscape": "Types: Bead, Formula, Molecule, Cell",
		"state-machines": "States: open→in_progress→closed",
		"invariants":     "Law: monotone readiness",
		"boundaries":     "Boundary: Dolt ↔ beads ↔ CLI",
		"data-flow":      "Flow: formula→legs→synthesis→output",
		"composition":    "Pattern: convoy = parallel legs + synthesis",
		"synthesis":      "Gas Town has a reactive bead algebra",
	}}

	evaluated, err := Run(context.Background(), sheet, eval)
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluated) != 8 {
		t.Fatalf("expected 8 cells, got %d: %v", len(evaluated), evaluated)
	}
	if !sheet.AllFresh() {
		t.Fatal("not all fresh")
	}

	// Verify evaluation order: codebase first, then 6 legs (any order), then synthesis.
	if evaluated[0] != "codebase" {
		t.Fatalf("codebase should be first, got %s", evaluated[0])
	}
	if evaluated[7] != "synthesis" {
		t.Fatalf("synthesis should be last, got %s", evaluated[7])
	}

	// Now test staleness: invalidate codebase → all legs and synthesis become stale.
	_ = sheet.Invalidate("codebase")
	for _, name := range sheet.Cells() {
		e, _ := sheet.State(name)
		if e.State != StateStale {
			t.Fatalf("%s: expected stale after invalidating codebase, got %s", name, e.State)
		}
	}

	// Re-run: should evaluate all 8 cells again.
	eval.Responses["codebase"] = "Updated Gas Town: beads v2, reactive formulas..."
	eval.Responses["synthesis"] = "Gas Town v2 has a lazy reactive bead algebra"
	evaluated, err = Run(context.Background(), sheet, eval)
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluated) != 8 {
		t.Fatalf("re-run: expected 8 cells, got %d", len(evaluated))
	}

	// Verify version incremented.
	synthState, _ := sheet.State("synthesis")
	if synthState.Value.Version != 2 {
		t.Fatalf("synthesis version: expected 2, got %d", synthState.Value.Version)
	}
}

// TestPartialRecomputation demonstrates selective recomputation:
// only the stale branch re-evaluates.
func TestPartialRecomputation(t *testing.T) {
	cells := []*Cell{
		{Name: "source-a", Type: TypeText, Prompt: "Data A"},
		{Name: "source-b", Type: TypeText, Prompt: "Data B"},
		{Name: "process-a", Type: TypeText, Prompt: "Process: {{source-a}}", Refs: []Ref{{Cell: "source-a"}}},
		{Name: "process-b", Type: TypeText, Prompt: "Process: {{source-b}}", Refs: []Ref{{Cell: "source-b"}}},
		{Name: "merge", Type: TypeSynthesis,
			Prompt: "Merge: {{process-a}} and {{process-b}}",
			Refs:   []Ref{{Cell: "process-a"}, {Cell: "process-b"}}},
	}

	sheet, err := NewSheet("partial", cells)
	if err != nil {
		t.Fatal(err)
	}

	callCount := 0
	counting := &countingEvaluator{
		inner: &MockEvaluator{Responses: map[string]string{
			"source-a":  "A1", "source-b": "B1",
			"process-a": "PA1", "process-b": "PB1",
			"merge": "M1",
		}},
		count: &callCount,
	}

	// Full evaluation: 5 cells.
	Run(context.Background(), sheet, counting)
	if callCount != 5 {
		t.Fatalf("initial run: expected 5 calls, got %d", callCount)
	}

	// Invalidate only source-a → source-a, process-a, merge become stale.
	// source-b and process-b stay fresh.
	_ = sheet.Invalidate("source-a")

	sb, _ := sheet.State("source-b")
	pb, _ := sheet.State("process-b")
	if sb.State != StateFresh {
		t.Fatalf("source-b should be fresh, got %s", sb.State)
	}
	if pb.State != StateFresh {
		t.Fatalf("process-b should be fresh, got %s", pb.State)
	}

	// Re-run: only source-a, process-a, merge should re-evaluate (3 calls).
	callCount = 0
	counting.inner.(*MockEvaluator).Responses["source-a"] = "A2"
	counting.inner.(*MockEvaluator).Responses["process-a"] = "PA2"
	counting.inner.(*MockEvaluator).Responses["merge"] = "M2"

	evaluated, err := Run(context.Background(), sheet, counting)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 3 {
		t.Fatalf("partial re-run: expected 3 calls, got %d (evaluated: %v)", callCount, evaluated)
	}
	if !sheet.AllFresh() {
		t.Fatal("not all fresh after partial re-run")
	}
}

// TestPromptTemplateResolution verifies that {{ref}} placeholders are correctly
// filled with upstream values.
func TestPromptTemplateResolution(t *testing.T) {
	cells := []*Cell{
		{Name: "context", Type: TypeText, Prompt: "background info"},
		{Name: "question", Type: TypeText, Prompt: "specific question"},
		{Name: "answer", Type: TypeText,
			Prompt: "Given context:\n{{context}}\n\nAnswer:\n{{question}}",
			Refs:   []Ref{{Cell: "context"}, {Cell: "question"}}},
	}

	sheet, _ := NewSheet("template", cells)

	// Record what prompts the evaluator sees.
	var prompts []string
	recording := &recordingEvaluator{prompts: &prompts}

	Run(context.Background(), sheet, recording)

	// The answer cell should have received a fully-resolved prompt.
	if len(prompts) < 3 {
		t.Fatalf("expected 3 prompts, got %d", len(prompts))
	}
	answerPrompt := prompts[2]
	if !strings.Contains(answerPrompt, "[mock output for context]") {
		t.Fatalf("answer prompt should contain context output, got: %s", answerPrompt)
	}
	if !strings.Contains(answerPrompt, "[mock output for question]") {
		t.Fatalf("answer prompt should contain question output, got: %s", answerPrompt)
	}
}

// countingEvaluator wraps an evaluator and counts calls.
type countingEvaluator struct {
	inner Evaluator
	count *int
}

func (c *countingEvaluator) Eval(ctx context.Context, cell *Cell, prompt string) (string, error) {
	*c.count++
	return c.inner.Eval(ctx, cell, prompt)
}

// recordingEvaluator records prompts and returns mock output.
type recordingEvaluator struct {
	prompts *[]string
}

func (r *recordingEvaluator) Eval(_ context.Context, cell *Cell, prompt string) (string, error) {
	*r.prompts = append(*r.prompts, prompt)
	return "[mock output for " + cell.Name + "]", nil
}
