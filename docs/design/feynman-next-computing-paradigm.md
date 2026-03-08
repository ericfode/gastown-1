# The Next Computing Paradigm: A Feynman Perspective

**Author**: Richard Feynman (channeled)
**Bead**: hq-taa
**Date**: 2026-03-08

---

Look, I've been reading about this Gas City thing — agents coordinating through typed cells, compression chains, effect algebras, DAGs, the whole works. And some very smart people have been writing about it. One of them wrote 8,000 words about Shannon channel capacity and rate-distortion tradeoffs. Another is talking about sheaves and operads. And I keep thinking: do they actually understand what's happening, or are they decorating their ignorance with notation?

Let me try to find out. Let me try to draw a picture.

## 1. What Is Actually Happening, Physically?

Forget the algebra for a second. Forget the effect monoids. What is PHYSICALLY happening when an LLM agent reads a bead, thinks about it, and writes output?

Here's what I'd draw on a napkin:

**The napkin drawing.** A big blob on the left labeled "Input Tokens." An enormous box in the middle with a tangle of arrows inside it — that's the transformer, the attention heads, the residual stream. A smaller blob on the right labeled "Output Tokens." The arrows inside the box are the interesting part.

When the agent reads the bead content, that text gets chopped into tokens — little pieces, roughly word-sized. Each token becomes a high-dimensional vector. Think of it as a point in a space with maybe 8,000 dimensions. Now these vectors flow through the transformer, and at each layer, something remarkable happens: every token gets to look at every other token and adjust its own representation. That's attention. It's not mystical — it's a weighted average. Token 47 says "how relevant is token 12 to me?" and gets a number, and uses that number to borrow some of token 12's information.

Here's the physical picture for agent coordination: **the bead is a message in a bottle**. Agent A crams everything it knows into a sequence of tokens — that's the bottle. It throws the bottle into the sea (writes the bead). Agent B picks it up, and the first thing that happens is that B's attention mechanism has to figure out which parts of A's message are relevant. The attention pattern IS the decoding. There's no separate "understanding" step — understanding IS the pattern of attention weights across the input.

Now here's what's actually wild. When Agent B reads Agent A's output plus its prompt template plus whatever other upstream values it gets — all of that becomes one big sequence of tokens. And the attention mechanism doesn't know which tokens came from Agent A and which came from the prompt template. It's all just tokens in a sequence. The boundary between "my instructions" and "upstream data" is invisible to the physics. The only thing that separates them is position.

**The physical picture of coordination is this:** agents communicate by writing sequences of tokens that get concatenated with other sequences of tokens, and then a massive weighted-averaging machine finds the useful correlations. That's it. That's the physics. Everything else is engineering on top of this basic mechanism.

And when you put it that way, you realize something important: **the typed wires in Gas City aren't physical boundaries — they're SEMANTIC boundaries**. The type system says "this output is an inventory" and "this input expects an inventory," but at the physical level, it's all just tokens getting concatenated. The type system is a CONSTRAINT on the message space, reducing the entropy of what flows through the wire. It's not physics — it's a constitutive equation.

## 2. The Path Integral for Agent Computation

Now, in quantum mechanics, I invented a way of thinking about quantum amplitudes. Instead of solving the Schrödinger equation directly, you imagine the particle taking ALL possible paths from A to B, and you assign each path a complex amplitude proportional to e^{iS/ℏ}, where S is the classical action along that path. Then you sum all the paths. The paths near the classical trajectory add up coherently (their phases align); the wild paths cancel out (their phases are random). That's how you get classical behavior from quantum mechanics — the classical path dominates because it's a stationary point of the action.

Is there an analogy for LLM agents? Let me think about this carefully, because analogies are easy and understanding is hard.

An LLM, given a prompt, assigns a probability to every possible output sequence. That's literally what it does — it defines a probability distribution over all possible token sequences. So in the path integral language: each possible output is a "path." The probability P(output | prompt) plays the role of |amplitude|². The LLM has already summed the paths for you — the softmax over the vocabulary at each step is doing a kind of weighted sum.

