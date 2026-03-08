# R5: Production Agent Deployments

Research survey of what actually works at scale in production AI coding agent
systems as of early 2026. Covers architectures, human intervention points,
scaling failures, and real bottlenecks.

## Systems Surveyed

| System | Org | Model | Production Status |
|--------|-----|-------|-------------------|
| [Devin](https://cognition.ai/) | Cognition | Compound (planner/coder/critic) | Thousands of enterprise customers |
| [Factory Code](https://factory.ai) | Factory AI | Multi-agent (planner/worker/judge) | Enterprise SaaS |
| [Cursor](https://cursor.com/) | Cursor Inc | Proprietary + frontier LLMs | Millions of developers |
| [Codex](https://openai.com/index/introducing-the-codex-app/) | OpenAI | GPT-5.3-Codex | Cloud SaaS, worktree-isolated |
| [SWE-agent](https://swe-agent.com/) | Princeton NLP | LLM-agnostic | Research → production bridge |
| [OpenHands](https://docs.all-hands.dev/) | OpenHands | LLM-agnostic | Open-source SDK + cloud |
| [Augment Code](https://www.augmentcode.com/) | Augment | Multi-model | Enterprise (100K+ file repos) |
| [Claude Code](https://docs.anthropic.com/en/docs/claude-code/overview) | Anthropic | Claude Opus/Sonnet | $1B+ ARR, enterprise + developer |

---

## 1. Architectures That Work at Scale

*See also: [R1: Orchestration Frontier](r1-orchestration-frontier.md)*

### 1.1 Compound Agent Systems (Devin)

Devin uses a **swarm of specialized models**, not a single monolithic LLM:

- **Planner**: High-reasoning model that outlines strategy
- **Coder**: Specialized model trained on trillions of tokens of code
- **Critic**: Adversarial model that reviews for security vulnerabilities and
  logic errors before execution

Each agent runs in its own **isolated virtual machine** in a cloud IDE. Multiple
Devin instances run in parallel, each with its own interactive environment.

**Key insight**: Specialization > generalization. Having distinct models for
planning, coding, and review yields higher merge rates than a single model
doing everything.

### 1.2 Role-Based Hierarchies ([Cursor](https://cursor.com/blog/scaling-agents), [Factory](https://factory.ai))

Cursor's scaling journey is the most instructive failure-to-success story:

**What failed**: Flat coordination. Agents of equal status self-coordinated
through a shared file. Each checked what others were doing, claimed tasks, and
updated status. Result: agents became risk-averse, avoided hard problems, made
small safe changes, and churned without progress. Locking mechanisms became
bottlenecks — twenty agents slowed to the throughput of two or three.

**What worked**: Role separation with hierarchy.

- **Planners**: Continuously explore the codebase and create tasks. Spawn
  sub-planners for specific areas, making planning itself parallel and recursive.
- **Workers**: Pick up tasks and focus entirely on completing them. No
  coordination with each other — just push changes when done.
- **Judge** (Factory variant): Determines whether to continue at each cycle end.

**Result**: Hundreds of agents working together on a single codebase for weeks,
making real progress. Cursor tested this by building a web browser from scratch
— agents ran for close to a week, writing 1M+ lines across 1,000 files.

### 1.3 Single-Threaded Master Loop (Claude Code)

Claude Code takes the **opposite** approach — deliberately simple architecture:

- Single-threaded agentic loop with disciplined tooling
- No vector databases or embeddings for search — relies on regex (Grep) and
  glob patterns, leveraging the model's inherent code understanding
- Layered tools: reading/discovery → code editing → execution
- Minimal diffs displayed, every change tracked for review/rollback

**Key insight**: Simplicity scales when the underlying model is capable enough.
Claude Code reached $1B+ ARR with this architecture, suggesting that
over-engineering agent coordination may be premature when the base model is
strong.

### 1.4 Event-Sourced Modular SDK ([OpenHands](https://docs.all-hands.dev/))

OpenHands V1 refactored from a monolithic sandbox-centric design to a modular
SDK with clear boundaries:

- **Event-sourcing pattern**: All interactions are immutable events appended to
  a log with deterministic replay (see also [R2: Reactive Dataflow](r2-reactive-dataflow.md))
- **Workspace abstraction**: Same agent runs locally for prototyping or remotely
  in secure containers with minimal code changes
- **Opt-in sandboxing**: Core SDK stays lightweight; sandboxing is a separate
  package
- **MCP integration**: Typed tool system with [Model Context Protocol](https://modelcontextprotocol.io/) support (see also [R4: Tool Ecosystems](r4-tool-ecosystems.md))

### 1.5 Sandbox-First Isolation ([Codex](https://openai.com/index/introducing-the-codex-app/))

OpenAI Codex operates entirely within secure, isolated containers:

- **Internet disabled** during task execution — agent can only access code
  provided via GitHub repos and pre-installed dependencies
- **Built-in worktree support** so multiple agents can work on the same repo
  without conflicts
- **System-level sandboxing** that is open-source and configurable
- Agents limited to editing files in their assigned folder/branch

---

## 2. Where Humans Still Intervene

### 2.1 Code Review Is the New Bottleneck

The most consistent finding across all systems: **human code review cannot keep
up with agent output velocity.**

- PR review time increases **91%** on teams with high AI adoption ([Google DORA 2025](https://dora.dev/research/2025/))
- PR sizes increase **154%** with AI adoption
- Bug rates climb **9%** with 90% AI adoption increase
- **67.3%** of AI-generated PRs get rejected vs **15.6%** for manual code ([LinearB](https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report))

Amdahl's Law applies: the system moves only as fast as its slowest link. AI
coding gains evaporate when review, testing, and release pipelines can't match
the new velocity.

### 2.2 Ambiguity Resolution

All systems struggle with ambiguous requirements:

- Devin is "senior-level at codebase understanding but junior at execution"
- Devin handles clear upfront scoping well but **not mid-task requirement changes**
- Agents perform worse when given incremental instructions after starting
- [UC San Diego/Cornell study (Dec 2025)](https://mikemason.ca/writing/ai-coding-agents-jan-2026/): professionals retain agency in design,
  insist on quality attributes, and deploy explicit control strategies

### 2.3 High-Stakes Decisions

The dominant 2026 pattern is **"bounded autonomy"**:

- Clear operational limits for agents
- **Mandatory escalation paths** to humans for high-stakes decisions
- Comprehensive audit trails
- Security, compliance, and auditability ranked as #1 priority (75% of leaders)

### 2.4 Deployment Models Reflect Trust Levels

Production systems use a spectrum of human involvement:

1. **Agent drafts**: Suggest actions, human executes (lowest trust)
2. **Approval gates**: Agent executes limited actions with human approval
3. **Controlled autonomy**: Agent operates autonomously in controlled environments
4. **Event-driven async**: Agent triggers on events, runs asynchronously (highest trust)

---

## 3. What Broke When They Scaled

### 3.1 Sandbox Provisioning (Cursor)

**Provisioning time becomes the bottleneck.** The model generates a solution in
milliseconds, but creating a secure, isolated environment takes much longer.
Cursor had to build custom sandboxing infrastructure and rewrite their VM
scheduler to handle bursty demand — spinning up thousands of sandboxes
simultaneously was the core infrastructure challenge.

### 3.2 Flat Coordination (Cursor)

**No hierarchy → no progress.** With equal-status agents and lock-based
coordination:

- Twenty agents slowed to the throughput of two or three
- Most time spent waiting for locks
- Agents avoided difficult tasks, making only small safe changes
- Work churned without progress

**Fix**: Explicit role separation (planner/worker hierarchy).

### 3.3 Quality Collapse at Volume

- **[METR study](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/)**: Experienced developers were **19% slower** with early-2025 AI
  tools while **believing they were 20% faster** — a 39-percentage-point
  perception gap
- No significant correlation between AI adoption and better outcomes at company
  level ([DORA metrics](https://dora.dev/research/2025/): deployment frequency, lead time, change fail rate, MTTR)
- Companies with heavy AI usage didn't ship faster or more reliably

### 3.4 MoE Expert Imbalance ([Cursor](https://cursor.com/blog/scaling-agents))

When using Mixture of Experts models, if every token routes to the same expert,
that expert becomes a bottleneck while others sit idle, causing high tail latency.
Sequential generation (plans, tool arguments, diffs, explanations) remains
fundamentally slow.

### 3.5 Test Pollution and Environment Drift

Orphan resources (test databases, temporary containers, stale sandboxes)
accumulate on production servers and degrade performance. This is a recurring
operational problem in any system running many agent instances.

---

## 4. Actual Bottlenecks

### 4.1 Human Review Throughput

The #1 bottleneck across all production systems. Agent output velocity has
outpaced human review capacity. No system has fully solved this.

Partial mitigations:
- Devin added an adversarial **Critic** model for pre-review
- Cursor uses automated testing gates before human review
- Factory uses a **Judge** agent to filter before human review
- Claude Code tracks all changes for review/rollback

### 4.2 Context Window and Codebase Understanding

*See also: [R3: Agent Memory](r3-agent-memory.md)*

Large codebases remain challenging:

- [Augment Code](https://www.augmentcode.com/)'s context engine processes 400K-500K files across multiple repos
  — purpose-built for this problem
- Claude Code uses regex/glob search rather than embeddings, relying on model
  capability
- SWE-agent's mini-agent variant simplified the approach for better flexibility

### 4.3 Quality Assurance at Scale

**32% of teams** cite quality as the top production barrier ([LangChain State of Agent Engineering](https://www.langchain.com/state-of-agent-engineering)). Observability is
table stakes (**89% have implemented it**), but quality assurance at agent
output volume remains unsolved.

### 4.4 Trust and Governance

Most teams struggle to productionize agentic systems due to trust, transparency,
and governance gaps. Agents stall at pilot stage because organizations can't
verify agent behavior meets compliance requirements.

### 4.5 Sequential Token Generation

Generation is fundamentally sequential. Agents spend significant time producing
plans, tool arguments, diffs, and explanations — generating these token by token
is slow. No architectural trick has eliminated this bottleneck.

---

## 5. Production Metrics That Matter

### 5.1 [Devin](https://cognition.ai/blog/devin-annual-performance-review-2025) (18 months in production)

- **Hundreds of thousands** of PRs merged
- PR merge rate: **67%** (up from 34%)
- **4x faster** problem solving, **2x more efficient** resource usage
- 20-30% of well-scoped agent PRs merge with no revisions
- 40-50% merge after one round of human feedback
- Deployed at Goldman Sachs, Santander, Nubank
- One org saved 5-10% of total developer time on security fixes
- Another saw **20x efficiency gain** (1.5 min vs 30 min per vulnerability)

### 5.2 Claude Code

- **$1B+ ARR** by Nov 2025, estimated $2B by early 2026
- Users running 24/7 development workflows (required usage limits)
- Spotify: **90% reduction** in engineering time, 650+ AI-generated changes/month,
  ~50% of updates flow through the system

### 5.3 Cursor Multi-Agent

- 1M+ lines of code across 1,000 files in a week-long run
- Hundreds of agents coordinating on a single codebase
- Required complete architecture rewrite (flat → hierarchical) to achieve this

### 5.4 [SWE-agent](https://swe-agent.com/) Evaluation Infrastructure ([AI21](https://www.ai21.com/blog/scaling-agentic-evaluation-swe-bench/))

- ~500 Kubernetes pods for parallel evaluation
- Up to **8K parallel runs**, 20 min wall time for full SWE-bench eval
- Each pod serves dozens of sequential/parallel runs

---

## 6. Implications for Gas Town

*See also: [S1: Gap Analysis](s1-gap-analysis.md), [S2: Abstraction Map](s2-abstraction-map.md), [S3: Architecture Sketch](s3-architecture-sketch.md)*

### 6.1 What Gas Town Already Gets Right

| Production Pattern | Gas Town Equivalent |
|---|---|
| Role-based hierarchy | Polecat/Witness/Refinery/Mayor |
| Isolated execution environments | Git worktrees per polecat |
| Event-sourced state | Beads (Dolt-backed issue tracking) |
| Mandatory escalation paths | `gt escalate` + Witness monitoring |
| Bounded autonomy | Formula-driven molecule workflows |
| Quality gates before merge | Refinery bisecting merge queue |

### 6.2 Gap Areas to Investigate

- **Adversarial review**: No equivalent to Devin's Critic model — Refinery runs
  gates but doesn't do adversarial code review
- **Recursive planning**: No sub-planner spawning — molecules are linear checklists,
  not recursive decomposition
- **Context engine for large repos**: No equivalent to Augment's 400K-file
  indexing — relies on agent search capability
- **Observability**: No structured agent telemetry beyond beads audit trail
- **Parallel agent scaling**: Current architecture is 1 polecat = 1 worktree = 1
  task, no multi-agent collaboration on single tasks (see [R6: Emergent Computation](r6-emergent-computation.md))

### 6.3 Key Takeaways

1. **Simplicity wins** when the base model is strong enough (Claude Code proves this)
2. **Role separation is essential** for multi-agent coordination (Cursor learned
   this the hard way)
3. **Human review is the universal bottleneck** — invest in automated quality
   gates, not just agent output velocity
4. **Bounded autonomy** is the production pattern — escalation paths and audit
   trails are table stakes
5. **Sandbox provisioning** becomes the infrastructure bottleneck at scale
6. **Quality > quantity** — high AI adoption doesn't correlate with better
   outcomes without the surrounding systems

---

## Sources

- [Devin 2025 Performance Review](https://cognition.ai/blog/devin-annual-performance-review-2025)
- [Devin 2.0 Announcement](https://cognition.ai/blog/devin-2)
- [Cursor: Scaling Agents](https://cursor.com/blog/scaling-agents)
- [Cursor 2.0 Multi-Agent Architecture](https://www.artezio.com/pressroom/blog/revolutionizes-architecture-proprietary/)
- [OpenAI Codex App](https://openai.com/index/introducing-the-codex-app/)
- [OpenAI Harness Engineering](https://www.infoq.com/news/2026/02/openai-harness-engineering-codex/)
- [SWE-agent Architecture](https://swe-agent.com/latest/background/architecture/)
- [AI21 Scaling Agentic Evaluation](https://www.ai21.com/blog/scaling-agentic-evaluation-swe-bench/)
- [OpenHands Software Agent SDK](https://arxiv.org/abs/2511.03690)
- [OpenHands Platform (ICLR 2025)](https://arxiv.org/abs/2407.16741)
- [Augment Code](https://www.augmentcode.com/)
- [Claude Code Architecture](https://www.zenml.io/llmops-database/claude-code-agent-architecture-single-threaded-master-loop-for-autonomous-coding)
- [Claude Code Production Lessons](https://www.zenml.io/llmops-database/building-production-ai-agents-lessons-from-claude-code-and-enterprise-deployments)
- [Anthropic 2026 Agentic Coding Trends Report](https://resources.anthropic.com/hubfs/2026%20Agentic%20Coding%20Trends%20Report.pdf)
- [LangChain State of Agent Engineering](https://www.langchain.com/state-of-agent-engineering)
- [METR Developer Productivity Study](https://metr.org/blog/2025-07-10-early-2025-ai-experienced-os-dev-study/)
- [Faros AI Productivity Paradox](https://www.faros.ai/blog/ai-software-engineering)
- [AI vs Human Code Gen Report](https://www.coderabbit.ai/blog/state-of-ai-vs-human-code-generation-report)
- [Mike Mason: AI Coding Agents 2026](https://mikemason.ca/writing/ai-coding-agents-jan-2026/)
- [Factory AI](https://factory.ai)
