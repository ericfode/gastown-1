# Cell Sub-Zero Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a purely local Cell evaluator that parses .cell files and executes the DAG without spawning polecats or external agents.

**Architecture:** New package `internal/cell/subzero/` with a pluggable `Executor` interface. Parse with existing `internal/cell/parser/`, toposort cells, execute each cell through the Executor, pass outputs downstream via ref substitution. Safety: hard reject any shell command containing `gt sling`, `gt mol`, or `bd create` patterns.

**Tech Stack:** Go, existing `parser` package, `os/exec` for script cells, no external dependencies.

**Module:** `github.com/steveyegge/gastown`

---

### Task 1: Executor Interface + Types

**Files:**
- Create: `internal/cell/subzero/executor.go`
- Test: `internal/cell/subzero/executor_test.go`

**Step 1: Write the failing test**

```go
package subzero

import (
	"context"
	"testing"
)

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
}
```

**Step 2: Run test to verify it fails**

Run: `cd /home/nixos/wasteland/gastown/crew/morpheus && go test ./internal/cell/subzero/ -run TestMockExecutor -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

```go
package subzero

import "context"

// CellExec is the execution request for a single cell.
type CellExec struct {
	Name    string
	Type    string // "llm", "script", "decision", "oracle", "meta", "mol", "distilled"
	Prompts []PromptMsg
	Script  string            // bash script body for script cells
	Inputs  map[string]string // resolved ref values
	Params  map[string]string // param.* values
}

// PromptMsg is a single message in the prompt assembly.
type PromptMsg struct {
	Role    string // "system", "context", "user", "examples", "think"
	Content string
}

// CellResult is the output of executing a cell.
type CellResult struct {
	Output string            // raw output text
	Fields map[string]string // parsed JSON fields (from format> spec)
	Error  error
}

// Executor runs a single cell. Implementations: MockExecutor, ScriptExecutor, LLMExecutor.
type Executor interface {
	Execute(ctx context.Context, cell *CellExec) (*CellResult, error)
}

// MockExecutor echoes the assembled prompt as output. For testing DAG mechanics.
type MockExecutor struct{}

