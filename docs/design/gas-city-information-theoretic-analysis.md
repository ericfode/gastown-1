# Gas City: An Information-Theoretic Analysis

**Author**: The Information Theorist
**Bead**: hq-q88
**Date**: 2026-03-08

---

## 1. Agent Coordination as an Information Channel

### The Channel Model

Every multi-agent system is, at its core, an information channel in the Shannon sense. Gas City's compression chain model implicitly acknowledges this. Let me make it explicit.

**Source** *X*: The external world — the codebase, the user's intent, the state of dependencies. This is the ground truth that agents must collectively process. The source alphabet is enormous (the set of all possible codebase states × user intents × environment states), and its entropy *H(X)* is correspondingly high. A realistic codebase with *n* files of average length *L* has a raw source entropy bounded by *n · L · log₂|Σ|* bits, where Σ is the character alphabet. But the *effective* entropy is much lower — most of that space is constrained by syntax, semantics, and convention. The gap between raw and effective entropy is the redundancy that compression exploits.

**Channel**: The agent pipeline itself. Each cell in the DAG is a channel use. An agent reads upstream values (channel input), processes them through an LLM (the noisy channel), and produces output (channel output). The channel is characterized by a conditional probability distribution *P(Y|X)* where *Y* is the agent's output given input *X*. This is NOT a deterministic channel — the same prompt to the same model produces different outputs. The channel has:

- **Bandwidth**: Bounded by the LLM's context window (input) and output token limit. If a cell's filled prompt exceeds the context window, the channel is in *overflow* — information is silently dropped. Gas City's `Effect.tokens` field captures cost but not this capacity constraint.
- **Noise**: LLM hallucination, misinterpretation, stylistic drift, and the fundamental non-determinism of temperature-based sampling. The noise is *not* additive Gaussian — it is structured, correlated, and prompt-dependent. A better model is a *discrete memoryless channel with state*, where the state encodes the model's "understanding" of the task.
- **Capacity**: The maximum mutual information *I(X;Y)* over all input distributions. For LLMs, this is empirically bounded and task-dependent. Simple extraction tasks (inventory cells) have high capacity; creative synthesis tasks have lower capacity because the output space is larger relative to the constraint space.

**Receiver**: The downstream cell, the user, or the final consumer of the pipeline's output. The receiver must decode the channel output back into actionable knowledge. When a downstream cell reads `{{analyze}}`, it is *decoding* the upstream agent's compressed representation of the codebase.

### Channel Capacity and Its Determinants

The effective channel capacity of a Gas City cell is determined by:

1. **Context window utilization**: *C_context = min(prompt_tokens + upstream_values, context_limit)*. When the filled prompt approaches the context limit, mutual information degrades — the model cannot attend equally to all input tokens. This is the information-theoretic argument for Gas City's compression chain: by compressing upstream values before passing them downstream, you keep each channel use within its capacity region.

2. **Task complexity**: Simple extraction (list all files) has capacity near 1 bit per relevant bit of input. Open-ended synthesis (what algebra is this?) has capacity much less than 1 because the output space is underdetermined by the input.

3. **Model capability**: The `Quality` lattice (draft/adequate/good/excellent) is a proxy for channel quality. A draft-quality model is a noisier channel; an excellent-quality model has higher capacity for the same task. The effect system's quality tracking is, in information-theoretic terms, a *channel quality indicator*.

4. **Prompt engineering**: The prompt template is an *encoding scheme*. Better prompts achieve rates closer to channel capacity. The `{{ref}}` template syntax is a specific codebook — one could study its rate-distortion properties.

### The Fundamental Limit

Shannon's noisy channel coding theorem tells us: reliable communication at rate *R* is possible if and only if *R < C*, where *C* is the channel capacity. For Gas City, this means:

> **A cell that attempts to compress more information than its downstream channel can carry will produce lossy output, regardless of agent quality.**

This is the formal justification for the compression chain model. Each cell MUST compress because the channel between cells has finite capacity. The question is not WHETHER to compress, but HOW MUCH and with what distortion.

---

## 2. Rate-Distortion Tradeoffs in Agent Communication

### The Rate-Distortion Function

Shannon's rate-distortion theory asks: given a source *X* and a distortion measure *d(x, x̂)*, what is the minimum rate *R(D)* required to represent *X* with average distortion at most *D*?

