# PRD: Cell Bootstrap Process

## Problem Statement

Cell is a DSL for reactive computation graphs that replaces Gas Town's TOML formula system. The bootstrap problem: Cell's parser and runtime don't exist yet as standalone tools. Currently, LLMs execute Cell programs by interpreting the spec (cell-reader.cell parses .cell files, cell-zero.cell distills patterns). This is expensive (~$$$/ per parse), nondeterministic, and doesn't scale.

We need a formalized convergence process that takes us from "LLMs interpret Cell" to "a deterministic parser and runtime execute Cell." The bootstrap is self-referential: Cell Forge (a TOML formula) pours cell-reader (a Cell program) against a corpus of .cell files, and cell-zero observes execution patterns. Consistent patterns distill into deterministic rules. When enough rules accumulate, you have a parser. When the parser can parse itself, you have convergence. When cell-forge replaces itself with a Cell program, the bootstrap is complete.

**For whom?** Gas Town developers and the multi-agent system itself. Every formula currently runs as TOML; Cell is the replacement. The bootstrap is the bridge.

**Why now?** Forge Cycle 1 just completed (2026-03-09). Results show lex distillation at 95%+ coverage across 35 corpus files, with mode transitions and block content at 100% consistency. The data says convergence is achievable. We need to formalize the process before ad-hoc iteration loses coherence.

## Goals

1. **Define the convergence pipeline**: A repeatable, measurable process from LLM-interpreted Cell to deterministic Cell tooling
2. **Establish convergence metrics**: How we know the parser is "done" (quantitative, not subjective)
3. **Design the distillation ratchet**: Each forge cycle is cheaper than the last; frozen rules never regress
4. **Specify the self-hosting gate**: When cell-reader can parse cell-reader.cell correctly, and cell-zero can distill cell-zero.cell
5. **Produce a phased roadmap**: Ordered phases with clear entry/exit criteria, from lex distillation through full runtime
6. **Minimize human intervention**: The bootstrap should be mostly autonomous (agent-driven forge cycles), with human gates only at phase transitions

## Non-Goals

- **Building a production Cell IDE/LSP** (future work, depends on parser existing first)
- **Migrating existing TOML formulas to Cell** (separate effort, depends on runtime existing first)
- **Designing the Cell language itself** (spec already exists at docs/plans/2026-03-08-cell-language-spec.md)
- **Optimizing LLM costs during bootstrap** (distillation IS the cost optimization; we spend now to save later)
- **Cross-molecule composition or advanced Cell features** (import/apply/selectors are post-bootstrap)
- **Tree-sitter grammar or editor integration** (depends on parser being stable first)

## User Stories / Scenarios

### Story 1: Forge Operator (Agent or Human)
As a forge operator, I run `gt formula run cell-forge` and it executes the next cycle of convergence:
- Inventories the corpus
- Pours cell-reader against each file
- Analyzes patterns for distillation candidates
- Proposes distill> blocks
- Validates proposed rules against held-out files
- Freezes validated rules or plans next iteration
- Reports convergence progress

### Story 2: Convergence Monitor
As a convergence monitor, I can check the current state:
- What percentage of lex rules are distilled?
- What percentage of parse rules are distilled?
- How many forge cycles have run?
- What's the per-cycle cost trend?
- Which cells are still LLM-powered?
- When did we last regress (thaw a frozen rule)?

### Story 3: Parser Consumer
As a downstream tool (gt, bd, or another Cell program), I can invoke the Cell parser:
- Input: .cell source text
- Output: token stream (lex) + AST (parse) + validation result
- For distilled rules: deterministic, instant, zero LLM cost
- For undistilled rules: LLM fallback with oracle validation

### Story 4: Self-Hosting Validator
As a self-hosting validator, I run the metacircular test:
- Parse cell-reader.cell using cell-reader (bootstrapped parser)
- Parse cell-zero.cell using cell-reader
- Compare output to LLM-generated parse
- Report match rate (this IS the convergence metric)

## Constraints

