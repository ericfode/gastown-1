# Ambiguity Analysis: cell-tracer PRD

**Reviewer**: Ambiguity Analysis
**Date**: 2026-03-09
**Document**: `.prd-reviews/cell-tracer/prd-draft.md`

---

## 1. Vague Qualifiers

### 1.1 "Deterministic enough to replay" (Goal 5)

**Location**: Goals, item 5 — "Traces should be deterministic enough to replay executions and audit decisions."

**Problem**: "Deterministic enough" is a subjective qualifier. It is unclear what degree of determinism is required. Does replay mean bit-identical output reproduction? Or does it mean the same control flow path is taken? What about non-deterministic cell inputs (timestamps, network calls, LLM responses)?

**Suggested clarification**: Define what "replay" means concretely. For example: "Given a trace, a replay tool can re-execute the same cell graph with the recorded inputs and verify that outputs match the recorded outputs (excluding non-deterministic fields listed in a schema-level `replay_exclude` set)."

### 1.2 "Structured execution traces" (Goal 1, Problem Statement)

**Location**: Problem Statement para 1, Goal 1.

**Problem**: "Structured" is used repeatedly but never defined. Does it mean typed fields in a JSON object? Does it mean conforming to a schema? Is there a distinction between "structured" and the existing bead history data (which already has input/output hashes)?

**Suggested clarification**: State the schema format explicitly (e.g., "Each trace event is a JSON object conforming to a versioned JSON Schema") and enumerate the minimum required fields.

### 1.3 "Coarse-grained data" (Problem Statement)

**Location**: Problem Statement — "bead history queries... capture only coarse-grained data."

**Problem**: What counts as "coarse" vs. "fine-grained"? The PRD lists what bead history captures (hashes, verdicts, latency) but does not specify what additional granularity cell-tracer adds beyond the fields listed in Goal 1.

**Suggested clarification**: Provide a concrete comparison table: "bead history captures X; cell-tracer additionally captures Y, Z."

---

## 2. Undefined or Under-Defined Terms

### 2.1 "Oracle verdict" / "oracle results"

**Location**: User story 1, Rough Approach step 2, wire structure.

**Problem**: "Oracle" is used as though it is a well-known concept, but it is never defined in this PRD. It appears to be a cell-zero concept (a correctness check), but the reader cannot determine: What is an oracle? What are the possible verdict values? Is it always present?

**Suggested clarification**: Add a Glossary section or inline definition. E.g., "An oracle is a validation function that compares a cell's output against a reference answer, yielding `pass`, `fail`, or `skip`."

### 2.2 "Frozen" cells

**Location**: User story 2 — "when cells get frozen."

**Problem**: "Frozen" is a distillation concept that is not defined. Does freezing mean the cell is permanently replaced with a deterministic version? Is it reversible? What triggers it?

**Suggested clarification**: Define freezing in context: "A cell is frozen when cell-zero determines its output is sufficiently consistent across N executions (per `param.freeze_threshold`) and replaces the LLM call with a cached deterministic output."

### 2.3 "Molecule" / "sub-molecule"

**Location**: Throughout (Constraint 2, user stories, rough approach).

**Problem**: The relationship between cells and molecules is assumed knowledge. For a reviewer outside the cell-language team, these terms are opaque.

**Suggested clarification**: Add a brief glossary entry: "A molecule is a directed graph of cells that executes as a unit. A sub-molecule is a molecule invoked by a cell within a parent molecule."

### 2.4 "Phase"

**Location**: Rough Approach step 2 — "wraps cell-zero phases."

**Problem**: What are cell-zero's phases? How many are there? The PRD wraps "each phase" but never lists them. This makes it impossible to assess the scope of the wrapping work.

**Suggested clarification**: Enumerate the phases (e.g., "cell-zero has three phases: observe, distill, execute") or reference the cell-zero documentation that defines them.

### 2.5 "Pour time"

**Location**: Constraint 4 — "Sink selection at pour time."

**Problem**: "Pour time" appears to be domain jargon for "execution time" or "invocation time," but it is not defined. A reader unfamiliar with the cell-language metaphor will not understand when this is.

**Suggested clarification**: Define pour time or use standard terminology: "at invocation time (when the molecule is poured/executed)."

