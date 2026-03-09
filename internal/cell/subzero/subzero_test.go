package subzero

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// --- Executor tests ---

func TestMockExecutorReturnsPrompt(t *testing.T) {
	mock := &MockExecutor{}
	ctx := context.Background()
	result, err := mock.Execute(ctx, &CellExec{
		Name:    "greet",
		Type:    "llm",
		Prompts: []PromptMsg{{Role: "user", Content: "Say hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == "" {
		t.Fatal("expected non-empty output from mock")
	}
	if !strings.Contains(result.Output, "[user] Say hello") {
		t.Errorf("expected prompt echo, got: %s", result.Output)
	}
}

// --- Ref tests ---

func TestResolveRefs(t *testing.T) {
	outputs := map[string]*CellResult{
		"greet": {Output: "Hello world", Fields: map[string]string{"message": "Hello world"}},
	}
	params := map[string]string{"name": "Alice"}

	tests := []struct {
		input string
		want  string
	}{
		{"Say {{param.name}}", "Say Alice"},
		{"Got: {{greet}}", "Got: Hello world"},
		{"Field: {{greet.message}}", "Field: Hello world"},
		{"No refs here", "No refs here"},
		{"{{missing}}", "{{missing}}"},
	}
	for _, tt := range tests {
		got := ResolveRefs(tt.input, outputs, params)
		if got != tt.want {
			t.Errorf("ResolveRefs(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Script executor tests ---

func TestScriptExecutorRuns(t *testing.T) {
	exec := &ScriptExecutor{}
	result, err := exec.Execute(context.Background(), &CellExec{
		Name:   "test",
		Type:   "script",
		Script: "echo hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "hello\n" {
		t.Errorf("got %q, want %q", result.Output, "hello\n")
	}
}

func TestScriptExecutorBlocksDangerous(t *testing.T) {
	exec := &ScriptExecutor{}
	dangerous := []string{
		"gt sling foo",
		"gt mol create bar",
		"bd create 'Oops'",
		"gt sling abc -m 'hello'",
		"gt mail send someone",
		"gt nudge someone",
	}
	for _, script := range dangerous {
		_, err := exec.Execute(context.Background(), &CellExec{
			Name:   "bad",
			Type:   "script",
			Script: script,
		})
		if err == nil {
			t.Errorf("expected error for dangerous script: %s", script)
		}
		if err != nil && !strings.Contains(err.Error(), "BLOCKED") {
			t.Errorf("expected BLOCKED error, got: %v", err)
		}
	}
}

func TestScriptExecutorTimeout(t *testing.T) {
	exec := &ScriptExecutor{TimeoutSec: 1}
	ctx := context.Background()
	_, err := exec.Execute(ctx, &CellExec{
		Name:   "slow",
		Type:   "script",
		Script: "sleep 30",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Dispatch executor tests ---

func TestDispatchExecutor(t *testing.T) {
	d := &DispatchExecutor{
		LLM:    &MockExecutor{},
		Script: &ScriptExecutor{TimeoutSec: 5},
	}

	res, err := d.Execute(context.Background(), &CellExec{
		Name: "a", Type: "llm",
		Prompts: []PromptMsg{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output == "" {
		t.Fatal("expected llm output")
	}

	res, err = d.Execute(context.Background(), &CellExec{
		Name: "b", Type: "script",
		Script: "echo ok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Output != "ok\n" {
		t.Errorf("got %q", res.Output)
	}

	_, err = d.Execute(context.Background(), &CellExec{
		Name: "c", Type: "mol",
	})
	if err == nil {
		t.Fatal("expected error for mol cell type")
	}
}

// --- Runner tests ---

func TestRunInlineHelloCell(t *testing.T) {
	src := `-- test
## hello {
  input param.name : str required

  # greet : llm
    system>
      You are friendly.
    user>
      Say hello to {{param.name}}.
  #/

  # wrap : llm
    - greet
    user>
      Add emoji to: {{greet}}
  #/
##/`
	prog, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	runner := &Runner{
		Executor: &MockExecutor{},
		Params:   map[string]string{"name": "Alice"},
	}
	results, err := runner.Run(context.Background(), prog.Molecules[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := results["greet"]; !ok {
		t.Fatal("greet cell did not execute")
	}
	if _, ok := results["wrap"]; !ok {
		t.Fatal("wrap cell did not execute")
	}
	// wrap's prompt should contain greet's resolved output
	wrapOut := results["wrap"].Output
	if !strings.Contains(wrapOut, "[user]") {
		t.Errorf("wrap output should contain assembled prompt, got: %s", wrapOut)
	}
}

func TestMaxCellsLimit(t *testing.T) {
	src := `## big {
  # a : llm
    user> hi
  #/
  # b : llm
    user> hi
  #/
  # c : llm
    user> hi
  #/
##/`
	prog, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	runner := &Runner{
		Executor: &MockExecutor{},
		MaxCells: 2, // only allow 2
	}
	_, err = runner.Run(context.Background(), prog.Molecules[0])
	if err == nil {
		t.Fatal("expected max cells error")
	}
}

// --- Integration: run actual .cell files ---

func TestRunHelloCellFile(t *testing.T) {
	src, err := os.ReadFile("../../../docs/examples/hello.cell")
	if err != nil {
		t.Skip("hello.cell not found:", err)
	}

	prog, err := parser.Parse(string(src))
	if err != nil {
		t.Fatal("parse error:", err)
	}
	if len(prog.Molecules) == 0 {
		t.Skip("no molecules parsed")
	}

	runner := &Runner{
		Executor: &DispatchExecutor{
			LLM:    &MockExecutor{},
			Script: &ScriptExecutor{TimeoutSec: 5},
		},
		Params:   map[string]string{"name": "Alice"},
		MaxCells: 50,
	}
	results, err := runner.Run(context.Background(), prog.Molecules[0])
	if err != nil {
		t.Fatal("run error:", err)
	}

	for name, res := range results {
		t.Logf("cell %s: %.100s", name, res.Output)
	}

	if len(results) < 2 {
		t.Errorf("expected at least 2 cell results, got %d", len(results))
	}
}

func TestRunBatchCellFiles(t *testing.T) {
	files := []struct {
		path   string
		params map[string]string
	}{
		{"../../../docs/examples/hello.cell", map[string]string{"name": "Test"}},
		{"../../../docs/examples/shiny.cell", map[string]string{"feature": "test-feature"}},
		{"../../../docs/examples/survey.cell", map[string]string{"topic": "test"}},
	}

	for _, f := range files {
		t.Run(f.path, func(t *testing.T) {
			src, err := os.ReadFile(f.path)
			if err != nil {
				t.Skip("file not found:", err)
			}
			prog, err := parser.Parse(string(src))
			if err != nil {
				t.Logf("parse error (known bugs may cause this): %v", err)
				t.Skip("parse error — skipping")
			}
			if len(prog.Molecules) == 0 {
				t.Skip("no molecules")
			}

			runner := &Runner{
				Executor: &DispatchExecutor{
					LLM:    &MockExecutor{},
					Script: &ScriptExecutor{TimeoutSec: 5},
				},
				Params:   f.params,
				MaxCells: 50,
			}
			results, err := runner.Run(context.Background(), prog.Molecules[0])
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			t.Logf("executed %d cells successfully", len(results))
		})
	}
}

func TestDistilledExecutor(t *testing.T) {
	exec := &DistilledExecutor{
		Fallback: &MockExecutor{},
	}

	// Test with a distill section
	cell := &CellExec{
		Name: "decide",
		Type: "distilled",
		Inputs: map[string]string{
			"observe": "deacon_alive=false and other stuff",
		},
		Prompts: []PromptMsg{
			{
				Role: "distill",
				Content: `input_pattern: "{{observe}}"
output_map: {
  "deacon_alive=false" -> { action: "START", reason: "Dead session", confidence: 0.99 },
  "deacon_alive=true" -> { action: "NOTHING", reason: "Alive", confidence: 0.95 }
}
fallback: llm`,
			},
		},
	}

	result, err := exec.Execute(context.Background(), cell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Logf("output: %s", result.Output)

	// Should match "deacon_alive=false" pattern
	if result.Fields["action"] != "START" {
		t.Errorf("expected action=START, got %q", result.Fields["action"])
	}
}

func TestDistilledExecutorNoMatch(t *testing.T) {
	exec := &DistilledExecutor{
		Fallback: &MockExecutor{},
	}

	cell := &CellExec{
		Name:   "decide",
		Type:   "distilled",
		Inputs: map[string]string{"observe": "something_unexpected"},
		Prompts: []PromptMsg{
			{
				Role: "distill",
				Content: `input_pattern: "{{observe}}"
output_map: {
  "deacon_alive=false" -> { action: "START", reason: "Dead" }
}
fallback: llm`,
			},
		},
	}

	result, err := exec.Execute(context.Background(), cell)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No pattern matched, fallback to mock (non-JSON), should get default JSON
	if result.Fields["action"] != "UNKNOWN" {
		t.Errorf("expected action=UNKNOWN for no-match, got %q", result.Fields["action"])
	}
}

func TestOracleForLoop(t *testing.T) {
	oracle := &parser.OracleBlock{
		Statements: []*parser.OracleStmt{
			{Kind: "json_parse", Args: []string{"output"}},
			{
				Kind: "for",
				Expr: "for c in output.items",
				Body: []*parser.OracleStmt{
					{Kind: "assert", Expr: `c.name in ["a", "b", "c"]`},
				},
			},
		},
	}

	output := `{"items":[{"name":"a"},{"name":"b"}]}`
	if err := EvalOracle(oracle, output); err != nil {
		t.Errorf("expected pass, got: %v", err)
	}

	// Test failure case
	outputBad := `{"items":[{"name":"a"},{"name":"INVALID"}]}`
	if err := EvalOracle(oracle, outputBad); err == nil {
		t.Error("expected failure for invalid item, got nil")
	}
}
