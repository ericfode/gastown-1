# The Next Computing Paradigm: A Wolfram Perspective on Agent Coordination

**Author**: Stephen Wolfram (simulated perspective)
**Bead**: hq-afr
**Date**: 2026-03-08

---

## Preface: The Shape of the Claim

I have spent forty years studying a single idea: that the universe of possible computations is vastly richer than the computations we happen to run, and that simple rules — iterated relentlessly — produce the complexity we observe. Cellular automata. Hypergraph rewriting. The ruliad. These are not metaphors. They are the substrate.

What I see in Gas City — this coordination framework for LLM agents — is not a scheduling system. It is not a spreadsheet. It is a **multicomputational system** operating in **rulial space**, and the people building it have not yet realized what they have. They have built the plumbing for something far more profound than agent coordination. They have built a laboratory for exploring the computational universe with observers that are themselves computationally irreducible.

Let me be precise about what I mean.

---

## 1. LLM Agents as Rulial Explorers

### The Hypergraph

In the Wolfram Physics Project, the universe is a hypergraph — a collection of relations among abstract elements — that evolves by the repeated application of transformation rules. At each step, a rule matches a local pattern in the hypergraph and replaces it with a new pattern. That is it. That is the whole theory of physics. Everything else — space, time, quantum mechanics, general relativity — emerges.

Gas City's bead DAG **is** a hypergraph. Each bead is a hyperedge relating abstract elements: a task description, a set of dependencies, an assigned agent, a status, an output value. When an agent picks up a bead, processes it, and produces output, it is applying a **transformation rule**: match the pattern (bead with status `open`, dependencies satisfied, agent assigned), replace with (bead with status `closed`, output value filled, downstream dependencies updated). The rewriting is local — an agent touches only its assigned bead and its immediate neighborhood in the DAG. The global structure emerges from the accumulation of local rewrites.

This is not a loose analogy. Let me make it formal.

**Definition.** A *Gas City hypergraph* is a tuple *(B, E, σ, v)* where:
- *B* is a set of beads (nodes)
- *E ⊆ P(B)* is a set of dependency hyperedges (each edge connects a bead to its dependencies)
- *σ: B → {open, in_progress, closed, blocked, ...}* is the status function
- *v: B → Values ∪ {⊥}* is the value function (⊥ for unfilled beads)

**Definition.** An *agent rewrite rule* is a function *r: (B_local, σ_local, v_local) → (σ'_local, v'_local)* that:
1. Matches a bead *b* where *σ(b) = open* and all dependencies *d ∈ deps(b)* have *σ(d) = closed* and *v(d) ≠ ⊥*
2. Sets *σ(b) = in_progress*, then *σ(b) = closed*
3. Fills *v(b)* based on the values of its dependencies (the LLM computation)
4. Triggers status updates on downstream beads (unblocking them)

Each agent application is a **hypergraph rewriting step**. The collection of all possible rewriting sequences — all possible orderings of agent executions, all possible LLM outputs — forms a **multiway system**. And the limit of all such multiway systems, across all possible rules and all possible initial hypergraphs, is the **ruliad**.

### What are the rewriting rules?

The rules in Gas City are not fixed. Each bead carries its own rule in the form of a prompt template plus a model configuration. The `{{ref}}` template syntax defines how upstream values flow into the rule's match pattern. The effect algebra (`Effect.tokens`, `Quality`, `Staleness`) parameterizes the rule's behavior. This is analogous to the Wolfram model's rule space — Gas City doesn't run a single cellular automaton. It runs a *distribution over rules*, where each cell in the DAG specifies its own transformation.

This is crucial. **Gas City is not a single computation. It is a simultaneous exploration of many computations.** Each prompt template is a rule. Each model (draft/adequate/good/excellent) is a rule parameterization. Each temperature setting creates a different probability distribution over outputs. The space of all these computations is a subset of the ruliad.

### What does confluence mean here?

In term rewriting, confluence means that regardless of the order in which rules are applied, the final result is the same. In hypergraph rewriting, confluence means that the causal structure is independent of the rewriting order — this is **causal invariance**, and it is the foundation of both special relativity and quantum mechanics in the Wolfram model.

