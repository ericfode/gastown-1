package engine

import (
	"strings"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// FromMolecule converts a parsed Cell molecule into an engine Sheet.
// Each cell in the molecule becomes a cell in the sheet. Prompt sections
// are concatenated into the prompt template. Refs become sheet refs.
func FromMolecule(mol *parser.Molecule) (*Sheet, error) {
	var cells []*Cell
	for _, c := range mol.Cells {
		ec := cellFromAST(c)
		cells = append(cells, ec)
	}
	for _, mc := range mol.MapCells {
		if mc.Body != nil {
			ec := cellFromAST(mc.Body)
			ec.Name = mc.Name
			cells = append(cells, ec)
		}
	}
	for _, rc := range mol.ReduceCells {
		if rc.Body != nil {
			ec := cellFromAST(rc.Body)
			ec.Name = rc.Name
			cells = append(cells, ec)
		}
	}
	return NewSheet(mol.Name, cells)
}

// cellFromAST converts a parser.Cell to an engine.Cell.
func cellFromAST(c *parser.Cell) *Cell {
	ec := &Cell{
		Name: c.Name,
		Type: mapCellType(c.Type.Name),
	}

	// Build prompt from all prompt sections.
	var parts []string
	for _, ps := range c.Prompts {
		if len(ps.Lines) > 0 {
			parts = append(parts, strings.Join(ps.Lines, "\n"))
		}
	}

	// Script cells: use the script body as prompt.
	if c.ScriptBody != "" {
		parts = append(parts, c.ScriptBody)
	}

	ec.Prompt = strings.Join(parts, "\n")

	// Extract refs.
	for _, r := range c.Refs {
		ec.Refs = append(ec.Refs, Ref{
			Cell:  r.Name,
			Field: r.Field,
		})
	}

	return ec
}

func mapCellType(name string) CellType {
	switch name {
	case "text":
		return TypeText
	case "inventory":
		return TypeInventory
	case "diagram":
		return TypeDiagram
	case "laws":
		return TypeLaws
	case "boundaries":
		return TypeBoundaries
	case "synthesis":
		return TypeSynthesis
	case "code":
		return TypeCode
	case "decision":
		return TypeDecision
	default:
		return TypeText // llm, script, distilled, etc. → text
	}
}
