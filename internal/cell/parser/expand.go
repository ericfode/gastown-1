package parser

import (
	"fmt"
	"strings"
)

// ExpandRecipe performs textual substitution of recipe parameters with actual
// arguments, returning a new Recipe with all {{param}} references replaced.
// This is the pre-application expansion step: params in cell names, wire
// endpoints, refs, and prompt text are all substituted.
func ExpandRecipe(recipe *Recipe, args []string) (*Recipe, error) {
	if len(args) != len(recipe.Params) {
		return nil, fmt.Errorf("recipe %q expects %d params (%v), got %d args",
			recipe.Name, len(recipe.Params), recipe.Params, len(args))
	}

	// Build substitution map: {{param}} → arg
	subs := make(map[string]string, len(args))
	for i, param := range recipe.Params {
		subs["{{"+param+"}}"] = args[i]
	}

	expanded := &Recipe{
		Name:   recipe.Name,
		Params: recipe.Params,
		Pos:    recipe.Pos,
	}

	for _, op := range recipe.Operations {
		expanded.Operations = append(expanded.Operations, expandOperation(op, subs))
	}

	return expanded, nil
}

func expandStr(s string, subs map[string]string) string {
	for k, v := range subs {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func expandOperation(op *Operation, subs map[string]string) *Operation {
	out := &Operation{
		Kind:   op.Kind,
		Target: expandStr(op.Target, subs),
		From:   expandStr(op.From, subs),
		To:     expandStr(op.To, subs),
		Pos:    op.Pos,
	}

	if op.Cell != nil {
		out.Cell = expandCell(op.Cell, subs)
	}
	for _, s := range op.Sources {
		out.Sources = append(out.Sources, expandStr(s, subs))
	}
	for _, t := range op.Targets {
		out.Targets = append(out.Targets, expandStr(t, subs))
	}
	for _, l := range op.Lines {
		out.Lines = append(out.Lines, expandStr(l, subs))
	}
	for _, s := range op.FromList {
		out.FromList = append(out.FromList, expandStr(s, subs))
	}
	for _, s := range op.ToList {
		out.ToList = append(out.ToList, expandStr(s, subs))
	}
	if op.Value != nil {
		v := expandValue(*op.Value, subs)
		out.Value = &v
	}

	return out
}

func expandCell(c *Cell, subs map[string]string) *Cell {
	out := &Cell{
		Name:       expandStr(c.Name, subs),
		Type:       c.Type,
		IsMeta:     c.IsMeta,
		ScriptBody: expandStr(c.ScriptBody, subs),
		Pos:        c.Pos,
	}

	for _, ref := range c.Refs {
		out.Refs = append(out.Refs, &RefDecl{
			Name:   expandStr(ref.Name, subs),
			Field:  ref.Field,
			OrJoin: ref.OrJoin,
			Pos:    ref.Pos,
		})
	}

	for _, ann := range c.Annotations {
		out.Annotations = append(out.Annotations, ann) // annotations are not parameterized
	}

	for _, ps := range c.Prompts {
		out.Prompts = append(out.Prompts, expandPromptSection(ps, subs))
	}

	if c.Oracle != nil {
		out.Oracle = c.Oracle // oracle blocks not parameterized
	}
	if c.AcceptBlock != nil {
		lines := make([]string, len(c.AcceptBlock.Lines))
		for i, l := range c.AcceptBlock.Lines {
			lines[i] = expandStr(l, subs)
		}
		out.AcceptBlock = &AcceptBlock{Lines: lines, Pos: c.AcceptBlock.Pos}
	}
	if c.VarsBlock != nil {
		out.VarsBlock = c.VarsBlock
	}

	for _, pa := range c.ParamAssigns {
		out.ParamAssigns = append(out.ParamAssigns, pa)
	}

	return out
}

func expandPromptSection(ps *PromptSection, subs map[string]string) *PromptSection {
	out := &PromptSection{
		Tag:    ps.Tag,
		Guard:  ps.Guard,
		Format: ps.Format,
		Each:   ps.Each,
		Pos:    ps.Pos,
	}
	for _, l := range ps.Lines {
		out.Lines = append(out.Lines, expandStr(l, subs))
	}
	return out
}

func expandValue(v Value, subs map[string]string) Value {
	switch v.Kind {
	case "string":
		return Value{Kind: "string", Str: expandStr(v.Str, subs)}
	case "ref":
		return Value{Kind: "ref", Ref: expandStr(v.Ref, subs)}
	default:
		return v
	}
}

// MatchSelector evaluates a selector expression against a cell.
// Returns true if the cell matches all predicates.
func MatchSelector(sel *SelectorExpr, cell *Cell) bool {
	if sel == nil {
		return true
	}
	for _, pred := range sel.Predicates {
		if !matchPred(pred, cell) {
			return false
		}
	}
	return true
}

func matchPred(pred *SelectorPred, cell *Cell) bool {
	fieldVal := ""
	switch pred.Field {
	case "type":
		fieldVal = cell.Type.Name
	case "name":
		fieldVal = cell.Name
	default:
		return false
	}

	switch pred.Op {
	case "==":
		return fieldVal == pred.Value
	case "!=":
		return fieldVal != pred.Value
	case "matches":
		return globMatch(pred.Value, fieldVal)
	case "contains":
		return strings.Contains(fieldVal, pred.Value)
	default:
		return false
	}
}

// globMatch implements simple glob matching (* matches any sequence).
func globMatch(pattern, s string) bool {
	// Simple glob: split on *, match segments in order.
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return s == pattern
	}

	// First part must be a prefix (unless pattern starts with *)
	if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]

	// Middle parts must appear in order
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(s, parts[i])
		if idx < 0 {
			return false
		}
		s = s[idx+len(parts[i]):]
	}

	// Last part must be a suffix (unless pattern ends with *)
	last := parts[len(parts)-1]
	if last != "" && !strings.HasSuffix(s, last) {
		return false
	}

	return true
}

