# PRD Review: Missing Requirements Analysis

**Reviewer**: PRD Gap Reviewer (agent)
**Date**: 2026-03-09
**PRD**: Cell Bootstrap Process
**Overall Gap Severity**: 4 / 5 (several critical gaps, multiple major gaps)

---

## Critical Gaps

### GAP-01: No grammar freeze policy — the spec is still moving under distillation (CRITICAL)

The PRD says "Grammar is fixed (Cell spec v1, with bug fixes)" but Cycle 1 found 7 grammar bugs (BUG-013 through BUG-019), and the earlier bug tracker shows BUG-003 and BUG-005 required grammar changes that were APPLIED to the spec. The spec also contains open questions (section 23) and features marked as post-bootstrap (import, apply, selectors in section 19) that are already present in corpus files (shiny-enterprise.cell, shiny-secure.cell use `import` and `apply`).

**What's missing**: A concrete grammar freeze protocol. When exactly does the grammar freeze? What happens to distilled rules when a bug fix changes the grammar? The PRD's Open Question #3 asks this but doesn't answer it. Without an answer, every distilled rule is at risk of invalidation by the next bug fix.

**Severity**: Critical. If the grammar keeps shifting, distillation cannot converge. The ratchet slips.

### GAP-02: Corpus is both training and test data — no holdout methodology specified (CRITICAL)

The PRD says "Cross-validation (hold-out sets) prevents overfitting" but never specifies:
- How many files are held out per cycle
- Whether held-out files rotate between cycles
- What happens when a distilled rule fails on a held-out file
- Whether new corpus files can be added mid-bootstrap (and what that does to frozen rules)

The corpus is 33 files (not 35 as stated — the count is wrong). This is a very small dataset. With 5 held out for validation (as suggested in cell-forge.formula.toml step "validate-distillation"), only 28 remain for training. Statistical significance is questionable.

**Severity**: Critical. Without a rigorous holdout strategy, "95% coverage" could mean "works on the files we tested."

### GAP-03: No definition of what "match" means for parse comparison (CRITICAL)

The PRD says the convergence metric is "compare output to LLM-generated parse" and "report match rate." But what counts as a match?

- Exact JSON equality? (Too strict — whitespace, key ordering)
- Structural equivalence? (Need to define what structural means)
- Token-level match for lex? (Position-sensitive or position-agnostic?)
- AST node match for parse? (Do annotations matter? Comments?)
- What about nondeterminism in the LLM baseline? If you run the LLM twice and get different parses, which one is "correct"?

The LLM is the oracle AND the thing being replaced. If the LLM is inconsistent, you're measuring match rate against a moving target.

**Severity**: Critical. The convergence metric is undefined.

---

## Major Gaps

### GAP-04: Grammar features in corpus but excluded from bootstrap scope (MAJOR)

The PRD non-goals include "Cross-molecule composition or advanced Cell features (import/apply/selectors are post-bootstrap)." But the corpus contains files that USE these features:
- `shiny-enterprise.cell`: uses `import` and `apply`
- `shiny-secure.cell`: uses `import` and `apply`
- `security-audit.cell`: defines a `recipe`
- `rule-of-five.cell`: defines a `recipe`
- `idea-to-plan.cell`: uses `mol()` cell type
- `cell-zero.cell`: uses `mol()` cell type

If the parser cannot handle these constructs, it will fail on 6+ corpus files. The PRD never addresses what the parser should do with grammar constructs that are "post-bootstrap." Reject them? Ignore them? Parse them but not validate?

**Severity**: Major. ~18% of the corpus uses features the PRD explicitly excludes.

### GAP-05: No error recovery strategy for the distilled parser (MAJOR)

The PRD describes the happy path: lex passes, parse passes, validate passes. But:
- What does the distilled lexer emit when it encounters an unknown token?
- What does the parser do on a syntax error? Panic? Skip to next cell? Skip to next molecule?
- The LLM can recover gracefully from malformed input. A deterministic parser cannot, unless error recovery is designed.
- The `fallback: llm` mechanism in distill blocks is only described for individual cells, not for partial failures (e.g., "lex succeeded on lines 1-50, failed on line 51 — what now?")

