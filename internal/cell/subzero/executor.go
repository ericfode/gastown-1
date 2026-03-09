package subzero

import (
	"context"
	"encoding/json"
)

// CellExec is the execution request for a single cell.
type CellExec struct {
	Name    string
	Type    string // "llm", "script", "decision", "oracle", "meta", "mol", "distilled", "human"
	Prompts []PromptMsg
	Script     string            // bash script body for script cells
	Inputs     map[string]string // resolved ref values
	Params     map[string]string // param.* values
	FormatJSON string            // mock-friendly JSON stub matching format> spec
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
// If the cell has a FormatJSON set, it returns minimal valid JSON matching the shape.
type MockExecutor struct{}

func (m *MockExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	// If we have a format hint, return valid JSON stub
	if cell.FormatJSON != "" {
		return &CellResult{
			Output: cell.FormatJSON,
			Fields: parseJSONFields(cell.FormatJSON),
		}, nil
	}

	var out string
	for _, p := range cell.Prompts {
		out += "[" + p.Role + "] " + p.Content + "\n"
	}
	if cell.Script != "" {
		out = "[script] " + cell.Script
	}
	return &CellResult{Output: out}, nil
}

// parseJSONFields extracts top-level string fields from JSON.
func parseJSONFields(jsonStr string) map[string]string {
	fields := make(map[string]string)
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return fields
	}
	for k, v := range obj {
		switch val := v.(type) {
		case string:
			fields[k] = val
		default:
			b, _ := json.Marshal(val)
			fields[k] = string(b)
		}
	}
	return fields
}
