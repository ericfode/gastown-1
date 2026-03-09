package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLexBasic(t *testing.T) {
	source := `## hello {
  # greet : llm
    user>
      Say hello.
  #/
##/`
	tokens, err := Lex(source)
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}

	// Should have tokens
	if len(tokens) == 0 {
		t.Fatal("expected tokens")
	}

	// Check first token is ##
	if tokens[0].Type != TokenDoubleHash {
		t.Errorf("expected ##, got %s", tokens[0].Type)
	}
}

func TestParseSimpleMolecule(t *testing.T) {
	source := `## shiny {
  input param.feature : str required

  # design : llm
    user>
      Design the architecture for {{param.feature}}.
    accept>
      Design doc committed.
  #/

  # implement : llm
    - design
    user>
      Implement {{param.feature}} per the design: {{design}}
    accept>
      All files modified/created and committed.
  #/

  design -> implement
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Molecules) != 1 {
		t.Fatalf("expected 1 molecule, got %d", len(prog.Molecules))
	}

	mol := prog.Molecules[0]
	if mol.Name != "shiny" {
		t.Errorf("expected molecule name 'shiny', got %q", mol.Name)
	}

	if len(mol.Cells) != 2 {
		t.Errorf("expected 2 cells, got %d", len(mol.Cells))
	}

	if len(mol.Inputs) != 1 {
		t.Errorf("expected 1 input, got %d", len(mol.Inputs))
	}

	if len(mol.Wires) != 1 {
		t.Errorf("expected 1 wire, got %d", len(mol.Wires))
	}

	// Check cell names
	if mol.Cells[0].Name != "design" {
		t.Errorf("expected first cell 'design', got %q", mol.Cells[0].Name)
	}
	if mol.Cells[1].Name != "implement" {
		t.Errorf("expected second cell 'implement', got %q", mol.Cells[1].Name)
	}

	// Check implement has a ref to design
	if len(mol.Cells[1].Refs) != 1 {
		t.Errorf("expected 1 ref on implement, got %d", len(mol.Cells[1].Refs))
	} else if mol.Cells[1].Refs[0].Name != "design" {
		t.Errorf("expected ref 'design', got %q", mol.Cells[1].Refs[0].Name)
	}
}

func TestParseMapCell(t *testing.T) {
	source := `## review {
  map # leg : llm over {{param.aspects}} as aspect
    user>
      Review: {{aspect.focus}}
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.MapCells) != 1 {
		t.Fatalf("expected 1 map cell, got %d", len(mol.MapCells))
	}

	mc := mol.MapCells[0]
	if mc.Name != "leg" {
		t.Errorf("expected map cell name 'leg', got %q", mc.Name)
	}
	if mc.OverRef != "param.aspects" {
		t.Errorf("expected over ref 'param.aspects', got %q", mc.OverRef)
	}
	if mc.AsIdent != "aspect" {
		t.Errorf("expected as ident 'aspect', got %q", mc.AsIdent)
	}
}

func TestParseOracle(t *testing.T) {
	source := `## pipeline {
  # check : llm
    ` + "```" + ` oracle
    json_parse(v);
    assert v.status == "ok";
    ` + "```" + `
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(mol.Cells))
	}

	cell := mol.Cells[0]
	if cell.Oracle == nil {
		t.Fatal("expected oracle block on cell")
	}

	if len(cell.Oracle.Statements) != 2 {
		t.Errorf("expected 2 oracle statements, got %d", len(cell.Oracle.Statements))
	}
}

func TestParseWireWithOracle(t *testing.T) {
	source := `## pipeline {
  # a : llm
    user>
      Do A.
  #/

  # gate : oracle
    ` + "```" + ` oracle
    json_parse(v);
    ` + "```" + `
  #/

  # b : llm
    user>
      Do B.
  #/

  a -> ? gate -> b
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Wires) != 1 {
		t.Fatalf("expected 1 wire, got %d", len(mol.Wires))
	}

	wire := mol.Wires[0]
	if wire.From != "a" {
		t.Errorf("expected from 'a', got %q", wire.From)
	}
	if wire.OracleGate != "gate" {
		t.Errorf("expected oracle gate 'gate', got %q", wire.OracleGate)
	}
	if wire.To != "b" {
		t.Errorf("expected to 'b', got %q", wire.To)
	}
}

