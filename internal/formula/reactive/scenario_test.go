package reactive

import (
	"os"
	"path/filepath"
	"testing"
)

// loadTestSheet reads a YAML file from the testdata dir relative to test-sheets.
func loadTestSheet(t *testing.T, filename string) *Sheet {
	t.Helper()
	// Try multiple paths to find the test sheets.
	paths := []string{
		filepath.Join("testdata", filename),
		filepath.Join("..", "..", "..", "..", "docs", "design", "mockups", "test-sheets", filename),
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s, err := ParseYAML(data)
		if err != nil {
			t.Fatalf("parse %s: %v", filename, err)
		}
		return s
	}
	t.Fatalf("could not find test sheet %s", filename)
	return nil
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 1: Linear Chain — Sequential Evaluation
//
// Theory: A → B → C → D. Only the head is ready. After evaluating
// each cell in order, the next becomes ready. After the full chain,
// all cells are fresh.
// ═══════════════════════════════════════════════════════════════

func TestScenarioLinearChain(t *testing.T) {
	s := loadTestSheet(t, "linear-chain.yaml")

	// PREDICTION: Only source is ready initially.
	ready := s.ReadySet()
	assertReadyExactly(t, ready, []string{"source"})

	// PREDICTION: After evaluating source, only extract becomes ready.
	mustEval(t, s, "source", "Raw data: struct Graph { nodes: []Node, edges: []Edge }")
	assertReadyExactly(t, s.ReadySet(), []string{"extract"})

	// PREDICTION: After evaluating extract, only analyze becomes ready.
	mustEval(t, s, "extract", "Found 3 types: Graph, Node, Edge")
	assertReadyExactly(t, s.ReadySet(), []string{"analyze"})

	// PREDICTION: After evaluating analyze, only decide becomes ready.
	mustEval(t, s, "analyze", "Graph forms a directed acyclic structure with typed edges")
	assertReadyExactly(t, s.ReadySet(), []string{"decide"})

	// PREDICTION: After evaluating decide, no cells are ready (all fresh).
	mustEval(t, s, "decide", "Proceed: the DAG structure is well-founded")
	assertReadyExactly(t, s.ReadySet(), nil)

	// PREDICTION: All cells are now fresh.
	if !s.AllFresh() {
		t.Error("expected all cells fresh after full chain evaluation")
	}

	// STALENESS TEST: Re-evaluating source should make extract, analyze,
	// decide all stale (transitive propagation through the full chain).
	s.states["source"].State = StateStale
	mustEval(t, s, "source", "Updated raw data: added struct Wire")

	assertState(t, s, "source", StateFresh)
	assertState(t, s, "extract", StateStale)
	assertState(t, s, "analyze", StateStale)
	assertState(t, s, "decide", StateStale)

	// PREDICTION: Only extract is ready (source is fresh, extract is stale).
	assertReadyExactly(t, s.ReadySet(), []string{"extract"})

	// PREDICTION: Input snapshot for source's v2 should be empty (no upstream).
	snap := s.GetInputSnapshot("source")
	if len(snap) != 0 {
		t.Errorf("source input snapshot = %v, want empty (no upstream)", snap)
	}
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 2: Diamond — Fan-out then Fan-in
//
// Theory: source → types, source → tests, types+tests → coverage.
// After evaluating source, BOTH types and tests become ready
// simultaneously. coverage only becomes ready after BOTH are fresh.
// Evaluating only one branch leaves coverage not-ready.
// ═══════════════════════════════════════════════════════════════

func TestScenarioDiamond(t *testing.T) {
	s := loadTestSheet(t, "diamond.yaml")

	// PREDICTION: Only source is ready.
	assertReadyExactly(t, s.ReadySet(), []string{"source"})

	// Evaluate source.
	mustEval(t, s, "source", "package main\n\ntype Foo struct{}\ntype Bar struct{}")

	// PREDICTION: Both types and tests are now ready (parallel fan-out).
	ready := s.ReadySet()
	assertReadyContains(t, ready, "types")
	assertReadyContains(t, ready, "tests")
	assertReadyDoesNotContain(t, ready, "coverage")
	if len(ready) != 2 {
		t.Errorf("ready set size = %d, want 2 (types, tests)", len(ready))
	}

	// Evaluate only types. coverage should NOT be ready (tests still empty).
	mustEval(t, s, "types", "Found: Foo, Bar")

	ready = s.ReadySet()
	assertReadyContains(t, ready, "tests")
	assertReadyDoesNotContain(t, ready, "coverage")

	// Now evaluate tests. coverage should become ready.
	mustEval(t, s, "tests", "TestFoo, TestBar — 2 tests found")

	ready = s.ReadySet()
	assertReadyContains(t, ready, "coverage")
	if len(ready) != 1 {
		t.Errorf("ready set = %v, want [coverage]", ready)
	}

	// Evaluate coverage. All fresh.
	mustEval(t, s, "coverage", "Bar has no test coverage. 50% coverage.")
	if !s.AllFresh() {
		t.Error("expected all fresh")
	}

	// STALENESS TEST: Re-evaluate source. Types, tests, coverage all go stale.
	s.states["source"].State = StateStale
	mustEval(t, s, "source", "package main\n\ntype Foo struct{}\ntype Bar struct{}\ntype Baz struct{}")

	assertState(t, s, "types", StateStale)
	assertState(t, s, "tests", StateStale)
	assertState(t, s, "coverage", StateStale)

	// PREDICTION: Both types and tests are ready again.
	ready = s.ReadySet()
	assertReadyContains(t, ready, "types")
	assertReadyContains(t, ready, "tests")
	assertReadyDoesNotContain(t, ready, "coverage")

	// PREDICTION: Input snapshot for coverage captured v1 of types and tests.
	snap := s.GetInputSnapshot("coverage")
	if snap["types"] != 1 {
		t.Errorf("coverage snapshot types version = %d, want 1", snap["types"])
	}
	if snap["tests"] != 1 {
		t.Errorf("coverage snapshot tests version = %d, want 1", snap["tests"])
	}
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 3: Wide Fan — Maximum Parallelism
//
// Theory: source → 4 parallel analyzers → report.
// After evaluating source, all 4 analyzers are ready simultaneously.
// Report only becomes ready after ALL 4 are fresh.
// Evaluating 3 of 4 leaves report not-ready.
// ═══════════════════════════════════════════════════════════════

func TestScenarioWideFan(t *testing.T) {
	s := loadTestSheet(t, "wide-fan.yaml")

	// PREDICTION: Only source ready.
	assertReadyExactly(t, s.ReadySet(), []string{"source"})

	mustEval(t, s, "source", "A web application with REST API and SQL database")

	// PREDICTION: All 4 analyzers ready simultaneously.
	ready := s.ReadySet()
	if len(ready) != 4 {
		t.Fatalf("ready set size = %d, want 4 (all analyzers), got %v", len(ready), ready)
	}
	for _, name := range []string{"security", "performance", "maintainability", "correctness"} {
		assertReadyContains(t, ready, name)
	}
	assertReadyDoesNotContain(t, ready, "report")

	// Evaluate 3 of 4. Report should NOT be ready.
	mustEval(t, s, "security", "SQL injection risk in query builder")
	mustEval(t, s, "performance", "N+1 queries in user listing")
	mustEval(t, s, "maintainability", "Good separation of concerns")

	ready = s.ReadySet()
	assertReadyContains(t, ready, "correctness") // Last unevaluated analyzer.
	assertReadyDoesNotContain(t, ready, "report") // Missing correctness.

	// Evaluate last analyzer. Report becomes ready.
	mustEval(t, s, "correctness", "Missing null checks on API responses")

	ready = s.ReadySet()
	assertReadyExactly(t, ready, []string{"report"})

	// Evaluate report. All fresh.
	mustEval(t, s, "report", "Full system review complete")
	if !s.AllFresh() {
		t.Error("expected all fresh")
	}
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 4: Multi-Source — Independent Ready Sets
//
// Theory: 3 independent sources (code, docs, issues) each feed
// their own analyzer, which converge on gap-report.
// All 3 sources are ready initially. They can be evaluated in any
// order. gap-report needs code-analysis + doc-analysis + issues.
// ═══════════════════════════════════════════════════════════════

func TestScenarioMultiSource(t *testing.T) {
	s := loadTestSheet(t, "multi-source.yaml")

	// PREDICTION: All 3 independent sources are ready.
	ready := s.ReadySet()
	if len(ready) != 3 {
		t.Fatalf("ready set = %v, want 3 sources", ready)
	}
	assertReadyContains(t, ready, "code")
	assertReadyContains(t, ready, "docs")
	assertReadyContains(t, ready, "issues")

	// Evaluate in non-topological order: issues first, then code, then docs.
	mustEval(t, s, "issues", "3 open bugs, 2 feature requests")

	// PREDICTION: code and docs still ready (independent). issues consumed.
	// gap-report NOT ready (needs code-analysis and doc-analysis).
	ready = s.ReadySet()
	assertReadyContains(t, ready, "code")
	assertReadyContains(t, ready, "docs")
	assertReadyDoesNotContain(t, ready, "issues")
	assertReadyDoesNotContain(t, ready, "gap-report")

	mustEval(t, s, "code", "Go codebase, 15 packages, 200 files")
	mustEval(t, s, "docs", "README, API docs, architecture guide")

	// PREDICTION: code-analysis and doc-analysis are now ready.
	ready = s.ReadySet()
	assertReadyContains(t, ready, "code-analysis")
	assertReadyContains(t, ready, "doc-analysis")
	assertReadyDoesNotContain(t, ready, "gap-report")

	mustEval(t, s, "code-analysis", "Core types: Sheet, Cell, Effect, Distortion")
	mustEval(t, s, "doc-analysis", "Docs cover: Sheet, Cell. Missing: Effect, Distortion")

	// PREDICTION: gap-report is now ready (all 3 upstreams fresh).
	ready = s.ReadySet()
	assertReadyExactly(t, ready, []string{"gap-report"})

	mustEval(t, s, "gap-report", "Gaps: Effect and Distortion undocumented. 2 bugs relate to missing docs.")
	if !s.AllFresh() {
		t.Error("expected all fresh")
	}

	// STALENESS ISOLATION: Re-evaluating 'docs' should make doc-analysis
	// and gap-report stale, but NOT code-analysis (independent branch).
	s.states["docs"].State = StateStale
	mustEval(t, s, "docs", "Updated: README v2, added Effect docs")

	assertState(t, s, "doc-analysis", StateStale)
	assertState(t, s, "gap-report", StateStale)
	assertState(t, s, "code-analysis", StateFresh) // Independent!
	assertState(t, s, "code", StateFresh)          // Independent!
	assertState(t, s, "issues", StateFresh)         // Independent!
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 5: Prompt Template Filling
//
// Theory: FillPrompt replaces {{ref}} with the fresh upstream value.
// It should fail when upstream is not fresh. It should handle
// multi-ref prompts correctly.
// ═══════════════════════════════════════════════════════════════

func TestScenarioPromptFilling(t *testing.T) {
	s := loadTestSheet(t, "diamond.yaml")

	// PREDICTION: FillPrompt for coverage should fail (types and tests empty).
	_, err := s.FillPrompt("coverage")
	if err == nil {
		t.Error("expected error: upstream not fresh")
	}

	// Evaluate source, types, tests.
	mustEval(t, s, "source", "the code")
	mustEval(t, s, "types", "Foo, Bar, Baz")
	mustEval(t, s, "tests", "TestFoo, TestBaz")

	// PREDICTION: FillPrompt for coverage should succeed and contain both values.
	prompt, err := s.FillPrompt("coverage")
	if err != nil {
		t.Fatalf("FillPrompt: %v", err)
	}

	if !containsSubstring(prompt, "Foo, Bar, Baz") {
		t.Error("prompt should contain types value")
	}
	if !containsSubstring(prompt, "TestFoo, TestBaz") {
		t.Error("prompt should contain tests value")
	}
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 6: Version Tracking Through Re-evaluation
//
// Theory: Each evaluation increments the version. Input snapshots
// record which version of each upstream was consumed.
// ═══════════════════════════════════════════════════════════════

func TestScenarioVersionTracking(t *testing.T) {
	s := loadTestSheet(t, "linear-chain.yaml")

	// First pass: all versions should be 1.
	mustEval(t, s, "source", "v1 data")
	mustEval(t, s, "extract", "v1 extract")
	mustEval(t, s, "analyze", "v1 analysis")
	mustEval(t, s, "decide", "v1 decision")

	assertVersion(t, s, "source", 1)
	assertVersion(t, s, "extract", 1)
	assertVersion(t, s, "analyze", 1)
	assertVersion(t, s, "decide", 1)

	// PREDICTION: decide's input snapshot shows analyze v1.
	snap := s.GetInputSnapshot("decide")
	if snap["analyze"] != 1 {
		t.Errorf("decide input snapshot analyze = %d, want 1", snap["analyze"])
	}

	// Re-evaluate source (simulate upstream change).
	s.states["source"].State = StateStale
	mustEval(t, s, "source", "v2 data — added new types")
	assertVersion(t, s, "source", 2)

	// Re-evaluate the chain.
	mustEval(t, s, "extract", "v2 extract — found 5 types now")
	assertVersion(t, s, "extract", 2)

	// PREDICTION: extract's v2 snapshot shows source v2.
	snap = s.GetInputSnapshot("extract")
	if snap["source"] != 2 {
		t.Errorf("extract v2 input snapshot source = %d, want 2", snap["source"])
	}

	mustEval(t, s, "analyze", "v2 analysis — richer structure detected")
	mustEval(t, s, "decide", "v2 decision — proceed with expanded model")
	assertVersion(t, s, "analyze", 2)
	assertVersion(t, s, "decide", 2)

	// PREDICTION: decide's v2 snapshot shows analyze v2.
	snap = s.GetInputSnapshot("decide")
	if snap["analyze"] != 2 {
		t.Errorf("decide v2 input snapshot analyze = %d, want 2", snap["analyze"])
	}
}

// ═══════════════════════════════════════════════════════════════
// SCENARIO 7: Error Handling — Invalid Operations
//
// Theory: Cannot evaluate a cell that's fresh (not empty/stale).
// Cannot fill a prompt when upstream isn't fresh.
// ═══════════════════════════════════════════════════════════════

func TestScenarioErrorHandling(t *testing.T) {
	s := loadTestSheet(t, "linear-chain.yaml")

	// Evaluate source.
	mustEval(t, s, "source", "data")

	// PREDICTION: Trying to evaluate source again should fail (it's fresh, not empty/stale).
	err := s.Evaluate("source", "new data")
	if err == nil {
		t.Error("expected error: cannot evaluate fresh cell")
	}

	// PREDICTION (REVISED): Evaluate is permissive — it doesn't check upstream freshness.
	// Ready() is advisory; Evaluate() only requires cell be in empty/stale state.
	// After evaluating source, PropagateStale marks the entire chain stale,
	// so analyze IS stale and CAN be evaluated (even though extract isn't fresh).
	err = s.Evaluate("analyze", "premature analysis")
	if err != nil {
		t.Errorf("Evaluate(analyze) should succeed (permissive): %v", err)
	}
	// But analyze is now fresh despite extract not being fresh — the scheduler
	// (not the engine) is responsible for ordering.

	// PREDICTION: FillPrompt for extract should work (source is fresh).
	prompt, err := s.FillPrompt("extract")
	if err != nil {
		t.Errorf("FillPrompt(extract): unexpected error: %v", err)
	}
	if !containsSubstring(prompt, "data") {
		t.Error("filled prompt should contain source's value")
	}
}

// ═══════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════

func mustEval(t *testing.T, s *Sheet, cell, content string) {
	t.Helper()
	if err := s.Evaluate(cell, content); err != nil {
		t.Fatalf("Evaluate(%q): %v", cell, err)
	}
}

func assertReadyExactly(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("ready set = %v (len %d), want %v (len %d)", got, len(got), want, len(want))
		return
	}
	wantMap := make(map[string]bool)
	for _, w := range want {
		wantMap[w] = true
	}
	for _, g := range got {
		if !wantMap[g] {
			t.Errorf("ready set contains unexpected %q", g)
		}
	}
}

func assertReadyContains(t *testing.T, ready []string, name string) {
	t.Helper()
	for _, r := range ready {
		if r == name {
			return
		}
	}
	t.Errorf("ready set %v does not contain %q", ready, name)
}

func assertReadyDoesNotContain(t *testing.T, ready []string, name string) {
	t.Helper()
	for _, r := range ready {
		if r == name {
			t.Errorf("ready set %v should not contain %q", ready, name)
			return
		}
	}
}

func assertState(t *testing.T, s *Sheet, cell string, want CellState) {
	t.Helper()
	got, err := s.State(cell)
	if err != nil {
		t.Fatalf("State(%q): %v", cell, err)
	}
	if got != want {
		t.Errorf("cell %q state = %s, want %s", cell, got, want)
	}
}

func assertVersion(t *testing.T, s *Sheet, cell string, want int) {
	t.Helper()
	v := s.GetValue(cell)
	if v == nil {
		t.Fatalf("cell %q has no value", cell)
	}
	if v.Version != want {
		t.Errorf("cell %q version = %d, want %d", cell, v.Version, want)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		// Simple contains check.
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
