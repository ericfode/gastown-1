# R4: Tool Ecosystems and MCP Evolution

**Research Phase** | Priority 1 | Bead: gt-xm8
**Date**: 2026-03-08

---

## Executive Summary

The agent tool ecosystem has undergone rapid standardization since late 2024. Three
complementary protocol layers have emerged: **[MCP](https://modelcontextprotocol.io/)** (tool access), **[A2A](https://google.github.io/A2A/)** (agent
coordination), and **Skills** (capability packaging). These are converging under the
**[Agentic AI Foundation (AAIF)](https://www.linuxfoundation.org/press/linux-foundation-forms-agentic-ai-foundation)** hosted by the Linux Foundation. The central question
for Gas Town: how to position within this stack.

---

## 1. MCP Spec Evolution

### Timeline

| Date | Version | Key Changes |
|------|---------|-------------|
| Nov 2024 | Initial | [Anthropic open-sources MCP](https://www.anthropic.com/news/model-context-protocol) |
| Jun 2025 | [2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18) | Structured tool outputs, OAuth auth, elicitation, JSON-RPC batching removed |
| Sep 2025 | — | MCP Registry preview launched |
| Nov 2025 | [2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25) | Tasks primitive (async/long-running), [OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1), SEP governance process |
| Dec 2025 | — | [MCP donated to AAIF](https://www.linuxfoundation.org/press/linux-foundation-forms-agentic-ai-foundation) under Linux Foundation |
| Mar 2026 | Current | 97M+ monthly SDK downloads, 10,000+ active servers |

### Key Architectural Shifts

**Structured Tool Outputs** (Jun 2025): Tools return typed JSON matching schemas
instead of free-text. This enables programmatic composition — an agent can reliably
pipe one tool's output into another.

**Elicitation** (Jun 2025): Servers can pause tool execution and request user input
via structured JSON schemas. This breaks the "fire-and-forget" model and enables
interactive workflows where tools need human decisions mid-execution.

**Tasks Primitive** (Nov 2025): The biggest shift. MCP servers can now create
long-running tasks that return handles, publish progress updates, and deliver
results asynchronously. This moves MCP from synchronous RPC into workflow territory.

**[OAuth 2.1](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1) Authorization** (Nov 2025): Protected Resource Metadata discovery plus
[OpenID Connect](https://openid.net/developers/how-connect-works/) for authorization server resolution. Enables enterprise deployment
where tools have real access control, not just API keys.

**SEP Governance** (Nov 2025): Specification Enhancement Proposals provide a formal
process for spec changes. This signals maturity — the protocol is stable enough
that changes need structured review.

### MCP Registry

Launched September 2025 as an open catalog and API for publicly available MCP
servers. Design principles: single source of truth, vendor-neutral, progressive
enhancement. The registry itself is open-source, allowing sub-registries.

As of late 2025: 40+ listed servers from Microsoft, GitHub, Dynatrace, Terraform,
and others. Kong announced an enterprise MCP Registry within Kong Konnect for
February 2026.

**API freeze** (v0.1) hit October 2025, signaling stability for integrators.

### Scale

- 97 million monthly SDK downloads
- 10,000+ active MCP servers
- First-class client support across all major AI platforms
- 17,000+ servers indexed on [MCP.so](https://mcp.so) alone

---

## 2. [A2A Protocol](https://google.github.io/A2A/) (Google)

### What It Solves

MCP handles **vertical integration** (agent-to-tool). A2A handles **horizontal
coordination** (agent-to-agent). They are explicitly complementary, not competing.

### Core Concepts

- **Agent Cards**: JSON documents at `.well-known/agent-card.json` describing
  capabilities, endpoints, and communication requirements. Registered with [IANA](https://www.iana.org/assignments/well-known-uris/)
  in April 2025 — first AI-agent-specific well-known entry.
- **Tasks**: Defined lifecycle (submitted, working, input-required, completed,
  failed, canceled) with streaming updates.
- **Client/Remote model**: Client agents formulate tasks, remote agents execute
  them. Clean delegation pattern.
- **UX Negotiation**: Agents adapt output format based on client capabilities.

### Timeline

| Date | Event |
|------|-------|
| Apr 2025 | [Google announces A2A](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/) with 50+ partners |
| Jun 2025 | [Linux Foundation launches A2A project](https://www.linuxfoundation.org/press/linux-foundation-launches-the-a2a-project), 100+ companies |
| Aug 2025 | [ACP](https://github.com/i-am-bee/beeai/tree/main/acp) (IBM) merges into A2A |
| Jan 2026 | A2A RC v1.0 |
| Feb 2026 | A2A v0.3 with gRPC support, signed security cards |

### Enterprise Adoption

S&P Global, ServiceNow (founding partner), and Twilio have adopted A2A for
cross-platform agent communication. See [R5: Production Deployments](r5-production-deployments.md)
for broader enterprise adoption patterns.

### Key Insight for Gas Town

A2A's Agent Card is the **DNS of agents** — it answers "what can this agent do
and how do I talk to it?" Gas Town's polecats could expose Agent Cards to enable
external agents to delegate work to the rig. The Task lifecycle maps cleanly to
Gas Town's bead lifecycle (open -> in_progress -> closed).

---

## 3. ACP -> A2A Merger

IBM launched the Agent Communication Protocol (ACP) in March 2025 for its [BeeAI](https://github.com/i-am-bee/beeai)
platform. ACP used standard HTTP conventions (cURL-compatible, no special
libraries). In August 2025, ACP officially merged with A2A under the Linux
Foundation, consolidating the agent-to-agent protocol space.

This merger is significant: it means the industry is converging on A2A as THE
agent-to-agent standard, rather than fragmenting across competing protocols.

---

## 4. [ANP](https://github.com/agent-network-protocol/agent-network-protocol) (Agent Network Protocol)

ANP targets **decentralized, internet-scale** agent networks. Unlike A2A's
client-server model, ANP enables peer-to-peer communications without central
registries.

ANP answers: "What does an internet of agents look like?" — no central authority,
agents discover each other organically. Still community-driven and early-stage
compared to MCP/A2A.

### Relevance to Gas Town

Gas Town's architecture is closer to ANP's vision than A2A's. The town is a
self-organizing network of agents (mayor, witness, polecats, refinery) that
discover work and coordinate without a central orchestrator dictating every
interaction (see also [R6: Emergent Computation](r6-emergent-computation.md)).
However, Gas Town's current scope is single-machine, not internet-scale.

---

## 5. Skills Ecosystem: The Package Manager for Agent Capabilities

### The Skills Revolution (Jan-Feb 2026)

Vercel launched **[skills.sh](https://skills.sh)** in January 2026 — a CLI and directory for
installing agent skill packages. Think npm for agent capabilities.

`npx skills add <package>` installs a skill to any supported agent.

### Scale (as of Feb 2026)

- 283,000+ skills on SkillsMP
- 5,700+ on ClawHub
- 17,000+ MCP servers on [MCP.so](https://mcp.so)
- Growing at ~147 new skills/day
- Supported by 30+ agents including Claude Code, Cursor, GitHub Copilot, Gemini

### What Skills Are

Skills are self-contained packages of procedural knowledge that agents can
invoke. They shift from "prompt engineering" to "skill engineering" — package
knowledge once, let every compatible agent use it.

Skills follow a shared Agent Skills specification, making them generally
cross-agent compatible.

### Security Concerns

A [Snyk](https://snyk.io/) scan of the ClawHub registry found **7.1% of skills (283) leak API
keys** — hardcoded credentials in source code that get copied into the user's
environment. This is the npm left-pad moment for agent skills: the ecosystem
is growing faster than security practices.

### Key Insight for Gas Town

Gas Town's "formula" system (mol-polecat-work, etc.) is a skills system. Each
formula is a reusable workflow template that any polecat can execute. The
difference: Gas Town formulas are tightly integrated with the bead lifecycle,
while skills.sh packages are stateless capability bundles.

Gas Town could expose formulas as skills, or consume external skills as
formula steps.

---

## 6. The AAIF: Governance Convergence

The **[Agentic AI Foundation (AAIF)](https://www.linuxfoundation.org/press/linux-foundation-forms-agentic-ai-foundation)**, formed December 2025 under the [Linux
Foundation](https://www.linuxfoundation.org/), provides neutral governance for agent protocols.

### Founding Members

Anthropic, Block, OpenAI (co-founders). Platinum: AWS, Bloomberg, Cloudflare,
Google, Microsoft.

### Projects Under AAIF

| Project | Origin | Purpose |
|---------|--------|---------|
| [MCP](https://modelcontextprotocol.io/) | Anthropic | Tool access protocol |
| [A2A](https://google.github.io/A2A/) | Google (+IBM ACP) | Agent-to-agent coordination |
| [goose](https://github.com/block/goose) | Block | Open-source agent framework |
| [AGENTS.md](https://github.com/anthropics/agents-spec) | OpenAI | Agent capability declaration |
| [BeeAI](https://github.com/i-am-bee/beeai) | IBM | Agent platform |
| [Docling](https://github.com/DS4SD/docling) | IBM | Document understanding |

### Significance

All major AI companies have committed to open, interoperable agent standards
under one governance body. This is unprecedented — it means the protocol layer
is not going to be a competitive battleground. The competition moves up-stack
to agent quality, not connectivity.

---

## 7. Capability Discovery: How Agents Find Each Other

### Current Discovery Mechanisms

| Mechanism | Protocol | How It Works |
|-----------|----------|-------------|
| [Agent Cards](https://google.github.io/A2A/#/documentation?id=agent-card) | A2A | `.well-known/agent-card.json` at known URLs |
| [MCP Registry](https://registry.modelcontextprotocol.io/) | MCP | Centralized catalog with API |
| [Skills.sh](https://skills.sh) | Skills | npm-style package registry |
| [AGENTS.md](https://github.com/anthropics/agents-spec) | OpenAI | Markdown file in repo root (convention) |
| DNS/well-known | General | Standard web discovery patterns |

### The Discovery Stack

1. **Static**: AGENTS.md in repo (what can this codebase's agent do?)
2. **Registry**: MCP Registry, skills.sh (search a catalog)
3. **Dynamic**: Agent Cards at well-known URLs (runtime discovery)
4. **Peer**: ANP-style gossip/DHT (no central authority)

### Gap

No unified discovery layer spans all protocols. An agent looking for
capabilities must check MCP Registry AND skills.sh AND Agent Cards AND
AGENTS.md. There is no "DNS for agent capabilities" yet.

---

## 8. Tool Composition Patterns

### The Four-Phase Tool Cycle

1. **Define**: Available tools described with structured schemas
2. **Select**: LLM chooses tool and parameterizes the call
3. **Invoke**: Tool executes
4. **Integrate**: Results feed back into the conversation/workflow

### Composition Patterns in 2026

**Sequential Pipelines**: Output of tool A feeds into tool B. MCP's structured
outputs (Jun 2025) made this reliable — typed JSON means no parsing ambiguity.

**Parallel Fan-out**: Multiple tools invoked simultaneously for independent
subtasks. Gas Town's polecat model is this pattern at the agent level.

**Orchestrator Pattern**: A "puppeteer" agent coordinates specialist agents
(see [R1: Orchestration Frontier](r1-orchestration-frontier.md) for a survey).
Gartner reports a 1,445% surge in multi-agent system inquiries (Q1 2024 to
Q2 2025). This is Gas Town's mayor/witness/polecat architecture.

**Reflection/Self-correction**: Agent invokes tool, evaluates result, retries
with adjusted parameters. MCP's Tasks primitive enables this for long-running
operations.

**Human-in-the-Loop**: MCP's elicitation enables tools to pause and request
human input. Gas Town's escalation pattern (polecat -> witness -> mayor)
is a multi-agent version of this.

---

## 9. How Do Tool Ecosystems Scale?

### Current Scaling Evidence

- MCP: 10,000+ servers, 97M monthly SDK downloads
- Skills: 283,000+ packages in 2 months
- A2A: 100+ enterprise partners

### Scaling Challenges

**Discovery**: Flat registries don't scale. With 10,000+ MCP servers, finding
the right one requires semantic search, not browsing. The MCP Registry and
skills.sh both face this.

**Security**: The skills ecosystem is growing faster than security tooling.
7.1% credential leak rate is dangerous at scale. Supply chain attacks on
agent skills are an emerging threat vector.

**Versioning**: No standard for skill/tool versioning and compatibility.
If a tool's schema changes, dependent agents break silently.

**Composition Complexity**: Tool A works alone. Tool B works alone. Together
they produce unexpected results. No standard for declaring tool compatibility
or conflicts.

**Trust**: Who verifies that a tool does what it claims? MCP servers and
skills are effectively arbitrary code. The trust model is "hope the author
is honest."

---

## 10. Key Questions Answered

### How do agents discover and compose capabilities?

Through a layered stack: static declarations (AGENTS.md), centralized
registries (MCP Registry, skills.sh), dynamic discovery (Agent Cards), and
emerging peer-to-peer protocols (ANP). Composition uses structured tool
schemas with typed JSON — the LLM selects tools, parameterizes calls, and
integrates results.

### What is the package manager for agent skills?

**skills.sh** (Vercel, Jan 2026) is the leading candidate. npm-style CLI,
283,000+ packages, 30+ agent support. But it's 2 months old — npm took years
to become stable. Security and governance are unsolved.

### How do tool ecosystems scale?

They scale like software package ecosystems (npm, PyPI) with the same
failure modes: security vulnerabilities, dependency hell, discovery problems.
The agent-specific challenge is **semantic discovery** — finding tools by
what they DO, not what they're named. MCP Registry and skills.sh both need
better search.

---

## 11. Implications for Gas Town

### What Gas Town Already Has

| Capability | Gas Town Equivalent | Industry Standard |
|------------|-------------------|-------------------|
| Agent coordination | Mayor/Witness/Polecat | A2A protocol |
| Tool access | MCP servers (lean-lsp, serena) | MCP |
| Capability packaging | Formulas (mol-polecat-work) | Skills.sh |
| Work discovery | Beads (bd ready) | Agent Cards + Registry |
| Task lifecycle | Bead statuses | A2A Task states |
| Orchestration | Mayor dispatch | Orchestrator pattern |

### What Gas Town Lacks

(See also [S1: Gap Analysis](s1-gap-analysis.md) for a structured assessment.)

1. **External interoperability**: No Agent Cards, no way for outside agents
   to discover or delegate to Gas Town polecats.
2. **Standardized capability declaration**: Formulas are internal; no
   AGENTS.md or equivalent for external consumption.
3. **Skills consumption**: No mechanism to install skills.sh packages as
   formula steps or MCP tool extensions.
4. **Dynamic tool discovery**: MCP servers are statically configured per
   agent, not dynamically discovered from a registry.
5. **Decentralized coordination**: All coordination flows through the mayor.
   No peer-to-peer agent communication (ANP-style).

### Strategic Options

(See [S3: Architecture Sketch](s3-architecture-sketch.md) for concrete designs.)

**Option A: Expose Agent Cards** — Make polecats discoverable via A2A Agent
Cards. External agents could delegate tasks to Gas Town rigs.

**Option B: Consume Skills** — Integrate skills.sh as a formula step type.
Polecats gain access to 283,000+ community capabilities.

**Option C: Federation via A2A** — Connect Gas Town rigs to each other and
to external systems via A2A. The mayor becomes an A2A client, polecats
become A2A remote agents.

**Option D: Publish Formulas as Skills** — Package Gas Town's proven
workflows (mol-polecat-work, etc.) as skills.sh packages. Contribute to the
ecosystem while gaining visibility.

---

## 12. The Protocol Stack (Summary)

```
Layer 4: Skills         (skills.sh, AGENTS.md)     — Capability packaging
Layer 3: Agent-Agent    (A2A, ANP)                  — Coordination
Layer 2: Agent-Tool     (MCP)                       — Tool access
Layer 1: Transport      (HTTP, gRPC, SSE, stdio)    — Communication
Layer 0: Identity/Auth  (OAuth 2.1, Agent Cards)    — Trust
```

Gas Town operates primarily at Layers 2-3 with custom implementations.
The industry is standardizing all layers under AAIF governance. The
convergence opportunity is real but not urgent — Gas Town's internal
protocols work. External interoperability is the gap.

---

## Cross-References

- [R1: Orchestration Frontier](r1-orchestration-frontier.md) — Framework survey, orchestration patterns
- [R2: Reactive Dataflow](r2-reactive-dataflow.md) — Event-driven architectures relevant to tool composition
- [R3: Agent Memory](r3-agent-memory.md) — Persistent state across tool invocations
- [R5: Production Deployments](r5-production-deployments.md) — Enterprise adoption of agent tooling
- [R6: Emergent Computation](r6-emergent-computation.md) — Decentralized coordination (ANP-aligned)
- [S1: Gap Analysis](s1-gap-analysis.md) — Gas Town's gaps vs. industry standards
- [S2: Abstraction Map](s2-abstraction-map.md) — Candidate Gas City abstractions
- [S3: Architecture Sketch](s3-architecture-sketch.md) — Concrete architecture for strategic options

## References

- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
- [MCP Specification — GitHub](https://github.com/modelcontextprotocol/specification)
- [MCP Spec 2025-06-18](https://modelcontextprotocol.io/specification/2025-06-18)
- [MCP Spec 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)
- [MCP Registry](https://registry.modelcontextprotocol.io/)
- [MCP.so — Server Directory](https://mcp.so)
- [Anthropic Announces MCP](https://www.anthropic.com/news/model-context-protocol)
- [A2A Protocol — Google](https://google.github.io/A2A/)
- [Google Announces A2A](https://developers.googleblog.com/en/a2a-a-new-era-of-agent-interoperability/)
- [A2A Protocol — IBM Explainer](https://www.ibm.com/think/topics/agent2agent-protocol)
- [ANP — Agent Network Protocol](https://github.com/agent-network-protocol/agent-network-protocol)
- [Skills.sh — Agent Skills CLI](https://skills.sh)
- [AGENTS.md Specification](https://github.com/anthropics/agents-spec)
- [Agentic AI Foundation (AAIF) — Linux Foundation](https://www.linuxfoundation.org/press/linux-foundation-forms-agentic-ai-foundation)
- [goose — Block](https://github.com/block/goose)
- [BeeAI — IBM](https://github.com/i-am-bee/beeai)
- [Docling — IBM](https://github.com/DS4SD/docling)
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1)
- [Snyk — Security Platform](https://snyk.io/)
- [IANA Well-Known URIs](https://www.iana.org/assignments/well-known-uris/)
