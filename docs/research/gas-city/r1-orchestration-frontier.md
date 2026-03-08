# R1: Orchestration Frontier Survey

**Bead**: gt-eth | **Date**: 2026-03-08 | **Author**: polecat/imperator

---

## Executive Summary

The multi-agent orchestration landscape in early 2026 is converging on a small
set of shared primitives — tool-calling agents, explicit handoffs, graph-based
state machines, and durable execution — while diverging on how much structure
to impose. The field is simultaneously maturing (production guardrails, tracing,
protocol standards) and hitting fundamental ceilings (error compounding, token
cost scaling, coordination overhead). This survey examines seven frameworks and
the emerging protocol layer to answer: what works, what failed, and where the
frontier is heading.

---

## 1. Framework-by-Framework Analysis

### 1.1 CrewAI

**Architecture**: Role-based agents organized into Crews (autonomous teams) and
Flows (deterministic event-driven pipelines). Crews handle adaptive
collaboration; Flows provide enterprise-grade auditability.

**DAG Composition**: Not graph-native. Crews use sequential or hierarchical
processes with inter-agent delegation. Flows add structured, event-driven
control on top. The composition model is implicit — agents decide routing
via role-based instructions rather than explicit graph edges.

**Strengths**:
- Highest-level abstraction; fast prototyping for multi-agent demos
- Crews + Flows duality covers both autonomous and deterministic needs
- Strong community adoption, well-documented

**Weaknesses**:
- ~3x token consumption vs single-agent LangChain equivalents ("managerial overhead")
- ~3x latency penalty from inter-agent messaging
- Implicit routing makes debugging hard; agent decisions are opaque
- Limited support for open protocol interoperability (MCP/A2A)

**Ceiling**: The role-playing abstraction trades precision for convenience.
Production teams hit walls when they need fine-grained control over agent
communication or deterministic replay of multi-agent conversations.

### 1.2 AutoGen (Microsoft) → Microsoft Agent Framework

**Architecture**: Originally conversation-based multi-agent orchestration.
AutoGen v0.4 adopted async, event-driven architecture with pluggable
components. Now in maintenance mode — merged with Semantic Kernel into the
Microsoft Agent Framework (1.0 GA targeted Q1 2026).

**DAG Composition**: Workflows are modeled as conversations between agents,
not explicit graphs. The Group Chat Manager pattern provides centralized
orchestration but becomes a bottleneck at scale.

**What Failed**:
- v0.2's rapid growth outpaced its API design — users hit architectural
  constraints, poor debugging, limited observability
- Centralized Group Chat Manager is a single point of failure
- Conversation-as-workflow metaphor breaks down for deterministic pipelines
- Two competing Microsoft frameworks (AutoGen + Semantic Kernel) caused
  ecosystem fragmentation

**Strategic Outcome**: Microsoft's answer was consolidation. Agent Framework
unifies both SDKs. AutoGen and Semantic Kernel enter maintenance mode
(security patches only, no new features). The lesson: competing internal
frameworks dilute ecosystem energy.

**Ceiling**: Conversation-based orchestration is intuitive but
non-deterministic. Enterprise users need replay, audit trails, and
guaranteed execution order — properties that conversation metaphors
resist.

### 1.3 LangGraph (LangChain)

**Architecture**: Low-level graph-based orchestration. Nodes are functions,
tools, or agents. Edges are conditional transitions. A centralized StateGraph
maintains context, supports parallel execution and conditional branching.

**DAG Composition**: Explicitly graph-native, but crucially supports cycles
(not just DAGs). This enables reflection loops, retry-with-feedback, and
multi-turn reasoning — patterns that pure DAG frameworks cannot express.

**Strengths**:
- Most flexible composition model; state machines with cycles
- Production-proven (Klarna, Replit, Elastic)
- Built-in checkpointing, state persistence, human-in-the-loop
- Strong integration with LangChain ecosystem

