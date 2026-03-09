package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandRecipeBasic(t *testing.T) {
	src := `recipe insert-gate(upstream, downstream) {
  !add # gate : llm
    - {{upstream}}
    user>
      Check: {{upstream}}
  #/
  !wire {{upstream}} -> gate
  !wire gate -> {{downstream}}
}`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(prog.Recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(prog.Recipes))
	}

	expanded, err := ExpandRecipe(prog.Recipes[0], []string{"analyze", "deploy"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	if len(expanded.Operations) != 3 {
		t.Fatalf("expected 3 ops, got %d", len(expanded.Operations))
	}

	// !add: cell should have ref to "analyze"
	addOp := expanded.Operations[0]
	if addOp.Kind != "add" {
		t.Fatalf("expected add, got %s", addOp.Kind)
	}
	if addOp.Cell.Refs[0].Name != "analyze" {
		t.Errorf("expected ref 'analyze', got %q", addOp.Cell.Refs[0].Name)
	}

	// !wire: upstream -> gate
	wireOp1 := expanded.Operations[1]
	if wireOp1.From != "analyze" || wireOp1.To != "gate" {
		t.Errorf("expected wire analyze->gate, got %s->%s", wireOp1.From, wireOp1.To)
	}

	// !wire: gate -> downstream
	wireOp2 := expanded.Operations[2]
	if wireOp2.From != "gate" || wireOp2.To != "deploy" {
		t.Errorf("expected wire gate->deploy, got %s->%s", wireOp2.From, wireOp2.To)
	}
}

func TestExpandRecipeRuleOfFive(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("../../../docs/examples/rule-of-five.cell"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	prog, err := Parse(string(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(prog.Recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(prog.Recipes))
	}

	expanded, err := ExpandRecipe(prog.Recipes[0], []string{"design"})
	if err != nil {
		t.Fatalf("expand: %v", err)
	}

	// Should have 5 !add + 1 !wire = 6 operations
	if len(expanded.Operations) != 6 {
		t.Fatalf("expected 6 ops, got %d", len(expanded.Operations))
	}

	// First cell name should be "design-draft"
	if expanded.Operations[0].Cell.Name != "design-draft" {
		t.Errorf("expected 'design-draft', got %q", expanded.Operations[0].Cell.Name)
	}

	// Last !wire should be "design-refine-4 -> design"
	lastOp := expanded.Operations[5]
	if lastOp.Kind != "wire" {
		t.Fatalf("expected wire, got %s", lastOp.Kind)
	}
	if lastOp.From != "design-refine-4" || lastOp.To != "design" {
		t.Errorf("expected wire design-refine-4->design, got %s->%s", lastOp.From, lastOp.To)
	}

	// Check ref substitution: design-refine-1 should ref design-draft
	refine1 := expanded.Operations[1]
	if refine1.Cell.Refs[0].Name != "design-draft" {
		t.Errorf("expected ref 'design-draft', got %q", refine1.Cell.Refs[0].Name)
	}
}

func TestExpandRecipeWrongArgCount(t *testing.T) {
	recipe := &Recipe{
		Name:   "test",
		Params: []string{"a", "b"},
	}
	_, err := ExpandRecipe(recipe, []string{"only-one"})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
	if !strings.Contains(err.Error(), "expects 2 params") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyRecipeToMolecule(t *testing.T) {
	// Parse a molecule + recipe, resolve, check result.
	src := `recipe add-gate(target) {
  !add # {{target}}-gate : human
    user>
      Approve {{target}}?
  #/
  !wire {{target}}-gate -> {{target}}
}

## pipeline
  # analyze : llm
    user>
      Analyze.
  #/
  # deploy : script
    - analyze
` + "```bash" + `
    echo deploying
` + "```" + `
  #/
  analyze -> deploy
  apply add-gate(deploy)
##/`

	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	mol := result.Program.Molecules[0]

	// Should now have 3 cells: analyze, deploy, deploy-gate
	if len(mol.Cells) != 3 {
		t.Fatalf("expected 3 cells, got %d", len(mol.Cells))
	}

	cellNames := make(map[string]bool)
	for _, c := range mol.Cells {
		cellNames[c.Name] = true
	}
	if !cellNames["deploy-gate"] {
		t.Error("missing deploy-gate cell after recipe application")
	}

	// Should have 2 wires: analyze->deploy, deploy-gate->deploy
	if len(mol.Wires) != 2 {
		t.Fatalf("expected 2 wires, got %d", len(mol.Wires))
	}

	wireFound := false
	for _, w := range mol.Wires {
		if w.From == "deploy-gate" && w.To == "deploy" {
			wireFound = true
		}
	}
	if !wireFound {
		t.Error("missing wire deploy-gate -> deploy after recipe application")
	}
}

func TestApplyRecipeSecurityAudit(t *testing.T) {
	dir := t.TempDir()

	// Write security-audit.cell
	writeTestCell(t, dir, "security-audit.cell", `recipe security-audit(targets) {
  !add # prescan : llm
    user>
      Pre-scan.
  #/
  !add # postscan : llm
    user>
      Post-scan.
  #/
  !wire prescan -> {{targets}}
  !wire {{targets}} -> postscan
}`)

	mainSrc := `## secure
  import security-audit
  # implement : llm
    user>
      Build it.
  #/
  apply security-audit(implement)
##/`

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

	// Should have: implement + prescan + postscan = 3 cells
	if len(mol.Cells) != 3 {
		t.Fatalf("expected 3 cells, got %d", len(mol.Cells))
	}

	// Should have wires: prescan->implement, implement->postscan
	wireMap := make(map[string]string)
	for _, w := range mol.Wires {
		wireMap[w.From] = w.To
	}
	if wireMap["prescan"] != "implement" {
		t.Error("missing wire prescan -> implement")
	}
	if wireMap["implement"] != "postscan" {
		t.Error("missing wire implement -> postscan")
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		{"impl*", "implement", true},
		{"impl*", "implementation", true},
		{"impl*", "deploy", false},
		{"*scan", "prescan", true},
		{"*scan", "postscan", true},
		{"*scan", "scanner", false},
		{"*-step-*", "build-step-1", true},
		{"*-step-*", "deploy", false},
		{"exact", "exact", true},
		{"exact", "notexact", false},
		{"*", "anything", true},
		{"pre*scan", "prescan", true},
		{"pre*scan", "pre-security-scan", true},
	}
	for _, tt := range tests {
		got := globMatch(tt.pattern, tt.s)
		if got != tt.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
		}
	}
}

func TestSelectorMatchType(t *testing.T) {
	cell := &Cell{Name: "deploy", Type: CellType{Name: "script"}}
	sel := &SelectorExpr{Predicates: []*SelectorPred{
		{Field: "type", Op: "==", Value: "script"},
	}}
	if !MatchSelector(sel, cell) {
		t.Error("expected match for type == script")
	}

	sel2 := &SelectorExpr{Predicates: []*SelectorPred{
		{Field: "type", Op: "==", Value: "llm"},
	}}
	if MatchSelector(sel2, cell) {
		t.Error("expected no match for type == llm")
	}
}

func TestSelectorMatchGlob(t *testing.T) {
	cell := &Cell{Name: "implement-feature", Type: CellType{Name: "llm"}}
	sel := &SelectorExpr{Predicates: []*SelectorPred{
		{Field: "name", Op: "matches", Value: "impl*"},
	}}
	if !MatchSelector(sel, cell) {
		t.Error("expected glob match for impl*")
	}
}

func TestSelectorContains(t *testing.T) {
	cell := &Cell{Name: "security-review", Type: CellType{Name: "llm"}}
	sel := &SelectorExpr{Predicates: []*SelectorPred{
		{Field: "name", Op: "contains", Value: "review"},
	}}
	if !MatchSelector(sel, cell) {
		t.Error("expected contains match for review")
	}
}

func TestRecipeWireFanOut(t *testing.T) {
	src := `recipe broadcast(source) {
  !wire {{source}} -> [deploy, report, notify]
}

## pipeline
  # analyze : llm
    user>
      Analyze.
  #/
  # deploy : script
` + "```bash\n    echo ok\n```" + `
  #/
  # report : text
    user>
      Report.
  #/
  # notify : text
    user>
      Notify.
  #/
  apply broadcast(analyze)
##/`

	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	mol := result.Program.Molecules[0]
	wireMap := make(map[string][]string)
	for _, w := range mol.Wires {
		wireMap[w.From] = append(wireMap[w.From], w.To)
	}

	targets := wireMap["analyze"]
	if len(targets) != 3 {
		t.Fatalf("expected 3 wires from analyze, got %d: %v", len(targets), targets)
	}
}

func TestRecipeWireFanIn(t *testing.T) {
	src := `recipe collect(target) {
  !wire [step1, step2, step3] -> {{target}}
}

## pipeline
  # step1 : llm
    user>
      Step 1.
  #/
  # step2 : llm
    user>
      Step 2.
  #/
  # step3 : llm
    user>
      Step 3.
  #/
  # summary : llm
    user>
      Summarize.
  #/
  apply collect(summary)
##/`

	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("resolve error: %v", e)
		}
		t.FailNow()
	}

	mol := result.Program.Molecules[0]
	count := 0
	for _, w := range mol.Wires {
		if w.To == "summary" {
			count++
		}
	}
	if count != 3 {
		t.Fatalf("expected 3 wires to summary, got %d", count)
	}
}

func TestApplyWithWhereSelector(t *testing.T) {
	src := `recipe add-gate(target) {
  !add # {{target}}-gate : human
    user>
      Approve {{target}}?
  #/
  !wire {{target}}-gate -> {{target}}
}

## pipeline
  # analyze : llm
    user>
      Analyze.
  #/
  # deploy : script
` + "```bash\n    echo deploying\n```" + `
  #/
  # report : text
    user>
      Report.
  #/
  analyze -> deploy -> report
  apply add-gate(deploy) where type == script
##/`

	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Resolve(prog, ResolveOptions{BaseDir: t.TempDir()})
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
	if !cellNames["deploy-gate"] {
		t.Error("missing deploy-gate after apply with where")
	}
}

func writeTestCell(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
