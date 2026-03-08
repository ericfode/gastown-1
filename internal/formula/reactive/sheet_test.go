package reactive

import (
	"fmt"
	"testing"
)

func TestInitCreatesEmptyCells(t *testing.T) {
	cells := []Cell{
		{Name: "a", Type: CellText, Prompt: "hello"},
		{Name: "b", Type: CellInventory, Prompt: "world", Refs: []Ref{{Cell: "a"}}},
	}
	s := Init("test", cells)
	if s.Name != "test" {
		t.Errorf("name = %q, want %q", s.Name, "test")
	}
	if len(s.Cells()) != 2 {
		t.Errorf("cells = %d, want 2", len(s.Cells()))
	}
	for _, c := range cells {
		st, err := s.State(c.Name)
		if err != nil {
			t.Fatal(err)
		}
		if st != StateEmpty {
			t.Errorf("cell %q state = %s, want empty", c.Name, st)
		}
	}
}

func TestReadyWithNoUpstream(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "hello"},
	})
	if !s.Ready("a") {
		t.Error("cell a should be ready (no upstream)")
	}
}

func TestReadyWithUpstreamNotFresh(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "hello"},
		{Name: "b", Prompt: "use {{a}}", Refs: []Ref{{Cell: "a"}}},
	})
	if s.Ready("b") {
		t.Error("cell b should not be ready (a is empty)")
	}
}

func TestReadyWithUpstreamFresh(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "hello"},
		{Name: "b", Prompt: "use {{a}}", Refs: []Ref{{Cell: "a"}}},
	})
	if err := s.Evaluate("a", "value-a"); err != nil {
		t.Fatal(err)
	}
	if !s.Ready("b") {
		t.Error("cell b should be ready (a is fresh)")
	}
}

func TestReadySet(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "hello"},
		{Name: "b", Prompt: "world"},
		{Name: "c", Prompt: "use {{a}} and {{b}}", Refs: []Ref{{Cell: "a"}, {Cell: "b"}}},
	})
	ready := s.ReadySet()
	if len(ready) != 2 {
		t.Fatalf("readySet = %v, want [a, b]", ready)
	}
	// After evaluating a and b, c should be ready.
	_ = s.Evaluate("a", "va")
	_ = s.Evaluate("b", "vb")
	ready = s.ReadySet()
	if len(ready) != 1 || ready[0] != "c" {
		t.Errorf("readySet = %v, want [c]", ready)
	}
}

func TestBeginComputeAndComplete(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})

	if err := s.BeginCompute("a"); err != nil {
		t.Fatal(err)
	}
	st, _ := s.State("a")
	if st != StateComputing {
		t.Errorf("state = %s, want computing", st)
	}

	if err := s.Complete("a", "result"); err != nil {
		t.Fatal(err)
	}
	st, _ = s.State("a")
	if st != StateFresh {
		t.Errorf("state = %s, want fresh", st)
	}
	v := s.GetValue("a")
	if v == nil || v.Content != "result" || v.Version != 1 {
		t.Errorf("value = %+v, want {result, 1}", v)
	}
}

func TestVersionIncrements(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})
	_ = s.Evaluate("a", "v1")
	v := s.GetValue("a")
	if v.Version != 1 {
		t.Errorf("version = %d, want 1", v.Version)
	}
	// Make stale and re-evaluate.
	s.states["a"].State = StateStale
	_ = s.Evaluate("a", "v2")
	v = s.GetValue("a")
	if v.Version != 2 {
		t.Errorf("version = %d, want 2", v.Version)
	}
}

func TestPropagateStale(t *testing.T) {
	// a -> b -> c
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}}", Refs: []Ref{{Cell: "a"}}},
		{Name: "c", Prompt: "use {{b}}", Refs: []Ref{{Cell: "b"}}},
	})
	_ = s.Evaluate("a", "va")
	_ = s.Evaluate("b", "vb")
	_ = s.Evaluate("c", "vc")
	if !s.AllFresh() {
		t.Fatal("all should be fresh")
	}

	// Simulate a change to a.
	s.PropagateStale("a")
	stB, _ := s.State("b")
	stC, _ := s.State("c")
	if stB != StateStale {
		t.Errorf("b state = %s, want stale", stB)
	}
	if stC != StateStale {
		t.Errorf("c state = %s, want stale", stC)
	}
	stA, _ := s.State("a")
	if stA != StateFresh {
		t.Errorf("a state = %s, want fresh (source of change)", stA)
	}
}

