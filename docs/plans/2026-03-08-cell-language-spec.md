# Cell Language Specification

**Date**: 2026-03-08
**Status**: Draft
**Epic**: hq-7vk (Cell Language — Formula Engine v2 DSL)
**Bead**: hq-606

---

## 1. What Cell Is

Cell is a context-free language for describing reactive computation graphs
where cells are agents, wires are typed data flows, and oracles gate quality.
It is the formula language for Gas City — a superset of Gas Town's current
TOML formula system.

Cell replaces three formula types (workflow, convoy, expansion) with one
language. A workflow is cells wired sequentially. A convoy is `map` +
synthesis. An expansion is `map` with template parameters.

### Design Constraints

1. **Superset of TOML formulas.** Every existing formula expressible in Cell.
2. **Context-free grammar.** Parseable by regex, LLMs, and proper parsers.
3. **LLM-writable.** Metacircular cells emit Cell source as their output.
4. **Prompt-native.** Structured prompt sections, not string templates.
5. **Content-addressed.** Cells identified by hash of definition.

---

## 2. Lexical Structure

```
IDENT       = [a-zA-Z_][a-zA-Z0-9_-]*
HASH        = "#" [0-9a-f]{8,64}
NUMBER      = [0-9]+ ("." [0-9]+)?
STRING      = '"' (escaped_char | [^"\\])* '"'
COMMENT     = "--" [^\n]*
SECTION_TAG = "system" | "context" | "user" | "think" | "examples" | "format" | "accept" | "each"
PROMPT_LINE = SECTION_TAG ">" REST_OF_LINE
REF         = "{{" IDENT ("." IDENT)* ("|" FILTER)* (":" ORACLE_EXPR)? "}}"
PARAM_REF   = "{{" "param." IDENT "}}"
FRAG_REF    = "{{" "@" IDENT "}}"
```

Whitespace is insignificant except inside strings and prompt lines.
Comments extend to end of line.

### Lexer Modes

The grammar is context-free. The **lexer** is modal — it tracks three modes
to disambiguate tokens that look identical in different contexts. This is
analogous to Python's INDENT/DEDENT handling: the grammar doesn't change,
but the scanner needs state.

**Mode transitions:**

```
NORMAL ──── SECTION_TAG ───→ PROMPT
NORMAL ──── distill> ──────→ BLOCK (ends at next cell-level token, not ```)
NORMAL ──── ```lang ───────→ BLOCK (ends at ```)
PROMPT ──── (outdent) ─────→ NORMAL
PROMPT ──── ```lang ───────→ BLOCK
BLOCK  ──── ``` ───────────→ (previous mode)
BLOCK  ──── (cell-level) ──→ NORMAL  (for distill> blocks only)
```

**Mode NORMAL** (default):
- `-- ...` → COMMENT
- `## IDENT` → MOL_OPEN
- `##/` → MOL_CLOSE
- `# IDENT : type` → CELL_OPEN (also `map #`, `reduce #`, `meta #`)
- `#/` → CELL_CLOSE (also `meta #/`)
- `- IDENT` → REF_DECL (dependency)
- `input param.IDENT ...` → INPUT_DECL
- `@ IDENT(...)` → ANNOTATION
- `{{ ... }}` → REF
- `IDENT -> IDENT` → WIRE
- `!verb ...` → OPERATION
- `squash> ...` → SQUASH
- `distill>` → DISTILL_OPEN (enters BLOCK mode; ends at next cell-level token)
- `format> IDENT` → FORMAT_TAG (note: not SECTION_TAG)
- `system>` `context>` `user>` etc → SECTION_TAG (enters PROMPT mode)
- `` ```lang `` → SCRIPT_OPEN or ORACLE_OPEN (enters BLOCK mode)

**Mode PROMPT** (after SECTION_TAG, inside prompt content):
- Indented lines → PROMPT_LINE (all content, including `- text`, `{{ refs }}`)
- Outdent to cell/molecule level → exit to NORMAL
- `` ```lang `` → enters BLOCK mode (nested code example in prompt)
- Another SECTION_TAG at cell indent level → exit PROMPT, enter new PROMPT

**Mode BLOCK** (inside ``` ... ``` or distill> delimiters):
- All lines → BODY_LINE (opaque content)
- `` ``` `` at same indent → BLOCK_CLOSE, return to previous mode
- For `distill>` blocks: cell-level token (oracle, cell close, section) → exit BLOCK
- `distill>` blocks have **no explicit closer** — they end implicitly

**Indent tracking**: The lexer tracks the indent depth of the containing
cell. Lines indented deeper than the cell body are content (PROMPT_LINE
or BODY_LINE). Lines at cell indent level or less are structure tokens.

**Why this matters for distillation**: Each mode is an independent
distillation domain. NORMAL mode token rules, PROMPT mode rules, and
BLOCK mode rules can crystallize separately. The mode transitions
themselves are simple enough to distill immediately.

---

## 3. Grammar (EBNF)

```ebnf
program       = { molecule | recipe | prompt_frag | oracle_decl | input_decl } ;

