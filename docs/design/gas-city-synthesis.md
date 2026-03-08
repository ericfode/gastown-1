# Gas City: Design Synthesis

**Synthesized from**: 5 design perspectives (Power User, Wolfram, Feynman, Information Theory, Tao), the Visualization Design, and the Lean 4 formalization (1055 lines).

**Date**: 2026-03-08

---

## 1. Executive Summary

Gas City is a reactive computation substrate for LLM agent coordination that extends Gas Town's existing bead/formula/polecat infrastructure with typed wires, an effect algebra, staleness propagation, and compression-aware dataflow. It models agent work as a spreadsheet: cells hold LLM-computed values, typed wires carry data between them, and a reactive engine tracks freshness and cost. The key insight, validated across all five perspectives, is that LLM computation has effects (cost, quality, freshness, provenance) that traditional computation lacks, and these effects compose algebraically in ways that can be formally verified. The Lean 4 formalization already proves the core laws; what remains is bridging the clean algebra (which works below the formula level) to the messy reality of multi-agent project execution.

---

## 2. Convergent Themes

### 2.1 LLM computation is fundamentally lossy and each handoff loses information

Every perspective agrees. The Power User observes that "molecule steps have no memory" and agents "start from scratch" (Power User, Section 4). The Information Theorist formalizes this as the Data Processing Inequality: "I(X;Z) <= I(X;Y)" for any chain X -> Y -> Z (Info Theory, Section 4). Tao models each cell as a "lossy codec" (Tao, Section 3.1). Feynman notes "tokens are not conserved; they're more like fuel that gets burned" (Feynman, Section 3). Wolfram frames it as "coarse-graining" from an irreducible substrate (Wolfram, Section 2).

**Concrete agreement**: The compression chain model (tracking how many lossy transformations separate a cell from raw data) is the RIGHT abstraction. All five perspectives endorse it.

### 2.2 The typed DAG is the right structural primitive

All perspectives treat the DAG as foundational. The Power User wants "typed wires instead of untyped dependencies" (Power User, Section 4). Wolfram models the DAG as a hypergraph with rewriting rules (Wolfram, Section 1). Feynman maps DAG nodes to Feynman diagram vertices (Feynman, Section 6). Tao identifies the DAG as a presheaf category with a Grothendieck topology (Tao, Section 1.4). The Information Theorist treats each DAG edge as a communication channel (Info Theory, Section 1).

**Concrete agreement**: Typed wires (not just binary dependency edges) are essential. The type carries semantic content about what data flows, enabling type-directed dispatch and better compression policies.

### 2.3 Binary staleness is a correct first approximation but insufficient long-term

The Power User calls staleness blindness a critical pain point (Power User, Section 1). The Information Theorist argues binary staleness is "0th-order" and proposes three alternatives: Bayesian confidence, entropy-based, and differential (Info Theory, Section 3). Tao connects staleness to sheafification (Tao, Section 1.4). Feynman frames re-evaluation as a perturbation series where the error rate epsilon controls convergence (Feynman, Section 6).

**Concrete agreement**: Keep binary staleness for v1 (it has proven properties in Lean). Extend to drift-magnitude tracking for v1.5, Bayesian confidence for v2.

### 2.4 The effect algebra captures something real and compositional

All perspectives validate that (Effect, seq, par, zero) with proven associativity, commutativity, and the par_le_seq bound is a genuine algebraic structure. Tao names it precisely: a graded monad with duoidal structure (Tao, Section 1.2-1.3). Feynman interprets it as a renormalization procedure from microscale (tokens) to macroscale (formulas) (Feynman, Section 4). The Information Theorist treats it as the cost side of a cost-distortion dual (Info Theory, Section 5). The Power User wants it for cost tracking and budget enforcement (Power User, Section 3).

**Concrete agreement across 5/5 perspectives**: The effect algebra is sound, proven, and practically useful. It is the foundation layer.

### 2.5 Approximate coordination works because of universality

Feynman asks the sharpest version of this question: "Why does approximate coordination work at all?" and answers with universality -- macroscopic behavior is insensitive to microscopic fluctuations above a quality threshold (Feynman, Section 5). Wolfram offers an alternative framing: partial confluence means the information content of the DAG is execution-order-independent even if token-level representations differ (Wolfram, Section 1). Tao's no-free-lunch theorem formalizes the limit: even perfect agents lose information through pipeline topology (Tao, Section 2.4).

**Concrete agreement**: There exists a quality threshold below which formulas fail catastrophically and above which improvements are marginal. Finding this threshold empirically is a high-value research target.

