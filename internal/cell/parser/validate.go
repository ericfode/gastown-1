package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError describes a semantic error in a Cell program.
type ValidationError struct {
	Message  string
	Severity string // "error" or "warning"
	Pos      Position
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s at %d:%d: %s", e.Severity, e.Pos.Line, e.Pos.Col, e.Message)
}

// Validate performs semantic validation on a parsed Cell program:
// - DAG well-formedness (no cycles)
// - Oracle reference validation
// - Undefined cell references
// - Duplicate names
// - Type consistency
func Validate(prog *Program) []*ValidationError {
	var errs []*ValidationError

	for _, mol := range prog.Molecules {
		errs = append(errs, validateMolecule(mol, prog)...)
	}

	return errs
}

func validateMolecule(mol *Molecule, prog *Program) []*ValidationError {
	var errs []*ValidationError

	// Build name registry
	cells := make(map[string]Position)
	oracles := make(map[string]Position)
	fragments := make(map[string]Position)

	for _, c := range mol.Cells {
		if prev, exists := cells[c.Name]; exists {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("duplicate cell name %q (previously defined at %d:%d)", c.Name, prev.Line, prev.Col),
				Severity: "error",
				Pos:      c.Pos,
			})
		}
		cells[c.Name] = c.Pos
	}

	for _, mc := range mol.MapCells {
		if prev, exists := cells[mc.Name]; exists {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("duplicate cell name %q (previously defined at %d:%d)", mc.Name, prev.Line, prev.Col),
				Severity: "error",
				Pos:      mc.Pos,
			})
		}
		cells[mc.Name] = mc.Pos
	}

	for _, rc := range mol.ReduceCells {
		if prev, exists := cells[rc.Name]; exists {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("duplicate cell name %q (previously defined at %d:%d)", rc.Name, prev.Line, prev.Col),
				Severity: "error",
				Pos:      rc.Pos,
			})
		}
		cells[rc.Name] = rc.Pos
	}

	for _, o := range mol.Oracles {
		oracles[o.Name] = o.Pos
	}

	for _, f := range mol.Fragments {
		fragments[f.Name] = f.Pos
	}

	// Also include program-level oracles and fragments
	for _, o := range prog.Oracles {
		oracles[o.Name] = o.Pos
	}
	for _, f := range prog.Fragments {
		fragments[f.Name] = f.Pos
	}

	// Build format field registry: cell name -> set of declared field names
	formatFields := make(map[string]map[string]bool)
	for _, c := range mol.Cells {
		for _, ps := range c.Prompts {
			if ps.Format != nil && len(ps.Format.Fields) > 0 {
				fields := make(map[string]bool)
				for _, f := range ps.Format.Fields {
					fields[f.Name] = true
				}
				formatFields[c.Name] = fields
			}
		}
	}

	// Validate cell references, field references, and human cell constraints
	for _, c := range mol.Cells {
		for _, ref := range c.Refs {
			refName := ref.Name
			if _, ok := cells[refName]; !ok {
				errs = append(errs, &ValidationError{
					Message:  fmt.Sprintf("undefined cell reference %q", refName),
					Severity: "error",
					Pos:      ref.Pos,
				})
			} else if ref.Field == "*" && ref.Name == c.Name {
				// Self-ref wildcard: {{self.*}} means "gather all dep outputs"
				// This is valid — not a cycle. No warning needed.
			} else if ref.Field != "" && ref.Field != "*" {
				// Check field reference against format> spec
				if fields, hasFormat := formatFields[refName]; hasFormat {
					if !fields[ref.Field] {
						errs = append(errs, &ValidationError{
							Message:  fmt.Sprintf("field %q not declared in format> of cell %q (available: %s)",
								ref.Field, refName, formatFieldNames(fields)),
							Severity: "warning",
							Pos:      ref.Pos,
						})
					}
				}
			}
		}
		if c.Type.Name == "human" {
			errs = append(errs, validateHumanCell(c)...)
		}
	}

	// Validate wire references
	for _, w := range mol.Wires {
		if _, ok := cells[w.From]; !ok {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("wire references undefined cell %q", w.From),
				Severity: "error",
				Pos:      w.Pos,
			})
		}
		if _, ok := cells[w.To]; !ok {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("wire references undefined cell %q", w.To),
				Severity: "error",
				Pos:      w.Pos,
			})
		}
		if w.OracleGate != "" {
			if _, ok := oracles[w.OracleGate]; !ok {
				// Could also be a cell of type oracle
				if _, ok := cells[w.OracleGate]; !ok {
					errs = append(errs, &ValidationError{
						Message:  fmt.Sprintf("wire references undefined oracle %q", w.OracleGate),
						Severity: "error",
						Pos:      w.Pos,
					})
				}
			}
		}
	}

	// Validate {{ref.field}} patterns in prompt text and scripts
	errs = append(errs, validateInlineFieldRefs(mol, cells, formatFields)...)

	// Build DAG and check for cycles
	errs = append(errs, checkDAGCycles(mol)...)

	// Validate input declarations
	errs = append(errs, validateInputs(mol)...)

	return errs
}

