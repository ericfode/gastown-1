package cell

import (
	"strings"
	"testing"
)

func TestParseSingleCell(t *testing.T) {
	input := `
cell find-bugs {
    type: inventory
    prompt: "Find all bugs in {{source-code}}"
    refs: [source-code]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(f.Cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(f.Cells))
	}
	c := f.Cells[0]
	if c.Name != "find-bugs" {
		t.Errorf("name = %q, want %q", c.Name, "find-bugs")
	}
	if c.Type != "inventory" {
		t.Errorf("type = %q, want %q", c.Type, "inventory")
	}
	if c.Prompt != "Find all bugs in {{source-code}}" {
		t.Errorf("prompt = %q, want %q", c.Prompt, "Find all bugs in {{source-code}}")
	}
	if len(c.Refs) != 1 || c.Refs[0] != "source-code" {
		t.Errorf("refs = %v, want [source-code]", c.Refs)
	}
}

func TestParseMultipleCells(t *testing.T) {
	input := `
cell source-code {
    type: text
    prompt: "Read the source files"
}

cell find-bugs {
    type: inventory
    prompt: "Find bugs in {{source-code}}"
    refs: [source-code]
    oracle: bug-validator
}

cell bug-validator {
    type: code
    prompt: "Check that {{find-bugs}} contains valid JSON"
    refs: [find-bugs]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(f.Cells) != 3 {
		t.Fatalf("expected 3 cells, got %d", len(f.Cells))
	}
	if f.Cells[1].Oracle != "bug-validator" {
		t.Errorf("oracle = %q, want %q", f.Cells[1].Oracle, "bug-validator")
	}
}

func TestParseTripleQuotedString(t *testing.T) {
	input := `
cell analysis {
    type: synthesis
    prompt: """
        Analyze the following data:
        - Source: {{data-source}}
        - Context: {{context}}
        Produce a summary.
    """
    refs: [data-source, context]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	c := f.Cells[0]
	if !strings.Contains(c.Prompt, "Analyze the following data:") {
		t.Errorf("prompt missing expected content: %q", c.Prompt)
	}
	if !strings.Contains(c.Prompt, "{{data-source}}") {
		t.Errorf("prompt missing template ref: %q", c.Prompt)
	}
}

func TestParseRecipe(t *testing.T) {
	input := `
recipe enrich(target, source_prompt, refined_prompt) {
    src = addCell({ prompt: source_prompt })
    addRef(target, src)
    refinePrompt(target, refined_prompt)
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(f.Recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(f.Recipes))
	}
	r := f.Recipes[0]
	if r.Name != "enrich" {
		t.Errorf("name = %q, want %q", r.Name, "enrich")
	}
	if len(r.Params) != 3 {
		t.Errorf("params = %v, want 3 params", r.Params)
	}
	if len(r.Body) != 3 {
		t.Fatalf("body has %d statements, want 3", len(r.Body))
	}
	// First statement is an assignment
	if r.Body[0].Assignment == nil {
		t.Error("first statement should be an assignment")
	}
	if r.Body[0].Assignment.Name != "src" {
		t.Errorf("assignment name = %q, want %q", r.Body[0].Assignment.Name, "src")
	}
	// Second and third are bare calls
	if r.Body[1].Call == nil {
		t.Error("second statement should be a call")
	}
	if r.Body[1].Call.Name != "addRef" {
		t.Errorf("call name = %q, want %q", r.Body[1].Call.Name, "addRef")
	}
}

func TestParseComments(t *testing.T) {
	input := `
# This is a comment
cell source {
    # Another comment
    type: text
    prompt: "Read source"
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(f.Cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(f.Cells))
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	input := `
cell foo {
    type: text
    prompt: "a"
}
cell foo {
    type: text
    prompt: "b"
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for duplicate names")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "duplicate cell name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'duplicate cell name' error, got: %v", errs)
	}
}

func TestValidateUnknownType(t *testing.T) {
	input := `
cell foo {
    type: invalid_type
    prompt: "a"
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation error for unknown type")
	}
	if !strings.Contains(errs[0].Message, "unknown cell type") {
		t.Errorf("expected 'unknown cell type' error, got: %s", errs[0].Message)
	}
}

func TestValidateUnresolvedRef(t *testing.T) {
	input := `
cell foo {
    type: text
    prompt: "a"
    refs: [nonexistent]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation error for unresolved ref")
	}
	if !strings.Contains(errs[0].Message, "refs unknown cell") {
		t.Errorf("expected 'refs unknown cell' error, got: %s", errs[0].Message)
	}
}

func TestValidateUnresolvedOracle(t *testing.T) {
	input := `
cell foo {
    type: text
    prompt: "a"
    oracle: ghost
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation error for unresolved oracle")
	}
	if !strings.Contains(errs[0].Message, "oracle references unknown cell") {
		t.Errorf("expected 'oracle references unknown cell' error, got: %s", errs[0].Message)
	}
}

func TestValidateCycleDetection(t *testing.T) {
	input := `
cell a {
    type: text
    prompt: "{{b}}"
    refs: [b]
}
cell b {
    type: text
    prompt: "{{c}}"
    refs: [c]
}
cell c {
    type: text
    prompt: "{{a}}"
    refs: [a]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation error for cycle")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "cycle") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cycle error, got: %v", errs)
	}
}