---

## 3. Key Tensions

### 3.1 Prediction vs. Irreducibility

**Wolfram** argues LLM agents are computationally irreducible: "You cannot predict what an LLM will output for a given prompt without running the inference" (Wolfram, Section 2). Therefore, cost estimation is fundamentally approximate, and builders should "stop trying to optimize the coordinator and start building better infrastructure for running the computation."

**The Power User** wants cost visibility: real-time tracking, hard budget caps, and historical cost data from prior digests (Power User, Section 3). Cost observability is listed as a 10x productivity factor.

**The Information Theorist** clarifies: token cost is not predictable before execution. The effect algebra tracks actual cost after the fact and composes costs across the DAG for accounting and budgeting (Info Theory, Section 5).

**Resolution**: Wolfram is right — you cannot predict token consumption before execution. LLM inference cost varies with input content, model behavior, and output length. The effect algebra is a measurement and accounting tool, not a prediction engine. Track actual cost, enforce runtime budget caps, and use historical digest data to inform (not predict) future runs.

### 3.2 Spreadsheet flatness vs. Hypergraph richness

**The Power User** wants a spreadsheet: cells, values, staleness, all visible at once (Power User, Section 2). The visualization design supports this with a Living Grid UI (Visualization, Section 1).

**Wolfram** argues the spreadsheet is a "one-dimensional projection" of a richer hypergraph structure where agents can modify the graph itself (create beads, add dependencies, split tasks) (Wolfram, Section 5). The spreadsheet model treats graph mutation as an external side effect; the hypergraph model makes it first-class.

**Tao** partially sides with Wolfram: the presheaf structure captures context-dependence that a flat grid cannot (Tao, Section 1.4). But Tao also acknowledges Gas City already has the right algebraic structure "named correctly" (Tao, Section 6.2).

**Resolution**: Build the spreadsheet UI first (the Power User needs it now), but design the internal model as a hypergraph that the spreadsheet projects. The visualization is the 2D shadow; the computation model is the full structure.

### 3.3 Where does the clean algebra break down?

**Feynman** identifies a "critical point between formula and project level" (Feynman, Section 4). Below it: clean algebra, Lean proofs hold. Above it: agents crash, contexts compact, networks fail. "These are relevant perturbations that change the universality class."

**Tao** is more optimistic: if Gas City forms a topos, it has an internal logic that extends the algebra to the project level (Tao, Section 5.5). Agents could reason about other agents' computations using the internal logic.

**The Information Theorist** is pragmatic: the algebra is "information-theoretically sound in its foundations but incomplete in its accounting" — it tracks cost, and the abstract fidelity preorder now provides the dual: an ordering on information preservation that composes alongside costs (Info Theory, Summary).

**Resolution**: This is the deepest open question. The algebra works beautifully below formula level. Whether it can be extended above that level (the Tao conjecture) or whether project-level coordination requires fundamentally different tools (the Feynman position) is unresolved.

### 3.4 Is multi-branch exploration worth the cost?

**Wolfram** proposes running cells multiple times to check branch stability (Wolfram, Section 3). Branch-stable values resist staleness and can be cached aggressively.

**Feynman** agrees in principle but quantifies the cost: the perturbation expansion adds cost proportional to n*epsilon per loop (Feynman, Section 6). If epsilon is small, tree-level (single run) suffices.

**The Power User** implicitly disagrees: "I have ZERO idea how many tokens my polecats are burning" (Power User, Section 4). Running cells multiple times for branch stability analysis would worsen the cost visibility problem.

**Resolution**: Multi-branch exploration is a v2+ feature. For v1, the Multiverse Diff visualization (Visualization, Section 7) provides the UX for comparing runs without mandating multi-branch execution.

---

## 4. The Wolfram Connection

Wolfram's contributions are structural. Specifically, the rulial graph framework suggests:

1. **Rule GC as the primitive rule**: "Read, compute, signal" (5 lines) is sufficient to generate all coordination patterns -- pipelines, fan-out, fan-in, compression chains, staleness, escalation (Wolfram, Section 6). Gas City's formula system is a programming language for the Rule GC machine.

2. **Confluence as correctness criterion**: If the bead DAG has a unique topological sort (or equivalent sorts), the computation is causally invariant. This formally justifies parallelism: execution order does not matter for independent beads (Wolfram, Section 1).

3. **Dimension estimation predicts parallelism**: The effective dimension of the computation graph (growth rate of the dependency light cone) predicts maximum useful polecat count. "The rig's polecat count should match the effective dimension of the current computation graph" (Wolfram, Section 5).

