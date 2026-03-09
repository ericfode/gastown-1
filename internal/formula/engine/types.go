// Package engine implements the Lazy Reactive Bead Spreadsheet — a formula
// engine where cells are typed computations with prompt templates, dependencies
// flow through a DAG, and staleness propagates reactively.
//
// This is a direct translation of lean4/BeadCalculus/Unified.lean.
// Cells map to beads, refs map to dependencies, values map to work products.
package engine

import "fmt"

// CellType describes what kind of value a cell produces.
// Closed enumeration — extending requires conscious decision.
type CellType string

const (
	TypeText       CellType = "text"       // Unstructured text (fallback)
	TypeInventory  CellType = "inventory"  // Named list of items with properties
	TypeDiagram    CellType = "diagram"    // State machine or graph description
	TypeLaws       CellType = "laws"       // Formal/semi-formal invariants
	TypeBoundaries CellType = "boundaries" // Connections to other subsystems
	TypeSynthesis  CellType = "synthesis"  // Summary judgment combining inputs
	TypeCode       CellType = "code"       // Source code
	TypeDecision   CellType = "decision"   // Yes/no/conditional with rationale
)

// Ref is a reference to another cell's value inside a prompt template.
type Ref struct {
	Cell  string // Name of the referenced cell
	Field string // Optional field within the cell's value (empty = whole value)
}

// Cell is a named computation with a type, prompt template, and dependencies.
type Cell struct {
	Name     string
	Type     CellType
	Prompt   string // Prompt template with {{ref}} or {{ref.field}} holes
	Refs     []Ref  // All references extracted from the prompt
}

// Deps returns the names of cells this cell depends on.
func (c *Cell) Deps() []string {
	seen := make(map[string]bool, len(c.Refs))
	var deps []string
	for _, r := range c.Refs {
		if !seen[r.Cell] {
			seen[r.Cell] = true
			deps = append(deps, r.Cell)
		}
	}
	return deps
}

// Value is a cell's computed content with versioning and staleness tracking.
type Value struct {
	Content string
	Version int
	Stale   bool
}

// CellState is the evaluation lifecycle of a cell.
type CellState int

const (
	StateEmpty     CellState = iota // Never computed
	StateStale                      // Has old value, upstream changed
	StateComputing                  // Evaluator is working on it
	StateFresh                      // Computed and up to date
	StateFailed                     // Computation failed
)

func (s CellState) String() string {
	switch s {
	case StateEmpty:
		return "empty"
	case StateStale:
		return "stale"
	case StateComputing:
		return "computing"
	case StateFresh:
		return "fresh"
	case StateFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// CellEntry holds the runtime state of a cell: its state, current value, and error.
type CellEntry struct {
	State CellState
	Value *Value  // Non-nil for fresh, stale, and computing (last value)
	Error string  // Non-empty for failed state
}