**Severity**: Major. A parser without error recovery is unusable for real-world files with bugs.

### GAP-06: Indent sensitivity is underspecified for distillation (MAJOR)

The Cell spec says "The lexer tracks the indent depth of the containing cell" and PROMPT mode exits on "outdent to cell level." The Cycle 1 lex rules use `^` (start of line) patterns but don't encode indent depth. The proposed regex rules have no indent tracking.

Specific problems:
- How deep is "cell body indent"? The spec says "indent+2 from cell open" but cell opens can be at molecule indent (2 spaces) or deeper.
- Nested code blocks inside prompts: the ``` closer must be "at same indent" — but the regex `^```$` doesn't check indent level.
- SOFT_BLOCK exit: "ends when the lexer sees something structural" — this requires knowing the current indent level, which a flat regex cannot track.

The distilled lex rules from Cycle 1 claim 95%+ coverage but don't handle indent. This means they will fail on any file with nested blocks or prompt content that contains structural-looking lines.

**Severity**: Major. The distillation approach (regex patterns) may be fundamentally inadequate for the indent-sensitive parts of the lexer.

### GAP-07: No cost budget or resource constraints (MAJOR)

The PRD says "Optimizing LLM costs during bootstrap" is a non-goal, but also says "Each forge cycle is cheaper than the last." There is no:
- Cost budget per cycle
- Total cost budget for the bootstrap
- Estimate of tokens consumed per corpus file per phase
- Limit on how many cycles can run before the project is abandoned
- Cost comparison: bootstrap via distillation vs. just writing a parser by hand

Cycle 1 processed 3 files manually and dispatched batches to 3 polecats. At ~35 files x 4 phases (lex, parse, validate, emit) x ~2000 tokens/phase, a single cycle is ~280K input tokens + output. 10 cycles is ~2.8M tokens minimum, likely 10x that with retries and analysis.

**Severity**: Major. Without a budget, the bootstrap could cost more than writing a parser manually.

### GAP-08: The "batch pour" delegation model has no failure handling (MAJOR)

Cycle 1 delegated to 3 polecats (rictus, furiosa, nux) for batch pours. The PRD says "Gas Town agents run the forge." But:
- What if a polecat crashes mid-pour?
- What if different polecats produce inconsistent lexes for the same file?
- What if a polecat's results arrive after the analysis step has already run?
- How is work partitioned? (The cycle 1 doc splits by complexity — is this a requirement?)
- What's the maximum parallelism? Can all 33 files be poured simultaneously?

**Severity**: Major. The PRD describes a multi-agent system but doesn't specify coordination semantics.

---

## Minor Gaps

### GAP-09: `format>` is both a SECTION_TAG and not a SECTION_TAG (MINOR)

The spec says format> is "not a SECTION_TAG" and enters SOFT_BLOCK mode. But the Cycle 1 lex rules list it alongside SECTION_TAGs. The cell-reader.cell's system prompt lists it under "Prompt tokens" as FORMAT_TAG. The grammar has both `format_block` (section 3) and the lexer mode transition for `format>` (section 2). The lex distillation needs to handle this correctly: `format>` with an IDENT enters SOFT_BLOCK, but `format>` without one is ambiguous.

**Severity**: Minor but could cause subtle parsing bugs.

### GAP-10: No versioning scheme for distilled rules (MINOR)

The PRD mentions "cell-reader-v{N}.cell" for frozen iterations but doesn't specify:
- How versions are tracked across the 6 phases
- Whether partial freezes are versioned (e.g., "lex frozen at v3 but parse at v1")
- How to compare two versions
- Whether old versions are kept or pruned

**Severity**: Minor. Can be figured out during implementation.

### GAP-11: The self-hosting test is ill-defined (MINOR)

Phase 4 says "parse cell-reader.cell using the distilled parser" and "if all pass: the parser is self-hosting." But cell-reader.cell uses `mol()` type (sub-molecule invocation) in its AST node types, and it references constructs the parser needs to handle metacircularly. The PRD doesn't address:
- Can the distilled parser handle the `format>` blocks in cell-reader.cell that define the AST schemas?
- cell-reader.cell has oracle blocks with complex expressions — can the parser parse those?
- What "correctness" means for self-hosting (AST matches? Round-trip passes? Can execute?)

