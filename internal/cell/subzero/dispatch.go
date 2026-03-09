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
	case "llm", "decision", "text":
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
		return &CellResult{Output: "oracle:pass"}, nil

	case "mol":
		return nil, fmt.Errorf("BLOCKED: mol() cells spawn external agents — not allowed in Sub-Zero")

	case "meta":
		return nil, fmt.Errorf("BLOCKED: meta cells emit Cell source — not allowed in Sub-Zero v0")

	case "distilled":
		if d.LLM != nil {
			return d.LLM.Execute(ctx, cell)
		}
		return &CellResult{Output: "distilled:stub"}, nil

	default:
		return nil, fmt.Errorf("unknown cell type %q", cell.Type)
	}
}
