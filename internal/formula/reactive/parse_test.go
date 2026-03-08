package reactive

import (
	"testing"
)

func TestParseYAML(t *testing.T) {
	yaml := `
name: algebraic-survey
cells:
  - name: extract-types
    type: inventory
    prompt: |
      Read the Go source and list all algebraic types.
  - name: extract-laws
    type: laws
    prompt: |
      Read the Go source and identify invariants.
  - name: synthesize
    type: synthesis
    refs: [extract-types, extract-laws]
    prompt: |
      Given the type inventory: {{extract-types}}
      And the invariants: {{extract-laws}}
      What algebra is this?
`
	s, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "algebraic-survey" {
		t.Errorf("name = %q, want %q", s.Name, "algebraic-survey")
	}
	cells := s.Cells()
	if len(cells) != 3 {
		t.Fatalf("cells = %d, want 3", len(cells))
	}

	// Check types.
	if cells[0].Type != CellInventory {
		t.Errorf("cell 0 type = %s, want inventory", cells[0].Type)
	}
	if cells[1].Type != CellLaws {
		t.Errorf("cell 1 type = %s, want laws", cells[1].Type)
	}
	if cells[2].Type != CellSynthesis {
		t.Errorf("cell 2 type = %s, want synthesis", cells[2].Type)
	}

	// Check refs on synthesize.
	if len(cells[2].Refs) != 2 {
		t.Fatalf("synthesize refs = %d, want 2", len(cells[2].Refs))
	}
	if cells[2].Refs[0].Cell != "extract-types" || cells[2].Refs[1].Cell != "extract-laws" {
		t.Errorf("synthesize refs = %v", cells[2].Refs)
	}

	// Verify ready set: only leaf cells without upstream deps.
	ready := s.ReadySet()
	if len(ready) != 2 {
		t.Errorf("readySet = %v, want [extract-types, extract-laws]", ready)
	}
}

func TestParseYAMLNoName(t *testing.T) {
	yaml := `
cells:
  - name: a
    prompt: hello
`
	_, err := ParseYAML([]byte(yaml))
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestParseYAMLBadType(t *testing.T) {
	yaml := `
name: test
cells:
  - name: a
    type: nonexistent
    prompt: hello
`
	_, err := ParseYAML([]byte(yaml))
	if err == nil {
		t.Error("expected error for bad cell type")
	}
}

func TestParseYAMLDefaultType(t *testing.T) {
	yaml := `
name: test
cells:
  - name: a
    prompt: hello
`
	s, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	cells := s.Cells()
	if cells[0].Type != CellText {
		t.Errorf("default type = %s, want text", cells[0].Type)
	}
}

func TestParseYAMLFieldRef(t *testing.T) {
	yaml := `
name: test
cells:
  - name: source
    prompt: data
  - name: consumer
    refs: [source.summary]
    prompt: "use {{source.summary}}"
`
	s, err := ParseYAML([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	cells := s.Cells()
	if len(cells[1].Refs) != 1 {
		t.Fatalf("refs = %d, want 1", len(cells[1].Refs))
	}
	if cells[1].Refs[0].Cell != "source" || cells[1].Refs[0].Field != "summary" {
		t.Errorf("ref = %+v, want {source, summary}", cells[1].Refs[0])
	}
}
