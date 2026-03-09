package engine

import (
	"context"
	"testing"
)

func TestNewSheet_Basic(t *testing.T) {
	cells := []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "Based on {{a}}", Refs: []Ref{{Cell: "a"}}},
	}
	s, err := NewSheet("test", cells)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Cells()) != 2 {
		t.Fatalf("expected 2 cells, got %d", len(s.Cells()))
	}
}

func TestNewSheet_DuplicateName(t *testing.T) {
	cells := []*Cell{
		{Name: "a", Type: TypeText, Prompt: "x"},
		{Name: "a", Type: TypeText, Prompt: "y"},
	}
	_, err := NewSheet("test", cells)
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestNewSheet_UnknownRef(t *testing.T) {
	cells := []*Cell{
		{Name: "a", Type: TypeText, Prompt: "{{b}}", Refs: []Ref{{Cell: "b"}}},
	}
	_, err := NewSheet("test", cells)
	if err == nil {
		t.Fatal("expected error for unknown ref")
	}
}

func TestNewSheet_Cycle(t *testing.T) {
	cells := []*Cell{
		{Name: "a", Type: TypeText, Prompt: "{{b}}", Refs: []Ref{{Cell: "b"}}},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
	}
	_, err := NewSheet("test", cells)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
}

func TestReady_EmptyCellNoRefs(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
	})
	if !s.Ready("a") {
		t.Fatal("cell with no refs in empty state should be ready")
	}
}

func TestReady_WaitForUpstream(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
	})
	if s.Ready("b") {
		t.Fatal("b should NOT be ready when a is empty")
	}
	if !s.Ready("a") {
		t.Fatal("a should be ready")
	}
}

func TestLifecycle_EmptyToFresh(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
	})
	if err := s.BeginCompute("a"); err != nil {
		t.Fatal(err)
	}
	e, _ := s.State("a")
	if e.State != StateComputing {
		t.Fatalf("expected computing, got %s", e.State)
	}
	if err := s.Complete("a", "world"); err != nil {
		t.Fatal(err)
	}
	e, _ = s.State("a")
	if e.State != StateFresh {
		t.Fatalf("expected fresh, got %s", e.State)
	}
	if e.Value.Content != "world" {
		t.Fatalf("expected 'world', got %q", e.Value.Content)
	}
	if e.Value.Version != 1 {
		t.Fatalf("expected version 1, got %d", e.Value.Version)
	}
}

func TestStalePropagation(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
		{Name: "c", Type: TypeText, Prompt: "{{b}}", Refs: []Ref{{Cell: "b"}}},
	})

	// Evaluate all cells.
	ctx := context.Background()
	eval := &MockEvaluator{Responses: map[string]string{
		"a": "alpha", "b": "beta", "c": "gamma",
	}}
	_, err := Run(ctx, s, eval)
	if err != nil {
		t.Fatal(err)
	}
	if !s.AllFresh() {
		t.Fatal("expected all fresh after run")
	}

	// Invalidate a → b and c should become stale.
	if err := s.Invalidate("a"); err != nil {
		t.Fatal(err)
	}
	ea, _ := s.State("a")
	eb, _ := s.State("b")
	ec, _ := s.State("c")
	if ea.State != StateStale {
		t.Fatalf("a: expected stale, got %s", ea.State)
	}
	if eb.State != StateStale {
		t.Fatalf("b: expected stale, got %s", eb.State)
	}
	if ec.State != StateStale {
		t.Fatalf("c: expected stale, got %s", ec.State)
	}
}

func TestFillPrompt(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "Say: {{a}} please", Refs: []Ref{{Cell: "a"}}},
	})
	// Evaluate a first.
	_ = s.BeginCompute("a")
	_ = s.Complete("a", "world")

	prompt, err := s.FillPrompt("b")
	if err != nil {
		t.Fatal(err)
	}
	if prompt != "Say: world please" {
		t.Fatalf("expected 'Say: world please', got %q", prompt)
	}
}

func TestFillPrompt_NotFresh(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
	})
	_, err := s.FillPrompt("b")
	if err == nil {
		t.Fatal("expected error when ref is not fresh")
	}
}

