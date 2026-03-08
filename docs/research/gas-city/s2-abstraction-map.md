# S2: Abstraction Map — Candidate City-Level Abstractions

**Bead**: gt-w6q | **Date**: 2026-03-08 | **Author**: polecat/cheedo
**Sources**: R1-R6 (Phase 1 Research), [S1](s1-gap-analysis.md) (Gap Analysis)

---

## Executive Summary

Gas Town operates on five core primitives: **beads** (state), **formulas**
(workflow), **polecats** (agents), **mail** (messaging), and **Dolt**
(persistence). These compose well for linear task execution, but they
cannot express reactive dependencies, adaptive coordination, or
compositional computation.

This document identifies **seven candidate abstractions** that could
elevate Gas Town to Gas City. Each is evaluated on four axes: **impact**
(what it unlocks), **durability** (will better models make it obsolete?),
**feasibility** (how hard to build on current primitives), and **risk**
(what can go wrong). The abstractions are grouped into three tiers.

The central thesis: Gas City is not a rewrite of Gas Town. It is Gas
Town's existing primitives **composed through a reactive dataflow
substrate** — where beads become cells, formulas become DAGs, and
staleness propagates through dependency edges rather than being manually
tracked.

---

## 1. The Abstraction Landscape

### 1.1 What Gas Town Has (Primitives)

| Primitive | Role | Limitation |
|-----------|------|------------|
| **Bead** | Unit of state (issue, mail, identity) | No dependency-driven invalidation |
| **Formula** | Linear workflow checklist | Cannot branch, cycle, or react to upstream changes |
| **Polecat** | Ephemeral worker agent | No cross-session learning, no self-selection |
| **Mail** | Agent-to-agent messaging | High-latency (Dolt commit per message), no pub/sub |
| **Dolt** | Version-controlled persistence | SQL-queryable but no reactive triggers |
| **Molecule** | Formula instance attached to a bead | Linear step sequence, no DAG structure |
| **Witness** | Health monitor + self-repair | Reactive to failure, not proactive optimization |
| **Refinery** | Merge queue + gate runner | No pre-merge semantic review |

### 1.2 What Gas City Needs (Capabilities)

The [S1](s1-gap-analysis.md) gap analysis identified five capability gaps. The abstractions
below are the **mechanisms** that close those gaps:

```
Gap                          → Abstraction(s)
─────────────────────────────────────────────────────────
Reactive computation         → Reactive Cells, Computation DAGs
Agent memory/reflection      → Reflection Cycles, Skill Crystals
Adversarial review           → Critic Lens
Adaptive coordination        → Agent Market
External interoperability    → Protocol Membrane
```

---

## 2. Candidate Abstractions

### 2.1 Reactive Cells

**What it is**: A bead that knows its dependencies and can be
marked stale when an upstream cell changes. The fundamental unit of
Gas City's reactive dataflow layer.

**Mechanism**: Every reactive cell has:
- A **value** (the current computed result)
- A **dependency set** (upstream cells it reads from)
- A **staleness flag** (dirty/clean)
- An **evaluator** (function/prompt that recomputes the value)
- A **cutoff predicate** (determines if a new value is "same enough"
  to stop downstream propagation)

When a source cell changes:
1. **Eager dirty marking** propagates downstream through dependency
   edges (cheap — no LLM calls, just flag-setting)
2. **Lazy recomputation** occurs only when an observer demands a
   stale cell's value
3. **Backdating/cutoff** compares the new value to the old; if
   semantically equivalent, downstream cells stay clean

