package engine

import (
	"context"
	"fmt"
)

// Evaluator computes a cell's value given its filled prompt.
// Implementations: MockEvaluator (testing), LLMEvaluator (production).
type Evaluator interface {
	Eval(ctx context.Context, cell *Cell, prompt string) (string, error)
}

// MockEvaluator returns canned responses for testing.
type MockEvaluator struct {
	Responses map[string]string // cell name → response
}

func (m *MockEvaluator) Eval(_ context.Context, cell *Cell, _ string) (string, error) {
	if resp, ok := m.Responses[cell.Name]; ok {
		return resp, nil
	}
	return fmt.Sprintf("[mock output for %s]", cell.Name), nil
}

// Run evaluates all cells in topological order until the sheet is fully fresh
// or an error occurs. Returns the names of cells evaluated.
func Run(ctx context.Context, s *Sheet, eval Evaluator) ([]string, error) {
	var evaluated []string
	for {
		ready := s.ReadyCells()
		if len(ready) == 0 {
			break
		}
		for _, name := range ready {
			if err := ctx.Err(); err != nil {
				return evaluated, err
			}
			prompt, err := s.FillPrompt(name)
			if err != nil {
				return evaluated, fmt.Errorf("fill prompt for %q: %w", name, err)
			}
			if err := s.BeginCompute(name); err != nil {
				return evaluated, fmt.Errorf("begin compute %q: %w", name, err)
			}
			cell, _ := s.Cell(name)
			result, err := eval.Eval(ctx, cell, prompt)
			if err != nil {
				_ = s.Fail(name, err.Error())
				return evaluated, fmt.Errorf("eval %q: %w", name, err)
			}
			if err := s.Complete(name, result); err != nil {
				return evaluated, fmt.Errorf("complete %q: %w", name, err)
			}
			evaluated = append(evaluated, name)
		}
	}
	return evaluated, nil
}

// Compose merges two sheets by connecting output cells to input cells.
// Connections maps source cell name → target cell name. The source cell's
// value feeds into the target cell's references.
func Compose(a, b *Sheet, connections map[string]string) (*Sheet, error) {
	// Collect all cells, prefixing names to avoid conflicts.
	var cells []*Cell
	nameMap := make(map[string]string) // original → prefixed

	for _, name := range a.Cells() {
		c, _ := a.Cell(name)
		prefixed := a.Name + "." + name
		nameMap[name] = prefixed
		clone := &Cell{
			Name:   prefixed,
			Type:   c.Type,
			Prompt: c.Prompt,
			Refs:   rewriteRefs(c.Refs, a.Name, nameMap),
		}
		cells = append(cells, clone)
	}
	for _, name := range b.Cells() {
		c, _ := b.Cell(name)
		prefixed := b.Name + "." + name
		nameMap[name] = prefixed
		clone := &Cell{
			Name:   prefixed,
			Type:   c.Type,
			Prompt: c.Prompt,
			Refs:   rewriteRefs(c.Refs, b.Name, nameMap),
		}
		cells = append(cells, clone)
	}

	// Apply connections: rewrite target refs to point at source cells.
	for src, tgt := range connections {
		srcPrefixed := a.Name + "." + src
		tgtPrefixed := b.Name + "." + tgt
		// Find the target cell and add/update its refs.
		for _, c := range cells {
			if c.Name == tgtPrefixed {
				c.Refs = append(c.Refs, Ref{Cell: srcPrefixed})
			}
		}
	}

	return NewSheet(a.Name+" + "+b.Name, cells)
}

func rewriteRefs(refs []Ref, sheetName string, nameMap map[string]string) []Ref {
	out := make([]Ref, len(refs))
	for i, r := range refs {
		if prefixed, ok := nameMap[r.Cell]; ok {
			out[i] = Ref{Cell: prefixed, Field: r.Field}
		} else {
			out[i] = Ref{Cell: sheetName + "." + r.Cell, Field: r.Field}
		}
	}
	return out
}