**Weaknesses**:
- Low-level: requires explicit graph construction (verbose)
- State management complexity grows with graph size
- Tight coupling to LangChain ecosystem
- Debugging cyclic graphs is harder than linear pipelines

**Ceiling**: LangGraph is the most expressive framework but imposes the
highest cognitive overhead. The gap between "possible" and "maintainable"
widens as graph complexity grows. Teams report that graphs beyond ~15 nodes
become difficult to reason about without visual tooling.

### 1.4 OpenAI Swarm → OpenAI Agents SDK

**Architecture**: Swarm was a minimalist reference design: agents with
instructions + tools, explicit handoffs (a tool call returning another Agent),
and shared conversation history. No persistent state between calls.

The Agents SDK (March 2025) is the production evolution: agents, tools,
handoffs, guardrails, and tracing as first-class primitives. Provider-agnostic
(supports 100+ LLMs). TypeScript SDK added in 2026. Voice agent support via
Realtime API.

**DAG Composition**: No explicit graph structure. Composition is via handoff
chains — each agent can hand off to any other by returning an Agent reference.
The topology is implicit in the handoff declarations.

**Key Design Insight**: "Provide the minimum set of primitives needed for agent
development and let developers compose them freely without imposing heavy
abstraction layers." The Agent class is a configuration object, not a state
machine.

**Strengths**:
- Minimal cognitive overhead; easy to understand and debug
- Built-in tracing captures full execution flow automatically
- Guardrails for input/output validation as first-class concept
- Provider-agnostic despite being OpenAI's SDK
- Temporal integration for durable execution

**Weaknesses**:
- Handoff chains are hard to visualize at scale
- No built-in graph structure limits complex routing patterns
- Stateless between calls — every handoff must carry full context
- Less suited for long-running, stateful workflows without Temporal

**Ceiling**: Simplicity is both strength and ceiling. Complex multi-agent
topologies require bolting on external orchestration (Temporal, custom state
management). The framework explicitly punts on durability.

### 1.5 Anthropic Agent Patterns

**Architecture**: Not a framework but a set of composable patterns documented
in "[Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)" (Dec 2025). Six patterns: prompt chaining,
routing, parallelization, orchestrator-workers, evaluator-optimizer, and
autonomous agents.

**Key Pattern — Orchestrator-Workers**: A central LLM dynamically decomposes
tasks, delegates to worker LLMs, and synthesizes results. Unlike
parallelization (pre-defined fan-out), the orchestrator determines subtasks
at runtime based on input.