4. **Branch stability as a caching criterion**: Values that are invariant across multiple LLM runs (branch-stable in multiway terminology) are safe to cache aggressively. The multiway merge protocol provides a concrete algorithm: run 3 times, extract canonical forms, compare (Wolfram, Section 3).

---

## 5. The Feynman Connection

Feynman's contributions are physical and testable. Specifically:

1. **LLM computation is statistical mechanics, not quantum mechanics**: Softmax IS the Boltzmann distribution. Temperature IS temperature. There is no interference between output paths. This is a mathematical identity, not an analogy (Feynman, Section 2).

2. **Quality is the effective temperature that survives renormalization**: At the token level, everything is stochastic. At the cell level, the stochasticity is absorbed into the quality label. Draft = high temperature (cheap, noisy). Excellent = low temperature (expensive, precise) (Feynman, Section 4).

3. **The perturbation series for re-evaluation converges when epsilon < 1**: Total formula cost including re-evaluation is tree-level cost * 1/(1-epsilon) where epsilon is the per-cell error rate. If epsilon >= 1, no amount of re-evaluation saves the formula -- restructure the DAG instead (Feynman, Section 6).

4. **The critical question: What is the universality class of Gas City?** Systematically degrade cell quality and measure formula output quality. Find the phase transition. This determines the error budget (Feynman, Section 5).

5. **The DAG forbids loops, so Gas City formulas are tree-level diagrams only**: Re-evaluation cycles look like self-energy diagrams, and staleness propagation is the regularization scheme (Feynman, Section 6).

---

## 6. The Tao Connection

Tao's contributions are categorical and structurally precise:

1. **Gas City already IS a graded monad**: The effect algebra (Effect, seq, zero) with proven associativity and identity laws IS the graded monad multiplication and unit. This is not a coincidence -- it is a discovered structure that should be named (Tao, Section 1.2).

2. **The duoidal structure**: Sequential and parallel composition form two monoidal products related by par_le_seq (the interchange lax morphism). This is a duoidal category in the sense of Aguiar-Mahajan (Tao, Section 1.3).

3. **Staleness propagation is sheafification**: The DAG with the freshness/staleness distinction forms a site with a Grothendieck topology. propagateStale is the associated sheaf functor. This is why the staleness proofs have such clean algebraic properties (Tao, Section 1.4).

4. **The Prompt-Evaluate adjunction**: Prompting (cheap, preserves colimits) is left adjoint to evaluating (expensive, preserves limits). The monad T = Evaluate . Prompt is the "one round-trip" monad. The comonad W = Prompt . Evaluate is the refinement comonad (Tao, Section 5.3).

5. **The representation theorem conjecture**: Every Gas City computation is equivalent to a navigation strategy in the fidelity preorder that maximizes information preservation. If true, pipeline design reduces to preorder-optimal resource allocation (Tao, Section 6.4).

---

## 7. What's Already Formalized

The Lean 4 file `GasCity.lean` (1055 lines, 13 sections) contains:

