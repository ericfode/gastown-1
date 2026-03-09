# PRD Review Synthesis: cell-tracer

**Review Date**: 2026-03-09
**PRD**: cell-tracer — Instrumented Cell-Zero Wrapper
**Reviewers**: 6-leg parallel review (requirements, gaps, ambiguity, feasibility, scope, stakeholders)

---

## Executive Summary

The cell-tracer PRD identifies a real need (structured execution tracing for cell-zero) but the proposed approach has a **fundamental feasibility gap**: the Cell grammar does not support intercepting individual phases within a sub-molecule. `mol(cell-zero)` is opaque — you get one entry and one exit, not per-phase hooks. The PRD's core assumption (wrapping each cell-zero phase with pre/post trace cells) is not achievable without modifying cell-zero or extending the language.

**Whole-execution tracing** (one trace-start/trace-end around the entire cell-zero run) IS feasible and delivers ~80% of the value. The scope should be aggressively reduced: JSONL output to a file is the only sink needed for MVP. The remaining 4 sinks (bead, mail, log, stdout-as-separate-from-JSONL) are gold-plating.

**Overall PRD Health**: NEEDS REVISION — feasibility gap must be resolved, scope must be cut.

---

## Cross-Cutting Findings

### 1. BLOCKING: Sub-molecule opacity (Feasibility + Gaps + Ambiguity)

All three reviewers independently flagged the same issue: `mol(cell-zero)` treats cell-zero as a black box. The PRD proposes per-phase tracing (`trace-start -> run-phase -> trace-end` for each phase) but the grammar provides no mechanism to intercept individual cells within a sub-molecule.

**Three options exist, all with tradeoffs:**
- (a) Whole-execution tracing only (feasible today, reduced granularity)
- (b) Refactor cell-zero into composable sub-molecules (breaks "zero modification" goal)
- (c) Add runtime-level cell event hooks (requires language extension, violates constraint)

**The PRD must pick one.** Option (a) is the only MVP-viable path.

### 2. BLOCKING: No Cell runtime for mol() (Feasibility)

Cell-zero currently dispatches via `gt sling`, which is a manual mechanism. There is no native Cell runtime that automatically pours sub-molecules. Until one exists, `mol(cell-zero)` is syntactically valid but unexecutable. Cell-tracer cannot be built until this prerequisite is met.

### 3. HIGH: Goal contradiction — "zero modification" vs "cell-zero consumes traces" (Ambiguity + Gaps)

Goal 3 ("zero modification to cell-zero") contradicts Goal 4 ("cell-zero should consume cell-tracer output directly"). If cell-zero's observe phase currently reads from `bd mol history`, switching it to read JSONL traces requires modifying cell-zero. The PRD must clarify whether Goal 4 means:
- (a) cell-tracer output format is compatible with what observe already reads (no change)
- (b) a shim layer translates trace output to bead history format (separate component)
- (c) cell-zero will be modified separately to consume traces (goal 3 is about source structure, not behavior)

### 4. HIGH: Scope is 3x larger than needed (Scope + Requirements)

The scope reviewer recommends cutting ~60% of described scope:
- **Cut entirely**: bead sink, mail sink, log sink, stdout-as-separate-sink, `param.trace_format`, `trace_distill` event, sink multiplexing, trace correlation/span IDs, `param.trace_filter`
- **Defer**: replay capability (Goal 5), schema versioning
- **Keep**: JSONL tracing (start/end events) to file, param pass-through

The requirements reviewer agrees: no priority levels are assigned, making scope tradeoffs impossible. Recommend P0/P1/P2 classification.

### 5. MEDIUM: Aggregation contradiction (Stakeholders + Gaps)

The PRD declares "trace aggregation/analytics" as a non-goal, but cell-zero's observe phase requires aggregated data (run count, pass rate, stability across N executions). Raw per-execution trace events don't satisfy this. Either:
- Cell-tracer must aggregate (contradicts non-goal)
- A separate aggregation layer must exist
- Cell-zero continues using `bd mol history` for aggregated data, and traces serve debugging/audit only

### 6. MEDIUM: Missing stakeholders (Stakeholders)

The PRD ignores several trace consumers:
- **Witness**: needs progress signals for stuck-agent detection
- **OTel pipeline**: Gas Town already has OTel infrastructure; cell-tracer creates a parallel observability system
- **cell-forge formula**: same execution path, different orchestration — traces must work in both contexts

### 7. MEDIUM: Undefined terms and vague qualifiers (Ambiguity)

Core terms (oracle, frozen, molecule, phase, pour time, candidate match rate) are used without definition. "Deterministic enough to replay" is unmeasurable. Cell-zero's phases are never enumerated, making wrapping scope impossible to assess.

