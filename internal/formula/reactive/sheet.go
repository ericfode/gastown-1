package reactive

import (
	"fmt"
	"strings"
)

// Sheet is a reactive spreadsheet of interconnected cells.
type Sheet struct {
	Name    string
	cells   []Cell               // Ordered list of cells.
	cellMap map[string]int        // cell name -> index in cells slice.
	states  map[string]*cellEntry // cell name -> runtime state.
}

// Init creates a new Sheet with all cells in the empty state.
func Init(name string, cells []Cell) *Sheet {
	s := &Sheet{
		Name:    name,
		cells:   make([]Cell, len(cells)),
		cellMap: make(map[string]int, len(cells)),
		states:  make(map[string]*cellEntry, len(cells)),
	}
	copy(s.cells, cells)
	for i, c := range s.cells {
		s.cellMap[c.Name] = i
		s.states[c.Name] = &cellEntry{State: StateEmpty}
	}
	return s
}

// Cells returns the ordered list of cells.
func (s *Sheet) Cells() []Cell {
	out := make([]Cell, len(s.cells))
	copy(out, s.cells)
	return out
}

// State returns the current state of a cell.
func (s *Sheet) State(cellName string) (CellState, error) {
	e, ok := s.states[cellName]
	if !ok {
		return StateEmpty, fmt.Errorf("unknown cell: %q", cellName)
	}
	return e.State, nil
}

// GetValue returns the current value of a cell, or nil if no value exists.
func (s *Sheet) GetValue(cellName string) *Value {
	e, ok := s.states[cellName]
	if !ok {
		return nil
	}
	return e.Value
}

// upstreamRefs returns all cell names that a given cell depends on.
func (s *Sheet) upstreamRefs(cellName string) []string {
	idx, ok := s.cellMap[cellName]
	if !ok {
		return nil
	}
	cell := s.cells[idx]

	// Collect from explicit refs.
	seen := make(map[string]bool)
	var refs []string
	for _, r := range cell.Refs {
		if !seen[r.Cell] {
			seen[r.Cell] = true
			refs = append(refs, r.Cell)
		}
	}

	// Also collect from prompt template {{ref}} and {{ref.field}} patterns,
	// but only if they reference actual cells in this sheet.
	for _, name := range extractTemplateRefs(cell.Prompt) {
		if !seen[name] {
			if _, exists := s.cellMap[name]; exists {
				seen[name] = true
				refs = append(refs, name)
			}
		}
	}
	return refs
}

// extractTemplateRefs extracts cell names from {{name}} and {{name.field}} placeholders.
func extractTemplateRefs(prompt string) []string {
	var refs []string
	seen := make(map[string]bool)
	rest := prompt
	for {
		start := strings.Index(rest, "{{")
		if start == -1 {
			break
		}
		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end == -1 {
			break
		}
		ref := strings.TrimSpace(rest[:end])
		rest = rest[end+2:]

		// Extract cell name (before optional .field).
		name := ref
		if dot := strings.Index(ref, "."); dot >= 0 {
			name = ref[:dot]
		}
		name = strings.TrimSpace(name)
		if name != "" && !seen[name] {
			seen[name] = true
			refs = append(refs, name)
		}
	}
	return refs
}

// downstreamDeps returns all cell names that directly depend on the given cell.
func (s *Sheet) downstreamDeps(cellName string) []string {
	var deps []string
	for _, c := range s.cells {
		if c.Name == cellName {
			continue
		}
		for _, upstream := range s.upstreamRefs(c.Name) {
			if upstream == cellName {
				deps = append(deps, c.Name)
				break
			}
		}
	}
	return deps
}

// Ready reports whether a cell is ready for evaluation: it must be empty or
// stale, and all upstream refs must be fresh.
func (s *Sheet) Ready(cellName string) bool {
	e, ok := s.states[cellName]
	if !ok {
		return false
	}
	if e.State != StateEmpty && e.State != StateStale {
		return false
	}
	for _, ref := range s.upstreamRefs(cellName) {
		re, ok := s.states[ref]
		if !ok || re.State != StateFresh {
			return false
		}
	}
	return true
}

// ReadySet returns all cell names that are ready for evaluation.
func (s *Sheet) ReadySet() []string {
	var ready []string
	for _, c := range s.cells {
		if s.Ready(c.Name) {
			ready = append(ready, c.Name)
		}
	}
	return ready
}

// BeginCompute transitions a cell from empty/stale to computing.
func (s *Sheet) BeginCompute(cellName string) error {
	e, ok := s.states[cellName]
	if !ok {
		return fmt.Errorf("unknown cell: %q", cellName)
	}
	if e.State != StateEmpty && e.State != StateStale {
		return fmt.Errorf("cell %q in state %s, cannot begin compute", cellName, e.State)
	}
	e.State = StateComputing
	return nil
}

// Complete transitions a cell from computing to fresh with the given content.
func (s *Sheet) Complete(cellName string, content string) error {
	e, ok := s.states[cellName]
	if !ok {
		return fmt.Errorf("unknown cell: %q", cellName)
	}
	if e.State != StateComputing {
		return fmt.Errorf("cell %q in state %s, expected computing", cellName, e.State)
	}
	version := 1
	if e.Value != nil {
		version = e.Value.Version + 1
	}
	e.Value = &Value{Content: content, Version: version}
	e.State = StateFresh

	// Capture input snapshot: record which upstream versions were consumed.
	snapshot := make(map[string]int)
	for _, ref := range s.upstreamRefs(cellName) {
		if re, ok := s.states[ref]; ok && re.Value != nil {
			snapshot[ref] = re.Value.Version
		}
	}
	e.InputSnapshot = snapshot
	return nil
}

