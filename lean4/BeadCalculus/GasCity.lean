/-
  BeadCalculus.GasCity — The Next Evolution: A Computation Substrate for LLM Agents.

  Gas Town = beads + DAGs + agents + formulas. Assembly language for agent work.
  Gas City = Gas Town + types + effects + composition laws. A PROGRAMMING LANGUAGE
  for agent computation where the CPU is an LLM and the memory is Dolt.

  The progression:
    Beads   (data)        — Universal data type, DAG dependencies
    Gas Town (runtime)    — Multi-agent execution, scheduling, persistence
    Bead Calculus (types) — Cell types, typed wires, well-typedness
    Gas City (effects)    — Cost, quality, freshness, provenance

  THE KEY INSIGHT: LLM computation has effects that traditional computation doesn't.
  Each cell evaluation:
    - COSTS tokens (bounded resource, not free like CPU cycles)
    - Produces OUTPUT of variable QUALITY (not just correct/incorrect)
    - Has FRESHNESS that decays (upstream changes make results stale)
    - Has PROVENANCE (who computed this, which model, what prompt)

  These are EFFECTS in the PL sense. An effect system lets us reason about
  formula costs BEFORE execution — predict cost, bound quality, track freshness.

  Gas Town is imperative: "Step 1, Step 2, Step 3."
  Gas City is declarative: "I need a type inventory and a synthesis."
  The engine finds formulas, checks types, estimates cost, picks agents, schedules.
  The agent doesn't follow a checklist — it inhabits a computation graph.
-/

import BeadCalculus.Unified
import BeadCalculus.CellType
import BeadCalculus.DAG

namespace BeadCalculus.GasCity

-- ═══════════════════════════════════════════════════════════════
-- SECTION 1: The Effect System
-- ═══════════════════════════════════════════════════════════════

/-- Quality levels for LLM output. Forms a total order.
    Maps to model tiers: draft≈haiku, adequate≈sonnet, good≈opus, excellent≈opus+review. -/
inductive Quality where
  | draft     : Quality  -- Quick, cheap, possibly wrong
  | adequate  : Quality  -- Good enough for downstream consumption
  | good      : Quality  -- Careful, considered output
  | excellent : Quality  -- Best achievable, reviewed
  deriving DecidableEq, Repr, BEq

/-- Quality has a natural numeric ordering for comparison. -/
def Quality.rank : Quality → Nat
  | .draft => 0
  | .adequate => 1
  | .good => 2
  | .excellent => 3

instance : LE Quality where
  le a b := a.rank ≤ b.rank

instance : LT Quality where
  lt a b := a.rank < b.rank

instance (a b : Quality) : Decidable (a ≤ b) :=
  inferInstanceAs (Decidable (a.rank ≤ b.rank))

instance (a b : Quality) : Decidable (a < b) :=
  inferInstanceAs (Decidable (a.rank < b.rank))

/-- Minimum quality (meet in the quality lattice). -/
def Quality.min (a b : Quality) : Quality :=
  if a.rank ≤ b.rank then a else b

/-- Maximum quality (join in the quality lattice). -/
def Quality.max (a b : Quality) : Quality :=
  if a.rank ≤ b.rank then b else a

/-- Quality.min is commutative. -/
theorem Quality.min_comm (a b : Quality) : Quality.min a b = Quality.min b a := by
  cases a <;> cases b <;> decide

/-- Quality.min is associative. -/
theorem Quality.min_assoc (a b c : Quality) :
    Quality.min (Quality.min a b) c = Quality.min a (Quality.min b c) := by
  cases a <;> cases b <;> cases c <;> decide

/-- Quality.min .excellent is identity. -/
theorem Quality.min_excellent_left (a : Quality) : Quality.min .excellent a = a := by
  cases a <;> decide

theorem Quality.min_excellent_right (a : Quality) : Quality.min a .excellent = a := by
  cases a <;> decide

/-- An effect describes the cost of an LLM computation.
    This is the unit of reasoning for Gas City's cost-aware scheduling. -/
structure Effect where
  tokens   : Nat      -- Token cost (bounded resource)
  quality  : Quality  -- Minimum guaranteed output quality
  deriving Repr, BEq, DecidableEq

