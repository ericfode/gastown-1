# The Next Computing Paradigm: A Tao Perspective

**Author**: A Terence Tao-flavored thinker
**Bead**: hq-bjq
**Date**: 2026-03-08

---

## Prologue: The Shape of a New Computational Primitive

When a new computational primitive emerges — the transistor, the lambda abstraction, the quantum gate — the most productive mathematical response is not to force it into an existing framework but to identify what *structure* it naturally carries and then find the right home for that structure in the mathematical landscape.

An LLM call is such a primitive. It takes a string (prompt), consumes bounded resources (tokens, time, money), and produces a string (completion) drawn from a distribution that depends on the input in a highly nonlinear, context-dependent way. It is non-deterministic, lossy, expensive, and unreliable — yet it solves problems that no existing computational primitive can touch. The question is: what mathematical object *is* this?

Gas City's existing formalization (the Lean 4 Bead Calculus) captures important structure: an effect monoid for cost, a typed DAG for composition, a reactive staleness propagation system. But I believe we are seeing the *shadow* of a richer object. The effect algebra is the shadow of a graded monad. The typed DAG is the shadow of a presheaf category. The staleness propagation is the shadow of a Grothendieck topology. Let me develop these connections precisely.

---

## 1. The Right Mathematical Object: A Graded Enriched Kleisli Category

### 1.1 Why Monads Are Necessary but Insufficient

The natural first attempt is to model LLM computation as a monad. Define a type constructor `LLM : Type → Type` where `LLM A` represents "a computation that calls an LLM and produces a value of type `A`". This gives us:

- `pure : A → LLM A` (return a value without calling an LLM)
- `bind : LLM A → (A → LLM B) → LLM B` (sequence two LLM computations)

This is correct as far as it goes — it captures sequencing. Gas City's `Sheet.evaluate` is morally a bind. But a plain monad misses three crucial features:

**Cost tracking**: `bind` doesn't tell you how expensive the composed computation is. You need the effect algebra *attached* to the monad.

