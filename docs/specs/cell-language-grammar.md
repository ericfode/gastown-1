# Cell Language — Formal Grammar (CFG)

**Bead**: hq-98i
**Date**: 2026-03-09
**Status**: Draft
**Parent**: hq-7vk (Epic: Cell Language — Formula Engine v2 DSL)

---

## Overview

Cell is a context-free DSL for defining reactive bead computation graphs. It has
two concrete surface syntaxes (TOML formulas and YAML reactive sheets) that share
a common abstract syntax. This document provides:

1. **Abstract syntax** in EBNF (the canonical grammar)
2. **TOML surface syntax** mapping (how the abstract syntax serializes to `.formula.toml`)
3. **YAML surface syntax** mapping (how it serializes to reactive sheet `.yaml`)
4. **Lexer specification**
5. **Context-freeness proof**

The grammar covers all four formula types (convoy, workflow, expansion, aspect),
the reactive sheet format, the recipe sub-language, and the graph operation
primitives.

---

## 1. Abstract Syntax (EBNF)

The canonical grammar for Cell. Every production has a single non-terminal on the
left-hand side.

### 1.1 Top-Level

```ebnf
(* A Cell program is either a Formula or a Sheet *)
Program         = Formula | Sheet ;

(* === Formula (TOML surface) === *)
Formula         = FormulaHeader FormulaBody ;
FormulaHeader   = Description FormulaName FormulaType Version [ Pour ] [ Agent ] ;
Description     = MULTILINE_STRING ;
FormulaName     = IDENT ;
FormulaType     = "convoy" | "workflow" | "expansion" | "aspect" ;
Version         = NATURAL ;
Pour            = BOOLEAN ;
Agent           = IDENT ;

FormulaBody     = [ Inputs ] [ Vars ] [ Prompts ] [ Output ]
                  ( ConvoyBody | WorkflowBody | ExpansionBody | AspectBody ) ;

(* === Sheet (YAML surface) === *)
Sheet           = SheetName CellDefList ;
SheetName       = IDENT ;
CellDefList     = CellDef { CellDef } ;
```

### 1.2 Formula Bodies (by type)

```ebnf
(* Convoy: parallel legs + synthesis *)
ConvoyBody      = LegList [ Synthesis ] ;
LegList         = Leg { Leg } ;
Leg             = LegID Title Focus [ LegDescription ] [ Agent ] ;
LegID           = IDENT ;
Synthesis       = Title [ SynthDescription ] DependsOn ;
DependsOn       = LegID { LegID } ;

(* Workflow: sequential steps with dependencies *)
WorkflowBody    = StepList ;
StepList        = Step { Step } ;
Step            = StepID Title StepDescription [ Needs ] [ Parallel ] [ Acceptance ] ;
StepID          = IDENT ;
Needs           = StepID { StepID } ;
Parallel        = BOOLEAN ;
Acceptance      = MULTILINE_STRING ;

(* Expansion: template-based step generation *)
ExpansionBody   = TemplateList ;
TemplateList    = TemplateStep { TemplateStep } ;
TemplateStep    = TemplateID Title TemplateDescription [ Needs ] ;
TemplateID      = IDENT ;

(* Aspect: multi-aspect parallel analysis *)
AspectBody      = AspectList ;
AspectList      = Aspect { Aspect } ;
Aspect          = AspectID Title Focus [ AspectDescription ] ;
AspectID        = IDENT ;
```

### 1.3 Inputs, Variables, Prompts, Output

```ebnf
Inputs          = InputDef { InputDef } ;
InputDef        = InputName InputSpec ;
InputName       = IDENT ;
InputSpec       = InputDescription InputType [ Required ] [ RequiredUnless ] [ Default ] ;
InputDescription = STRING ;
InputType       = "string" | "number" | "boolean" ;
Required        = BOOLEAN ;
RequiredUnless  = IDENT { IDENT } ;
Default         = STRING ;

Vars            = VarDef { VarDef } ;
VarDef          = VarName ( STRING | VarSpec ) ;
VarName         = IDENT ;
VarSpec         = [ VarDescription ] [ VarRequired ] [ VarDefault ] ;
VarDescription  = STRING ;
VarRequired     = BOOLEAN ;
VarDefault      = STRING ;

Prompts         = PromptDef { PromptDef } ;
PromptDef       = PromptName PromptTemplate ;
PromptName      = IDENT ;
PromptTemplate  = MULTILINE_STRING ;

Output          = OutputDirectory [ LegPattern ] [ SynthesisOutput ] ;
OutputDirectory = STRING ;
LegPattern      = STRING ;
SynthesisOutput = STRING ;
```