**Proven theorems (not sorry'd)**:
- Effect algebra: `seq_assoc`, `seq_zero_left`, `seq_zero_right`, `par_comm`, `par_assoc`, `par_le_seq`
- Quality lattice: `min_comm`, `min_assoc`, `min_excellent_left`, `min_excellent_right`
- Staleness: `propagateStale_sound`, `propagateStale_preserves`, `propagateStale_non_fresh`
- DAG readiness: `source_ready_initially`, `sink_ready_after_source`, `monotone_witness`
- Compression: `extend_increases_depth`, `extend_fidelity_le` (monotone decrease under composition)
- Pins: `pin_blocks_evaluation`, `unpin_restores_state`
- Input snapshots: `log_captures_inputs`
- Recomputation: `eager_always_recomputes`, `budgeted_respects_limit`, `convergent_stops`
- Map operations: `map_cell_count`, `instantiate_preserves_cell_count`

**Defined structures**:
- `Quality`, `Effect`, `EffCell`, `EffSheet` (effect system)
- `Provenance`, `FullEffect`, `AgentCapability` (dispatch)
- `CompressionPolicy`, `CompressionStep`, `CompressionChain` (information decay)
- `SheetTemplate`, `ParamSet`, `Aggregation` (parameterized maps)
- `CellView`, `ProvenanceLink`, `ProvenanceTrace`, `SankeyNode` (visualization)
- `PinState`, `PinnedSheet` (debugging)
- `InputSnapshot`, `ComputationRecord`, `ExecutionLog` (provenance)
- `RecomputePolicy`, `RecomputeDecision` (staleness policy)

**Non-vacuity witnesses**: Every section includes concrete `example` terms that demonstrate the structures are inhabited and the theorems' hypotheses are satisfiable.

---

## 8. What's Missing

### 8.1 Between the formalization and the design documents

| Gap | Described in | Not in Lean |
|-----|-------------|-------------|
| Fidelity preorder (dual of cost algebra) | Info Theory Section 4, Tao Section 2.2 | Abstract preorder model (reflexive, transitive ≤, monotone composition) |
| Graded monad naming | Tao Section 1.2 | Effect monoid exists but not named as graded monad |
| Presheaf/sheaf structure | Tao Section 1.4 | propagateStale exists but not connected to sheaf theory |
| Prompt-Evaluate adjunction | Tao Section 5.3 | Not formalized |
| Context window capacity | Info Theory Section 1 | Effect tracks tokens but not context overflow |
| Multi-branch merge protocol | Wolfram Section 3 | No branch stability formalization |
| Iteration/convergence | Feynman Section 6, Tao Section 2.3 | convergent policy exists but no contraction mapping proof |
| Conditional (gate) cells | GasCity.lean open question 2 | CellType has gate, but gate semantics not formalized |
| Multi-town federation | GasCity.lean open question 5 | Not started |

### 8.2 Between the formalization and a working system

- No runtime: the Lean code is a specification, not an executable engine
- No integration with Gas Town's actual bead/Dolt/polecat infrastructure
- No `gt eval` or `gt map` CLI commands
- No visualization layer (though CellView and SankeyNode are defined as types)
- No actual LLM dispatch -- `AgentCapability.canHandle` is defined but never invoked by a real scheduler

---

## 9. Concrete Next Steps

Ranked by impact and feasibility:

1. **Add `stale:bool` field to beads in Gas Town** (low effort, high impact). The single most impactful change. Enables reactive staleness tracking on the existing infrastructure. Required by Power User, formalized in Lean, trivial to implement in Dolt schema. Without this, nothing else works.

2. **Implement `gt eval` command** (medium effort, high impact). Fill a prompt template with upstream bead values and dispatch to a polecat. This is the minimum viable Gas City operation: one cell evaluation with input snapshots. The Lean `evaluateWithLog` function is the spec.

3. **Add `compression_depth:int` and `input_snapshot` to beads** (low effort, medium impact). Track how many compressions separate a bead from source data, and which upstream versions were consumed at compute time. Enables the debugging workflow in the visualization design (Section: Debugging a Wrong Synthesis).

4. **Build the Living Grid view** (medium effort, high impact). The Power User's most requested feature: a single view showing all cells, their values, staleness, cost, and dependencies. Start with a terminal-based grid (ASCII art like the mockup in the visualization doc). Graphical UI can follow.

5. **Formalize the fidelity preorder** (low effort, medium impact). The abstract preorder on fidelity (reflexive, transitive ≤, with monotone seq/par composition and a top element for lossless) captures the information-preservation structure that sensitivity tracking was trying to approximate. The preorder composes alongside the cost algebra without requiring quantitative distortion values.

6. **Implement the Multiverse Diff** (low effort, medium impact). Run a cell 3 times, compare outputs, identify stable vs. volatile claims. This is Wolfram's branch stability protocol in a practical UX. The visualization design already specifies the UI.

7. **Formalize the graded monad structure** (medium effort, medium impact). Refactor existing Lean theorems to explicitly name the graded monad, connecting Gas City to Mathlib's category theory library. Mostly a renaming, but establishes the bridge to Tao's deeper conjectures.

8. **Add differential staleness (drift magnitude)** (low effort, medium impact). Implement the Information Theorist's Option C: keep binary staleness but add an `estimatedDrift : Nat` alongside it. Prevents unnecessary recomputation when upstream changes are cosmetic.

9. **Implement `gt map` (parameterized template instantiation)** (medium effort, medium impact). The drag-to-fill operation: apply one formula template across N inputs. The Lean `SheetTemplate.mapOver` function is the spec. Enables the fan-out pattern from the Feynman diagrams.

10. **Run the universality experiment** (medium effort, high research value). Feynman's testable prediction: systematically degrade cell quality and measure formula output quality to find the phase transition. This determines the error budget for the entire system and validates (or falsifies) the universality hypothesis.

---

## 10. The One-Sentence Vision

Gas City is a reactive spreadsheet for LLM agent coordination where typed cells compose through an algebraically verified effect system, making the cost, quality, freshness, and information flow of multi-agent computation visible, predictable, and formally guaranteed.