// checkDAGCycles detects cycles in the molecule's dependency graph.
func checkDAGCycles(mol *Molecule) []*ValidationError {
	// Build adjacency list from refs and wires
	graph := make(map[string][]string)

	// All cell names
	for _, c := range mol.Cells {
		graph[c.Name] = nil
	}
	for _, mc := range mol.MapCells {
		graph[mc.Name] = nil
	}
	for _, rc := range mol.ReduceCells {
		graph[rc.Name] = nil
	}

	// Add edges from cell refs (ref means "depends on")
	for _, c := range mol.Cells {
		for _, ref := range c.Refs {
			// ref.Name -> c.Name means c depends on ref.Name
			graph[ref.Name] = append(graph[ref.Name], c.Name)
		}
	}

	// Add edges from wires
	for _, w := range mol.Wires {
		graph[w.From] = append(graph[w.From], w.To)
	}

	// Detect cycles using DFS with coloring
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)
	color := make(map[string]int)
	parent := make(map[string]string)

	var errs []*ValidationError
	var cyclePath []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, neighbor := range graph[node] {
			if color[neighbor] == gray {
				// Found a cycle — reconstruct path
				cyclePath = []string{neighbor, node}
				cur := node
				for cur != neighbor {
					cur = parent[cur]
					if cur == "" {
						break
					}
					cyclePath = append(cyclePath, cur)
				}
				// Reverse
				for i, j := 0, len(cyclePath)-1; i < j; i, j = i+1, j-1 {
					cyclePath[i], cyclePath[j] = cyclePath[j], cyclePath[i]
				}
				return true
			}
			if color[neighbor] == white {
				parent[neighbor] = node
				if dfs(neighbor) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	for node := range graph {
		if color[node] == white {
			if dfs(node) {
				errs = append(errs, &ValidationError{
					Message:  fmt.Sprintf("cycle detected in DAG: %s", strings.Join(cyclePath, " -> ")),
					Severity: "error",
					Pos:      mol.Pos,
				})
				break // Report first cycle only
			}
		}
	}

	return errs
}

// validateHumanCell checks constraints specific to human cells.
func validateHumanCell(c *Cell) []*ValidationError {
	var errs []*ValidationError

	hasUser := false
	for _, ps := range c.Prompts {
		switch ps.Tag {
		case "user":
			hasUser = true
		case "system", "context", "think", "examples":
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("human cell %q must not have %s> section (LLM-specific)", c.Name, ps.Tag),
				Severity: "error",
				Pos:      ps.Pos,
			})
		}
	}

	if !hasUser {
		errs = append(errs, &ValidationError{
			Message:  fmt.Sprintf("human cell %q must have a user> section", c.Name),
			Severity: "error",
			Pos:      c.Pos,
		})
	}

	return errs
}

// inlineRefPattern matches {{cellname.field}} in prompt text (excludes param.X).
var inlineRefPattern = regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)\}\}`)

// validateInlineFieldRefs scans prompt text for {{ref.field}} and warns on unknown fields.
func validateInlineFieldRefs(mol *Molecule, cells map[string]Position, formatFields map[string]map[string]bool) []*ValidationError {
	var errs []*ValidationError

	for _, c := range mol.Cells {
		// Scan prompt lines
		for _, ps := range c.Prompts {
			for _, line := range ps.Lines {
				matches := inlineRefPattern.FindAllStringSubmatch(line, -1)
				for _, m := range matches {
					cellName, field := m[1], m[2]
					if cellName == "param" {
						continue
					}
					if _, ok := cells[cellName]; !ok {
						continue // Already caught by ref validation
					}
					if fields, hasFormat := formatFields[cellName]; hasFormat {
						if !fields[field] {
							errs = append(errs, &ValidationError{
								Message:  fmt.Sprintf("field %q not declared in format> of cell %q (available: %s)",
									field, cellName, formatFieldNames(fields)),
								Severity: "warning",
								Pos:      ps.Pos,
							})
						}
					}
				}
			}
		}

		// Scan script body
		if c.ScriptBody != "" {
			matches := inlineRefPattern.FindAllStringSubmatch(c.ScriptBody, -1)
			for _, m := range matches {
				cellName, field := m[1], m[2]
				if cellName == "param" {
					continue
				}
				if _, ok := cells[cellName]; !ok {
					continue
				}
				if fields, hasFormat := formatFields[cellName]; hasFormat {
					if !fields[field] {
						errs = append(errs, &ValidationError{
							Message:  fmt.Sprintf("field %q not declared in format> of cell %q (available: %s)",
								field, cellName, formatFieldNames(fields)),
							Severity: "warning",
							Pos:      c.Pos,
						})
					}
				}
			}
		}
	}

	return errs
}

// formatFieldNames returns a sorted comma-separated list of field names.
func formatFieldNames(fields map[string]bool) string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	// Sort for deterministic output
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	return strings.Join(names, ", ")
}

// validateInputs checks input declarations for consistency.
func validateInputs(mol *Molecule) []*ValidationError {
	var errs []*ValidationError
	seen := make(map[string]Position)

	for _, input := range mol.Inputs {
		if prev, exists := seen[input.ParamName]; exists {
			errs = append(errs, &ValidationError{
				Message:  fmt.Sprintf("duplicate input declaration %q (previously at %d:%d)", input.ParamName, prev.Line, prev.Col),
				Severity: "error",
				Pos:      input.Pos,
			})
		}
		seen[input.ParamName] = input.Pos

		// Check required_unless references
		for _, ref := range input.RequiredUnless {
			if _, ok := seen[ref]; !ok {
				// Could be declared later — just warn
				errs = append(errs, &ValidationError{
					Message:  fmt.Sprintf("required_unless references %q which may not be declared", ref),
					Severity: "warning",
					Pos:      input.Pos,
				})
			}
		}
	}

	return errs
}