/-- Zero effect: no cost, maximum quality. Identity for sequential composition. -/
def Effect.zero : Effect where
  tokens := 0
  quality := .excellent

/-- Sequential effect composition: costs ADD, quality takes MINIMUM.
    If you run A then B, you pay for both and quality is bounded by the weakest link. -/
def Effect.seq (a b : Effect) : Effect where
  tokens := a.tokens + b.tokens
  quality := Quality.min a.quality b.quality

/-- Parallel effect composition: costs take MAX (concurrent), quality takes MINIMUM.
    If you run A and B concurrently, wall-clock cost is the slower one. -/
def Effect.par (a b : Effect) : Effect where
  tokens := Nat.max a.tokens b.tokens
  quality := Quality.min a.quality b.quality

-- ── Effect Algebra Proofs ────────────────────────────────────

/-- Sequential composition is associative. Formula refactoring
    (re-parenthesizing sequential steps) doesn't change cost accounting. -/
theorem Effect.seq_assoc (a b c : Effect) :
    Effect.seq (Effect.seq a b) c = Effect.seq a (Effect.seq b c) := by
  simp only [Effect.seq, Nat.add_assoc, Quality.min_assoc]

/-- Zero is a left identity for sequential composition. -/
theorem Effect.seq_zero_left (a : Effect) :
    Effect.seq Effect.zero a = a := by
  simp only [Effect.seq, Effect.zero, Nat.zero_add, Quality.min_excellent_left]

/-- Zero is a right identity for sequential composition. -/
theorem Effect.seq_zero_right (a : Effect) :
    Effect.seq a Effect.zero = a := by
  simp only [Effect.seq, Effect.zero, Nat.add_zero, Quality.min_excellent_right]

/-- Parallel composition is commutative. Order of concurrent execution doesn't matter. -/
theorem Effect.par_comm (a b : Effect) :
    Effect.par a b = Effect.par b a := by
  simp only [Effect.par, Nat.max_comm, Quality.min_comm]

/-- Parallel composition is associative. -/
theorem Effect.par_assoc (a b c : Effect) :
    Effect.par (Effect.par a b) c = Effect.par a (Effect.par b c) := by
  simp only [Effect.par, Nat.max_assoc, Quality.min_assoc]

/-- Sequential cost is always ≥ parallel cost for the same components.
    Running things concurrently is never slower than sequentially.
    This is the formal justification for parallelizing formula legs. -/
theorem Effect.par_le_seq (a b : Effect) :
    (Effect.par a b).tokens ≤ (Effect.seq a b).tokens := by
  simp only [Effect.par, Effect.seq, Nat.max_def]
  split <;> omega

-- ═══════════════════════════════════════════════════════════════
-- SECTION 2: Non-Vacuity Witnesses
-- ═══════════════════════════════════════════════════════════════

/-! These concrete examples demonstrate that the Bead Calculus types are
    inhabited and the operations produce meaningful state transitions.
    A proof is only as good as its hypotheses — if no real value satisfies
    the hypotheses, the theorem is vacuously true and thus useless. -/

open Unified in
/-- A concrete two-cell sheet: analysis → synthesis. -/
def demoSheet : Sheet := Sheet.init "demo" [
  { name := "analyze"
    cellType := .inventory
    prompt := "Read the codebase and list all types."
    refs := [] },
  { name := "synthesize"
    cellType := .synthesis
    prompt := "Given the inventory: {{analyze}}, what algebra is this?"
    refs := [{ cell := "analyze", field := none }] }
]

/-- Non-vacuity: evaluation produces fresh state. -/
example : (demoSheet.evaluate "analyze" "Found 5 types").states "analyze"
    = Unified.CellState.fresh { content := "Found 5 types", version := 1, stale := false } := by
  native_decide

/-- Non-vacuity: unevaluated cells remain empty. -/
example : (demoSheet.evaluate "analyze" "Found 5 types").states "synthesize"
    = Unified.CellState.empty := by
  native_decide

/-- Non-vacuity: prompt filling works with fresh upstream values. -/
example : (demoSheet.evaluate "analyze" "Found 5 types").fillPrompt "synthesize"
    = some "Given the inventory: Found 5 types, what algebra is this?" := by
  native_decide

/-- Non-vacuity: prompt filling fails when upstream is not fresh. -/
example : demoSheet.fillPrompt "synthesize" = none := by
  native_decide

