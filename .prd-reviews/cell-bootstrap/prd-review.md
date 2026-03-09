# PRD Review Synthesis: Cell Bootstrap Process

**Date**: 2026-03-09
**PRD**: `.prd-reviews/cell-bootstrap/prd-draft.md`
**Reviewers**: 6 parallel agents (requirements, gaps, ambiguity, feasibility, scope, stakeholders)

---

## Executive Summary

The PRD articulates a compelling vision: bootstrap a Cell parser and runtime from LLM-interpreted execution via iterative distillation. The core concept is sound — Cycle 1 data validates that lex distillation works. However, the PRD has **critical definitional gaps** that prevent convergence from being measured, **significant scope creep** (6 phases when 2 would deliver value), and **a major blind spot**: the existing Go parser in `internal/cell/` that the PRD doesn't acknowledge.

**Overall PRD Health: 2.5 / 5** — Strong vision, weak specification. Needs focused revision before implementation.

---

## Cross-Cutting Themes (Across All 6 Reviews)

### Theme 1: The convergence metric is undefined (5/6 reviewers flagged)
The PRD's central claim — "iterate until convergence" — lacks a formal definition. Thresholds appear as 0.9, 0.95, and 0.99 in different places with no reconciliation. "Match rate" is undefined (JSON equality? structural equivalence? token sequence?). The LLM baseline is itself nondeterministic, so the reference standard moves. **This must be resolved before Phase 1 begins.**

### Theme 2: The grammar is not actually frozen (4/6 reviewers flagged)
Constraint 2 says "Grammar is fixed" but Cycle 1 found 7 bugs, several of which required spec changes. The spec is marked "Draft." Open Question #3 asks about grammar freeze policy but doesn't answer it. Distillation against a moving grammar cannot converge. **Either freeze the grammar or define a re-distillation protocol.**

### Theme 3: The scope is ~3x what's needed for a viable first milestone (3/6 reviewers flagged)
The scope reviewer recommends cutting Phases 3-6 entirely. The feasibility reviewer rates Phases 3-6 at 2/5 feasibility. The requirements reviewer notes Phase 5 (Runtime) has no testable acceptance criteria. **The MVP is a deterministic lexer + parser (Phases 1-2), not a full runtime with TOML replacement.**

### Theme 4: Parse distillation requires a different mechanism than lex (2/6 reviewers flagged)
Lex distillation uses regex patterns — this works because lexing is line-oriented. Parse distillation requires expressing recursive grammar productions, which regex cannot do. The PRD uses the same `distill> block` abstraction for both. **The parse distillation mechanism needs to be designed, not assumed.**

### Theme 5: An existing Go parser is not acknowledged (1 reviewer, but critical)
The stakeholder review discovered `internal/cell/lexer.go` (6.8K), `parser.go` (7.9K), `validate.go` (6.3K), and `ast.go` — a functioning Cell parser with tests. The PRD proposes building a parser from distillation without mentioning this existing code. **The relationship between the existing parser and the distilled parser must be defined.**

---

## Findings by Severity

### Critical (Must Fix Before Implementation)

| ID | Finding | Source |
|----|---------|--------|
| C1 | No formal convergence metric — 3 different thresholds (0.9, 0.95, 0.99) with no reconciliation | Requirements, Gaps, Ambiguity |
| C2 | Grammar is not frozen — 7 bugs in Cycle 1, spec is "Draft" status, no freeze policy | Gaps, Feasibility, Ambiguity |
| C3 | No holdout methodology — corpus is 33 files (not 35), used for both training and validation | Gaps, Requirements |
| C4 | "Match rate" is undefined — measuring against a nondeterministic LLM baseline | Gaps, Ambiguity |
| C5 | Parse distillation mechanism is undesigned — regex patterns can't express recursive grammar | Feasibility |
| C6 | Existing Go parser (`internal/cell/`) not acknowledged — unclear if it's being replaced, validated against, or extended | Stakeholders |
| C7 | No cost budget or abort criteria — no way to determine if distillation is economically viable vs. hand-writing a parser | Requirements, Gaps, Scope |

### Major (Should Fix Before Implementation)

| ID | Finding | Source |
|----|---------|--------|
| M1 | Phases 3-6 are separate projects masquerading as phases — Runtime bootstrap alone is an entire PRD | Scope |
| M2 | Indent sensitivity can't be handled by flat regex — PROMPT/SOFT_BLOCK exit requires indent tracking | Gaps, Feasibility |
| M3 | 18% of corpus uses features excluded from scope (import, apply, recipe, mol()) | Gaps |
| M4 | No error recovery strategy for the deterministic parser | Gaps |
| M5 | "Frozen rules never regress" contradicts Open Question about thawing | Requirements, Ambiguity |
| M6 | No grammar spec owner identified — no one can authorize grammar freeze or changes | Stakeholders |
| M7 | cell-zero circularity unresolved — it needs cell-reader to validate cell-reader | Feasibility, Ambiguity |
| M8 | Self-hosting gate may be too ambitious for v1 — cell-reader.cell uses every Cell feature | Scope |
| M9 | Oracle language has no formal spec — `expr` production is just "standard expression grammar" | Gaps |
| M10 | Implementation language (Go vs Rust) is unresolved — blocks Phase 1 deliverables | Requirements, Scope |