func TestEvaluateTopologicalOrder(t *testing.T) {
	// a -> b -> c (linear chain)
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}}"},
		{Name: "c", Prompt: "use {{b}}"},
	})

	// Evaluate in topological order.
	_ = s.Evaluate("a", "va")
	_ = s.Evaluate("b", "vb")
	_ = s.Evaluate("c", "vc")

	if !s.AllFresh() {
		t.Error("all cells should be fresh after topological evaluation")
	}
}

func TestEvaluateChangePropagation(t *testing.T) {
	// a -> b, a -> c
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}}"},
		{Name: "c", Prompt: "use {{a}}"},
	})

	_ = s.Evaluate("a", "v1")
	_ = s.Evaluate("b", "from-a")
	_ = s.Evaluate("c", "from-a-too")
	if !s.AllFresh() {
		t.Fatal("expected all fresh")
	}

	// Re-evaluate a with new content: b and c should become stale.
	s.states["a"].State = StateStale
	_ = s.Evaluate("a", "v2")
	stB, _ := s.State("b")
	stC, _ := s.State("c")
	if stB != StateStale {
		t.Errorf("b = %s, want stale", stB)
	}
	if stC != StateStale {
		t.Errorf("c = %s, want stale", stC)
	}
}

func TestFillPrompt(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "types", Prompt: "list types"},
		{Name: "laws", Prompt: "list laws"},
		{Name: "synth", Prompt: "Types: {{types}}\nLaws: {{laws}}\nDone."},
	})

	_ = s.Evaluate("types", "int, string, bool")
	_ = s.Evaluate("laws", "associativity, commutativity")

	got, err := s.FillPrompt("synth")
	if err != nil {
		t.Fatal(err)
	}
	want := "Types: int, string, bool\nLaws: associativity, commutativity\nDone."
	if got != want {
		t.Errorf("fillPrompt =\n%s\nwant:\n%s", got, want)
	}
}

func TestFillPromptNotFresh(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}}"},
	})
	_, err := s.FillPrompt("b")
	if err == nil {
		t.Error("expected error when upstream not fresh")
	}
}

func TestCompose(t *testing.T) {
	s1 := Init("sheet1", []Cell{
		{Name: "a", Prompt: "hello"},
		{Name: "b", Prompt: "world"},
	})
	s2 := Init("sheet2", []Cell{
		{Name: "c", Prompt: "use {{a}}"},
		{Name: "d", Prompt: "standalone"},
	})

	composed, err := s1.Compose(s2, []Connection{
		{Cell: "c", Refs: []Ref{{Cell: "b"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(composed.Cells()) != 4 {
		t.Errorf("cells = %d, want 4", len(composed.Cells()))
	}
	if composed.Name != "sheet1+sheet2" {
		t.Errorf("name = %q, want %q", composed.Name, "sheet1+sheet2")
	}

	// c should depend on both a (from template) and b (from connection).
	ups := composed.upstreamRefs("c")
	if len(ups) != 2 {
		t.Errorf("c upstreams = %v, want [a, b]", ups)
	}
}

func TestComposeDuplicateNames(t *testing.T) {
	s1 := Init("s1", []Cell{{Name: "a", Prompt: "one"}})
	s2 := Init("s2", []Cell{{Name: "a", Prompt: "two"}})
	_, err := s1.Compose(s2, nil)
	if err == nil {
		t.Error("expected error on duplicate cell names")
	}
}

func TestAllFresh(t *testing.T) {
	s := Init("test", []Cell{
		{Name: "a", Prompt: "hello"},
		{Name: "b", Prompt: "world"},
	})
	if s.AllFresh() {
		t.Error("should not be all fresh when empty")
	}
	_ = s.Evaluate("a", "va")
	if s.AllFresh() {
		t.Error("should not be all fresh (b still empty)")
	}
	_ = s.Evaluate("b", "vb")
	if !s.AllFresh() {
		t.Error("should be all fresh")
	}
}

func TestFail(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})
	_ = s.BeginCompute("a")
	if err := s.Fail("a", fmt.Errorf("boom")); err != nil {
		t.Fatal(err)
	}
	st, _ := s.State("a")
	if st != StateFailed {
		t.Errorf("state = %s, want failed", st)
	}
}

func TestShinyFormulaAsReactiveSheet(t *testing.T) {
	// Re-express the shiny.formula.toml (5 linear steps) as a reactive sheet.
	// design -> implement -> review -> test -> submit
	s := Init("shiny", []Cell{
		{Name: "design", Type: CellText, Prompt: "Design {{feature}}"},
		{Name: "implement", Type: CellCode, Prompt: "Implement based on {{design}}", Refs: []Ref{{Cell: "design"}}},
		{Name: "review", Type: CellText, Prompt: "Review {{implement}}", Refs: []Ref{{Cell: "implement"}}},
		{Name: "test", Type: CellText, Prompt: "Test {{implement}} after {{review}}", Refs: []Ref{{Cell: "review"}}},
		{Name: "submit", Type: CellText, Prompt: "Submit after {{test}}", Refs: []Ref{{Cell: "test"}}},
	})

	// Only design should be ready initially.
	ready := s.ReadySet()
	if len(ready) != 1 || ready[0] != "design" {
		t.Fatalf("initial readySet = %v, want [design]", ready)
	}

	// Walk through the chain.
	steps := []string{"design", "implement", "review", "test", "submit"}
	for i, step := range steps {
		if !s.Ready(step) {
			t.Fatalf("step %d (%s) not ready", i, step)
		}
		_ = s.Evaluate(step, "output-"+step)
	}

	if !s.AllFresh() {
		t.Error("all steps should be fresh after sequential evaluation")
	}
}

func TestTemplateRefsFromPromptOnly(t *testing.T) {
	// Cell with no explicit refs, only template refs.
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}} here"},
	})

	if s.Ready("b") {
		t.Error("b should not be ready (a is empty)")
	}
	_ = s.Evaluate("a", "va")
	if !s.Ready("b") {
		t.Error("b should be ready (a is fresh)")
	}
}

