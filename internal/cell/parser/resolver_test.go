package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSimpleImport(t *testing.T) {
	// shiny-secure imports shiny and security-audit.
	examplesDir := "../../../docs/examples"

	src, err := os.ReadFile(filepath.Join(examplesDir, "shiny-secure.cell"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	prog, err := Parse(string(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(prog.Molecules) != 1 {
		t.Fatalf("expected 1 molecule, got %d", len(prog.Molecules))
	}
	mol := prog.Molecules[0]
	if len(mol.Imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(mol.Imports))
	}

	// Before resolve: molecule has no cells of its own (just imports + apply).
	if len(mol.Cells) != 0 {
		t.Fatalf("expected 0 cells before resolve, got %d", len(mol.Cells))
	}

	result := Resolve(prog, ResolveOptions{BaseDir: examplesDir})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	// After resolve: should have shiny's 5 cells merged in.
	mol = result.Program.Molecules[0]
	if len(mol.Cells) < 5 {
		t.Errorf("expected at least 5 cells after resolving shiny, got %d", len(mol.Cells))
	}

	// Check specific cells from shiny.
	cellNames := make(map[string]bool)
	for _, c := range mol.Cells {
		cellNames[c.Name] = true
	}
	for _, expected := range []string{"design", "implement", "review", "test", "submit"} {
		if !cellNames[expected] {
			t.Errorf("missing expected cell %q from shiny import", expected)
		}
	}

	// Should also have recipes from security-audit.
	recipeNames := make(map[string]bool)
	for _, r := range result.Program.Recipes {
		recipeNames[r.Name] = true
	}
	if !recipeNames["security-audit"] {
		t.Error("missing security-audit recipe after resolve")
	}
}

func TestResolveEnterprise(t *testing.T) {
	examplesDir := "../../../docs/examples"

	src, err := os.ReadFile(filepath.Join(examplesDir, "shiny-enterprise.cell"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	prog, err := Parse(string(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: examplesDir})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	mol := result.Program.Molecules[0]

	// Should have shiny's 5 cells.
	if len(mol.Cells) < 5 {
		t.Errorf("expected at least 5 cells, got %d", len(mol.Cells))
	}

	// Should have rule-of-five recipe.
	recipeNames := make(map[string]bool)
	for _, r := range result.Program.Recipes {
		recipeNames[r.Name] = true
	}
	if !recipeNames["rule-of-five"] {
		t.Error("missing rule-of-five recipe after resolve")
	}

	// Should have inputs from shiny.
	if len(mol.Inputs) < 2 {
		t.Errorf("expected at least 2 inputs from shiny, got %d", len(mol.Inputs))
	}
}

func TestResolveCellCollision(t *testing.T) {
	// Create a temp file that defines the same cell name as another import.
	dir := t.TempDir()

	// File a.cell with cell "step1".
	writeCell(t, dir, "a.cell", `
## a
  # step1 : llm
    user>
      Do step 1.
  #/
##/
`)

	// File b.cell with cell "step1" (collision).
	writeCell(t, dir, "b.cell", `
## b
  # step1 : llm
    user>
      Do step 1 differently.
  #/
##/
`)

	// Main file imports both.
	mainSrc := `
## main
  import a
  import b
##/
`
	prog, err := Parse(mainSrc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: dir})

	if len(result.Errors) == 0 {
		t.Fatal("expected collision error, got none")
	}

	found := false
	for _, e := range result.Errors {
		if e.Import == "b" && contains(e.Reason, "step1") && contains(e.Reason, "collision") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected collision error for step1 from import b, got: %v", result.Errors)
	}
}

func TestResolveCircularImport(t *testing.T) {
	dir := t.TempDir()

	writeCell(t, dir, "x.cell", `
## x
  import y
  # xstep : llm
    user>
      X step.
  #/
##/
`)

	writeCell(t, dir, "y.cell", `
## y
  import x
  # ystep : llm
    user>
      Y step.
  #/
##/
`)

	src, err := os.ReadFile(filepath.Join(dir, "x.cell"))
	if err != nil {
		t.Fatal(err)
	}

	prog, err := Parse(string(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: dir})

	if len(result.Errors) == 0 {
		t.Fatal("expected circular import error, got none")
	}

	found := false
	for _, e := range result.Errors {
		if contains(e.Reason, "circular") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected circular import error, got: %v", result.Errors)
	}
}

func TestResolveFileNotFound(t *testing.T) {
	src := `
## main
  import nonexistent
##/
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})

	if len(result.Errors) == 0 {
		t.Fatal("expected file-not-found error, got none")
	}

	found := false
	for _, e := range result.Errors {
		if contains(e.Reason, "not found") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'not found' error, got: %v", result.Errors)
	}
}

func TestResolveNoImports(t *testing.T) {
	src := `
## simple
  # step : llm
    user>
      Do something.
  #/
##/
`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})

	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
	if len(result.Program.Molecules[0].Cells) != 1 {
		t.Errorf("expected 1 cell, got %d", len(result.Program.Molecules[0].Cells))
	}
}

func TestResolveSearchPaths(t *testing.T) {
	mainDir := t.TempDir()
	libDir := t.TempDir()

	// Put the importable file in a different directory.
	writeCell(t, libDir, "lib.cell", `
## lib
  # helper : llm
    user>
      Help.
  #/
##/
`)

	mainSrc := `
## main
  import lib
##/
`
	prog, err := Parse(mainSrc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Without search path: should fail.
	result := Resolve(prog, ResolveOptions{BaseDir: mainDir})
	if len(result.Errors) == 0 {
		t.Error("expected error without search path")
	}

	// Re-parse (resolve modifies in place).
	prog, _ = Parse(mainSrc)

	// With search path: should succeed.
	result = Resolve(prog, ResolveOptions{
		BaseDir:     mainDir,
		SearchPaths: []string{libDir},
	})
	if len(result.Errors) > 0 {
		t.Errorf("expected no errors with search path, got: %v", result.Errors)
	}

	mol := result.Program.Molecules[0]
	if len(mol.Cells) != 1 || mol.Cells[0].Name != "helper" {
		t.Errorf("expected helper cell from lib, got %d cells", len(mol.Cells))
	}
}

func TestResolveTransitiveImport(t *testing.T) {
	dir := t.TempDir()

	writeCell(t, dir, "base.cell", `
## base
  # foundation : llm
    user>
      Foundation step.
  #/
##/
`)

	writeCell(t, dir, "mid.cell", `
## mid
  import base
  # middle : llm
    - foundation
    user>
      Middle step.
  #/
##/
`)

	mainSrc := `
## top
  import mid
  # final : llm
    - middle
    user>
      Final step.
  #/
##/
`
	prog, err := Parse(mainSrc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: dir})

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	mol := result.Program.Molecules[0]
	cellNames := make(map[string]bool)
	for _, c := range mol.Cells {
		cellNames[c.Name] = true
	}

	// Should have: foundation (from base, via mid), middle (from mid), final (own).
	for _, expected := range []string{"foundation", "middle", "final"} {
		if !cellNames[expected] {
			t.Errorf("missing expected cell %q", expected)
		}
	}
}

func TestResolvePathTraversal(t *testing.T) {
	dir := t.TempDir()

	// Create a file outside the base dir to prove we can't reach it.
	parentDir := filepath.Dir(dir)
	writeCell(t, parentDir, "secret.cell", `
## secret
  # leaked : llm
    user>
      This should not be importable.
  #/
##/
`)

	tests := []struct {
		name       string
		importName string
		wantErr    string
	}{
		{"dot-dot slash", "../secret", "path separators not allowed"},
		{"forward slash", "sub/secret", "path separators not allowed"},
		{"backslash", `sub\secret`, "path separators not allowed"},
		{"bare dot-dot", "..", "path traversal not allowed"},
		{"bare dot", ".", "path traversal not allowed"},
		{"deep traversal", "../../etc/passwd", "path separators not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct the program directly (bypass parser) to test
			// resolver validation against programmatically-crafted imports.
			prog := &Program{
				Molecules: []*Molecule{{
					Name: "main",
					Imports: []*ImportDecl{{
						Name: tt.importName,
						Pos:  Position{Line: 1, Col: 1},
					}},
				}},
			}

			result := Resolve(prog, ResolveOptions{BaseDir: dir})

			if len(result.Errors) == 0 {
				t.Fatal("expected path traversal error, got none")
			}
			found := false
			for _, e := range result.Errors {
				if contains(e.Reason, tt.wantErr) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, result.Errors)
			}
		})
	}
}

func writeCell(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