func TestRun_TopologicalOrder(t *testing.T) {
	cells := []*Cell{
		{Name: "input", Type: TypeText, Prompt: "raw data"},
		{Name: "extract", Type: TypeInventory, Prompt: "Extract from: {{input}}", Refs: []Ref{{Cell: "input"}}},
		{Name: "analyze", Type: TypeSynthesis, Prompt: "Analyze: {{extract}}", Refs: []Ref{{Cell: "extract"}}},
	}
	s, err := NewSheet("pipeline", cells)
	if err != nil {
		t.Fatal(err)
	}

	eval := &MockEvaluator{Responses: map[string]string{
		"input":   "some data",
		"extract": "extracted items",
		"analyze": "analysis result",
	}}
	evaluated, err := Run(context.Background(), s, eval)
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluated) != 3 {
		t.Fatalf("expected 3 cells evaluated, got %d", len(evaluated))
	}
	// Must be in order: input → extract → analyze.
	if evaluated[0] != "input" || evaluated[1] != "extract" || evaluated[2] != "analyze" {
		t.Fatalf("wrong order: %v", evaluated)
	}
	if !s.AllFresh() {
		t.Fatal("expected all fresh")
	}
}

func TestRun_PartialEvaluation(t *testing.T) {
	// b depends on a; evaluate only a, then check state.
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
	})

	// Manually evaluate just a.
	_ = s.BeginCompute("a")
	_ = s.Complete("a", "world")

	ea, _ := s.State("a")
	eb, _ := s.State("b")
	if ea.State != StateFresh {
		t.Fatalf("a: expected fresh, got %s", ea.State)
	}
	if eb.State != StateEmpty {
		t.Fatalf("b: expected empty, got %s", eb.State)
	}
	// Now b should be ready.
	if !s.Ready("b") {
		t.Fatal("b should be ready after a is fresh")
	}
}

func TestVersionIncrement(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
	})
	eval := &MockEvaluator{Responses: map[string]string{"a": "v1"}}

	Run(context.Background(), s, eval)
	e, _ := s.State("a")
	if e.Value.Version != 1 {
		t.Fatalf("expected version 1, got %d", e.Value.Version)
	}

	// Invalidate and re-evaluate.
	_ = s.Invalidate("a")
	eval.Responses["a"] = "v2"
	Run(context.Background(), s, eval)
	e, _ = s.State("a")
	if e.Value.Version != 2 {
		t.Fatalf("expected version 2, got %d", e.Value.Version)
	}
	if e.Value.Content != "v2" {
		t.Fatalf("expected 'v2', got %q", e.Value.Content)
	}
}

func TestFail(t *testing.T) {
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "{{a}}", Refs: []Ref{{Cell: "a"}}},
	})
	_ = s.BeginCompute("a")
	_ = s.Fail("a", "LLM timeout")

	e, _ := s.State("a")
	if e.State != StateFailed {
		t.Fatalf("expected failed, got %s", e.State)
	}
	if e.Error != "LLM timeout" {
		t.Fatalf("expected error 'LLM timeout', got %q", e.Error)
	}
	// b should still be empty — not ready because a is not fresh.
	if s.Ready("b") {
		t.Fatal("b should not be ready when a is failed")
	}
}

func TestReadyCells_Parallel(t *testing.T) {
	// Two independent cells should both be ready.
	s, _ := NewSheet("test", []*Cell{
		{Name: "a", Type: TypeText, Prompt: "Hello"},
		{Name: "b", Type: TypeText, Prompt: "World"},
		{Name: "c", Type: TypeSynthesis, Prompt: "{{a}} and {{b}}", Refs: []Ref{{Cell: "a"}, {Cell: "b"}}},
	})
	ready := s.ReadyCells()
	if len(ready) != 2 {
		t.Fatalf("expected 2 ready cells, got %d: %v", len(ready), ready)
	}
}

func TestDeps(t *testing.T) {
	c := &Cell{
		Name: "x",
		Refs: []Ref{{Cell: "a"}, {Cell: "b"}, {Cell: "a"}}, // a appears twice
	}
	deps := c.Deps()
	if len(deps) != 2 {
		t.Fatalf("expected 2 unique deps, got %d: %v", len(deps), deps)
	}
}
