# PRD: cell-tracer — Instrumented Cell-Zero Wrapper

## Problem Statement

Cell-zero distills expensive LLM cells into deterministic ones by observing execution history. But today, that observation relies on bead history queries (`bd mol history`) which capture only coarse-grained data: input/output hashes, oracle verdicts, and latency. There's no structured, configurable execution trace that captures the full lifecycle of every cell execution — start events, end events, distillation decisions, and intermediate state.

**For whom?** Cell language developers, distillation pipeline operators, and the cell-zero runtime itself.

**Why now?** The cell language has a working parser (cell-reader), a distillation engine (cell-zero), and a bootstrap forge formula. The next bottleneck is observability: you can't debug, audit, or improve what you can't see. Cell-tracer produces the structured execution traces that feed cell-zero's observation phase and enable humans to understand what happened during a molecule execution.

## Goals

1. **Structured execution tracing**: Emit trace events for every cell phase — start, end, distill — with timestamps, inputs, outputs, oracle results, and duration.
2. **Pluggable trace sinks**: Support multiple output destinations (stdout, JSONL file, bead creation, gt mail, log file) via `param.trace_sink`.
3. **Zero modification to cell-zero**: Cell-tracer wraps cell-zero as a sub-molecule; cell-zero's source remains untouched.
4. **Training data pipeline**: The trace output IS the training data for distillation — cell-zero should be able to consume cell-tracer output directly.
5. **Replay and audit**: Traces should be deterministic enough to replay executions and audit decisions.

## Non-Goals

- **Runtime performance optimization** — tracing adds overhead; that's acceptable.
- **Real-time streaming** — traces are batch, not live dashboards.
- **Modifying cell-zero internals** — cell-tracer is a pure wrapper.
- **Trace aggregation/analytics** — downstream tools handle analysis; cell-tracer just emits.
- **UI or visualization** — traces are data; visualization is a separate concern.

## User Stories / Scenarios

### Developer debugging a failed distillation
> "I ran cell-zero on my molecule and it said 'consistency too low' — but I can't see which cells had inconsistent outputs. I need a trace showing every cell execution with its oracle verdict."

Cell-tracer wraps the molecule execution and emits `trace_end` events with oracle results. The developer greps the JSONL trace for `oracle: fail` to find the inconsistent cells.

### Distillation pipeline operator monitoring convergence
> "I'm running cell-zero repeatedly to build up execution history. I need to see how many runs each cell has, what the pass rates are, and when cells get frozen."

Cell-tracer emits `trace_distill` events when cells are frozen or skipped, showing candidate match rates and freeze decisions.

### Cell-zero consuming its own traces
> "Cell-zero's observe phase needs structured execution data. Instead of querying bead history, it should consume cell-tracer's JSONL output directly."

Cell-tracer with `param.trace_sink = jsonl` writes structured events that cell-zero's observe phase can parse — same data, richer format.

### Operator auditing a production molecule
> "Something went wrong in the overnight run. I need to see exactly what happened, cell by cell, with timestamps and durations."

Cell-tracer with `param.trace_sink = bead` creates one bead per execution, forming a permanent audit trail queryable via `bd`.

## Constraints

1. **Must be a valid .cell program** — cell-tracer.cell follows the Cell language grammar exactly.
2. **Sub-molecule invocation** — wraps cell-zero via `mol(cell-zero)` cell type; does not fork or duplicate cell-zero logic.
3. **Param pass-through** — all cell-zero params must flow through cell-tracer transparently.
4. **Sink selection at pour time** — `param.trace_sink` is an input parameter, not a compile-time choice.
5. **Depends on cell-zero.cell and cell-reader.cell** — these must be stable (they are, per forge cycle 1).
6. **Cell language constraints** — must parse cleanly through cell-reader; no grammar extensions.

## Open Questions

1. **Trace event schema**: What fields are required vs. optional? Should we version the schema?
2. **Sub-molecule interception**: Can `mol(cell-zero)` cells have pre/post wrapper cells in the current grammar, or does this require new wire patterns?
3. **Sink multiplexing**: Can multiple sinks be active simultaneously (e.g., stdout + jsonl)?
4. **Trace correlation**: How do we correlate trace events across sub-molecule boundaries? Is there a trace_id / span_id concept?
5. **Error tracing**: What happens when a cell fails? Does cell-tracer emit a trace_error event, or does the oracle verdict in trace_end suffice?
6. **Volume control**: For large molecules, tracing every cell may be noisy. Should there be a `param.trace_filter` for cell name patterns?
7. **Distill event source**: Where does cell-tracer get distillation metadata? Does it query cell-zero's output, or does cell-zero emit it?

## Rough Approach

Cell-tracer is a Cell program (cell-tracer.cell) that:

1. **Accepts the same inputs as cell-zero** plus `param.trace_sink` (default: stdout) and `param.trace_format` (default: json).

2. **Wraps cell-zero phases** with pre/post cells:
   - `# trace-start : script` — emits `trace_start` event before each phase
   - `# run-phase : mol(cell-zero)` — invokes the actual cell-zero phase
   - `# trace-end : script` — emits `trace_end` event with duration and results
   - `# trace-distill : script` — emits distillation-specific events

3. **Wire structure**: Sequential wrapper pattern:
   ```
   trace-start -> run-phase -> trace-end
   trace-end -> ? oracle -> trace-distill (conditional on distillation events)
   ```

4. **Sink implementations**: Each sink is a script cell that formats and outputs the trace event:
   - `stdout`: `echo` to stdout
   - `jsonl`: append to `param.trace_file`
   - `bead`: `bd create` with trace data
   - `mail`: `gt mail send` with trace summary
   - `log`: append to log file with human-readable format

5. **Decision cell** routes to the correct sink based on `param.trace_sink`.

The cell-tracer.cell file lives alongside cell-zero.cell in `docs/examples/` and follows the same conventions.