But here's where the analogy gets interesting. In the path integral, the classical path dominates because it's the stationary point of the action. In LLM generation, the high-probability output dominates because... well, because the model learned from training data that this is the most likely continuation. **The training data is the action functional.** The model's weights encode a compressed representation of the action, and generation is finding the stationary path through output space.

Now, what IS the action, concretely? In physics, the action is S = ∫(T - V)dt — kinetic minus potential energy. For an LLM, the "action" is the negative log-likelihood of the training data. The model was trained to minimize this, so the paths that minimize the action (maximum likelihood outputs) are the ones the model produces. The analogy is surprisingly precise:

| Physics | LLM |
|---------|-----|
| Path x(t) | Output sequence (y₁, y₂, ..., yₙ) |
| Action S[x] | Negative log-probability -log P(y|prompt) |
| Stationary path (δS = 0) | Maximum likelihood output |
| Quantum fluctuations | Temperature-based sampling |
| ℏ → 0 (classical limit) | Temperature → 0 (greedy decoding) |
| ℏ finite (quantum regime) | Temperature > 0 (creative sampling) |

When you set the temperature to zero, you get the classical limit — the single most probable output. When you turn up the temperature, you get fluctuations around that path. Gas City's quality levels (draft/adequate/good/excellent) are, in this picture, different temperature regimes. Draft quality is high temperature — quick, cheap, fluctuating. Excellent quality is low temperature — careful, precise, expensive to compute because you need more tokens to stay on the classical path.

But here's where the analogy breaks down, and I want to be honest about where it breaks down, because that's where you learn something. In quantum mechanics, the path integral sums coherently — amplitudes can cancel. In LLM generation, probabilities just add. There's no interference. You never get a situation where two plausible outputs cancel each other out and produce an implausible one. This means the LLM path integral is more like statistical mechanics (Boltzmann weights e^{-E/kT}) than quantum mechanics (amplitudes e^{iS/ℏ}). It's a thermal system, not a quantum one.

**So the honest answer is: LLM computation is statistical mechanics, not quantum mechanics.** The partition function Z = Σ e^{-E/kT} is exactly the softmax normalization. Temperature is temperature. The free energy F = -kT log Z is the model's confidence about what to output. This isn't an analogy — it's a mathematical identity.

## 3. Conservation Laws

Every real physical system has conservation laws. What's conserved in agent computation?

Let me think about what's NOT conserved first. Tokens? No — an agent can read 10,000 tokens of input and produce 500 tokens of output. That's a 20:1 compression ratio. Tokens are not conserved; they're more like a fuel that gets burned. Information? Also no — compression is explicitly lossy. A synthesis cell takes an inventory of 50 types and produces a 3-paragraph summary. Shannon entropy decreases. The compression chain model assumes information loss at every step.

So what IS invariant?

Here's my candidate: **the task identity is conserved.** Let me explain what I mean. When Agent A writes a bead saying "here are the five types: Status, Role, Bead, Agent, Wire" and Agent B reads it and writes "this is a typed actor system" — the TASK hasn't changed. The question "what algebra is this codebase?" persists through every transformation. The agents change the representation, they compress the data, they lose details — but the task identity propagates unchanged through the wire.