### 1.4 Cell Definitions (Reactive Sheet)

```ebnf
CellDef         = CellName CellType CellPrompt [ RefList ] ;
CellName        = IDENT ;
CellType        = "text" | "inventory" | "diagram" | "laws"
                | "boundaries" | "synthesis" | "code" | "decision" ;
CellPrompt      = PromptTemplate ;
RefList         = Ref { Ref } ;
Ref             = CellName [ "." FieldName ] ;
FieldName       = IDENT ;
```

### 1.5 Prompt Templates

```ebnf
(* Prompt templates contain literal text interspersed with references *)
PromptTemplate  = { PromptFragment } ;
PromptFragment  = LITERAL_TEXT | TemplateRef | GoTemplateExpr ;

(* Simple Cell references: {{cellName}} or {{cellName.field}} *)
TemplateRef     = "{{" RefPath "}}" ;
RefPath         = IDENT [ "." IDENT ] ;

(* Go text/template expressions (convoy formulas) *)
GoTemplateExpr  = "{{" GoExpr "}}" ;
GoExpr          = GoIdent { "." GoIdent }
                | "if" GoExpr GoBlock [ "else" GoBlock ] "end"
                | "range" GoExpr GoBlock "end"
                | GoIdent GoExpr { GoExpr } ;
GoIdent         = "." IDENT | IDENT ;
GoBlock         = { PromptFragment } ;
```

### 1.6 Typed DAG (Abstract Semantic Layer)

These productions describe the abstract typed DAG that both surface syntaxes
desugar into. This is the semantic core from the Lean4 formalization.

```ebnf
(* Typed DAG — the semantic representation *)
TypedDAG        = DAGName CellSpecList WireList ;
DAGName         = IDENT ;

CellSpecList    = CellSpec { CellSpec } ;
CellSpec        = CellName CellSig CellPrompt ;
CellSig         = InputPorts OutputPort ;
InputPorts      = { Port } ;
OutputPort      = Port ;
Port            = PortName CellType ;
PortName        = IDENT ;

WireList        = { Wire } ;
Wire            = SourceCell SourcePort TargetCell TargetPort ;
SourceCell      = IDENT ;
SourcePort      = IDENT ;
TargetCell      = IDENT ;
TargetPort      = IDENT ;
```

### 1.7 Graph Operations

The eight primitive operations that transform a molecule's structure.

```ebnf
Operation       = AddCell | RemoveCell | AddRef | RemoveRef
                | SplitCell | MergeCell | RefinePrompt | SeedValue ;

(* Minimal basis — can express any DAG-to-DAG transformation *)
AddCell         = "addCell" "(" CellSpec ")" ;
RemoveCell      = "removeCell" "(" CellName ")" ;
AddRef          = "addRef" "(" CellName "," Ref ")" ;
RemoveRef       = "removeRef" "(" CellName "," Ref ")" ;

(* Semantic operations — derivable but preserve wiring invariants *)
SplitCell       = "splitCell" "(" CellName "," "[" CellSpec { "," CellSpec } "]" ")" ;
MergeCell       = "mergeCell" "(" "[" CellName { "," CellName } "]" "," CellSpec ")" ;
RefinePrompt    = "refinePrompt" "(" CellName "," PromptTemplate ")" ;
SeedValue       = "seedValue" "(" CellName "," STRING ")" ;

OperationList   = Operation { Operation } ;
```

### 1.8 Recipes

Named, parameterized sequences of graph operations.

```ebnf
Recipe          = "recipe" RecipeName "(" ParamList ")" ":" RecipeBody ;
RecipeName      = IDENT ;
ParamList       = [ Param { "," Param } ] ;
Param           = IDENT ;
RecipeBody      = { RecipeStatement } ;
RecipeStatement = Assignment | OperationCall ;
Assignment      = IDENT "=" OperationCall ;
OperationCall   = Operation
                | RecipeName "(" ArgList ")" ;
ArgList         = [ Expr { "," Expr } ] ;
Expr            = IDENT | STRING | CellSpec ;
```