(* === Top-level constructs === *)

molecule      = "##" IDENT "{" mol_body "##/" ;
mol_body      = { cell | map_cell | reduce_cell | wire | preset
                | input_decl | prompt_frag | oracle_decl
                | import_decl | apply_stmt | squash_decl | COMMENT } ;

import_decl   = "import" IDENT ;
apply_stmt    = "apply" IDENT "(" ident_list ")" [ "where" selector_expr ] ;
selector_expr = selector_pred { "and" selector_pred } ;
selector_pred = "type" "==" cell_type
              | "depth" CMP NUMBER
              | "tag" "==" STRING
              | "name" "==" STRING
              | "name" "~" STRING ;   (* glob/regex match — BUG-004 *)

recipe        = "recipe" IDENT "(" param_list ")" "{" { operation } "}" ;
              (* Recipe parameters are textual substitution. {{param}} in
                 recipe bodies is expanded BEFORE parsing. This means recipe
                 parameters can appear in identifier positions, wire endpoints,
                 and cell names. Expansion is a pre-parse phase. — BUG-003 *)
prompt_frag   = "prompt@" IDENT prompt_lines ;
oracle_decl   = "#" IDENT ":" "oracle" oracle_block "#/" ;

(* === Cells === *)

cell          = "#" IDENT ":" cell_type cell_body "#/"
              | "meta" "#" IDENT ":" cell_type cell_body "meta" "#/" ;

map_cell      = "map" "#" IDENT ":" cell_type "over" REF "as" IDENT
                cell_body "#/" ;

reduce_cell   = "reduce" "#" IDENT ":" cell_type "over" REF "as" IDENT
                "with" IDENT "=" value cell_body "#/" ;

cell_body     = { ref_decl | annotation | prompt_section
                | oracle_block | accept_block | vars_block
                | distill_block } ;
              (* Note: each> blocks may appear nested inside other prompt
                 sections (e.g. inside user>). The parser treats each> as
                 a sub-section that iterates within its parent section. *)

cell_type     = "llm" | "script" | "oracle" | "decision" | "meta"
              | "distilled"
              | "text" | "inventory" | "synthesis" | "code"
              | "laws" | "boundaries" | "diagram"
              | "mol" "(" IDENT ")" ;           (* sub-molecule invocation *)

(* === Cell body elements === *)

ref_decl      = "-" IDENT ( "." IDENT )? [ "(" "or" ")" ] ;

vars_block    = "vars>" { IDENT "=" value } ;

annotation    = "@" IDENT "(" annot_args ")" ;
annot_args    = IDENT ":" value { "," IDENT ":" value } ;

prompt_section = section_header prompt_content ;
section_header = SECTION_TAG ">" [ "?" guard ] ;
guard          = IDENT "(" IDENT { "," IDENT } ")" | IDENT ;
prompt_content = { INDENT REST_OF_LINE | each_block } ;
prompt_lines   = { INDENT REST_OF_LINE } ;

oracle_block  = "```" "oracle" NEWLINE oracle_body "```" ;
oracle_body   = { oracle_stmt } ;
oracle_stmt   = "json_parse" "(" IDENT ")" ";"
              | "keys_present" "(" IDENT "," "[" string_list "]" ")" ";"
              | "assert" expr ";"
              | "for" IDENT "in" expr "{" { oracle_stmt } "}"
              | "if" expr "{" { oracle_stmt } "}"
              | "score" IDENT "{" { score_clause } "}"
              | "reject" "if" expr ";"
              | "accept" "if" expr ";"
              | "score" "(" expr ")" "if" expr ";" ;

script_block  = "```" IDENT NEWLINE script_body "```" ;
script_body   = { ANY_LINE } ;

accept_block  = "accept>" prompt_lines ;

(* === Prompt sections === *)

(* system>, context>, user>, think>, examples>, format>, each>, accept> *)

examples_block = "examples>" { example_pair } ;
example_pair   = STRING "->" value ;

format_block   = "format>" IDENT format_body ;
format_body    = "{" format_fields "}" ;
format_fields  = { IDENT ":" format_type } ;
format_type    = "str" | "number" | "boolean"
               | "[" format_type "]"
               | "[" "_" "]"
               | STRING { "|" STRING }
               | "{" format_fields "}" ;

each_block     = "each>" IDENT "in" REF prompt_lines ;

(* === Wires === *)

wire          = wire_endpoint "->" wire_endpoint
              | wire_endpoint "->" "?" IDENT "->" wire_endpoint ;