func TestValidateCycleDetection(t *testing.T) {
	source := `## cycle {
  # a : llm
    - b
    user>
      Do A.
  #/

  # b : llm
    - a
    user>
      Do B.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	hasCycle := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cycle") {
			hasCycle = true
			break
		}
	}
	if !hasCycle {
		t.Error("expected cycle detection error, got none")
	}
}

func TestValidateUndefinedRef(t *testing.T) {
	source := `## test {
  # a : llm
    - nonexistent
    user>
      Do A.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	hasUndef := false
	for _, e := range errs {
		if strings.Contains(e.Message, "undefined") {
			hasUndef = true
			break
		}
	}
	if !hasUndef {
		t.Error("expected undefined ref error, got none")
	}
}

func TestPrettyPrint(t *testing.T) {
	source := `## shiny {
  input param.feature : str required

  # design : llm
    user>
      Design the architecture.
    accept>
      Design doc committed.
  #/

  # implement : llm
    - design
    user>
      Implement the feature.
  #/

  design -> implement
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	output := PrettyPrint(prog)
	if !strings.Contains(output, "## shiny {") {
		t.Error("pretty print should contain molecule header")
	}
	if !strings.Contains(output, "# design : llm") {
		t.Error("pretty print should contain cell definition")
	}
	if !strings.Contains(output, "design -> implement") {
		t.Error("pretty print should contain wire")
	}
	if !strings.Contains(output, "##/") {
		t.Error("pretty print should contain molecule closing")
	}
}

func TestParsePreset(t *testing.T) {
	source := `## review {
  preset gate {
    aspects = [
      { id: "security", focus: "vulnerabilities" }
    ]
  }

  # check : llm
    user>
      Check.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Presets) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(mol.Presets))
	}
	if mol.Presets[0].Name != "gate" {
		t.Errorf("expected preset name 'gate', got %q", mol.Presets[0].Name)
	}
}

func TestParseInputModifiers(t *testing.T) {
	source := `## test {
  input param.pr : number required_unless(param.files, param.branch)
  input param.scope : str default("medium")

  # a : llm
    user>
      Hello.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(mol.Inputs))
	}

	pr := mol.Inputs[0]
	if pr.ParamName != "pr" {
		t.Errorf("expected param name 'pr', got %q", pr.ParamName)
	}
	if len(pr.RequiredUnless) != 2 {
		t.Errorf("expected 2 required_unless refs, got %d", len(pr.RequiredUnless))
	}

	scope := mol.Inputs[1]
	if scope.Default == nil {
		t.Fatal("expected default value on scope")
	}
	if scope.Default.Str != "medium" {
		t.Errorf("expected default 'medium', got %q", scope.Default.Str)
	}
}

func TestParseTestdataFiles(t *testing.T) {
	testdataDir := "testdata"
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Skipf("no testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".cell") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(testdataDir, entry.Name()))
			if err != nil {
				t.Fatalf("read error: %v", err)
			}

			prog, parseErr := Parse(string(data))
			if parseErr != nil {
				t.Fatalf("parse error: %v", parseErr)
			}

			if len(prog.Molecules) == 0 {
				t.Error("expected at least one molecule")
			}

			// Pretty-print round-trip: should not panic
			output := PrettyPrint(prog)
			if len(output) == 0 {
				t.Error("pretty-print produced empty output")
			}
		})
	}
}

func TestParseComment(t *testing.T) {
	source := `-- This is a Cell file
## hello {
  -- A greeting cell
  # greet : llm
    user>
      Say hello.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Molecules) != 1 {
		t.Fatalf("expected 1 molecule, got %d", len(prog.Molecules))
	}
}

func TestParseChainedWires(t *testing.T) {
	source := `## pipeline {
  # a : llm
    user>
      A
  #/
  # b : llm
    user>
      B
  #/
  # c : llm
    user>
      C
  #/
  a -> b -> c
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Wires) != 2 {
		t.Fatalf("expected 2 wires from chain, got %d", len(mol.Wires))
	}

	if mol.Wires[0].From != "a" || mol.Wires[0].To != "b" {
		t.Errorf("wire 0: expected a->b, got %s->%s", mol.Wires[0].From, mol.Wires[0].To)
	}
	if mol.Wires[1].From != "b" || mol.Wires[1].To != "c" {
		t.Errorf("wire 1: expected b->c, got %s->%s", mol.Wires[1].From, mol.Wires[1].To)
	}
}

func TestParsePromptFragment(t *testing.T) {
	source := `prompt@ analyst-persona
  You are a senior equity analyst.
  You cite sources. You flag uncertainty.

## test {
  # a : llm
    user>
      Hello.
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Fragments) != 1 {
		t.Fatalf("expected 1 fragment, got %d", len(prog.Fragments))
	}
	if prog.Fragments[0].Name != "analyst-persona" {
		t.Errorf("expected fragment name 'analyst-persona', got %q", prog.Fragments[0].Name)
	}
}