### 2.6 "Candidate match rates"

**Location**: User story 2 — "showing candidate match rates."

**Problem**: What is a "candidate"? What constitutes a "match"? This is distillation-specific vocabulary that is not introduced.

**Suggested clarification**: Define the term, e.g., "A candidate is a proposed deterministic replacement for an LLM cell. The match rate is the percentage of historical executions where the candidate's output matches the LLM's output."

---

## 3. Contradictions and Tensions

### 3.1 "Zero modification to cell-zero" vs. "cell-zero consuming cell-tracer output"

**Location**: Goal 3 vs. Goal 4 and User Story 3.

**Problem**: Goal 3 says cell-zero's source remains untouched. Goal 4 and User Story 3 say cell-zero should consume cell-tracer's JSONL output directly in its observe phase. If cell-zero currently reads from bead history, switching it to read JSONL traces requires modifying cell-zero's observe phase. These two goals appear contradictory.

**Suggested clarification**: Clarify the boundary. Options: (a) cell-zero already has a pluggable input mechanism and JSONL is just a new adapter (no source change); (b) the JSONL consumption is a future goal requiring a separate cell-zero change; (c) a shim/adapter layer outside cell-zero handles the translation.

### 3.2 Non-goal "real-time streaming" vs. trace sink "stdout"

**Location**: Non-Goals item 2 vs. Rough Approach step 4.

**Problem**: The non-goal says "traces are batch, not live dashboards." But stdout output during execution IS real-time streaming — events appear as they happen. Is the intent that stdout is acceptable as a debugging aid but not a production pattern? This needs clarification.

**Suggested clarification**: Reframe: "Real-time dashboards and streaming aggregation are non-goals. Stdout trace output during execution is supported for debugging but is not a streaming API."

---

## 4. Ambiguous Scope Boundaries

### 4.1 Training data pipeline scope

**Location**: Goal 4 — "The trace output IS the training data for distillation."

**Problem**: If trace output IS the training data, then the trace schema is also the training data schema. This means any schema change to cell-tracer is a breaking change for the distillation pipeline. The PRD does not address schema versioning, backward compatibility, or migration. Open Question 1 asks about versioning but provides no guidance.

**Suggested clarification**: State whether schema stability is a hard requirement. Define the versioning strategy (e.g., "Trace events include a `schema_version` field. Cell-zero's observe phase must handle all versions >= 1.0").

### 4.2 "All cell-zero params must flow through" (Constraint 3)

**Location**: Constraint 3.

**Problem**: "All params" is absolute. Does this include future params added to cell-zero? How does cell-tracer know which params are cell-zero's vs. its own? If cell-zero adds a `param.trace_sink` in the future, there is a namespace collision. There is no param namespacing strategy.

**Suggested clarification**: Define a namespacing convention (e.g., "cell-tracer owns `param.trace_*`; all other params are forwarded to cell-zero") and state how collisions are handled.

### 4.3 Where does cell-tracer.cell live?

**Location**: Rough Approach final line — "lives alongside cell-zero.cell in `docs/examples/`."

**Problem**: Is cell-tracer a production component or an example? Living in `docs/examples/` suggests it is illustrative, but the PRD describes it as a critical observability wrapper and training data source. This creates confusion about its status.

**Suggested clarification**: If cell-tracer is a production artifact, it should live in a production path (e.g., `src/` or `cells/`). If it is an example, the PRD should scale back the production-grade requirements.

---

## 5. Statements with Multiple Interpretations

### 5.1 "Wraps cell-zero as a sub-molecule" (Goal 3)

**Location**: Goal 3, Constraint 2.

**Problem**: "Wraps" can mean: (a) cell-tracer invokes cell-zero as a single opaque step; (b) cell-tracer interposes between every cell in cell-zero's internal graph; (c) cell-tracer wraps each phase of cell-zero separately. The rough approach suggests (c), but the goal statement reads as (a).

**Suggested clarification**: State explicitly: "Cell-tracer invokes cell-zero phase-by-phase, wrapping each phase invocation with pre/post trace cells" (if that is the intent).

### 5.2 Wire structure conditional arrow