wire_endpoint = IDENT | "[" ident_list "]" ;
              (* Array endpoints fan-out/fan-in. — BUG-005
                 !wire prescan -> [impl, test] means:
                 prescan -> impl AND prescan -> test *)

(* === Graph operations === *)

operation     = "!add" cell
              | "!drop" IDENT
              | "!wire" wire_endpoint "->" wire_endpoint
              | "!cut" wire_endpoint "->" wire_endpoint
              | "!split" IDENT "=>" "[" ident_list "]"
              | "!merge" "[" ident_list "]" "=>" IDENT
              | "!refine" IDENT "{" prompt_lines "}"
              | "!seed" IDENT "{" value "}" ;

(* === Presets and inputs === *)

preset        = "preset" IDENT "{" { preset_field } "}" ;
preset_field  = IDENT "=" value ;

input_decl    = "input" "param." IDENT ":" type_name
                { input_modifier } ;
input_modifier = "required"
               | "required_unless" "(" ident_list ")"
               | "default" "(" value ")" ;

(* === Squash directive === *)

squash_decl   = "squash>" { squash_field } ;
squash_field  = IDENT ":" value ;
              (* trigger: on_complete | on_all_complete | manual
                 template: IDENT  — squash template name
                 include_metrics: boolean *) ;

(* === Typed holes (inline in prompt text) === *)

typed_hole    = "{{" IDENT ("." IDENT)* (":" inline_oracle)? "}}" ;
inline_oracle = "json" [ format_body ]
              | "len" "(" NUMBER "," NUMBER ")"
              | "enum" "(" string_list ")"
              | "?" IDENT ;

(* === Primitives === *)

value         = STRING | NUMBER | "true" | "false" | "null"
              | "[" { value "," } "]"
              | "{" { IDENT ":" value "," } "}" ;

ident_list    = IDENT { "," IDENT } ;
string_list   = STRING { "," STRING } ;
param_list    = IDENT { "," IDENT } ;
type_name     = "str" | "number" | "boolean" | "json" | "[" type_name "]" ;
expr          = (* standard expression grammar with ==, !=, <, >, <=, >=,
                   and, or, not, in, contains, matches, typeof, len,
                   function calls, field access *) ;
```

### Context-Free Proof

Every production has a single non-terminal on the left side. Every delimiter
is explicitly matched: `#`/`#/`, `##`/`##/`, `{`/`}`, `` ``` ``/`` ``` ``,
`meta #`/`meta #/`. No production depends on context from other productions.
The grammar is context-free by construction — it is a set of BNF productions
with no context-sensitive rules.

The only subtlety is `meta # ... meta #/` vs `# ... #/`. These are distinct
token sequences (the lexer sees `meta` as a keyword prefix), not context-
dependent parses. A recursive descent parser handles them with one token of
lookahead.

**LL(1) with minor exceptions.** The grammar is LL(1) except for:
- `cell` vs `map_cell` vs `reduce_cell` (distinguished by leading keyword)
- `cell` vs `meta cell` (distinguished by `meta` keyword prefix)

Both are resolved with one extra token of lookahead (LL(2) at those points).
A PEG parser handles this trivially.

---

## 4. Prompt Sections

Prompts are not strings. They are structured, multi-section constructs that
map to LLM API message roles.

### Section Types

| Section | Maps To | Purpose |
|---------|---------|---------|
| `system>` | System message | Persona, constraints, behavioral rules |
| `context>` | System or user context | Data injection from upstream cells |
| `user>` | User message | The actual instruction |
| `think>` | Prefill scaffold | Chain-of-thought structure |
| `examples>` | Few-shot turns | Input → output demonstration pairs |
| `format>` | Appended instruction + auto-oracle | Output shape declaration |
| `accept>` | Human/agent gate | Acceptance criteria for step completion |
| `each>` | Repeated section | Iterate over collection in prompt |

### Conditional Sections

```cell
context> ?has-src-files(file-categories)
  Source files requiring review:
  {{file-categories | where(category == "src")}}
```

`?guard` gates inclusion. The section is omitted if the guard fails. Guards
can be:
- `?ref` — include if ref exists and is non-empty
- `?predicate(ref)` — include if predicate passes on ref

### Prompt Fragments

```cell
prompt@ analyst-persona
  You are a senior equity analyst.
  You cite sources. You flag uncertainty.

# valuation : llm
  system>
    {{@analyst-persona}}
  user>
    Build a DCF model for {{param.ticker}}.
#/
```

`prompt@` declares a reusable text fragment. `{{@name}}` embeds it.
Compile-time substitution only — no runtime identity, no oracle, no cost.

---

## 5. Typed Holes

Every `{{ref}}` interpolation point can carry an inline oracle:

```
{{ref : json { key: type }}}     -- structural shape check
{{ref : len(100, 5000)}}         -- length bounds
{{ref : enum("a", "b", "c")}}   -- enum membership
{{ref : ?named-oracle}}          -- reference a declared oracle
```