### 1.9 Oracle Verdicts

```ebnf
OracleVerdict   = "accept"
                | "score" "(" QualityLevel ")"
                | "redirect" "(" CellName ")"
                | "reject" "(" STRING ")" ;
QualityLevel    = IDENT ;    (* e.g., draft, adequate, good, excellent *)
```

### 1.10 Cell State Machine

```ebnf
CellState       = "empty" | "stale" | "computing" | "fresh" | "failed" ;

(* Valid state transitions *)
StateTransition = "empty"     "->" "computing"
                | "stale"     "->" "computing"
                | "computing" "->" "fresh"
                | "computing" "->" "failed"
                | "fresh"     "->" "stale"
                | "failed"    "->" "computing" ;
```

### 1.11 Matter Model

```ebnf
Phase           = "proto" | "liquid" | "crystal" ;

(* Phase transitions *)
PhaseTransition = "proto"   "->" "liquid"       (* pour *)
                | "liquid"  "->" "crystal"      (* squash/distill *)
                | "crystal" "->" "liquid" ;     (* re-liquify *)
```

---

## 2. Lexer Specification

### 2.1 Character Classes

```
LETTER          = [a-zA-Z_]
DIGIT           = [0-9]
HEX_DIGIT       = [0-9a-fA-F]
WHITESPACE      = [ \t\r\n]+
```

### 2.2 Token Types

```
IDENT           = LETTER ( LETTER | DIGIT | '-' )*
                  (* Identifiers: cell names, step IDs, variable names.
                     Hyphens allowed for kebab-case (e.g., "blast-radius"). *)

NATURAL         = DIGIT+
                  (* Non-negative integers: version numbers, priorities. *)

STRING          = '"' ( [^"\\] | ESCAPE )* '"'
                | "'" ( [^'\\] | ESCAPE )* "'"
                  (* Single-line strings. *)

MULTILINE_STRING = '"""' .* '"""'
                 | "'''" .* "'''"
                 | '|' INDENT NEWLINE ( INDENT .* NEWLINE )*
                  (* Multi-line strings. Triple-quote (TOML) or block scalar (YAML). *)

BOOLEAN         = "true" | "false"

ESCAPE          = '\\' ( '"' | "'" | '\\' | 'n' | 'r' | 't' | 'u' HEX_DIGIT{4} )

COMMENT         = '#' [^\n]*
                  (* Line comments (TOML style). *)

YAML_COMMENT    = '#' [^\n]*
                  (* Also line comments (YAML style). *)
```

### 2.3 Template Tokens (within prompt strings)

```
TEMPLATE_OPEN   = '{{'
TEMPLATE_CLOSE  = '}}'
LITERAL_TEXT     = ( [^{] | '{' [^{] )+
                  (* Any text not starting a template expression. *)
REF_PATH        = IDENT ( '.' IDENT )?
                  (* Cell reference within {{ }}. *)
```

### 2.4 Punctuation and Delimiters

```
LBRACKET        = '['
RBRACKET        = ']'
LPAREN          = '('
RPAREN          = ')'
LBRACE          = '{'
RBRACE          = '}'
COMMA           = ','
DOT             = '.'
COLON           = ':'
EQUALS          = '='
ARROW           = '->'
DOUBLE_BRACKET  = '[[' | ']]'    (* TOML array-of-tables *)
```

### 2.5 Keywords

```
(* Formula types *)
KW_CONVOY       = "convoy"
KW_WORKFLOW     = "workflow"
KW_EXPANSION    = "expansion"
KW_ASPECT       = "aspect"

(* Cell types *)
KW_TEXT         = "text"
KW_INVENTORY    = "inventory"
KW_DIAGRAM      = "diagram"
KW_LAWS         = "laws"
KW_BOUNDARIES   = "boundaries"
KW_SYNTHESIS    = "synthesis"
KW_CODE         = "code"
KW_DECISION     = "decision"

(* Cell states *)
KW_EMPTY        = "empty"
KW_STALE        = "stale"
KW_COMPUTING    = "computing"
KW_FRESH        = "fresh"
KW_FAILED       = "failed"

(* Graph operations *)
KW_ADDCELL      = "addCell"
KW_REMOVECELL   = "removeCell"
KW_ADDREF       = "addRef"
KW_REMOVEREF    = "removeRef"
KW_SPLITCELL    = "splitCell"
KW_MERGECELL    = "mergeCell"
KW_REFINEPROMPT = "refinePrompt"
KW_SEEDVALUE    = "seedValue"

(* Recipe language *)
KW_RECIPE       = "recipe"

(* Oracle verdicts *)
KW_ACCEPT       = "accept"
KW_SCORE        = "score"
KW_REDIRECT     = "redirect"
KW_REJECT       = "reject"

(* Phases *)
KW_PROTO        = "proto"
KW_LIQUID       = "liquid"
KW_CRYSTAL      = "crystal"

(* Boolean *)
KW_TRUE         = "true"
KW_FALSE        = "false"
```

