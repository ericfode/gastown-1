# Cell Language Bug Reports — From Pour Execution

**Source**: Morpheus pouring `cell-proof-of-concept.cell` by hand
**Date**: 2026-03-09

---

## BUG-001: rtk mangles script cell output
**Severity**: warning
**Phase**: inventory (script cell)
**Description**: `ls -1` through rtk adds size annotations (e.g., `  6.3K`)
that break script parsing. Script cells that shell out get transformed output.
**Impact**: Any script cell using `ls`, `wc`, or file-listing commands
**Proposed fix**: Script cells should bypass rtk, OR Cell runtime should
strip rtk annotations from script output.

## BUG-002: grep in script cells matches wrong lines
**Severity**: warning
**Phase**: inventory (script cell)
**Description**: `grep '^type'` in molecule formulas matches description text
containing "type" (e.g., "types, operations, and laws"). Molecule formulas
don't have a `type` field.
**Impact**: Inventory misclassification
**Lesson**: Script cells need robust parsing. The shell is fragile.

## BUG-003: Recipe parameter interpolation in identifiers
**Severity**: error (grammar gap)
**Phase**: translate (rule-of-five)
**Description**: Recipes use `{{target}}` inside cell names like
`{{target}}-draft`. The grammar's REF production expects data flow refs, not
metaprogramming interpolation. There's no distinction between:
- `{{cell-name}}` — data ref (read output of cell)
- `{{param}}` — recipe parameter (textual substitution)
when used in identifier positions (cell names, wire endpoints).
**Impact**: All recipe definitions
**Proposed fix**: Define recipe parameter expansion as a pre-parse phase.
`{{param}}` in recipe bodies is textual substitution before parsing. Document
this as "recipe expansion happens before Cell parsing."

## BUG-004: No glob matching in selector predicates
**Severity**: warning (grammar gap)
**Phase**: translate (security-audit)
**Description**: TOML formulas use `glob = "implement"` for pointcut matching.
Cell's `selector_pred` has `name == STRING` (exact match) but no glob/pattern
matching.
**Impact**: AOP formulas that use pointcut globs
**Proposed fix**: Add `"name" "matches" STRING` to `selector_pred` in EBNF,
where STRING is a glob pattern. Or add `"name" "~" STRING` for regex.

## BUG-005: Recipe wire fan-out to parameter lists
**Severity**: error (grammar gap)
**Phase**: translate (security-audit)
**Description**: `!wire prescan -> targets` where `targets` is a recipe
parameter that expands to multiple cell names. The grammar's wire production
only supports `IDENT -> IDENT` (single endpoint).
**Impact**: Any recipe that wires to/from multiple cells
**Proposed fix**: Either:
(a) `!wire prescan -> [target1, target2]` — array wire endpoint
(b) `!wire prescan -> targets.*` — splat expansion
(c) Recipe parameter lists auto-expand: if `targets` is a list, `!wire`
    iterates over it.
Recommend (a) — explicit and grammar-friendly.

---

## Summary

| Bug | Severity | Category | Fix Complexity |
|-----|----------|----------|----------------|
| 001 | warning | runtime | low (bypass rtk) |
| 002 | warning | runtime | low (better parsing) |
| 003 | error | grammar | medium (pre-parse expansion) |
| 004 | warning | grammar | low (add matches pred) |
| 005 | error | grammar | medium (array wire syntax) |

**Grammar changes needed**: 2 (BUG-003, BUG-005)
**Grammar additions needed**: 1 (BUG-004)
**Runtime lessons**: 2 (BUG-001, BUG-002)