The oracle runs BEFORE prompt assembly. If the upstream cell's output fails
the hole's oracle, the prompt is never built and the upstream gets REJECT.
This prevents expensive downstream cells from consuming garbage.

Short form (no oracle) is still valid: `{{ref}}` passes anything through.

### Filter Expressions

```
{{ref | where(severity == "critical")}}  -- filter collection
{{ref | select(name, score)}}            -- project fields
{{ref | sort(score, desc)}}              -- order
{{ref | first(3)}}                       -- take N
```

Filters transform the interpolated value before insertion. They compose
left-to-right via `|`.

---

## 6. Combinators

### map — parallel fan-out

```cell
map # review : llm over {{param.aspects}} as aspect
  - source-data
  user>
    Focus on: {{aspect.focus}}
    {{aspect.description}}
#/
```

Creates one cell per item in the collection. All cells execute in parallel.
Downstream cells reference `review.*` to get the full collection of outputs.

### reduce — sequential fold

```cell
reduce # summarize : llm over {{documents}} as doc with acc = ""
  context> ?acc
    Running summary: {{acc}}
  user>
    Incorporate: {{doc}}
#/
```

Processes items sequentially. Each invocation receives `{{acc}}` (prior
output) and `{{doc}}` (current item). Final output is the last accumulator.

### each — iteration in prompts

```cell
# synthesis : llm
  - review.*
  each> r in {{review.*}}
    ### {{r.aspect.title}}
    {{r}}
  user>
    Synthesize all findings.
#/
```

Expands one prompt section per item in the collection. Unlike `map` (which
creates cells), `each` creates prompt text within a single cell.

### Collection References

`name.*` refers to all outputs from a `map` cell. It is a first-class
collection:
- `{{len(name.*)}}` — count
- `- name.*` — depend on all mapped cells
- `each> x in {{name.*}}` — iterate

---

## 7. Presets

```cell
preset gate {
  aspects = [
    { id: "security", focus: "vulnerabilities", ... },
    { id: "smells",   focus: "anti-patterns",   ... },
  ]
}

preset full {
  aspects = [
    { id: "security",    ... },
    { id: "correctness", ... },
    { id: "performance", ... },
    ...
  ]
}
```

Named parameter sets. Applied at pour time: `gt mol pour X --preset=gate`.
The preset's fields override `param.*` values.

---

## 8. Inputs

```cell
input param.pr     : number required_unless(param.files, param.branch)
input param.files  : str    required_unless(param.pr, param.branch)
input param.branch : str    required_unless(param.pr, param.files)
input param.scope  : str    default("medium")
```

Declared at molecule level. Types: `str`, `number`, `boolean`, `json`,
`[type]`. Modifiers: `required`, `required_unless(...)`, `default(...)`.

Maps directly to current TOML `[inputs]` section.

---

## 9. Oracles

### Inline (on cells)

```cell
# report : llm
  ``` oracle
  json_parse(v);
  keys_present(v, ["thesis", "recommendation"]);
  assert v.recommendation in ["buy", "hold", "sell"];
  ```
#/
```

### Standalone (reusable)

```cell
# json-report : oracle
  ``` oracle
  json_parse(v);
  assert len(v) >= 100;
  ```
#/
```

### On wires

```cell
source -> ? json-report -> consumer
```

### On holes

```cell
user>
  Ratios: {{ratios : json { pe: number, debt_equity: number }}}
```

### Verdict Semantics

```
accept              -- output passes, propagate
score(quality)      -- passes with quality annotation (0.0–1.0)
redirect(cell)      -- valid but wrong destination
reject(reason)      -- fails, retry or escalate
```

Most restrictive verdict wins when multiple oracles check the same wire.

### Quality Scoring

```cell
score quality {
  +0.3 if all(v.metrics, fun(m) { typeof(m.value) == "number" });
  +0.2 if len(v.thesis) >= 50;
  +0.1 if contains(v, param.ticker);
}
reject if quality < 0.3;
accept if quality >= 0.7;
score(quality) if otherwise;
```

---

## 10. Graph Operations

Eight primitives, prefixed with `!` inside recipes and metacircular cells:

| Op | Syntax | Effect |
|----|--------|--------|
| add | `!add # name : type ... #/` | Insert a cell |
| drop | `!drop name` | Remove a cell |
| wire | `!wire A -> B` | Add dependency edge |
| cut | `!cut A -> B` | Remove dependency edge |
| split | `!split X => [A, B]` | Decompose cell, fork wires |
| merge | `!merge [A, B] => X` | Combine cells, union wires |
| refine | `!refine X { new prompt }` | Change cell instruction |
| seed | `!seed X { value }` | Pre-fill from prior digest |

### Recipes

