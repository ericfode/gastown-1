# Gas City Formula Engine: Unified Vision

**Author**: morpheus (crew)
**Date**: 2026-03-08
**Epic**: gt-wg6 (Formula Engine v2 — The Reactive Bead Calculus)

---

## What Gas City Already Is

Gas City is a **reactive computation surface** where typed cells compose through
an algebraically verified effect system. This is not aspirational — the algebra
is proven in Lean 4 (20 sections, zero sorries, zero custom axioms), the
primitives map entirely to existing Gas Town commands, and agents already
intuit the paradigm across three independent framings.

The core insight: **every reactive spreadsheet primitive is already a `bd` command.**

| Spreadsheet Concept | Gas Town Primitive |
|---------------------|--------------------|
| Cell | Bead |
| Cell state (empty/fresh/stale) | `bd set-state cell=X` |
| DAG wiring | `--deps` on `bd create` |
| Ready set | `bd ready` |
| Evaluate | `bd update --claim` + `bd close` |
| Staleness propagation | `bd label propagate` or molecule re-pour |
| History | Digest chain (squash → distill → pour) |
| Gate | `bd gate` |
| DAG visualization | `bd graph --html` |

Zero new code is needed for the core algebra. Gas City is an interpretation
layer over Gas Town, not a separate system.

---

## The Three-Phase Matter Model

Work items have physical phases, borrowed from chemistry and validated by the
Chemical Abstract Machine (CHAM) formalism:

```
Proto (solid)  ──pour──▶  Molecule (liquid)  ──execute──▶  All closed
                                                               │
                                                          squash
                                                               │
                                                          Digest (crystal)
                                                               │
                                                    upstream changes
                                                               │
                                                   distill ──▶ Proto'
                                                               │
                                                          pour ──▶ Molecule₂
```

| Phase | State | Properties |
|-------|-------|------------|
| **Solid** (Proto) | Template, not instantiated | Portable, reusable, version-controlled |
| **Liquid** (Molecule) | Active, beads being evaluated | Mutable cell states, DAG scheduling |
| **Vapor** (Wisp) | Ephemeral molecule | Auto-cleanup, no digest |
| **Crystal** (Digest) | Squashed, immutable record | Content-addressed, audit trail |

**Staleness IS re-instantiation.** Completed work is never mutated. When upstream
changes, distill the proto and pour a fresh molecule. The old digest remains as
immutable history. This eliminates:
- Mutable state ambiguity ("what state was this bead at time T?")
- Reopen/relabel confusion
- Lock contention on shared beads
- Error-prone rollback through undo mutations

---

## Annotated Evolution: Cycles Across Generations

Within each molecule, the computation is a DAG (acyclic). Across generations,
user annotations drive cyclic refinement:

```
Proto₁ ──pour──▶ Mol₁ ──squash──▶ Digest₁
                                      │
                             annotate (addRef, splitCell, ...)
                                      │
                             evolve ──▶ Proto₂ ──pour──▶ Mol₂ ──squash──▶ Digest₂
                                                                              │
                                                                         annotate
                                                                              │
                                                                    evolve ──▶ Proto₃ ...
```

Annotations are first-class operations on the proto graph:

| Annotation | Effect |
|------------|--------|
| `addRef(cell, newRef)` | Wire a new dependency |
| `removeRef(cell, oldRef)` | Remove a dependency |
| `splitCell(cell, [a, b])` | Decompose into sub-cells |
| `mergeCell([a, b], merged)` | Combine cells |
| `refinePrompt(cell, prompt)` | Improve a cell's instruction |
| `seedValue(cell, value)` | Pre-fill from prior digest |
| `addCell(spec)` | Extend the graph |
| `removeCell(cell)` | Prune the graph |

The cycle is: execute → observe → annotate → evolve → re-pour. Each generation
unrolls one step of the cycle. The DAG invariant holds within each generation;
the cycle emerges across generations.

This is formalized in Lean 4 (Section 20) with `applyAnnotation`, `evolve`,
and `EvolutionHistory` tracking the annotation chain.

---

## What the Research Says

### Academic Foundations (warboy/gt-xo9)

Gas City sits at the intersection of five established fields:

1. **Self-Adjusting Computation** (Acar 2002): Gas City's staleness propagation
   is precisely Adapton's dirtying phase. The missing optimization: demand-driven
   cleaning (defer recomputation until output is demanded). For large DAGs where
   only some outputs are observed, this avoids computing unobserved branches.

2. **Graded Monads with Duoidal Structure** (Katsumata 2014, Aguiar & Mahajan):
   Gas City's effect algebra `(seq, par, zero)` IS a graded monad with duoidal
   interchange. Not an analogy — a direct mathematical identification, proven in
   Lean. Coeffects (Petricek 2014) would add backward-flowing quality demands.