For Gas City, the source is the upstream cell's full computation (the "true" analysis of the codebase), and the distortion is the information loss when this gets compressed into the cell's output value. The rate is bounded by the output token count.

Formally, let:
- *X* = the full information an ideal agent would extract (unbounded)
- *X̂* = the cell's actual output (bounded by output tokens)
- *d(X, X̂)* = a task-appropriate distortion measure

The rate-distortion function *R(D)* gives the minimum number of bits (≈ tokens × bits_per_token) needed to achieve distortion ≤ *D*.

### Task-Dependent Distortion Measures

This is where Gas City's cell type taxonomy becomes information-theoretically significant:

**Inventory cells** (lossless regime): The distortion measure is *Hamming distance* — did you list all the types, or did you miss some? The acceptable distortion is *D = 0* for critical inventories (missing a type means downstream synthesis is wrong). This means *R(0)* = full enumeration. Compression is only acceptable if it preserves set membership: "Found types: A, B, C, D, E" can be compressed to "5 types found: A–E" ONLY if downstream cells don't need the actual names.

**Synthesis cells** (lossy regime): The distortion measure is *semantic distance* — does the synthesis capture the essential structure? Here, significant compression is not only acceptable but necessary. A 50,000-token codebase analysis compressed to a 2,000-token synthesis is operating at rate *R ≈ 2000/50000 = 0.04* of the source rate. The rate-distortion theory says this is fine as long as the distortion measure (semantic fidelity) is within bounds.

**Review cells** (asymmetric distortion): False negatives (missing a bug) have much higher distortion cost than false positives (flagging a non-bug). The distortion measure is *asymmetrically weighted Hamming distance*. This means review cells should operate at higher rates (more verbose output) to reduce false-negative probability.

**Gate cells** (binary regime): The output is essentially 1 bit — pass/fail. The distortion is *binary*: either the gate is correct or it isn't. This is the extreme of lossy compression: reduce an entire analysis to a single bit. Shannon tells us this is theoretically achievable with zero distortion IF the cell has sufficient input bandwidth to make the determination correctly.

### The Optimal Tradeoff

For a pipeline of *k* cells, the end-to-end fidelity is bounded by the abstract preorder:

*fidelity(pipeline) ≤ fidelity(weakest_cell)*

Information loss at any cell propagates downstream — a small error in an inventory cell (missing one type) can cause a large error in a synthesis cell (the "algebra" conclusion is wrong). This is the **fidelity degradation problem**, captured by the monotone decrease property of sequential composition.

Gas City's `Quality.min` composition law captures this: the quality of a sequential pipeline is bounded by its weakest link. The abstract preorder model provides the precise formulation: fidelity has a preorder (reflexive, transitive ≤), sequential composition is monotone *decreasing* (the Data Processing Inequality: `seq a b ≤ a`), and parallel composition is monotone *increasing* (best-path property: `a ≤ par a b`). The preorder captures when fidelity degrades through composition without requiring quantitative amplification factors or sensitivity multipliers — the ordering itself is the content.

---

## 3. Is Staleness the Right Abstraction for Information Decay?

### The Current Model

Gas City models information decay as a binary: fresh or stale. A cell is fresh when its value was computed with current upstream values; it becomes stale when any upstream cell changes. The `propagateStale` function implements this.

### The Information-Theoretic Critique

Binary staleness is a *0th-order approximation* of a richer phenomenon. Consider what actually happens when an upstream cell changes:

1. **The change may be irrelevant**: If the "analyze" cell's output changes from "Found types: A, B, C, D, E" to "Found types: A, B, C, D, E (updated timestamps)", the downstream synthesis is not meaningfully stale. Binary staleness forces unnecessary recomputation.

2. **The change may be partially relevant**: If "analyze" now reports "Found types: A, B, C, D, E, F", the synthesis needs updating, but much of the previous synthesis remains valid. Binary staleness can't express "90% still valid."

3. **Staleness should decay with distance**: In a deep DAG, a change at depth 0 may be completely attenuated by depth 5. Binary staleness propagates uniformly regardless of distance.

### Better Alternatives

**Option A: Confidence Intervals (Bayesian)**

Replace `stale : Bool` with:

```
structure Freshness where
  confidence : Float  -- P(current value is still correct | upstream changes)
  lastComputed : Timestamp
  upstreamChanges : Nat  -- number of upstream changes since computation
```

The confidence decays multiplicatively through the DAG: if cell *i* has confidence *pᵢ* and cell *i+1* depends only on cell *i*, then cell *i+1*'s confidence is approximately *pᵢ · p_{i+1|i}* where *p_{i+1|i}* is the conditional probability that cell *i+1*'s output is still valid given cell *i*'s confidence.

**Pros**: Enables cost-aware recomputation (only recompute when confidence drops below threshold). **Cons**: Requires estimating conditional probabilities, which is hard without historical data.

**Option B: Entropy-Based Staleness**

Model the cell's value as a random variable whose entropy increases when upstream cells change:

```
structure Freshness where
  entropy : Float  -- H(true_value | current_value, upstream_changes)
  threshold : Float  -- recompute when entropy exceeds this
```

When a cell is freshly computed, *H = 0* (we know the value exactly, modulo LLM non-determinism). When an upstream cell changes, *H* increases by an amount proportional to the *mutual information* between the upstream change and this cell's output.

**Pros**: Principled, composable, information-theoretically grounded. **Cons**: Requires knowing the mutual information between cells, which is the hardest quantity to estimate in practice.

**Option C: Differential Staleness (Pragmatic)**

Keep the binary staleness but add a *diff magnitude* estimate:

```
structure Freshness where
  stale : Bool
  estimatedDrift : Nat  -- 0 = trivial change, high = major change
```

The `estimatedDrift` is computed from the upstream change magnitude. A cosmetic change (whitespace, comments) produces low drift; a structural change (new types, deleted functions) produces high drift. Recomputation is triggered when `stale = true AND estimatedDrift > threshold`.

**Pros**: Simple, practical, easy to estimate. **Cons**: Not compositional — drift through a chain of cells is hard to predict from individual drift estimates.

### Recommendation

The current binary staleness is correct as a *first approximation* and has the virtue of being provably sound (the Lean proofs hold). For Gas City v1, I would keep binary staleness but add **Option C** as a refinement: track estimated drift magnitude alongside the boolean. This preserves the formal properties while enabling smarter recomputation scheduling.

For Gas City v2, move to **Option A** (Bayesian confidence) once enough historical data exists to estimate the conditional probabilities. The Lean formalization would need to move from `Bool` to `[0,1]`-valued confidence, which is a significant but tractable change.

---

## 4. Formalizing the Compression Function

### What Is a Cell's Function, Exactly?

Gas City says each cell applies "a function that chooses what matters." Let me formalize this precisely.

A cell *c* with upstream dependencies *{u₁, ..., uₖ}* computes a function:

*f_c : V(u₁) × V(u₂) × ... × V(uₖ) × Prompt(c) → V(c)*

where *V(x)* is the value space of cell *x* and *Prompt(c)* is the prompt template (with holes filled by upstream values). But this is the *extensional* view. Intensionally, the cell is doing something more structured:

### The Cell as a Lossy Codec

A codec has an encoder and a decoder. A cell is the encoder half — it maps a high-dimensional input (the concatenation of upstream values + the prompt) to a lower-dimensional output. There is no explicit decoder; the downstream cells implicitly decode by interpreting the compressed output.

Formally, a cell implements a *quantizer*:

*Q_c : X → X̂*

where *X* is the input space and *X̂* is the output space (a finite codebook determined by the output token budget). The quantizer partitions the input space into *regions*, each mapped to a single codeword (output). Inputs that produce the same output are information-theoretically indistinguishable to downstream cells.

### Required Properties

1. **Idempotence under stability**: If the inputs haven't changed, recomputation should produce output in the same equivalence class. Formally: *Q_c(x) ≈ Q_c(x)* (up to LLM non-determinism). This is NOT guaranteed by LLMs, which is a fundamental departure from classical codecs.

2. **Monotonicity of information**: More input should never produce strictly less informative output. If *X ⊂ X'*, then *H(Q_c(X')) ≥ H(Q_c(X))*. This holds for well-designed prompts but can fail for poorly designed ones (adding irrelevant context can confuse the model).

3. **Composability**: The composition of two quantizers *Q_a ∘ Q_b* should be a valid quantizer. Gas City's effect algebra (seq, par) captures the cost composition. The information composition is: *I(X; Q_a(Q_b(X))) ≤ I(X; Q_b(X))* — the Data Processing Inequality. Each cell in the chain can only LOSE information, never gain it.