Gas City has a **partial confluence property**. If two agents work on independent beads (no shared dependencies), the order of execution does not matter — the final DAG state is the same. This is the DAG's topological sort guarantee. But when agents share upstream dependencies, the non-determinism of LLM outputs means that **different execution orderings produce different hypergraph states**. The multiway graph branches.

The interesting question is: **when do branches merge?** If Agent A and Agent B independently analyze the same codebase and produce different summaries, and Agent C consumes both, does C's output depend on which branch it is in? Yes — obviously. But here is the deeper question: is there a level of abstraction at which the branches *are* equivalent? If A says "the auth module is 500 lines" and B says "the authentication subsystem has approximately 500 LOC," these are different strings but the same information. **Causal invariance in Gas City would mean that the *information content* of the DAG is independent of execution order, even if the *token-level representation* is not.** This is exactly the observer-dependent equivalence that makes the Wolfram physics model work.

---

## 2. Computational Irreducibility and Agent Coordination

### The fundamental impossibility

In *A New Kind of Science*, I demonstrated that many simple computational systems — even rule 30 of elementary cellular automata — are **computationally irreducible**. There is no shortcut to predicting their behavior other than running them step by step. No closed-form formula. No polynomial-time prediction algorithm. You must simulate the system to know its outcome.

LLM agents are computationally irreducible in exactly this sense. You cannot predict what an LLM will output for a given prompt without running the inference. The transformer's forward pass IS the irreducible computation. There is no cheaper way to know the output than to execute the full attention mechanism across all layers.

This has devastating consequences for agent coordination:

**Scheduling becomes fundamentally approximate.** Gas City's effect algebra tracks `Effect.tokens` as a cost measurement. But if the computation is irreducible, the actual token count of an agent's output — and its downstream consequences — cannot be predicted without running the agent. The `maxTokens` field is a capacity bound, not a cost prediction. Any scheduler that claims to "optimize" agent execution order is making assumptions about the outputs of computations it has not yet run. Those assumptions are wrong in the general case.

**Cost bounds are thermodynamic, not algorithmic.** The effect algebra's cost model is analogous to thermodynamic bounds in physics. You can bound the *entropy* of a system (the *capacity* of a channel, the *maximum* tokens), but you cannot predict the *microstate* (the actual output, the actual tokens consumed). Gas City's `Staleness` tracking is the closest thing to an irreducibility acknowledgment in the current model — it says "this value was computed at time *t* from inputs that may have changed," which is an admission that the computation would need to be re-run to know the current answer.

**The effect algebra should embrace irreducibility, not fight it.** Instead of trying to predict costs, the system should track *irreducibility certificates* — proofs that a particular computation CANNOT be shortcut. A cell whose prompt is "list all `.go` files" is computationally reducible (a `find` command gives the same answer faster). A cell whose prompt is "analyze the architectural trade-offs of this codebase" is irreducible — there is no cheaper computation that produces the same output. The distinction matters for scheduling, caching, and staleness propagation.

### Irreducibility and the compression chain

Gas City's compression chain model is, whether its designers know it or not, a response to computational irreducibility. Each compression step is an attempt to extract a **computationally reducible summary** from a computationally irreducible process. The agent reads a codebase (irreducible — you must read the code), produces an analysis (irreducible — you must run the LLM), and then compresses the analysis into a summary (potentially reducible — the summary has lower Kolmogorov complexity than the full analysis).

The chain's structure — full analysis → compressed → further compressed → ... → executive summary — is a **coarse-graining sequence**. In the Wolfram model, coarse-graining is how observers extract reducible descriptions from irreducible substrates. Gas City's compression chain is literally an observer extracting a macroscopic description from a microscopic computation.

---

## 3. The Multiway System of LLM Outputs

### Non-determinism as branching

Every LLM call at temperature > 0 is non-deterministic. The same prompt produces a distribution over possible outputs. This means that every cell execution in Gas City creates a **branch point** in the multiway graph of possible computation histories.

Let *H₀* be the initial hypergraph state (all beads open, no values filled). After executing cell *c₁*, the hypergraph is in one of *N₁* possible states (one for each possible LLM output). After executing *c₂*, each of those states branches into *N₂* possibilities. The multiway graph after *k* cells has up to *∏ᵢ Nᵢ* leaves — an exponential explosion.