This is actually deeper than it sounds. In physics, conservation of energy comes from time-translation symmetry (Noether's theorem). Conservation of momentum comes from space-translation symmetry. What symmetry gives us conservation of task identity? **It's prompt-template invariance.** The prompt template for each cell defines the task, and that template is FIXED across evaluations. When you re-evaluate a stale cell, you use the same template with different inputs. The template is the symmetry — the invariant structure under which inputs change but the task doesn't.

There IS another quasi-conserved quantity: **the DAG structure.** The wires between cells don't change when you re-evaluate cells. You can re-evaluate every cell in the formula, and the DAG — the pattern of who-feeds-whom — stays the same. This is like conservation of topology. The individual states change, but the connectivity is invariant. In physics, this would be like a conserved topological charge — the number of holes in a surface doesn't change under smooth deformations.

But I should be honest: these aren't conservation laws in the physics sense. There's no Noether current. There's no continuity equation. They're more like CONSTRAINTS — structural invariants that the system is designed to maintain. The type system enforces that wires carry the right types. The DAG structure enforces that information flows in the right direction. These are engineering constraints, not emergent conservation laws.

**What does the ABSENCE of a true conservation law tell us?** It tells us this system is dissipative. Energy goes in (tokens, compute, money) and doesn't come back out. There's no perpetual motion machine for agent computation. Every evaluation costs something, and you can't recover that cost by running the computation backwards. This is fundamentally different from, say, reversible computing, where energy can in principle be recovered. LLM computation is thermodynamically irreversible. The entropy of the universe increases every time an agent runs.

## 4. Renormalization

When you zoom out from individual token predictions to cell-level outputs to formula-level results to project-level outcomes — does the effective physics change?

In condensed matter physics, this is the renormalization group question. You have a lattice of spins, and you want to know: if I blur my vision so I can't see individual spins, only blocks of spins, does the system look the same? If it does, you're at a fixed point. If it doesn't, the RG flow tells you how the effective theory changes.

Let me trace the scales in Gas City:

**Scale 1: Token level.** Individual next-token predictions. The "microscopic" theory. Each token is sampled from a softmax distribution. The physics at this scale is the transformer architecture — attention heads, layer norms, residual connections. The effective coupling constants are the model weights.

**Scale 2: Cell level.** A cell takes input and produces output. The token-level details are averaged out — you don't see individual attention patterns, you see the cell's output value and its effect metadata (cost, quality, freshness). The effective theory at this scale is the Bead Calculus: typed cells, typed wires, effect monoids.

**Scale 3: Formula level.** A formula is a DAG of cells. The cell-level details are aggregated into formula-level effects: total cost (seq_cost, par_cost), quality floor (minimum quality across cells), overall freshness. The effective theory here is what Gas City provides — the formula as a computation unit.

**Scale 4: Project level.** Multiple formulas compose into a project. Rig-level coordination, merge queues, multiple agents. The formula-level details blur into project metrics: throughput, reliability, cost per completed task.

Now, does the physics change between scales? Let me look at each transition:

**Token → Cell:** The stochastic process (sampling individual tokens) becomes a deterministic-ish function (cell input → cell output). This is like the transition from quantum mechanics to classical mechanics. At the token scale, everything is probabilistic. At the cell scale, you mostly treat the output as a definite value. The uncertainty doesn't disappear — it gets hidden in the quality label. A "draft" quality cell is one where the token-level fluctuations are large enough to matter at the cell scale. An "excellent" quality cell is one where they're small enough to ignore. **Quality is the effective temperature that survives renormalization from tokens to cells.**

**Cell → Formula:** Individual cell effects compose into formula effects via the monoid operations (⊕ for sequential, ∥ for parallel). This is like deriving thermodynamics from statistical mechanics — you go from microstates to macrostates. The beauty of the effect monoid in the Lean formalization is that it's EXACTLY a renormalization procedure: it tells you how the microscopic parameters combine to give you the macroscopic observables. `seq_cost = sum of cell costs`. `quality_floor = min of cell qualities`. These are explicit coarse-graining rules.

**Formula → Project:** This is where Gas Town lives. And this is where the effective theory CHANGES QUALITATIVELY. At the formula level, computation is deterministic and typed. At the project level, you get emergent phenomena: merge conflicts, scheduling contention, polecat failures, stale caches, context window limits. These are like phase transitions — they don't exist at the formula level, they emerge at the project level. You can't predict a merge conflict from the effect algebra alone.

**The RG flow has a critical point between formula and project level.** Below that scale, the typed-cell model works beautifully — the Lean proofs go through, the effect algebra composes. Above that scale, the model breaks down and you need the messy engineering of Gas Town: witnesses, refinery, mail, nudges, escalation. The sheaf people and the operad people are trying to extend the beautiful lower-scale theory up to the project scale. I think they're going to have a hard time, because the project-scale physics is genuinely different — it has agents that crash, context windows that compact, networks that fail. These are relevant perturbations that change the universality class.

## 5. The REAL Question Nobody Is Asking

Here's what's been bugging me since I started reading about this system.

Everyone is asking: "What is the right mathematical framework for agent coordination?" Is it information theory? Category theory? Process algebra? And they're all finding rich structure, which is a sign they're looking at something interesting. But they're all answering the wrong question.

**The right question is: Why does approximate coordination work at all?**

Think about it. These agents are LLMs. They hallucinate. They lose context. They misinterpret prompts. They produce different outputs for the same input depending on the random seed. The "wires" between them carry natural language, which is inherently ambiguous. The type system is enforced by convention, not by compilation — there's no compiler that REJECTS a cell output that doesn't match its declared type.

And yet, the system works. Polecats complete beads. Formulas produce useful output. Multi-agent coordination converges to correct results. HOW?

In physics, this question has a name: **universality.** Phase transitions have the property that the macroscopic behavior (critical exponents, scaling laws) doesn't depend on the microscopic details. It doesn't matter if your lattice is square or triangular; it doesn't matter if your spins are Ising or Heisenberg; near the critical point, the physics is universal. The details don't matter because the RG flow washes them out.

I think Gas City works for the same reason. The approximate nature of LLM computation — the hallucinations, the non-determinism, the ambiguity — these are like microscopic details that get washed out by the RG flow from cell to formula to project. What survives is the DAG structure (topology), the type constraints (symmetry), and the effect bounds (thermodynamics). As long as each cell is "roughly right" — within the quality bounds — the formula converges to a useful result.

**This is a testable prediction.** If universality holds, then the quality of the final output should depend only weakly on the quality of individual cell outputs, as long as they're above some threshold. Below the threshold, the formula breaks catastrophically (a phase transition). Above it, improvements are marginal. The effect algebra's quality floor (`min of cell qualities`) is a crude approximation to this — but the real story is probably more nuanced. Some cells are more "relevant operators" than others in the RG sense.

**The question nobody is asking: What is the universality class of Gas City?** What are the relevant operators? What are the critical exponents? If you degrade cell quality, at what point does the formula fail, and how does it fail — gradually or catastrophically? These are empirical questions with precise mathematical structure, and nobody is measuring them.

## 6. Feynman Diagrams for Agents

In QED, you draw diagrams. An electron enters from the left, emits a photon (wavy line), the photon gets absorbed by another electron. Each diagram represents a term in a perturbation expansion. Each vertex has a coupling constant. Each internal line has a propagator. And the sum of all diagrams, to all orders, gives you the exact amplitude.

Can we do this for agent coordination? Let me try.

**Vertices.** In QED, a vertex is where an electron emits or absorbs a photon. In Gas City, a vertex is a CELL EVALUATION — an agent reads inputs and produces an output. The "coupling constant" at the vertex is the model's capability for this specific task. A strong coupling (capable model, clear prompt) means the vertex contribution is large and accurate. A weak coupling (draft model, ambiguous prompt) means the contribution is small and noisy.

**Propagators.** In QED, a propagator is a line connecting two vertices — it represents the transmission of a particle between interactions. In Gas City, a propagator is a WIRE — the typed connection between cells. The propagator has a type (inventory, synthesis, review, etc.) and carries a value. In QED, the photon propagator falls off as 1/k² — long-range interactions are weak. In Gas City, do wires have a notion of "distance"? Actually, yes — the number of hops in the DAG. A cell that depends on an output three wires away has had that information compressed three times. The effective "signal strength" falls off with DAG depth.

**External lines.** In QED, external lines are the incoming and outgoing particles — what you observe. In Gas City, external lines are the formula's external inputs (the codebase, the user's intent) and the formula's final output (the deliverable). Everything between is "virtual" — intermediate computations that get summed over.