3. **Fidelity Preorder** (Shannon 1959, Data Processing Inequality): Each LLM
   cell is a lossy codec. Fidelity has an abstract preorder: sequential
   composition is monotone decreasing, parallel is monotone increasing. The
   preorder captures information preservation without quantitative distortion.

4. **Chemical Abstract Machine** (Berry & Boudol 1992): Gas City's "molecule"
   terminology is literal. Beads = molecules, formulas = reaction rules,
   rigs = membranes. CHAM semantics enable deadlock detection and confluence
   analysis.

5. **Stigmergy** (Grassé 1959): Gas City IS a stigmergic system. Polecats
   coordinate by modifying the bead environment, not by messaging each other.
   `bd` operations are fully stigmergic; `gt nudge` is the only departure.

### Sci-Fi Precedents (imperator/gt-cza)

Eight works map to Gas City's architecture:

| Concept | Source | Gas City Analog |
|---------|--------|-----------------|
| Pack mind = emergent identity | Vinge | Molecule > sum of beads |
| Primer = reactive surface | Stephenson | Reactive sheet = live computation surface |
| Freshness is thermodynamic | Chiang | Staleness tracking = resource budgeting |
| Chinese Room execution works | Watts, Liu | Formula-driven polecats don't need to "understand" |
| Trust-based autonomy scales | Banks | Capability ledger = trust substrate |
| Stubs = forked sandboxes | Gibson | Git branches = sandboxes |
| Substrate is programmable | Egan | The computation substrate is the thing being designed |

The deepest insight: **Gas City is not a task runner or agent orchestrator. It is
a reactive computation surface for software engineering** — a Primer (Stephenson)
where cells are agents and formulas are workflows.

### Creator's Philosophy (organic/gt-k79)

Mapping Yegge's corpus to Gas City reveals the design philosophy already in
play:

1. **Platform, not product.** Every internal interface should be externalizable.
   Typed wires complete the Bezos Mandate for Gas City.

2. **Formulas are the extension language.** TOML checklists are the `.emacs` of
   1985. Gas City needs its Elisp moment — a formula language expressive enough
   for third-party workflow authors.

3. **Compression is the central challenge.** Multi-agent systems are compression
   pipelines. Each agent is a lossy codec. The effect algebra tracks compression
   cost and quality.

4. **Liberal base, conservative accent.** Keep the runtime flexible (agents are
   non-deterministic). Add optional static guarantees (typed wires, Lean proofs)
   for those who want them. Never force the math on users.

5. **Agent UX is the new developer UX.** Every `gt` and `bd` command is an
   agent-facing API. Design for agents first, humans second.

---

## The A/B/C Test: What We Learned

We tested three paradigm framings across 6 polecats (2 per paradigm):

- **Paradigm A (YAML-first)**: Sheet YAML is the schema, beads are runtime instances
- **Paradigm B (Bead-first)**: Beads ARE the cells, labels ARE the state
- **Paradigm C (Molecule lifecycle)**: Staleness = re-instantiation, digests are immutable

**Result: 6/6 correct.** All polecats correctly executed reactive spreadsheets
regardless of paradigm framing.

**Qualitative winner: Paradigm C.** Both molecule lifecycle polecats spontaneously
explored re-instantiation, partial staleness, digest chains, and version history.
The paradigm's metaphors naturally invited deeper engagement with the system's
semantics. A and B were correct but mechanical.

