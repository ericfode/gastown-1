# R3: Agent Memory and Identity — Frontier Research Survey

**Bead**: gt-0g1
**Date**: 2026-03-08
**Scope**: Frontier agent memory and identity systems — [MemGPT/Letta](#31-memgpt--letta--the-memory-operating-system), [Voyager](#32-voyager--the-skill-library), [Stanford Generative Agents](#33-stanford-generative-agents--memory-stream--reflection--planning), [O-Mem](#36-o-mem--omni-memory-for-personalization), [CoALA](#34-coala--cognitive-architectures-for-language-agents), [Reflexion](#35-reflexion--verbal-reinforcement-learning), and reflection patterns. Key questions: How do agents build persistent world models? What lies beyond context windows? How does identity persist across sessions?

---

## 1. Executive Summary

Agent memory is the critical bottleneck separating today's stateless LLM agents from persistent, self-improving systems. The frontier has converged on a shared taxonomy (working, episodic, semantic, procedural memory), a shared insight (context windows are RAM, external stores are disk), and a shared gap (no system yet solves identity persistence across sessions at scale). This survey maps six landmark systems onto Gas Town's architecture, identifies what Gas Town already does well (Dolt-backed persistence, formula-driven procedural memory), and highlights the key capabilities it lacks (structured reflection, self-editing memory, skill libraries).

---

## 2. Memory Taxonomy: The Consensus Framework

### 2.1 The Cognitive Memory Model

Recent surveys ([ACM TOIS 2025](https://arxiv.org/abs/2404.13501), ICLR 2026 MemAgents Workshop) converge on a taxonomy drawn from cognitive science:

| Memory Type | Cognitive Analog | Function | Example Systems |
|-------------|-----------------|----------|-----------------|
| **Working** | Short-term / scratchpad | Active context for current decision cycle | All agents (context window) |
| **Episodic** | Autobiographical | Records of past events and experiences | [Stanford Generative Agents](#33-stanford-generative-agents--memory-stream--reflection--planning), [Reflexion](#35-reflexion--verbal-reinforcement-learning) |
| **Semantic** | Factual knowledge | Stable knowledge about the world and self | [CoALA](#34-coala--cognitive-architectures-for-language-agents), [O-Mem](#36-o-mem--omni-memory-for-personalization), knowledge graphs |
| **Procedural** | Skills / how-to | Executable behaviors and action patterns | [Voyager](#32-voyager--the-skill-library) skill library, code-based agents |

An evolutionary framework (2026) formalizes three stages of memory maturation:
1. **Storage** — raw trajectory preservation (conversation logs)
2. **Reflection** — trajectory refinement (pattern extraction, error analysis)
3. **Experience** — trajectory abstraction (generalized skills, principles)

### 2.2 The OS Metaphor

[MemGPT](https://arxiv.org/abs/2310.08560) established the dominant metaphor: LLM context windows are RAM; external stores are disk. The agent manages its own virtual memory through function calls that page data between tiers. This metaphor is now near-universal:

- **In-context (RAM)**: Working memory, core memory blocks — fast, limited, expensive
- **Recall (swap)**: Conversation history — searchable, automatically persisted
- **Archival (disk)**: Vector/graph databases — large, indexed, requires explicit retrieval

**Key insight**: The agent must manage its own memory. Passive storage (dumping everything into a vector DB) fails because retrieval without curation produces noise. Active self-editing — where the agent rewrites its own memory blocks — is what separates MemGPT-class systems from naive RAG.

---

## 3. Landmark Systems

### 3.1 [MemGPT](https://arxiv.org/abs/2310.08560) / [Letta](https://github.com/letta-ai/letta) — The Memory Operating System

**Core idea**: Treat the LLM as a processor with a fixed-size context window (RAM), and give it system calls to manage a tiered memory hierarchy.

**Architecture**:
- **Core Memory**: Editable blocks pinned to context (persona, user info). The agent can rewrite these at any time.
- **Recall Memory**: Full conversation history, auto-persisted, searchable by text or date.
- **Archival Memory**: External vector/graph store for processed, indexed knowledge.
- **Eviction & Summarization**: When context fills, ~70% of messages are evicted with recursive summarization. Older messages progressively lose influence.

**Key innovations**:
- Self-editing memory: Agents autonomously update their own core memory blocks
- Sleep-time compute: Memory management happens asynchronously during idle periods
- Conversations API: Shared memory across parallel user interactions

**Relevance to Gas Town**: Gas Town polecats currently start from scratch each session. MemGPT's self-editing core memory maps directly to the `MEMORY.md` auto-memory pattern already in use, but Gas Town lacks the tiered retrieval (recall + archival) and the autonomous eviction/summarization loop.

**[Letta](https://github.com/letta-ai/letta) V1 (2025-2026)**: Moved away from heartbeat-driven loops toward native reasoning with direct assistant message generation. The architecture now follows modern ReAct/Claude Code patterns rather than the original MemGPT loop.

### 3.2 [Voyager](https://arxiv.org/abs/2305.16291) — The Skill Library

**Core idea**: An embodied agent in Minecraft that continuously explores, acquires skills as executable code, and stores them in a growing library for reuse.

**Architecture** (three components):
1. **Automatic Curriculum**: Maximizes exploration by proposing progressively harder tasks
2. **Skill Library**: Executable code snippets stored with natural-language descriptions, retrievable by semantic similarity
3. **Iterative Prompting**: Environment feedback + execution errors + self-verification drive program improvement

**Key innovations**:
- Skills are temporally extended, interpretable, and compositional — they compound
- Skill library transfers to new worlds (generalization without retraining)
- No fine-tuning required — pure blackbox GPT-4 queries

**Performance**: 3.3x more unique items, 2.3x longer distances, 15.3x faster tech tree milestones vs. prior SOTA.

**Relevance to Gas Town**: Voyager's skill library is the closest analog to Gas Town's [formula system](../../design/gas-city-formula-engine-vision.md). Formulas ARE procedural memory — reusable, composable work patterns. The gap: Gas Town formulas are human-authored and static. Voyager's skills are agent-discovered and grow autonomously. A Gas Town analog would be polecats that propose new formula steps based on successful completion patterns.

### 3.3 [Stanford Generative Agents](https://arxiv.org/abs/2304.03442) — Memory Stream + Reflection + Planning

**Core idea**: 25 agents in a sandbox town (Smallville) maintain memory streams, generate reflections, and plan daily activities, producing emergent social behavior.

**Architecture** (three components):
1. **Memory Stream**: Timestamped natural-language records of all observations. Retrieval combines:
   - Recency (exponential decay)
   - Importance (LLM-rated 1-10 score)
   - Relevance (embedding similarity to current context)
2. **Reflection**: Periodically synthesizes memories into higher-level inferences ("Klaus Mueller is dedicated to his research" from dozens of individual observations). Reflections are themselves memories, creating a recursive hierarchy.
3. **Planning**: Translates reflections + environment into high-level plans, recursively decomposed into hourly/minute-level actions.

**Key innovations**:
- Reflection creates abstraction layers over raw experience
- The retrieval function's three-factor scoring (recency × importance × relevance) is elegant and effective
- Ablation studies prove each component is necessary — removing reflection or planning degrades believability

**Relevance to Gas Town**: The reflection mechanism is the most important missing piece for Gas Town. Polecats accumulate experience (bead history, completion ledger) but never synthesize it into higher-level patterns. A reflection system could periodically analyze a polecat's completion history and generate reusable insights ("when implementing formula engine features, always check the Lean formalization first").

### 3.4 [CoALA](https://arxiv.org/abs/2309.02427) — Cognitive Architectures for Language Agents

**Core idea**: A unifying framework (not a system) that maps language agents onto cognitive science concepts, providing a design vocabulary.

**Framework**:
- **Working Memory**: Active variables for the current decision cycle — perceptual inputs, active knowledge, goals
- **Long-Term Memory**: Episodic (event records), Semantic (world knowledge), Procedural (skills as code + LLM weights)
- **Action Space**: Internal (reasoning, retrieval, learning) + External (grounding in environments; see [R4: Tool Ecosystems](r4-tool-ecosystems.md))
- **Decision Cycle**: Plan (propose → evaluate → select) → Execute → Observe → repeat

**Key insight on procedural memory**: Writing to procedural memory (modifying the agent's own code or behavior patterns) is "significantly riskier" than other memory types. This matches Gas Town's experience — [formula](../../design/gas-city-formula-engine-vision.md) modifications have higher blast radius than data changes.

**Relevance to Gas Town**: CoALA provides the vocabulary to describe Gas Town's existing architecture:
- Beads = episodic memory (event records of work)
- Design docs = semantic memory (world knowledge)
- Formulas = procedural memory (how-to-work patterns)
- Context window = working memory
- `gt prime` = memory retrieval (loading relevant context)

Gas Town is already a cognitive architecture — it just doesn't know it yet.

### 3.5 [Reflexion](https://arxiv.org/abs/2303.11366) — Verbal Reinforcement Learning

**Core idea**: Instead of gradient-based learning, agents generate natural-language reflections on their failures, store them as episodic memory, and use them to improve on subsequent attempts.

**Architecture**:
- Agent attempts a task
- On failure, generates a verbal self-reflection ("I failed because I didn't check the edge case where...")
- Reflection is stored in an episodic memory buffer
- Next attempt includes prior reflections as context
- Repeat until success or budget exhausted

**Performance**: 91% pass@1 on HumanEval (vs. GPT-4's 80% at the time). 8% improvement from self-reflection over episodic memory alone.

**Key insight**: "Semantic gradient" — natural language reflections provide richer learning signal than scalar rewards. The agent doesn't just know it failed; it knows WHY it failed and WHAT to try differently.

**Relevance to Gas Town**: Gas Town already has the infrastructure for this. Bead notes and design fields can store reflections. The completion ledger records success/failure. What's missing is the reflection generation step — a systematic process where agents analyze their failures and persist structured lessons. The Witness role is closest to this (monitoring polecat health), but it doesn't generate transferable lessons.

### 3.6 [O-Mem](https://arxiv.org/abs/2511.13593) — Omni Memory for Personalization

**Core idea**: Active user profiling that dynamically extracts and updates characteristics from interactions, supporting hierarchical retrieval of persona attributes and topic-related context.

**Architecture**:
- Proactive extraction of user characteristics from interactions
- Hierarchical memory: persona attributes (stable) + topic context (dynamic)
- Avoids semantic-grouping-only retrieval (catches semantically irrelevant but important information)
- Reduces retrieval noise through structured organization

**Relevance to Gas Town**: O-Mem's approach to user profiling maps to Gas Town's need for operator profiling. Different operators have different preferences, codebases, and conventions. The auto-memory (MEMORY.md) system is a primitive version of this.

---

## 4. Key Questions Answered

### 4.1 How Do Agents Build Persistent World Models?

The frontier answer has three layers:

1. **Episodic accumulation**: Raw experience records (memory streams, conversation logs, bead histories). Every system does this. It's necessary but insufficient — raw data grows without bound and retrieval degrades.

2. **Semantic consolidation**: Periodic compression of episodes into stable knowledge. Stanford agents do this through reflection. EverMemOS uses engram-inspired consolidation cycles. Temporal Semantic Memory (TSM) distinguishes recording time from event time. The key operation is MERGING — multiple episodes into one stable fact.

3. **Procedural crystallization**: Successful action patterns become reusable skills. Voyager stores skills as code. CoALA recognizes this as the highest-risk, highest-reward memory operation. When a pattern is crystallized into procedural memory, it persists indefinitely but is also hardest to correct if wrong.

**The missing layer**: No current system builds genuine causal world models — representations that predict "if I do X, Y will happen" beyond pattern matching. Current agents accumulate correlations, not causal structure. This is an open research frontier. See also [R6: Emergent Computation](r6-emergent-computation.md) for related self-organization patterns.

### 4.2 What Lies Beyond Context Windows?

Three approaches compete:

| Approach | Mechanism | Tradeoff |
|----------|-----------|----------|
| **Longer windows** | 100K-2M token contexts | Expensive, attention degrades with length, no compression |
| **External memory** | Vector/graph DBs with retrieval | Cheap storage, but retrieval is lossy and noisy |
| **Hybrid (MemGPT-class)** | Tiered memory with active management | Best of both, but complex to implement correctly |

Recent benchmarks (March 2026) show fact-based memory achieves **91% reduction in p95 response latency** vs. full-context processing, with comparable or better accuracy (see also [R5: Production Deployments](r5-production-deployments.md)). The verdict: external memory with smart retrieval beats longer context windows for persistent agents.

**The cost-performance frontier**: One-time extraction cost (writing to memory) amortized over many reads is fundamentally cheaper than re-processing full context every call. This is the same insight that makes databases faster than flat files.

### 4.3 How Does Identity Persist Across Sessions?

This is the hardest question, and current answers are unsatisfying:

**What works today**:
- **Persona blocks** (MemGPT): Static descriptions ("You are agent X with role Y") loaded at session start. Simple, effective for role-playing, but not genuine identity.
- **Memory accumulation** (Stanford agents): Identity emerges from accumulated experiences. "I am the agent who solved these problems, made these mistakes, learned these lessons." More authentic but fragile — depends on retrieval quality.
- **Completion ledgers** (Gas Town): Track record as identity. "I am what I have done." This is actually one of the more robust approaches.

**What doesn't work yet**:
- **Self-modification of core values**: Agents can update their persona blocks, but there's no mechanism to prevent drift toward adversarial or degenerate states.
- **Cross-session continuity**: Even MemGPT requires explicit session initialization. True continuity (agent "wakes up" knowing exactly where it left off) requires solving the context reconstruction problem.
- **Identity under distribution shift**: When the underlying model changes (upgrades), accumulated memories may become incoherent with the new model's reasoning patterns.

**The frontier question**: Is persistent identity even desirable for task-execution agents? Gas Town polecats are ephemeral by design — they execute and self-destruct. The Capability Ledger approach (identity-as-track-record) may be more appropriate than identity-as-continuous-experience for this use case.

---

## 5. Multi-Agent Memory Coordination

*See also [R1: Orchestration Frontier](r1-orchestration-frontier.md) for the broader multi-agent coordination landscape.*

### 5.1 Shared vs. Distributed Memory

| Pattern | Description | Example |
|---------|-------------|---------|
| **Centralized** | Shared repository accessible by all agents | Knowledge graphs, shared databases |
| **Distributed** | Each agent maintains own memory + sync protocols | Gas Town (each polecat has own worktree) |
| **Hybrid** | Private working memory + shared long-term store | Most practical multi-agent systems |

Gas Town is already hybrid: polecats have private worktrees (working memory) with shared Dolt-backed beads (long-term semantic/episodic store). This is architecturally sound per the frontier literature.

### 5.2 Communication as Memory

Multi-agent memory research identifies four communication paradigms:
1. **Memory-based**: Shared knowledge repositories (Gas Town beads)
2. **Report-based**: Status updates and progress (Gas Town mail)
3. **Relay**: Sequential information passing (Gas Town formulas/molecules)
4. **Debate**: Argumentative consensus building (not yet in Gas Town)

Gas Town implements 3 of 4 paradigms. The missing one (debate/consensus) would enable agents to jointly refine shared knowledge — useful for design decisions.

### 5.3 Consistency and Synchronization

The key challenge in multi-agent memory is **information asymmetry** — different agents may have different knowledge. Gas Town handles this through Dolt (version-controlled database), which provides:
- Atomic transactions (each bead write is a Dolt commit)
- Full history (any prior state is recoverable)
- Conflict detection (concurrent writes are caught)

This is actually ahead of most multi-agent frameworks, which use eventually-consistent vector stores without transactional guarantees.

---

## 6. Implications for Gas Town

*For a systematic comparison of Gas Town capabilities vs. frontier, see [S1: Gap Analysis](s1-gap-analysis.md).*

### 6.1 What Gas Town Already Has (and Does Well)

| Capability | Gas Town Implementation | Frontier Equivalent |
|-----------|------------------------|-------------------|
| Episodic memory | Beads (issue history, completion ledger) | Memory streams |
| Semantic memory | Design docs, CLAUDE.md, auto-memory | Knowledge bases |
| Procedural memory | Formulas, skills, molecule steps | Skill libraries |
| Working memory | Context window + `gt prime` | CoALA working memory |
| Multi-agent coordination | Dolt-backed shared state | Centralized + distributed hybrid |
| Identity-as-track-record | Capability Ledger | Novel — not common in literature |

### 6.2 What Gas Town Lacks (Ranked by Impact)

**High impact, achievable now:**

1. **Structured reflection**: Polecats should generate natural-language reflections after task completion. These reflections feed into the Capability Ledger and become retrievable episodic memory for future polecats working on similar tasks.

2. **Self-editing memory blocks**: The auto-memory (`MEMORY.md`) pattern is primitive. A MemGPT-style system where polecats can update structured memory blocks (not just append to a flat file) would enable richer context engineering.

3. **Skill discovery from completions**: Voyager-style skill extraction from successful formula completions. When a polecat discovers an effective pattern (e.g., "always run `lake build` before checking diagnostics in Lean projects"), it should be extractable as a reusable skill.

**Medium impact, requires design:**

4. **Retrieval-augmented context loading**: `gt prime` currently loads a fixed context. A smarter system would retrieve relevant episodic memories based on the current task (similar bead histories, past failures on this codebase).

5. **Sleep-time memory consolidation**: Between sessions, a background process could consolidate raw bead notes into structured knowledge. This maps to Letta's sleep-time compute. See [R2: Reactive Dataflow](r2-reactive-dataflow.md) for incremental computation patterns applicable here.

6. **Debate protocol for design decisions**: Enable multiple agents to jointly refine shared knowledge through structured argumentation.

**Long-term, requires research:**

7. **Causal world models**: Move beyond "I've seen this pattern before" to "I know this will happen because of this mechanism." Requires advances in the underlying LLM reasoning capabilities.

8. **Identity continuity under model upgrades**: Ensure accumulated memories remain coherent when the underlying model changes. No current system solves this well.

### 6.3 The Gas Town Advantage

Gas Town has an underappreciated structural advantage: **Dolt provides version-controlled, transactional memory with full history**. Most frontier systems use vector stores (lossy, eventually consistent) or flat files (no transactional guarantees). Dolt gives Gas Town:
- Atomic memory operations (no partial writes)
- Full audit trail (every memory change is tracked)
- Branch-based memory isolation (polecats can't corrupt shared state)
- SQL-queryable memory (structured retrieval without embedding search)

This is architecturally superior to what MemGPT, Voyager, or Stanford agents use for persistence. The gap is not in storage — it's in the intelligence layer that decides WHAT to store and WHEN to retrieve.

---

## 7. Research Frontier: Open Questions

1. **Memory forgetting**: All systems focus on remembering. None systematically forget. But bounded-resource agents MUST forget — the question is how to forget wisely. Gas Town's session ephemerality is actually a form of aggressive forgetting. Is it too aggressive?

2. **Memory verification**: How do you know a memory is correct? Agents can accumulate false beliefs. No current system has robust memory verification. Gas Town's bead system (with linked evidence) is a primitive form of provenance tracking.

3. **Transfer between agents**: Can one agent's memories be useful to another? Voyager's skill library says yes for procedural memory. Stanford agents suggest yes for episodic memory (agents gossip). Gas Town's shared beads enable this, but there's no mechanism for one polecat to learn from another's reflection.

4. **Memory as attack surface**: Self-editing memory creates a new vulnerability. If an agent can be tricked into writing false information to its core memory, that false belief persists indefinitely. Poisoned memory is the agent-era equivalent of prompt injection.

5. **Compression vs. fidelity**: The Gas City effect algebra already models this tradeoff (see [S2: Abstraction Map](s2-abstraction-map.md) and [S3: Architecture Sketch](s3-architecture-sketch.md)). Memory consolidation IS lossy compression. The question is whether the fidelity preorder from Gas City's formal model can be applied to memory operations.

---

## 8. References

### Core Systems
- Packer et al. (2023). ["MemGPT: Towards LLMs as Operating Systems."](https://arxiv.org/abs/2310.08560) arXiv:2310.08560
- Wang et al. (2023). ["Voyager: An Open-Ended Embodied Agent with Large Language Models."](https://arxiv.org/abs/2305.16291) arXiv:2305.16291
- Park et al. (2023). ["Generative Agents: Interactive Simulacra of Human Behavior."](https://arxiv.org/abs/2304.03442) UIST '23. arXiv:2304.03442
- Sumers et al. (2024). ["Cognitive Architectures for Language Agents."](https://arxiv.org/abs/2309.02427) TMLR. arXiv:2309.02427
- Shinn et al. (2023). ["Reflexion: Language Agents with Verbal Reinforcement Learning."](https://arxiv.org/abs/2303.11366) NeurIPS 2023. arXiv:2303.11366
- Li et al. (2025). ["O-Mem: Omni Memory System for Personalized, Long Horizon, Self-Evolving Agents."](https://arxiv.org/abs/2511.13593) arXiv:2511.13593

### Surveys and Frameworks
- Zhang et al. (2024). ["A Survey on the Memory Mechanism of Large Language Model-based Agents."](https://arxiv.org/abs/2404.13501) ACM TOIS. arXiv:2404.13501
- "Memory in LLM-based Multi-agent Systems: Mechanisms, Challenges, and Collective." TechRxiv 2025.
- ["Rethinking Memory Mechanisms of Foundation Agents in the Second Half: A Survey."](https://arxiv.org/abs/2602.06052) arXiv:2602.06052 (2026).
- "From Storage to Experience: A Survey on the Evolution of LLM Agent Memory Mechanisms." Preprints.org 2026.
- ICLR 2026 Workshop Proposal: "MemAgents: Memory for LLM-Based Agentic Systems."

### Recent Advances
- ["Beyond the Context Window: Cost-Performance Analysis of Fact-Based Memory vs. Long-Context LLMs."](https://arxiv.org/abs/2603.04814) arXiv:2603.04814 (2026).
- "EverMemOS: Engram-inspired Memory Architecture for Persistent Agents." 2025.
- "Temporal Semantic Memory (TSM): Time-aware fact consolidation for agents." 2025.
- [Letta V1 Architecture Blog](https://www.letta.com/blog/letta-v1) (2025): Rearchitecting from MemGPT to modern agent patterns.
- Qu et al. (2024). ["RISE: Recursive Introspection for Self-improvement."](https://arxiv.org/abs/2407.18219) NeurIPS 2024.

### Gas Town Internal
- [Gas City Design Synthesis](../../design/gas-city-synthesis.md) (2026-03-08)
- [Gas City Formula Engine Vision](../../design/gas-city-formula-engine-vision.md) (2026-03-08)
- Capability Ledger design (polecat lifecycle docs)