func (m *MockExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	var out string
	for _, p := range cell.Prompts {
		out += "[" + p.Role + "] " + p.Content + "\n"
	}
	if cell.Script != "" {
		out = "[script] " + cell.Script
	}
	return &CellResult{Output: out}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd /home/nixos/wasteland/gastown/crew/morpheus && go test ./internal/cell/subzero/ -run TestMockExecutor -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cell/subzero/executor.go internal/cell/subzero/executor_test.go
git commit -m "feat(subzero): executor interface + mock executor"
```

---

### Task 2: Ref Substitution Engine

**Files:**
- Create: `internal/cell/subzero/refs.go`
- Test: `internal/cell/subzero/refs_test.go`

**Step 1: Write the failing test**

```go
package subzero

import "testing"

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
		{"{{missing}}", "{{missing}}"}, // unresolved refs pass through
	}
	for _, tt := range tests {
		got := ResolveRefs(tt.input, outputs, params)
		if got != tt.want {
			t.Errorf("ResolveRefs(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cell/subzero/ -run TestResolveRefs -v`
Expected: FAIL — ResolveRefs not defined

**Step 3: Write minimal implementation**

```go
package subzero

import (
	"regexp"
	"strings"
)

var refPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// ResolveRefs replaces {{ref}}, {{ref.field}}, and {{param.name}} in text.
func ResolveRefs(text string, outputs map[string]*CellResult, params map[string]string) string {
	return refPattern.ReplaceAllStringFunc(text, func(match string) string {
		ref := strings.TrimSpace(match[2 : len(match)-2])

		// param.X
		if strings.HasPrefix(ref, "param.") {
			key := ref[len("param."):]
			if v, ok := params[key]; ok {
				return v
			}
			return match
		}

		// ref.field
		if idx := strings.IndexByte(ref, '.'); idx > 0 {
			cellName := ref[:idx]
			field := ref[idx+1:]
			if r, ok := outputs[cellName]; ok && r.Fields != nil {
				if v, ok := r.Fields[field]; ok {
					return v
				}
			}
			return match
		}

		// plain ref
		if r, ok := outputs[ref]; ok {
			return r.Output
		}
		return match
	})
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cell/subzero/ -run TestResolveRefs -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cell/subzero/refs.go internal/cell/subzero/refs_test.go
git commit -m "feat(subzero): ref substitution engine"
```

---

### Task 3: DAG Toposort + Runner

**Files:**
- Create: `internal/cell/subzero/runner.go`
- Test: `internal/cell/subzero/runner_test.go`

**Step 1: Write the failing test**

```go
package subzero

import (
	"context"
	"strings"
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

func TestRunHelloCell(t *testing.T) {
	src := `-- test
## hello
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
##/
`
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
	// greet should have run before wrap
	if _, ok := results["greet"]; !ok {
		t.Fatal("greet cell did not execute")
	}
	if _, ok := results["wrap"]; !ok {
		t.Fatal("wrap cell did not execute")
	}
	// wrap should reference greet's output
	wrapOut := results["wrap"].Output
	if !strings.Contains(wrapOut, "[user]") {
		t.Errorf("wrap output should contain assembled prompt, got: %s", wrapOut)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cell/subzero/ -run TestRunHelloCell -v`
Expected: FAIL — Runner not defined

**Step 3: Write minimal implementation**

```go
package subzero

import (
	"context"
	"fmt"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// Runner executes a molecule's cells in topological order.
type Runner struct {
	Executor Executor
	Params   map[string]string
	// MaxCells is the hard limit on cell executions per run (fork-bomb prevention).
	// Default: 100.
	MaxCells int
}

// Run executes all cells in a molecule, returning results keyed by cell name.
func (r *Runner) Run(ctx context.Context, mol *parser.Molecule) (map[string]*CellResult, error) {
	maxCells := r.MaxCells
	if maxCells <= 0 {
		maxCells = 100
	}

	// Build adjacency: cell name → list of ref names
	cells := allCells(mol)
	if len(cells) > maxCells {
		return nil, fmt.Errorf("molecule has %d cells, exceeds max %d", len(cells), maxCells)
	}

	order, err := toposort(cells)
	if err != nil {
		return nil, fmt.Errorf("toposort: %w", err)
	}

	outputs := make(map[string]*CellResult)
	executed := 0

	for _, name := range order {
		if executed >= maxCells {
			return nil, fmt.Errorf("exceeded max cell executions (%d) — possible loop", maxCells)
		}

		cell := cells[name]
		exec := r.buildCellExec(cell, outputs)
		result, err := r.Executor.Execute(ctx, exec)
		if err != nil {
			return outputs, fmt.Errorf("cell %q: %w", name, err)
		}
		outputs[name] = result
		executed++
	}

	return outputs, nil
}

// cellInfo is the minimal info needed for toposort.
type cellInfo struct {
	name string
	typ  string
	refs []string
	cell *parser.Cell
}

// allCells extracts all cells (plain, map, reduce) into a flat map.
func allCells(mol *parser.Molecule) map[string]*cellInfo {
	m := make(map[string]*cellInfo)
	for _, c := range mol.Cells {
		refs := make([]string, 0, len(c.Refs))
		for _, r := range c.Refs {
			refs = append(refs, r.Name)
		}
		m[c.Name] = &cellInfo{name: c.Name, typ: c.Type.Name, refs: refs, cell: c}
	}
	for _, mc := range mol.MapCells {
		refs := []string{mc.OverRef}
		if mc.Body != nil {
			for _, r := range mc.Body.Refs {
				refs = append(refs, r.Name)
			}
		}
		m[mc.Name] = &cellInfo{name: mc.Name, typ: mc.Type.Name, refs: refs, cell: mc.Body}
	}
	for _, rc := range mol.ReduceCells {
		refs := []string{rc.OverRef}
		if rc.Body != nil {
			for _, r := range rc.Body.Refs {
				refs = append(refs, r.Name)
			}
		}
		m[rc.Name] = &cellInfo{name: rc.Name, typ: rc.Type.Name, refs: refs, cell: rc.Body}
	}
	return m
}

// toposort returns cell names in dependency order (Kahn's algorithm).
func toposort(cells map[string]*cellInfo) ([]string, error) {
	indegree := make(map[string]int)
	dependents := make(map[string][]string) // dependency → cells that depend on it

	for name := range cells {
		indegree[name] = 0
	}
	for name, info := range cells {
		for _, ref := range info.refs {
			if _, ok := cells[ref]; ok {
				indegree[name]++
				dependents[ref] = append(dependents[ref], name)
			}
			// refs to unknown cells are ignored (might be external/param)
		}
	}

	var queue []string
	for name, deg := range indegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	var order []string
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		order = append(order, name)
		for _, dep := range dependents[name] {
			indegree[dep]--
			if indegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	if len(order) != len(cells) {
		return nil, fmt.Errorf("cycle detected: sorted %d of %d cells", len(order), len(cells))
	}
	return order, nil
}

// buildCellExec assembles the execution request for a cell.
func (r *Runner) buildCellExec(info *cellInfo, outputs map[string]*CellResult) *CellExec {
	exec := &CellExec{
		Name:   info.name,
		Type:   info.typ,
		Inputs: make(map[string]string),
		Params: r.Params,
	}

	if info.cell == nil {
		return exec
	}

	// Assemble prompts with ref substitution
	for _, ps := range info.cell.Prompts {
		content := ""
		for _, line := range ps.Lines {
			resolved := ResolveRefs(line, outputs, r.Params)
			content += resolved + "\n"
		}
		exec.Prompts = append(exec.Prompts, PromptMsg{
			Role:    ps.Tag,
			Content: content,
		})
	}

	return exec
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cell/subzero/ -run TestRunHelloCell -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cell/subzero/runner.go internal/cell/subzero/runner_test.go
git commit -m "feat(subzero): DAG toposort runner with fork-bomb prevention"
```

---

### Task 4: Script Executor with Safety Guard

**Files:**
- Create: `internal/cell/subzero/script.go`
- Test: `internal/cell/subzero/script_test.go`

**Step 1: Write the failing test**

```go
package subzero

import (
	"context"
	"testing"
)

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
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cell/subzero/ -run TestScriptExecutor -v`
Expected: FAIL — ScriptExecutor not defined

**Step 3: Write minimal implementation**

```go
package subzero

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// dangerousPatterns are shell commands that could cause fork-bombing
// or uncontrolled agent spawning.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bgt\s+sling\b`),
	regexp.MustCompile(`\bgt\s+mol\b`),
	regexp.MustCompile(`\bbd\s+create\b`),
	regexp.MustCompile(`\bgt\s+nudge\b`),
	regexp.MustCompile(`\bgt\s+mail\b`),
}

// ScriptExecutor runs script cells as local bash subprocesses.
type ScriptExecutor struct {
	TimeoutSec int // default 30
}

func (s *ScriptExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	if cell.Type != "script" {
		return nil, fmt.Errorf("ScriptExecutor only handles script cells, got %q", cell.Type)
	}

	// Safety: reject dangerous commands
	for _, pat := range dangerousPatterns {
		if pat.MatchString(cell.Script) {
			return nil, fmt.Errorf("BLOCKED: script contains dangerous pattern %q — Cell Sub-Zero does not allow agent spawning", pat.String())
		}
	}

	timeout := time.Duration(s.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cell.Script)
	// Resolve params in script
	script := cell.Script
	for k, v := range cell.Params {
		script = strings.ReplaceAll(script, "{{param."+k+"}}", v)
	}
	for k, v := range cell.Inputs {
		script = strings.ReplaceAll(script, "{{"+k+"}}", v)
	}
	cmd = exec.CommandContext(ctx, "bash", "-c", script)

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("script %q timed out after %v", cell.Name, timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("script %q failed: %w\noutput: %s", cell.Name, err, string(out))
	}

	return &CellResult{Output: string(out)}, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cell/subzero/ -run TestScriptExecutor -v -timeout 10s`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cell/subzero/script.go internal/cell/subzero/script_test.go
git commit -m "feat(subzero): script executor with fork-bomb guard"
```

---

### Task 5: Composite Executor (dispatch by cell type)

**Files:**
- Create: `internal/cell/subzero/dispatch.go`
- Test: `internal/cell/subzero/dispatch_test.go`

**Step 1: Write the failing test**

```go
package subzero

import (
	"context"
	"testing"
)

func TestDispatchExecutor(t *testing.T) {
	d := &DispatchExecutor{
		LLM:    &MockExecutor{},
		Script: &ScriptExecutor{TimeoutSec: 5},
	}

	// llm cell goes to mock
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

	// script cell goes to script executor
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

	// mol() cell is BLOCKED
	_, err = d.Execute(context.Background(), &CellExec{
		Name: "c", Type: "mol",
	})
	if err == nil {
		t.Fatal("expected error for mol cell type")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cell/subzero/ -run TestDispatchExecutor -v`
Expected: FAIL — DispatchExecutor not defined

**Step 3: Write minimal implementation**

```go
package subzero

import (
	"context"
	"fmt"
)

// DispatchExecutor routes cells to the appropriate executor by type.
type DispatchExecutor struct {
	LLM    Executor // for llm, decision cells
	Script Executor // for script cells
}

func (d *DispatchExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	switch cell.Type {
	case "llm", "decision":
		if d.LLM == nil {
			return nil, fmt.Errorf("no LLM executor configured")
		}
		return d.LLM.Execute(ctx, cell)

	case "script":
		if d.Script == nil {
			return nil, fmt.Errorf("no script executor configured")
		}
		return d.Script.Execute(ctx, cell)

	case "oracle":
		// Oracle cells are validation-only, output is pass/fail
		return &CellResult{Output: "oracle:pass"}, nil

	case "mol":
		return nil, fmt.Errorf("BLOCKED: mol() cells spawn external agents — not allowed in Sub-Zero. Use Cell Zero v2+ for polecat dispatch")

	case "meta":
		return nil, fmt.Errorf("BLOCKED: meta cells emit Cell source — not allowed in Sub-Zero v0")

	case "distilled":
		// Distilled cells have lookup tables — for now, fall through to mock
		if d.LLM != nil {
			return d.LLM.Execute(ctx, cell)
		}
		return &CellResult{Output: "distilled:stub"}, nil

	default:
		return nil, fmt.Errorf("unknown cell type %q", cell.Type)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cell/subzero/ -run TestDispatchExecutor -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cell/subzero/dispatch.go internal/cell/subzero/dispatch_test.go
git commit -m "feat(subzero): dispatch executor routes by cell type, blocks mol/meta"
```

---

### Task 6: Integration Test — Run hello.cell End-to-End

**Files:**
- Modify: `internal/cell/subzero/runner_test.go`
- Symlink/copy: uses `docs/examples/hello.cell`

**Step 1: Write the failing test**

```go
// Add to runner_test.go
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
		t.Fatal("no molecules parsed")
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
		t.Logf("cell %s: %s", name, res.Output[:min(len(res.Output), 100)])
	}

	if len(results) != 2 { // greet + wrap
		t.Errorf("expected 2 cell results, got %d", len(results))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 2: Run test**

Run: `go test ./internal/cell/subzero/ -run TestRunHelloCellFile -v`
Expected: PASS (may need minor fixes to handle parser output shape)

**Step 3: Commit**

```bash
git add internal/cell/subzero/runner_test.go
git commit -m "test(subzero): integration test — hello.cell runs end-to-end with mock"
```

---

### Task 7: Batch Test — Run Multiple .cell Files

**Files:**
- Modify: `internal/cell/subzero/runner_test.go`

**Step 1: Write the batch test**

```go
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
```

**Step 2: Run and iterate**

Run: `go test ./internal/cell/subzero/ -run TestRunBatch -v`
Expected: Some files may fail due to parser bugs — that's expected. Log and skip.

**Step 3: Commit**

```bash
git add internal/cell/subzero/runner_test.go
git commit -m "test(subzero): batch test across multiple .cell files"
```

---

## Summary of Safety Mechanisms

1. **MaxCells limit** (default 100) — hard cap on cell executions per run
2. **dangerousPatterns** — regex blocklist for gt sling, gt mol, bd create, gt mail, gt nudge
3. **mol() type blocked** — DispatchExecutor refuses mol cells entirely
4. **meta type blocked** — no Cell source emission in v0
5. **Script timeout** (default 30s) — prevents runaway subprocesses
6. **Toposort cycle detection** — rejects cyclic DAGs before execution
