# R6: Emergent Computation and Self-Organization

Research on emergent computation paradigms applicable to agent systems.

## 1. Swarm Intelligence

### Core Principles

Swarm intelligence (SI) studies how simple agents following local rules produce
complex global behavior without central control. Classic examples: ant colony
optimization, particle swarm optimization, bird flocking (boids).

Three foundational rules ([Reynolds, 1987](https://en.wikipedia.org/wiki/Boids)): separation, alignment, cohesion.
From these local interactions, global coordination emerges — no agent has a
global view, yet the swarm navigates, forages, and adapts.

### LLM-Powered Swarms (2025)

Recent work replaces hard-coded agent programs with LLM-driven prompts in
multi-agent simulations. Key papers:

- **["Multi-Agent Systems Powered by Large Language Models: Applications in
  Swarm Intelligence"](https://arxiv.org/abs/2503.03800)** (Frontiers in AI, 2025). Integrates
  LLMs with the NetLogo simulation platform via GPT-4o. Demonstrates LLMs
  can induce emergent behaviors in ant colony foraging and bird flocking
  scenarios. The approach enables studying self-organizing processes where
  agent rules are expressed in natural language rather than code.

- **["Benchmarking LLMs' Swarm Intelligence"](https://arxiv.org/abs/2505.04364)** (2025).
  Introduces SwarmBench with five coordination tasks: Pursuit, Synchronization,
  Foraging, Flocking, Transport. Finding: current LLMs significantly struggle
  with long-range planning and adaptive strategy formation under swarm-like
  constraints (limited local perception, restricted communication).

- **["Multi-Agent LLM Systems: From Emergent Collaboration to Structured
  Collective Intelligence"](https://www.preprints.org/manuscript/202511.1370)** (Preprints.org, 2025). Reports that naive "agent
  swarms" are prone to failure modes: degeneration of thought, majority
  herding, and overconfident consensus. Groups of interacting LLMs can improve
  factuality on some benchmarks, but the gains are fragile.

### Implications for Agent Systems

Swarm intelligence offers a model for coordination without central planning,
but LLM-based swarms reveal important limitations. Pure emergent coordination
works well for spatial/physical tasks but struggles with sequential reasoning.
Agent systems need hybrid approaches: swarm-like autonomy for parallelizable
work, structured coordination for reasoning-heavy tasks (see [R1: Orchestration Frontier](r1-orchestration-frontier.md)).

**Gas Town relevance**: Polecats operating independently on beads, with the
Witness and Refinery providing environmental signals (not direct control),
is a form of swarm coordination. The merge queue acts as a stigmergic medium.


## 2. Stigmergy

### Definition

Stigmergy is indirect coordination through the environment. The trace left
by one agent's action stimulates subsequent action by the same or different
agent. Coined by [Pierre-Paul Grassé (1959)](https://en.wikipedia.org/wiki/Stigmergy) studying termite nest construction.

Two forms:
- **Sematectonic stigmergy**: Physical changes to the environment (termites
  adding mud, ants depositing pheromones)
- **Sign-based stigmergy**: Markers/signals left in the environment
  (digital artifacts, status updates, shared state)

### Properties

- No direct agent-to-agent communication required
- Scales naturally — each agent interacts with the environment, not with
  every other agent (O(n) vs O(n^2) communication)
- Robust to agent failure — the environment persists
- Self-reinforcing: popular paths get more pheromone (positive feedback)
- Evaporative: unused paths decay (negative feedback / forgetting)

### Applications in Software Systems

Stigmergy has been applied to manufacturing control systems, where agents
coordinate through shared work-order states rather than direct messaging.
The MOSAIK framework (ESWC 2023) uses stigmergic principles for decentralized
energy system control.

In reinforcement learning, ["Stigmergic Independent Reinforcement Learning"](https://arxiv.org/abs/1911.12504)
uses environmental markers as an implicit communication
channel between independently learning agents, avoiding the complexity of
explicit message-passing protocols.

### Implications for Agent Systems

Stigmergy maps directly to shared-state coordination patterns:
- Git repositories as stigmergic media (code changes = pheromone traces)
- Issue trackers (beads) as sign-based stigmergy
- Build/test results as environmental feedback
- Merge queues as pheromone-reinforced paths

The key insight: agents don't need to talk to each other. They need to
read and write to a shared environment. The environment mediates all
coordination.

**Gas Town relevance**: The beads system is pure stigmergy. Polecats don't
message each other — they read/write beads, commit to git, and the Refinery
processes the environmental state. Mail exists but is discouraged ("default
to nudge"). The system already embodies stigmergic principles.


## 3. Market-Based Multi-Agent Coordination

### Core Mechanisms

Market-based control (MBC) applies economic metaphors to distributed
resource allocation:
- **Auctions**: Agents bid on tasks; highest bidder wins. Scalable,
  computationally cheap, reduced communication requirements.
- **Negotiation**: Multi-attribute negotiation for complex resource
  allocation under constraints.
- **Price signals**: Dynamic pricing communicates scarcity/demand without
  central planning.

### Recent Work

- **["From Competition to Coordination: Market Making as a Scalable Framework
  for Safe and Aligned Multi-Agent LLM Systems"](https://arxiv.org/abs/2511.17621)** (Nov 2025).
  Organizes agent interactions as structured economic exchanges. Each agent
  acts as a market participant, updating and trading probabilistic beliefs.
  Market-based coordination yields accuracy gains of up to 10% over single-shot
  baselines while preserving interpretability. By aligning local incentives
  with collective epistemic goals, the framework promotes self-organizing,
  verifiable reasoning without external enforcement.

- **["Decentralized Adaptive Task Allocation for Dynamic Multi-Agent Systems"](https://www.nature.com/articles/s41598-025-21709-9)**
  (Scientific Reports, 2025). Proposes adaptive mechanisms for task allocation
  in dynamic environments where agent capabilities and task requirements
  change over time.

- **[LLM-based Data Marketplace Simulation](https://arxiv.org/abs/2511.13233)** (2025). Buyer
  and seller agents powered by LLMs autonomously perform strategic actions
  (planning, searching, purchasing, pricing) in data marketplaces.

### Advantages Over Central Planning

- No single point of failure
- Naturally handles heterogeneous agent capabilities
- Price signals encode complex information compactly
- Agents self-select for tasks matching their comparative advantage
- Scales with number of agents (auction overhead is O(n), not O(n^2))

### Implications for Agent Systems

Market mechanisms could replace explicit task assignment:
- Agents bid on work items based on their current load, expertise, and
  available context
- Priority acts as a price signal
- The "merge queue" could become a marketplace where completed work
  competes for integration slots
- Resource allocation (context window, compute) could use internal pricing

**Gas Town relevance**: Currently Gas Town uses dispatch (central assignment
by Mayor/Witness). Market mechanisms would enable polecats to self-select
work, reducing the coordinator bottleneck. The beads priority system is a
primitive price signal.


## 4. Cellular Automata and Computational Emergence

### Foundations

Cellular automata (CA) demonstrate how simple local rules produce complex
global computation. Key results:
- [Conway's Game of Life](https://en.wikipedia.org/wiki/Conway%27s_Game_of_Life): Turing-complete from four rules
- [Wolfram's Rule 110](https://en.wikipedia.org/wiki/Rule_110): Proven universal (simple 1D automaton)
- [Evoloops](https://direct.mit.edu/artl/article/31/1/81/124368) (1999, 25th anniversary 2024): Darwinian evolution of
  self-reproducing organisms within deterministic cellular automata

### Neural Cellular Automata (NCA)

Modern extension: replace fixed rules with trainable neural networks.
Each cell runs a small neural network that observes its neighborhood and
updates its state. Properties:
- Self-organizing: global patterns emerge from local updates
- Self-repairing: damage to the pattern is autonomously repaired
- Differentiable: can be trained end-to-end with gradient descent

[Growing Neural Cellular Automata](https://distill.pub/2020/growing-ca/) (Mordvintsev et al., Distill 2020) showed
that complex morphogenesis can emerge from simple learned local rules.

### Locally Adaptive CA

["Locally Adaptive Cellular Automata for Goal-Oriented Self-Organization"](https://arxiv.org/abs/2306.07067)
(2023) introduces CA where cells adapt their rules based
on local context, enabling goal-directed self-organization without global
coordination.

### Implications for Agent Systems

The CA paradigm suggests that complex system behavior can emerge from
agents with:
1. Simple, uniform local rules
2. Only neighborhood awareness (no global state)
3. Synchronous or asynchronous update cycles
4. Self-repair capability (system recovers from agent failure)

The key question: can we design agent rules simple enough that emergence
happens reliably, yet expressive enough for useful computation?

**Gas Town relevance**: The formula system (mol-polecat-work) gives each
polecat uniform local rules. The Witness provides self-repair (restarting
stuck agents). But current rules are complex (8-step checklists), far from
CA-style simplicity. A more emergent design would have simpler rules with
richer environmental feedback.


## 5. Actor Model and Its Evolution

### Classical Actor Model

The [Actor Model](https://en.wikipedia.org/wiki/Actor_model) (Hewitt, 1973) defines concurrent computation through:
- **Actors**: Fundamental units with private state
- **Messages**: Only way to communicate (no shared state)
- **Behaviors**: Response to messages (can create actors, send messages,
  change own state)

Properties: location transparency, supervision hierarchies, fault isolation.
Implemented in [Erlang/OTP](https://www.erlang.org/), [Akka](https://akka.io/), [Microsoft Orleans](https://learn.microsoft.com/en-us/dotnet/orleans/), [Apache Pekko](https://pekko.apache.org/).

### Modern Evolution: Agentic Mesh

["Agentic Mesh"](https://medium.com/@visrow/agentic-mesh-revolutionizing-distributed-ai-systems-in-the-agentic-ecosystem-1062d036769a) (2025) extends the actor model for AI agent systems:
- Agents as autonomous nodes with specialized capabilities
- Dynamic capability discovery and composition
- Mesh networking for peer-to-peer coordination
- Service mesh patterns adapted for agent communication

### Agent-to-Agent Protocol (A2A)

Google introduced [A2A](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/) in 2025 for agent interoperability across frameworks
and vendors. Complements MCP (tool access — see [R4: Tool Ecosystems](r4-tool-ecosystems.md)) with agent-to-agent communication
standards.

### Evolving Orchestration

"Multi-Agent Collaboration via Evolving Orchestration" (OpenReview, 2025)
proposes dynamic orchestration where the coordination structure itself
evolves based on task demands — a puppeteer-style paradigm where
orchestration adapts to evolving task states.

### Implications for Agent Systems

The actor model provides proven patterns for:
- Fault isolation (one agent crash doesn't cascade)
- Supervision trees (Witness/Mayor hierarchy)
- Location transparency (agents don't care where other agents run)
- Mailbox semantics (async message processing)

The evolution toward agentic mesh suggests agents should be discoverable
services with well-defined capability interfaces, not just workers assigned
tasks.

**Gas Town relevance**: Gas Town already follows actor-model patterns —
polecats have private state (worktrees), communicate via mail (messages),
and the Witness acts as a supervisor. The gap is dynamic capability
discovery: polecats are currently homogeneous workers, not specialized
services.


## 6. Economic Coordination Mechanisms

### Beyond Auctions: Mechanism Design

Mechanism design (inverse game theory) asks: given desired outcomes, what
rules produce them? Key mechanisms:
- **Vickrey-Clarke-Groves (VCG)**: Truthful reporting is dominant strategy
- **Combinatorial auctions**: Bid on bundles, not individual items
- **Prediction markets**: Aggregate distributed information via prices
- **Token economies**: Internal currencies for resource allocation

### Relevance to Agent Resource Allocation

Agent systems face classic economic problems:
- **Allocation**: Which agent gets which task?
- **Scheduling**: In what order?
- **Pricing**: How much compute/context does each task deserve?
- **Information aggregation**: How to combine distributed observations?

Prediction markets could aggregate agent confidence: "how likely is this
PR to pass CI?" — agents bet internal tokens, and the market price reflects
collective belief. This provides better signal than any single agent's
estimate.

### The Scaling Science (2025)

["Towards a Science of Scaling Agent Systems"](https://arxiv.org/abs/2512.08296) (Dec 2025,
Google Research) provides rigorous empirical analysis:

**Three dominant effects:**
1. **Tool-coordination trade-off**: Tool-heavy tasks suffer
   disproportionately from multi-agent overhead under fixed compute budgets
2. **Capability saturation**: Coordination yields diminishing/negative
   returns once single-agent baseline exceeds ~45% accuracy
3. **Error amplification**: Independent agents amplify errors 17.2x;
   centralized coordination contains this to 4.4x

**Task-dependent findings:**
- Centralized coordination: +80.8% on parallelizable tasks
- Decentralized coordination: +9.2% on web navigation
- Sequential reasoning: every multi-agent variant degraded performance
  by 39-70%

**Predictive framework**: Predicts optimal coordination strategy for 87%
of held-out configurations.

### Implications for Agent Systems

The scaling science results are sobering: multi-agent coordination has
real costs, and the benefits are task-dependent. The key insight is that
coordination strategy must match task structure:
- Parallelizable work → multi-agent with centralized coordination
- Navigation/exploration → decentralized agents
- Sequential reasoning → single agent (multi-agent hurts)

**Gas Town relevance**: Gas Town dispatches independent polecats for
independent beads — this matches the "parallelizable work" category where
multi-agent coordination helps most. The Refinery's centralized merge queue
provides the centralized coordination that contains error amplification.
However, for complex reasoning tasks (architecture decisions, cross-cutting
refactors), a single-agent approach may outperform the current multi-polecat
dispatch.


## 7. Emergent Social Behavior

### Stanford Generative Agents (2023)

["Generative Agents: Interactive Simulacra of Human Behavior"](https://arxiv.org/abs/2304.03442) (Park et al.,
UIST 2023) placed 25 LLM-powered agents in a simulated
town (Smallville). Architecture: experience storage in natural language,
memory synthesis into higher-level reflections, dynamic retrieval for
behavior planning.

**Three emergent social behaviors observed:**
1. Information diffusion — agents spread initially private information
2. Relationship formation — agents developed social connections
3. Coordination — agents collectively organized a party without central
   planning (one agent planned it, told friends, friends invited others,
   everyone showed up)

This demonstrated that LLM agents can exhibit emergent social coordination
from simple architectural principles: memory, reflection, planning.

### Implications for Agent Systems

Generative agents show that meaningful coordination can emerge from:
- Persistent memory (agents remember past interactions)
- Reflection (agents synthesize experiences into beliefs)
- Planning (agents form and execute multi-step plans)
- Social protocols (agents follow conversational norms)

The party-planning emergence is significant: no agent was programmed to
organize a party. It emerged from individual agents' goals, memories, and
social interactions.

**Gas Town relevance**: Gas Town polecats currently lack persistent memory
across sessions (context dies with the session). The handoff mechanism
(`gt handoff`) provides primitive memory transfer, but there's no reflection
or synthesis. Adding [agent memory](r3-agent-memory.md) and reflection could enable emergent
coordination patterns beyond explicit formula steps.


## 8. Key Questions Answered

### What happens when agents coordinate without central control?

Three outcomes, depending on task structure:
1. **Parallelizable tasks**: Emergent coordination works well. Agents
   self-organize around independent work units. Stigmergic media (shared
   repos, issue trackers) provide sufficient coordination.
2. **Exploration tasks**: Decentralized coordination slightly outperforms
   centralized. Agents benefit from diverse search strategies.
3. **Sequential reasoning**: Multi-agent coordination consistently hurts.
   Context fragmentation and communication overhead destroy performance.

The critical factor is information fragmentation: parallel agents explore
diverse paths but must compress global context into inter-agent messages.
This lossy communication increases synchronization overhead.

### Can agent systems self-organize?

Yes, under specific conditions:
- **Simple, uniform local rules** (CA paradigm)
- **Rich environmental feedback** (stigmergy)
- **Aligned incentives** (market mechanisms)
- **Memory and reflection** (generative agents)

Self-organization fails when:
- Tasks require deep sequential reasoning
- Agent capabilities are heterogeneous but not discoverable
- Environmental signals are too sparse or too noisy
- Error amplification exceeds the coordination benefit

### What economic models work for agent resource allocation?

**Auction-based task allocation** is the most proven: scalable, low
communication overhead, naturally handles heterogeneous agents. More
sophisticated approaches:

- **Market-making** for epistemic tasks (belief trading)
- **Prediction markets** for aggregating distributed estimates
- **Dynamic pricing** for resource contention (compute, context window)
- **Token economies** for internal resource management

The key insight from mechanism design: the rules of interaction matter
more than individual agent intelligence. A well-designed market with
mediocre agents outperforms a poorly designed system with brilliant agents.


## 9. Synthesis: Patterns for Gas City

### What Gas Town Already Does Right

| Pattern | Gas Town Implementation |
|---------|----------------------|
| Stigmergy | Beads system, git repos as shared medium |
| Actor model | Polecats with private state, message passing |
| Supervision | Witness monitors, Mayor coordinates |
| Parallel dispatch | Independent beads to independent polecats |
| Self-repair | Witness restarts stuck agents |

### What the Research Suggests Adding

| Paradigm | Potential Enhancement |
|----------|---------------------|
| Market mechanisms | Self-selecting work (polecats bid on beads) |
| Prediction markets | Aggregate confidence on MR quality |
| Agent memory | Persistent cross-session learning |
| Reflection | Agents synthesize experience into reusable patterns |
| Dynamic orchestration | Coordination structure adapts to task type |
| Capability discovery | Polecats advertise specializations |
| Task-aware routing | Sequential tasks → single agent; parallel → swarm |

### Critical Insight from Scaling Science

The most important finding: **coordination overhead is real and measurable**.
Multi-agent systems are not universally better than single agents. The
optimal strategy depends on:
- Task parallelizability (high → multi-agent wins)
- Single-agent baseline capability (>45% → diminishing returns from coordination)
- Tool complexity (more tools → higher coordination tax)
- Error tolerance (independent agents amplify errors 17x)

Gas City should not assume "more agents = better." It should dynamically
select coordination strategy based on task structure, using the predictive
framework from the scaling science research. See also [S1: Gap Analysis](s1-gap-analysis.md)
and [S3: Architecture Sketch](s3-architecture-sketch.md) for how these patterns inform the design.


## Sources

- [Multi-Agent Systems Powered by LLMs: Applications in Swarm Intelligence](https://arxiv.org/abs/2503.03800) (arXiv, 2025)
- [Benchmarking LLMs' Swarm Intelligence — SwarmBench](https://arxiv.org/abs/2505.04364) (arXiv, 2025)
- [Multi-Agent LLM Systems: From Emergent Collaboration to Structured Collective Intelligence](https://www.preprints.org/manuscript/202511.1370) (Preprints.org, 2025)
- [From Competition to Coordination: Market Making for Multi-Agent LLM Systems](https://arxiv.org/abs/2511.17621) (arXiv, Nov 2025)
- [Towards a Science of Scaling Agent Systems](https://arxiv.org/abs/2512.08296) (arXiv, Dec 2025)
- [Generative Agents: Interactive Simulacra of Human Behavior](https://arxiv.org/abs/2304.03442) (UIST 2023)
- [Growing Neural Cellular Automata](https://distill.pub/2020/growing-ca/) (Distill, 2020)
- [Locally Adaptive CA for Goal-Oriented Self-Organization](https://arxiv.org/abs/2306.07067) (arXiv, 2023)
- [Self-Reproduction and Evolution in Cellular Automata: 25 Years After Evoloops](https://direct.mit.edu/artl/article/31/1/81/124368) (Artificial Life, 2024)
- [Stigmergy in Antetic AI](https://www.alphanome.ai/post/stigmergy-in-antetic-ai-building-intelligence-from-indirect-communication)
- [Stigmergic Independent Reinforcement Learning](https://arxiv.org/pdf/1911.12504) (arXiv, 2019)
- [Decentralized Adaptive Task Allocation](https://www.nature.com/articles/s41598-025-21709-9) (Scientific Reports, 2025)
- [ANTS 2026: International Conference on Swarm Intelligence](https://ants2026.org/)
- [Multi-Agent Coordination Patterns: Architectures Beyond the Hype](https://medium.com/@ohusiev_6834/multi-agent-coordination-patterns-architectures-beyond-the-hype-3f61847e4f86)
- [Agentic Mesh: Revolutionizing Distributed AI Systems](https://medium.com/@visrow/agentic-mesh-revolutionizing-distributed-ai-systems-in-the-agentic-ecosystem-1062d036769a)
- [LLM-based Data Marketplace Simulation](https://arxiv.org/abs/2511.13233) (arXiv, 2025)
- [Google A2A Protocol Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [The Confluence of Evolutionary Computation and Multi-Agent Systems](https://www.ieee-jas.net/article/doi/10.1109/JAS.2025.125246) (IEEE JAS, 2025)