func TestWireFanOut(t *testing.T) {
	src := `## pipeline
  # source : llm
    user>
      Generate data.
  #/
  # sink1 : llm
    user>
      Process A.
  #/
  # sink2 : llm
    user>
      Process B.
  #/
  # sink3 : llm
    user>
      Process C.
  #/
  source -> [sink1, sink2, sink3]
##/`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Wires) != 1 {
		t.Fatalf("expected 1 wire, got %d", len(mol.Wires))
	}

	w := mol.Wires[0]
	if w.From != "source" {
		t.Errorf("expected From=source, got %q", w.From)
	}
	if len(w.ToList) != 3 {
		t.Fatalf("expected 3 fan-out targets, got %d", len(w.ToList))
	}
	expected := []string{"sink1", "sink2", "sink3"}
	for i, e := range expected {
		if w.ToList[i] != e {
			t.Errorf("ToList[%d]: expected %q, got %q", i, e, w.ToList[i])
		}
	}
}

func TestWireFanIn(t *testing.T) {
	src := `## pipeline
  # a : llm
    user>
      A.
  #/
  # b : llm
    user>
      B.
  #/
  # merge : llm
    user>
      Merge.
  #/
  [a, b] -> merge
##/`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Wires) != 1 {
		t.Fatalf("expected 1 wire, got %d", len(mol.Wires))
	}

	w := mol.Wires[0]
	if len(w.FromList) != 2 {
		t.Fatalf("expected 2 fan-in sources, got %d", len(w.FromList))
	}
	if w.FromList[0] != "a" || w.FromList[1] != "b" {
		t.Errorf("expected FromList=[a, b], got %v", w.FromList)
	}
	if w.To != "merge" {
		t.Errorf("expected To=merge, got %q", w.To)
	}
}

func TestWireChainEndingFanOut(t *testing.T) {
	src := `## pipeline
  # a : llm
    user>
      A.
  #/
  # b : llm
    user>
      B.
  #/
  # c : llm
    user>
      C.
  #/
  # d : llm
    user>
      D.
  #/
  a -> b -> [c, d]
##/`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Wires) != 2 {
		t.Fatalf("expected 2 wires (a->b, b->[c,d]), got %d", len(mol.Wires))
	}

	// First: a -> b
	if mol.Wires[0].From != "a" || mol.Wires[0].To != "b" {
		t.Errorf("wire 0: expected a->b, got %s->%s", mol.Wires[0].From, mol.Wires[0].To)
	}
	// Second: b -> [c, d]
	if mol.Wires[1].From != "b" || len(mol.Wires[1].ToList) != 2 {
		t.Errorf("wire 1: expected b->[c,d], got %s->%v", mol.Wires[1].From, mol.Wires[1].ToList)
	}
}

func TestParseHumanCell(t *testing.T) {
	source := `## deploy {
  # run-tests : script
    ` + "```" + `bash
    echo "tests passed"
    ` + "```" + `
  #/

  # approve : human
    - run-tests
    user>
      Tests passed: {{run-tests}}
      Deploy to production?
    format> approval
      { approved: boolean, reason: str }
  #/

  run-tests -> approve
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	mol := prog.Molecules[0]
	if len(mol.Cells) != 2 {
		t.Fatalf("expected 2 cells, got %d", len(mol.Cells))
	}

	human := mol.Cells[1]
	if human.Name != "approve" {
		t.Errorf("expected cell name 'approve', got %q", human.Name)
	}
	if human.Type.Name != "human" {
		t.Errorf("expected cell type 'human', got %q", human.Type.Name)
	}
	if len(human.Refs) != 1 || human.Refs[0].Name != "run-tests" {
		t.Errorf("expected ref to run-tests, got %v", human.Refs)
	}

	// Should have user> and format> sections
	hasUser := false
	hasFormat := false
	for _, ps := range human.Prompts {
		if ps.Tag == "user" {
			hasUser = true
		}
		if ps.Tag == "format" {
			hasFormat = true
		}
	}
	if !hasUser {
		t.Error("expected user> section on human cell")
	}
	if !hasFormat {
		t.Error("expected format> section on human cell")
	}
}

func TestParseHumanCellRawText(t *testing.T) {
	source := `## feedback {
  # ask : human
    user>
      What changes do you want?
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	human := prog.Molecules[0].Cells[0]
	if human.Type.Name != "human" {
		t.Errorf("expected type 'human', got %q", human.Type.Name)
	}

	// Should validate clean (no format = raw text)
	errs := Validate(prog)
	for _, e := range errs {
		if e.Severity == "error" {
			t.Errorf("unexpected validation error: %s", e.Message)
		}
	}
}

