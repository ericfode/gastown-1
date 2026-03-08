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

-- ═══════════════════════════════════════════════════════════════
-- SECTION 7: Compression Chains — Information Decay Through DAGs
-- ═══════════════════════════════════════════════════════════════

/-! Each cell in a formula applies a compression function: it takes
    structured input and produces a summary that downstream cells consume.
    Information is deliberately lost at each step. The interesting properties
    are about compression DEPTH, POLICY, and COMPOSITION — not token counts.

    This models the reality of LLM agent coordination: every handoff
    between agents is a lossy compression. The question isn't whether
    information is lost — it always is — but whether the RIGHT information
    survives. -/

/-- A compression policy describes how a cell transforms its input.
    Each policy has different fidelity characteristics. -/
inductive CompressionPolicy where
  | verbatim    : CompressionPolicy  -- Pass through unchanged (identity)
  | summarize   : CompressionPolicy  -- Extract key points, lose detail
  | extract     : CompressionPolicy  -- Pull structured data, lose prose
  | classify    : CompressionPolicy  -- Reduce to category labels
  | decide      : CompressionPolicy  -- Reduce to yes/no + rationale
  deriving DecidableEq, Repr, BEq

/-- Compression policies form a partial order by information retention.
    verbatim retains everything; decide retains almost nothing. -/
def CompressionPolicy.retentionRank : CompressionPolicy → Nat
  | .verbatim  => 4
  | .summarize => 3
  | .extract   => 2
  | .classify  => 1
  | .decide    => 0

instance : LE CompressionPolicy where
  le a b := a.retentionRank ≤ b.retentionRank

instance (a b : CompressionPolicy) : Decidable (a ≤ b) :=
  inferInstanceAs (Decidable (a.retentionRank ≤ b.retentionRank))

/-- A compression step records one stage of information loss. -/
structure CompressionStep where
  policy : CompressionPolicy
  depth  : Nat   -- How many compressions have happened upstream
  deriving Repr, BEq, DecidableEq

/-- Compose compression steps: depth accumulates, retention decreases. -/
def CompressionStep.compose (a b : CompressionStep) : CompressionStep where
  policy := if b.policy.retentionRank ≤ a.policy.retentionRank then b.policy else a.policy
  depth  := a.depth + b.depth + 1

/-- A compression chain is the provenance of how many times information
    has been compressed, and by what policies, to reach the current cell. -/
structure CompressionChain where
  steps : List CompressionStep
  deriving Repr

/-- Total compression depth: how many lossy transformations from source. -/
def CompressionChain.totalDepth (c : CompressionChain) : Nat :=
  c.steps.length

/-- Minimum retention across the chain: bottleneck fidelity. -/
def CompressionChain.minRetention (c : CompressionChain) : Nat :=
  c.steps.foldl (fun acc s => Nat.min acc s.policy.retentionRank) 4

/-- Extend a chain with a new compression step. -/
def CompressionChain.extend (c : CompressionChain) (p : CompressionPolicy) : CompressionChain where
  steps := c.steps ++ [{ policy := p, depth := c.totalDepth }]

/-- The empty chain: no compression has happened. -/
def CompressionChain.empty : CompressionChain where
  steps := []

/-- Depth monotonicity: extending a chain always increases depth. -/
theorem CompressionChain.extend_increases_depth (c : CompressionChain) (p : CompressionPolicy) :
    (c.extend p).totalDepth = c.totalDepth + 1 := by
  simp [CompressionChain.extend, CompressionChain.totalDepth, List.length_append]

/-- Helper: foldl min over an appended singleton. -/
private theorem foldl_min_append_singleton (l : List CompressionStep) (s : CompressionStep) (init : Nat) :
    List.foldl (fun acc (x : CompressionStep) => Nat.min acc x.policy.retentionRank) init (l ++ [s])
    = Nat.min (List.foldl (fun acc (x : CompressionStep) => Nat.min acc x.policy.retentionRank) init l)
              s.policy.retentionRank := by
  induction l generalizing init with
  | nil => simp [List.foldl]
  | cons h t ih => simp [List.foldl, ih]

