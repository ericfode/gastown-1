package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSheet(t *testing.T) {
	// Create a temp YAML file.
	dir := t.TempDir()
	yaml := `name: test-sheet
cells:
  - name: source
    type: text
    prompt: "Read the source code."
  - name: analyze
    type: inventory
    prompt: "Analyze {{source}}"
    refs: [source]
`
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	sheet, err := loadSheet(path)
	if err != nil {
		t.Fatal(err)
	}
	if sheet.Name != "test-sheet" {
		t.Errorf("name = %q, want %q", sheet.Name, "test-sheet")
	}
	if len(sheet.Cells()) != 2 {
		t.Errorf("cells = %d, want 2", len(sheet.Cells()))
	}

	// Source should be ready, analyze should not.
	if !sheet.Ready("source") {
		t.Error("source should be ready")
	}
	if sheet.Ready("analyze") {
		t.Error("analyze should not be ready (source empty)")
	}

	// Evaluate source, then analyze should be ready.
	if err := sheet.Evaluate("source", "found types"); err != nil {
		t.Fatal(err)
	}
	if !sheet.Ready("analyze") {
		t.Error("analyze should be ready after source evaluated")
	}
}