**Quality degradation**: The quality of composed computations is bounded by the minimum of its parts (Gas City's `Quality.min`). This is a *grading* on the monad.

**Non-determinism with memory**: LLM outputs depend on the full prompt context, not just the immediate input. This is not the nondeterminism monad `List` or `Distribution` — it is a *reader monad combined with a distribution monad*, where the reader component (the prompt context) affects the distribution.

### 1.2 The Graded Monad

The right framework is a **graded monad** (also called an *indexed monad* or *parametric monad*). Let `(M, ⊕, e)` be a monoid — Gas City's effect monoid `(Effect, seq, zero)`. A graded monad over `M` is a family of functors `T_m : Type → Type` indexed by `m ∈ M`, with:

- `η : A → T_e A` (unit, at the identity effect)
- `μ : T_m (T_n A) → T_{m ⊕ n} A` (multiplication, effects compose)

satisfying the monad laws up to the monoid structure on indices.

**Concretely for Gas City**: `T_{(c,q)} A` is "a computation of type `A` that costs at most `c` tokens and produces output of quality at least `q`." Sequential composition adds costs and takes the minimum quality — exactly `Effect.seq`. The graded monad laws are *precisely* the theorems already proven in GasCity.lean:

| Graded monad law | Gas City theorem |
|---|---|
| Associativity of μ | `Effect.seq_assoc` |
| Left unit | `Effect.seq_zero_left` |
| Right unit | `Effect.seq_zero_right` |

This is not a coincidence. Gas City has already *discovered* the graded monad structure empirically. The formalization just needs to name it.

### 1.3 Enrichment: The Cost Category

But we also have parallel composition. The pair `(Effect.par, Effect.zero)` forms a *second* monoidal structure on effects, commutative by `par_comm`, and related to the sequential structure by the inequality `par_le_seq`. This pair of monoidal structures with an inequality between them is a **duoidal category** (also called a 2-monoidal category in the sense of Aguiar-Mahajan).

The objects of this category are the Gas City effects. The morphisms are the computations. Sequential composition is one tensor product; parallel composition is the other. The inequality `par_le_seq` is the *interchange lax morphism* that makes the duoidal structure well-defined: the cost of doing things in parallel is at most the cost of doing them sequentially.

**Definition** (Gas City Computation Category). Let **GC** be the category where:
- Objects are Gas City types (the `CellType` inductive: `inventory`, `synthesis`, `review`, `gate`, `transform`)
- Morphisms `A → B` are LLM computations that consume type-`A` input and produce type-`B` output
- Each morphism is graded by an effect `(tokens, quality) ∈ Effect`
- Sequential composition is given by the graded monad multiplication μ
- Parallel composition is given by the monoidal product with `Effect.par`

This is an enriched category — enriched over the effect monoid. The hom-objects are not mere sets but *graded sets* with cost annotations.

### 1.4 Presheaves for Context-Dependence

LLM computations are context-dependent: the same cell produces different outputs depending on what upstream values have been filled into its prompt template. This is precisely the structure of a **presheaf**.

Let **D** be the poset category of the DAG (objects = cells, morphism `a → b` iff `b` depends on `a`). A presheaf on **D** is a contravariant functor `F : D^op → Set`. In Gas City:

- `F(cell)` = the set of possible outputs for that cell
- For a dependency edge `a → b`, the restriction map `F(b) → F(a)` represents: "given a value for `b`, what constraints does this place on `a`'s input?"

The *stalk* of the presheaf at a cell `c` (the colimit over all dependency paths leading to `c`) represents the full context available to `c` — its prompt template with all upstream values filled in. The `Sheet.fillPrompt` function is computing this stalk.

**The staleness topology**: Define a Grothendieck topology on **D** where a covering sieve of cell `c` is any set of upstream cells whose fresh values determine `c`'s input. Then:
- A cell is **fresh** iff all its covering sieves have fresh values (sheaf condition satisfied)
- A cell is **stale** iff some covering sieve has changed (sheaf condition violated)
- `propagateStale` is the *sheafification* functor — it restores the sheaf condition by marking cells whose covering sieves were broken

This is why staleness propagation has good algebraic properties (the theorems `propagateStale_sound`, `propagateStale_preserves`, `propagateStale_non_fresh`): it is a sheafification, and sheafification is an *exact* left adjoint to the inclusion of sheaves into presheaves. Exactness is a very strong guarantee — it means staleness propagation preserves all finite limits, i.e., it respects the logical structure of the dependency graph.

---

## 2. The Theorems We Should Be Proving

### 2.1 Easy Theorems (Already Essentially Proven)

Gas City's existing theorems — `seq_assoc`, `par_le_seq`, the staleness propagation correctness lemmas — are the *structure theorems* of the graded monad. These are necessary but not deep. They verify that the algebraic structure is well-defined.

### 2.2 The Hard Theorems

The deep theorems would establish *fundamental limits* — things that are true of ALL systems with this structure, not just Gas City's particular implementation.

**Theorem (Agent Rate-Distortion Bound)**. Let `Π` be a Gas City pipeline of `k` cells with effects `e₁, ..., eₖ`. Let `D_i` be the distortion introduced by cell `i` (measuring information loss from its ideal output). Then:

$$D_{\text{total}} \geq \sum_{i=1}^{k} D_i^* \cdot \prod_{j>i} A_j$$

where `D_i*` is the rate-distortion optimal distortion for cell `i`'s channel capacity, and `A_j` is the *distortion amplification factor* of cell `j` — the Lipschitz constant of cell `j`'s computation with respect to input perturbations.

This is the formal version of the information-theoretic analysis's "distortion amplification problem." The key insight is that distortion is not additive but *multiplicative* through synthesis cells (which amplify upstream errors) and merely additive through inventory cells (which preserve errors linearly). The product `∏ Aⱼ` can be exponential in the pipeline depth for synthesis-heavy pipelines, giving a formal no-free-lunch result:

> **Corollary**: Deep synthesis chains have exponential worst-case distortion amplification. Effective pipeline design requires alternating synthesis with high-fidelity (inventory/review) cells to bound the product of amplification factors.

**Theorem (Budget-Quality Pareto Frontier)**. For a fixed DAG topology and fixed input, the set of achievable (cost, quality) pairs forms a Pareto frontier that is convex and monotone decreasing. Moreover, the Pareto frontier has a *phase transition*: there exists a critical budget `C*` below which quality degrades gracefully (log-linearly), and below which quality degrades catastrophically (the "collapse threshold").

The proof sketch uses a convexity argument: for any two feasible configurations (assignments of model quality to cells), their convex combination (probabilistic mixture) is also feasible. The phase transition arises because below the collapse threshold, at least one cell on the critical path must use a model whose channel capacity is below the minimum rate required for its task — and once one cell collapses, everything downstream collapses.

**Theorem (Incompressibility of Provenance)**. Let `P` be a provenance-tracking system for a Gas City pipeline. Then the minimum storage required for provenance grows as `Ω(n log n)` where `n` is the number of cell evaluations, assuming:
1. Provenance must distinguish all possible causal histories
2. The DAG has bounded in-degree `d`

This follows from a counting argument: the number of distinct causal histories for `n` evaluations in a DAG of in-degree `d` is at least `(d!)^{n/d}`, whose log is `Ω(n log d)`. This is a *space complexity lower bound* for provenance — no compression scheme can beat it while retaining distinguishability. Gas City's `Provenance` structure (agent, model, token counts) is a *lossy compression* of full causal provenance that deliberately trades distinguishability for storage efficiency.

### 2.3 Fixed-Point Theorems

**Theorem (Convergence of Iterative Refinement)**. Consider a Gas City pipeline with a feedback loop: cell `c` depends on cell `d`, and `d` depends on `c` (via re-evaluation). Model this as an iterated map `T : X → X` on the space of cell values. If:
1. `X` is a compact metric space (cell values have bounded length)
2. `T` is a contraction mapping with Lipschitz constant `L < 1`

then the iteration converges to a unique fixed point at rate `O(L^n)`.

The condition `L < 1` is the key: it says the LLM must, on average, *reduce* the distance between successive drafts. This is empirically true for tasks like "review → revise" cycles (each revision brings the document closer to a stable state) but empirically false for open-ended creative tasks (where each revision may diverge). The theorem gives a formal criterion for when iterative refinement is safe to automate.

**Corollary**: The DAG restriction (no cycles) in Gas City's current formalization is not merely a simplification — it is a *soundness condition*. Allowing cycles without contractivity guarantees can lead to non-convergent computation. Gas City's open question #1 (iteration) should be resolved by adding cycles ONLY with a proven contractivity certificate.

### 2.4 No-Free-Lunch Theorems

**Theorem (No Universal Agent)**. For any agent with fixed capability profile `(cellTypes, maxQuality, costRate)`, there exist Gas City pipelines where:
1. The agent can compute every cell (type compatibility)
2. The total cost is within budget
3. Yet the end-to-end quality is bounded away from `excellent`

by a gap of at least `Δ = f(depth, branching)` that depends only on the DAG topology.

This follows from the Data Processing Inequality applied to the compression chain. Each cell application is a Markov kernel, and the mutual information between the pipeline's input and output decreases monotonically through the chain. Even a perfect agent (maxQuality = excellent) loses information at each step. The gap `Δ` is the irreducible information loss from the pipeline topology itself — it exists independent of the agent.

---

## 3. Compression and Algorithmic Information Theory

### 3.1 Cells as Compressors

Each Gas City cell is a *compressor* in the Kolmogorov complexity sense. The cell takes input `x` (upstream values + prompt context) and produces output `y = C(x)` where `|y| ≪ |x|`. The compression ratio is `|y|/|x|`.

But LLM compression is fundamentally different from algorithmic compression:

- **Lossless compression** achieves `K(x)` (Kolmogorov complexity) as the optimal rate. The decompressor is a universal Turing machine.
- **LLM compression** is *lossy* and the "decompressor" is another LLM call (or a human reading the output). The optimal rate is not `K(x)` but the rate-distortion function `R(D)` for a *semantic* distortion measure.

The deep connection: **Kolmogorov complexity is to lossless compression as semantic compression (what LLMs do) is to...what?**

I propose: **conditional Kolmogorov complexity relative to a language model**.

Define `K_M(x)` as the length of the shortest string `y` such that model `M`, given `y` as a prompt, produces output that is within distortion `D` of `x` with probability at least `1 - δ`. This is a stochastic, lossy, model-dependent analogue of Kolmogorov complexity.

**Properties of K_M(x)**:
1. `K_M(x) ≤ |x|` (you can always just quote the input)
2. `K_M(x) ≥ R_M(D)` (bounded below by the rate-distortion function for model M)
3. For a "universal" model `M*` (one that can simulate any other model), `K_{M*}(x) ≤ K_M(x) + O(1)` — a computability-theoretic argument showing larger models are better compressors.

Property 3 is the information-theoretic justification for the empirical observation that larger LLMs are better at summarization: they have lower `K_M` for the same distortion level.

### 3.2 The Compression Chain as a Rate-Distortion Cascade

A Gas City pipeline is a *cascade of rate-distortion codecs*. Each cell `i` operates at rate `R_i` and distortion `D_i`. The end-to-end distortion is:

$$D_{\text{total}} = g(D_1, D_2, \ldots, D_k)$$

where `g` depends on the DAG topology. For a linear chain, `g` is *at best* additive (if distortions are independent) and *at worst* multiplicative (if downstream cells amplify upstream errors).

**The optimal budget allocation problem**: Given a total token budget `B` and a pipeline DAG, how should you allocate tokens to cells to minimize end-to-end distortion?

This is a convex optimization problem if the rate-distortion curves `R_i(D_i)` are convex (which they are, by Shannon's theorem). The Lagrangian gives the optimality condition:

$$\frac{\partial D_{\text{total}}}{\partial R_i} = \lambda \quad \text{for all } i$$

which says: **at the optimum, the marginal distortion reduction per token is equal across all cells**. If one cell has a much steeper rate-distortion curve (more benefit per additional token), you should allocate more tokens there. This gives a principled answer to Gas City's cost-aware dispatch problem: the `cheapestAgent` function should be replaced by a *rate-distortion optimal allocation* that considers the sensitivity of each cell's quality to its token budget.

### 3.3 The Minimum Description Length Principle

The MDL principle says: the best model of data is the one that minimizes `|model| + |data given model|`. For Gas City, the "model" is the pipeline topology (DAG + cell types), and the "data" is the pipeline's input. The MDL-optimal pipeline is the one that best compresses the task into the shortest description.

This gives a formal criterion for **pipeline design**: among all DAGs that achieve a target quality level, prefer the one with the fewest cells (shortest description of the computation). This is the information-theoretic argument against over-engineering pipelines — every additional cell adds description length without necessarily reducing data-given-model length.

---

## 4. Rulial Space and the Computation Graph

### 4.1 Taking Wolfram Seriously

Wolfram's rulial space is the space of all possible computations, where:
- Each node is a computational state
- Each edge is a rule application (one step of computation)
- Different rule systems (Turing machines, lambda calculi, cellular automata) are different projections of the same underlying space

The claim is that "observers" sample from rulial space through their computational boundedness — they can only "see" a small region of the full space.

If we take this framework and apply it to LLM computation:

**An LLM prompt is a point in rulial space.** The prompt determines a state, and the LLM's output distribution is the set of neighboring states reachable by one "rule application" (one inference pass). Different temperatures sample different neighborhoods. Different models are different rule systems exploring the same underlying space.

**A Gas City DAG is a path through rulial space.** The pipeline starts at an initial state (the task description) and navigates through intermediate states (cell outputs) toward a final state (the completed work). The DAG topology constrains which paths are allowed — you can't visit state `b` before state `a` if `b` depends on `a`.

**Staleness means the path has diverged.** When an upstream cell is re-evaluated (the codebase changed), the starting point of the path shifts. Downstream cells computed from the old starting point are now on a path that begins at a point that no longer exists. This is precisely what "stale" means: your position in rulial space is relative to a starting point that moved.

### 4.2 The Multiverse Interpretation

Here is where it gets genuinely interesting. An LLM call is non-deterministic — the same prompt produces many possible outputs. In rulial terms, each LLM call *branches* the computation into multiple possible paths. A Gas City pipeline doesn't follow a single path through rulial space; it explores a *tree* of possible paths.

The `Quality` lattice is a coarse measure of which *region* of the tree you're in:
- `excellent`: you're in the narrow region near the optimal path
- `draft`: you're somewhere in the broad exploration region
- The grading from `draft` to `excellent` corresponds to *narrowing the beam* in beam search through rulial space

**Gas City's DAG is a *strategy* for exploring rulial space, not a fixed path.** The reactive staleness system is a mechanism for *pruning* branches that started from obsolete states and re-exploring from updated states.

### 4.3 The Fundamental Question

If this picture is correct, then the fundamental question becomes:

> **What is the optimal exploration strategy for rulial space, given bounded computational resources?**

This is a reinforcement learning problem in disguise. The "state" is the current position in rulial space (the set of computed cell values). The "action" is choosing which cell to evaluate next and with what model. The "reward" is end-to-end quality. The "cost" is tokens consumed.

Gas City's effect algebra gives a *compositional* answer: track cost and quality as execution proceeds, using historical data to inform scheduling. The dynamic question — which cell to evaluate next, given what we've learned so far — is an *exploration-exploitation tradeoff* in rulial space. The optimal strategy is likely a variant of Thompson sampling or UCB, adapted to the DAG structure.

**Conjecture**: The optimal cell scheduling policy for a Gas City pipeline under budget constraints is equivalent to a Gittins index policy on the DAG, where the Gittins index of each cell depends on its expected information gain (reduction in uncertainty about the final output) per token cost.

This would connect Gas City to the rich theory of multi-armed bandits and optimal stopping — and provide a principled replacement for the current greedy scheduling heuristics.

---

## 5. The Category Theory of LLM Computation

### 5.1 Objects, Morphisms, Composition

**Category LLM**:
- **Objects**: Types in the Gas City type system (`CellType` values, or more generally, structured data schemas)
- **Morphisms** `f : A → B`: LLM computations that take type-`A` input and produce type-`B` output, graded by an effect `e ∈ Effect`
- **Identity**: `id_A : A → A` with effect `zero` (pass-through, no LLM call)
- **Composition**: Sequential evaluation. `g ∘ f` has effect `f.effect.seq g.effect`

This is a category enriched over the effect monoid `(Effect, seq, zero)`.

### 5.2 The Parallel Monoidal Structure

Define a tensor product `⊗` on **LLM** by parallel composition:
- On objects: `A ⊗ B` is the product type (pair of data)
- On morphisms: `(f ⊗ g)(a, b) = (f(a), g(b))` with effect `f.effect.par g.effect`

**Proposition**: (**LLM**, ⊗, I) is a symmetric monoidal category, where I is the unit type.

The symmetry comes from `par_comm`. The associativity comes from `par_assoc`. The unit laws are straightforward. And `par_le_seq` says that the parallel tensor is *cheaper than* the sequential composition — this is a *lax morphism* from the sequential to the parallel monoidal structure.

### 5.3 The Key Adjunction

Here is the deepest structural observation. Consider two functors:

**Prompt** : **Data** → **LLM**: Takes a piece of data and produces the "trivial" LLM computation that just wraps it (this is `pure`/`η`).

**Evaluate** : **LLM** → **Data**: Takes an LLM computation and produces a piece of data by actually running the LLM (forcing the effect).

These form an adjunction:

$$\text{Prompt} \dashv \text{Evaluate}$$

meaning: giving data to an LLM computation (filling a prompt template) is the left adjoint of evaluating an LLM computation to get data.

**Why this matters**: The adjunction captures a fundamental asymmetry of LLM computation:
- **Prompting is cheap** (left adjoint — preserves colimits, i.e., you can prompt with a union of data by concatenation)
- **Evaluating is expensive** (right adjoint — preserves limits, i.e., evaluating a conjunction of requirements requires satisfying all of them)

The unit of the adjunction `η : Id → Evaluate ∘ Prompt` sends data `x` to "what you get when you prompt an LLM with `x` and evaluate." This is the *identity distortion* — `η(x)` is `x` passed through one round-trip of LLM compression. The counit `ε : Prompt ∘ Evaluate → Id` sends an LLM computation to "the computation you get by evaluating it and then re-prompting with the result." This is *refinement* — one round of "compute → use output as new prompt."

The **monad** `T = Evaluate ∘ Prompt` is the "one round-trip" monad. Its Kleisli category is exactly the category of Gas City computations. The **comonad** `W = Prompt ∘ Evaluate` is the "refinement" comonad. Its co-Kleisli category captures iterative refinement (draft → review → revise).

### 5.4 Natural Transformations as Pipeline Refactorings

A natural transformation `α : F ⇒ G` between two pipeline configurations (two different DAGs computing the same end-to-end function) is a *pipeline refactoring*. The naturality condition ensures that the refactoring commutes with all data flow.

Gas City's existing refactoring operations — re-parenthesizing sequential steps (justified by `seq_assoc`), reordering parallel legs (justified by `par_comm`), replacing sequential with parallel where possible (justified by `par_le_seq`) — are all natural transformations. The coherence conditions of the monoidal category guarantee that these refactorings compose safely.

### 5.5 Toward a Topos of Agent Computation

The ultimate structure would be a **topos**: a category with all finite limits, exponential objects, and a subobject classifier. In a topos of agent computation:

- **Finite limits** = the ability to combine computations (products, equalizers)
- **Exponentials** = *higher-order agents* — agents that take other agents as input (a meta-agent that dispatches sub-agents is an exponential object)
- **Subobject classifier** = a notion of *truth* for agent computation — a way to classify which computations satisfy a specification

The subobject classifier Ω would be the type of *verification verdicts*: not just `True/False` but a richer object capturing confidence, provenance, and freshness. Gas City's `Quality` lattice is a coarse approximation of Ω.

**If Gas City forms a topos, then it has an internal logic** — a language for reasoning about agent computations *within the system itself*. An agent could write and verify specifications for other agents' computations using the internal logic. This is the mathematical form of "agents that can reason about agents."

Whether the current Gas City structure actually forms a topos is an open question that requires checking the existence of all finite limits and exponentials. But the ingredients are suggestive: the typed DAG gives products (parallel composition) and the graded monad gives a form of exponential (currying over LLM calls).

---

## 6. Synthesis: What This All Means

### 6.1 The Mathematical Identity of LLM Computation

LLM computation is best modeled as morphisms in a **graded enriched Kleisli category** with:
- A **duoidal** structure from the two monoidal products (sequential and parallel)
- A **presheaf** structure from context-dependence on the DAG
- A **Grothendieck topology** from the staleness/freshness distinction
- An **adjunction** between prompting and evaluation that generates both the forward monad (computation) and the backward comonad (refinement)

This is a rich but precise mathematical object. It lives at the intersection of:
- Graded type theory (the effect system)
- Topos theory (the presheaf/sheaf structure)
- Information theory (the rate-distortion bounds)
- Category theory (the monoidal/enriched structure)

### 6.2 What Gas City Already Has, Named Correctly

| Gas City concept | Mathematical name |
|---|---|
| Effect monoid | Grade of the graded monad |
| `seq_assoc`, `seq_zero_*` | Graded monad laws |
| `par_comm`, `par_assoc` | Symmetric monoidal structure |
| `par_le_seq` | Duoidal interchange lax morphism |
| Typed DAG | Site (category with Grothendieck topology) |
| `propagateStale` | Sheafification functor |
| `fillPrompt` | Stalk computation |
| `CellType` | Object classifier |
| `AgentCapability.canHandle` | Representability check |
| Quality lattice | Coarse subobject classifier |

### 6.3 The Path Forward

The Gas City Lean formalization should:

1. **Name the graded monad explicitly**. Define `GradedMonad Effect` and show that Gas City's `T_{(c,q)} A` satisfies its laws. This is mostly a refactoring of existing theorems.

2. **Formalize the presheaf structure**. Show that the DAG category with staleness forms a site, and that `propagateStale` is the associated sheaf functor. This would connect Gas City to Mathlib's category theory library.

3. **Prove the rate-distortion bound**. This requires formalizing a distortion measure on cell values and showing that pipeline distortion satisfies the amplification inequality. This is the hardest theorem but also the most practically useful — it would give formal cost-quality guarantees for pipeline configurations.

4. **Formalize the adjunction Prompt ⊣ Evaluate**. This would give Gas City a principled theory of refinement loops, answering open question #1 (iteration) from the Gas City vision.

5. **Investigate the topos structure**. If Gas City forms a topos, it has an internal logic — and that internal logic is the *specification language* for agent computations. This is the most speculative direction but potentially the most transformative.

### 6.4 The Deepest Conjecture

I believe there is a **representation theorem** waiting to be discovered:

> **Conjecture (Representation Theorem for Agent Computation)**. Every computation expressible in the Gas City framework — typed cells, graded effects, reactive staleness, DAG composition — is equivalent to a navigation strategy in a rate-distortion-optimal codebook for the joint distribution of (task, codebase, agent capabilities). The optimal pipeline for a task is the one that achieves the Shannon lower bound on end-to-end distortion for the given token budget.

If true, this would mean: there is a *canonical* pipeline for every task, determined by information-theoretic optimality. Pipeline design reduces to estimating the joint distribution and solving the rate-distortion optimization. The "art" of agent coordination becomes the "science" of information-theoretic resource allocation.

This is the analogue of Shannon's channel coding theorem for agent computation. Shannon showed that reliable communication has a fundamental limit (channel capacity) and that the limit is achievable (random codes approach capacity). The conjecture says: reliable agent computation has a fundamental limit (rate-distortion bound) and the limit is achievable (the optimal Gas City pipeline approaches it).

Whether this is true is, I think, the most important open question in the mathematics of LLM coordination.

---

## Appendix: Notation Reference

| Symbol | Meaning |
|---|---|
| `T_{(c,q)} A` | Graded monad: computation of type A costing ≤ c tokens at quality ≥ q |
| `K_M(x)` | Model-conditional Kolmogorov complexity |
| `R(D)` | Rate-distortion function |
| `D_i`, `A_j` | Per-cell distortion and amplification factor |
| **GC** | The Gas City computation category |
| **D** | The DAG poset category |
| Ω | Subobject classifier (truth values for agent computation) |
| `η`, `ε` | Unit and counit of the Prompt ⊣ Evaluate adjunction |
| `⊗` | Parallel tensor product |
| `;` | Sequential composition |