1. **No external parser generators** (PEG, ANTLR, etc) for the initial bootstrap. The parser must emerge from distillation. External tools may validate or cross-check, but the source of truth is the distilled rules.
2. **Grammar is fixed** (Cell spec v1, with bug fixes). The bootstrap process doesn't change the language, it implements it.
3. **Corpus is the test suite**. The 35 .cell files in docs/examples/ are both training data and validation data. Cross-validation (hold-out sets) prevents overfitting.
4. **Beads is the tracking system**. All forge cycles, convergence metrics, and bug reports are beads.
5. **Gas Town agents run the forge**. Polecats execute forge cycles autonomously. Humans review phase transitions.
6. **TOML formula system must keep working** during bootstrap. Cell doesn't replace TOML until the bootstrap is complete.
7. **Existing cycle 1 results are the starting point**. We don't redo work that's already validated.

## Open Questions

1. **What's the minimum viable parser?** Lex-only? Lex+parse? Full pipeline (lex+parse+validate+emit)?
   - Cycle 1 suggests lex is closest to distilling. Should we ship lex-only first?
2. **Implementation language for the extracted parser?** Go (matches GT codebase), Rust (matches bd), or both?
   - The distilled rules are language-agnostic (regex patterns). Extraction is a separate step.
3. **How do we handle grammar bugs found during bootstrap?**
   - Cycle 1 found 7 bugs (BUG-013 through BUG-019). Grammar changes invalidate distilled rules.
   - Need a policy: freeze grammar before distillation, or allow grammar evolution with re-distillation?
4. **When do we thaw a frozen rule?**
   - If a new .cell file fails a distilled rule, do we thaw immediately or add the file to the next cycle?
5. **How many forge cycles to full convergence?**
   - Cycle 1 estimates: lex in 2-3 cycles, parse in 5-7, full pipeline in ~10.
   - Are these estimates calibrated? What's the cost budget?
6. **What's the role of cell-zero in the bootstrap?**
   - Cell-zero is designed to run as a background molecule alongside any other molecule.
   - During bootstrap, it's the thing being bootstrapped (circular). How do we break the circularity?
7. **How do we validate the runtime (not just the parser)?**
   - Parser correctness is measurable (does the AST match?). Runtime correctness is harder (does execution produce the right result?).

## Rough Approach

### Phase 1: Lex Distillation (Cycles 2-4)
- Freeze NORMAL mode token rules (95%+ coverage from Cycle 1)
- Freeze mode transitions (100% coverage from Cycle 1)
- Freeze HARD_BLOCK/SOFT_BLOCK content rules (trivial, 100%)
- Validate against full 35-file corpus with hold-out
- Extract distilled lex rules into a standalone lexer (Go or Rust)
- Exit criteria: lexer passes on all 35 files, matches LLM lex output at >99%

### Phase 2: Parse Distillation (Cycles 5-8)
- Analyze parse cell output patterns across corpus
- Distill molecule envelope structure (MOL_OPEN -> cells -> MOL_CLOSE)
- Distill cell body structure by type (llm, script, decision, etc.)
- Handle cell type-specific bodies (prompt sections, oracle blocks, etc.)
- Exit criteria: parser produces correct AST for all 35 files

### Phase 3: Validate + Emit Distillation (Cycles 9-10)
- Distill semantic validation rules (dependency resolution, DAG check, etc.)
- Distill canonical emit (round-trip: source -> lex -> parse -> emit matches source)
- Exit criteria: full pipeline round-trips all 35 files correctly

### Phase 4: Self-Hosting Gate
- Parse cell-reader.cell using the distilled parser
- Parse cell-zero.cell using the distilled parser
- Parse cell-forge.formula.toml's Cell counterpart using the distilled parser
- If all pass: the parser is self-hosting
- Exit criteria: cell-reader parses itself correctly

### Phase 5: Runtime Bootstrap
- Extract the distilled execution model from cell-zero
- Build a deterministic Cell runtime that executes distilled cells natively
- LLM cells still require LLM API calls (they're the fallback, not the default)
- Exit criteria: cell-forge can run as a Cell program (not TOML formula)

### Phase 6: TOML Replacement
- Migrate cell-forge.formula.toml to cell-forge.cell
- The formula system can load .cell files natively
- Existing TOML formulas continue to work (backward compat)
- Exit criteria: at least one production formula runs as Cell
