# Formula Engine v2: Typed Operations, Oracles, and Metacircular Evolution

**Date**: 2026-03-08
**Status**: Approved
**Epic**: gt-wg6 (Formula Engine v2 — The Reactive Bead Calculus)

## Context

The current formula system (TOML checklists with `{{variable}}` interpolation) has
three fundamental limitations identified by cross-domain analysis:

1. **No typed wires** — cells don't declare input/output contracts, so composition is stringly-typed
2. **No real language** — formulas are static step lists, not composable operations
3. **No convergence mechanism** — when an LLM cell produces bad output, nothing catches it before downstream cells waste tokens on garbage

Six domain personas (spreadsheet power user, Haskell developer, DevOps engineer,
game designer, Gas Town LLM engineer, PL designer) independently converged on the
same architecture. This document captures the unified design.

## The Layered Architecture

Four layers, each building on the one below:

```
┌─────────────────────────────────────────────────────┐
│ Layer 3: Metacircular Cells                         │
│   Cells that emit List[Annotation]                  │
│   Runtime-validated before apply                    │
│   Costs tokens. For novel, one-off restructuring.   │
├─────────────────────────────────────────────────────┤
│ Layer 2: Recipes                                    │
│   Named, parameterized, type-checked.               │
│   Desugar to List[Annotation]. Proven correct.      │
│   Zero token cost. The common case.                 │
├─────────────────────────────────────────────────────┤
│ Layer 1: 8 Annotation Primitives                    │
│   addRef, removeRef, splitCell, mergeCell,          │
│   refinePrompt, seedValue, addCell, removeCell      │
│   Lean-formalized. evolve : Proto → List Ann → Proto│
├─────────────────────────────────────────────────────┤
│ Layer 0: Matter Model                               │
│   Proto (solid) → pour → Molecule (liquid)          │
│   → squash → Digest (crystal)                       │
│   Oracle-typed wires. DAG per generation. Reactive. │
└─────────────────────────────────────────────────────┘
```

Layers 2 and 3 both produce `List Annotation` — two on-ramps to the same
Lean-verified `evolve` function. Recipes are the fast lane (verified, free).
Metacircular cells are the escape hatch (validated, costs tokens).

## Oracles as the Type System

Every wire has an **oracle** — a cheap, deterministic predicate that gates what
flows through it.

### Oracle Attachment Points

| Level | Where | What it checks | Cost |
|-------|-------|---------------|------|
| Wire oracle | On a wire between cells | Structural properties of cell output | 0 tokens (deterministic) |
| Cell oracle | After cell evaluation | Cell output + input snapshot | 0-200 tokens (cheap LLM or deterministic) |
| Digest oracle | At squash time | Cross-cell invariants | Low (structural checks) |

### Oracle Verdicts

Oracles return a verdict from a four-valued lattice:

```
reject < redirect < score < accept
```

- **accept**: Output passes; propagate downstream.
- **score(quality)**: Output passes with a quality annotation that propagates through the effect algebra.
- **redirect(cell)**: Output is structurally valid but answers the wrong question; route to a different downstream cell.
- **reject(reason)**: Output fails; trigger retry or escalation.

Composition takes the meet (most restrictive verdict wins).

### Subtyping via Oracle Compatibility

Wire A → B is valid iff: `∀ x, A.output_oracle.accepts(x) → B.input_oracle.accepts(x)`

This is behavioral subtyping (Liskov) for LLM outputs. No declared schemas.
The oracle IS the type. Subtyping IS oracle implication.

### Exploiting Generation/Verification Asymmetry

The core insight: **checking an answer is cheaper than generating it.**

Oracles create a speculative execution pattern:

1. Evaluate cell with a fast/cheap model (e.g., Haiku for draft quality).
2. Run the oracle check (deterministic, zero tokens).
3. If oracle accepts, use the cheap output. If oracle rejects, retry with a more capable model.

The savings are multiplicative in pipeline depth: rejecting at cell A gates
ALL downstream computation. A 200-token oracle check that catches a bad
10,000-token cell output saves 9,800 tokens plus the cost of every downstream
cell that would have consumed the garbage.