```cell
recipe insert-gate(upstream, downstream, check) {
  !add # gate : oracle
    - upstream
    ``` oracle
    {{check}}
    ```
  #/
  !cut upstream -> downstream
  !wire upstream -> gate
  !wire gate -> downstream
}
```

Zero-token graph transformations. Agents fill parameters.

### Metacircular Cells

```cell
meta # evolve : meta
  - traces
  - history
  user>
    Emit Cell operations to improve the pipeline.
    ONLY emit !add, !wire, !split, !refine, !seed.
meta #/
```

Output is parsed as Cell operations. Applied to **next generation's proto**,
never the current molecule. Stratification prevents self-modification cycles.

---

## 11. Molecule Lifecycle

```cell
## pipeline {
  squash>
    trigger: on_complete
    template: work
    include_metrics: true
  ...cells and wires...
##/
```

`squash>` configures digest generation when the molecule completes.
Maps to current TOML `[squash]` section.

Lifecycle: Proto → pour → Molecule → execute → squash → Digest → annotate → evolve → Proto'

---

## 12. Content Addressing

```
cell_hash = blake3(canonical(prompt + sorted(ref_hashes) + oracle_hash))
```

- Names are metadata, not identity. Two names can alias the same hash.
- Edits create new hashes. Old cells persist immutably.
- Staleness = downstream refs point at old hashes.
- Evaluate once, cache forever. Cache key = hash.
- Hash literal: `@abc123def` pins a specific version.

---

## 13. Mapping from TOML

| TOML Formula | Cell Pattern |
|---|---|
| `type = "workflow"`, `[[steps]]` with `needs` | Sequential cells with `->` wires |
| `type = "convoy"`, `[[legs]]` | `map # leg over {{param.aspects}} as aspect` |
| `[synthesis]` with `depends_on` | Cell with `- leg.*` dependency |
| `[presets]` | `preset name { ... }` |
| `[prompts] base` | `prompt@ name` |
| `[vars]` / `[inputs]` | `input param.X : type` |
| `acceptance` | `accept>` block |
| `{{.variable}}` Go templates | `{{ref}}`, `{{param.X}}` |
| `{{range .items}}` | `map` / `each>` / `reduce` |
| `{{if .condition}}` | `context> ?guard` |
| `[squash]` | `squash>` |
| `parallel = true` | Cells with no dependency path between them |

---

## 14. Example: Workflow (shiny)

```cell
## shiny {
  input param.feature : str required

  # design : llm
    user>
      Design the architecture for {{param.feature}}.
      Consider edge cases and trade-offs.
    accept>
      Design doc committed covering approach and files to change.
  #/

  # implement : llm
    - design
    user>
      Implement {{param.feature}} per the design: {{design}}
    accept>
      All files modified/created and committed.
  #/

  # review : llm
    - implement
    user>
      Review the implementation. Check for bugs, readability, security.
    accept>
      Self-review complete, no obvious issues.
  #/

  # test : llm
    - review
    user>
      Write and run tests for {{param.feature}}.
    accept>
      All tests pass, no regressions.
  #/

  # submit : llm
    - test
    user>
      Submit for merge. Final git check, clear commit message.
    accept>
      Clean git status, pushed to feature branch.
  #/

  design -> implement -> review -> test -> submit
##/
```

## 15. Example: Convoy (code-review)

```cell
## code-review {
  input param.pr     : number required_unless(param.files, param.branch)
  input param.files  : str    required_unless(param.pr, param.branch)
  input param.branch : str    required_unless(param.pr, param.files)

  prompt@ review-base
    You are a specialized code reviewer in a convoy review.
    Your focus: {{aspect.focus}}

  map # leg : llm over {{param.aspects}} as aspect
    @ cost(max: 5000) @ quality(min: good)
    system>
      {{@review-base}}
    user>
      {{aspect.description}}
    format> json
      { "findings": [{ "severity": "P0" | "P1" | "P2",
                        "file": str, "description": str }] }
  #/

  # synthesis : llm
    - leg.*
    @ cost(max: 8000) @ quality(min: excellent) @ model(opus)
    each> findings in {{leg.*}}
      ### {{findings.aspect.title}}
      {{findings}}
    user>
      Synthesize: deduplicate, prioritize, produce verdict.
    format> json
      { "verdict": "approve" | "request-changes" | "comment",
        "blocking": [_], "suggestions": [_] }
  #/

  preset gate {
    aspects = [
      { id: "security", focus: "vulnerabilities", description: "..." },
      { id: "smells",   focus: "anti-patterns",   description: "..." },
      { id: "wiring",   focus: "unused deps",     description: "..." },
    ]
  }

  preset full {
    aspects = [
      { id: "correctness",  focus: "logic errors",    description: "..." },
      { id: "performance",  focus: "bottlenecks",     description: "..." },
      { id: "security",     focus: "vulnerabilities",  description: "..." },
      { id: "elegance",     focus: "design clarity",   description: "..." },
      { id: "resilience",   focus: "error handling",   description: "..." },
      { id: "style",        focus: "conventions",      description: "..." },
      { id: "smells",       focus: "anti-patterns",    description: "..." },
      { id: "wiring",       focus: "unused deps",      description: "..." },
      { id: "commit-discipline", focus: "commit quality", description: "..." },
      { id: "test-quality", focus: "meaningful tests",  description: "..." },
    ]
  }
##/
```