// GetInputSnapshot returns the input versions consumed for the last evaluation.
func (s *Sheet) GetInputSnapshot(cellName string) map[string]int {
	e, ok := s.states[cellName]
	if !ok || e.InputSnapshot == nil {
		return nil
	}
	out := make(map[string]int, len(e.InputSnapshot))
	for k, v := range e.InputSnapshot {
		out[k] = v
	}
	return out
}

// TotalEffect computes the total effect for the sheet by summing all cells
// with observed effects. Returns the sequentially composed cost.
func (s *Sheet) TotalEffect() Effect {
	total := EffectZero()
	for _, c := range s.cells {
		if c.ObservedEffect != nil {
			total = total.Seq(*c.ObservedEffect)
		}
	}
	return total
}

// Fail transitions a cell from computing to failed with an error.
func (s *Sheet) Fail(cellName string, err error) error {
	e, ok := s.states[cellName]
	if !ok {
		return fmt.Errorf("unknown cell: %q", cellName)
	}
	if e.State != StateComputing {
		return fmt.Errorf("cell %q in state %s, expected computing", cellName, e.State)
	}
	e.State = StateFailed
	e.LastError = err
	return nil
}

// PropagateStale marks all direct dependents of the given cell as stale.
// It recurses transitively through the dependency graph.
func (s *Sheet) PropagateStale(changedCell string) {
	for _, dep := range s.downstreamDeps(changedCell) {
		e := s.states[dep]
		if e.State == StateFresh || e.State == StateEmpty {
			e.State = StateStale
			s.PropagateStale(dep) // Transitive propagation.
		}
	}
}

// Evaluate is a convenience that performs beginCompute + complete + propagateStale.
func (s *Sheet) Evaluate(cellName string, content string) error {
	if err := s.BeginCompute(cellName); err != nil {
		return err
	}
	if err := s.Complete(cellName, content); err != nil {
		return err
	}
	s.PropagateStale(cellName)
	return nil
}

// FillPrompt fills {{ref}} and {{ref.field}} template holes in a cell's prompt
// with upstream fresh values. Returns an error if any upstream is not fresh.
func (s *Sheet) FillPrompt(cellName string) (string, error) {
	idx, ok := s.cellMap[cellName]
	if !ok {
		return "", fmt.Errorf("unknown cell: %q", cellName)
	}
	cell := s.cells[idx]
	prompt := cell.Prompt
	result := &strings.Builder{}
	rest := prompt

	for {
		start := strings.Index(rest, "{{")
		if start == -1 {
			result.WriteString(rest)
			break
		}
		result.WriteString(rest[:start])
		rest = rest[start+2:]
		end := strings.Index(rest, "}}")
		if end == -1 {
			result.WriteString("{{")
			result.WriteString(rest)
			break
		}
		ref := strings.TrimSpace(rest[:end])
		rest = rest[end+2:]

		// Extract cell name.
		name := ref
		if dot := strings.Index(ref, "."); dot >= 0 {
			name = ref[:dot]
		}
		name = strings.TrimSpace(name)

		re, ok := s.states[name]
		if !ok {
			return "", fmt.Errorf("unknown ref %q in cell %q prompt", name, cellName)
		}
		if re.State != StateFresh || re.Value == nil {
			return "", fmt.Errorf("ref %q not fresh (state: %s) in cell %q prompt", name, re.State, cellName)
		}
		result.WriteString(re.Value.Content)
	}
	return result.String(), nil
}

// AllFresh returns true if every cell in the sheet is in the fresh state.
func (s *Sheet) AllFresh() bool {
	for _, e := range s.states {
		if e.State != StateFresh {
			return false
		}
	}
	return true
}

// Compose merges two sheets into a new sheet, with optional cross-wiring
// connections. Each connection maps a cell in the result to additional refs.
type Connection struct {
	Cell string // Target cell name (must exist in one of the sheets).
	Refs []Ref  // Additional refs to wire in.
}

// Compose creates a new sheet combining cells from s and other, with
// additional cross-wiring connections. All cells start in empty state.
func (s *Sheet) Compose(other *Sheet, connections []Connection) (*Sheet, error) {
	// Collect all cells, checking for name conflicts.
	allCells := make([]Cell, 0, len(s.cells)+len(other.cells))
	names := make(map[string]bool)

	for _, c := range s.cells {
		if names[c.Name] {
			return nil, fmt.Errorf("duplicate cell name: %q", c.Name)
		}
		names[c.Name] = true
		allCells = append(allCells, c)
	}
	for _, c := range other.cells {
		if names[c.Name] {
			return nil, fmt.Errorf("duplicate cell name %q in composed sheets", c.Name)
		}
		names[c.Name] = true
		allCells = append(allCells, c)
	}

	// Apply connections: add refs to target cells.
	cellsByName := make(map[string]*Cell)
	for i := range allCells {
		cellsByName[allCells[i].Name] = &allCells[i]
	}
	for _, conn := range connections {
		target, ok := cellsByName[conn.Cell]
		if !ok {
			return nil, fmt.Errorf("connection target %q not found", conn.Cell)
		}
		target.Refs = append(target.Refs, conn.Refs...)
	}

	return Init(s.Name+"+"+other.Name, allCells), nil
}