### 2.6 Lexer Modes

The lexer operates in three modes:

| Mode | Active when | Tokens produced |
|------|-------------|-----------------|
| **Normal** | Default | All standard tokens |
| **Template** | Inside prompt strings | `LITERAL_TEXT`, `TEMPLATE_OPEN`, `TEMPLATE_CLOSE`, `REF_PATH` |
| **GoTemplate** | Inside `{{` in convoy prompts | Go template expression tokens |

Mode transitions:
- Normal → Template: on entering a prompt string value
- Template → GoTemplate: on encountering `{{` with Go syntax (`.` prefix)
- GoTemplate → Template: on encountering `}}`
- Template → Normal: on exiting the prompt string

---

## 3. TOML Surface Syntax Mapping

How the abstract grammar maps to `.formula.toml` files.

| Abstract | TOML |
|----------|------|
| `FormulaHeader` | Top-level keys: `description`, `formula`, `type`, `version`, `pour`, `agent` |
| `Inputs` | `[inputs]` table with `[inputs.<name>]` sub-tables |
| `Vars` | `[vars]` table (shorthand string or `[vars.<name>]` sub-table) |
| `Prompts` | `[prompts]` table with named template strings |
| `Output` | `[output]` table |
| `Leg` | `[[leg]]` array-of-tables |
| `Step` | `[[steps]]` array-of-tables |
| `TemplateStep` | `[[template]]` array-of-tables |
| `Aspect` | `[[aspects]]` array-of-tables |
| `Synthesis` | `[synthesis]` table |
| `Needs` | `needs = ["step-1", "step-2"]` array |
| `DependsOn` | `depends_on = ["leg-1", "leg-2"]` array |

### Example (workflow)

```toml
description = """Multi-step workflow example."""
formula = "example-workflow"
type = "workflow"
version = 1

[vars]
target = "default-value"

[[steps]]
id = "analyze"
title = "Analyze the target"
description = """Examine {{target}} for issues."""

[[steps]]
id = "fix"
title = "Apply fixes"
needs = ["analyze"]
description = """Fix issues found in {{target}}."""
```

---

## 4. YAML Surface Syntax Mapping

How the abstract grammar maps to reactive sheet `.yaml` files.

| Abstract | YAML |
|----------|------|
| `SheetName` | `name:` key |
| `CellDefList` | `cells:` sequence |
| `CellDef` | Mapping with `name`, `type`, `prompt`, `refs` |
| `CellType` | `type:` string value |
| `CellPrompt` | `prompt:` block scalar |
| `RefList` | `refs:` sequence of strings |
| `Ref` | String: `"cellName"` or `"cellName.field"` |
| `TemplateRef` | `{{cellName}}` or `{{cellName.field}}` within prompt text |

### Example (reactive sheet)

```yaml
name: example-sheet
cells:
  - name: source
    type: text
    prompt: |
      Gather data about the topic.

  - name: analysis
    type: synthesis
    refs: [source]
    prompt: |
      Analyze the following data:
      {{source}}

  - name: verdict
    type: decision
    refs: [analysis]
    prompt: |
      Based on the analysis, decide whether to proceed.
      {{analysis}}
```

---

## 5. Context-Freeness Proof

**Claim**: The Cell language grammar defined in Section 1 is context-free.

**Proof**: A grammar is context-free if and only if every production rule has the
form `A → α` where `A` is a single non-terminal and `α` is a (possibly empty)
string of terminals and non-terminals.

Inspection of every production in Section 1:

