package cell

import (
	"fmt"
	"sort"
	"strings"
)

// ValidCellTypes lists the recognized cell types from the reactive sheet model.
var ValidCellTypes = map[string]bool{
	"text":       true,
	"inventory":  true,
	"diagram":    true,
	"laws":       true,
	"boundaries": true,
	"synthesis":  true,
	"code":       true,
	"decision":   true,
}

// ValidOperations lists the 8 graph primitives from the Formula Engine v2 spec.
var ValidOperations = map[string]bool{
	"addCell":       true,
	"removeCell":    true,
	"addRef":        true,
	"removeRef":     true,
	"splitCell":     true,
	"mergeCell":     true,
	"refinePrompt":  true,
	"seedValue":     true,
}

// Error represents a validation error with position information.
type Error struct {
	Pos     Position
	Message string
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// Validate checks a parsed Cell file for semantic errors:
// - Duplicate cell/recipe names
// - Unknown cell types
// - Unresolved refs (cell references that don't exist)
// - Unresolved oracle references
// - DAG well-formedness (no cycles in cell refs)
// - Recipe operation validity
// - Prompt template ref consistency
func Validate(f *File) []Error {
	var errs []Error

	cellNames := make(map[string]*CellDecl)
	recipeNames := make(map[string]*RecipeDecl)

	// Check duplicate cell names
	for _, c := range f.Cells {
		if prev, ok := cellNames[c.Name]; ok {
			errs = append(errs, Error{
				Pos:     c.Pos,
				Message: fmt.Sprintf("duplicate cell name %q (first defined at %s)", c.Name, prev.Pos),
			})
		}
		cellNames[c.Name] = c
	}

	// Check duplicate recipe names
	for _, r := range f.Recipes {
		if prev, ok := recipeNames[r.Name]; ok {
			errs = append(errs, Error{
				Pos:     r.Pos,
				Message: fmt.Sprintf("duplicate recipe name %q (first defined at %s)", r.Name, prev.Pos),
			})
		}
		recipeNames[r.Name] = r
	}

	// Check cell types
	for _, c := range f.Cells {
		if c.Type != "" && !ValidCellTypes[c.Type] {
			errs = append(errs, Error{
				Pos:     c.Pos,
				Message: fmt.Sprintf("unknown cell type %q (valid types: %s)", c.Type, validTypeList()),
			})
		}
	}

	// Check refs point to existing cells
	for _, c := range f.Cells {
		for _, ref := range c.Refs {
			if _, ok := cellNames[ref]; !ok {
				errs = append(errs, Error{
					Pos:     c.Pos,
					Message: fmt.Sprintf("cell %q refs unknown cell %q", c.Name, ref),
				})
			}
		}
	}

	// Check oracle references
	for _, c := range f.Cells {
		if c.Oracle != "" {
			if _, ok := cellNames[c.Oracle]; !ok {
				errs = append(errs, Error{
					Pos:     c.Pos,
					Message: fmt.Sprintf("cell %q oracle references unknown cell %q", c.Name, c.Oracle),
				})
			}
		}
	}

	// Check prompt template references match declared refs
	for _, c := range f.Cells {
		promptRefs := extractPromptRefs(c.Prompt)
		declaredRefs := make(map[string]bool)
		for _, r := range c.Refs {
			declaredRefs[r] = true
		}
		for _, pr := range promptRefs {
			// Extract the cell name (before any ".field" suffix)
			cellRef := pr
			if dot := strings.Index(pr, "."); dot >= 0 {
				cellRef = pr[:dot]
			}
			if !declaredRefs[cellRef] {
				errs = append(errs, Error{
					Pos:     c.Pos,
					Message: fmt.Sprintf("cell %q prompt references {{%s}} but %q is not in refs", c.Name, pr, cellRef),
				})
			}
		}
	}

	// Check for cycles in cell ref graph
	if cycleErr := checkCellCycles(f.Cells); cycleErr != nil {
		errs = append(errs, *cycleErr)
	}

	// Validate recipe operations
	for _, r := range f.Recipes {
		errs = append(errs, validateRecipe(r)...)
	}

	return errs
}

// extractPromptRefs pulls {{name}} and {{name.field}} references from a prompt template.
func extractPromptRefs(prompt string) []string {
	var refs []string
	for {
		start := strings.Index(prompt, "{{")
		if start < 0 {
			break
		}
		prompt = prompt[start+2:]
		end := strings.Index(prompt, "}}")
		if end < 0 {
			break
		}
		ref := strings.TrimSpace(prompt[:end])
		if ref != "" {
			refs = append(refs, ref)
		}
		prompt = prompt[end+2:]
	}
	return refs
}

// checkCellCycles detects cycles in the cell dependency graph.
func checkCellCycles(cells []*CellDecl) *Error {
	deps := make(map[string][]string)
	for _, c := range cells {
		deps[c.Name] = c.Refs
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var cyclePath []string

	var visit func(name string) bool
	visit = func(name string) bool {
		if inStack[name] {
			cyclePath = append(cyclePath, name)
			return true
		}
		if visited[name] {
			return false
		}
		visited[name] = true
		inStack[name] = true
		cyclePath = append(cyclePath, name)

		for _, dep := range deps[name] {
			if visit(dep) {
				return true
			}
		}

		inStack[name] = false
		cyclePath = cyclePath[:len(cyclePath)-1]
		return false
	}

	// Sort for deterministic output
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cyclePath = nil
		if visit(name) {
			// Find the start of the cycle in cyclePath
			last := cyclePath[len(cyclePath)-1]
			start := 0
			for i, n := range cyclePath[:len(cyclePath)-1] {
				if n == last {
					start = i
					break
				}
			}
			cycle := cyclePath[start:]
			return &Error{
				Message: fmt.Sprintf("dependency cycle detected: %s", strings.Join(cycle, " → ")),
			}
		}
	}

	return nil
}

// validateRecipe checks that recipe operations are valid.
func validateRecipe(r *RecipeDecl) []Error {
	var errs []Error
	params := make(map[string]bool)
	for _, p := range r.Params {
		params[p] = true
	}

	// Track locally-bound variables
	locals := make(map[string]bool)

	for _, stmt := range r.Body {
		var call *Call
		if stmt.Assignment != nil {
			call = stmt.Assignment.Call
			locals[stmt.Assignment.Name] = true
		} else {
			call = stmt.Call
		}

		if call != nil && !ValidOperations[call.Name] {
			errs = append(errs, Error{
				Pos:     call.Pos,
				Message: fmt.Sprintf("recipe %q uses unknown operation %q", r.Name, call.Name),
			})
		}
	}

	return errs
}

func validTypeList() string {
	types := make([]string, 0, len(ValidCellTypes))
	for t := range ValidCellTypes {
		types = append(types, t)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}