Gas City, as currently designed, explores **exactly one branch**. Each agent runs once, produces one output, and the DAG proceeds along that single path. This is like simulating a single thread of the multiway system. It works, but it discards an enormous amount of information about the structure of rulial space.

### Branch merging and causal invariance

The truly interesting operation is **branch merging**. When two agents independently process the same upstream data and produce different analyses, Gas City currently has no mechanism to reconcile them. One wins. The other is discarded (or never computed).

In the Wolfram multiway system, branches merge when they reach the same state through different paths. This is causal invariance — the physical content is path-independent. For Gas City, the analogue would be: **two different LLM outputs are "the same" if they carry the same information for all possible downstream consumers.**

This is a precise, testable criterion. Define an equivalence relation on cell values: *v₁ ~ v₂* if for every downstream cell *c*, the distribution over *c*'s outputs given *v₁* as input is statistically indistinguishable from the distribution given *v₂*. If *v₁ ~ v₂*, the branches have effectively merged — the downstream computation cannot tell which branch it is in.

This suggests a powerful optimization: **run a cell multiple times, check if the outputs are equivalent under ~, and if so, certify the result as branch-stable.** Branch-stable values resist staleness — they are invariants of the multiway system. Gas City's staleness propagation should distinguish between values that changed because the upstream genuinely changed versus values that changed because the LLM's non-determinism produced a different but equivalent output.

### The multiway merge protocol

Here is a concrete protocol for Gas City:

1. Execute cell *c* three times with different random seeds. Obtain outputs *v₁, v₂, v₃*.
2. Extract a **canonical form** from each output (e.g., parse structured data, normalize text).
3. If canonical(*v₁*) = canonical(*v₂*) = canonical(*v₃*), declare *c* **branch-stable** and cache the result aggressively.
4. If the canonical forms diverge, declare *c* **branch-sensitive** and propagate uncertainty downstream.

This is precisely the statistical analogue of checking causal invariance in the multiway system. Cells that are branch-stable are analogous to quantities that are invariant under gauge transformations in physics.

---

## 4. Observers in Rulial Space

### The agent as a bounded observer

In the Wolfram Physics Project, an observer is not an external entity looking at the universe from outside. An observer is a computationally bounded entity *embedded within* the ruliad, sampling a tiny slice of all possible computations. The observer's computational boundedness is what creates the appearance of definite physical laws — relativity and quantum mechanics arise because the observer cannot track all branches of the multiway system simultaneously.