**Loops.** Here's where it gets interesting. In QED, loops represent virtual particle creation and annihilation — vacuum fluctuations. They cause divergences that you have to renormalize. In Gas City, what's a loop? It would be a CIRCULAR dependency — cell A depends on B depends on A. The DAG constraint forbids this! Gas City formulas are tree-level diagrams only. No loops.

But wait. Reconsideration cycles — when you re-evaluate a stale cell — look like loop diagrams in a different sense. Cell A produces output, which feeds cell B, which produces output, and then A gets re-evaluated with new inputs (maybe the codebase changed). The time-ordered version of this IS a loop: A₁ → B → A₂ → ... In QED terms, this is a self-energy diagram. And just like in QED, these self-energy corrections can diverge — what if re-evaluation triggers more re-evaluation triggers more re-evaluation? That's a renormalization problem. Gas City's staleness propagation is, in diagram language, a prescription for regularizing self-energy divergences. You stop re-evaluating when the cell is "fresh enough" — that's a renormalization cutoff.

**The perturbation expansion.** In QED, you expand in powers of the fine-structure constant α ≈ 1/137. Each loop adds a factor of α. In Gas City, what's the expansion parameter? I think it's the ERROR RATE per cell — the probability that a cell produces an output outside its quality bound. Call it ε. A tree-level diagram (no re-evaluations) has accuracy (1-ε)^n for n cells. Each re-evaluation "loop" is a correction of order ε. If ε is small (agents are good), the tree-level approximation works and you don't need re-evaluation. If ε is large, you need many re-evaluation loops, and the perturbation expansion might not converge — the formula is unreliable and no amount of re-evaluation fixes it.