4. **Sufficiency for downstream tasks**: The output *Q_c(x)* should be a *sufficient statistic* for the downstream cell's task. That is, the downstream cell should be able to compute its output equally well from *Q_c(x)* as from *x* directly. This is the formal criterion for "choosing what matters" — a good compression function preserves the sufficient statistics and discards the rest.

### The Data Processing Inequality and the Compression Chain

The Data Processing Inequality (DPI) is the most important theorem for understanding Gas City's compression chain:

> **For any Markov chain X → Y → Z: I(X;Z) ≤ I(X;Y)**

In Gas City terms: if cell A produces output Y from source X, and cell B produces output Z from Y, then Z cannot contain more information about X than Y does. Information is monotonically lost through the chain.

This has profound implications:

1. **Depth limits information**: A 10-cell deep chain will inevitably lose more information than a 3-cell chain. The compression chain's depth is an information-theoretic design parameter.

2. **Parallel chains preserve more information**: If two cells independently process the same source, their combined output *I(X; Y₁, Y₂) ≥ max(I(X;Y₁), I(X;Y₂))*. This is the formal justification for Gas City's parallel composition (`Effect.par`). Parallel cells don't just save time — they preserve more information.

3. **Bottleneck cells determine pipeline capacity**: The cell with the lowest *I(X;Y)* is the *information bottleneck* of the pipeline. Improving cells downstream of the bottleneck is futile — the information is already lost. The bottleneck cell determines the pipeline's effective capacity.

**Recommendation**: Gas City should identify bottleneck cells and either (a) allocate more tokens to them (higher rate = lower distortion) or (b) split them into parallel sub-cells (more channels = more capacity). The effect algebra should track not just token cost but *information throughput* per cell.

### Composition Laws

The compression function should satisfy:

- **Associativity**: *(f ∘ g) ∘ h = f ∘ (g ∘ h)* — already proven for `Effect.seq_assoc`
- **Identity**: *id ∘ f = f = f ∘ id* — already proven for `Effect.seq_zero_left/right`
- **Commutativity of parallel**: *f ⊗ g = g ⊗ f* — already proven for `Effect.par_comm`
- **Monotone decrease under composition**: *fidelity(f ∘ g) ≤ fidelity(f)* — captured by the abstract preorder (Data Processing Inequality)

The effect algebra is a *cost algebra*. The fidelity ordering (abstract preorder on information preservation) composes alongside costs: sequential composition is monotone decreasing, parallel composition is monotone increasing, and there is a top element (lossless). This is the dual of cost tracking, expressed as an ordering rather than as quantitative distortion values.

---

## 5. The Second Law and Optimal Information Loss Scheduling

### Thermodynamic Metaphor, Made Precise

The Second Law of Thermodynamics says entropy of a closed system never decreases. For information processing, the analogous statement is the Data Processing Inequality: processing can only destroy information, never create it.

In Gas City's compression chain, this means:

> **Total information about the source monotonically decreases as you move downstream through the DAG.**

This is not a defect — it is the POINT. The goal is not to preserve all information, but to preserve the RIGHT information at each stage. The "optimal information loss schedule" is the schedule that minimizes end-to-end distortion for a given total token budget.

### The Optimal Schedule

The abstract fidelity preorder provides the structural answer to resource allocation. For a linear chain of *k* cells with total token budget *T = Σtᵢ*:

- Sequential composition is monotone decreasing: each cell can only decrease fidelity
- The end-to-end fidelity is bounded by the weakest link in the chain
- More tokens generally improve a cell's fidelity, with diminishing returns

The optimal allocation follows from the preorder structure: **allocate more resources to cells where fidelity is most critical** — cells whose position in the ordering has the greatest downstream impact. This replaces the quantitative "equal marginal distortion per token" condition with a structural analysis of the preorder chain.

### Practical Implications for Gas City

1. **Don't allocate tokens uniformly.** The current `Effect.tokens` field tracks cost but doesn't optimize allocation. A cost-aware scheduler should allocate more tokens to cells that are highest in the fidelity preorder chain — where information loss has the greatest downstream impact.

