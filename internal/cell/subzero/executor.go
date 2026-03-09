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
