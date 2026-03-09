# Cell Parser — PRD

**Poured from**: idea-to-plan.cell (intake cell)
**Date**: 2026-03-09

---

## Problem Statement

Cell needs a parser. The language spec is written. 34 example files exist. But
there is no tool that reads .cell source and produces a structured AST. Without
a parser, Cell programs cannot be validated, executed, or distilled.

The insight: the parser should be the **first distillation target**. Write an LLM
cell that parses Cell source. Feed it every .cell file. The oracle validates each
parse against the grammar. After enough consistent parses, the LLM's parse rules
crystallize into deterministic mappings. The distilled parser IS the parser.

No hand-written parser. The parser emerges from Cell Zero observing cell-parse.

## Success Criteria

1. cell-parse.cell parses all 34 existing .cell files with zero oracle rejections
2. Round-trip test passes: source → lex → parse → emit → re-parse produces identical AST
3. At least 5 lex rules distill after 10+ consistent runs (token patterns are highly regular)
4. Validation gate in Cell Zero rejects malformed .cell files
5. Grammar bugs discovered during parsing are documented and fixed in the spec

## Scope — In

- Lexer (cell-parse lex cell): tokenize .cell source
- Parser (cell-parse parse cell): token stream → AST
- Validator (cell-parse validate cell): semantic constraint checks
- Emitter (cell-parse emit cell): AST → canonical .cell source
- Fingerprinter (cell-parse fingerprint cell): content-addressable hash
- Cell Zero Phase 0: validation gate using cell-parse as sub-molecule
- Test harness: pour cell-parse against all 34 .cell files
- Distillation tracking: which parse rules are crystallizing

## Scope — Out

- Hand-written parser in Go/Rust/etc (that's what distillation produces)
- LSP support (future — after parser stabilizes)
- Editor syntax highlighting (future)
- TOML migration tooling (separate effort)
- Runtime execution engine (separate from parsing)

## Key Requirements

1. **cell-parse.cell must be valid Cell** — it must parse itself (metacircular test)
2. **Oracle strictness** — every token type and AST node type must be enumerated; no wildcards
3. **Canonical output** — the emitter must produce deterministic, diff-friendly Cell source
4. **Error recovery** — parser reports all errors, doesn't stop at the first one
5. **Grammar coverage** — must handle: molecules, cells (all types), prompts, oracles, scripts, wires, operations, inputs, presets, squash, distill blocks, meta cells, map/reduce cells, sub-molecule invocations, annotations, format blocks, each blocks, accept blocks
6. **Distillation readiness** — each cell in cell-parse should have stable input→output patterns that can crystallize

## Open Questions

1. How do we feed .cell source to cell-parse when we don't have a runtime yet?
   → Pour it by hand (morpheus IS the runtime). Use bd mol pour or equivalent.
2. Should the lex cell be the first distillation target? Token patterns are the most regular.
   → Yes — lexing is the most deterministic part. Distill lex first, then parse.
3. How do we handle prompt content (which contains arbitrary text including Cell-like syntax)?
   → Lexer must track nesting depth: ``` ... ``` blocks are opaque. Prompt lines are opaque after the section tag.
4. What about the oracle language itself — is it parsed by cell-parse?
   → Yes, oracle blocks are parsed as opaque script bodies (like bash blocks). The oracle language has its own grammar but cell-parse treats it as content.
5. Format block type syntax — is `"PROCEED" | "REJECT"` a string union or an enum?
   → It's a format_type: STRING { "|" STRING }. The spec already handles this.