| Production | LHS | Form |
|------------|-----|------|
| `Program = Formula \| Sheet` | `Program` (single NT) | `A → α` |
| `Formula = FormulaHeader FormulaBody` | `Formula` (single NT) | `A → α` |
| `FormulaHeader = Description FormulaName ...` | `FormulaHeader` (single NT) | `A → α` |
| `FormulaType = "convoy" \| "workflow" \| ...` | `FormulaType` (single NT) | `A → α` |
| `FormulaBody = [ Inputs ] [ Vars ] ...` | `FormulaBody` (single NT) | `A → α` |
| `Sheet = SheetName CellDefList` | `Sheet` (single NT) | `A → α` |
| `CellDef = CellName CellType CellPrompt [ RefList ]` | `CellDef` (single NT) | `A → α` |
| `CellType = "text" \| "inventory" \| ...` | `CellType` (single NT) | `A → α` |
| `Ref = CellName [ "." FieldName ]` | `Ref` (single NT) | `A → α` |
| `TemplateRef = "{{" RefPath "}}"` | `TemplateRef` (single NT) | `A → α` |
| `Operation = AddCell \| RemoveCell \| ...` | `Operation` (single NT) | `A → α` |
| `AddCell = "addCell" "(" CellSpec ")"` | `AddCell` (single NT) | `A → α` |
| `Recipe = "recipe" RecipeName ...` | `Recipe` (single NT) | `A → α` |
| `OracleVerdict = "accept" \| ...` | `OracleVerdict` (single NT) | `A → α` |
| `CellState = "empty" \| "stale" \| ...` | `CellState` (single NT) | `A → α` |
| *(all remaining productions follow the same pattern)* | | |

Every production has exactly one non-terminal on the left-hand side. No
production conditions on context (no `αAβ → αγβ` rules). Therefore the grammar
is context-free by definition.

**Note on TOML/YAML hosting**: The Cell language is embedded in TOML and YAML,
which are themselves context-free. TOML's grammar is specified in
[ABNF](https://toml.io/en/v1.0.0) and YAML's in its
[production rules](https://yaml.org/spec/1.2.2/). The Cell grammar is a
context-free refinement of the host format's value space — it constrains which
TOML/YAML documents are valid Cell programs without introducing context
sensitivity. Template references (`{{ref}}`) within strings are parsed by a
separate lexer mode (Section 2.6) that is also context-free.

**Note on Go templates**: The `GoTemplateExpr` production (Section 1.5) covers
the subset of Go `text/template` syntax used in convoy formula prompts. The full
Go template language includes features (custom functions, pipeline chaining) that
extend beyond this grammar. In practice, Cell formulas use only simple field
access and conditionals, which remain context-free.

---

## 6. Grammar Properties

### 6.1 Ambiguity

The grammar is unambiguous under the following disambiguation rules:

1. **Formula vs Sheet**: Determined by the presence of `formula =` (TOML) vs
   `name:` + `cells:` (YAML) at the top level.
2. **Formula type**: The `type =` field selects exactly one of the four body
   alternatives (convoy, workflow, expansion, aspect).
3. **TemplateRef vs GoTemplateExpr**: Distinguished by whether the content after
   `{{` starts with `.` (Go template) or is a bare identifier (Cell ref).

### 6.2 Grammar Class

The grammar is **LL(1)** for the abstract syntax with the disambiguation rules
above. The TOML and YAML surface syntaxes are parsed by their respective standard
parsers, with the Cell grammar applied as a validation pass over the parsed
document tree.

### 6.3 Relationship to Lean4 Formalization

The abstract syntax (Section 1.6) corresponds directly to the Lean4 types:

| EBNF | Lean4 | File |
|------|-------|------|
| `TypedDAG` | `Formula` | `BeadCalculus/Formula.lean` |
| `CellSpec` | `Cell` | `BeadCalculus/Formula.lean` |
| `CellSig` | `CellSig` | `BeadCalculus/CellType.lean` |
| `Port` | `Port` | `BeadCalculus/CellType.lean` |
| `CellType` | `CellType` | `BeadCalculus/CellType.lean` |
| `Wire` | `Wire` | `BeadCalculus/Formula.lean` |
| `Ref` | `CellRef` | `BeadCalculus/Spreadsheet.lean` |
| `CellState` | (enum in `Unified.lean`) | `BeadCalculus/Unified.lean` |

The Lean4 formalization proves properties (DAG acyclicity, readiness monotonicity,
well-typedness) that the grammar itself does not enforce. The grammar specifies
syntax; the Lean4 code specifies semantics.