---

## Before You Build: Critical Questions

These must be answered before proceeding to implementation planning:

1. **Per-phase vs. whole-execution tracing**: Given that `mol(cell-zero)` is opaque, do you accept whole-execution tracing (one start/end pair per cell-zero run) as the MVP? Or is per-phase granularity a hard requirement that demands cell-zero refactoring?

2. **Runtime prerequisite**: There is no Cell runtime that executes `mol()` cells today. Cell-zero dispatches via `gt sling`. Is cell-tracer intended for (a) the future native runtime, (b) the current formula-driven execution (cell-forge), or (c) both? This fundamentally changes the design.

3. **cell-zero modification boundary**: Goal 3 says "zero modification." Goal 4 says cell-zero should consume traces. Which takes priority? Is Goal 4 actually "traces are FORMAT-COMPATIBLE with what cell-zero already reads" rather than "cell-zero switches to reading traces"?

4. **Sink count for MVP**: The scope review strongly recommends JSONL-only for MVP. Do you agree, or are specific sinks (e.g., bead) required from day one?

5. **Aggregation responsibility**: Cell-zero's observe phase needs aggregated data across runs. Is cell-tracer responsible for aggregation, or does a separate component handle this?

6. **trace_distill events**: The gaps reviewer found that cell-tracer has no clear source for distillation metadata (Open Question 7). Should `trace_distill` be deferred until cell-zero explicitly exposes this data?

7. **OTel integration**: Gas Town has existing OTel infrastructure. Should cell-tracer emit OTLP events, or is the custom JSONL format intentionally separate?

8. **File location**: Is cell-tracer a production artifact or an example? `docs/examples/` suggests the latter, but the PRD describes production-grade requirements.

---

## Leg Summaries

### Requirements Completeness
- No measurable success criteria for any goal
- No acceptance conditions per sink
- No priority levels assigned
- Implicit requirements missing: error handling, concurrency, data sensitivity, idempotency
- Open Questions 1, 2, 7 are blocking (must resolve before implementation)

### Missing Requirements (Gaps)
- Sub-molecule wrapping semantics undefined (per-phase vs. whole-execution)
- Pour-wait creates 30-minute trace black holes (no progress events)
- Map cell tracing unspecified (N iterations per map cell)
- Oracle failure/retry semantics missing
- Decision cell skipped-branch tracing missing
- Bead sink creates unbounded growth (3000+ beads in a forge cycle)
- Self-distillation paradox (cell-zero distilling cell-tracer which wraps cell-zero)
- Missing runtime dependencies (bd mol, gt sling, blake3sum, python3)

### Ambiguity Analysis
- HIGH: Goals 3 vs 4 contradict each other
- HIGH: "Deterministic enough" undefined
- HIGH: Cell-zero phases never enumerated
- HIGH: Grammar support for wrapper pattern unconfirmed
- MEDIUM: Core domain terms undefined throughout
- MEDIUM: Production artifact in docs/examples/
- MEDIUM: Param namespace collision risk

### Technical Feasibility
- **Whole-execution tracing**: FEASIBLE today via mol() wrapping
- **Per-phase tracing**: NOT FEASIBLE without cell-zero refactoring or runtime hooks
- **Sink routing**: FEASIBLE via oracle-gated conditional wires (not decision cells)
- **BLOCKING**: No Cell runtime executes mol() cells; gt sling is manual dispatch
- **Bead sink**: Risk of Dolt write amplification under high-frequency tracing

### Scope Analysis
- Scope creep risk: HIGH (5 sinks = 5x work for marginal value over JSONL)
- MVP = JSONL tracing + param pass-through (cuts ~60% of scope)
- Cut: bead/mail/log sinks, trace_distill, replay, sink multiplexing, trace filters
- Resolve before building: grammar feasibility (Q2), error representation (Q5)

### Stakeholder Analysis
- Missing consumers: Witness, Deacon, Mayor, Refinery, OTel pipeline, cell-forge
- Aggregation contradiction: non-goal conflicts with cell-zero's observe phase needs
- Producer/consumer map shows 5 unconsidered consumer needs
- Human vs. automated consumer profiles conflated across sinks

---

## Individual Leg Reports

- Requirements: (inline in task output)
- Gaps: (inline in task output)
- Ambiguity: `.prd-reviews/cell-tracer/ambiguity-analysis.md`
- Feasibility: `.prd-reviews/cell-tracer/tech-feasibility.md`
- Scope: (inline in task output)
- Stakeholders: (inline in task output)
