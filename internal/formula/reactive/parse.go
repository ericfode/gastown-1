package reactive

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormulaDefinition is the YAML schema for a reactive sheet formula.
type FormulaDefinition struct {
	Name  string           `yaml:"name"`
	Cells []CellDefinition `yaml:"cells"`
}

// CellDefinition is a single cell in a YAML formula.
type CellDefinition struct {
	Name   string   `yaml:"name"`
	Type   string   `yaml:"type"`
	Prompt string   `yaml:"prompt"`
	Refs   []string `yaml:"refs"`
}

// ParseYAML parses a YAML formula definition into a Sheet.
func ParseYAML(data []byte) (*Sheet, error) {
	var def FormulaDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	if def.Name == "" {
		return nil, fmt.Errorf("formula name is required")
	}

	cells := make([]Cell, 0, len(def.Cells))
	for _, cd := range def.Cells {
		ct := CellText
		if cd.Type != "" {
			var err error
			ct, err = ParseCellType(cd.Type)
			if err != nil {
				return nil, fmt.Errorf("cell %q: %w", cd.Name, err)
			}
		}

		refs := make([]Ref, 0, len(cd.Refs))
		for _, r := range cd.Refs {
			ref := Ref{Cell: r}
			if dot := strings.Index(r, "."); dot >= 0 {
				ref.Cell = r[:dot]
				ref.Field = r[dot+1:]
			}
			refs = append(refs, ref)
		}

		cells = append(cells, Cell{
			Name:   cd.Name,
			Type:   ct,
			Prompt: cd.Prompt,
			Refs:   refs,
		})
	}

	return Init(def.Name, cells), nil
}
