# Technical Feasibility Review: cell-tracer

**Reviewer**: Technical Feasibility
**PRD**: cell-tracer -- Instrumented Cell-Zero Wrapper
**Date**: 2026-03-09

---

## 1. Can cell-tracer intercept cell-zero phases purely through sub-molecule invocation?

**Partially. The PRD's wrapper model is feasible but mischaracterizes the interception granularity.**

The Cell grammar supports `mol(cell-zero)` as a cell type (spec section 20). A `mol()` cell pours the target molecule as a nested execution: inputs come from `vars>`, output is the squash digest. This is confirmed by working examples (`idea-to-plan.cell` lines 41-45, 65-69).

However, `mol(cell-zero)` invokes cell-zero as **one atomic unit**. The PRD proposes wrapping individual cell-zero *phases* (observe, analyze, propose, validate, freeze) with trace-start/trace-end cells. This requires one of two approaches:

- **Option A**: One `mol()` call per phase. This means cell-tracer must know cell-zero's internal structure and invoke each phase as a separate sub-molecule. Cell-zero is currently one molecule (`## cell-zero`), not five separate molecules. Cell-zero would need to be refactored into `## cell-zero-observe`, `## cell-zero-analyze`, etc., or cell-tracer would need to duplicate cell-zero's wire graph -- both violate the "zero modification" goal.

- **Option B**: Wrap cell-zero as a single `mol(cell-zero)` call with trace-start before and trace-end after. This gives you whole-execution tracing (start time, end time, final output) but **no visibility into individual phases**. You get a black box trace, not the per-phase instrumentation the PRD describes.

**Verdict**: The PRD's rough approach (section "Wraps cell-zero phases with pre/post cells") is not achievable without modifying cell-zero or the runtime. The grammar does not support intercepting internal cells of a sub-molecule from the parent molecule.

---

## 2. Hard Technical Problems

### 2.1 Accessing intermediate state between phases

This is the blocking problem. The `mol()` invocation model treats the sub-molecule as opaque. The parent molecule receives only the final squash digest. There is no mechanism in the Cell grammar to:

- Subscribe to events from individual cells within a sub-molecule
- Access intermediate cell outputs (`{{cell-zero.observe}}`) from outside the molecule boundary
- Inject pre/post hooks around cells inside a nested molecule

The spec is explicit (section 20): "The nested molecule pours, executes, squashes, and **the digest becomes this cell's output**." No intermediate outputs leak.

### 2.2 Decision cell routing to sink

The PRD proposes a "decision cell routes to the correct sink based on `param.trace_sink`." The `decision` cell type exists in the grammar (line 163 of the spec) and is used in cell-zero itself (`gate-valid : decision`, line 49) and in `idea-to-plan.cell` (`human-approve : decision`, line 80).

However, the `decision` cell type produces a verdict output -- it does not perform conditional wire activation by itself. **Conditional branching is done via oracle-gated wires** (spec section 21):

```
trace-end -> ? sink-is-stdout -> emit-stdout
trace-end -> ? sink-is-jsonl -> emit-jsonl
trace-end -> ? sink-is-bead -> emit-bead
```

Each `?` oracle evaluates `param.trace_sink` and only the matching branch fires. This pattern **is expressible** in the current grammar. The PRD's description is slightly imprecise (calls it a "decision cell" when the actual mechanism is oracle-gated conditional wires), but the underlying requirement is achievable.

### 2.3 Trace correlation across sub-molecule boundaries

The PRD's open question 4 (trace correlation) is a real gap. The Cell language has no concept of trace_id, span_id, or execution context propagation. A `mol()` invocation starts a new execution context. Correlating parent and child traces would require either:

- A convention (pass `param.trace_id` through `vars>`)
- Runtime support for automatic context propagation

Neither exists today.

---

## 3. Prerequisites That Must Exist First

### 3.1 Required (blocking)

1. **cell-reader must parse `mol()` cells** -- the parser must handle sub-molecule cell types. This is in the grammar but needs implementation verification.

2. **A Cell runtime that actually pours molecules** -- cell-zero currently uses `gt sling` to dispatch execution (line 87-89 of cell-zero.cell). There is no native Cell runtime that pours sub-molecules. Until one exists, `mol(cell-zero)` is syntactically valid but has no execution semantics.

3. **Squash digest format specification** -- cell-tracer needs to parse cell-zero's output. The squash digest format must be stable and documented.

