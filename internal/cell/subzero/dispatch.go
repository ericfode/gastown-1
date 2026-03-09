package subzero

import (
	"context"
	"fmt"
)

// DispatchExecutor routes cells to the appropriate executor by type.
type DispatchExecutor struct {
	LLM     Executor // for llm, decision cells
	Script  Executor // for script cells
	Polecat Executor // for mol() cells (nil = blocked)
	Human   Executor // for human cells (nil = blocked)
}

func (d *DispatchExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	switch cell.Type {
	case "llm", "decision":
		if d.LLM == nil {
			return nil, fmt.Errorf("no LLM executor configured")
		}
		return d.LLM.Execute(ctx, cell)

	case "text":
		// Text cells are pass-through: output is rendered prompt content.
		// No LLM call, no script. Just concatenate prompt sections.
		var out string
		for _, p := range cell.Prompts {
			out += p.Content
		}
		return &CellResult{Output: out, Fields: parseJSONFields(out)}, nil

	case "script":
		if d.Script == nil {
			return nil, fmt.Errorf("no script executor configured")
		}
		return d.Script.Execute(ctx, cell)

	case "oracle":
		return &CellResult{Output: "oracle:pass"}, nil

	case "mol":
		if d.Polecat != nil {
			return d.Polecat.Execute(ctx, cell)
		}
		return nil, fmt.Errorf("BLOCKED: mol() cells spawn external agents — not allowed in Sub-Zero (enable with Polecat executor)")

	case "human":
		if d.Human != nil {
			return d.Human.Execute(ctx, cell)
		}
		return nil, fmt.Errorf("BLOCKED: human cells require interactive input — configure HumanExecutor")

	case "meta":
		return nil, fmt.Errorf("BLOCKED: meta cells emit Cell source — not allowed in Sub-Zero v0")

	case "distilled":
		// Use dedicated distilled executor with LLM fallback
		de := &DistilledExecutor{Fallback: d.LLM}
		return de.Execute(ctx, cell)

	default:
		return nil, fmt.Errorf("unknown cell type %q", cell.Type)
	}
}