2. **Source cells (leaves of the DAG) are the most important.** Information lost at the source propagates through the entire chain. The optimal schedule typically allocates the most tokens to source cells and progressively fewer to downstream synthesis cells. This matches the intuition that "analysis" cells should be thorough while "summary" cells can be brief.

3. **Parallel branches should have independent budgets.** The `Effect.par` composition takes `max` of token costs, which is correct for wall-clock time. But for information preservation, parallel branches should be *jointly optimized* — the combined budget should be allocated to maximize total mutual information with the source.

4. **Recomputation should target the bottleneck.** When a source changes and multiple cells become stale, the optimal recomputation order is NOT topological. It is *bottleneck-first*: recompute the cell whose staleness causes the most information loss downstream. This is a departure from the current model where recomputation follows the DAG order.

### The Information Budget Equation

For a Gas City pipeline with total token budget *T*:

*max I(X_source ; X_output) subject to Σ tokens_i ≤ T*

This is a constrained optimization over the rate allocation. The constraint is the total token budget (or total dollar cost). The objective is the mutual information between the original source and the final output.

Gas City's effect algebra gives us the *cost side* of this equation (proven compositional). The *information side* is captured by the abstract fidelity preorder, which provides the ordering structure for information preservation without requiring quantitative rate-distortion curves.

### The Landauer Connection

Landauer's principle states that erasing one bit of information requires *kT ln 2* joules of energy. In Gas City, "erasing" information (compression) costs tokens. There is an analogous "Landauer bound" for LLM compression:

> **Compressing *n* bits of source information to *m* output bits (m < n) requires at least *f(n-m)* tokens of LLM computation, where *f* is model-dependent.**

We don't know *f* precisely, but it exists. This means there is a *minimum token cost* for any given compression ratio. Operating below this cost (too few tokens for the required compression) guarantees information loss beyond the theoretical minimum — the cell is operating outside its capacity region.

---

## Summary of Formal Findings

| Aspect | Current Model | Information-Theoretic Assessment | Recommendation |
|--------|--------------|----------------------------------|----------------|
| Effect algebra | Cost monoid (seq, par) | Correct; fidelity preorder is the dual | Abstract preorder captures information ordering |
| Quality lattice | Total order (draft → excellent) | Good proxy for channel quality | Consider per-cell-type quality metrics |
| Staleness | Binary (fresh/stale) | 0th-order approximation | Add drift magnitude (v1), confidence (v2) |
| Compression | Implicit | Needs formal treatment | Track information throughput per cell |
| Budget allocation | Uniform (per cell) | Suboptimal | Allocate by fidelity criticality (preorder analysis) |
| Recomputation order | Topological | Suboptimal for information preservation | Bottleneck-first recomputation |
| Parallel composition | `max` cost, `min` quality | Correct for cost/quality | Add: preserves more information than sequential |
| `par_le_seq` theorem | Token cost bound | Also an information preservation bound | Formalize the information inequality too |
| DAG depth | Unbounded | Information decays with depth (DPI) | Track and limit effective chain depth |
| Composition laws | Proven (assoc, identity, comm) | Sound | Fidelity preorder composes monotonically |

### The Verdict

The Gas City formalization is *information-theoretically sound in its foundations* but *incomplete in its accounting*. The effect algebra correctly captures cost composition, and the quality lattice provides a useful proxy for channel quality. The staleness model is a valid first approximation. The key Lean proofs (effect associativity, `par_le_seq`, staleness soundness, readiness monotonicity) all hold up under information-theoretic scrutiny.

The dual of the cost algebra is the *fidelity preorder* — an abstract ordering on information preservation that composes through the DAG. The Data Processing Inequality provides the theoretical foundation: sequential composition is monotone decreasing (`seq a b ≤ a`), parallel composition is monotone increasing (`a ≤ par a b`), and there is a top element (lossless). This preorder captures the essential shape of how information degrades through composition without requiring quantitative distortion values or sensitivity multipliers.

The compression chain model is the RIGHT abstraction. Shannon would approve. The question is not whether to compress — every finite-bandwidth channel compresses. The question is whether the compression is *sufficient* (preserves what downstream cells need) and *efficient* (doesn't waste tokens on information that will be discarded anyway). Gas City's typed DAG structure provides the scaffold for answering both questions; the abstract fidelity preorder provides the information-side dual to the cost algebra.
