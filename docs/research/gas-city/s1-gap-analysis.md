# S1: Gap Analysis — Gas Town vs Frontier

**Bead**: gt-026 | **Date**: 2026-03-08 | **Author**: polecat/splendid
**Sources**: [R1](r1-orchestration-frontier.md) (Orchestration), [R2](r2-reactive-dataflow.md) (Reactive Dataflow), [R3](r3-agent-memory.md) (Agent Memory),
[R4](r4-tool-ecosystems.md) (Tool Ecosystems), [R5](r5-production-deployments.md) (Production Deployments), [R6](r6-emergent-computation.md) (Emergent Computation)

---

## Executive Summary

Gas Town is architecturally well-positioned relative to the multi-agent
frontier. Its core design — role-based hierarchy, stigmergic coordination
through Dolt-backed beads, isolated worktrees, formula-driven workflows, and
a bisecting merge queue — independently arrived at patterns that production
systems ([Cursor](https://cursor.com/), [Devin](https://devin.ai/), Codex) discovered through painful trial and error.

However, Gas Town has significant gaps in five areas: **reactive
computation** (no demand-driven evaluation or cutoff), **agent memory**
(no structured reflection or cross-session learning), **external
interoperability** (no MCP/A2A integration for federation), **automated
quality gates** (no adversarial review before merge), and **adaptive
coordination** (fixed dispatch model, no task-aware routing).

Gas Town's unique contribution is the **convergence of version-controlled
state (Dolt), stigmergic coordination (beads), and formal workflow
templates (formulas)** — a combination not found in any surveyed system.

---

## 1. Where Gas Town Sits: Frontier Alignment

### 1.1 Patterns Gas Town Already Embodies

| Frontier Pattern | Gas Town Implementation | Evidence |
|------------------|------------------------|----------|
| Role-based hierarchy | Mayor/Witness/Polecat/Refinery | [R1](r1-orchestration-frontier.md) §3.2: role-playing failed ([CrewAI](https://github.com/crewAIInc/crewAI)), hierarchy works (Cursor). Gas Town's hierarchy is structural, not role-played. |
| Explicit handoffs | Hook system, `gt done`, mail | [R1](r1-orchestration-frontier.md) §3.4: all frameworks converging on explicit handoffs. Gas Town had this from inception. |
| Durable execution | Dolt-backed beads + git worktrees | [R1](r1-orchestration-frontier.md) §1.7: [Temporal](https://temporal.io/) identified as gold standard. Gas Town achieves equivalent durability through Dolt + git, not workflow engines. |
| Bounded autonomy | Formula checklists, escalation paths | [R5](r5-production-deployments.md) §2.3: dominant 2026 production pattern. Gas Town's formula system IS bounded autonomy. |
| Stigmergic coordination | Beads as shared medium, minimal direct messaging | [R6](r6-emergent-computation.md) §2: O(n) scaling vs O(n^2) direct messaging. Gas Town's "default to nudge" policy is stigmergic by design. |
| Sandbox isolation | Git worktrees per polecat | [R5](r5-production-deployments.md) §1.5: Codex uses worktree isolation. Gas Town matches this pattern. |
| Quality gates before merge | Refinery bisecting merge queue | [R5](r5-production-deployments.md) §4.1: human review is the universal bottleneck. Gas Town's automated gates partially address this. |
| Event-sourced state | Dolt commit history on every bead write | [R5](r5-production-deployments.md) §1.4: [OpenHands](https://github.com/All-Hands-AI/OpenHands) V1 adopted event-sourcing. Gas Town has this via Dolt. |
| Actor model | Private state (worktrees) + message passing (mail) | [R6](r6-emergent-computation.md) §5: Erlang/OTP patterns. Gas Town follows actor semantics naturally. |
| Self-repair | Witness restarts stuck agents | [R6](r6-emergent-computation.md) §4: CA self-repair property. Witness is Gas Town's self-repair mechanism. |

### 1.2 Frontier Maturity Assessment

```
                    Behind    At Parity    Ahead
                      |          |           |
Orchestration         |          ●           |     Role hierarchy matches Cursor/Devin
Durability            |          |           ●     Dolt > vector stores for memory
Coordination          |          ●           |     Stigmergy matches production patterns
Isolation             |          ●           |     Worktrees match Codex model
Quality gates         |     ●    |           |     No adversarial review (Devin has Critic)
Memory                ●          |           |     No reflection, no cross-session learning
Reactivity            ●          |           |     No demand-driven computation
Interoperability      ●          |           |     No MCP/A2A federation
Observability         |     ●    |           |     Beads audit trail but no structured telemetry
Adaptive routing      ●          |           |     Fixed dispatch, no task-aware strategy
```

---

## 2. What Others Have That Gas Town Lacks

### 2.1 Demand-Driven Reactive Computation (HIGH IMPACT)

**Gap**: Gas Town has no reactive computation model. When upstream state
changes, there is no automatic staleness propagation, no demand-driven
recomputation, and no cutoff to prevent unnecessary cascades.

**What the frontier has** ([R2](r2-reactive-dataflow.md)):
- **[Adapton](https://github.com/Adapton/adapton.rust)**: Demand-driven incremental computation — dirty-mark eagerly,
  recompute lazily only when an observer demands the value.
- **[Salsa](https://github.com/salsa-rs/salsa)**: Backdating — if a recomputed value hasn't changed, downstream
  dependents skip recomputation. Durability levels classify inputs by
  volatility.
- **Incremental (Jane Street)**: Observer-scoped computation — only cells
  reachable from active observers are maintained.

**Why it matters**: Gas City's future as a reactive DAG of agent
computations requires these primitives. Without demand-driven cleaning,
every upstream change triggers wasteful LLM re-evaluations. Without
cutoff, cascades propagate unconditionally.

**Recommended hybrid** ([R2](r2-reactive-dataflow.md) §9): Eager dirty marking (Excel/Adapton) +
lazy recomputation on demand (Adapton/[Noria](https://github.com/mit-pdos/noria)) + backdating/cutoff (Salsa)
+ batched stabilization in topological order (Incremental) + observer
scoping (Incremental/Noria).

### 2.2 Structured Agent Memory and Reflection (HIGH IMPACT)

**Gap**: Polecats start from scratch each session. The auto-memory
(`MEMORY.md`) pattern is primitive. No structured reflection on
completions. No skill discovery from successful patterns.

**What the frontier has** ([R3](r3-agent-memory.md)):
- **[MemGPT](https://github.com/letta-ai/letta)/Letta**: Tiered memory hierarchy (core/recall/archival) with
  self-editing memory blocks and sleep-time consolidation.
- **[Voyager](https://github.com/MineDojo/Voyager)**: Skill library where successful action patterns are stored
  as reusable, composable code with natural-language descriptions.
- **Stanford Generative Agents**: Memory stream + reflection + planning.
  Reflection creates abstraction layers over raw experience.
- **[Reflexion](https://github.com/noahshinn/reflexion)**: Verbal self-reflection on failures stored as episodic
  memory, yielding 8%+ improvement on subsequent attempts.

**What Gas Town already has (but doesn't leverage)**:
- Beads ARE episodic memory (event records of work)
- Formulas ARE procedural memory (reusable work patterns)
- The Capability Ledger IS identity-as-track-record
- Dolt provides transactionally consistent, version-controlled storage
  (superior to the vector stores used by MemGPT/Voyager)

**The gap is not storage — it's the intelligence layer**: Gas Town has
the best storage substrate in the surveyed landscape but the weakest
mechanisms for deciding WHAT to store and WHEN to retrieve.

**Recommended additions** (ranked by impact):
1. **Structured reflection after task completion** — polecats generate
   natural-language reflections, persisted to beads, retrievable by
   future polecats on similar tasks.
2. **Skill discovery from completions** — Voyager-style extraction of
   reusable patterns from successful formula completions.
3. **Retrieval-augmented context loading** — `gt prime` retrieves relevant
   episodic memories based on current task similarity, not just fixed context.

### 2.3 External Interoperability (MEDIUM IMPACT)

**Gap**: No Agent Cards, no way for external agents to discover or delegate
to Gas Town. No mechanism to consume community skills. No federation
between Gas Town rigs via standard protocols.

**What the frontier has** ([R4](r4-tool-ecosystems.md)):
- **MCP**: 97M+ monthly SDK downloads, 10,000+ active servers. De facto
  standard for agent-to-tool communication.
- **A2A**: Agent-to-agent coordination with Agent Cards for capability
  discovery, task lifecycle management, and UX negotiation.
- **Skills.sh**: 283,000+ packages. npm-style capability packaging.
- **AAIF**: All major AI companies committed to open, interoperable agent
  standards under Linux Foundation governance.

**Gas Town's position**: Operates at protocol Layers 2-3 with custom
implementations. The internal protocols work well but are proprietary.
External interoperability is the gap.

**Strategic options** ([R4](r4-tool-ecosystems.md) §11):
- Expose polecats as A2A Agent Cards (external delegation)
- Consume skills.sh packages as formula steps
- Federate Gas Town rigs via A2A
- Publish formulas as skills.sh packages

### 2.4 Adversarial Quality Review (MEDIUM IMPACT)

**Gap**: Refinery runs automated gates (build, test, lint) but does not
perform adversarial code review. No equivalent to Devin's Critic model.

**What the frontier has** ([R5](r5-production-deployments.md)):
- **Devin**: Dedicated Critic model reviews for security vulnerabilities
  and logic errors before execution.
- **Cursor**: Automated testing gates before human review.
- **Factory**: Judge agent filters before human review.

**Why it matters**: Human review is the universal bottleneck ([R5](r5-production-deployments.md) §2.1).
PR review time increases 91% with high AI adoption. An adversarial
review agent would catch issues before they reach the merge queue,
reducing human review burden.

**Recommended addition**: A Critic/Judge role in the Refinery pipeline
that reviews diffs for bugs, security issues, and style violations
before running gates.

### 2.5 Adaptive Coordination Strategy (MEDIUM IMPACT)

**Gap**: Gas Town uses fixed central dispatch (Mayor assigns beads to
polecats). No task-aware routing. No self-selection. No distinction
between parallelizable and sequential work.

**What the frontier has** ([R6](r6-emergent-computation.md)):
- **Scaling science** (Google, Dec 2025): Centralized coordination yields
  +80.8% on parallelizable tasks but degrades sequential reasoning by
  39-70%. Coordination strategy must match task structure.
- **Market mechanisms**: Agents self-select work based on capability,
  load, and comparative advantage. Scales as O(n).
- **Dynamic orchestration**: Coordination structure adapts to task type
  at runtime.

**Key finding**: Multi-agent is not universally better. Single-agent
outperforms multi-agent for sequential reasoning tasks. The optimal
strategy depends on task parallelizability, baseline agent capability,
and error tolerance.

**Recommended additions**:
- Task-aware routing: sequential reasoning tasks → single agent;
  parallelizable work → multi-polecat dispatch.
- Self-selection option: polecats can bid on beads based on current
  context and demonstrated capability.

---

## 3. What Gas Town Has That Nobody Else Does

### 3.1 Version-Controlled Transactional State (Dolt)

No surveyed system uses a version-controlled database as its coordination
substrate. The typical stack is:
- Vector stores (MemGPT, Voyager): lossy, eventually consistent, no
  transactional guarantees
- Flat files (Stanford agents): no conflict detection, no audit trail
- Custom event stores (OpenHands): event-sourced but not SQL-queryable

Dolt provides:
- **Atomic memory operations** — no partial writes
- **Full audit trail** — every state change is a versioned commit
- **Branch-based isolation** — polecats can't corrupt shared state
- **SQL-queryable** — structured retrieval without embedding search
- **Diffable** — any two states can be compared

This is architecturally superior to anything in the surveyed landscape
for persistent agent state management.

### 3.2 Formula-Driven Bounded Autonomy

Gas Town's formula system (mol-polecat-work) is a unique combination of:
- **Procedural memory** — reusable workflow templates
- **Bounded autonomy** — agents follow structured checklists
- **Lifecycle integration** — formula steps map to bead states

No surveyed framework has this tight integration between workflow
templates, issue tracking, and agent autonomy. The closest analogs are:
- Voyager's skill library (procedural memory, but no lifecycle integration)
- OpenAI Agents SDK's guardrails (bounds autonomy, but no workflow templates)
- Temporal's workflow definitions (lifecycle integration, but no agent autonomy)

Gas Town's formulas combine all three.

### 3.3 Stigmergic Coordination with Version-Controlled Medium

The beads system is pure stigmergy ([R6](r6-emergent-computation.md) §2): polecats don't message each
other — they read/write beads, commit to git, and the Refinery processes
environmental state. The "default to nudge" policy minimizes direct
communication overhead.

What makes this unique is the combination with Dolt: the stigmergic
medium is itself version-controlled, transactional, and SQL-queryable.
Traditional stigmergic systems (ant colonies, wiki-based coordination)
lack these properties.

### 3.4 The Bisecting Merge Queue

The Refinery's bisecting merge queue is more sophisticated than any
surveyed CI/merge system:
- Batches MRs and runs gates on the merged stack
- If gates fail, bisects to isolate the culprit
- Merges the good MRs, quarantines the bad one
- Fixes failures inline or via helper polecats

This provides production-grade quality guarantees without requiring
human intervention in the merge process. No surveyed system has an
equivalent automated bisection-and-fix pipeline.

### 3.5 Self-Cleaning Agent Lifecycle

Polecats are **ephemeral by design** — they execute and self-destruct
(`gt done` nukes the sandbox). This is a deliberate architectural choice
that avoids the state accumulation problems that plague persistent
agent systems:
- No memory corruption over time
- No identity drift
- No resource leaks from long-running agents
- Clean handoff semantics

The Capability Ledger (identity-as-track-record) provides continuity
without persistent state. This approach is not found in any surveyed
system — most assume agents should be persistent.

---

## 4. Cross-Report Synthesis: Recurring Themes

### 4.1 Simplicity vs. Sophistication

A consistent finding across [R1](r1-orchestration-frontier.md), [R5](r5-production-deployments.md), and [R6](r6-emergent-computation.md): **simpler systems outperform
more complex ones** when the base model is capable enough.

- Claude Code ($1B+ ARR) uses a single-threaded agentic loop ([R5](r5-production-deployments.md))
- Anthropic advocates "patterns over frameworks" ([R1](r1-orchestration-frontier.md))
- Heavier frameworks (CrewAI, [AutoGen](https://github.com/microsoft/autogen)) have higher overhead and lower
  production adoption ([R1](r1-orchestration-frontier.md))
- Multi-agent coordination degrades sequential reasoning by 39-70% ([R6](r6-emergent-computation.md))

**Implication for Gas Town**: Resist the temptation to add complexity.
Each gap identified above should be evaluated against the question: "Does
the base model's capability make this unnecessary?" As models improve,
the sweet spot for multi-agent decomposition shrinks ([R1](r1-orchestration-frontier.md) §5.5).

### 4.2 The Cost Structure Inversion

[R2](r2-reactive-dataflow.md) identifies a fundamental insight: Gas Town's cost structure is the
inverse of traditional reactive systems.

| | Traditional | Gas Town |
|---|---|---|
| Cell evaluation | Cheap (ns-ms) | Expensive (seconds, dollars) |
| Dependency tracking | Expensive at scale | Cheap (small DAGs) |
| Cutoff benefit | Saves CPU cycles | Saves dollars |
| Recomputation | Acceptable to over-compute | Must minimize |

This inversion means Gas Town should not copy reactive system designs
wholesale. It should use **eager marking** (cheap) with **lazy
evaluation** (expensive) — the opposite of systems like [Observable](https://observablehq.com/)
that auto-rerun eagerly.

### 4.3 The Error Amplification Constraint

Multi-agent error amplification is a hard constraint, not just a concern:
- 17x error rate in naive "bag of agents" ([R1](r1-orchestration-frontier.md) §3.3)
- Independent agents amplify errors 17.2x; centralized coordination
  contains to 4.4x ([R6](r6-emergent-computation.md) §6)
- 67.3% rejection rate for AI-generated PRs ([R5](r5-production-deployments.md) §2.1)

Gas Town's architecture already mitigates this through:
- The Refinery's bisecting merge queue (catches integration errors)
- Formula checklists (prevent process errors)
- Witness monitoring (catches stuck agents)

But Gas Town lacks **pre-merge adversarial review** (Devin's Critic) and
**structured self-reflection** (Reflexion) — two mechanisms that would
reduce the error rate before work reaches the merge queue.

### 4.4 The Protocol Convergence Window

[R4](r4-tool-ecosystems.md) identifies a convergence window: MCP, A2A, and skills.sh are
standardizing under AAIF governance. The protocol layer is stabilizing.

Gas Town's custom protocols (beads, mail, formulas) work well internally
but are not interoperable. The question is **when** to invest in
external interoperability, not **whether**.

Current assessment: not urgent. Gas Town's internal protocols work. But
as the ecosystem matures, lack of MCP/A2A integration will limit Gas
Town's ability to federate with other systems or consume community
capabilities.

---

## 5. Prioritized Gap Remediation

### Tier 1: High Impact, Achievable Now

| # | Gap | What to Build | Source |
|---|-----|---------------|--------|
| 1 | No structured reflection | Post-completion reflection generation, persisted to beads | [R3](r3-agent-memory.md): Reflexion, Stanford agents |
| 2 | No demand-driven computation | Eager dirty marking + lazy recomputation in Gas City DAG | [R2](r2-reactive-dataflow.md): Adapton, Incremental |
| 3 | No adversarial review | Critic/Judge role in Refinery pipeline | [R5](r5-production-deployments.md): Devin, Factory |

### Tier 2: Medium Impact, Requires Design

| # | Gap | What to Build | Source |
|---|-----|---------------|--------|
| 4 | No cutoff/backdating | Structural comparison of cell outputs to stop cascades | [R2](r2-reactive-dataflow.md): Salsa |
| 5 | No skill discovery | Extract reusable patterns from successful completions | [R3](r3-agent-memory.md): Voyager |
| 6 | No task-aware routing | Route sequential tasks to single agent, parallel to swarm | [R6](r6-emergent-computation.md): Scaling science |
| 7 | No retrieval-augmented priming | `gt prime` loads relevant episodic memories for current task | [R3](r3-agent-memory.md): MemGPT |

### Tier 3: Strategic, Requires Ecosystem Investment

| # | Gap | What to Build | Source |
|---|-----|---------------|--------|
| 8 | No external interoperability | A2A Agent Cards for polecats, MCP for tool federation | [R4](r4-tool-ecosystems.md): AAIF stack |
| 9 | No skills consumption | Integrate skills.sh packages as formula steps | [R4](r4-tool-ecosystems.md): Skills ecosystem |
| 10 | No market-based self-selection | Polecats bid on beads based on capability and load | [R6](r6-emergent-computation.md): Market mechanisms |

---

## 6. Strategic Positioning

### 6.1 Gas Town's Moat

Gas Town's defensible advantages are:
1. **Dolt as coordination substrate** — no one else has version-controlled
   transactional state for agent coordination
2. **Formula-driven bounded autonomy** — tight integration of workflow
   templates, issue tracking, and agent lifecycle
3. **Stigmergic architecture** — O(n) coordination scaling without
   direct agent-to-agent communication
4. **Ephemeral agents with persistent records** — clean lifecycle that
   avoids state corruption

These advantages compound: Dolt enables stigmergy, stigmergy enables
ephemeral agents, ephemeral agents enable clean formulas, formulas
enable bounded autonomy.

### 6.2 Risk: Over-Engineering

The strongest signal from the research is that **simpler systems win**.
Claude Code's single-threaded loop outearns all multi-agent frameworks
combined. Multi-agent coordination degrades performance on reasoning
tasks. The framework wars are consolidating toward minimal primitives.

Gas Town should add capabilities surgically. Each addition should be
tested against: "Would a more capable base model make this unnecessary?"
The reactive computation layer (Gap #2) and structured reflection
(Gap #1) are likely durable — they solve problems that better models
won't eliminate. External interoperability (Gap #8) is infrastructure
that grows more valuable over time. Market-based self-selection (Gap #10)
may become unnecessary as models improve at self-coordination.

### 6.3 The Gas City Transition

Gas City (the next evolution) should be designed as a **demand-driven,
batch-stabilized reactive dataflow engine** ([R2](r2-reactive-dataflow.md) §9) where:
- Molecules are reactive DAGs, not linear checklists
- Cells are lazily evaluated on demand, not eagerly dispatched
- Cutoff prevents unnecessary cascade recomputation
- Observers drive scope (only compute what someone needs)
- The cost model is inverted: cheap marking, expensive evaluation

This positions Gas City at the intersection of reactive computation
([R2](r2-reactive-dataflow.md)), agent orchestration ([R1](r1-orchestration-frontier.md)), and production patterns ([R5](r5-production-deployments.md)) — a
combination no existing system has achieved.

---

## Sources

This synthesis draws from all six Phase 1 research reports:
- [R1](r1-orchestration-frontier.md): Orchestration Frontier Survey (gt-eth)
- [R2](r2-reactive-dataflow.md): Reactive Dataflow and Incremental Computation (gt-m9z)
- [R3](r3-agent-memory.md): Agent Memory and Identity (gt-0g1)
- [R4](r4-tool-ecosystems.md): Tool Ecosystems and MCP Evolution (gt-xm8)
- [R5](r5-production-deployments.md): Production Agent Deployments (gt-36n)
- [R6](r6-emergent-computation.md): Emergent Computation and Self-Organization (gt-djy)
