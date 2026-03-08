# S3: Architecture Sketch — What Gas City Adds

**Bead**: gt-5km | **Date**: 2026-03-08 | **Author**: polecat/imperator
**Sources**: [S1](s1-gap-analysis.md) (Gap Analysis), [S2](s2-abstraction-map.md) (Abstraction Map), R1-R6 (Phase 1 Research)

---

## Executive Summary

Gas City is not a rewrite. It is four architectural layers composed on top
of Gas Town's unchanged primitives — beads, formulas, polecats, mail, Dolt,
Witness, Refinery, Mayor. The layers are: a **reactive dataflow substrate**
(cells + DAGs), a **learning pipeline** (reflections + crystals), a **quality
gate** (Critic Lens), and eventually a **protocol membrane** (MCP/A2A).

This document specifies each layer's concrete design: what it adds, how it
composes with existing Gas Town primitives, what changes in the schema and
CLI, the migration path from Gas Town to Gas City, and the risks that could
kill the project.

**The governing constraint**: Gas Town's simplicity is its moat. Every addition
must pass the **Durability Test** ("Would a more capable base model make this
unnecessary?") and the **Simplicity Test** ("Does this reduce operational
complexity per unit of output?"). If the answer to either is no, defer.

---

## 1. Design Principles

### 1.1 Gas City Is a Superset, Not a Successor

Gas Town continues to run unmodified. Gas City layers are opt-in. A rig
that doesn't need reactive cells doesn't pay for them. A polecat that
doesn't generate reflections still completes work. The upgrade path is
incremental adoption, not migration.

### 1.2 The Inverted Cost Model

Gas City's reactive computation operates under an inverted cost structure
([R2](r2-reactive-dataflow.md) §4):

| Operation | Traditional reactive | Gas City |
|-----------|---------------------|----------|
| Marking dirty | Moderate | **Cheap** (flag set on bead, no LLM) |
| Evaluating a cell | Cheap (ns-ms) | **Expensive** (seconds, dollars — LLM call) |
| Tracking dependencies | Expensive at scale | **Cheap** (small DAGs, SQL queries) |
| Over-computing | Acceptable | **Catastrophic** (wasted budget) |

**Design consequence**: Mark aggressively, evaluate lazily. Never recompute
a cell that no observer has demanded. Never propagate past a cutoff.

### 1.3 Composition Over Invention

Each Gas City abstraction maps to existing Gas Town primitives with minimal
extensions:

| Gas City concept | Implemented as |
|------------------|---------------|
| Reactive Cell | Bead + `dirty` flag + `reactive_deps` field |
| Computation DAG | Molecule + input/output declarations on steps |
| Reflection | Structured bead fields + formula step |
| Skill Crystal | Typed bead in a `crystals` table |
| Critic review | Refinery formula step |
| Protocol endpoint | Thin HTTP adapter over `bd` CLI |

No new storage systems. No new coordination protocols. No new process types.

---

## 2. Layer 1: Reactive Dataflow Substrate

### 2.1 Reactive Cells

**What changes**: Beads gain three new fields and one new CLI command.

**Schema additions** (Dolt table `beads`):

```sql
ALTER TABLE beads ADD COLUMN dirty BOOLEAN DEFAULT FALSE;
ALTER TABLE beads ADD COLUMN cell_value TEXT;           -- cached output
ALTER TABLE beads ADD COLUMN cell_evaluator TEXT;       -- prompt template or command
ALTER TABLE beads ADD COLUMN cutoff_mode VARCHAR(20)    -- 'exact', 'structural', 'semantic'
  DEFAULT 'exact';
```

**Dependency reuse**: The existing `bd dep add` system already tracks edges
between beads. Reactive dependencies ARE regular dependencies, annotated
with a `reactive` flag:

```sql
ALTER TABLE dependencies ADD COLUMN reactive BOOLEAN DEFAULT FALSE;
```

When `bd dep add --reactive child parent` is used, the edge participates
in dirty-marking propagation.

**CLI additions**:

```bash
bd mark-stale <id>          # Manually mark a cell dirty + propagate
bd stabilize <id>           # Demand-evaluate a stale cell (lazy recomputation)
bd cell set <id> --value    # Set cell value directly (source cells)
bd cell deps <id>           # Show reactive dependency graph
```

**Dirty-marking algorithm** (eager, synchronous, cheap):

```
mark_stale(cell_id):
    if cell.dirty: return           # Already dirty, stop
    cell.dirty = true
    for each downstream in reactive_dependents(cell_id):
        mark_stale(downstream)
```

This is a depth-first traversal of the reactive dependency graph. Cost:
O(edges), no LLM calls. A change to a source cell propagates dirty flags
through the entire downstream graph in milliseconds.

**Lazy recomputation** (expensive, on-demand):

```
stabilize(cell_id):
    if not cell.dirty: return cell.value    # Clean — return cached
    for each upstream in reactive_deps(cell_id):
        stabilize(upstream)                  # Ensure inputs are fresh
    new_value = evaluate(cell.evaluator, inputs)
    if cutoff(cell.cutoff_mode, cell.value, new_value):
        cell.dirty = false                   # Value unchanged — cutoff
        return cell.value
    cell.value = new_value
    cell.dirty = false
    return new_value
```

Recomputation is triggered ONLY when someone calls `bd stabilize` or when
an observer (e.g., `gt prime`, a polecat reading cell values) demands a
fresh value. No background recomputation.

**Cutoff predicates**:

| Mode | Comparison | Use case |
|------|-----------|----------|
| `exact` | String equality | Deterministic outputs (build results, test results) |
| `structural` | Normalized JSON/YAML comparison | Structured data (configs, schemas) |
| `semantic` | LLM-judged equivalence | Natural language outputs (research, analysis) |

The `semantic` mode IS an LLM call, making it expensive. Use only for cells
whose downstream graph is expensive enough to justify the comparison cost.
Default to `exact` — it's free and correct for most automation outputs.

**Dynamic dependency discovery**: A cell's true dependencies aren't known
until it evaluates. Solution: the evaluator records which cells it read
during execution. After evaluation, the dependency set is updated:

```
evaluate(evaluator, inputs):
    tracker = DependencyTracker()
    result = run_evaluator(evaluator, inputs, tracker)
    cell.reactive_deps = tracker.recorded_reads  # Update deps
    return result
```

This mirrors [Adapton](https://github.com/Adapton/adapton.rust)'s approach ([R2](r2-reactive-dataflow.md)): dependencies are recorded, not
declared.

### 2.2 Computation DAGs (Reactive Molecules)

**What changes**: Formula steps gain input/output declarations. The molecule
scheduler gains topological awareness.

**Formula step schema extension**:

```yaml
# Current formula step:
- name: "implement"
  body: "Do the actual implementation work..."

# Gas City formula step:
- name: "implement"
  inputs: ["requirements"]         # reads from step outputs
  outputs: ["code_changes"]        # produces for downstream steps
  body: "Do the actual implementation work..."
```

Steps without `inputs`/`outputs` declarations behave as today (linear
sequence). This is backwards-compatible — existing formulas work unchanged.

**Scheduling rules**:

1. Steps with satisfied inputs are **eligible** for dispatch
2. Independent eligible steps MAY execute in parallel (if polecats available)
3. A step whose inputs haven't changed since last evaluation is **skipped**
   (cutoff at the DAG level)
4. If a step fails, its outputs are marked stale and downstream steps are
   not dispatched

**Example: Parallel review and build**

```yaml
steps:
  - name: "load-context"
    outputs: ["context"]
  - name: "setup-branch"
    inputs: ["context"]
    outputs: ["branch"]
  - name: "implement"
    inputs: ["context", "branch"]
    outputs: ["code_changes"]
  - name: "self-review"
    inputs: ["code_changes"]       # Independent of build
    outputs: ["review_findings"]
  - name: "build-check"
    inputs: ["code_changes"]       # Independent of review
    outputs: ["build_result"]
  - name: "commit"
    inputs: ["code_changes", "review_findings", "build_result"]
    outputs: ["commits"]
  - name: "submit"
    inputs: ["commits"]
```

Steps `self-review` and `build-check` can execute in parallel after
`implement` completes. This is a scheduling optimization, not a semantic
change — the formula still expresses the same work.

**Cycle handling**: DAGs cannot express cycles. Reflection loops (implement
→ review → revise → review) are handled by **bounded unrolling**: the formula
declares a maximum iteration count, and the scheduler unrolls the cycle
into a finite DAG:

```yaml
- name: "implement-review-cycle"
  max_iterations: 3
  cycle:
    - name: "implement"
      outputs: ["code"]
    - name: "review"
      inputs: ["code"]
      outputs: ["findings"]
    - name: "revise"
      inputs: ["findings"]
      outputs: ["code"]
      converge_on: "findings.severity == 'none'"
```

If the convergence predicate is satisfied or `max_iterations` is reached,
the cycle exits and downstream steps proceed.

**Multi-agent DAG steps**: Each step is assigned to exactly one polecat.
Fan-out parallelism happens at the step level (independent steps → different
polecats), not within a step. This preserves Gas Town's isolation model —
each polecat has its own worktree and works independently.

---

## 3. Layer 2: Learning Pipeline

### 3.1 Reflection Cycles

**What changes**: A new formula step in `mol-polecat-work`, new bead fields,
and a retrieval mechanism in `gt prime`.

**New formula step** (inserted between current steps 7 and 8):

```yaml
- name: "reflect"
  body: |
    Generate a structured reflection on this completion.

    1. Review your commits: `git log origin/{{base_branch}}..HEAD`
    2. Review the original bead: `bd show {{issue}}`
    3. Generate reflection with these fields:

    ```json
    {
      "what_worked": ["List of approaches/patterns that succeeded"],
      "what_failed": ["List of approaches that didn't work and why"],
      "would_do_differently": ["What you'd change on a second attempt"],
      "patterns_discovered": ["Reusable patterns worth remembering"],
      "difficulty": "trivial|routine|challenging|novel",
      "blockers_hit": ["Any blockers encountered and how resolved"]
    }
    ```

    4. Persist to bead:
    ```bash
    bd update {{issue}} --reflection '<json>'
    ```

    This step is SKIPPED for trivial tasks (< 2 commits, no blockers hit).
```

**Schema additions**:

```sql
ALTER TABLE beads ADD COLUMN reflection JSON;
ALTER TABLE beads ADD COLUMN difficulty VARCHAR(20);
```

**Retrieval in `gt prime`**: When a polecat primes, it receives relevant
reflections from past completions. The retrieval mechanism is SQL-based
(not embedding search — Dolt is a SQL database):

```sql
SELECT b.id, b.title, b.reflection
FROM beads b
WHERE b.reflection IS NOT NULL
  AND b.status = 'closed'
  AND b.difficulty IN ('challenging', 'novel')
  AND (
    -- Title keyword overlap with current task
    b.title LIKE '%keyword1%'
    OR b.title LIKE '%keyword2%'
    -- Same project/area
    OR b.labels LIKE '%same-label%'
  )
ORDER BY b.updated_at DESC
LIMIT 5;
```

Keywords are extracted from the current bead's title and description. This
is deliberately simple — SQL LIKE queries on a small dataset. No vector
store, no embedding model, no RAG pipeline. If the reflection corpus
grows large enough to need semantic search, that's a future problem.

**Consolidation**: Periodically (weekly or after N reflections accumulate),
a consolidation pass synthesizes raw reflections into higher-level insights.
This is a scheduled formula, not a polecat step:

```bash
bd reflection consolidate --since 7d
```

The consolidation produces a summary bead with aggregated patterns, stored
as a "meta-reflection" that future retrievals prioritize over raw
reflections.

### 3.2 Skill Crystals (Trial)

**What changes**: A new Dolt table, an extraction command, and a matching
step in `gt prime`.

**Schema** (new table `crystals`):

```sql
CREATE TABLE crystals (
    id VARCHAR(20) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    trigger_description TEXT,       -- when to suggest this crystal
    solution_template TEXT,         -- steps or code skeleton
    provenance_bead VARCHAR(20),   -- which completion it came from
    category VARCHAR(50),          -- 'infrastructure', 'debugging', etc.
    times_used INT DEFAULT 0,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Extraction** (manual initially, automated later):

```bash
bd crystal extract <bead-id>     # Extract crystal from a closed bead
```

The extraction examines the bead's reflection, commit diff, and description
to identify reusable patterns. Initially this is a prompted LLM call;
later it could be automated as a post-completion formula step.

**Matching in `gt prime`**: Similar to reflection retrieval — SQL keyword
matching on the crystal's `trigger_description` against the current bead.
Crystals with higher `times_used` are ranked higher.

**Garbage collection**: Crystals unused for 90 days are marked `stale`.
Stale crystals are excluded from matching. After 180 days of staleness,
they're archived (not deleted — Dolt history preserves them).

**Trial scope**: Initially limited to infrastructure patterns (adding
formula steps, configuring MCP servers, fixing common Dolt issues). Expand
only after evaluating hit rate on the initial category.

---

## 4. Layer 3: Quality Gate

### 4.1 Critic Lens

**What changes**: A new step in the Refinery's merge queue pipeline and
a structured review schema.

**Refinery pipeline modification**:

```
Current:  MR arrives → Run gates (build/test/lint) → Merge or bisect
Gas City: MR arrives → Critic review → Run gates → Merge or bisect
```

The Critic runs BEFORE expensive gates. A BLOCK verdict prevents wasting
gate compute on code that has known issues.

**Critic implementation**: The Critic is a lightweight polecat (or a direct
LLM call without polecat overhead) that receives:

1. The full diff (`git diff origin/main...HEAD`)
2. The bead description (what was the task?)
3. The bead's reflection (if available — what did the implementer think?)

**Critic prompt structure**:

```
You are reviewing a code change. Your job is adversarial: find problems.

## The task
{bead_description}

## The diff
{diff}

## Self-assessment (from implementer)
{reflection}

## Review checklist
1. Does the diff match the stated task? (requirement match)
2. Are there bugs? (logic errors, off-by-one, null handling)
3. Are there security issues? (injection, auth bypass, secrets)
4. Are there style violations? (naming, formatting, patterns)
5. Is there dead code, debug prints, or TODO comments?

## Output format
{
  "verdict": "PASS" | "CONCERNS" | "BLOCK",
  "findings": [
    {
      "severity": "info" | "warning" | "error",
      "file": "path/to/file",
      "line": 42,
      "description": "What's wrong",
      "suggestion": "How to fix it"
    }
  ],
  "confidence": 0.0-1.0
}
```

**Verdict thresholds** (calibrated over time):

| Condition | Verdict |
|-----------|---------|
| No findings with severity > info | PASS |
| Warnings present, confidence < 0.7 | CONCERNS (logged, doesn't block) |
| Warnings present, confidence ≥ 0.7 | CONCERNS (logged, Refinery decides) |
| Any error finding, confidence ≥ 0.8 | BLOCK (quarantine for review) |

**Advisory mode**: For the first N MRs (suggest N=50), the Critic runs in
advisory mode: all verdicts are logged but nothing blocks. This generates
calibration data. After calibration, promote to blocking mode.

**Critic independence**: The Critic MUST use a different system prompt than
the implementing polecat. Ideally, use a different model or temperature
setting. The adversarial framing is the key mechanism — a model reviewing
its own output with the same persona exhibits confirmation bias.

**Review bead**: Each Critic review creates a typed bead attached to the
MR, providing an audit trail. The Refinery can query historical reviews
to calibrate thresholds.

---

## 5. Layer 4: Protocol Membrane (Deferred — Design Only)

This layer is deferred per [S2](s2-abstraction-map.md)'s recommendation. Included here as a design
sketch for future reference.

### 5.1 Outbound: Agent Cards

Each polecat's role definition maps to an A2A Agent Card:

```json
{
  "name": "gastown-polecat",
  "description": "Autonomous code implementation agent",
  "url": "https://gastown.example.com/.well-known/agent-card.json",
  "capabilities": {
    "streaming": false,
    "pushNotifications": false
  },
  "skills": [
    {
      "id": "code-implementation",
      "name": "Code Implementation",
      "description": "Implements features, fixes bugs, writes tests"
    }
  ],
  "authentication": {
    "schemes": ["oauth2"]
  }
}
```

### 5.2 Inbound: Task Translation

External A2A tasks map to beads:

```
A2A Task → bd create --from-a2a <task-json>
```

The translation preserves A2A task lifecycle (submitted → working →
completed/failed) by mapping to bead statuses (open → in_progress →
closed).

### 5.3 Federation

Multiple Gas Town rigs communicate via A2A instead of custom mail:

```
Rig A → A2A task → Rig B (replaces gt mail send between rigs)
```

This enables heterogeneous deployment: Rig A might run Gas Town, Rig B
might run a different agent framework — they coordinate through A2A.

**Implementation constraint**: The membrane must be thin. A single HTTP
adapter that translates between A2A JSON and `bd` CLI commands. No custom
servers, no persistent connections, no complex auth flows until needed.

---

## 6. Composition: The Full Gas City Stack

```
┌─────────────────────────────────────────────────────────────┐
│                         GAS CITY                            │
│                                                             │
│  Layer 4: Protocol Membrane (DEFERRED)                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ A2A Agent Cards · MCP tool consumption · Federation   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  Layer 3: Quality Gate                                      │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Critic Lens: adversarial review → PASS/CONCERNS/BLOCK │  │
│  │ Runs before gates · Advisory mode first · Calibrated  │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  Layer 2: Learning Pipeline                                 │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Reflection Cycles: structured post-completion memory   │  │
│  │ Skill Crystals: reusable patterns from completions     │  │
│  │ Retrieval: SQL-based matching during gt prime          │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  Layer 1: Reactive Dataflow Substrate                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Reactive Cells: dirty marking + lazy evaluation        │  │
│  │ Computation DAGs: topological scheduling + parallelism │  │
│  │ Cutoff predicates: exact/structural/semantic           │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  ═══════════════════════════════════════════════════════════ │
│  Gas Town Foundation (UNCHANGED)                            │
│  Beads · Formulas · Polecats · Mail · Dolt · Witness        │
│  Refinery · Mayor · Capability Ledger · Worktrees           │
│  ═══════════════════════════════════════════════════════════ │
└─────────────────────────────────────────────────────────────┘
```

### 6.1 Cross-Layer Interactions

**Reactive Cells ↔ Critic Lens**: A Critic BLOCK on an MR can mark
downstream reactive cells as stale. If the blocked code was a cell's
evaluator, cells depending on it know they can't trust their cached
values.

**Reflection Cycles ↔ Skill Crystals**: Reflections feed crystal
extraction. Crystals reference their source reflections for provenance.
Consolidation (§3.1) identifies cross-reflection patterns that become
crystal candidates.

**Computation DAGs ↔ Reflection Cycles**: DAG step outputs become inputs
to reflection. A polecat's reflection includes which DAG path it took,
which steps were skipped by cutoff, and where parallelism helped or
hindered.

**Reactive Cells ↔ Computation DAGs**: DAG steps are cells. When a step's
output changes, downstream steps are marked stale. The DAG scheduler
is the observer that demands stabilization.

---

## 7. Migration Path: Gas Town → Gas City

### 7.1 Guiding Principle: Additive, Not Breaking

Every Gas City feature is implemented as an addition to Gas Town's existing
schema and CLI. No existing commands change behavior. No existing fields
change semantics. A Gas Town rig that doesn't opt into Gas City features
operates exactly as before.

### 7.2 Phase 1: Learning Layer (Week 1-2)

**Changes**:
- Add `reflection` (JSON) and `difficulty` (VARCHAR) columns to `beads` table
- Add `reflect` step to `mol-polecat-work` formula
- Add reflection retrieval query to `gt prime`

**Migration**: Zero. New columns have NULL defaults. Existing beads are
unaffected. The `reflect` step is added to formulas; existing molecules
in flight continue with their original formula.

**Rollback**: Drop the columns, remove the formula step. No data loss
(reflections in existing beads are just ignored).

**Verification**:
- [ ] Polecat completes a task with the new formula
- [ ] Reflection is persisted to bead
- [ ] Next polecat on a similar task receives the reflection in `gt prime`
- [ ] Trivial tasks (< 2 commits) skip reflection

### 7.3 Phase 2: Quality Gate (Week 3-4)

**Changes**:
- Add Critic review step to Refinery pipeline
- New `reviews` table in Dolt for review audit trail
- New `bd review show <mr-id>` command

**Migration**: The Refinery pipeline is modified, but MR processing is
backwards-compatible. MRs submitted before the Critic was enabled skip
the review step.

**Rollback**: Remove the Critic step from the Refinery pipeline. Historical
reviews remain in the `reviews` table for reference.

**Verification**:
- [ ] MR triggers Critic review
- [ ] PASS verdict → gates proceed normally
- [ ] CONCERNS verdict → logged, gates proceed (advisory mode)
- [ ] Review bead is created with structured findings
- [ ] After 50 MRs: evaluate false positive rate and calibrate thresholds

### 7.4 Phase 3: Reactive Foundation (Week 5-8)

**Changes**:
- Add `dirty`, `cell_value`, `cell_evaluator`, `cutoff_mode` columns to `beads`
- Add `reactive` column to `dependencies` table
- New `bd mark-stale`, `bd stabilize`, `bd cell` commands
- Dirty-marking propagation in `bd` write path

**Migration**: New columns have safe defaults (`dirty=FALSE`,
`cell_value=NULL`, `cell_evaluator=NULL`, `cutoff_mode='exact'`). Existing
beads are non-reactive until explicitly configured.

**Rollback**: Drop columns and commands. Reactive behavior stops, beads
revert to static state. No data loss.

**Verification**:
- [ ] `bd mark-stale` propagates dirty flags through reactive deps
- [ ] `bd stabilize` triggers lazy recomputation
- [ ] Cutoff prevents unnecessary downstream propagation
- [ ] Non-reactive beads are completely unaffected

### 7.5 Phase 4: DAG Composition (Week 9-14)

**Changes**:
- Formula step schema extended with `inputs`/`outputs`
- Topological scheduler in molecule dispatch
- Parallel step execution support

**Migration**: Existing linear formulas work unchanged (no inputs/outputs
declared = linear sequential execution). DAG features are opt-in per
formula.

**Rollback**: Remove DAG scheduling. Molecules revert to linear execution.
Formulas with input/output declarations execute linearly (declarations
are ignored).

**Verification**:
- [ ] Linear formulas execute unchanged
- [ ] DAG formula with parallel steps dispatches correctly
- [ ] Cutoff skips steps whose inputs haven't changed
- [ ] Bounded cycles converge or terminate at max_iterations

### 7.6 Phase 5: Skill Crystals (Week 15+, Trial)

**Changes**:
- New `crystals` table
- `bd crystal extract` and `bd crystal match` commands
- Crystal matching in `gt prime`

**Migration**: Purely additive. No impact on existing systems.

**Rollback**: Drop table and commands. No other systems affected.

---

## 8. Risk Register

### 8.1 Complexity Creep (CRITICAL)

**Risk**: Gas City's layers accumulate complexity that outweighs their
benefits. The reactive dataflow substrate is the most dangerous — it adds
a new computation model on top of Gas Town's simple task queue.

**Mitigation**:
- Each phase is independently deployable and rollbackable
- Hard rule: if a phase doesn't demonstrate measurable improvement within
  4 weeks of deployment, roll it back
- Complexity budget: Gas City adds at most 5 new `bd` commands and 10 new
  schema columns per phase. Exceeding this triggers a design review.

**Monitoring**: Track time-to-completion for polecats before and after each
phase. If mean completion time increases, the phase is net negative.

### 8.2 Reactive Over-Engineering (HIGH)

**Risk**: Building a full reactive computation engine when Gas Town has 3-5
polecats and small DAGs. The infrastructure cost outweighs the benefit at
current scale.

**Mitigation**:
- Phase 3 (reactive cells) is deliberately sequenced AFTER Phases 1-2
  (learning + quality), which provide immediate value at any scale
- Reactive features are opt-in per bead — don't reactify everything
- Start with manual `bd mark-stale` before automatic propagation
- Define a scale threshold: reactive cells are only justified when the
  average molecule has > 5 steps with shared dependencies

### 8.3 Reflection Noise (MEDIUM)

**Risk**: Every polecat writes reflections on every task, creating a
reflection haystack that obscures useful patterns.

**Mitigation**:
- Reflection is skipped for trivial tasks (< 2 commits, no blockers)
- Only `challenging` and `novel` difficulty reflections are retrieved
- Consolidation pass synthesizes raw reflections into meta-reflections
- Retrieval is limited to 5 most relevant reflections per prime

### 8.4 Critic False Positives (MEDIUM)

**Risk**: The Critic blocks valid MRs, becoming a bottleneck worse than
human review.

**Mitigation**:
- Start in advisory mode (50 MR calibration period)
- High confidence threshold for BLOCK (≥ 0.8)
- Refinery override: the Refinery can override a BLOCK if its own
  analysis disagrees
- Track false positive rate. If > 10%, lower sensitivity or revert
  to advisory mode.

### 8.5 Crystal Rot (LOW)

**Risk**: Skill crystals extracted from old codebases or deprecated
patterns become misleading.

**Mitigation**:
- 90-day staleness marking for unused crystals
- 180-day archive policy
- Crystals linked to provenance beads — if the source bead's code is
  reverted or superseded, the crystal is automatically marked stale

### 8.6 DAG Debugging Complexity (MEDIUM)

**Risk**: Computation DAGs are harder to debug than linear formulas. When
a polecat's work fails, understanding which DAG step failed and why
requires tracing dependency paths.

**Mitigation**:
- `bd mol status` shows DAG visualization with step states
- Each step's output is stored as a cell value (inspectable)
- Linear fallback: any DAG can be executed linearly by ignoring
  parallelism hints (for debugging)
- Start with simple fan-out DAGs only; add cycles later

---

## 9. What Gas City Does NOT Add

Equally important as what's built is what's deliberately excluded:

### 9.1 No Persistent Agents

Polecats remain ephemeral. Gas City does NOT introduce long-running agent
processes. The learning pipeline (reflections, crystals) provides
continuity without persistence. The state lives in Dolt, not in agents.

**Rationale**: Ephemeral agents avoid memory corruption, identity drift,
and resource leaks ([S1](s1-gap-analysis.md) §3.5). Better models don't fix these problems —
they're architectural.

### 9.2 No Vector Store

Gas City does NOT add embedding-based semantic search. All retrieval is
SQL-based (keyword matching, metadata queries). Dolt is a SQL database;
shoehorning vector similarity into it would be an architectural mismatch.

**Rationale**: The reflection and crystal corpus is small enough for SQL
queries. If it grows to need semantic search, that's a signal to curate
better, not to add infrastructure.

### 9.3 No Agent Market (Deferred)

Central dispatch (Mayor assigns work) continues unchanged. Self-selection
and market-based coordination are deferred per [S2](s2-abstraction-map.md) analysis — not justified
at current fleet scale (<10 polecats).

### 9.4 No MCP/A2A Integration (Deferred)

Protocol membrane is designed but not built. The ecosystem is still
maturing. Gas Town's custom protocols work well internally.

### 9.5 No Background Recomputation

Reactive cells are lazily evaluated — only when demanded. There is NO
background process that eagerly recomputes stale cells. This prevents
runaway LLM costs and keeps the cost model transparent.

---

## 10. Success Criteria

Gas City succeeds if it demonstrates measurable improvement on three metrics:

### 10.1 Polecat Effectiveness

**Metric**: First-attempt success rate (MRs that pass gates without
bisection-driven rejection).

**Baseline**: Measure current rate over 50 MRs before Phase 1 deployment.

**Target**: 15% improvement within 8 weeks of Phase 1 (learning pipeline)
deployment. Reflections from past completions should reduce repeated
mistakes.

### 10.2 Merge Queue Throughput

**Metric**: Time from MR submission to merge (or rejection).

**Baseline**: Measure current merge latency.

**Target**: 20% reduction within 4 weeks of Phase 2 (Critic Lens)
deployment. Pre-gate review catches issues earlier, reducing gate
re-runs.

### 10.3 Cascade Efficiency (Phase 3+)

**Metric**: Ratio of cells marked dirty to cells actually recomputed
(cutoff effectiveness).

**Target**: > 50% cutoff rate — more than half of dirty markings are
stopped by cutoff, preventing unnecessary LLM calls.

---

## 11. Decision Log

| Decision | Rationale | Alternative considered |
|----------|-----------|----------------------|
| SQL retrieval, not vector search | Dolt is SQL; corpus is small | RAG pipeline with embeddings |
| Ephemeral polecats, not persistent | Avoids state corruption | Long-running agents with memory |
| Lazy evaluation, not eager | Inverted cost model | Background recomputation |
| Exact cutoff default | Free and correct for most cases | Semantic cutoff default |
| Advisory Critic first | Calibration prevents false positive bottleneck | Blocking from day 1 |
| Bounded cycle unrolling | Simple, predictable | True cycle support in DAG |
| Phase 1 = Learning (not Reactive) | Highest feasibility-to-impact ratio | Start with reactive substrate |
| Agent Market deferred | Not justified at fleet size < 10 | Build market from day 1 |
| Protocol Membrane deferred | Ecosystem still maturing | Early MCP/A2A adoption |

---

## Sources

- [S1](s1-gap-analysis.md): Gap Analysis — Gas Town vs Frontier (gt-026)
- [S2](s2-abstraction-map.md): Abstraction Map — Candidate City-Level Abstractions (gt-w6q)
- [R1](r1-orchestration-frontier.md): Orchestration Frontier Survey (gt-eth)
- [R2](r2-reactive-dataflow.md): Reactive Dataflow and Incremental Computation (gt-m9z)
- [R3](r3-agent-memory.md): Agent Memory and Identity (gt-0g1)
- [R4](r4-tool-ecosystems.md): Tool Ecosystems and MCP Evolution (gt-xm8)
- [R5](r5-production-deployments.md): Production Agent Deployments (gt-36n)
- [R6](r6-emergent-computation.md): Emergent Computation and Self-Organization (gt-djy)