## 16. Example: Metacircular Evolution

```cell
## self-evolving {
  # traces : script
    @ cost(max: 0)
    ``` sh
    bd mol traces {{param.mol_id}} --format json
    ```
  #/

  # history : script
    @ cost(max: 0)
    ``` sh
    bd mol digests {{param.mol_id}} --last 5 --format json
    ```
  #/

  meta # evolve : meta
    - traces
    - history
    @ cost(max: 20000) @ quality(min: excellent)
    system>
      You optimize reactive bead DAGs by emitting Cell operations.
    user>
      Traces: {{traces : json}}
      History: {{history : json}}
      Find the slowest cell. Split it into parallel sub-cells.
      If a cell is stable for 5+ generations, seed it.
      Emit ONLY valid Cell operations.
    ``` oracle
    assert v == "" or contains(v, "!");
    assert not contains(v, "!drop");
    ```
  meta #/

  traces -> evolve
  history -> evolve
##/
```

## 17. Example: Complex Oracle

```cell
# compliance-gate : oracle
  ``` oracle
  json_parse(v);
  keys_present(v, ["action", "ticker", "quantity", "rationale"]);

  -- Structural
  assert v.action in ["BUY", "SELL", "HOLD"];
  assert v.quantity >= 1;
  assert v.quantity <= 1000000;

  -- Cross-field consistency
  if v.action in ["BUY", "SELL"] {
    assert len(v.rationale) >= 200;
  }

  -- Negative patterns
  assert not contains(v, "placeholder");
  assert not contains(v, "TBD");

  -- Quality scoring
  score quality {
    +0.3 if contains(v.rationale, v.ticker);
    +0.3 if len(v.rationale) >= 500;
    +0.2 if v.quantity <= 100000;
    +0.2 if not contains(v, "uncertain");
  }

  reject if quality < 0.3;
  accept if quality >= 0.7;
  score(quality) if otherwise;
  ```
#/
```

## 18. Example: Operational Molecule (Script Cells)

```cell
## dog-reaper {
  input param.threshold : str default("24h")

  squash>
    trigger: on_complete
    template: work
    include_metrics: true

  # find-stale : script
    @ cost(max: 0)
    ``` sh
    bd list --json --status=open \
      | jq '[.[] | select(.updated_at < (now - {{param.threshold}}))]'
    ```
  #/

  # triage : llm
    - find-stale
    @ cost(max: 3000)
    user>
      These beads haven't been updated in {{param.threshold}}:
      {{find-stale : json}}
      Categorize each as: close (abandoned), nudge (still active),
      or escalate (blocked).
    format> json
      { "close": [str], "nudge": [str], "escalate": [str] }
  #/

  # execute : script
    - triage
    @ cost(max: 0)
    ``` sh
    echo '{{triage}}' | jq -r '.close[]' | xargs -I{} bd close {} --reason "stale"
    echo '{{triage}}' | jq -r '.escalate[]' | xargs -I{} gt escalate -s MEDIUM "Stale: {}"
    ```
  #/

  find-stale -> triage -> execute
##/
```

---

## 19. Composition: Import, Apply, Aspects

Three constructs close the gap with TOML's composition features.

### Import

```cell
## shiny-secure {
  import shiny           -- load all cells and wires from shiny molecule

  -- now modify: insert security steps around implement
  apply insert-security-scan(implement)
##/
```

`import name` loads another molecule's cells, wires, inputs, and presets
into the current molecule. The imported names are available for recipes
and wiring. This is how `extends` works in TOML.

### Apply with Selectors

```cell
-- Apply a recipe to cells matching a selector
apply insert-gate(*, synthesis, LENGTH(100, 5000))
  where type == llm and depth > 0

-- Apply to a specific cell
apply rule-of-five(implement)

-- Apply to all cells with a tag
apply add-timeout(*)
  where tag == "expensive"
```

Selectors filter cells by:
- `type == llm` — cell type
- `depth > 0` — DAG depth (0 = source cells)
- `tag == "expensive"` — cell tag
- `name == "implement"` — exact name
- `*` — all cells

This is the AOP mechanism. Instead of `advice.around`, you write a recipe
that performs the graph transformation and `apply` it with a selector.

### Aspect Pattern (AOP via Recipes)

The TOML `security-audit` aspect with `advice.around` translates to:

```cell
-- Define the security aspect as a pair of recipes
recipe security-prescan(target) {
  !add # prescan : llm
    > Pre-implementation security check for {{target}}.
    > Review for secrets/credentials in scope.
    accept> No pre-existing security issues
  #/
  !wire prescan -> target
}

recipe security-postscan(target) {
  !add # postscan : llm
    - target
    > Post-implementation security audit.
    > Scan {{target}} output for: injection, XSS, secrets, path traversal.
    accept> Security audit passed
  #/
  -- Rewire: anything downstream of target now depends on postscan
  !wire target -> postscan
}

-- Apply aspect: wrap implement with pre/post security scans
## shiny-secure {
  import shiny
  apply security-prescan(implement)
  apply security-postscan(implement)
##/
```

### Expansion Pattern (Rule of Five)

The TOML `expansion` template translates to a recipe using `!split`:

```cell
recipe rule-of-five(target) {
  !split target => [draft, refine-1, refine-2, refine-3, refine-4]

  !refine draft {
    user>
      Initial attempt. Breadth over depth. Get the shape right.
  }
  !refine refine-1 {
    user>
      First pass: CORRECTNESS. Fix errors and bugs.
  }
  !refine refine-2 {
    user>
      Second pass: CLARITY. Simplify. Can someone else understand this?
  }
  !refine refine-3 {
    user>
      Third pass: COMPLETENESS. What's missing? Edge cases?
  }
  !refine refine-4 {
    user>
      Final pass: POLISH. Style, naming, documentation.
  }
}

## shiny-enterprise {
  import shiny
  apply rule-of-five(implement)
##/
```

### Grammar Additions

```ebnf
(* Add to mol_body *)
mol_body      = { cell | map_cell | reduce_cell | wire | preset
                | input_decl | prompt_frag | oracle_decl
                | import_decl | apply_stmt | squash_decl | COMMENT } ;

import_decl   = "import" IDENT ;
apply_stmt    = "apply" IDENT "(" ident_list ")" [ where_clause ] ;
where_clause  = "where" selector_expr ;
selector_expr = selector_pred { "and" selector_pred } ;
selector_pred = "type" "==" cell_type
              | "depth" CMP NUMBER
              | "tag" "==" STRING
              | "name" "==" STRING ;
```

These three additions (import, apply, selectors) close all 4 gaps
identified in the formula survey: AOP aspects, expansion templates,
formula composition, and selector-based application.

---

## 20. Sub-Molecule Invocation

A cell can reference another molecule as its computation. The molecule is
poured as a nested execution — its inputs come from the parent cell's refs,
its output is the digest.

```cell
## idea-to-plan {
  # gather-requirements : llm
    user>
      Gather requirements for {{param.idea}}.
  #/

  # generate-plan : mol(design)
    - gather-requirements
    vars>
      problem = {{gather-requirements}}
      scope = "medium"
  #/

  # review-plan : llm
    - generate-plan
    user>
      Review this plan: {{generate-plan}}
      Verdict: APPROVE or REVISE with feedback.
    format> json
      { "verdict": "approve" | "revise", "feedback": str }
  #/

  gather-requirements -> generate-plan -> review-plan
##/
```

`# name : mol(molecule-name)` declares a cell whose computation is another
molecule. `vars>` passes parameters. The nested molecule pours, executes,
squashes, and the digest becomes this cell's output.

### Grammar Addition

```ebnf
cell_type     = ... | "mol" "(" IDENT ")" ;
vars_block    = "vars>" { IDENT "=" value } ;
cell_body     = { ref_decl | annotation | prompt_section
                | oracle_block | accept_block | vars_block } ;
```

---

## 21. Wire Modifiers: OR-Join and Conditional Branching

By default, a cell waits for ALL upstream refs (AND-join). Two modifiers
change this:

### OR-Join

```cell
-- Cell executes when ANY predecessor completes (not all).
# pardon : llm
  - eval-a (or)
  - eval-b (or)
  - eval-c (or)
  user>
    One of the evaluations completed. Proceed with shutdown.
#/
```

`- ref (or)` marks a ref as OR-join. The cell fires when at least one
OR-ref delivers a value. Useful for shutdown dances and fallback patterns.

### Conditional Wires

```cell
-- Wire only activates if a predicate on the source output passes.
review -> ? verdict-is-revise -> generate-plan    -- loop back if REVISE
review -> ? verdict-is-approve -> submit          -- proceed if APPROVE
```

Combined with oracle-gated wires, this gives conditional branching.
The oracle on the wire acts as the branch predicate:

```cell
# verdict-is-revise : oracle
  ``` oracle
  json_parse(v);
  assert v.verdict == "revise";
  ```
#/

# verdict-is-approve : oracle
  ``` oracle
  json_parse(v);
  assert v.verdict == "approve";
  ```
#/
```

