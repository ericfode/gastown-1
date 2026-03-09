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

---

## 3. Grammar (EBNF)

```ebnf
program       = { molecule | recipe | prompt_frag | oracle_decl | input_decl } ;

(* === Top-level constructs === *)

molecule      = "##" IDENT "{" mol_body "##/" ;
mol_body      = { cell | map_cell | reduce_cell | wire | preset
                | input_decl | prompt_frag | oracle_decl
                | import_decl | apply_stmt | COMMENT } ;

import_decl   = "import" IDENT ;
apply_stmt    = "apply" IDENT "(" ident_list ")" [ "where" selector_expr ] ;
selector_expr = selector_pred { "and" selector_pred } ;
selector_pred = "type" "==" cell_type
              | "depth" CMP NUMBER
              | "tag" "==" STRING
              | "name" "==" STRING ;

recipe        = "recipe" IDENT "(" param_list ")" "{" { operation } "}" ;
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
                | oracle_block | accept_block } ;

cell_type     = "llm" | "script" | "oracle" | "decision" | "meta"
              | "distilled"
              | "text" | "inventory" | "synthesis" | "code"
              | "laws" | "boundaries" | "diagram" ;

(* === Cell body elements === *)

ref_decl      = "-" IDENT ( "." IDENT )? ;

annotation    = "@" IDENT "(" annot_args ")" ;
annot_args    = IDENT ":" value { "," IDENT ":" value } ;

prompt_section = section_header prompt_lines ;
section_header = SECTION_TAG ">" [ "?" guard ] ;
guard          = IDENT "(" IDENT { "," IDENT } ")" | IDENT ;
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

wire          = IDENT "->" IDENT
              | IDENT "->" "?" IDENT "->" IDENT ;

(* === Graph operations === *)

operation     = "!add" cell
              | "!drop" IDENT
              | "!wire" IDENT "->" IDENT
              | "!cut" IDENT "->" IDENT
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
                | import_decl | apply_stmt | COMMENT } ;

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

## 20. Open Questions

1. **Parser implementation language.** Go (matches Gas Town), Rust (matches bd), or tree-sitter grammar (editor support)?
2. **TOML migration tooling.** Auto-convert existing formulas or manual migration?
3. **LSP support.** Provide autocomplete, hover docs, go-to-definition for `.cell` files?
4. **Distillation triggers.** Manual, automatic (N identical outputs), or oracle-driven?
5. **Cross-molecule oracle composition.** When molecules wire together via `gt sling`, do oracles inherit?