## Distillation: From LLM to Deterministic

A cell starts as an LLM call (expensive, non-deterministic, universal). As
digests accumulate across generations, the oracle set tightens. At some point,
the oracles uniquely determine the output, and the LLM becomes replaceable
with a deterministic function.

### The Distillation Lifecycle

| Phase | Cell state | Cost per eval | Oracle role |
|-------|-----------|--------------|-------------|
| **Liquid** | LLM evaluates, oracle checks output | High (tokens) | Gatekeeper: reject bad outputs |
| **Thickening** | Digests accumulate, oracle set tightens | High but narrowing | Pattern detector: constraints tighten |
| **Distilled** | Deterministic function replaces LLM | Zero (no LLM call) | Equivalence guard: verify function = LLM |
| **Re-liquified** | Oracle detects drift, falls back to LLM | High again | Drift detector: input distribution shifted |

### Constraint Narrowing

```
Generation 1:  Oracle says "output must be valid JSON"           → many valid outputs
Generation 5:  + "must contain 'types' key"                      → fewer
Generation 12: + "types must match input schema"                 → narrow set
Generation 20: Oracle set uniquely determines output for input   → DISTILL
```

The phase transition: when `|{x : all_oracles_accept(x)}| = 1` for a given
input class, solving the constraints is cheaper than running the LLM. The
oracle set IS the deterministic function.

Fallback: if upstream data changes enough that the distilled function's output
starts failing oracles, the cell re-liquifies — falls back to LLM evaluation.
This is analogous to a JIT guard: the compiled path is fast, but guard failure
triggers deoptimization.

### The Endgame

The system discovers its own program. It starts as a graph of LLM calls and
gradually crystallizes into a graph of deterministic functions, with the LLMs
serving as the search process that found the right functions. Oracles are the
proof that the found functions are correct.

## The 8 Operations

### Minimal Basis (Graph-Rewrite-Complete)

| Operation | Effect |
|-----------|--------|
| `addCell(spec)` | Add a node to the DAG |
| `removeCell(cell)` | Remove a node from the DAG |
| `addRef(cell, ref)` | Add an edge |
| `removeRef(cell, ref)` | Remove an edge |

These four are DAG-rewrite-complete: any well-formed Proto can be transformed
into any other well-formed Proto. (Proof: delete all cells, rebuild from
scratch. Formalization assigned to polecat — gt-769.)

### Semantic Operations (Invariant-Preserving)

| Operation | Effect | Invariant preserved |
|-----------|--------|-------------------|
| `splitCell(cell, [a, b])` | Decompose one cell into many | Downstream refs fork correctly |
| `mergeCell([a, b], merged)` | Combine many cells into one | Upstream refs union correctly |
| `refinePrompt(cell, prompt)` | Change a cell's instruction | Refs in prompt ⊆ cell's refs |
| `seedValue(cell, value)` | Pre-fill from prior digest | No structural change (runtime) |

Each semantic operation is derivable from the minimal basis but preserves
wiring invariants that raw add/remove do not.

## Recipes (Layer 2)

Recipes are named, parameterized functions from parameters to `List Annotation`.
They are proven correct in Lean (preserve DAG acyclicity, wire oracle
compatibility). They cost zero tokens to invoke.

### Example Recipes

```
recipe enrich(target, source_prompt, refined_prompt):
  src = addCell({ prompt: source_prompt, ... })
  addRef(target, src)
  refinePrompt(target, refined_prompt)

recipe insert_gate(after, before, gate_type):
  gate = addCell({ type: gate_type, refs: [after] })
  removeRef(before, after)
  addRef(before, gate)

recipe parallelize(cell, N):
  legs = [addCell(clone(cell, i)) for i in 1..N]
  agg = addCell({ type: synthesis, refs: legs })
  removeCell(cell)
  -- downstream refs rewire to agg
```

Agents use recipes by filling in parameters. They don't need to understand
DAG invariants or the annotation type system.