**Recommendation**: Gas City should lead with Paradigm C (molecule lifecycle) as
the primary mental model, with A and B available as implementation details:
- Protos are defined in TOML (Paradigm A's schema)
- Individual beads use labels for state (Paradigm B's primitives)
- The lifecycle is molecule-based (Paradigm C's semantics)

---

## Architecture: What to Build

### Already Done (Zero New Code)

The core algebra maps to existing Gas Town primitives. The Lean formalization
(2116 lines, 20 sections) proves the mathematical properties. The molecule
lifecycle (pour/squash/distill) already exists in the bead formula system.

### Near-Term Additions

1. **Typed Wires** — Cell inputs/outputs get declared types. This is the Bezos
   Mandate: without typed interfaces, Gas City is a task tracker, not a platform.

   ```toml
   [[steps]]
   id = "find-bugs"
   type = "inventory"
   inputs = { code = "text" }
   output = "inventory"
   refs = ["read-code"]
   ```

2. **Demand-Driven Cleaning** (Adapton) — Mark stale eagerly (already done),
   but defer recomputation until output is demanded. For large DAGs, this avoids
   computing unobserved branches. Cost: nearly zero. Savings: proportional to
   DAG breadth.

3. **Delta-Aware Recomputation** — Instead of full re-evaluation, construct
   delta prompts: "your previous output was X, the input changed by ΔY, update
   accordingly." Dramatically reduces token cost for incremental updates.

4. **Annotation CLI** — Expose the evolution annotations as `bd mol annotate`
   commands, making the evolve cycle accessible from the shell:

   ```bash
   bd mol annotate gt-digest-a1 addRef find-bugs security-scan
   bd mol annotate gt-digest-a1 splitCell review-report [code-review, security-review]
   bd mol evolve gt-digest-a1 mol-code-review-v2
   bd mol pour mol-code-review-v2 --var repo=myapp
   ```

### Longer-Term Vision

5. **Formula Language v2** — Evolve from TOML checklists to a typed functional
   language for agent workflows. The graded monad formalization is the type
   theory; the formula language is the user-facing syntax.

6. **Living Grid** — A spreadsheet-like interface showing the computation graph:
   cells with values, freshness states, typed wires, cost tracking. Excel's
   grid as the UX target for agent coordination visibility.

7. **Coeffect-Based Demand Propagation** — Quality requirements flow backward
   through the DAG. If a downstream decision cell only needs yes/no, upstream
   synthesis doesn't need "excellent" quality. This reduces token budgets
   optimally.

---

## Design Principles

1. **The reactive surface is the paradigm.** Gas City is not a feature of Gas
   Town — it is the conceptual frame that makes Gas Town's primitives compose
   into something greater than the sum.

2. **Immutability over mutation.** Completed work is crystal. Evolution happens
   by pouring new molecules, not reopening old beads.

3. **The intelligence is in the topology, not the nodes.** Polecats are Chinese
   Rooms executing formulas. The system is smarter than any individual agent.
   Invest in better formulas, not smarter agents.

4. **Freshness is a consumable resource.** Every recomputation costs tokens.
   Budget freshness like energy — cheap cells stay fresh, expensive cells
   tolerate staleness.

5. **Liberal base, conservative accent.** The runtime is flexible and forgiving.
   The algebra is rigorous and proven. Users see the flexibility; the proofs
   are invisible substrate.

6. **Compression is survival.** Each agent in the pipeline performs semantic
   compression. The Data Processing Inequality governs multi-agent chains.
   Wide and shallow beats deep and narrow.

7. **Stigmergy is the coordination model.** Agents coordinate by modifying the
   shared bead environment, not by messaging each other. The environment IS
   the communication channel.

---

## Concept Map

```
                    ┌─────────────────────────────────┐
                    │     Gas City Formula Engine      │
                    │                                  │
                    │  ┌───────────┐  ┌─────────────┐ │
                    │  │  Proto    │  │  Annotation  │ │
                    │  │  (solid)  │──│  (evolve)    │ │
                    │  └─────┬─────┘  └──────┬──────┘ │
                    │        │pour            │        │
                    │        ▼                │        │
                    │  ┌───────────┐          │        │
                    │  │ Molecule  │          │        │
                    │  │ (liquid)  │          │        │
                    │  │           │          │        │
                    │  │ Cell DAG: │          │        │
                    │  │  A → B ─┐ │          │        │
                    │  │  C → D ─┤ │          │        │
                    │  │         ▼ │          │        │
                    │  │    sink   │          │        │
                    │  └─────┬─────┘          │        │
                    │        │squash          │        │
                    │        ▼                │        │
                    │  ┌───────────┐          │        │
                    │  │  Digest   │──────────┘        │
                    │  │ (crystal) │   distill+evolve  │
                    │  └───────────┘                   │
                    │                                  │
                    │  Substrate: Gas Town primitives  │
                    │  Algebra: Lean 4 (proven)        │
                    │  Metaphor: Reactive spreadsheet  │
                    └─────────────────────────────────┘
```

---

## References

### Internal
- Lean formalization: `lean4/BeadCalculus/GasCity.lean` (2116 lines, 20 sections)
- Paradigm A/B/C test: `docs/design/paradigm-ab-test.md`
- Scenario tests: `internal/formula/reactive/scenario_test.go`
- Molecule lifecycle examples: `polecats/rictus/gastown/docs/examples/molecule-lifecycle-examples.md`

### Research Campaign
- Sci-fi precedents: `polecats/imperator/gastown/docs/design/scifi-reactive-computation-precedents.md`
- Academic precedents: `polecats/warboy/gastown/docs/design/academic-precedents-reactive-dataflow-effects.md`
- Yegge synthesis: `polecats/organic/gastown/docs/design/yegge-gas-city-synthesis.md`

### Academic (Top 5)
1. Acar, U.A. "Self-Adjusting Computation." CMU, 2005.
2. Hammer, M.A. et al. "Adapton: Composable, Demand-Driven Incremental Computation." PLDI 2014.
3. Katsumata, S. "Parametric Effect Monads and Semantics of Effect Systems." POPL 2014.
4. Berry, G., Boudol, G. "The Chemical Abstract Machine." TCS 96(1), 1992.
5. Shannon, C.E. "Coding Theorems for a Discrete Source with a Fidelity Criterion." IRE 1959.