**Location**: Rough Approach step 3 — `trace-end -> ? oracle -> trace-distill (conditional on distillation events)`.

**Problem**: The `?` syntax is not defined. Is this a conditional wire in the cell language? Is it pseudocode? What determines whether the distillation branch fires? "Conditional on distillation events" is circular — it says the condition is that there are distillation events, without saying how those events are detected.

**Suggested clarification**: Specify the condition explicitly: "If the current phase is `distill` and cell-zero emits freeze/skip metadata, then `trace-distill` fires. Otherwise, execution proceeds to the next cell in the parent wire."

### 5.3 "One bead per execution" (User Story 4)

**Location**: User story 4 — "creates one bead per execution."

**Problem**: "Execution" is ambiguous. Does this mean one bead per molecule execution (the entire run), one bead per cell execution, or one bead per trace event? For a molecule with 50 cells, these differ by orders of magnitude.

**Suggested clarification**: State the granularity: "One bead per molecule execution, containing all trace events for that run" or "One bead per cell execution."

---

## 6. Implicit Assumptions

### 6.1 Cell language supports the wrapper pattern

**Location**: Rough Approach step 2.

**Assumption**: The PRD assumes that the cell language grammar supports wrapping a sub-molecule with pre/post script cells in a sequential wire. Open Question 2 acknowledges this uncertainty, but the rough approach is written as though the answer is "yes." If the answer is "no," the entire approach may need redesign.

**Suggested clarification**: Resolve Open Question 2 before finalizing the approach. If the grammar does not support this pattern, document what grammar extension or workaround is needed.

### 6.2 Script cells can emit structured JSON

**Location**: Rough Approach steps 2 and 4.

**Assumption**: The PRD assumes script cells can construct and emit JSON objects (with timestamps, durations, nested fields). If script cells are limited to simple shell commands, JSON construction may be fragile or impossible without external tooling.

**Suggested clarification**: State what capabilities script cells have. Can they call `jq`? Do they have access to a JSON-emitting built-in? Is there a cell-language native way to construct structured output?

### 6.3 Timestamps and duration measurement are available

**Location**: Goal 1, Rough Approach.

**Assumption**: The PRD assumes the cell runtime provides high-resolution timestamps and that script cells can measure duration. If cells execute in isolated environments, clock access may be restricted.

**Suggested clarification**: Confirm that the cell runtime exposes `$EPOCHREALTIME` or equivalent, and that pre/post cells share enough state to compute duration (e.g., via a temp file or environment variable).

### 6.4 `bd create` is callable from within a cell

**Location**: Rough Approach step 4, bead sink.

**Assumption**: The bead sink calls `bd create` from a script cell. This assumes the `bd` CLI is available in the cell execution environment and that authentication/database connectivity is present.

**Suggested clarification**: State the runtime environment requirements for each sink (e.g., "bead sink requires `bd` CLI v0.4+ and a running Dolt server on port 3307").

### 6.5 Trace file write concurrency

**Location**: Rough Approach step 4, JSONL sink.

**Assumption**: The JSONL sink appends to `param.trace_file`. If multiple cells execute concurrently (parallel wires), appending to the same file without locking will produce corrupted output.

**Suggested clarification**: State whether cell execution is sequential or parallel. If parallel, specify the file-locking strategy for the JSONL sink.

---

## 7. Summary of Critical Ambiguities

| # | Ambiguity | Severity | Section |
|---|-----------|----------|---------|
| 1 | Goal 3 vs. Goal 4 contradiction (zero modification vs. consuming traces) | High | Goals |
| 2 | "Deterministic enough" undefined | High | Goals |
| 3 | Cell-zero phases not enumerated | High | Rough Approach |
| 4 | Grammar support for wrapper pattern unconfirmed | High | Approach / Open Q2 |
| 5 | "One bead per execution" granularity ambiguous | Medium | User Stories |
| 6 | Production artifact in `docs/examples/` | Medium | Approach |
| 7 | Trace schema versioning undefined | Medium | Goals / Open Q1 |
| 8 | Param namespace collision risk | Medium | Constraints |
| 9 | Core terms undefined (oracle, frozen, molecule, phase, pour time) | Medium | Throughout |
| 10 | Concurrent write safety for JSONL sink | Low | Approach |
