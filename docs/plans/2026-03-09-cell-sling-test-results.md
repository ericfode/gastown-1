# Cell Sling Test Results

**Date**: 2026-03-09
**Bead**: gt-nde
**Epic**: hq-7vk (Cell Language)
**Test Runner**: `internal/cell/parser/sling_test.go`

---

## Summary

**Total example .cell files**: 34
**Lex pass**: 22 (65%)
**Lex fail**: 12 (35%)
**Parse pass (of lex-pass)**: 8 (36% of lexed)
**Parse fail (of lex-pass)**: 14 (64% of lexed)
**Full pass (lex + parse + validate + prettyprint)**: 8 (24% overall)

The parser was built for a brace-delimited variant (`## name { ... ##/`) while the
spec and all example files use a brace-free variant (`## name ... ##/`). Combined
with a non-modal lexer that can't handle Unicode or prompt-embedded structural
characters, only 8 of 34 examples survive the full pipeline.

---

## Per-Level Results

### Level 1 — Trivial sling: `hello.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| hello.cell | 118 | 2 | FAIL | Parser: `expected {, got Newline` — molecule needs optional `{` |

**Bugs filed**: gt-an8 (molecule brace syntax)

### Level 2 — Simple with oracle: `rule-of-five.cell`, `security-audit.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| rule-of-five.cell | — | — | FAIL | Lex error: curly quote `'` (U+2019) |
| security-audit.cell | 201 | — | FAIL | Parse: `/` in prompt text treated as structural |

**Bugs filed**: gt-1u2 (Unicode in lexer), gt-hkr (structural chars in prompts)

### Level 3 — Distilled cell: `boot-triage-distilled.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| boot-triage-distilled.cell | — | — | FAIL | Parser: no `{` after molecule + no `distill>` support |

**Bugs filed**: gt-an8 (molecule brace), gt-a7a (distill> blocks)

### Level 4 — Parallel dependencies: `deacon-patrol.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| deacon-patrol.cell | — | — | FAIL | Lex error: curly quote `'` at line 127 |

**Bugs filed**: gt-1u2 (Unicode in lexer)

### Level 5 — Sub-molecule invocation: `idea-to-plan.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| idea-to-plan.cell | — | — | FAIL | Parser: no `{` after molecule + format field parsing |

**Bugs filed**: gt-an8 (molecule brace)

### Level 6 — Map/reduce cells: `code-review.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| code-review.cell | — | — | FAIL | Lex error: superscript `²` in prompt text |

**Bugs filed**: gt-1u2 (Unicode in lexer)

### Level 7 — Meta cell (metacircular): `cell-migration.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| cell-migration.cell | — | — | FAIL | Lex error: curly quote `'` at line 70 |

**Bugs filed**: gt-1u2 (Unicode in lexer)

### Level 8 — The bootstrap: `cell-reader.cell`, `cell-zero.cell`

| File | Tokens | Cells | Valid | Errors |
|------|--------|-------|-------|--------|
| cell-reader.cell | — | — | FAIL | Lex error: backslash `\` in prompt text |
| cell-zero.cell | — | — | FAIL | Lex error: em-dash `—` in prompt text |

**Bugs filed**: gt-1u2 (Unicode in lexer)

---

## Bug Summary

| Bead | Title | Priority | Files Affected |
|------|-------|----------|---------------|
| gt-1u2 | Lexer rejects Unicode characters in prompt text | P1 | 12 files |
| gt-an8 | Parser requires `{` after molecule name (spec uses no braces) | P1 | 11+ files |
| gt-a7a | Parser does not support `distill>` blocks (SOFT_BLOCK mode) | P1 | 1+ files |
| gt-hkr | Prompt text with structural characters misinterpreted | P2 | 3+ files |

### Root Cause Analysis

All four bugs share a common root cause: **the lexer is not modal**.

The Cell spec (section 2) defines four lexer modes: NORMAL, PROMPT, HARD_BLOCK,
and SOFT_BLOCK. The current lexer implementation has only a rudimentary
`inCodeFence` boolean (equivalent to HARD_BLOCK mode). It has no PROMPT mode
or SOFT_BLOCK mode.

Without modal lexing:
- **PROMPT mode missing** → prompt text is tokenized character-by-character
  instead of line-by-line, causing Unicode and structural character failures
- **SOFT_BLOCK mode missing** → `distill>` and `format>` body content can't
  be captured as opaque blocks
- **Molecule syntax** → the brace-delimited variant was a workaround for the
  parser, not the spec

**Recommended fix order**:
1. gt-an8 (molecule braces) — unblocks 11 files, small change
2. gt-1u2 (Unicode) — unblocks 12 files, requires adding PROMPT mode to lexer
3. gt-hkr (structural chars in prompts) — shares fix with gt-1u2
4. gt-a7a (distill> blocks) — requires SOFT_BLOCK mode

After fixing gt-an8 + gt-1u2, we'd expect ~22 files to pass (65% → ~65% of those
that currently lex + the ones unblocked by brace fix).

---

## Files That Pass (8 of 34)

These files use the `## name {` brace syntax and contain only ASCII in prompts:

1. dep-propagate.cell
2. convoy-feed.cell
3. dog-phantom-db.cell
4. dog-reaper.cell
5. dog-doctor.cell
6. polecat-lease.cell
7. session-gc.cell
8. town-shutdown.cell

---

## Test Infrastructure

The sling test is at `internal/cell/parser/sling_test.go` with two test functions:

- `TestSlingProgression` — 8-level progression with per-level semantic checks
- `TestSlingAllExamples` — bulk smoke test of all 34 .cell files in `docs/examples/`

Run with:
```bash
go test ./internal/cell/parser/ -v -run TestSling
```