## Metacircular Cells (Layer 3)

For novel restructuring that no recipe covers, a cell can emit raw
`List Annotation` as its output. These operations are runtime-validated
before being applied:

1. Simulate `evolve` on the current proto with the emitted operations.
2. Check well-formedness: all refs point to existing cells, no cycles, oracle
   compatibility on new wires.
3. If valid, apply. If invalid, reject (the cell oracle fires with `reject`).

This is the same postcondition system that recipes use, but applied at runtime
instead of at definition time.

### Stratification

Metacircular cells can only modify the **next generation's** proto, never the
current molecule. The molecule boundary is the predicativity constraint:

```
Gen N: metacircular cell emits [addCell(orchestrator)]
       → orchestrator added to Proto_{N+1}

Gen N+1: orchestrator cell emits [refinePrompt(...), addRef(...)]
         → changes applied to Proto_{N+2}
```

No self-modification within a generation. The DAG is fixed during execution.
Cycles emerge across generations, never within them.

## Addressing Yegge's Criticisms

| Criticism | Resolution |
|-----------|-----------|
| "Formulas aren't a real language" | Recipes + metacircular cells = typed, composable graph evolution language |
| "No typed wires" | Oracles on every wire = behavioral type system with subtyping |
| "Platform not product" | Oracle-typed wires = Bezos Mandate (cells are services with contracts) |
| "Formalization should be invisible" | Lean proofs verify recipes. Users see recipes and oracles, not theorems |
| "Compression is the central challenge" | Distillation = compression (replace expensive LLM with cheap deterministic function) |

## Design Principles

1. **Oracles are types.** No declared schemas. Wire validity = oracle compatibility.
   Subtyping = behavioral (oracle implication).

2. **Recipes are the common case.** Zero token cost. Proven correct in Lean.
   Agents fill parameters, not graph operations.

3. **Metacircular cells are the escape hatch.** For novel restructuring that
   no recipe covers. Runtime-validated. Costs tokens.

4. **Stratified evolution.** Molecule boundary = predicativity constraint.
   No self-modification within a generation.

5. **Distillation is convergence.** Oracle constraints tighten across
   generations until the LLM is replaceable with a deterministic function.

6. **Generation/verification asymmetry everywhere.** Cheap oracles gate
   expensive LLM calls. Savings are multiplicative in pipeline depth.

7. **The graph discovers its own program.** The endgame is a mostly-deterministic
   system with LLM fallback for novelty.

## Risks and Mitigations

| Risk | Mitigation |
|------|-----------|
| Dolt write amplification from reactive staleness cascades | Batch commits: all state changes from one evaluation round = one Dolt commit |
| Oracle rigidity (too strict, rejects valid LLM output) | Oracles start loose, tighten from observation. `score` verdict allows soft quality signals |
| Agent skill gap for raw graph operations | Recipes abstract away DAG invariants. Layer 3 is the escape hatch, not the common path |
| Staleness cascades cause runaway re-evaluation | Lazy reactivity: stale cells don't auto-evaluate. Orchestrator decides when. Budget caps on re-evaluation rounds |
| Metacircular drift (evolution produces unusable protos) | Well-formedness oracle on emitted annotations. Rejection before apply, not after |
| Distillation false confidence (deterministic function wrong on new inputs) | Re-liquification on oracle failure. The oracle is always running, even on distilled cells |

## Research Questions (Open)

1. Can oracle compatibility (behavioral subtyping) be checked statically, or only
   at pour time? If static, what's the decidability boundary?

2. What triggers distillation? Manual annotation ("this cell is stable enough"),
   automatic detection (N consecutive identical outputs), or oracle-driven
   (constraint set uniquely determines output)?

3. How do oracles compose across `Sheet.compose` (inter-sheet wires)? Do
   cross-sheet oracles inherit from the composed sheets, or do they need
   independent specification?

4. What is the right representation for recipes in the Lean formalization?
   Functions `Params → List Annotation` with pre/postconditions? Or a
   dedicated `Recipe` inductive type?
