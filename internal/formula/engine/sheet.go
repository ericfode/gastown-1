package engine

import (
	"fmt"
	"strings"
)

// Sheet is the core data structure: a named collection of cells with tracked state.
// Direct translation of BeadCalculus.Unified.Sheet.
type Sheet struct {
	Name  string
	cells map[string]*Cell      // name → cell definition
	order []string              // insertion order for deterministic iteration
	state map[string]*CellEntry // name → runtime state
}

// NewSheet creates a sheet with all cells in the empty state.
func NewSheet(name string, cells []*Cell) (*Sheet, error) {
	s := &Sheet{
		Name:  name,
		cells: make(map[string]*Cell, len(cells)),
		order: make([]string, 0, len(cells)),
		state: make(map[string]*CellEntry, len(cells)),
	}
	for _, c := range cells {
		if _, exists := s.cells[c.Name]; exists {
			return nil, fmt.Errorf("duplicate cell name: %q", c.Name)
		}
		s.cells[c.Name] = c
		s.order = append(s.order, c.Name)
		s.state[c.Name] = &CellEntry{State: StateEmpty}
	}
	// Validate: all refs point to existing cells.
	for _, c := range cells {
		for _, ref := range c.Refs {
			if _, ok := s.cells[ref.Cell]; !ok {
				return nil, fmt.Errorf("cell %q references unknown cell %q", c.Name, ref.Cell)
			}
		}
	}
	// Validate: no cycles.
	if err := s.checkAcyclic(); err != nil {
		return nil, err
	}
	return s, nil
}

// Cell returns the definition of a cell by name.
func (s *Sheet) Cell(name string) (*Cell, bool) {
	c, ok := s.cells[name]
	return c, ok
}

// State returns the runtime state of a cell.
func (s *Sheet) State(name string) (*CellEntry, bool) {
	e, ok := s.state[name]
	return e, ok
}

// Cells returns cell names in insertion order.
func (s *Sheet) Cells() []string {
	return s.order
}

// Ready reports whether a cell can be evaluated:
// 1. Cell is empty or stale (not fresh, computing, or failed)
// 2. All referenced cells are fresh
func (s *Sheet) Ready(name string) bool {
	entry, ok := s.state[name]
	if !ok {
		return false
	}
	if entry.State != StateEmpty && entry.State != StateStale {
		return false
	}
	cell := s.cells[name]
	for _, ref := range cell.Refs {
		refEntry := s.state[ref.Cell]
		if refEntry.State != StateFresh {
			return false
		}
	}
	return true
}

// ReadyCells returns all cells that are ready for evaluation.
func (s *Sheet) ReadyCells() []string {
	var ready []string
	for _, name := range s.order {
		if s.Ready(name) {
			ready = append(ready, name)
		}
	}
	return ready
}

// BeginCompute transitions a cell from empty/stale to computing.
func (s *Sheet) BeginCompute(name string) error {
	entry, ok := s.state[name]
	if !ok {
		return fmt.Errorf("unknown cell: %q", name)
	}
	switch entry.State {
	case StateEmpty:
		entry.State = StateComputing
	case StateStale:
		entry.State = StateComputing
		// Value is preserved as "last" for potential recovery
	default:
		return fmt.Errorf("cell %q is %s, cannot begin compute", name, entry.State)
	}
	return nil
}

// Complete transitions a cell from computing to fresh with a new value.
// Automatically propagates staleness to downstream cells.
func (s *Sheet) Complete(name string, content string) error {
	entry, ok := s.state[name]
	if !ok {
		return fmt.Errorf("unknown cell: %q", name)
	}
	if entry.State != StateComputing {
		return fmt.Errorf("cell %q is %s, cannot complete", name, entry.State)
	}
	ver := 1
	if entry.Value != nil {
		ver = entry.Value.Version + 1
	}
	entry.State = StateFresh
	entry.Value = &Value{Content: content, Version: ver, Stale: false}
	entry.Error = ""

	s.propagateStale(name)
	return nil
}

// Fail transitions a cell from computing to failed.
func (s *Sheet) Fail(name string, err string) error {
	entry, ok := s.state[name]
	if !ok {
		return fmt.Errorf("unknown cell: %q", name)
	}
	if entry.State != StateComputing {
		return fmt.Errorf("cell %q is %s, cannot fail", name, entry.State)
	}
	entry.State = StateFailed
	entry.Error = err
	return nil
}

// Invalidate marks a fresh cell as stale, triggering downstream staleness.
// Use this when an external change requires recomputation.
func (s *Sheet) Invalidate(name string) error {
	entry, ok := s.state[name]
	if !ok {
		return fmt.Errorf("unknown cell: %q", name)
	}
	if entry.State != StateFresh {
		return fmt.Errorf("cell %q is %s, cannot invalidate", name, entry.State)
	}
	entry.State = StateStale
	if entry.Value != nil {
		entry.Value.Stale = true
	}
	s.propagateStale(name)
	return nil
}

// FillPrompt resolves a cell's prompt template by substituting {{ref}} placeholders
// with the fresh values of referenced cells. Returns error if any ref is not fresh.
func (s *Sheet) FillPrompt(name string) (string, error) {
	cell, ok := s.cells[name]
	if !ok {
		return "", fmt.Errorf("unknown cell: %q", name)
	}
	prompt := cell.Prompt
	for _, ref := range cell.Refs {
		refEntry := s.state[ref.Cell]
		if refEntry.State != StateFresh || refEntry.Value == nil {
			return "", fmt.Errorf("ref %q is not fresh (state: %s)", ref.Cell, refEntry.State)
		}
		placeholder := "{{" + ref.Cell + "}}"
		if ref.Field != "" {
			placeholder = "{{" + ref.Cell + "." + ref.Field + "}}"
		}
		prompt = strings.ReplaceAll(prompt, placeholder, refEntry.Value.Content)
	}
	return prompt, nil
}

// AllFresh reports whether every cell in the sheet is fresh.
func (s *Sheet) AllFresh() bool {
	for _, name := range s.order {
		if s.state[name].State != StateFresh {
			return false
		}
	}
	return true
}

// propagateStale marks all cells that depend on changedCell as stale (if fresh).
func (s *Sheet) propagateStale(changedCell string) {
	for _, name := range s.order {
		entry := s.state[name]
		if entry.State != StateFresh {
			continue
		}
		cell := s.cells[name]
		for _, ref := range cell.Refs {
			if ref.Cell == changedCell {
				entry.State = StateStale
				if entry.Value != nil {
					entry.Value.Stale = true
				}
				// Recurse: this cell is now stale, so its dependents may also be stale.
				s.propagateStale(name)
				break
			}
		}
	}
}

// checkAcyclic verifies the dependency graph has no cycles using DFS.
func (s *Sheet) checkAcyclic() error {
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)
	color := make(map[string]int, len(s.cells))
	var visit func(string) error
	visit = func(name string) error {
		color[name] = gray
		cell := s.cells[name]
		for _, ref := range cell.Refs {
			switch color[ref.Cell] {
			case gray:
				return fmt.Errorf("cycle detected: %s → %s", name, ref.Cell)
			case white:
				if err := visit(ref.Cell); err != nil {
					return err
				}
			}
		}
		color[name] = black
		return nil
	}
	for name := range s.cells {
		if color[name] == white {
			if err := visit(name); err != nil {
				return err
			}
		}
	}
	return nil
}