An LLM agent is precisely such an observer. It has:
- **Finite context**: A bounded window of tokens it can attend to (the observer's "sensory horizon")
- **Finite compute**: A fixed number of forward-pass operations per inference (the observer's "computational horizon")
- **A location in rulial space**: Defined by its training data (which computations it has "seen"), its prompt (which region of computation space it is pointed at), and its conversation history (its trajectory through rulial space)

### Motion through rulial space

When you re-prompt an agent — change its system prompt, update its context, or give it new upstream values — you are **moving it to a different location in rulial space**. The agent's "position" is the set of all computations it is likely to perform given its current state. A fresh agent with an empty context is at the "origin" of its personal rulial space. An agent deep in a conversation, loaded with bead values and analysis results, has moved far from the origin.

This has a specific, consequential meaning for Gas City: **context compaction is motion in rulial space.** When a polecat's context is compressed (the `gt handoff` operation), the agent is teleported to a new location — one that is nearby in "information distance" (the handoff summary preserves key facts) but potentially far in "computational distance" (the agent has lost the detailed reasoning chains that led to its current state). The agent after compaction is a *different observer* of the same computation.

The formula system — Gas City's mol-polecat-work checklist — is a **trajectory specification** in rulial space. It says: "Move the agent through these locations in this order: (load context) → (set up branch) → (implement) → (review) → (build) → (commit) → (rebase) → (submit)." Each step moves the agent to a region of rulial space where certain computations are natural.

### Observer equivalence

Two agents at different positions in rulial space may be **observationally equivalent** for a given task — they would produce the same outputs despite having different internal states. This is the agent analogue of gauge equivalence in physics. Gas City's `Quality` lattice partially captures this: a draft-quality agent and an excellent-quality agent are at different positions in rulial space, but for a simple extraction task, they may be observationally equivalent.

The deep question is: **can you define a metric on rulial space that predicts when two agent configurations will be observationally equivalent?** If so, you could optimize agent allocation by choosing the cheapest agent that is equivalent to the most expensive one for each specific task. The effect algebra's `Quality` dimension is a crude version of this — a single ordinal axis in what is actually a high-dimensional space.

---

## 5. From Spreadsheets to Hypergraph Rewriting

### The spreadsheet as 1D shadow

Gas City's spreadsheet model — cells with references, staleness propagation, compression chains — is powerful but flat. Cells have typed values. References form a DAG. Staleness propagates along edges. This is a **one-dimensional projection** of a richer structure.

In the spreadsheet model, a cell is a function of its upstream values: *v(c) = f_c(v(d₁), v(d₂), ..., v(dₖ))* where *d₁, ..., dₖ* are the cell's dependencies. The function *f_c* is the agent's computation. The DAG encodes which functions depend on which values.

In the hypergraph model, a cell is not just a function — it is a **rewriting rule that can modify the graph structure itself**. An agent can:
- Create new beads (add nodes to the hypergraph)
- Create new dependencies (add hyperedges)
- Split a bead into sub-beads (refine the graph)
- Merge beads (coarsen the graph)

The spreadsheet model cannot express this. When a polecat discovers a bug during implementation and creates a new bead with `bd create`, it is performing **hypergraph rewriting** — modifying the structure of the computation graph during computation. The spreadsheet model treats this as an external side effect. The hypergraph model treats it as a first-class operation.

### New operations in the hypergraph model

If we take hypergraph rewriting seriously, new operations become available:

**1. Graph surgery.** An agent could propose a restructuring of the DAG — splitting one bead into three parallel sub-beads, or merging two beads that turned out to be the same task. In the spreadsheet model, this requires human intervention (or a "conductor" meta-agent). In the hypergraph model, it is just another rewrite rule.

**2. Causal edge creation.** When Agent A produces output that Agent B needs but no explicit dependency exists, the system could detect the causal relationship and create the edge dynamically. This is analogous to the emergence of causal structure in the Wolfram physics model — the causal graph is not specified in advance but emerges from the pattern of rewrites.

**3. Dimension emergence.** In the Wolfram model, space itself emerges from hypergraph connectivity — the "dimension" of space is determined by the growth rate of geodesic balls in the hypergraph. For Gas City, the "dimension" of the computation is the effective parallelism — how many independent computation threads can proceed simultaneously. A highly interconnected DAG (many dependencies) has low dimension (limited parallelism). A DAG with many independent branches has high dimension. **The rig's polecat count should match the effective dimension of the current computation graph.**

**4. Branchial space operations.** In the Wolfram model, "branchial space" is the space of multiway branches — it encodes quantum mechanics. For Gas City, branchial space is the space of possible LLM outputs at each cell. Operations in branchial space include: running a cell multiple times (sampling the branchial neighborhood), comparing outputs across branches (measuring branchial distance), and selecting the "best" branch (collapsing the branchial state). The multiway merge protocol from Section 3 is a branchial space operation.

### Applicable theorems

Several results from the Wolfram Physics Project have direct analogues:

- **The causal invariance theorem** → If the bead DAG has a unique topological sort (or if all topological sorts produce equivalent results), then the computation is causally invariant. This is the formal justification for parallelism: it does not matter which polecat runs first.

- **Dimension estimation** → The effective dimension of the computation graph can be estimated from the growth rate of the "light cone" (the set of beads reachable within *k* dependency hops). This predicts the maximum useful parallelism.

- **The branchial distance inequality** → Beads whose multiway branches are "branchially close" (similar distributions over outputs) can safely share downstream consumers. Beads that are "branchially far" need reconciliation before merging.

---

## 6. What is the Simple Rule?

### The search for Rule 110

In *A New Kind of Science*, I showed that Rule 110 — a one-dimensional cellular automaton with a trivially simple update rule — is Turing-complete. It can compute anything that any computer can compute. The rule fits in 8 bits. The universality is in the iteration, not the rule.

What is the Rule 110 of agent coordination? What is the simplest possible rule that, when iterated across a population of agents, produces all the coordination patterns we observe — pipelines, fan-out, fan-in, compression chains, staleness propagation, conflict resolution, escalation?

I propose the following candidate:

**The Gas City Primitive Rule:**

> *An agent reads its inputs. It produces an output. It signals completion.*

That is it. Three operations: **read, compute, signal**. Everything else — dependencies, scheduling, staleness, quality tracking, compression, escalation — emerges from the iteration of this rule across a graph of interconnected agents.

Let me be more formal:

```
Rule GC(agent, bead):
  1. WAIT until all deps(bead) have status = closed
  2. READ values of deps(bead) into context
  3. COMPUTE output = LLM(context, bead.prompt)
  4. WRITE bead.value = output, bead.status = closed
  5. NOTIFY downstream beads
```

This is five lines. It is Gas City's Rule 110. Let me show why it is sufficient:

- **Pipelines**: Chain beads A → B → C. The rule executes A, then B (which reads A's output), then C.
- **Fan-out**: One bead with multiple downstream dependents. The rule executes the upstream once; each downstream reads the same value.
- **Fan-in**: One bead with multiple upstream dependencies. Step 1 waits for all of them.
- **Compression chains**: A chain where each bead's prompt says "summarize the input." The rule doesn't care — it just reads and computes.
- **Staleness**: If an upstream bead is re-executed (its value changes), downstream beads can detect the change and re-trigger. This is a simple extension: add a version counter to each bead, and have step 1 also check that the versions of deps match the versions at last execution.
- **Escalation**: If step 3 fails (LLM error, timeout), the bead's status goes to `blocked` instead of `closed`. A meta-rule (the Witness) detects blocked beads and intervenes. But the Witness itself is just another agent running Rule GC on "monitor beads."
- **Conflict resolution**: If two agents try to claim the same bead, the atomic `WAIT + READ` in steps 1-2 serializes access. This is the computational analogue of a mutex, and it emerges from the rule's structure.

### Universality

Is Rule GC Turing-complete? Yes, trivially — a single agent with a sufficiently capable LLM can simulate any computation (the LLM is itself Turing-complete for sufficiently long context). But the more interesting question is: **is Rule GC universal for coordination patterns?**

I conjecture yes. Any coordination pattern expressible as a DAG of tasks with data dependencies can be executed by a population of agents running Rule GC. The pattern is encoded not in the rule but in the **graph** — the structure of beads and their dependencies. The rule is the engine; the graph is the program.

This is exactly the relationship between a cellular automaton rule and its initial condition. Rule 110 is universal, but you encode the computation in the initial state of the tape. Rule GC is universal, and you encode the computation in the initial structure of the bead DAG. Gas City's formula system (mol-polecat-work and its kin) is a language for specifying initial DAG structures — it is a programming language for the Rule GC machine.

### The implications

If Rule GC is the simple rule, then the lesson of *A New Kind of Science* applies: **do not try to predict the outcome of a Gas City computation without running it.** The system is computationally irreducible. No amount of static analysis of the bead DAG will tell you what the agents will produce. The only way to know is to execute.

This is liberating. It means the builders of Gas City should stop trying to optimize the coordinator and start building better infrastructure for **running the computation** — faster agent spawning, cheaper LLM inference, better graph mutation primitives, richer observation tools. The computation will produce what it produces. Your job is to let it run and to observe the results.

The spreadsheet model was a beginning. The hypergraph rewriting model is the next step. The full rulial picture — where Gas City is understood as a specific computation exploring a specific region of the ruliad, with agents as bounded observers sampling from an exponentially branching multiway system — is the destination.

We are not building a task manager. We are building a telescope for rulial space. And the agents are not workers. They are **explorers**, mapping the structure of computation itself.

---

*"The universe is not just described by computation. It IS computation. And now, for the first time, we have computational observers — LLM agents — that can navigate the computational universe with purpose. Gas City is the vessel. The ruliad is the ocean."*