When `review` completes, both wires evaluate their oracle. Only the
matching branch fires. This is **conditional branching via oracle gates**,
not a control flow primitive — the DAG structure is fixed, but oracle
verdicts determine which paths actually propagate.

### Loop-Back (Patrol Pattern)

Patrols and iterative refinement use conditional wires to loop back:

```cell
## patrol {
  # scan : script
    ``` sh
    gt dolt status --json
    ```
  #/

  # triage : llm
    - scan
    user>
      Assess: {{scan : json}}
      Verdict: clean (nothing to do), act (take action), escalate.
    format> json
      { "verdict": "clean" | "act" | "escalate", "details": str }
  #/

  # act : llm
    - triage
    user>
      Execute remediation: {{triage.details}}
  #/

  # wait-and-rescan : script
    - act
    ``` sh
    sleep {{param.interval}}
    ```
  #/

  scan -> triage
  triage -> ? verdict-is-act -> act -> wait-and-rescan
  wait-and-rescan -> scan                          -- loop back

  triage -> ? verdict-is-clean -> done
  triage -> ? verdict-is-escalate -> escalate
##/
```

**Important**: within a single molecule pour, the loop is bounded by the
runtime's generation limit (`--max-loops N`). Unbounded loops use the
stratified evolution model: each loop iteration is a new generation
(pour → execute → squash → distill → pour).

### Grammar Additions

```ebnf
ref_decl      = "-" IDENT ( "." IDENT )? [ "(" "or" ")" ] ;
```

---

## 22. Distillation

LLM cells are expensive and nondeterministic. Distillation crystallizes them
into deterministic functions once they prove stable.

### Lifecycle

```
# start : llm           -- LLM evaluates, oracle validates
    ↓ (N consistent runs, oracle always passes)
# start : distilled      -- frozen: input→output mapping cached
    ↓ (oracle fails on new input pattern)
# start : llm           -- thawed: back to LLM for novel cases
```

### Distillation Record

When a cell is distilled, its execution history is captured:

```cell
# classify : distilled
  -- Provenance: distilled from llm after 12 consistent runs
  -- Distilled: 2026-03-09T04:00:00Z
  -- Input hash: blake3(prompt + refs)
  -- Output hash: blake3(output)
  -- Oracle: all 12 runs passed json_parse + keys_present

  distill>
    input_pattern: "{{param.category}}"
    output_map: {
      "bug" -> { priority: 1, type: "bug" },
      "feature" -> { priority: 2, type: "feature" },
      "chore" -> { priority: 3, type: "chore" }
    }
    fallback: llm
  #/
```

### Distillation Block Grammar

```ebnf
distill_block = "distill>" { distill_field } ;
distill_field = "input_pattern" ":" value
              | "output_map" ":" "{" { value "->" value } "}"
              | "fallback" ":" cell_type ;
```

A `distilled` cell body contains a `distill>` block instead of prompt sections.
The runtime checks the input against `input_pattern`. If it matches a known
mapping, the cached output is returned (zero cost, deterministic). If no match,
the `fallback` cell type is used (typically `llm`).

### Triggers

Distillation can be triggered by:

1. **Manual**: `cell distill <molecule> <cell>` — freeze a cell now
2. **Automatic**: runtime tracks oracle pass rate. After N consecutive passes
   with identical output shape, suggest distillation
3. **Oracle-driven**: a meta-oracle observes execution history and proposes
   distillations as part of the molecule's evolution cycle

### Phase Gates

Each phase of a Cell proof-of-concept MUST include distillation before
advancing:

1. **Pour** — Execute all cells in the phase (LLM-powered)
2. **Observe** — Track which cells produce consistent, oracle-passing output
3. **Distill** — Freeze stable cells into `distilled` type
4. **Validate** — Re-pour with distilled cells, verify equivalent behavior
5. **Advance** — Move to next phase only after distillation validates

This creates a ratchet: each phase is cheaper than the last because proven
cells no longer need LLM calls. The language literally teaches itself to be
more efficient through use.

### Cost Model

| Cell Type | Cost | Deterministic | Speed |
|-----------|------|--------------|-------|
| `llm` | $$$ | no | slow |
| `distilled` | $0 | yes | instant |
| `script` | $0 | yes* | fast |
| `text` | $0 | yes | instant |

*Script cells are deterministic given the same input, but shell commands
may have side effects.

---

## 23. Open Questions

1. **Parser implementation language.** Go (matches Gas Town), Rust (matches bd), or tree-sitter grammar (editor support)?
2. **TOML migration tooling.** Auto-convert existing formulas or manual migration?
3. **LSP support.** Provide autocomplete, hover docs, go-to-definition for `.cell` files?
4. **Distillation triggers.** Manual, automatic (N identical outputs), or oracle-driven?
5. **Cross-molecule oracle composition.** When molecules wire together via `gt sling`, do oracles inherit?