func TestBeginComputeInvalidState(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})
	_ = s.Evaluate("a", "done")
	err := s.BeginCompute("a")
	if err == nil {
		t.Error("expected error: cannot begin compute on fresh cell")
	}
}

func TestCompleteInvalidState(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})
	err := s.Complete("a", "value")
	if err == nil {
		t.Error("expected error: cannot complete an empty cell")
	}
}

func TestInputSnapshot(t *testing.T) {
	// a -> b, a -> c -> d
	s := Init("test", []Cell{
		{Name: "a", Prompt: "root"},
		{Name: "b", Prompt: "use {{a}}", Refs: []Ref{{Cell: "a"}}},
		{Name: "c", Prompt: "use {{a}}", Refs: []Ref{{Cell: "a"}}},
		{Name: "d", Prompt: "use {{b}} and {{c}}", Refs: []Ref{{Cell: "b"}, {Cell: "c"}}},
	})

	_ = s.Evaluate("a", "va")
	_ = s.Evaluate("b", "vb")
	_ = s.Evaluate("c", "vc")
	_ = s.Evaluate("d", "vd")

	snap := s.GetInputSnapshot("d")
	if snap == nil {
		t.Fatal("expected non-nil snapshot for d")
	}
	if snap["b"] != 1 || snap["c"] != 1 {
		t.Errorf("snapshot = %v, want {b:1, c:1}", snap)
	}

	// After re-evaluating a, b becomes stale and re-evaluates as v2.
	s.states["a"].State = StateStale
	_ = s.Evaluate("a", "va2")
	_ = s.Evaluate("b", "vb2")

	snap = s.GetInputSnapshot("b")
	if snap["a"] != 2 {
		t.Errorf("b snapshot = %v, want {a:2}", snap)
	}
}

func TestTotalEffect(t *testing.T) {
	cells := []Cell{
		{Name: "extract", ObservedEffect: &Effect{Tokens: 5000, Quality: QualityGood}},
		{Name: "synth", ObservedEffect: &Effect{Tokens: 12000, Quality: QualityExcellent}},
	}
	s := Init("test", cells)
	total := s.TotalEffect()

	if total.Tokens != 17000 {
		t.Errorf("total tokens = %d, want 17000", total.Tokens)
	}
	if total.Quality != QualityGood {
		t.Errorf("total quality = %s, want good", total.Quality)
	}
}

func TestUnknownCell(t *testing.T) {
	s := Init("test", []Cell{{Name: "a", Prompt: "hello"}})
	if s.Ready("nonexistent") {
		t.Error("nonexistent cell should not be ready")
	}
	_, err := s.State("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent cell")
	}
}