// FilterCellsBySelector returns cells from a molecule that match the selector.
func FilterCellsBySelector(mol *Molecule, sel *SelectorExpr) []string {
	var names []string
	for _, c := range mol.Cells {
		if MatchSelector(sel, c) {
			names = append(names, c.Name)
		}
	}
	return names
}

// ApplyRecipe executes an expanded recipe's graph operations against a molecule.
// Operations: !add, !drop, !wire, !cut, !split, !merge, !refine, !seed.
func ApplyRecipe(mol *Molecule, recipe *Recipe) error {
	for _, op := range recipe.Operations {
		if err := applyOp(mol, op); err != nil {
			return fmt.Errorf("recipe %q operation %s: %w", recipe.Name, op.Kind, err)
		}
	}
	return nil
}

func applyOp(mol *Molecule, op *Operation) error {
	switch op.Kind {
	case "add":
		if op.Cell == nil {
			return fmt.Errorf("!add without cell")
		}
		// Idempotent: skip if cell with same name already exists
		// (supports multi-target recipe application).
		for _, c := range mol.Cells {
			if c.Name == op.Cell.Name {
				return nil
			}
		}
		mol.Cells = append(mol.Cells, op.Cell)

	case "drop":
		found := false
		filtered := make([]*Cell, 0, len(mol.Cells))
		for _, c := range mol.Cells {
			if c.Name == op.Target {
				found = true
				continue
			}
			filtered = append(filtered, c)
		}
		if !found {
			return fmt.Errorf("!drop: cell %q not found", op.Target)
		}
		mol.Cells = filtered

	case "wire":
		// Expand fan-in/fan-out into individual wires.
		var wirePairs [][2]string
		if len(op.FromList) > 0 && op.To != "" {
			for _, f := range op.FromList {
				wirePairs = append(wirePairs, [2]string{f, op.To})
			}
		} else if op.From != "" && len(op.ToList) > 0 {
			for _, t := range op.ToList {
				wirePairs = append(wirePairs, [2]string{op.From, t})
			}
		} else {
			wirePairs = [][2]string{{op.From, op.To}}
		}
		for _, pair := range wirePairs {
			// Idempotent: skip duplicate wires.
			dup := false
			for _, w := range mol.Wires {
				if w.From == pair[0] && w.To == pair[1] {
					dup = true
					break
				}
			}
			if !dup {
				mol.Wires = append(mol.Wires, &Wire{From: pair[0], To: pair[1], Pos: op.Pos})
			}
		}

	case "cut":
		found := false
		filtered := make([]*Wire, 0, len(mol.Wires))
		for _, w := range mol.Wires {
			if w.From == op.From && w.To == op.To {
				found = true
				continue
			}
			filtered = append(filtered, w)
		}
		if !found {
			return fmt.Errorf("!cut: wire %s -> %s not found", op.From, op.To)
		}
		mol.Wires = filtered

	case "split":
		// Split a cell's output to multiple targets
		for _, target := range op.Targets {
			mol.Wires = append(mol.Wires, &Wire{From: op.Target, To: target, Pos: op.Pos})
		}

	case "merge":
		// Merge multiple sources into a target
		for _, src := range op.Sources {
			mol.Wires = append(mol.Wires, &Wire{From: src, To: op.Target, Pos: op.Pos})
		}

	case "refine":
		// Update prompt lines on a cell
		for _, c := range mol.Cells {
			if c.Name == op.Target {
				if len(c.Prompts) > 0 {
					c.Prompts[len(c.Prompts)-1].Lines = append(
						c.Prompts[len(c.Prompts)-1].Lines, op.Lines...)
				}
				return nil
			}
		}
		return fmt.Errorf("!refine: cell %q not found", op.Target)

	case "seed":
		// Set initial value on a cell (via VarsBlock)
		for _, c := range mol.Cells {
			if c.Name == op.Target {
				if c.VarsBlock == nil {
					c.VarsBlock = &VarsBlock{Vars: make(map[string]Value), Pos: op.Pos}
				}
				if op.Value != nil {
					c.VarsBlock.Vars["_seed"] = *op.Value
				}
				return nil
			}
		}
		return fmt.Errorf("!seed: cell %q not found", op.Target)

	default:
		return fmt.Errorf("unknown operation %q", op.Kind)
	}
	return nil
}
