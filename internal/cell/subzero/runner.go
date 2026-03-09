package subzero

import (
	"context"
	"encoding/json"
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

		// Reduce cells get special sequential iteration
		if cell.reduce != nil && cell.reduce.TimesN > 0 {
			result, n, err := r.executeReduceLoop(ctx, cell, outputs, maxCells-executed)
			if err != nil {
				return outputs, fmt.Errorf("cell %q: %w", name, err)
			}
			outputs[name] = result
			executed += n
			continue
		}

		exec := r.buildCellExec(cell, outputs)
		result, err := r.Executor.Execute(ctx, exec)
		if err != nil {
			return outputs, fmt.Errorf("cell %q: %w", name, err)
		}

		// Run oracle validation if present
		if cell.cell != nil && cell.cell.Oracle != nil {
			if oracleErr := EvalOracle(cell.cell.Oracle, result.Output); oracleErr != nil {
				return outputs, fmt.Errorf("cell %q oracle failed: %w", name, oracleErr)
			}
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
	// reduce cell metadata (nil for plain/map cells)
	reduce *parser.ReduceCell
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
		var refs []string
		if rc.OverRef != "" {
			refs = append(refs, rc.OverRef)
		}
		if rc.Body != nil {
			for _, r := range rc.Body.Refs {
				refs = append(refs, r.Name)
			}
		}
		m[rc.Name] = &cellInfo{name: rc.Name, typ: rc.Type.Name, refs: refs, cell: rc.Body, reduce: rc}
	}
	return m
}

// toposort returns cell names in dependency order (Kahn's algorithm).
func toposort(cells map[string]*cellInfo) ([]string, error) {
	indegree := make(map[string]int)
	dependents := make(map[string][]string)

	for name := range cells {
		indegree[name] = 0
	}
	for name, info := range cells {
		for _, ref := range info.refs {
			if _, ok := cells[ref]; ok {
				indegree[name]++
				dependents[ref] = append(dependents[ref], name)
			}
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

	// Populate inputs from resolved refs
	for _, ref := range info.refs {
		if r, ok := outputs[ref]; ok {
			exec.Inputs[ref] = r.Output
		}
	}

	// Extract script body for script cells
	if info.cell.ScriptBody != "" {
		exec.Script = ResolveRefsWithContext(info.cell.ScriptBody, outputs, r.Params, info.name, info.refs)
	}

	// Assemble prompts with ref substitution
	for _, ps := range info.cell.Prompts {
		content := ""
		for _, line := range ps.Lines {
			resolved := ResolveRefsWithContext(line, outputs, r.Params, info.name, info.refs)
			content += resolved + "\n"
		}
		exec.Prompts = append(exec.Prompts, PromptMsg{
			Role:    ps.Tag,
			Content: content,
		})

		// Generate mock-friendly JSON from format> spec
		if ps.Format != nil && len(ps.Format.Fields) > 0 {
			exec.FormatJSON = generateMockJSON(ps.Format)
		}
	}

	return exec
}

// executeReduceLoop runs a bounded reduce cell N times with accumulator and optional early exit.
// Returns the final result, number of iterations executed, and any error.
func (r *Runner) executeReduceLoop(ctx context.Context, cell *cellInfo, outputs map[string]*CellResult, budget int) (*CellResult, int, error) {
	rc := cell.reduce
	n := rc.TimesN
	if n > budget {
		return nil, 0, fmt.Errorf("reduce %q needs %d iterations but only %d executions remain", cell.name, n, budget)
	}

	// Initialize accumulator from default value
	acc := valueToString(rc.AccDefault)

	var lastResult *CellResult
	for i := 0; i < n; i++ {
		// Build iteration-specific outputs: inject acc and iteration index
		iterOutputs := make(map[string]*CellResult, len(outputs)+2)
		for k, v := range outputs {
			iterOutputs[k] = v
		}
		// Make accumulator available as {{accIdent}} and iteration as {{asIdent}}
		iterOutputs[rc.AccIdent] = &CellResult{
			Output: acc,
			Fields: parseJSONFields(acc),
		}
		iterOutputs[rc.AsIdent] = &CellResult{
			Output: fmt.Sprintf("%d", i),
		}

		exec := r.buildCellExec(cell, iterOutputs)
		result, err := r.Executor.Execute(ctx, exec)
		if err != nil {
			return nil, i, fmt.Errorf("iteration %d: %w", i, err)
		}

		// Oracle validation per iteration
		if cell.cell != nil && cell.cell.Oracle != nil {
			if oracleErr := EvalOracle(cell.cell.Oracle, result.Output); oracleErr != nil {
				return nil, i, fmt.Errorf("iteration %d oracle failed: %w", i, oracleErr)
			}
		}

		lastResult = result
		acc = result.Output

		// Early exit: check until(field)
		if rc.UntilField != "" && result.Fields != nil {
			if v, ok := result.Fields[rc.UntilField]; ok && isTruthy(v) {
				return result, i + 1, nil
			}
		}
	}

	if lastResult == nil {
		lastResult = &CellResult{Output: acc, Fields: parseJSONFields(acc)}
	}
	return lastResult, n, nil
}

// valueToString converts a parser.Value to a string representation.
func valueToString(v parser.Value) string {
	switch v.Kind {
	case "string":
		return v.Str
	case "number":
		if v.Num == float64(int(v.Num)) {
			return fmt.Sprintf("%d", int(v.Num))
		}
		return fmt.Sprintf("%g", v.Num)
	case "bool":
		if v.Bool {
			return "true"
		}
		return "false"
	case "null":
		return ""
	default:
		b, _ := json.Marshal(v.Str)
		return string(b)
	}
}

// isTruthy checks if a string value is truthy (non-empty, non-"false", non-"0").
func isTruthy(s string) bool {
	return s != "" && s != "false" && s != "0" && s != "null"
}

// generateMockJSON creates a minimal valid JSON object from a FormatSpec.
func generateMockJSON(spec *parser.FormatSpec) string {
	obj := make(map[string]interface{})
	for _, f := range spec.Fields {
		obj[f.Name] = mockValue(f.Type)
	}
	b, _ := json.Marshal(obj)
	return string(b)
}

func mockValue(ft parser.FormatType) interface{} {
	switch ft.Kind {
	case "str":
		return "mock_value"
	case "number":
		return 1
	case "boolean":
		return true
	case "array":
		if ft.ElementType != nil {
			return []interface{}{mockValue(*ft.ElementType)}
		}
		return []interface{}{}
	case "object":
		obj := make(map[string]interface{})
		for _, f := range ft.Fields {
			obj[f.Name] = mockValue(f.Type)
		}
		return obj
	case "enum":
		if len(ft.EnumValues) > 0 {
			return ft.EnumValues[0]
		}
		return "mock_enum"
	default:
		return "mock"
	}
}