### Minor

| ID | Finding | Source |
|----|---------|--------|
| m1 | "Pour" and other domain jargon undefined — needs glossary | Ambiguity |
| m2 | Phase 6 exit criteria ("at least one production formula") is a very low bar | Requirements, Scope |
| m3 | No versioning scheme for distilled rules across phases | Gaps |
| m4 | Cost language is vague ("~$$$/ per parse") | Ambiguity |
| m5 | Cycle time estimates presented without confidence intervals | Ambiguity |
| m6 | Dolt server impact not assessed — forge cycles create sustained write load | Stakeholders |
| m7 | Polecat coordination for batch pours has no failure handling | Gaps |
| m8 | Round-trip "matches" is underspecified (byte-identical? semantically equivalent?) | Requirements |

---

## Before You Build: Critical Questions

These questions MUST be answered before implementation proceeds. They are ordered by dependency — answer #1 before #2, etc.

### 1. What is the relationship between `internal/cell/` and the bootstrap?
Does the existing Go parser (lexer.go, parser.go, validate.go, ast.go) serve as:
- (a) The reference implementation that distillation validates against?
- (b) Code to be replaced by distilled rules?
- (c) Something entirely unrelated?
This answer changes the entire framing of the bootstrap.

### 2. When does the grammar freeze?
Options: (a) Freeze now, before Cycle 2 — accept known bugs as spec behavior. (b) Fix known bugs, freeze before Cycle 2. (c) Allow evolution with a re-distillation protocol. Each has trade-offs. Pick one and commit.

### 3. What is the formal convergence metric?
Define precisely: what is measured (token type? value? position?), at what granularity (per-file? per-rule? per-token?), against what baseline (LLM output? existing Go parser? golden files?), and at what threshold (0.9? 0.95? 0.99?).

### 4. What is the MVP scope?
The scope reviewer recommends 2 phases (deterministic lexer + deterministic parser), not 6. Do you agree? If not, what is the minimum deliverable that proves the approach?

### 5. What is the cost budget?
What is the maximum spend (in LLM tokens or dollars) for the bootstrap? At what point do you abandon distillation and just use/extend the existing Go parser?

### 6. How does parse distillation work?
Regex patterns work for lex. What representation captures recursive grammar productions? Grammar rules? Production tables? Something else? This needs a prototype before committing to "5-7 cycles for parse distillation."

### 7. How do you handle indent sensitivity in the lexer?
The distilled lex rules are flat regex. PROMPT and SOFT_BLOCK mode exits require indent tracking. Options: (a) Stateful scanner wrapping regex rules. (b) Indent-aware patterns. (c) Accept that indent handling requires non-regex code and adjust the distillation claim.

### 8. What happens when a grammar bug is found?
Policy for Cycle N bug discovery: (a) Fix grammar, thaw affected rules, re-distill. (b) Fix grammar, batch updates at phase boundaries. (c) Defer grammar fixes until after bootstrap. Pick one.

### 9. Who owns what?
- Grammar spec owner: ___
- Cost budget owner: ___
- Phase transition approver: ___
- Existing Go parser maintainer: ___

### 10. What are the abort criteria?
At what point (cycles elapsed, cost spent, convergence stalled) is the bootstrap deemed infeasible? What is the fallback plan?

---

## Recommended Revised Scope

Based on the 6 reviews, the consensus recommendation is:

**Phase A: Deterministic Lexer (Cycles 2-4)**
- Freeze grammar (fix known bugs first)
- Implement Go lexer with 4-mode state machine from Cycle 1 regex rules
- Add indent tracking (stateful scanner, not pure regex)
- Validate against full corpus (k-fold cross-validation)
- Compare against existing `internal/cell/lexer.go` for consistency
- Exit: `gt cell lex` works on all corpus files at >99% match vs. existing parser

**Phase B: Deterministic Parser (Cycles 5-8)**
- Design parse distillation representation (not regex — grammar rules or production tables)
- Distill parse rules using token streams from Phase A
- Implement recursive descent parser in Go
- Validate against corpus and existing `internal/cell/parser.go`
- Exit: `gt cell parse` produces correct AST for 95%+ of corpus files

**Deferred (separate PRDs):**
- Emit / canonical formatting
- Self-hosting gate
- Runtime bootstrap
- TOML migration

---

## Individual Review Reports

The full individual reviews are available for reference:
- **Requirements**: Overall 3/5 — 4 critical, 6 major, 5 minor findings
- **Gaps**: Overall severity 4/5 — 3 critical, 5 major, 6 minor gaps
- **Ambiguity**: Overall 3/5 — 3 critical, 5 major, 5 minor ambiguities
- **Feasibility**: Overall 3.5/5 — Phase 1: 4.5/5, Phase 2: 2.5/5, Phases 3-6: 2/5
- **Scope**: Overall 2/5 — recommends cutting to 2 phases, removing Phases 3-6 entirely
- **Stakeholders**: 12 stakeholders mapped, 5 conflicts identified, existing Go parser is critical blind spot