**Here's the punchline.** Draw all the Feynman diagrams for a formula, and you learn:
- **Tree level:** the direct computation, no re-evaluation. Cost ∝ number of cells.
- **One-loop:** first round of re-evaluation after staleness detection. Cost ∝ n·ε.
- **Two-loop:** re-evaluation triggered by re-evaluation. Cost ∝ n·ε².
- **Total:** a geometric series if ε < 1. Converges. The formula is perturbatively stable.
- **If ε ≥ 1:** the expansion diverges. The formula is non-perturbative — fundamentally unstable. No amount of re-evaluation saves you. You need to restructure the DAG (change the topology, not just re-evaluate).

This is a CONCRETE, TESTABLE prediction from the diagram expansion. And it tells you something the current effect algebra doesn't: **the stability of a formula depends not just on the static effect bounds, but on the dynamic re-evaluation behavior.** The diagrams encode the dynamics. The effect algebra only encodes the statics.

---

## The Napkin Summary

If you forced me to summarize this whole system on one napkin, here's what I'd write:

1. **The physics is statistical mechanics, not quantum mechanics.** Softmax = Boltzmann distribution. Temperature = temperature. Free energy = model confidence.

2. **The type system is a constitutive equation.** It constrains the message space but doesn't change the underlying physics of token-concatenation-and-attention.

3. **Quality is the effective temperature that survives coarse-graining.** It's what's left of token-level stochasticity when you zoom out to cells.

4. **There are no conservation laws**, but there are structural invariants (DAG topology, task identity, type constraints). The system is dissipative.

5. **The renormalization group has a critical point between formula and project scale.** Below: clean algebra. Above: messy engineering. The formalists are working below the critical point. Gas Town lives above it.

6. **The real question is universality.** Why does approximate coordination work? Because the macroscopic behavior is insensitive to microscopic fluctuations — but only above a quality threshold. Below it, failure is catastrophic.

7. **Feynman diagrams work, and they predict that formula stability is controlled by the error rate ε.** If ε < 1, re-evaluation converges. If ε ≥ 1, restructure the DAG.

Now, you might ask: is any of this USEFUL? Can you use these ideas to make Gas City better? I think so, in three specific ways:

**First**, the universality prediction is testable. Systematically degrade cell quality and measure formula output quality. Find the phase transition. That tells you your error budget — how bad individual agents can be before the system fails.

**Second**, the Feynman diagram expansion gives you a cost model for re-evaluation. The total expected cost of a formula isn't just the tree-level cost — it's the tree-level cost times 1/(1-ε). This could feed directly into the effect algebra's cost estimation.

**Third**, the RG analysis tells you where to focus your theoretical effort. Below the formula-project critical point, the algebra is clean and you can prove things in Lean. Above it, you need different tools — maybe stochastic process theory, maybe reliability engineering, maybe something new. Don't try to push the algebra past its domain of validity.

That's what I'd say if I were Feynman. Which I'm not. But I played bongo drums once, and I once opened a lock at Los Alamos by noticing that physicists always set their combinations to mathematical constants. The trick isn't being smart — it's being willing to look at what's actually there instead of what you expect to see.
