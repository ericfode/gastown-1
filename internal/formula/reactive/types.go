// Package reactive implements a lazy reactive bead spreadsheet engine.
//
// A Sheet is a collection of Cells connected by references (Refs). Each cell
// has a prompt template that may reference upstream cells via {{cellName}} or
// {{cellName.field}} syntax. When a cell's value changes, all downstream
// dependents are marked stale, triggering re-evaluation.
//
// The evaluation model is pull-based and lazy: cells are only computed when
// they appear in the ReadySet (all upstream refs are fresh).
package reactive

import (
	"fmt"
	"strings"
)

// CellType enumerates the kinds of cells in a sheet.
type CellType int

const (
	CellText CellType = iota
	CellInventory
	CellDiagram
	CellLaws
	CellBoundaries
	CellSynthesis
	CellCode
	CellDecision
)

var cellTypeNames = map[CellType]string{
	CellText:       "text",
	CellInventory:  "inventory",
	CellDiagram:    "diagram",
	CellLaws:       "laws",
	CellBoundaries: "boundaries",
	CellSynthesis:  "synthesis",
	CellCode:       "code",
	CellDecision:   "decision",
}

var cellTypeFromString = map[string]CellType{
	"text":       CellText,
	"inventory":  CellInventory,
	"diagram":    CellDiagram,
	"laws":       CellLaws,
	"boundaries": CellBoundaries,
	"synthesis":  CellSynthesis,
	"code":       CellCode,
	"decision":   CellDecision,
}

// String returns the string representation of a CellType.
func (ct CellType) String() string {
	if s, ok := cellTypeNames[ct]; ok {
		return s
	}
	return fmt.Sprintf("CellType(%d)", ct)
}

// ParseCellType converts a string to a CellType.
func ParseCellType(s string) (CellType, error) {
	if ct, ok := cellTypeFromString[strings.ToLower(s)]; ok {
		return ct, nil
	}
	return CellText, fmt.Errorf("unknown cell type: %q", s)
}

// Ref identifies a dependency on another cell, optionally a specific field.
type Ref struct {
	Cell  string // Name of the referenced cell.
	Field string // Optional field within the cell (empty = whole cell).
}

// Cell defines a node in the reactive sheet.
type Cell struct {
	Name     string   // Unique name within the sheet.
	Type     CellType // Kind of cell.
	Prompt   string   // Template string with {{ref}} placeholders.
	Refs     []Ref    // Explicit upstream dependencies.

	// Effect tracking (Gas City Sections 1, 14).
	ObservedEffect *FullEffect // Measured cost and distortion from last evaluation. Nil = not yet evaluated.
}

// Value holds the computed content of a cell.
type Value struct {
	Content string // The computed content.
	Version int    // Monotonically increasing version number.
}

// CellState represents the current evaluation state of a cell.
type CellState int

const (
	StateEmpty    CellState = iota // Never evaluated.
	StateStale                     // Has a previous value, but upstream changed.
	StateComputing                 // Currently being evaluated.
	StateFresh                     // Value is up-to-date.
	StateFailed                    // Evaluation failed.
)

var stateNames = map[CellState]string{
	StateEmpty:    "empty",
	StateStale:    "stale",
	StateComputing: "computing",
	StateFresh:    "fresh",
	StateFailed:   "failed",
}

func (cs CellState) String() string {
	if s, ok := stateNames[cs]; ok {
		return s
	}
	return fmt.Sprintf("CellState(%d)", cs)
}

// cellEntry holds the runtime state of a single cell in a sheet.
type cellEntry struct {
	State     CellState
	Value     *Value // nil when empty; holds last value in stale/computing/failed states.
	LastError error  // Non-nil only in failed state.

	// Observed effect from the last evaluation.
	ObservedEffect *FullEffect

	// Input snapshot: which upstream versions were consumed.
	InputSnapshot map[string]int // cell name -> version consumed.
}
