# R2: Reactive Dataflow and Incremental Computation

**Bead**: gt-m9z
**Date**: 2026-03-08
**Purpose**: Survey reactive dataflow paradigms applicable to [Gas City's](s3-architecture-sketch.md) agent
computation model. Identify what happens when agent outputs become cells in a
reactive DAG, how staleness propagates at scale, and whether reactive vs
pull-based recomputation matters for LLM workloads.

**See also**: [R1: Orchestration Frontier](r1-orchestration-frontier.md) · [R3: Agent Memory](r3-agent-memory.md) · [S1: Gap Analysis](s1-gap-analysis.md) · [S2: Abstraction Map](s2-abstraction-map.md) · [S3: Architecture Sketch](s3-architecture-sketch.md)

---

## 1. Systems Surveyed

Seven systems spanning compilers, databases, notebooks, trading, and the
original spreadsheet:

| System | Domain | Language | Key Innovation |
|--------|--------|----------|----------------|
| [**Adapton**](https://doi.org/10.1145/2594291.2594324) | PL research | OCaml/Racket/Rust | Demand-driven incremental computation |
| [**Salsa**](https://salsa-rs.github.io/salsa/) | Compiler ([rust-analyzer](https://rust-analyzer.github.io/)) | Rust | Red-green memoization with durability levels |
| [**Observable**](https://observablehq.com/) | Notebooks | JavaScript | Language-level reactive dataflow |
| **Excel** | Spreadsheet | C++ | Dirty-marking + calculation chain |
| [**Noria**](https://github.com/mit-pdos/noria) | Database | Rust | Partially-stateful materialized views |
| [**Differential Dataflow**](https://github.com/TimelyDataflow/differential-dataflow) | Stream processing | Rust | Lattice-indexed difference collections |
| [**Incremental**](https://blog.janestreet.com/introducing-incremental/) (Jane Street) | Trading systems | OCaml | Self-adjusting computation with cutoffs |

---

## 2. System-by-System Analysis

### 2.1 Adapton — Demand-Driven Incremental Computation

**Origin**: Hammer, Acar et al., [PLDI 2014](https://doi.org/10.1145/2594291.2594324). Extended from Acar's
[self-adjusting computation](https://www.cs.cmu.edu/~umut/papers/thesis.pdf) (SAC) work at CMU (2002-2005).

**Core idea**: Traditional incremental computation is *push-based* — when an
input changes, all dependents are eagerly recomputed. Adapton inverts this
with *demand-driven* (pull-based) semantics: dirty-mark eagerly, but defer
recomputation until an observer demands the result.

**Architecture**:
- **Demanded Computation Graph (DCG)**: A hierarchical dependency graph
  tracking both data dependencies and control flow. Unlike flat dependency
  graphs, the DCG captures which computations *created* other computations.
- **Two-phase operation**:
  1. **Dirty phase** (push): When an input changes, walk forward through the
     DCG marking all transitive dependents as dirty. This is O(affected nodes).
  2. **Demand phase** (pull): When an observer requests a value, walk backward
     through the DCG. If a node is dirty, check if its inputs actually changed
     (they may have been marked dirty but produced the same value). Only
     recompute if inputs genuinely differ.
- **Clean-but-not-demanded optimization**: If no observer demands a dirty
  node's value, it is never recomputed. This is the key advantage over
  push-based systems.

**Composability**: Adapton's DCG supports nested incremental computations —
an incremental computation can itself be an input to another incremental
computation. The λ_ic^dd calculus formalizes this with an explicit separation
between inner (incremental) and outer (observer) computations.

**Gas City relevance**:
- [Gas City's](s3-architecture-sketch.md) `propagateStale` is exactly Adapton's dirty phase.
- Gas City currently lacks the demand phase — once marked stale, cells are
  eagerly recomputed. Adding demand-driven cleaning would avoid recomputing
  cells that no downstream observer needs.
- The DCG's hierarchical structure maps to Gas City's [molecule/cell nesting](s3-architecture-sketch.md).

**Key paper**: Hammer, M.A. et al. ["Adapton: Composable, Demand-Driven
Incremental Computation."](https://doi.org/10.1145/2594291.2594324) PLDI 2014.

---

### 2.2 Salsa — Red-Green Memoization for Compilers

**Origin**: Matsakis (Rust team), inspired by Adapton + Roslyn's red-green
trees. Powers [rust-analyzer](https://rust-analyzer.github.io/) and rustc's query system.

**Core idea**: Define a program as a set of *tracked queries*
(functions K → V). Inputs are base queries set by the user. Derived queries
are pure functions of other queries. Salsa memoizes results and uses
*revision numbers* to determine when cached values are still valid.

**Architecture — The Red-Green Algorithm**:
1. Salsa maintains a global **revision counter** R. Each input mutation
   increments R.
2. For each memoized query, Salsa stores:
   - The computed value
   - The revision R_computed when it was last computed
   - The list of dependency queries and the revision when each last changed
3. When a query is demanded:
   - If R_computed == R_current → return cached value (green)
   - If any dependency's last-changed revision > R_computed → re-execute (red)
   - If no dependency actually changed → **backdate**: mark the result as
     valid at R_current without re-executing (the key optimization)
4. **Backdating cascade**: If a re-executed query produces the same value as
   before, Salsa backdates the result. Downstream queries that depend on it
   may then also skip re-execution. This is the incremental equivalent of
   "cutoff" — stopping propagation when a value doesn't actually change.

**Durability levels**:
- Inputs are classified by stability: volatile (user code), normal
  (dependencies), durable (stdlib).
- Salsa maintains a **version vector** (one component per durability level)
  instead of a single revision number.
- When only volatile inputs change, queries depending solely on durable
  inputs skip validation entirely.
- In rust-analyzer, this [eliminates ~300ms](https://rust-analyzer.github.io/blog/2023/07/24/durable-incrementality.html) of unnecessary stdlib query
  validation per keystroke.

**Lazy invalidation**: When an input changes, Salsa does O(1) work
(increment revision + record change). All validation work happens lazily
when queries are demanded. This is pull-based like Adapton but with a
different mechanism (revision comparison vs. dirty flags).

**Gas City relevance**:
- Salsa's revision system maps to [Gas City's molecule generations](s3-architecture-sketch.md). Each
  `evolve` creates a new revision. The question is whether cell values from
  the previous generation can be reused.
- **Backdating is critical for LLM workloads**: If an upstream cell is
  re-evaluated and produces semantically equivalent output, downstream cells
  should not be recomputed. This requires a notion of semantic equality for
  LLM outputs (non-trivial but high-value).
- Durability levels map to Gas City's cell stability: some cells (e.g.,
  "list all files in repo") are highly durable; others (e.g., "summarize
  latest changes") are volatile.
- The version vector optimization could save significant token cost by
  skipping recomputation of cells whose inputs haven't changed at their
  durability level.

**Key source**: [salsa-rs.github.io/salsa/reference/algorithm.html](https://salsa-rs.github.io/salsa/reference/algorithm.html)

---

### 2.3 Observable — Language-Level Reactive Dataflow

**Origin**: Mike Bostock ([D3.js](https://d3js.org/) creator). [Observable](https://observablehq.com/) notebooks (2018),
[Observable Framework](https://observablehq.com/framework/) (2024).

**Core idea**: Code cells in a notebook form a DAG based on variable
references. The runtime [topologically sorts cells](https://observablehq.com/@observablehq/reactive-dataflow) and re-executes
downstream cells when upstream values change. Reactivity is at the
*language level* — no API, no library calls, just variable references.

**Architecture**:
- **Static analysis**: Variable definitions and references are inferred at
  parse time. No runtime tracing overhead.
- **Topological execution**: Cells execute in dependency order, not document
  order. A cell at the bottom of the document can be a dependency of a cell
  at the top.
- **Automatic re-execution**: When a cell's value changes, all downstream
  cells re-execute automatically. No manual `stabilize()` call needed.
- **Implicit memoization**: Cells that don't depend on changed inputs are
  not re-executed.

**Gas City relevance**:
- Observable validates Gas City's core metaphor: cells as computational
  units in a reactive DAG, with automatic staleness propagation.
- Observable's static dependency analysis contrasts with Gas City's
  explicit `--deps` wiring. Gas City's approach is more flexible (deps
  can be dynamic) but requires manual specification.
- Observable notebooks are single-user, single-machine. Gas City extends
  the paradigm to multi-agent, distributed execution where "cell
  evaluation" means "dispatch to a [polecat](s3-architecture-sketch.md)."

---

### 2.4 Excel — The Original Reactive Spreadsheet

**Architecture — [Smart Recalculation](https://learn.microsoft.com/en-us/office/client-developer/excel/excel-recalculation)**:
1. **Dependency tracking**: Excel maintains a precedent/dependent graph for
   every formula cell. When data changes, Excel marks the cell and all
   transitive dependents as **dirty**.
2. **Calculation chain**: A pre-computed list of all formulas in dependency
   order. Excel walks the chain, computing each dirty cell. If a cell depends
   on a not-yet-computed cell, it's pushed down the chain (deferred).
3. **Early termination**: Unlike what the name might suggest, Excel does NOT
   stop propagation when a recalculated cell produces the same value. The
   dirty mark propagates unconditionally. (This is a known limitation that
   Adapton and Salsa both address.)
4. **Volatile functions**: `NOW()`, `RAND()`, `INDIRECT()` are always dirty.
   They force recalculation of all dependents every time.
5. **Multi-threaded recalculation**: Excel partitions the calculation chain
   into independent subtrees and evaluates them on separate threads. Cells
   with cross-thread dependencies synchronize via barriers.
6. **Threshold behavior**: Beyond 65,536 dependencies, Excel falls back to
   full recalculation. The smart recalc overhead exceeds the cost of just
   computing everything.

**Gas City relevance**:
- Excel's dirty-marking is Gas City's `propagateStale`. Both are push-based.
- Excel's lack of cutoff (no backdating) means it over-recomputes. Gas City
  should learn from Salsa/Adapton and add cutoff.
- The 65K dependency threshold is a warning: at scale, tracking fine-grained
  dependencies can cost more than just recomputing. Gas City's DAGs are
  small (tens to low hundreds of cells), so this threshold won't bite, but
  it's a design consideration.
- Excel's volatile functions map to Gas City cells that depend on external
  state (e.g., "latest git log"). These should always be considered stale.
- Multi-threaded recalculation maps to multi-polecat evaluation of
  independent cells.

---

### 2.5 Noria — Partially-Stateful Materialized Views

**Origin**: Gjengset et al., MIT PDOS, [OSDI 2018](https://www.usenix.org/conference/osdi18/presentation/gjengset). A dataflow database for
web applications. [GitHub](https://github.com/mit-pdos/noria).

**Core idea**: Compile SQL queries into a dataflow graph that incrementally
maintains materialized views. The key innovation is *partial state*: only
materialize results for query parameters that have been recently requested.
Evict cold entries. Re-materialize on demand.

**Architecture**:
- **Dataflow operators**: Standard relational operators (join, filter,
  aggregate) connected in a DAG. Each operator maintains local state.
- **Incremental updates**: Writes propagate as deltas (insert/delete) through
  the dataflow graph. An article with a million votes gets `count += 1`, not
  a full re-aggregation.
- **Partial materialization**: Views only hold results for "hot" parameters.
  Cold entries are evicted. When a query hits a missing entry, Noria triggers
  *upquery* — a backward traversal through the dataflow graph to reconstruct
  the missing state.
- **Dynamic graph mutation**: Unlike traditional dataflow systems, Noria can
  add new queries (and their corresponding dataflow subgraphs) at runtime
  without restarting the system.

**Gas City relevance**:
- Noria's partial materialization is the database analog of demand-driven
  computation. Gas City could apply the same principle: only evaluate cells
  that are actually observed. Cells that no one is looking at remain stale
  without penalty.
- Noria's upquery mechanism (backward traversal to reconstruct missing
  state) maps to Gas City's potential "demand-driven cleaning" — when an
  observer demands a stale cell, walk backward through the DAG to find
  what needs recomputation.
- Noria's dynamic graph mutation maps to [Gas City's `evolve` operation](s3-architecture-sketch.md) —
  adding new cells and wires to a running computation without restart.
- Noria's delta propagation (deltas, not full values) maps to Gas City's
  potential "delta-aware recomputation" — sending diffs to cells instead
  of full re-evaluation.

---

### 2.6 Differential Dataflow — Lattice-Indexed Differences

**Origin**: McSherry, Murray, Isaacs (Microsoft Research), CIDR 2013.
Extended in the [Timely Dataflow](https://github.com/TimelyDataflow/timely-dataflow) framework. Foundations formalized by
Abadi and McSherry. [GitHub](https://github.com/TimelyDataflow/differential-dataflow).

**Core idea**: Track collections as *sets of differences* indexed by a
*partially ordered set* (lattice) of versions. This generalizes both
streaming (totally ordered time) and iterative (nested loops) computation.

**Architecture**:
- **Difference collections**: Instead of storing the current state of a
  collection, store (element, time, diff) triples. `diff = +1` means the
  element was added at that time; `diff = -1` means removed.
- **Partially ordered times**: Times form a lattice, not a total order.
  This enables:
  - **Nested iteration**: Inner loop iterations are ordered within an outer
    iteration, but independent outer iterations are incomparable.
  - **Concurrent updates**: Updates from different sources can be
    incomparable, merged at a join point.
- **Frontier advancement**: Each operator tracks a *frontier* — the set of
  times at which new differences may still arrive. When the frontier
  advances past a time, all differences at that time are final and can be
  compacted.
- **Arrangement**: Persistent, indexed representations of difference
  collections that enable efficient random access. Critical for joins.

**DBSP relationship**: [DBSP](https://doi.org/10.14778/3611479.3611521) (Budiu et al.) is a simplified version where
time is a single totally-ordered counter. DBSP trades expressiveness
(no nested iteration) for simplicity and automatic incrementalization
of arbitrary SQL queries.

**Gas City relevance**:
- Differential dataflow's lattice-indexed versions could model Gas City's
  molecule generations. Each generation is a version. Across-generation
  evolution creates a partial order (not all generations are linear — forked
  experiments create branches).
- The difference collection representation could track *what changed*
  between molecule generations, enabling efficient delta propagation.
- Frontier advancement maps to Gas City's freshness tracking: once all
  upstream cells at a given generation are evaluated, the frontier advances
  and downstream cells can be computed.
- **Caveat**: Differential dataflow is designed for high-throughput stream
  processing (millions of updates/second). Gas City's workload is the
  opposite: few cells, expensive evaluation (LLM calls). The overhead of
  maintaining difference collections may not pay off at Gas City's scale.

---

### 2.7 Incremental (Jane Street) — Self-Adjusting Computation for Trading

**Origin**: Developed at [Jane Street](https://blog.janestreet.com/introducing-incremental/) for trading systems. OCaml library.
Based on Acar's [self-adjusting computation](https://www.cs.cmu.edu/~umut/papers/thesis.pdf) theory.

**Core idea**: Build a computation graph where nodes are incremental values.
When inputs change, explicitly `stabilize()` the graph to propagate updates
efficiently. The graph structure can change dynamically at runtime.

**Architecture**:
- **Variables**: Input nodes created with `Var.create`. Setting a variable
  marks it for propagation.
- **Map/bind combinators**: `map` creates a static dependency (the graph
  structure doesn't change). `bind` creates a dynamic dependency (the graph
  structure can change based on data values).
- **Observers**: Output nodes that tell the system which computations matter.
  Nodes not reachable from any observer are not maintained.
- **Stabilization**: Explicit `stabilize()` call propagates all pending
  changes through the graph. Changes are batched — multiple input
  modifications are propagated in a single pass.
- **Cutoff**: If a recomputed node produces the same value as before,
  propagation stops (early termination). This is the key optimization
  that Excel lacks.
- **Performance**: ~30ns per node firing. Beneficial when individual
  computations are expensive relative to this overhead, or when the
  recomputation subgraph is much smaller than the full graph.
- **Generative functor**: Creates independent computational worlds with
  fresh types, preventing accidental cross-world value mixing.
- **Unordered array fold**: For reversible operations (e.g., sum), use
  inverse functions to achieve O(1) updates instead of O(n) recomputation.

**Gas City relevance**:
- Incremental's observer model is exactly what Gas City needs for
  demand-driven computation. Only cells reachable from an observer are
  maintained. Gas City observers = "which cells does the user/downstream
  agent actually need?"
- The `stabilize()` call maps to Gas City's evaluation trigger. Rather
  than continuous reactivity, Gas City should batch changes and stabilize
  on demand (when a polecat asks for a cell's value).
- Cutoff is critical: if an LLM re-evaluation produces semantically
  equivalent output, stop propagation. This requires semantic comparison
  (harder than structural equality but essential for token savings).
- The 30ns/node overhead is irrelevant for Gas City (LLM calls cost
  seconds, not nanoseconds). The overhead concern is tracking metadata,
  not computation.

---

## 3. Key Question: What Happens When Agent Outputs Are Cells in a Reactive DAG?

### 3.1 The fundamental difference from traditional reactive systems

In every system surveyed, cell evaluation is:
- **Fast** (nanoseconds to milliseconds)
- **Deterministic** (same inputs → same output)
- **Cheap** (CPU cycles, not dollars)

In Gas City, cell evaluation is:
- **Slow** (seconds to minutes per LLM call)
- **Non-deterministic** (same prompt → different output each time)
- **Expensive** (tokens cost money)

This inverts the optimization priorities:

| Traditional | Gas City |
|-------------|----------|
| Minimize recomputation overhead | Minimize recomputation count |
| Fine-grained dependency tracking | Coarse-grained is fine |
| Cutoff saves CPU cycles | Cutoff saves dollars |
| Eager recomputation is acceptable | Eager recomputation is wasteful |
| Determinism enables exact cutoff | Non-determinism requires semantic cutoff |

### 3.2 Semantic cutoff is the critical missing primitive

In Salsa, cutoff is trivial: compare the old and new values with `==`.
In Gas City, two LLM outputs may differ textually but be semantically
equivalent. "The function returns 42" and "This function's return value
is 42" should trigger cutoff, but string equality says they differ.

**Options for semantic cutoff**:
1. **Structural extraction**: Parse LLM output into structured data (JSON,
   typed fields). Compare structures, not text. This is the most reliable
   approach but requires structured output from cells.
2. **LLM-as-judge**: Use a fast model to determine if two outputs are
   semantically equivalent. Adds cost but handles free-text.
3. **Hash-of-intent**: Hash the key claims/decisions in the output, not the
   full text. Requires cell-specific extractors.
4. **Conservative default**: Treat all re-evaluations as producing different
   values (no cutoff). This is safe but wasteful. It's what Gas City does
   today.

**Recommendation**: Start with structural extraction (option 1) for cells
with typed outputs. Fall back to conservative (option 4) for free-text.
Add LLM-as-judge (option 2) as an optimization for high-value cells where
cutoff would save significant downstream recomputation.

### 3.3 Non-determinism creates branch instability

In a deterministic system, re-evaluating a cell with unchanged inputs
produces the same output. In Gas City, re-evaluation produces a *different*
output. This means:
- Cutoff never triggers for re-evaluated cells (unless using semantic cutoff)
- Every re-evaluation cascades through the entire downstream DAG
- The system is inherently "noisy" — small upstream changes cause large
  downstream recomputation waves

**Mitigation**: Wolfram's branch stability protocol (run 3 times, extract
canonical form) identifies which cells are inherently stable vs. volatile.
Stable cells can be cached aggressively; volatile cells should be close to
observers (short downstream chains) to minimize cascade cost.

---

## 4. Key Question: Staleness Propagation at Scale

### 4.1 Push-based (dirty marking) scales linearly

All surveyed systems use some form of push-based dirty marking:
- Excel: mark all transitive dependents as dirty
- Adapton: mark all transitive dependents in the DCG
- Salsa: increment revision counter (O(1)), validate lazily
- Observable: re-execute all downstream cells

For Gas City's DAG sizes (tens to low hundreds of cells), push-based dirty
marking is trivially fast. The concern is not the marking itself but the
*recomputation* it triggers.

### 4.2 The cascade problem

In a traditional system, marking 100 cells dirty and recomputing them takes
milliseconds. In Gas City, marking 100 cells dirty could trigger 100 LLM
calls costing minutes and dollars.

**The cascade problem**: A single upstream change can trigger an exponential
number of recomputations in a diamond-shaped DAG. If cell A feeds cells B
and C, which both feed cell D, a change to A triggers recomputation of B,
C, and D. But D is triggered twice (once by B, once by C). In a traditional
system, the second trigger is cheap (cutoff on same value). In Gas City,
the second trigger is another LLM call.

**Solutions from the surveyed systems**:

1. **Salsa's backdating**: Recompute B. If B produces the same value,
   backdate it. D's dependency on B hasn't changed, so D may not need
   recomputation. (Requires semantic cutoff.)

2. **Incremental's stabilization**: Batch all changes. During stabilization,
   process nodes in topological order. Each node is processed exactly once.
   The second trigger from C doesn't cause a re-recomputation of D because
   D hasn't been processed yet when C finishes.

3. **Noria's partial materialization**: Don't maintain all cells. Only
   maintain the cells that observers demand. Cells deep in the DAG that
   no one is looking at stay stale without penalty.

4. **Differential dataflow's compaction**: Merge multiple deltas at the
   same frontier point. D receives a single merged delta from both B and
   C, not two separate triggers.

**Recommendation for Gas City**: Use Incremental's approach — batch changes
and stabilize in topological order, processing each cell at most once per
stabilization pass. Combine with Noria's partial materialization: only
recompute cells that have active observers.

### 4.3 Staleness is cheaper than recomputation

A key insight from Noria: maintaining stale state is free. The cost is in
recomputation, not in marking things stale. Gas City should embrace
staleness as the default state:
- Mark stale eagerly (push-based, like today)
- Recompute lazily (pull-based, on demand)
- Budget recomputation (Feynman's perturbation bound: total cost is
  tree-level * 1/(1-ε))

---

## 5. Key Question: Reactive vs Pull-Based Recomputation

### 5.1 The spectrum

The surveyed systems span a spectrum from fully reactive to fully lazy:

```
Fully Reactive                                        Fully Lazy
     |                                                     |
  Observable    Excel    Incremental    Adapton    Noria/Salsa
  (auto-run)   (dirty+   (stabilize    (demand-   (on-demand
               recompute  on demand)    driven)    materialization)
               eagerly)
```

### 5.2 Where Gas City should sit

Gas City's workload characteristics push it toward the lazy end:
- **Expensive evaluation** → don't compute unless needed
- **Non-deterministic** → recomputation may produce worse results
- **Multi-agent** → not all agents need all cells
- **Budget-constrained** → every recomputation costs tokens

**Recommendation**: Gas City should use a **demand-driven** model
(Adapton-style) with **explicit stabilization** (Incremental-style):

1. **Eager staleness propagation** (already implemented as `propagateStale`).
   When an input changes, mark all downstream cells stale. This is O(n) in
   the number of affected cells and costs zero LLM calls.

2. **Lazy recomputation on demand**. When a polecat or user requests a
   cell's value:
   - If fresh → return cached value
   - If stale → check if upstream inputs actually changed (Salsa's
     validation). If not, mark fresh (backdate). If yes, recompute.

3. **Batched stabilization**. When multiple cells are demanded at once
   (e.g., a formula step needs several inputs), stabilize all of them in
   a single topologically-ordered pass. Each cell is evaluated at most once.

4. **Observer-driven scope**. Only maintain freshness for cells reachable
   from active observers (Incremental's model). Cells with no observers
   can be stale indefinitely without cost.

### 5.3 The hybrid insight

No single system perfectly matches Gas City's needs. The recommendation is
a hybrid:

| Component | Borrowed From | Why |
|-----------|---------------|-----|
| Dirty marking | Excel / Adapton | Simple, proven, O(n) |
| Lazy recomputation | Adapton / Noria | Avoid unnecessary LLM calls |
| Backdating / cutoff | Salsa | Stop cascades when values don't change |
| Batched stabilization | Incremental | Evaluate each cell at most once per pass |
| Observer scoping | Incremental / Noria | Only maintain what's needed |
| Durability levels | Salsa | Skip validation for stable inputs |
| Partial materialization | Noria | Evict cold cells, reconstruct on demand |

---

## 6. Comparative Analysis Matrix

| Feature | Adapton | Salsa | Observable | Excel | Noria | Diff. DF | Incremental |
|---------|---------|-------|------------|-------|-------|----------|-------------|
| Staleness propagation | Push (dirty) | Lazy (revision) | Push (re-run) | Push (dirty) | Push (delta) | Push (diff) | Push (dirty) |
| Recomputation trigger | Pull (demand) | Pull (query) | Push (auto) | Push (chain) | Pull (upquery) | Push (frontier) | Pull (stabilize) |
| Cutoff / backdating | Yes | Yes (backdate) | Implicit | **No** | N/A (deltas) | N/A (diffs) | Yes |
| Dynamic graph | Yes (DCG) | No | No | No | Yes (runtime) | No | Yes (bind) |
| Partial evaluation | Yes (demand) | Yes (lazy) | No | No | Yes (partial) | No | Yes (observers) |
| Nested iteration | No | No | No | No | No | **Yes** (lattice) | No |
| Concurrency | No | **Yes** (lock-free) | No | Yes (threads) | Yes (sharding) | Yes (workers) | No |
| Persistence | No | No | No | File save | Database | Arrangement | No |

---

## 7. Applicability to Gas City

### 7.1 Direct mappings

| Gas City Concept | Reactive Dataflow Analog | Best Source |
|------------------|--------------------------|------------|
| Cell | Incremental variable / Adapton thunk | Incremental |
| `propagateStale` | Dirty marking / invalidation | Excel, Adapton |
| `bd ready` (ready set) | Topological frontier | Diff. Dataflow |
| Molecule generation | Revision / version | Salsa |
| `evolve` | Dynamic graph mutation | Noria |
| Cell evaluation | Query execution | Salsa |
| Digest (crystal) | Compacted arrangement | Diff. Dataflow |
| Observer (user/agent) | Observer node | Incremental |
| Durability (stdlib vs user code) | Input durability levels | Salsa |

### 7.2 What Gas City should adopt

**Immediate (v1 — near-term additions)**:

1. **Demand-driven cleaning**: Don't recompute stale cells until someone
   asks for their value. Cost: near-zero implementation effort. Savings:
   proportional to unobserved DAG breadth. Source: Adapton.

2. **Topological stabilization**: When evaluating, process cells in
   topological order. Each cell evaluated at most once per pass. Prevents
   the diamond-DAG double-evaluation problem. Source: Incremental.

3. **Observer registration**: Track which cells have active observers
   (agents waiting on their value). Unobserved cells don't trigger
   recomputation. Source: Incremental.

**Near-term (v1.5)**:

4. **Structural cutoff**: For cells with typed/structured output, compare
   structures to determine if the value actually changed. Stop downstream
   propagation if it didn't. Source: Salsa (backdating).

5. **Durability classification**: Classify cells by volatility. Cells
   depending only on durable inputs (repo structure, static config) skip
   validation when only volatile inputs (latest commit, user edits) change.
   Source: Salsa.

**Medium-term (v2)**:

6. **Delta-aware prompts**: Instead of full re-evaluation, send cells a
   diff of what changed upstream. "Your previous output was X, input
   changed by ΔY, update accordingly." Source: Noria (delta propagation),
   Differential Dataflow.

7. **Partial materialization**: Allow cells to be evicted from memory.
   Reconstruct on demand via upquery. Source: Noria.

### 7.3 What Gas City should NOT adopt

- **Differential dataflow's lattice-indexed differences**: Over-engineered
  for Gas City's scale. Designed for millions of records, not tens of cells.
- **Observable's automatic re-execution**: Too eager for expensive LLM
  evaluation. Gas City needs explicit demand, not auto-run.
- **Excel's lack of cutoff**: A known limitation. Gas City must have cutoff.
- **Salsa's lock-free concurrency**: Gas City's concurrency is at the
  polecat level (process isolation), not the thread level. Lock-free
  data structures are irrelevant.

---

## 8. Open Questions

1. **Semantic equality for LLM outputs**: How to implement cutoff when
   outputs are non-deterministic free text? Structural extraction works
   for typed cells; what about synthesis/analysis cells?

2. **Staleness budget allocation**: Feynman's bound says total cost is
   tree-level * 1/(1-ε). How should Gas City allocate its recomputation
   budget across cells? Prioritize cells closest to observers? Cells with
   highest downstream fan-out?

3. **Dynamic dependency discovery**: In Adapton and Incremental, dependencies
   are discovered at runtime (dynamic graphs). Should Gas City allow cells
   to discover new dependencies during evaluation? (Currently, deps are
   static in the proto.)

4. **Cross-molecule staleness**: When a digest from molecule A is referenced
   by molecule B, how does staleness propagate across molecule boundaries?
   The Salsa revision model (per-molecule revisions) vs. a global revision
   counter.

5. **Convergence detection**: When a cell is re-evaluated and produces a
   different result, should Gas City re-evaluate it again to check for
   convergence? (Feynman's perturbation series.) At what point is the
   result "stable enough" to propagate?

---

## 9. Synthesis: The Gas City Reactive Model

Gas City should implement a **demand-driven, batch-stabilized reactive
dataflow** engine combining ideas from Adapton (demand-driven), Salsa
(backdating + durability), Incremental (observers + stabilization), and
Noria (partial materialization):

```
Input changes
    │
    ▼
[Eager dirty marking]          ← Excel/Adapton: O(n), zero LLM cost
    │
    ▼
[Wait for demand]              ← Adapton: don't compute unobserved cells
    │
    ▼
[Batch stabilization]          ← Incremental: topological order, once per cell
    │
    ▼
[Validate inputs]              ← Salsa: check if inputs actually changed
    │                               If not → backdate, skip recomputation
    ▼
[Evaluate cell (LLM call)]     ← The expensive part
    │
    ▼
[Structural cutoff]            ← Salsa: if output unchanged, stop cascade
    │
    ▼
[Propagate to next cell]       ← Continue stabilization pass
```

This model minimizes LLM calls (the dominant cost) while maintaining
correctness (all observed cells reflect current inputs). The key insight
is that Gas City's cost structure — cheap marking, expensive evaluation —
is the exact opposite of traditional reactive systems, and the reactive
model should be inverted accordingly: eager marking, lazy evaluation.

---

## References

### Primary Sources
- Hammer, M.A. et al. ["Adapton: Composable, Demand-Driven Incremental Computation."](https://doi.org/10.1145/2594291.2594324) PLDI 2014.
- McSherry, F. et al. ["Differential Dataflow."](https://github.com/TimelyDataflow/differential-dataflow/blob/master/differentialdataflow.pdf) CIDR 2013.
- Gjengset, J. et al. ["Noria: Dynamic, Partially-Stateful Data-Flow for High-Performance Web Applications."](https://www.usenix.org/conference/osdi18/presentation/gjengset) OSDI 2018.
- Katsumata, S. ["Parametric Effect Monads and Semantics of Effect Systems."](https://doi.org/10.1145/2535838.2535846) POPL 2014.
- Acar, U.A. ["Self-Adjusting Computation."](https://www.cs.cmu.edu/~umut/papers/thesis.pdf) CMU PhD Thesis, 2005.

### System Documentation
- Salsa red-green algorithm: [salsa-rs.github.io/salsa/reference/algorithm.html](https://salsa-rs.github.io/salsa/reference/algorithm.html)
- rust-analyzer durable incrementality: [rust-analyzer.github.io/blog/2023/07/24/durable-incrementality.html](https://rust-analyzer.github.io/blog/2023/07/24/durable-incrementality.html)
- Jane Street Incremental: [blog.janestreet.com/introducing-incremental/](https://blog.janestreet.com/introducing-incremental/)
- Observable reactive dataflow: [observablehq.com/@observablehq/reactive-dataflow](https://observablehq.com/@observablehq/reactive-dataflow)
- Excel recalculation: [learn.microsoft.com/en-us/office/client-developer/excel/excel-recalculation](https://learn.microsoft.com/en-us/office/client-developer/excel/excel-recalculation)
- Noria: [github.com/mit-pdos/noria](https://github.com/mit-pdos/noria)
- Differential Dataflow: [github.com/TimelyDataflow/differential-dataflow](https://github.com/TimelyDataflow/differential-dataflow)