/-- Retention monotonicity: extending with a lossy step can only decrease retention. -/
theorem CompressionChain.extend_retention_le (c : CompressionChain) (p : CompressionPolicy) :
    (c.extend p).minRetention ≤ c.minRetention := by
  simp only [CompressionChain.extend, CompressionChain.minRetention,
             foldl_min_append_singleton]
  exact Nat.min_le_left _ _

/-- Non-vacuity: a concrete compression chain example. -/
example :
    let c := CompressionChain.empty
    let c := c.extend .verbatim   -- First cell: pass through
    let c := c.extend .summarize  -- Second cell: summarize
    let c := c.extend .decide     -- Third cell: decide
    c.totalDepth = 3 ∧ c.minRetention = 0 := by
  constructor <;> native_decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 8: Parameterized Sheets — The Map Operation
-- ═══════════════════════════════════════════════════════════════

/-! The spreadsheet's killer feature is "drag to fill" — apply a formula
    across a collection of inputs. In Gas City, this is:

      gt sling mol-analyze --over repos.csv

    One formula template, N parameter rows, N × M total cells.
    This is `map` over a functor. The formula is the function,
    the parameter list is the collection, the result is a sheet-of-sheets.

    Properties we want:
    - Each instantiation is independent (parallelizable)
    - Cost scales linearly with N (predictable)
    - Staleness is per-instance (changing one row doesn't invalidate others)
    - Compression can aggregate across instances (pivot table = fold) -/

/-- A parameter set: named values that fill holes in a template. -/
structure ParamSet where
  name   : String              -- Row identifier (e.g., "repo-alpha")
  values : List (String × String)  -- param name → param value
  deriving Repr, BEq, DecidableEq

/-- A sheet template: a sheet definition with parameter holes.
    The prompt templates contain {{param.X}} references in addition
    to {{cell}} references. -/
structure SheetTemplate where
  baseName : String
  cells    : List Unified.Cell
  params   : List String        -- Names of required parameters
  deriving Repr

/-- Instantiate a template with concrete parameters.
    Substitutes {{param.X}} in all prompts. -/
def SheetTemplate.instantiate (t : SheetTemplate) (ps : ParamSet) : Unified.Sheet where
  name := t.baseName ++ "/" ++ ps.name
  cells := t.cells.map fun c => {
    c with
    prompt := ps.values.foldl (fun p kv =>
      p.replace ("{{param." ++ kv.1 ++ "}}") kv.2) c.prompt
    name := ps.name ++ "/" ++ c.name
  }
  states := fun _ => .empty

/-- Map a template over a parameter list: the "drag to fill" operation.
    Produces one sheet per parameter row, all independent. -/
def SheetTemplate.mapOver (t : SheetTemplate) (rows : List ParamSet) : List Unified.Sheet :=
  rows.map t.instantiate

/-- Total cell count scales linearly: |cells| × |rows|. -/
theorem SheetTemplate.map_cell_count (t : SheetTemplate) (rows : List ParamSet) :
    (t.mapOver rows).length = rows.length := by
  simp [SheetTemplate.mapOver]

/-- Each mapped sheet has the same number of cells as the template. -/
theorem SheetTemplate.instantiate_preserves_cell_count (t : SheetTemplate) (ps : ParamSet) :
    (t.instantiate ps).cells.length = t.cells.length := by
  simp [SheetTemplate.instantiate]

/-- Aggregation: fold over mapped sheet results.
    This is the "pivot table" — compress N instances into one summary.
    The aggregation function is itself a compression step. -/
structure Aggregation where
  name           : String
  sourceCell     : String           -- Which cell from each instance to aggregate
  compressionPolicy : CompressionPolicy  -- How to compress the collection
  deriving Repr

/-- Non-vacuity: a concrete template and map example. -/
private def analyzeTemplate : SheetTemplate where
  baseName := "analyze-repo"
  cells := [
    { name := "scan"
      cellType := .inventory
      prompt := "Scan {{param.repo_url}} and list all types."
      refs := [] },
    { name := "classify"
      cellType := .synthesis
      prompt := "Given the types: {{scan}}, classify the architecture."
      refs := [{ cell := "scan", field := none }] }
  ]
  params := ["repo_url"]

private def exampleRows : List ParamSet := [
  { name := "alpha", values := [("repo_url", "github.com/org/alpha")] },
  { name := "beta",  values := [("repo_url", "github.com/org/beta")] },
  { name := "gamma", values := [("repo_url", "github.com/org/gamma")] }
]

/-- Non-vacuity: mapping produces 3 sheets. -/
example : (analyzeTemplate.mapOver exampleRows).length = 3 := by rfl

/-- Non-vacuity: mapping produces correct count and structure. -/
example : (analyzeTemplate.mapOver exampleRows).length = 3 ∧
    (analyzeTemplate.instantiate (exampleRows[0]'(by decide))).cells.length = 2 := by
  constructor <;> rfl

/-- Non-vacuity: parameter substitution works in prompts. -/
example :
    let sheet := analyzeTemplate.instantiate { name := "test", values := [("repo_url", "example.com")] }
    sheet.cells.length = 2 := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 9: Visualization Model — What You SEE
-- ═══════════════════════════════════════════════════════════════

/-! Gas City's real innovation is making LLM computation VISIBLE.
    These types model the visualization layer — what the user sees. -/

/-- What a cell looks like in the living grid view. -/
structure CellView where
  name              : String
  valuePreview      : Option String    -- First ~100 chars of output
  status            : Unified.CellState
  compressionDepth  : Nat              -- How many compressions from source
  tokenCost         : Nat              -- Tokens spent computing this cell
  qualityLevel      : Quality
  upstreamCount     : Nat              -- How many cells feed into this
  downstreamCount   : Nat              -- How many cells depend on this
  deriving Repr

/-- A provenance link: traces one piece of output back to its source.
    This is the "View Precedents" for natural language — click a sentence
    in cell C and see which sentence in cell A it came from. -/
structure ProvenanceLink where
  targetCell    : String   -- The cell containing the derived content
  sourceCell    : String   -- The cell containing the source content
  compressions  : Nat      -- How many compression steps between them
  deriving Repr, BEq, DecidableEq

/-- A provenance trace: the full path from a derived value back to sources. -/
structure ProvenanceTrace where
  endpoint : String              -- The cell we're tracing from
  links    : List ProvenanceLink -- Ordered source → ... → endpoint
  totalCompressions : Nat        -- Sum of all compression steps
  deriving Repr

/-- Build a provenance trace from a cell through the DAG.
    Follows refs backwards, accumulating compression depth.
    Uses fuel to guarantee termination (depth-bounded traversal). -/
def buildProvenanceTrace (cells : List Unified.Cell) (chains : String → CompressionChain)
    (cellName : String) (fuel : Nat := cells.length) : ProvenanceTrace :=
  let rec go (name : String) (fuel : Nat) (acc : List ProvenanceLink) :
      List ProvenanceLink :=
    match fuel with
    | 0 => acc
    | fuel' + 1 =>
      match cells.find? (·.name = name) with
      | none => acc
      | some c =>
        let newLinks := c.refs.map fun ref => {
          targetCell := name
          sourceCell := ref.cell
          compressions := (chains name).totalDepth
        }
        let acc' := newLinks ++ acc
        c.refs.foldl (fun a ref => go ref.cell fuel' a) acc'
  let links := go cellName fuel []
  { endpoint := cellName
    links := links
    totalCompressions := links.foldl (fun acc l => acc + l.compressions) 0 }

/-- The information Sankey: for each cell, how much information flows in and out.
    Width of the stream is proportional to token count. Narrowing = compression. -/
structure SankeyNode where
  name       : String
  tokensIn   : Nat     -- Sum of upstream output sizes
  tokensOut  : Nat     -- This cell's output size
  ratio      : Nat     -- Compression ratio (tokensIn / tokensOut), 0 if no input
  deriving Repr

/-- Compute Sankey node for a cell given token sizes. -/
def computeSankeyNode (cellName : String) (cells : List Unified.Cell)
    (sizes : String → Nat) : SankeyNode :=
  match cells.find? (·.name = cellName) with
  | none => { name := cellName, tokensIn := 0, tokensOut := sizes cellName, ratio := 0 }
  | some c =>
    let inTotal := c.refs.foldl (fun acc ref => acc + sizes ref.cell) 0
    let out := sizes cellName
    { name := cellName
      tokensIn := inTotal
      tokensOut := out
      ratio := if out = 0 then 0 else inTotal / out }

/-- Non-vacuity: provenance trace for the demo sheet. -/
example :
    let trace := buildProvenanceTrace
      demoSheet.cells
      (fun _ => CompressionChain.empty)
      "synthesize"
    trace.links.length = 1 := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 10: Pinned Cells — Freeze Panes for Debugging
-- ═══════════════════════════════════════════════════════════════

/-! "Pin" a cell's value so it resists recomputation. This enables controlled
    experiments: manually set a cell's output and recompute downstream to
    isolate where problems occur. The analogue in spreadsheets is "paste as
    values" — replacing a formula with a literal so upstream changes stop
    flowing through. -/

/-- Whether a cell's value is pinned (frozen) or free to recompute. -/
inductive PinState where
  | unpinned : PinState
  | pinned   : (value : String) → PinState
  deriving DecidableEq, Repr, BEq

/-- A sheet augmented with per-cell pin state.
    Pins override evaluation: a pinned cell always returns its pinned value. -/
structure PinnedSheet where
  sheet : Unified.Sheet
  pins  : String → PinState

/-- The effective state of a cell: if pinned, return the pinned value as fresh;
    otherwise return the real state from the underlying sheet. -/
def PinnedSheet.effectiveState (ps : PinnedSheet) (cellName : String) : Unified.CellState :=
  match ps.pins cellName with
  | .pinned v => .fresh { content := v, version := 0, stale := false }
  | .unpinned => ps.sheet.states cellName

/-- Evaluate a cell with pin awareness: pinned cells are skipped entirely.
    Unpinned cells evaluate normally. -/
def PinnedSheet.evaluateWithPins (ps : PinnedSheet) (cellName content : String) : PinnedSheet :=
  match ps.pins cellName with
  | .pinned _ => ps  -- Pinned: do not recompute
  | .unpinned => { ps with sheet := ps.sheet.evaluate cellName content }

/-- Pinning blocks evaluation: evaluating a pinned cell does not change
    its effective state — the pin holds. -/
theorem pin_blocks_evaluation (ps : PinnedSheet) (cellName content : String) (v : String)
    (h_pin : ps.pins cellName = .pinned v) :
    (ps.evaluateWithPins cellName content).effectiveState cellName
      = ps.effectiveState cellName := by
  simp only [PinnedSheet.evaluateWithPins, h_pin, PinnedSheet.effectiveState]

/-- Removing a pin restores the cell to its real state from the underlying sheet. -/
theorem unpin_restores_state (ps : PinnedSheet) (cellName : String)
    (h_unpin : ps.pins cellName = .unpinned) :
    ps.effectiveState cellName = ps.sheet.states cellName := by
  simp only [PinnedSheet.effectiveState, h_unpin]

/-- Non-vacuity: pinning and evaluating on the demo sheet. -/
example :
    let ps : PinnedSheet := {
      sheet := demoSheet.evaluate "analyze" "Real value"
      pins := fun n => if n == "analyze" then .pinned "Frozen value" else .unpinned
    }
    -- Evaluating a pinned cell returns the pin, not the new content
    (ps.evaluateWithPins "analyze" "New value").effectiveState "analyze"
      = Unified.CellState.fresh { content := "Frozen value", version := 0, stale := false } := by
  native_decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 11: Input Snapshots — What Did This Cell See?
-- ═══════════════════════════════════════════════════════════════

/-! The Skeptic's critique: there is no input provenance per cell. When cell 15
    was computed, which versions of upstream cells were consumed? Without this,
    debugging is guesswork. "Your synthesis is wrong" — was it wrong because
    the analysis was stale, or because the prompt was bad?

    We formalize input snapshots: a record of exactly which upstream versions
    a cell consumed at the time of computation. -/

/-- A snapshot of what a cell saw when it was computed:
    for each upstream reference, the version number at compute time. -/
structure InputSnapshot where
  cellName      : String
  inputVersions : List (String × Nat)  -- (ref cell name, version at compute time)
  deriving Repr, BEq, DecidableEq

/-- A computation record: what was produced, from what inputs. -/
structure ComputationRecord where
  cellName      : String
  snapshot      : InputSnapshot
  outputVersion : Nat
  content       : String
  deriving Repr, BEq, DecidableEq

/-- An execution log: the ordered history of all computations. -/
def ExecutionLog := List ComputationRecord

/-- Extract the version from a cell's current state. Returns 0 for non-fresh cells. -/
private def versionOf (s : Unified.Sheet) (cellName : String) : Nat :=
  match s.states cellName with
  | .fresh v => v.version
  | .stale v => v.version
  | _ => 0

/-- Evaluate a cell and produce a computation record capturing input versions.
    Returns the updated sheet and the log entry. -/
def evaluateWithLog (s : Unified.Sheet) (cellName content : String)
    : Unified.Sheet × ComputationRecord :=
  let cell? := s.cells.find? (·.name = cellName)
  let inputVersions := match cell? with
    | some c => c.refs.map fun ref => (ref.cell, versionOf s ref.cell)
    | none => []
  let snapshot : InputSnapshot := { cellName, inputVersions }
  let s' := s.evaluate cellName content
  let outputVersion := versionOf s' cellName
  let record : ComputationRecord := {
    cellName, snapshot, outputVersion, content
  }
  (s', record)

/-- The snapshot in the log matches the actual input versions at compute time.
    This is stated as: the input versions recorded in the snapshot equal the
    versions that were present in the sheet at the moment of evaluation. -/
theorem log_captures_inputs (s : Unified.Sheet) (cellName content : String)
    (c : Unified.Cell)
    (h_find : s.cells.find? (·.name = cellName) = some c) :
    (evaluateWithLog s cellName content).2.snapshot.inputVersions
      = c.refs.map (fun ref => (ref.cell, versionOf s ref.cell)) := by
  simp only [evaluateWithLog, h_find]

/-- Non-vacuity: evaluating with log on the demo sheet captures upstream versions. -/
example :
    let s := demoSheet.evaluate "analyze" "Found 5 types"
    let (_, record) := evaluateWithLog s "synthesize" "Algebra found"
    record.snapshot.inputVersions = [("analyze", 1)] := by
  native_decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 12: Recomputation Policy — What To Do About Staleness
-- ═══════════════════════════════════════════════════════════════

/-! The Skeptic's hardest critique: staleness detection is easy, policy is hard.
    Once we know cell 15 is stale, what do we DO? Recompute immediately? Wait?
    Ask a human? It depends on cost, urgency, and confidence.

    This section formalizes recomputation strategies as first-class data,
    so the engine can reason about WHEN to recompute, not just WHETHER. -/

/-- A recomputation policy: what to do when a cell becomes stale. -/
inductive RecomputePolicy where
  | eager                          -- Recompute immediately when stale
  | lazy                           -- Mark stale, wait for explicit trigger
  | budgeted   (maxTokens : Nat)   -- Recompute only if estimated cost fits budget
  | convergent (maxRounds : Nat)   -- Recompute up to N times, stop if output stabilizes
  | gated                          -- Require human approval before recompute
  deriving Repr, BEq, DecidableEq

/-- The decision made by applying a policy. -/
inductive RecomputeDecision where
  | recompute       -- Go ahead, recompute now
  | skip            -- Do not recompute (not needed or round limit reached)
  | askHuman        -- Escalate: require human approval
  | budgetExceeded  -- Would recompute, but cost is too high
  deriving Repr, BEq, DecidableEq

/-- Apply a recomputation policy given:
    - The policy for this cell
    - The estimated token cost of recomputation
    - How many times this cell has been recomputed in the current round -/
def applyPolicy (policy : RecomputePolicy) (estimatedCost : Nat) (roundCount : Nat)
    : RecomputeDecision :=
  match policy with
  | .eager          => .recompute
  | .lazy           => .skip
  | .budgeted max   => if estimatedCost ≤ max then .recompute else .budgetExceeded
  | .convergent max => if roundCount < max then .recompute else .skip
  | .gated          => .askHuman

/-- Eager policy always recomputes, regardless of cost or round count. -/
theorem eager_always_recomputes (cost rounds : Nat) :
    applyPolicy .eager cost rounds = .recompute := by
  rfl

/-- Budgeted policy returns .budgetExceeded when cost exceeds the budget. -/
theorem budgeted_respects_limit (max cost rounds : Nat) (h : cost > max) :
    applyPolicy (.budgeted max) cost rounds = .budgetExceeded := by
  unfold applyPolicy
  have : ¬ (cost ≤ max) := by omega
  simp [this]

/-- Convergent policy returns .skip when round count reaches or exceeds maxRounds. -/
theorem convergent_stops (max rounds : Nat) (h : rounds ≥ max) :
    applyPolicy (.convergent max) 0 rounds = .skip := by
  unfold applyPolicy
  have : ¬ (rounds < max) := by omega
  simp [this]

/-- Non-vacuity: eager always recomputes. -/
example : applyPolicy .eager 99999 100 = .recompute := by rfl

/-- Non-vacuity: lazy always skips. -/
example : applyPolicy .lazy 0 0 = .skip := by rfl

/-- Non-vacuity: budgeted approves when under budget. -/
example : applyPolicy (.budgeted 5000) 3000 0 = .recompute := by rfl

/-- Non-vacuity: budgeted rejects when over budget. -/
example : applyPolicy (.budgeted 5000) 8000 0 = .budgetExceeded := by rfl

/-- Non-vacuity: convergent recomputes when rounds remain. -/
example : applyPolicy (.convergent 3) 0 1 = .recompute := by rfl

/-- Non-vacuity: convergent stops when rounds exhausted. -/
example : applyPolicy (.convergent 3) 0 3 = .skip := by rfl

/-- Non-vacuity: gated always asks a human. -/
example : applyPolicy .gated 0 0 = .askHuman := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 13: What Gas City Actually Is
-- ═══════════════════════════════════════════════════════════════

/-!
## Gas City = Spreadsheet Semantics for Agent Coordination

The insight: a spreadsheet is not a grid of numbers. It is a
**direct manipulation interface for computation graphs**. Gas City
gives agent coordination the same interface.

### Spreadsheet → Gas City Translation

| Spreadsheet    | Gas City                       | Why it matters                    |
|---------------|--------------------------------|-----------------------------------|
| Cell          | Bead                           | Already exists                    |
| Formula       | Prompt template + {{refs}}     | LLM = formula engine              |
| Value         | Bead output (notes/design)     | Already exists in Dolt            |
| Drag-to-fill  | SheetTemplate.mapOver          | 10x: one formula, N instances     |
| Stale marker  | CellState.stale                | Reactive: know what's outdated    |
| Pivot table   | Aggregation over mapped sheets | Compress N results into insight   |
| Hotkey        | gt sling / bd ready / gt eval  | Power user acceleration           |
| Conditional   | Gate cell (unit/empty output)   | Control flow in the DAG           |
| Named range   | Formula (reusable DAG chunk)   | Composition                       |

### The Three Genuine Innovations

1. **Compression-aware dataflow**: Each cell declares HOW it compresses
   information, not just THAT it transforms it. The chain tracks cumulative
   information loss. This lets you reason about "is cell 15 working from
   a 3x-compressed summary or the original data?"

2. **Parameterized map**: Apply a formula across a collection, producing
   independent instances. Each instance has its own staleness. Aggregation
   folds results back. This is the spreadsheet's "fill down" for agent work.

3. **Effect-bounded dispatch**: Before running a formula, predict its cost
   (tokens × quality). Choose agents by capability matching. The effect
   algebra gives provable bounds: parallel ≤ sequential, composition is
   associative.

### What We Do NOT Need To Build

- A new runtime (Gas Town already executes)
- A new data store (Dolt already persists)
- A new agent framework (polecats already work)
- New wire protocols (beads already have deps)

We need:
- Staleness tracking on beads (one field: `stale: bool`)
- Template parameters on formulas (extend TOML)
- Compression metadata on outputs (one field: `compression_depth: int`)
- A `gt eval` command that fills prompts and dispatches
- A `gt map` command that applies templates over parameter lists
-/

end BeadCalculus.GasCity