func TestValidateHumanCellMissingUser(t *testing.T) {
	source := `## broken {
  # gate : human
    format> approval
      { approved: boolean }
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "must have a user> section") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for human cell without user>")
	}
}

func TestValidateHumanCellForbiddenSections(t *testing.T) {
	source := `## broken {
  # gate : human
    system>
      You are a gate.
    user>
      Approve?
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "must not have system>") {
			found = true
		}
	}
	if !found {
		t.Error("expected validation error for human cell with system>")
	}
}

func TestValidateFieldRefAgainstFormat(t *testing.T) {
	source := `## test {
  # decide : llm
    user>
      Make a decision.
    format> decision
      { action: str, reason: str }
  #/

  # execute : script
    - decide.action
    ` + "```" + `bash
    echo "doing {{decide.action}}"
    ` + "```" + `
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	// action is declared in format> — no warnings expected
	for _, e := range errs {
		if strings.Contains(e.Message, "not declared in format>") {
			t.Errorf("unexpected field warning: %s", e.Message)
		}
	}
}

func TestValidateFieldRefUnknownField(t *testing.T) {
	source := `## test {
  # decide : llm
    user>
      Make a decision.
    format> decision
      { action: str, reason: str }
  #/

  # execute : llm
    - decide.typo
    user>
      Do {{decide.bogus}}
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	errs := Validate(prog)
	warnings := 0
	for _, e := range errs {
		if strings.Contains(e.Message, "not declared in format>") {
			warnings++
		}
	}
	if warnings < 2 {
		t.Errorf("expected at least 2 field warnings (ref decl + inline), got %d", warnings)
	}
}

func TestReduceBoundedLoop(t *testing.T) {
	source := `## retry-test {
  reduce # attempt : llm over 3 as i with acc = "none"
    user>
      Attempt {{i}}, previous: {{acc}}
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(prog.Molecules) != 1 {
		t.Fatalf("expected 1 molecule, got %d", len(prog.Molecules))
	}
	mol := prog.Molecules[0]
	if len(mol.ReduceCells) != 1 {
		t.Fatalf("expected 1 reduce cell, got %d", len(mol.ReduceCells))
	}

	rc := mol.ReduceCells[0]
	if rc.TimesN != 3 {
		t.Errorf("TimesN = %d, want 3", rc.TimesN)
	}
	if rc.AsIdent != "i" {
		t.Errorf("AsIdent = %q, want %q", rc.AsIdent, "i")
	}
	if rc.AccIdent != "acc" {
		t.Errorf("AccIdent = %q, want %q", rc.AccIdent, "acc")
	}
	if rc.AccDefault.Str != "none" {
		t.Errorf("AccDefault = %q, want %q", rc.AccDefault.Str, "none")
	}
	if rc.OverRef != "" {
		t.Errorf("OverRef should be empty for bounded loop, got %q", rc.OverRef)
	}
}

func TestReduceUntilGuard(t *testing.T) {
	source := `## retry-test {
  reduce # attempt : llm over 5 as i with acc = "none" until(done)
    user>
      Attempt {{i}}, previous: {{acc}}
  #/
##/`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	rc := prog.Molecules[0].ReduceCells[0]
	if rc.TimesN != 5 {
		t.Errorf("TimesN = %d, want 5", rc.TimesN)
	}
	if rc.UntilField != "done" {
		t.Errorf("UntilField = %q, want %q", rc.UntilField, "done")
	}
}