**Production Implementation**: Anthropic's own [multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system)
uses an orchestrator-worker pattern. [Claude Code](https://paddo.dev/blog/claude-code-hidden-swarm/) uses a hidden multi-agent
swarm with TeammateTool (launched with Opus 4.6, still experimental — no
session resumption, no nested teams).

**Design Philosophy**: "Start with simple prompts, optimize them with
comprehensive evaluation, and add multi-step agentic systems only when
simpler solutions fall short." Anti-framework: composable patterns over
monolithic orchestration.

**Strengths**:
- Pattern-oriented thinking prevents over-engineering
- Each pattern has clear "when to use" criteria
- Framework-agnostic — implementable in any language/framework
- Proven at scale in Anthropic's own products

**Weaknesses**:
- Not a framework — no reusable runtime, no built-in persistence
- TeammateTool is experimental with known limitations
- Requires significant engineering to make production-ready
- No built-in tracing, guardrails, or protocol support

**Ceiling**: Patterns are necessary but insufficient. Teams need runtime
infrastructure (durability, observability, failure recovery) that patterns
alone don't provide. The gap between "understand the pattern" and "run it
in production" is where frameworks earn their value.

### 1.6 Prefect

**Architecture**: Python-native workflow orchestration engine. Decorators
(`@flow`, `@task`) convert functions into observable, retryable pipeline
steps. Hybrid architecture: managed orchestration service + private execution
infrastructure.

**AI Agent Support**: Wraps Pydantic AI agents with durable execution —
automatic retries, result caching, task-level observability. ControlFlow
(introduced 2025) provides structured AI workflow management. [MCP](r4-tool-ecosystems.md)
server integration enables connecting Claude Code, Cursor, and other AI tools.

**DAG Composition**: Traditional DAG-based task dependencies with decorators.
Tasks can depend on upstream task results. No native support for cycles or
agent-to-agent handoffs.

**Strengths**:
- Battle-tested orchestration (6M+ monthly downloads, early 2026)
- Python-native with minimal boilerplate
- Strong observability and retry semantics
- Hybrid architecture good for enterprise data sovereignty

**Weaknesses**:
- DAG-only: no native cycles or dynamic agent routing
- AI agent support is bolted on (ControlFlow, Pydantic AI wrappers)
- Not designed for real-time agent-to-agent communication
- Better suited for data pipelines than conversational multi-agent systems

**Ceiling**: Prefect excels at structured data workflows but its DAG model
is a poor fit for the dynamic, cyclic nature of agent reasoning. ControlFlow
bridges some gaps but the fundamental paradigm is batch-oriented.

### 1.7 Temporal

**Architecture**: Durable execution engine. Workflows are code (Go, Java,
Python, TypeScript, .NET, Ruby). Activities are retryable units of external
interaction. The platform guarantees exactly-once execution semantics even
across failures, restarts, and deployments.

**AI Agent Integration**: Official [OpenAI Agents SDK integration](https://www.devopsdigest.com/temporal-integrates-with-openai) (2025-2026).
Workflows orchestrate agents with durability, visibility, and auto-saved
state. Temporal Nexus (GA) connects workflows across isolated namespaces.
Multi-Region Replication with 99.99% SLA.

**DAG Composition**: Workflows are imperative code, not declared graphs.
Loops, conditionals, fan-out/fan-in, and dynamic spawning are all native.
The topology emerges from code execution, not a static declaration.

**Strengths**:
- Strongest durability guarantees in the ecosystem
- Workflows as code — full language expressiveness (loops, branches, etc.)
- Battle-tested at scale (Uber, Netflix, Stripe origins)
- Exactly-once semantics, automatic retries, timeouts
- Language-agnostic with multiple SDK options

**Weaknesses**:
- Infrastructure overhead: requires Temporal Server (or Temporal Cloud)
- Higher operational complexity than framework-only solutions
- Learning curve for workflow/activity separation pattern
- Overkill for simple agent interactions
- Not agent-aware natively — agent semantics are user-space

**Ceiling**: Temporal solves the durability problem definitively but doesn't
provide agent-specific abstractions (reasoning, tool-calling, handoffs).
It's infrastructure, not a framework. The ceiling is not in Temporal itself
but in the difficulty of mapping agent concepts onto workflow primitives.

---

## 2. The Protocol Layer: MCP and A2A

See also [R4: Tool Ecosystems and MCP Evolution](r4-tool-ecosystems.md) for
deeper analysis of MCP's tool ecosystem.

### 2.1 Model Context Protocol (MCP)

Anthropic's MCP has become the **de facto standard** for connecting AI agents
to tools and context. MCP defines a structured interface between agents and
external capabilities (databases, APIs, file systems).

**Key distinction**: MCP treats external capabilities as **tools** — structured
I/O, predictable behavior, no autonomy. This is deliberate: tools are
controllable; agents are not.

### 2.2 Agent2Agent Protocol (A2A)

Google's A2A (April 2025) complements MCP by enabling agent-to-agent
communication where both parties are autonomous. Key capabilities:

- **Agent Cards**: JSON-format capability discovery
- **Task lifecycle**: Defined states for cross-agent task management
- **Context sharing**: Agents exchange instructions and context
- **UI negotiation**: Agents adapt to different client capabilities

50+ technology partners at launch (Atlassian, Salesforce, SAP, etc.). Now
[housed by the Linux Foundation](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/) as an open-source project.

**Status**: MCP has wider adoption. A2A addresses a real gap (agent-to-agent
vs agent-to-tool) but adoption lags behind MCP. The two protocols are
complementary, not competing.

### 2.3 Protocol Convergence

The ecosystem is converging on a two-layer model:
- **MCP** for agent-to-tool communication (dominant, widely adopted)
- **A2A** for agent-to-agent communication (emerging, Linux Foundation stewardship)

Frameworks that support both protocols will have a structural advantage.
Currently, LangGraph and the OpenAI Agents SDK have the strongest protocol
integration stories.

---

## 3. Cross-Cutting Analysis

### 3.1 How Do They Handle DAG Composition?

| Framework | Composition Model | Cycles? | Dynamic Routing? |
|-----------|-------------------|---------|------------------|
| CrewAI | Implicit (role-based delegation) | No | Via agent decisions |
| AutoGen/MAF | Conversation-based | Limited | Via Group Chat Manager |
| LangGraph | Explicit state graph | **Yes** | Conditional edges |
| OpenAI Agents SDK | Handoff chains | No | Via tool-call returns |
| Anthropic Patterns | Pattern-dependent | Pattern-dependent | Orchestrator decides |
| Prefect | DAG with decorators | No | No |
| Temporal | Imperative code | **Yes** | Full language control |

**Observation**: Only LangGraph and Temporal natively support cycles. This
matters because reflection, retry-with-feedback, and iterative refinement
are core agent behaviors that require cyclic execution.

### 3.2 What Abstractions Failed?

1. **Conversation-as-workflow** (AutoGen): Modeling structured work as
   multi-party chat conflates communication with control flow. Led to
   Microsoft consolidating into Agent Framework.

2. **Role-playing-as-architecture** (CrewAI): Assigning "roles" to agents
   is intuitive but expensive (3x token overhead) and opaque. The role
   metaphor obscures what's actually happening at the execution level.

3. **Centralized orchestration** (AutoGen Group Chat Manager): Single-point
   coordination becomes a bottleneck and failure point. Distributed
   coordination (even if harder) scales better.

4. **DAG-only composition** (Prefect, early pipeline tools): Pure DAGs
   cannot express the iterative, reflective patterns that agents need.
   The industry moved to cyclic graphs and imperative workflows.

5. **Framework maximalism**: The heaviest frameworks (early AutoGen, complex
   CrewAI setups) demonstrated that more abstraction != more capability.
   Anthropic's pattern-first philosophy and OpenAI's minimal-primitive
   approach are winning mindshare.

### 3.3 What Is the Ceiling of Current Approaches?

**Error compounding**: When agents relay information through text, errors
compound. A hallucination in step 1 becomes "fact" in step 5. Research
shows multi-agent systems can exhibit [17x error rates](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) vs single agents
("bag of agents" anti-pattern).

**Token cost scaling**: Multi-agent systems consume 15x more tokens for ~90%
performance improvement. The cost curve is superlinear; the benefit curve
is sublinear.

**Coordination overhead**: Sequential agent execution is the primary latency
bottleneck. Most frameworks wait for one agent to complete before the next
begins.

**Fundamental tension**: Multi-agent systems are a workaround for the limits
of current LLMs. As base models become more capable, fewer tasks will require
multi-agent decomposition. The field may be building elaborate scaffolding
around a temporary limitation. See [R6: Emergent Computation](r6-emergent-computation.md)
for how self-organizing patterns may shift this calculus.

**Gartner projection**: 40% of enterprise apps will feature task-specific AI
agents by end of 2026 (up from <5% in 2025), but [40% of agentic AI projects
will fail](https://github.blog/ai-and-ml/generative-ai/multi-agent-workflows-often-fail-heres-how-to-engineer-ones-that-dont/) by 2027 due to inadequate risk controls.

### 3.4 What Are They Converging Toward?

Despite surface differences, the frameworks are converging on shared primitives:

1. **Tool-calling agents**: Every framework models agents as entities that
   reason and invoke tools. The tool-call is the universal unit of agent
   action.

2. **Explicit handoffs**: Moving away from implicit routing toward declared
   transfer points between agents (OpenAI's handoff pattern, CrewAI's
   delegation, LangGraph's conditional edges).

3. **Structured state management**: All production frameworks now include
   some form of persistent, inspectable [state](r3-agent-memory.md) (LangGraph's StateGraph,
   Temporal's workflow state, Agents SDK's tracing).

4. **Guardrails as first-class**: Input/output validation, not as an
   afterthought but as a core primitive (OpenAI Agents SDK leading here).

5. **Observability/tracing**: Automatic capture of execution traces for
   debugging and audit (OpenAI, LangGraph, Temporal all converging).

6. **Protocol adoption**: MCP for tool access, A2A for agent interop.
   Frameworks are becoming protocol-aware rather than protocol-defining.

7. **Durable execution**: Recognition that agent workflows need crash
   recovery, exactly-once semantics, and long-running execution support.
   Temporal's model is being adopted/integrated by others.

8. **Minimal primitives over maximal frameworks**: The trend is toward
   providing small, composable building blocks rather than comprehensive
   platforms. Anthropic's patterns and OpenAI's Agents SDK exemplify this.

---

## 4. Implications for Gas Town

See also [S1: Gap Analysis](s1-gap-analysis.md), [S2: Abstraction Map](s2-abstraction-map.md),
and [S3: Architecture Sketch](s3-architecture-sketch.md) for the synthesis that builds on
this survey.

Gas Town's architecture already embodies several frontier patterns:

| Frontier Pattern | Gas Town Equivalent |
|------------------|---------------------|
| Orchestrator-workers | Mayor → Polecats |
| Explicit handoffs | Hook system, `gt done` |
| Durable execution | Dolt-backed beads, git worktrees |
| Structured state | Beads with lifecycle states |
| Protocol layer | `gt mail`, `bd` CLI as internal protocols |
| Observability | Witness monitoring, capability ledger |
| Guardrails | Formula checklists, molecule workflows |

**Gaps relative to frontier**:
- No native cyclic execution (formulas are linear checklists)
- No standardized [MCP/A2A](r4-tool-ecosystems.md) integration for external agent interop
- Coordination is mail-based (high latency vs [event-driven](r2-reactive-dataflow.md))
- No built-in tracing/replay of agent execution paths

**Opportunities**:
- The formula/molecule system could evolve toward LangGraph-style [state
  graphs](r2-reactive-dataflow.md) with conditional edges and cycles
- [MCP server](r4-tool-ecosystems.md) integration would allow polecats to
  expose capabilities to external agents
- [Temporal-style durability](r5-production-deployments.md) guarantees could
  strengthen the already-durable worktree + Dolt foundation

---

## 5. Key Takeaways

1. **The framework wars are consolidating.** Microsoft merged AutoGen +
   Semantic Kernel. The market is gravitating toward LangGraph (graph
   expressiveness), OpenAI Agents SDK (minimal primitives), and Temporal
   (durable execution). CrewAI retains mindshare for rapid prototyping.

2. **Patterns beat frameworks.** Anthropic's "[Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)"
   has more lasting influence than any single SDK. The best practitioners
   pick patterns first, then choose (or build) minimal infrastructure.

3. **Durability is the unsolved problem.** Most agent frameworks punt on
   crash recovery and long-running execution. Temporal solves this but
   at infrastructure cost. The integration of Temporal with OpenAI Agents
   SDK signals where the industry is heading.

4. **Protocols will matter more than frameworks.** MCP and A2A are creating
   a shared substrate. Frameworks that are protocol-native will outlast
   those that are closed systems.

5. **Multi-agent is a transitional architecture.** As models improve, the
   sweet spot for multi-agent decomposition will shrink. Systems should be
   designed so that agent boundaries can be collapsed as model capability
   grows — not calcified into permanent architecture. See
   [R3: Agent Memory](r3-agent-memory.md) on how persistent identity may
   outlast the multi-agent pattern itself.

6. **The 17x error trap is real.** Naive multi-agent decomposition
   multiplies failure modes. The key engineering discipline is: decompose
   only when the task genuinely requires it, minimize inter-agent
   communication, and validate at every handoff boundary. See
   [R5: Production Deployments](r5-production-deployments.md) for real-world
   failure modes and mitigation strategies.

---

## Sources

- [CrewAI Documentation](https://docs.crewai.com/en/introduction)
- [CrewAI — AWS Prescriptive Guidance](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-frameworks/crewai.html)
- [AutoGen — Microsoft Research](https://www.microsoft.com/en-us/research/project/autogen/)
- [Microsoft Agent Framework Overview](https://learn.microsoft.com/en-us/agent-framework/overview/)
- [Microsoft Agent Framework: Convergence of AutoGen and Semantic Kernel](https://cloudsummit.eu/blog/microsoft-agent-framework-production-ready-convergence-autogen-semantic-kernel)
- [LangGraph: Agent Orchestration Framework](https://www.langchain.com/langgraph)
- [LangGraph Multi-Agent Orchestration Guide](https://latenode.com/blog/ai-frameworks-technical-infrastructure/langgraph-multi-agent-orchestration/)
- [Choosing the Right Multi-Agent Architecture — LangChain Blog](https://blog.langchain.com/choosing-the-right-multi-agent-architecture/)
- [OpenAI Agents SDK](https://openai.github.io/openai-agents-python/)
- [OpenAI Swarm GitHub](https://github.com/openai/swarm)
- [OpenAI Swarm Multi-Agent Framework in 2026](https://lexogrine.com/blog/openai-swarm-multi-agent-framework-2026)
- [Anthropic: Building Effective AI Agents](https://www.anthropic.com/research/building-effective-agents)
- [Anthropic: How We Built Our Multi-Agent Research System](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Claude Code's Hidden Multi-Agent System](https://paddo.dev/blog/claude-code-hidden-swarm/)
- [Prefect — Workflow Orchestration](https://www.prefect.io/)
- [Prefect ControlFlow Introduction](https://www.prefect.io/blog/controlflow-intro)
- [Temporal: Durable Multi-Agentic AI Architecture](https://temporal.io/blog/using-multi-agent-architectures-with-temporal)
- [Temporal + OpenAI Integration](https://www.devopsdigest.com/temporal-integrates-with-openai)
- [Google A2A Protocol Announcement](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [A2A Protocol — IBM](https://www.ibm.com/think/topics/agent2agent-protocol)
- [CrewAI vs LangGraph vs AutoGen vs OpenAgents (2026)](https://openagents.org/blog/posts/2026-02-23-open-source-ai-agent-frameworks-compared)
- [Why Multi-Agent LLM Systems Fail (arXiv)](https://arxiv.org/html/2503.13657v1)
- [17x Error Trap of the "Bag of Agents"](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/)
- [What ICLR 2026 Taught Us About Multi-Agent Failures](https://llmsresearch.substack.com/p/what-iclr-2026-taught-us-about-multi)
- [Multi-Agent Workflows Often Fail — GitHub Blog](https://github.blog/ai-and-ml/generative-ai/multi-agent-workflows-often-fail-heres-how-to-engineer-ones-that-dont/)
- [Top 5 Open-Source Agentic AI Frameworks in 2026](https://aimultiple.com/agentic-frameworks)
- [2026 Agentic Coding Trends Report — Anthropic](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf)