func TestValidatePromptRefConsistency(t *testing.T) {
	input := `
cell source {
    type: text
    prompt: "Read source"
}
cell analysis {
    type: synthesis
    prompt: "Analyze {{source}} and {{extra}}"
    refs: [source]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "not in refs") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected prompt ref consistency error, got: %v", errs)
	}
}

func TestValidateRecipeUnknownOp(t *testing.T) {
	input := `
recipe bad(target) {
    frobulate(target)
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) == 0 {
		t.Fatal("expected validation error for unknown operation")
	}
	if !strings.Contains(errs[0].Message, "unknown operation") {
		t.Errorf("expected 'unknown operation' error, got: %s", errs[0].Message)
	}
}

func TestValidateCleanFile(t *testing.T) {
	input := `
cell source-code {
    type: text
    prompt: "Read the source files"
}

cell find-bugs {
    type: inventory
    prompt: "Find bugs in {{source-code}}"
    refs: [source-code]
    oracle: bug-validator
}

cell bug-validator {
    type: code
    prompt: "Check that {{find-bugs}} output is valid"
    refs: [find-bugs]
}

recipe enrich(target, source_prompt, refined_prompt) {
    src = addCell({ prompt: source_prompt })
    addRef(target, src)
    refinePrompt(target, refined_prompt)
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	errs := Validate(f)
	if len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestPrettyPrintRoundTrip(t *testing.T) {
	input := `
cell source-code {
    type: text
    prompt: "Read the source files"
}

cell find-bugs {
    type: inventory
    prompt: "Find bugs in {{source-code}}"
    refs: [source-code]
    oracle: bug-validator
}

cell bug-validator {
    type: code
    prompt: "Check output"
    refs: [find-bugs]
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	output := PrettyPrint(f)

	// Parse the pretty-printed output again
	f2, err := Parse(output, "roundtrip.cell")
	if err != nil {
		t.Fatalf("round-trip parse error: %v\nOutput was:\n%s", err, output)
	}
	if len(f2.Cells) != len(f.Cells) {
		t.Errorf("round-trip cell count: got %d, want %d", len(f2.Cells), len(f.Cells))
	}
	for i, c := range f.Cells {
		c2 := f2.Cells[i]
		if c.Name != c2.Name {
			t.Errorf("cell %d name: got %q, want %q", i, c2.Name, c.Name)
		}
		if c.Type != c2.Type {
			t.Errorf("cell %d type: got %q, want %q", i, c2.Type, c.Type)
		}
	}
}

func TestLexerErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unexpected character",
			input: "cell foo @",
			want:  "unexpected character",
		},
		{
			name:  "unterminated string",
			input: `cell foo { prompt: "unterminated`,
			want:  "unterminated string",
		},
		{
			name:  "unterminated triple string",
			input: `cell foo { prompt: """unterminated`,
			want:  "unterminated triple-quoted string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input, "test.cell")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseRecipeWithListArg(t *testing.T) {
	input := `
recipe split_review(cell) {
    splitCell(cell, [code-review, security-review])
}
`
	f, err := Parse(input, "test.cell")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(f.Recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(f.Recipes))
	}
	stmt := f.Recipes[0].Body[0]
	if stmt.Call == nil {
		t.Fatal("expected call statement")
	}
	if len(stmt.Call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(stmt.Call.Args))
	}
	listArg := stmt.Call.Args[1]
	if len(listArg.List) != 2 {
		t.Fatalf("expected list with 2 items, got %d", len(listArg.List))
	}
}

func TestExtractPromptRefs(t *testing.T) {
	tests := []struct {
		prompt string
		want   []string
	}{
		{"no refs here", nil},
		{"{{foo}}", []string{"foo"}},
		{"{{foo}} and {{bar}}", []string{"foo", "bar"}},
		{"{{foo.field}}", []string{"foo.field"}},
		{"text {{a}} more {{b.x}} end", []string{"a", "b.x"}},
	}

	for _, tt := range tests {
		got := extractPromptRefs(tt.prompt)
		if len(got) != len(tt.want) {
			t.Errorf("extractPromptRefs(%q) = %v, want %v", tt.prompt, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractPromptRefs(%q)[%d] = %q, want %q", tt.prompt, i, got[i], tt.want[i])
			}
		}
	}
}