**Severity**: Minor. The concept is sound but needs sharper exit criteria.

### GAP-12: PROMPT mode exit conditions have edge cases (MINOR)

The spec says PROMPT mode exits on "outdent to cell level" or "another SECTION_TAG at cell indent level." But:
- What about blank lines inside prompts? (Common in real files)
- What about prompt content that starts with `--` (a comment-looking line in prompt text)?
- What about `{{ }}` refs that span multiple lines?
- The Cycle 1 rules don't address any of these.

**Severity**: Minor per case, but in aggregate could affect many corpus files.

### GAP-13: expr production is a black box (MINOR)

The grammar defines `expr` as "(* standard expression grammar *)" with no actual productions. Oracle blocks use expressions extensively. The bootstrap parser will need to parse oracle bodies, but there's no grammar to distill from — just the comment "standard expression grammar."

**Severity**: Minor for lex distillation (oracle bodies are BODY_LINE in HARD_BLOCK mode), but becomes critical for Phase 3 (validate distillation) when oracle semantics matter.

### GAP-14: No rollback mechanism (MINOR)

The PRD mentions "thawing" frozen rules (Open Question #4) but never specifies:
- How to detect that a frozen rule needs thawing
- How to roll back to a previous version of the distilled parser
- Whether thawing one rule invalidates downstream rules that depend on it
- Whether git history is the rollback mechanism or something more structured is needed

**Severity**: Minor. The ratchet design implies rollback is rare, but when it happens, the process needs to be defined.

---

## Questions That Must Be Answered Before Implementation

1. **Grammar freeze date**: When does the Cell grammar freeze? Before Cycle 2 starts? What is the process for grammar changes after the freeze? (blocks GAP-01)

2. **Match metric definition**: Write a concrete specification for what "output matches" means at each phase. JSON schema comparison? AST diff? Token sequence equality? (blocks GAP-03)

3. **Corpus accounting**: The PRD says 35 files but there are 33. Which files? Are files that use `import`/`apply`/`recipe`/`mol()` included or excluded from the corpus? (blocks GAP-02, GAP-04)

4. **Indent handling strategy**: Can the distilled lexer handle indent-sensitive mode transitions with regex alone, or does it need a stateful scanner? If stateful, the "distilled rules are language-agnostic (regex patterns)" claim in Open Question #2 is false. (blocks GAP-06)

5. **Budget**: What is the maximum number of cycles and maximum total LLM cost before the bootstrap is deemed infeasible and a hand-written parser is built instead? (blocks GAP-07)

6. **Error handling**: What does the distilled parser emit for invalid input? A partial result? An error? Fallback to LLM for the whole file? (blocks GAP-05)

7. **Polecat coordination**: What happens when two polecats produce different lex results for the same file? Which is authoritative? (blocks GAP-08)

8. **What "95% coverage" actually covers**: The Cycle 1 results say "95% of NORMAL mode tokens." Does this include or exclude: blank lines, whitespace-only lines, lines inside recipes, lines inside `import`/`apply` statements, inline `{{ }}` refs within prompt content? (blocks GAP-02)

---

## Summary

The PRD has a sound high-level architecture: the distillation ratchet concept is well-motivated, the phase progression (lex -> parse -> validate -> emit -> self-host -> runtime) is logical, and the connection between cell-reader, cell-zero, and cell-forge is clearly articulated.

The critical gaps are all around **measurement and methodology**: what exactly is being measured (GAP-03), what data is being measured against (GAP-02), and whether the thing being measured is stable enough to measure (GAP-01). These are the foundational gaps — if the convergence metric is undefined, convergence cannot be assessed.

The major gaps are around **operational reality**: indent sensitivity may break the regex approach (GAP-06), grammar features in the corpus are excluded from scope (GAP-04), and the multi-agent execution model lacks failure handling (GAP-08).

Recommendation: Address GAP-01, GAP-02, and GAP-03 before any implementation. These are prerequisite definitions, not implementation details.