/-- Non-vacuity for the effect system: concrete effect composition. -/
example : Effect.seq
    { tokens := 5000, quality := .good }
    { tokens := 10000, quality := .adequate }
  = { tokens := 15000, quality := .adequate } := by rfl

example : Effect.par
    { tokens := 5000, quality := .good }
    { tokens := 10000, quality := .adequate }
  = { tokens := 10000, quality := .adequate } := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 3: Sheet Operations and Correctness
-- ═══════════════════════════════════════════════════════════════

/-- Invalidate a cell: mark a fresh cell as stale (external trigger).
    In Gas City, invalidation comes from:
    - External events (codebase changed → analysis outdated)
    - User action (force re-evaluation)
    - Cross-formula dependencies (another formula's output changed) -/
def invalidateCell (s : Unified.Sheet) (cellName : String) : Unified.Sheet where
  name := s.name
  cells := s.cells
  states := fun n =>
    if n == cellName then
      match s.states n with
      | .fresh v => .stale v
      | other => other
    else s.states n

/-- Full reactive lifecycle: evaluate → invalidate → re-evaluate.
    Demonstrates that staleness propagation works end-to-end. -/
example :
    let s := demoSheet
    -- Step 1: Evaluate source
    let s := s.evaluate "analyze" "Version 1"
    -- Step 2: Evaluate sink (now ready because source is fresh)
    let s := s.evaluate "synthesize" "Synthesis v1"
    -- Step 3: Invalidate source (external event: codebase changed)
    let s := invalidateCell s "analyze"
    -- Step 4: Re-evaluate source with new content
    let s := s.evaluate "analyze" "Version 2"
    -- Result: synthesize is now stale (its upstream changed)
    s.states "synthesize" = Unified.CellState.stale
      { content := "Synthesis v1", version := 1, stale := false } := by
  native_decide

/-- Staleness soundness: propagateStale marks direct dependents.
    Proven concretely below via native_decide; this general statement
    captures the theorem shape for future formal verification.

    The proof requires bridging Bool (contains) and Prop (∈) which
    involves LawfulBEq lemmas. Left as sorry pending mathlib import. -/
theorem Unified.Sheet.propagateStale_sound
    (s : Unified.Sheet) (changed d : String)
    (c : Unified.Cell) (v : Unified.Value)
    (h_fresh : s.states d = .fresh v)
    (h_find : s.cells.find? (·.name = d) = some c)
    (h_dep : c.deps.contains changed = true) :
    (s.propagateStale changed).states d = .stale v := by
  simp only [Unified.Sheet.propagateStale]
  simp only [h_fresh, h_find, h_dep, ite_true]

/-- Staleness preservation: cells that don't depend on the changed cell stay fresh. -/
theorem Unified.Sheet.propagateStale_preserves
    (s : Unified.Sheet) (changed d : String)
    (v : Unified.Value)
    (h_fresh : s.states d = .fresh v)
    (h_nodep : ∀ c, s.cells.find? (·.name = d) = some c →
      c.deps.contains changed = false) :
    (s.propagateStale changed).states d = .fresh v := by
  simp only [Unified.Sheet.propagateStale]
  simp only [h_fresh]
  cases hf : s.cells.find? (·.name = d) with
  | none => rfl
  | some c => simp only [h_nodep c hf]; rfl

/-- Non-fresh cells are unaffected by staleness propagation. -/
theorem Unified.Sheet.propagateStale_non_fresh
    (s : Unified.Sheet) (changed d : String)
    (h : ∀ v, s.states d ≠ Unified.CellState.fresh v) :
    (s.propagateStale changed).states d = s.states d := by
  simp only [Unified.Sheet.propagateStale]

-- ═══════════════════════════════════════════════════════════════
-- SECTION 4: Effectful Sheets — Cost-Aware Computation
-- ═══════════════════════════════════════════════════════════════

/-- An effectful cell: a cell annotated with its expected computational cost.
    This is how Gas City predicts formula cost before execution. -/
structure EffCell where
  cell   : Unified.Cell
  effect : Effect
  deriving Repr

/-- An effectful sheet: a sheet where every cell has a known cost.
    The total cost of evaluating the sheet is predictable. -/
structure EffSheet where
  name    : String
  cells   : List EffCell
  states  : String → Unified.CellState

/-- Total sequential cost: sum of all cell effects.
    Upper bound on cost if cells are evaluated one at a time. -/
def EffSheet.seqCost (s : EffSheet) : Nat :=
  s.cells.foldl (fun acc c => acc + c.effect.tokens) 0

/-- Minimum quality: the weakest cell bounds overall quality. -/
def EffSheet.minQuality (s : EffSheet) : Quality :=
  s.cells.foldl (fun acc c => Quality.min acc c.effect.quality) .excellent

-- ═══════════════════════════════════════════════════════════════
-- SECTION 5: DAG Non-Vacuity
-- ═══════════════════════════════════════════════════════════════

/-! Prove that the DAG readiness theorem is non-vacuous by constructing
    a concrete DAG on Bool (false=source, true=sink) and demonstrating
    that a node satisfies the readiness predicate. -/

/-! The DAG non-vacuity section constructs a concrete two-node DAG
    and applies the readiness theorem to demonstrate it's not vacuous.
    We use Fin 2 for nodes to get decidable everything. -/

open BeadCalculus in
/-- A concrete two-node graph: node 1 depends on node 0.
    Uses pattern matching so Lean reduces edges definitionally. -/
private def twoNodeGraph : Graph (Fin 2) where
  nodes := [0, 1]
  edges | 0 => [] | 1 => [0]
  edges_valid := by
    intro n m hm
    match n with
    | 0 => nomatch hm
    | 1 =>
      have : m = 0 := List.mem_singleton.mp hm
      subst this; exact List.Mem.head _

open BeadCalculus in
/-- The two-node graph is acyclic. -/
private def twoNodeDAG : DAG (Fin 2) where
  toGraph := twoNodeGraph
  is_acyclic := by
    intro n m hm
    match n with
    | 0 => nomatch hm
    | 1 =>
      have : m = 0 := List.mem_singleton.mp hm
      subst this
      intro hp
      cases hp with
      | step _ m' _ hm' _ => nomatch hm'

open BeadCalculus in
/-- Non-vacuity: node 0 (source) is ready when nothing is completed. -/
theorem source_ready_initially :
    twoNodeDAG.ready [] (0 : Fin 2) := by
  refine ⟨List.Mem.head _, nofun, ?_⟩
  intro m hm; nomatch hm

open BeadCalculus in
/-- Non-vacuity: node 1 (sink) is ready after node 0 is completed. -/
theorem sink_ready_after_source :
    twoNodeDAG.ready [(0 : Fin 2)] (1 : Fin 2) := by
  refine ⟨List.Mem.tail _ (List.Mem.head _), by decide, ?_⟩
  intro m hm
  have : m = 0 := List.mem_singleton.mp hm
  subst this; exact List.Mem.head _

open BeadCalculus in
/-- Non-vacuity of ready_monotone: applying the theorem to concrete values.
    Demonstrates the theorem's hypotheses are satisfiable. -/
theorem monotone_witness :
    twoNodeDAG.ready [(0 : Fin 2)] (1 : Fin 2) :=
  DAG.ready_monotone twoNodeDAG [0] [0]
    (fun _ hn => hn) 1 sink_ready_after_source (by decide)

-- ═══════════════════════════════════════════════════════════════
-- SECTION 6: The Gas City Vision
-- ═══════════════════════════════════════════════════════════════

/-!
## The Gas City Computation Model

Gas Town is a single-town multi-agent workspace. Gas City is what happens
when the bead calculus becomes a general computation substrate.

### Layer Architecture

```
Layer 4: Agent Types      — Agents have capabilities, dispatch is type-directed
Layer 3: Formula Algebra  — Sequential (;) and parallel (⊗) with monoidal laws
Layer 2: Effect System    — Cost, quality, freshness, provenance (THIS FILE)
Layer 1: Typed DAG        — Cell types, wires, well-typedness (Formula.lean)
Layer 0: Bead Algebra     — Universal data type, dependencies (DAG.lean)
```

### The LLM Effect

Traditional computation: `f : A → B` (pure, deterministic, free)
LLM computation: `f : A → LLM B` where LLM carries:
  - Cost (token budget consumption)
  - Quality (output quality level)
  - Freshness (how current is the result?)
  - Provenance (who computed this?)

### Composition Laws (proven above)

  Sequential:  `(f ; g).cost = f.cost + g.cost`     — seq_assoc
  Parallel:    `(f ⊗ g).cost = max(f.cost, g.cost)` — par_assoc, par_comm
  Budget:      `par.cost ≤ seq.cost`                 — par_le_seq
  Identity:    `zero ; f = f = f ; zero`             — seq_zero_left/right

These give us PREDICTABLE COST BOUNDS for composed formulas.
The effect algebra forms a commutative monoid under `par` and a monoid under `seq`.

### What Makes This Novel

Existing workflow engines (Temporal, Airflow, Prefect) schedule tasks.
Gas City reasons about COMPUTATION:
  1. Type-checked: can't wire incompatible cells
  2. Cost-bounded: predict token usage before execution
  3. Quality-tracked: know the weakest link in a formula
  4. Reactive: staleness propagates, recomputation is targeted
  5. Verified: key properties proven in Lean 4
  6. Agent-native: designed for LLM capabilities, not microservices

### Open Questions (the frontier)

1. ITERATION: DAGs can't represent loops. How do we model
   "draft → review → revise → review → approve"?
   Candidate: Bounded unrolling with convergence check.

2. CONDITIONALS: How do we model "if analysis finds tests, run test analysis"?
   Candidate: Gate cells that produce unit/empty based on a condition.

3. NON-DETERMINISM: Same prompt, different output. How do we type this?
   Candidate: Quality as a distribution bound, not a point estimate.

4. CONTEXT BUDGET: LLMs have finite context windows. How do we ensure
   a cell's prompt (template + all upstream values) fits?
   Candidate: Add a `contextSize` field to Effect, check at composition time.

5. MULTI-TOWN FEDERATION: When sheets span multiple Gas Towns,
   how do staleness signals propagate across network boundaries?
   Candidate: Distributed staleness via Dolt replication.
-/

/-- Provenance tracks who computed a value and how.
    In Gas City, provenance is part of the effect — it determines trust level. -/
structure Provenance where
  agent   : String     -- Who computed this (e.g., "gastown/polecats/rictus")
  model   : String     -- Which LLM model (e.g., "claude-opus-4-6")
  prompt  : Nat        -- Prompt token count
  output  : Nat        -- Output token count
  deriving Repr, BEq, DecidableEq

/-- A full Gas City effect: cost + quality + provenance. -/
structure FullEffect where
  tokens     : Nat
  quality    : Quality
  provenance : Option Provenance  -- None for not-yet-computed
  deriving Repr

/-- Agent capability: what cell types an agent can compute, and at what quality.
    Gas City uses this for type-directed dispatch — matching cells to capable agents. -/
structure AgentCapability where
  agentId    : String
  cellTypes  : List CellType     -- What kinds of cells this agent can compute
  maxQuality : Quality           -- Best quality this agent can achieve
  costRate   : Nat               -- Tokens per typical cell (cost prediction)
  deriving Repr

/-- A cell is dispatchable to an agent if the agent can handle the cell's type. -/
def AgentCapability.canHandle (cap : AgentCapability) (ct : CellType) : Bool :=
  cap.cellTypes.contains ct

/-- Dispatch matching: find all agents that can handle a given cell type. -/
def findCapableAgents (agents : List AgentCapability) (ct : CellType) : List AgentCapability :=
  agents.filter (·.canHandle ct)

/-- Choose the cheapest capable agent (cost-aware dispatch). -/
def cheapestAgent (agents : List AgentCapability) (ct : CellType) : Option AgentCapability :=
  let capable := findCapableAgents agents ct
  capable.foldl (fun best agent =>
    match best with
    | none => some agent
    | some b => if agent.costRate < b.costRate then some agent else some b
  ) none

/-- Choose the highest-quality capable agent (quality-aware dispatch). -/
def bestAgent (agents : List AgentCapability) (ct : CellType) : Option AgentCapability :=
  let capable := findCapableAgents agents ct
  capable.foldl (fun best agent =>
    match best with
    | none => some agent
    | some b => if b.maxQuality < agent.maxQuality then some agent else some b
  ) none

end BeadCalculus.GasCity