**Grounding in research**:
- [Adapton](https://github.com/Adapton/adapton.rust) ([R2](r2-reactive-dataflow.md)): demand-driven incremental computation
- [Salsa](https://github.com/salsa-rs/salsa) ([R2](r2-reactive-dataflow.md)): red-green backdating for rust-analyzer
- Incremental/Jane Street ([R2](r2-reactive-dataflow.md)): observer-scoped stabilization

**Mapping to Gas Town primitives**:

| Reactive Cell concept | Gas Town primitive | Extension needed |
|-----------------------|-------------------|------------------|
| Cell value | Bead fields (notes, design, status) | Add `dirty` flag, `deps` edge list |
| Dependency edge | `bd dep add` | Already exists — repurpose for reactive edges |
| Evaluator | Formula step body | Generalize to callable unit (prompt template + agent) |
| Observer | Whoever runs `bd show` or `gt prime` | Track active observers to scope computation |
| Cutoff | (none) | Structural/semantic diff on cell outputs |

**The cost structure inversion** ([R2](r2-reactive-dataflow.md) §4): Traditional reactive systems
optimize for cheap evaluation with expensive dependency tracking. Gas
City inverts this — dependency tracking is cheap (small DAGs of beads)
while evaluation is expensive (LLM calls costing seconds and dollars).
This means:
- **Over-marking is cheap**: Mark ten cells dirty for free
- **Over-evaluating is catastrophic**: Each unnecessary LLM call wastes
  time and money
- The system should be **aggressive about marking, conservative about
  evaluating**

**Impact**: HIGH — enables compositional computation where downstream
tasks automatically know when to re-execute.

**Durability**: HIGH — the cost structure inversion is fundamental.
Better models make evaluation faster but not free. Reactive dependency
management will remain valuable.

**Feasibility**: MEDIUM — beads already have dependency edges. The
dirty-marking pass is straightforward. The hard parts are: (a) defining
cutoff predicates for non-deterministic LLM outputs, and (b) dynamic
dependency discovery (a cell doesn't know its deps until it runs).

**Risk**: Over-engineering the reactive layer before Gas Town has enough
cells to justify it. Premature reactive infrastructure with three
polecats is YAGNI. The abstraction should emerge from concrete needs,
not be built speculatively.

**Verdict**: **ADOPT** — but incrementally. Start with manual dirty-marking
on beads (`bd mark-stale <id>`). Add automatic propagation only when
molecule DAGs exist.

---

### 2.2 Computation DAGs (Reactive Molecules)

**What it is**: Molecules that are directed acyclic graphs instead
of linear checklists. Steps have explicit dependency edges, enabling
parallel execution of independent steps and reactive propagation when
a step's output changes.

**Mechanism**: A computation DAG extends the current molecule:
- Steps declare **inputs** (which upstream step outputs they read)
- Steps declare **outputs** (what they produce for downstream steps)
- The molecule scheduler topologically sorts steps and executes
  independent branches in parallel
- If a step is re-evaluated and its output changes, downstream steps
  are marked stale (reactive cell semantics)
- If a step's output is unchanged (cutoff), downstream steps are
  skipped

**Example**: The current `mol-polecat-work` is linear:

```
step1 → step2 → step3 → step4 → step5 → step6 → step7 → step8
```

As a DAG, steps 4 (self-review) and 5 (build check) could be
parallelized after step 3 (implement):

```
                    ┌→ step4 (review) ──┐
step1 → step2 → step3 (implement) ──────┤→ step6 (commit) → step7 → step8
                    └→ step5 (build) ───┘
```

**Grounding in research**:
- [LangGraph](https://github.com/langchain-ai/langgraph) ([R1](r1-orchestration-frontier.md)): graph-based workflow with cycles and branching
- [Temporal](https://temporal.io/) ([R1](r1-orchestration-frontier.md)): durable execution with dependency-driven scheduling
- [Noria](https://github.com/mit-pdos/noria) ([R2](r2-reactive-dataflow.md)): partially stateful dataflow with demand-driven
  materialization

**Mapping to Gas Town**:

| DAG concept | Current primitive | Extension needed |
|-------------|-------------------|------------------|
| DAG step | Molecule wisp (formula step) | Add input/output declarations |
| Parallel branch | (none — linear only) | Scheduler that dispatches independent steps |
| Cycle | (none) | Allow back-edges with convergence criteria |
| Partial result | (none) | Intermediate cell values stored in beads |

**Impact**: HIGH — unlocks parallel step execution within molecules,
reactive cascade when upstream steps change, and conditional branching
(skip steps whose inputs haven't changed).

**Durability**: HIGH — DAG composition is a fundamental computational
pattern. Models cannot make dependency structure unnecessary.

**Feasibility**: MEDIUM-HIGH — the formula system already defines step
sequences. Converting to DAGs requires: (a) input/output declarations
on formula steps, (b) a topological scheduler (straightforward), (c)
mechanism to dispatch multiple polecats or steps concurrently.

**Risk**: Complexity explosion. DAGs are harder to debug than linear
lists. A polecat working step 4 might invalidate step 3's output,
creating a cycle that the DAG can't express. Need clear separation
between DAG structure (static) and reactive updates (dynamic).

**Verdict**: **ADOPT** — but design carefully. Start with fan-out
parallelism (independent steps after a shared predecessor). Add
cycles only when reflection loops require them.

---

### 2.3 Reflection Cycles

**What it is**: A structured post-completion phase where an agent
generates natural-language reflections on its work — what worked,
what failed, what it would do differently — persisted to beads for
retrieval by future agents on similar tasks.

**Mechanism**:
1. After `gt done`, before sandbox destruction, the polecat generates
   a reflection (prompted by a formula step or automatic hook)
2. The reflection is persisted as a typed bead field (not free-text
   notes, but structured: `what_worked`, `what_failed`,
   `would_do_differently`, `patterns_discovered`)
3. Future polecats, during `gt prime`, receive relevant reflections
   retrieved by task similarity (keyword match on bead titles/tags,
   not embedding search — Dolt is SQL, not a vector store)
4. Periodically, a consolidation pass synthesizes raw reflections
   into higher-level insights (Stanford Generative Agents pattern)

**Grounding in research**:
- [Reflexion](https://github.com/noahshinn/reflexion) ([R3](r3-agent-memory.md)): verbal self-reflection yields 8%+ improvement
- Stanford Generative Agents ([R3](r3-agent-memory.md)): reflection creates abstraction
  layers over raw experience; ablation shows each component
  (memory, reflection, planning) is necessary
- [MemGPT](https://github.com/letta-ai/letta) ([R3](r3-agent-memory.md)): sleep-time consolidation of short-term to long-term
  memory

**Mapping to Gas Town**:

| Reflection concept | Gas Town primitive | Extension needed |
|--------------------|-------------------|------------------|
| Raw reflection | Bead notes/design fields | Add structured reflection schema |
| Retrieval | `gt prime` context loading | Add similarity-based retrieval from past reflections |
| Consolidation | (none) | Periodic synthesis pass (could be a formula) |
| Identity accumulation | Capability Ledger | Ledger already records completions; add reflection summaries |

**Impact**: MEDIUM-HIGH — directly addresses the "intelligence layer"
gap identified in [S1](s1-gap-analysis.md) §2.2. Storage is solved (Dolt); the gap is
deciding what to store and when to retrieve.

**Durability**: HIGH — structured reflection is not a capability that
improves with model quality. Better models produce better reflections,
but the mechanism of persisting and retrieving them remains necessary.
Even human engineers benefit from post-mortems.

**Feasibility**: HIGH — this is primarily a prompt engineering and
schema design task. No new infrastructure required. Beads already have
structured fields. The retrieval mechanism (SQL queries on bead
metadata) is native to Dolt.

**Risk**: Low risk of over-engineering; high risk of noise. If every
polecat writes reflections on every task, the reflection store becomes
a haystack. Mitigation: only generate reflections for tasks above a
complexity threshold, or where the polecat encountered and overcame
blockers.

**Verdict**: **ADOPT** — highest feasibility-to-impact ratio of any
candidate. Can be implemented as a new formula step in `mol-polecat-work`
without infrastructure changes.

---

### 2.4 Skill Crystals

**What it is**: Reusable, composable patterns extracted from
successful formula completions. A skill crystal is a proven solution
pattern — not just documentation, but an executable template that
can be instantiated in future workflows.

**Mechanism**:
1. After a successful completion, a synthesis step examines the
   commit diff, the reflection, and the original bead description
2. If a reusable pattern is detected (e.g., "added a new MCP tool
   server", "fixed a Dolt connection timeout", "wrote a reactive
   cell implementation"), a skill crystal is created
3. The crystal contains: pattern name, trigger conditions (when to
   suggest it), solution template (steps or code skeleton), and
   provenance (which completion it was extracted from)
4. Future polecats see relevant crystals during `gt prime`, suggested
   based on bead description similarity

**Grounding in research**:
- [Voyager](https://github.com/MineDojo/Voyager) ([R3](r3-agent-memory.md)): skill library where successful Minecraft actions are
  stored as reusable JavaScript programs with descriptions and
  pre/post conditions
- Skills.sh ([R4](r4-tool-ecosystems.md)): 283,000+ packages of agent capabilities, npm-style
- Gas Town's own formulas: already a form of procedural memory, but
  human-authored and static

**Mapping to Gas Town**:

| Skill Crystal concept | Gas Town primitive | Extension needed |
|-----------------------|-------------------|------------------|
| Crystal store | Formulas directory | New `skills/` directory with crystal schema |
| Trigger conditions | (none) | Matching engine during `gt prime` |
| Solution template | Formula step body | Parameterized templates |
| Provenance | Capability Ledger | Link crystal to originating bead |

**Impact**: MEDIUM — accelerates routine tasks by providing proven
patterns. Reduces the "cold start" problem for polecats encountering
familiar task types.

**Durability**: MEDIUM — as models improve, they may need fewer
explicit skill templates (they'll "just know" how to do things).
But domain-specific patterns (this codebase's conventions, this
team's workflows) remain valuable regardless of model capability.

**Feasibility**: MEDIUM — the extraction step requires judgment about
what constitutes a reusable pattern (not every completion is a skill).
The matching engine is straightforward (SQL on bead metadata). The
main challenge is quality control — preventing crystal proliferation
and staleness.

**Risk**: Crystal rot. Skills extracted from old codebases or
deprecated patterns become misleading. Requires a garbage collection
mechanism or expiration policy.

**Verdict**: **TRIAL** — implement extraction for a narrow category
(e.g., "infrastructure patterns" like adding new formula steps or
configuring MCP servers) before generalizing.

---

### 2.5 Agent Market

**What it is**: A coordination mechanism where polecats self-select
work based on capability, current context, and load — rather than
being centrally dispatched by the Mayor.

**Mechanism**:
1. Available beads are listed on a **market board** (a Dolt table)
   with metadata: complexity estimate, required skills, priority
2. Polecats **bid** on beads they can handle, citing relevant
   capability ledger entries and current context
3. A **clearing function** (could be the Mayor, could be algorithmic)
   matches bids to beads based on: capability fit, load balancing,
   priority, and estimated completion time
4. Unmatched beads escalate to central dispatch as a fallback

**Grounding in research**:
- Market-based coordination ([R6](r6-emergent-computation.md) §3): up to 10% accuracy gains from
  market-making in multi-agent LLM systems
- Scaling science ([R6](r6-emergent-computation.md) §6): centralized coordination yields +80.8%
  on parallelizable tasks; the market should route sequential tasks
  to single agents
- CRAG ([R6](r6-emergent-computation.md) §3): rules of interaction matter more than individual
  agent intelligence

**Mapping to Gas Town**:

| Market concept | Gas Town primitive | Extension needed |
|----------------|-------------------|------------------|
| Market board | `bd ready` output | Add bidding interface |
| Bid | (none) | New bead type or field |
| Capability signal | Capability Ledger | Query ledger for relevant completions |
| Clearing function | Mayor dispatch | Market-clearing algorithm in Mayor |
| Fallback | Central dispatch | Already exists |

**Impact**: MEDIUM — improves work allocation as the fleet scales.
With 3-5 polecats, central dispatch is fine. With 20+, market-based
self-selection reduces Mayor bottleneck and improves task-capability
matching.

**Durability**: LOW-MEDIUM — as models become more capable, the
capability differentiation between polecats shrinks. If any polecat
can handle any task, market-based selection reduces to load balancing,
which simpler mechanisms (round-robin, least-loaded) handle well.

**Feasibility**: MEDIUM — the Dolt infrastructure exists. Bidding is
a schema addition. The hard part is the clearing function: how do you
compare bids from agents who are incentivized to overclaim capability?

**Risk**: Adverse selection. Polecats might bid on easy work and
avoid hard work. Or bid on work they can't actually do, wasting time
before escalating. The Mayor's central dispatch avoids this by making
authoritative assignments.

**Second-order risk**: Complexity. The market adds a coordination
layer that must be debugged, monitored, and maintained. Gas Town's
current simplicity is an asset ([S1](s1-gap-analysis.md) §4.1 — simpler systems win).

**Verdict**: **DEFER** — not justified at current fleet scale. The
Mayor's central dispatch is adequate. Revisit when fleet size exceeds
10-15 polecats and dispatch latency becomes a bottleneck.

---

### 2.6 Critic Lens

**What it is**: An adversarial review phase in the Refinery pipeline
where a dedicated model examines diffs for bugs, security issues,
logic errors, and style violations before gates run.

**Mechanism**:
1. When an MR enters the merge queue, the Refinery extracts the
   diff and relevant context (bead description, test results)
2. A **Critic agent** (could be a polecat, could be a lightweight
   model) reviews the diff with adversarial prompting:
   - "Find bugs in this diff"
   - "What security vulnerabilities does this introduce?"
   - "Does this match the stated requirements in the bead?"
3. Critic output is either: **PASS** (merge proceeds), **CONCERNS**
   (attached to the MR for the Refinery to evaluate), or **BLOCK**
   (MR is quarantined for human review)
4. Low-confidence CONCERNS are logged but don't block; high-confidence
   findings block the merge

**Grounding in research**:
- [Devin](https://devin.ai/) ([R5](r5-production-deployments.md)): dedicated Critic model catches security and logic
  errors before execution
- Factory ([R5](r5-production-deployments.md)): Judge agent filters before human review
- The universal bottleneck ([R5](r5-production-deployments.md) §2.1): PR review time up 91% with
  AI adoption, 67.3% AI-generated PR rejection rate

**Mapping to Gas Town**:

| Critic concept | Gas Town primitive | Extension needed |
|----------------|-------------------|------------------|
| Critic agent | Refinery helper polecat | Dedicated review role/formula |
| Review input | MR diff + bead context | Already available in Refinery |
| Review output | (none) | Structured review bead attached to MR |
| Confidence scoring | (none) | Scoring rubric in Critic prompt |
| Block/pass decision | Refinery gate results | Add semantic review as a gate |

**Impact**: MEDIUM-HIGH — directly addresses the human review
bottleneck. Catches issues that automated gates (build, test, lint)
miss: logic errors, requirement mismatches, security vulnerabilities.

**Durability**: MEDIUM — as base models improve, the implementing
polecat produces fewer bugs, reducing the Critic's value. But
adversarial review catches systematic blind spots that self-review
misses (you can't proofread your own writing). The mechanism remains
valuable even with perfect agents, because the adversarial framing
surfaces issues that collaborative framing doesn't.

**Feasibility**: HIGH — can be implemented as a Refinery formula step
that runs a review prompt against the diff. No new infrastructure
required. The main design decision is the confidence threshold for
blocking.

**Risk**: False positives. An overly aggressive Critic blocks valid
MRs and becomes a bottleneck itself. Mitigation: start with
CONCERNS-only mode (log findings without blocking) and calibrate
thresholds on historical data.

**Verdict**: **ADOPT** — high feasibility, addresses a known bottleneck.
Start in advisory mode (CONCERNS only), promote to blocking after
calibration.

---

### 2.7 Protocol Membrane

**What it is**: A translation layer that exposes Gas Town's internal
capabilities through standard protocols (MCP, A2A) and consumes
external capabilities through the same protocols.

**Mechanism**:
- **Outbound**: Each polecat's capabilities are described in an A2A
  Agent Card (`.well-known/agent-card.json`). External agents can
  discover and delegate work to Gas Town polecats via A2A task
  lifecycle.
- **Inbound**: External MCP servers are consumed as tool providers.
  External A2A agents can post tasks that become beads.
- **Federation**: Multiple Gas Town rigs coordinate via A2A rather
  than custom mail protocols, enabling heterogeneous multi-rig
  deployment.

**Grounding in research**:
- MCP ([R4](r4-tool-ecosystems.md)): 97M+ monthly SDK downloads, de facto agent-tool standard
- A2A ([R4](r4-tool-ecosystems.md)): agent-to-agent coordination under Linux Foundation (AAIF)
- ANP ([R4](r4-tool-ecosystems.md)): decentralized agent network protocol (early stage)
- AAIF ([R4](r4-tool-ecosystems.md)): convergence of all major AI companies on open standards

**Mapping to Gas Town**:

| Membrane concept | Gas Town primitive | Extension needed |
|------------------|-------------------|------------------|
| Agent Card | Polecat role definition | Generate Agent Card from role config |
| Task intake | `bd create` | A2A task → bead translation |
| Tool consumption | MCP server config | Dynamic MCP server discovery |
| Federation | Mail between rigs | A2A-based inter-rig communication |
| Auth | (none) | OAuth 2.1 for external callers |

**Impact**: MEDIUM — enables ecosystem participation. Gas Town can
consume community tools and be consumed by external orchestrators.
Value increases as the AAIF ecosystem matures.

**Durability**: HIGH — protocol adoption is a one-way door. Once
MCP/A2A are standard, non-participation means isolation.

**Feasibility**: LOW-MEDIUM — significant implementation effort.
OAuth 2.1, Agent Card generation, task translation, dynamic MCP
discovery. Each is individually tractable but the aggregate is
substantial.

**Risk**: Premature standardization. If A2A pivots or ANP wins, the
investment is wasted. Mitigation: implement the thinnest possible
membrane (Agent Card + basic task intake) and defer deep integration.

**Verdict**: **DEFER** — the ecosystem is still maturing (A2A under
Linux Foundation since Aug 2025, ANP is pre-production). Build the
membrane when Gas Town needs to federate or consume external
capabilities, not speculatively.

---

## 3. Abstraction Evaluation Matrix

| # | Abstraction | Impact | Durability | Feasibility | Risk | Verdict |
|---|-------------|--------|------------|-------------|------|---------|
| 1 | Reactive Cells | HIGH | HIGH | MEDIUM | Over-engineering | **ADOPT** (incremental) |
| 2 | Computation DAGs | HIGH | HIGH | MEDIUM-HIGH | Complexity | **ADOPT** (careful design) |
| 3 | Reflection Cycles | MEDIUM-HIGH | HIGH | HIGH | Noise | **ADOPT** (immediate) |
| 4 | Skill Crystals | MEDIUM | MEDIUM | MEDIUM | Crystal rot | **TRIAL** |
| 5 | Agent Market | MEDIUM | LOW-MEDIUM | MEDIUM | Adverse selection | **DEFER** |
| 6 | Critic Lens | MEDIUM-HIGH | MEDIUM | HIGH | False positives | **ADOPT** (advisory first) |
| 7 | Protocol Membrane | MEDIUM | HIGH | LOW-MEDIUM | Premature standardization | **DEFER** |

---

## 4. Composition: How Abstractions Interact

The abstractions are not independent. They compose into a coherent
system through three interaction patterns:

### 4.1 The Reactive Core (Cells + DAGs)

Reactive Cells and Computation DAGs form the **computation substrate**
of Gas City. Cells are the atoms; DAGs are the molecules.

```
                     Gas Town              Gas City
                     ────────              ────────
Unit of state:       Bead           →      Reactive Cell (bead + deps + dirty flag)
Workflow:            Linear formula →      Computation DAG (formula + dependency edges)
Scheduling:          Sequential    →      Topological (parallel where possible)
Invalidation:        Manual        →      Automatic (dirty marking + cutoff)
```

Together, they transform Gas Town from a task queue into a reactive
computation engine. A change in one cell's output propagates staleness
through the DAG, triggering lazy recomputation only where needed.

### 4.2 The Learning Loop (Reflections + Crystals)

Reflection Cycles and Skill Crystals form a **learning pipeline**:

```
Completion → Reflection → Consolidation → Crystal Extraction
    ↓              ↓              ↓                ↓
(bead closes)  (what worked?)  (patterns?)     (reusable template)
                                                   ↓
                                           Future gt prime
                                           retrieves crystal
```

Reflections are the raw material; crystals are the refined product.
Without reflections, there's nothing to crystallize. Without crystals,
reflections remain episodic (useful for similar tasks) but don't
generalize into procedural knowledge.

### 4.3 The Quality Gate (Critic + Reactive Cells)

The Critic Lens interacts with Reactive Cells through **confidence-
driven cutoff**:

```
MR arrives → Critic reviews diff → PASS/CONCERNS/BLOCK
                                        ↓
                              If CONCERNS: downstream cells
                              can treat review findings as
                              inputs (reactive dependency)
```

A Critic finding on an MR could mark downstream cells as stale —
e.g., if the Critic identifies a security concern, related beads
in the reactive DAG are flagged for re-evaluation.

### 4.4 Dependency Graph of Abstractions

Some abstractions depend on others:

```
                Protocol Membrane (standalone, defer)

Agent Market (standalone, defer)

Skill Crystals ──── depends on ──── Reflection Cycles
       ↑                                   ↑
       │                                   │
       └─── benefits from ─── Computation DAGs
                                   ↑
                                   │
                            Reactive Cells (foundation)

Critic Lens (standalone, can compose with Reactive Cells)
```

**Build order implication**:
1. Reflection Cycles (no dependencies, high feasibility)
2. Critic Lens (no dependencies, high feasibility)
3. Reactive Cells (foundation for DAGs)
4. Computation DAGs (requires Reactive Cells)
5. Skill Crystals (requires Reflection Cycles)
6. Agent Market (defer)
7. Protocol Membrane (defer)

---

## 5. The Gas City Architecture Sketch

With the ADOPT abstractions in place, Gas City's architecture emerges:

```
┌──────────────────────────────────────────────────────────┐
│                      GAS CITY                            │
│                                                          │
│  ┌──────────────────────────────────────────────────┐    │
│  │              Reactive Dataflow Layer              │    │
│  │                                                   │    │
│  │  ┌─────┐     ┌─────┐     ┌─────┐     ┌─────┐    │    │
│  │  │Cell │────→│Cell │────→│Cell │────→│Cell │    │    │
│  │  │(bead)│     │(bead)│     │(bead)│     │(bead)│    │    │
│  │  └─────┘     └─────┘     └─────┘     └─────┘    │    │
│  │       ↑           │           ↑                   │    │
│  │       │      dirty marking    │                   │    │
│  │       └───────────────────────┘                   │    │
│  │                cutoff                             │    │
│  └──────────────────────────────────────────────────┘    │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐   │
│  │  Reflection  │  │  Critic Lens  │  │  Computation  │   │
│  │   Cycles     │  │  (Refinery)   │  │    DAGs       │   │
│  │              │  │               │  │  (Molecules)  │   │
│  └──────┬───────┘  └───────────────┘  └───────────────┘   │
│         │                                                 │
│         ▼                                                 │
│  ┌─────────────┐                                         │
│  │    Skill     │                                         │
│  │  Crystals    │  (trial)                               │
│  └─────────────┘                                         │
│                                                          │
│  ═══════════════════════════════════════════════════════  │
│                    Gas Town (unchanged)                   │
│  Beads · Formulas · Polecats · Mail · Dolt · Witness     │
│  Refinery · Mayor · Capability Ledger                    │
└──────────────────────────────────────────────────────────┘
```

Key design principle: **Gas City is a layer ON TOP of Gas Town, not a
replacement.** Gas Town's primitives remain the foundation. Gas City
adds reactive composition, learning, and quality mechanisms.

---

## 6. Durability Analysis: What Survives Better Models?

The strongest signal from the research ([R1](r1-orchestration-frontier.md), [R5](r5-production-deployments.md), [R6](r6-emergent-computation.md)) is that simpler
systems outperform complex ones when the base model is capable enough.
Each abstraction must pass the durability test: **"Would a more
capable base model make this unnecessary?"**

| Abstraction | Durability rationale |
|-------------|---------------------|
| **Reactive Cells** | YES — durable. The cost structure inversion (cheap marking, expensive evaluation) is fundamental. Better models reduce evaluation cost but don't eliminate it. Dependency tracking remains valuable. |
| **Computation DAGs** | YES — durable. DAG composition is a mathematical structure, not a model capability. Parallelism and dependency ordering don't become unnecessary with better models. |
| **Reflection Cycles** | YES — durable. Even expert human engineers benefit from post-mortems. The mechanism of structured retrospection is independent of capability level. Better models produce better reflections. |
| **Skill Crystals** | MAYBE — partially durable. Domain-specific patterns (this codebase, this team) remain valuable. General programming patterns may become unnecessary as models internalize them. |
| **Agent Market** | PROBABLY NOT — as models become more capable, capability differentiation between agents shrinks. Market-based selection reduces to load balancing. |
| **Critic Lens** | MAYBE — adversarial framing surfaces blind spots that self-review misses. But better models have fewer blind spots. The mechanism remains valuable for catching systematic errors. |
| **Protocol Membrane** | YES — durable. Protocol adoption is infrastructure, not intelligence. Interoperability doesn't become unnecessary with better models. |

---

## 7. Open Design Questions

Each ADOPT abstraction raises design questions that [S3](s3-architecture-sketch.md) (Architecture
Sketch) should address:

### 7.1 Reactive Cells

1. **Semantic equality**: How do you define "same enough" for LLM
   outputs? Exact string match is too strict (non-deterministic
   outputs differ superficially). Semantic embedding similarity?
   Structural comparison of extracted facts?

2. **Dynamic dependencies**: A cell's dependency set isn't known until
   it runs. How do you track dependencies that are discovered during
   evaluation? (Adapton solves this with dynamic dependency recording.)

3. **Staleness budget**: Not all stale cells need immediate
   recomputation. How do you allocate a "staleness budget" — which
   cells can tolerate being stale longer?

### 7.2 Computation DAGs

4. **Cycle handling**: Reflection loops (implement → review → revise →
   review) are inherently cyclic. DAGs can't express cycles. Options:
   unroll cycles with a bounded iteration count, or use a separate
   "cycle manager" that wraps a sub-DAG.

5. **Multi-agent DAG steps**: Can a single DAG step be executed by
   multiple polecats? Or is each step assigned to exactly one agent?
   Fan-out at the step level vs. fan-out at the molecule level.

6. **DAG versioning**: When a DAG's structure changes (new step added,
   dependency rewired), how do in-flight molecules handle the
   migration? Dolt's branching model could help here.

### 7.3 Reflection Cycles

7. **Reflection threshold**: Which completions warrant reflection?
   Every task? Only tasks above a complexity threshold? Only tasks
   where the polecat hit blockers?

8. **Retrieval ranking**: When multiple reflections match a new task,
   how do you rank them? Recency? Similarity? Author capability
   score from the ledger?

### 7.4 Critic Lens

9. **Confidence calibration**: How do you set the BLOCK vs. CONCERNS
   threshold? Too low = everything blocks. Too high = nothing blocks.
   Calibrate against historical merge queue data (which MRs caused
   bisection failures?).

10. **Critic independence**: The Critic must not be the same model
    instance that wrote the code (confirmation bias). Use a different
    model, different temperature, or different system prompt?

---

## 8. Recommended Build Sequence

Based on the dependency graph (§4.4), feasibility, and impact:

### Phase 1: Learning Layer (immediate — no infrastructure changes)

| Step | Abstraction | What to build |
|------|-------------|---------------|
| 1a | Reflection Cycles | Add `reflect` step to `mol-polecat-work` formula |
| 1b | Reflection Cycles | Add structured reflection fields to bead schema |
| 1c | Reflection Cycles | Add reflection retrieval to `gt prime` |

**Why first**: Highest feasibility-to-impact ratio. Zero infrastructure
changes. Immediately improves polecat performance on recurring task
types. Generates data that Skill Crystals (Phase 2) will consume.

### Phase 2: Quality Layer (near-term — Refinery enhancement)

| Step | Abstraction | What to build |
|------|-------------|---------------|
| 2a | Critic Lens | Add semantic review step to Refinery pipeline |
| 2b | Critic Lens | Start in advisory mode (CONCERNS only) |
| 2c | Critic Lens | Calibrate thresholds, promote to blocking |

**Why second**: Independent of Phase 1. High feasibility. Addresses
the human review bottleneck identified across [R5](r5-production-deployments.md) and [S1](s1-gap-analysis.md).

### Phase 3: Reactive Foundation (mid-term — beads enhancement)

| Step | Abstraction | What to build |
|------|-------------|---------------|
| 3a | Reactive Cells | Add `dirty` flag and `reactive_deps` to bead schema |
| 3b | Reactive Cells | Implement eager dirty-marking propagation |
| 3c | Reactive Cells | Implement lazy recomputation on `bd show` / `gt prime` |
| 3d | Reactive Cells | Design and implement cutoff predicates |

**Why third**: Foundational for Computation DAGs but requires schema
changes and new propagation logic. More complex than Phases 1-2.

### Phase 4: DAG Composition (later — formula system evolution)

| Step | Abstraction | What to build |
|------|-------------|---------------|
| 4a | Computation DAGs | Add input/output declarations to formula steps |
| 4b | Computation DAGs | Implement topological scheduler |
| 4c | Computation DAGs | Enable parallel step execution |
| 4d | Computation DAGs | Connect to reactive cell staleness propagation |

**Why fourth**: Requires Reactive Cells (Phase 3) as foundation.
Most complex abstraction. Highest architectural impact.

### Phase 5: Skill Extraction (trial — dependent on Phase 1 data)

| Step | Abstraction | What to build |
|------|-------------|---------------|
| 5a | Skill Crystals | Define crystal schema and storage |
| 5b | Skill Crystals | Implement extraction from reflections |
| 5c | Skill Crystals | Add crystal matching to `gt prime` |

**Why last among ADOPTs**: Depends on Phase 1 generating enough
reflection data to extract patterns from. Trial status — evaluate
after Phase 1 has been running.

---

## Sources

This synthesis draws from all six Phase 1 research reports and the
[S1](s1-gap-analysis.md) gap analysis:
- [R1](r1-orchestration-frontier.md): Orchestration Frontier Survey (gt-eth)
- [R2](r2-reactive-dataflow.md): Reactive Dataflow and Incremental Computation (gt-m9z)
- [R3](r3-agent-memory.md): Agent Memory and Identity (gt-0g1)
- [R4](r4-tool-ecosystems.md): Tool Ecosystems and MCP Evolution (gt-xm8)
- [R5](r5-production-deployments.md): Production Agent Deployments (gt-36n)
- [R6](r6-emergent-computation.md): Emergent Computation and Self-Organization (gt-djy)
- [S1](s1-gap-analysis.md): Gap Analysis — Gas Town vs Frontier (gt-026)