### 3.2 Required for the full vision (non-blocking for MVP)

4. **Cell-zero refactored into composable phases** -- if per-phase tracing is needed, cell-zero's phases must be extractable as independent molecules or the runtime must support cell-level event hooks.

5. **`bd mol history` must return structured JSON** -- the observe phase depends on this. Currently referenced in cell-zero but may not be implemented.

---

## 4. Performance Concerns

### 4.1 Overhead per phase (acceptable)

If cell-tracer wraps cell-zero as a single `mol()` call, the overhead is two script cells (trace-start, trace-end) plus one sink cell. Script cells are `$0` cost (spec section 22 cost model). Latency overhead: milliseconds for timestamp capture and JSONL append. Negligible relative to LLM cell execution times (seconds to minutes).

### 4.2 Overhead per cell within cell-zero (concerning if implemented)

If per-phase tracing is achieved (via refactoring), cell-zero has 8 cells. Tracing each adds 2-3 script cells = 16-24 additional cells per execution. For `map` cells (`propose`, `validate`), the fan-out multiplies this by the number of distillation candidates. With 10 candidates: ~60 additional trace cells. Still cheap in token cost, but adds wire-graph complexity and scheduling overhead.

### 4.3 Bead sink overhead (potentially significant)

The `bead` sink calls `bd create` per trace event. Each `bd create` is a Dolt write + commit. For a full cell-zero run with 8+ cells, this means 8+ Dolt transactions. Given the documented Dolt fragility (CLAUDE.md war room), high-frequency bead creation during tracing could stress the server. Recommend JSONL as the default sink, with bead creation as a post-hoc batch operation.

---

## 5. Is "decision cell routes to sink" expressible?

**Yes, but not via a `decision` cell type.** The correct Cell pattern is oracle-gated conditional wires:

```cell
# sink-check-stdout : oracle
  ```oracle
  assert "{{param.trace_sink}}" == "stdout";
  ```
#/

# sink-check-jsonl : oracle
  ```oracle
  assert "{{param.trace_sink}}" == "jsonl";
  ```
#/

trace-end -> ? sink-check-stdout -> emit-stdout
trace-end -> ? sink-check-jsonl -> emit-jsonl
```

This is well-supported by the grammar (spec section 21). Multiple sinks (open question 3) would require multiple wires without the `?` oracle gate, or an explicit fan-out.

---

## 6. Does mol(cell-zero) allow pre/post wrapper cells in the wire graph?

**Yes, at the molecule boundary. No, inside the sub-molecule.**

The pattern from `idea-to-plan.cell` is definitive:

```cell
# intake : llm         -- pre-work
#/
# prd-review : mol(X)  -- sub-molecule
  - intake              -- depends on pre-work
#/
# human-clarify : llm  -- post-work
  - prd-review          -- depends on sub-molecule output
#/
```

Cell-tracer can wire `trace-start -> run-cell-zero -> trace-end` where `run-cell-zero : mol(cell-zero)`. The `trace-start` script captures timestamp and inputs; `trace-end` captures timestamp, duration, and the digest output. This gives **whole-execution tracing**.

What it cannot do: insert wrapper cells *between* `observe` and `analyze` inside cell-zero's wire graph. The sub-molecule boundary is opaque.

---

## 7. Recommendations

1. **Rescope MVP to whole-execution tracing.** Wrap `mol(cell-zero)` with trace-start/trace-end. This is fully expressible today and delivers audit trails, timing, and input/output capture.

2. **Defer per-phase tracing** until either:
   - Cell-zero is refactored into composable sub-molecules (each phase is its own `## phase-name` molecule), or
   - The Cell runtime gains cell-level event hooks (an observer/plugin API)

3. **Use oracle-gated wires for sink routing**, not a `decision` cell. Update the PRD's rough approach accordingly.

4. **Default to JSONL sink**, not bead creation. Batch-create beads from JSONL traces as a separate step to avoid Dolt write amplification.

5. **Add `param.trace_id`** as an input parameter that propagates through `vars>` into the sub-molecule, enabling manual trace correlation. This is a convention, not a language change.

6. **Prerequisite gate**: cell-tracer cannot be implemented until there is a Cell runtime that executes `mol()` cells. Currently, cell-zero uses `gt sling` as a manual dispatch mechanism -- there is no automatic sub-molecule pouring.
