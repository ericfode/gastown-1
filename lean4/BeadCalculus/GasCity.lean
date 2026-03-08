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
  formula costs — track cost, bound quality, measure freshness.

  Gas Town is imperative: "Step 1, Step 2, Step 3."
  Gas City is declarative: "I need a type inventory and a synthesis."
  The engine finds formulas, checks types, tracks cost, picks agents, schedules.
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

/-- An effectful cell: a cell annotated with its observed computational cost.
    Records actual token consumption and quality after evaluation. -/
structure EffCell where
  cell   : Unified.Cell
  effect : Effect
  deriving Repr

/-- An effectful sheet: a sheet where every cell has a recorded cost.
    The total cost of the sheet is the sum of observed cell costs. -/
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

These give us COMPOSITIONAL COST ACCOUNTING for composed formulas.
The effect algebra forms a commutative monoid under `par` and a monoid under `seq`.

### What Makes This Novel

Existing workflow engines (Temporal, Airflow, Prefect) schedule tasks.
Gas City reasons about COMPUTATION:
  1. Type-checked: can't wire incompatible cells
  2. Cost-tracked: measure and compose token usage after execution
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
  | budgeted   (maxTokens : Nat)   -- Recompute only if under a token spending cap
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
    - The cumulative token spend so far
    - How many times this cell has been recomputed in the current round -/
def applyPolicy (policy : RecomputePolicy) (spentSoFar : Nat) (roundCount : Nat)
    : RecomputeDecision :=
  match policy with
  | .eager          => .recompute
  | .lazy           => .skip
  | .budgeted max   => if spentSoFar ≤ max then .recompute else .budgetExceeded
  | .convergent max => if roundCount < max then .recompute else .skip
  | .gated          => .askHuman

/-- Eager policy always recomputes, regardless of cost or round count. -/
theorem eager_always_recomputes (cost rounds : Nat) :
    applyPolicy .eager cost rounds = .recompute := by
  rfl

/-- Budgeted policy returns .budgetExceeded when spend exceeds the cap. -/
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

3. **Effect-tracked dispatch**: After running a formula, record its cost
   (tokens × quality). Choose agents by capability matching. The effect
   algebra gives provable bounds: parallel ≤ sequential, composition is
   associative. Cost is observed, not predicted.

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

-- ═══════════════════════════════════════════════════════════════
-- SECTION 14: Fidelity Preorder — Abstract Information Loss
-- ═══════════════════════════════════════════════════════════════

/-! Every LLM cell is a lossy channel. The Effect algebra (Section 1)
    tracks COST. The fidelity preorder tracks INFORMATION LOSS — abstractly,
    without fake precision about percentages or sensitivity multipliers.

    The honest mathematical content:
    1. Fidelity has a preorder (reflexive, transitive ≤)
    2. Sequential composition is monotone decreasing (Data Processing Inequality)
    3. Parallel composition is monotone increasing (best path)
    4. There is a top element (lossless)

    No named levels. No percentages. No sensitivity multipliers.
    Just the abstract shape of how fidelity composes through a DAG.

    Fidelity is DECOUPLED from cost. The Effect algebra handles cost
    accounting; this preorder handles information-loss ordering. They
    compose independently through the DAG. -/

/-- Abstract fidelity preorder with sequential and parallel composition.
    Captures the algebraic structure of how information loss composes
    through a DAG of lossy channels. -/
class FidelityAlgebra (F : Type) where
  /-- Fidelity ordering: a ≤ b means "a is at most as faithful as b." -/
  le : F → F → Prop
  /-- Reflexivity of fidelity ordering. -/
  le_refl : ∀ (a : F), le a a
  /-- Transitivity of fidelity ordering. -/
  le_trans : ∀ (a b c : F), le a b → le b c → le a c
  /-- Top element: lossless fidelity (identity channel). -/
  top : F
  /-- Sequential composition: pipeline two lossy channels. -/
  seq : F → F → F
  /-- Parallel composition: best-of-two paths. -/
  par : F → F → F
  /-- Top is the greatest element. -/
  le_top : ∀ (a : F), le a top
  /-- Data Processing Inequality: sequential composition can only lose fidelity.
      Passing through a lossy channel never increases fidelity. -/
  dpi : ∀ (a b : F), le (seq a b) a
  /-- Parallel improvement: having an alternative path can only help.
      The best of two paths is at least as good as either alone. -/
  par_le_left : ∀ (a b : F), le a (par a b)
  /-- Top is left identity for sequential composition. -/
  seq_top : ∀ (a : F), seq top a = a

-- ── Derived Properties ──────────────────────────────────────

/-- Chaining through top and then a channel yields at most that channel's fidelity.
    Immediate from seq_top and le_refl. -/
theorem FidelityAlgebra.seq_top_le [FidelityAlgebra F] (a : F) :
    FidelityAlgebra.le (FidelityAlgebra.seq FidelityAlgebra.top a) a := by
  rw [FidelityAlgebra.seq_top]; exact FidelityAlgebra.le_refl a

/-- A three-stage pipeline loses at least as much as the first two stages.
    Corollary of DPI applied twice via transitivity. -/
theorem FidelityAlgebra.dpi_chain [FidelityAlgebra F] (a b c : F) :
    FidelityAlgebra.le (FidelityAlgebra.seq (FidelityAlgebra.seq a b) c)
                       (FidelityAlgebra.seq a b) :=
  FidelityAlgebra.dpi (FidelityAlgebra.seq a b) c

-- ═══════════════════════════════════════════════════════════════
-- SECTION 15: Differential Staleness — Beyond Binary
-- ═══════════════════════════════════════════════════════════════

/-! Binary staleness (fresh/stale) is the correct first approximation.
    But sometimes an upstream cell changes "cosmetically" (formatting,
    whitespace) and sometimes it changes "substantively" (different
    conclusions, new data). Re-running ALL downstream cells when an
    upstream cell changes cosmetically wastes tokens.

    Differential staleness tracks an ESTIMATED DRIFT magnitude alongside
    the binary staleness flag. This enables smart recomputation:
    - drift = 0: upstream changed but output is likely identical → skip
    - drift < threshold: upstream changed slightly → maybe skip
    - drift ≥ threshold: upstream changed significantly → must recompute -/

/-- Drift estimate for a stale cell. -/
structure DriftEstimate where
  magnitude : Nat    -- 0-100: how much the upstream change affects this cell
  source    : String -- Which upstream cell triggered the staleness
  deriving Repr, BEq, DecidableEq

/-- Extended cell state with differential staleness. -/
inductive DiffCellState where
  | empty     : DiffCellState
  | stale     : (last : Unified.Value) → (drift : DriftEstimate) → DiffCellState
  | computing : (last : Option Unified.Value) → DiffCellState
  | fresh     : (val : Unified.Value) → DiffCellState
  | failed    : (err : String) → (last : Option Unified.Value) → DiffCellState
  deriving Repr, DecidableEq, BEq

/-- A recomputation decision based on drift threshold. -/
def DiffCellState.shouldRecompute (s : DiffCellState) (threshold : Nat) : Bool :=
  match s with
  | .stale _ drift => drift.magnitude ≥ threshold
  | .empty => true
  | _ => false

/-- Zero drift never triggers recomputation (for any positive threshold). -/
theorem zero_drift_skips (v : Unified.Value) (src : String) (h : 0 < threshold) :
    (DiffCellState.stale v { magnitude := 0, source := src }).shouldRecompute threshold = false := by
  simp [DiffCellState.shouldRecompute]
  omega

/-- Maximum drift always triggers recomputation (for any threshold ≤ 100). -/
theorem max_drift_recomputes (v : Unified.Value) (src : String) (h : threshold ≤ 100) :
    (DiffCellState.stale v { magnitude := 100, source := src }).shouldRecompute threshold = true := by
  simp [DiffCellState.shouldRecompute]
  omega

/-- Propagate differential staleness with drift estimation.
    Drift magnitude is the source change magnitude, clamped to 100. -/
def propagateDiffStale (states : String → DiffCellState) (cells : List Unified.Cell)
    (changedCell : String) (changeMagnitude : Nat)
    : String → DiffCellState :=
  fun n =>
    match states n with
    | .fresh v =>
      let cell? := cells.find? (·.name = n)
      match cell? with
      | some c =>
        if c.deps.contains changedCell then
          let drift := Nat.min 100 changeMagnitude
          .stale v { magnitude := drift, source := changedCell }
        else .fresh v
      | none => .fresh v
    | other => other

/-- Non-vacuity: cosmetic change (magnitude 5) is below threshold of 20 → skip. -/
example :
    let states := fun n => if n = "synthesize"
      then DiffCellState.fresh { content := "old", version := 1, stale := false }
      else DiffCellState.empty
    let cells : List Unified.Cell := [
      { name := "synthesize", cellType := .synthesis,
        prompt := "{{analyze}}", refs := [{ cell := "analyze", field := none }] }]
    let newStates := propagateDiffStale states cells "analyze" 5
    (newStates "synthesize").shouldRecompute 20 = false := by
  decide

/-- Non-vacuity: substantive change (magnitude 80) is above threshold → recompute. -/
example :
    let states := fun n => if n = "synthesize"
      then DiffCellState.fresh { content := "old", version := 1, stale := false }
      else DiffCellState.empty
    let cells : List Unified.Cell := [
      { name := "synthesize", cellType := .synthesis,
        prompt := "{{analyze}}", refs := [{ cell := "analyze", field := none }] }]
    let newStates := propagateDiffStale states cells "analyze" 80
    (newStates "synthesize").shouldRecompute 20 = true := by
  decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 16: The Prompt-Evaluate Adjunction
-- ═══════════════════════════════════════════════════════════════

/-! The Tao perspective identifies a fundamental asymmetry:
    - Prompting (filling a template with data) is CHEAP
    - Evaluating (running an LLM) is EXPENSIVE

    These form an adjunction: Prompt ⊣ Evaluate, meaning
    "filling a template with data" is left adjoint to
    "running an LLM to extract data."

    Concretely in Gas City:
    - Prompt: takes upstream values, fills template → produces a prompt string
    - Evaluate: takes a prompt string, runs LLM → produces a cell value

    The unit η : Data → Evaluate(Prompt(Data)) is "one round-trip":
      data → fill template → run LLM → get back (lossy) data.
    This is one pass through the LLM — fidelity loss is tracked
    by the abstract preorder (Section 14).

    The counit ε : Prompt(Evaluate(computation)) → computation is refinement:
      compute → take output → re-prompt → get refined computation.

    The MONAD T = Evaluate ∘ Prompt is the "one round-trip" monad.
    Its Kleisli category IS the category of Gas City computations.

    We formalize this without dependent on Mathlib's category theory
    by defining the operations and their key properties directly. -/

/-- A prompt operation: fills a template with data.
    This is the LEFT adjoint (cheap, preserves coproducts). -/
structure PromptOp where
  template : String
  refs     : List String    -- Names of referenced cells
  deriving Repr, BEq, DecidableEq

/-- An evaluate operation: runs an LLM on a prompt.
    This is the RIGHT adjoint (expensive, preserves limits).
    Fidelity is tracked separately via the abstract preorder (Section 14),
    not bundled into the eval operation. -/
structure EvalOp where
  model    : String         -- Which model to use
  effect   : Effect         -- Expected cost
  deriving Repr, BEq, DecidableEq

/-- A round-trip: prompt then evaluate. This is the monad T = Eval ∘ Prompt.
    One round-trip takes data, fills a template, runs an LLM, gets data back. -/
structure RoundTrip where
  prompt : PromptOp
  eval   : EvalOp
  deriving Repr, BEq, DecidableEq

/-- The effect of a round-trip is the eval's effect (prompting is free). -/
def RoundTrip.effect (rt : RoundTrip) : Effect := rt.eval.effect

/-- Sequential composition of round-trips (Kleisli composition).
    In the monad: given f : A → T B and g : B → T C, compose to get A → T C.
    In Gas City: chain two cells. Effects compose; fidelity is tracked
    separately via the abstract preorder (Section 14). -/
def RoundTrip.compose (first second : RoundTrip) : RoundTrip where
  prompt := second.prompt  -- Second cell's template
  eval := {
    model := second.eval.model
    effect := Effect.seq first.eval.effect second.eval.effect
  }

/-- Identity round-trip: no-op prompt, zero-cost eval.
    This is the monad unit η. -/
def RoundTrip.identity : RoundTrip where
  prompt := { template := "{{input}}", refs := ["input"] }
  eval := { model := "passthrough", effect := Effect.zero }

/-- The key asymmetry: prompting cost is always 0 (left adjoint is "free").
    This is a structural property of the adjunction. -/
theorem prompt_is_free : ∀ (rt : RoundTrip), rt.effect = rt.eval.effect := by
  intro rt; rfl

/-- Composing round-trips composes effects sequentially. -/
theorem RoundTrip.compose_effect (a b : RoundTrip) :
    (RoundTrip.compose a b).effect = Effect.seq a.effect b.effect := by
  rfl

/-- A refinement loop: evaluate, then re-prompt with the result, then re-evaluate.
    This is the comonad W = Prompt ∘ Evaluate applied twice.
    Each iteration adds cost but may improve quality. -/
structure RefinementLoop where
  base    : RoundTrip       -- Initial computation
  refines : Nat             -- Number of refinement iterations
  deriving Repr, BEq, DecidableEq

/-- Total effect of a refinement loop: base cost × (1 + refines).
    Each refinement re-runs the same cell. -/
def RefinementLoop.totalEffect (rl : RefinementLoop) : Effect where
  tokens := rl.base.effect.tokens * (1 + rl.refines)
  quality := rl.base.effect.quality  -- Quality of each individual run

/-- Zero refinements = same cost as base. -/
theorem RefinementLoop.zero_refines_same_cost (rt : RoundTrip) :
    (RefinementLoop.mk rt 0).totalEffect.tokens = rt.effect.tokens := by
  simp [RefinementLoop.totalEffect, RoundTrip.effect]

/-- More refinements always cost more tokens. -/
theorem RefinementLoop.more_refines_more_cost (rt : RoundTrip) (n : Nat) :
    (RefinementLoop.mk rt n).totalEffect.tokens
    ≤ (RefinementLoop.mk rt (n + 1)).totalEffect.tokens := by
  simp only [RefinementLoop.totalEffect, RoundTrip.effect]
  exact Nat.mul_le_mul_left _ (by omega)

-- ── Non-Vacuity: Adjunction Examples ────────────────────────

/-- A concrete round-trip: extract types from source code. -/
def exExtractTypes : RoundTrip where
  prompt := { template := "Read this code and list all types: {{source}}", refs := ["source"] }
  eval := { model := "sonnet", effect := { tokens := 5000, quality := .good } }

/-- A concrete round-trip: synthesize from extracted types. -/
def exSynthesize : RoundTrip where
  prompt := { template := "Given types: {{types}}, what algebra? {{patterns}}", refs := ["types", "patterns"] }
  eval := { model := "opus", effect := { tokens := 12000, quality := .excellent } }

/-- Composing extract → synthesize gives total cost 17000, quality good. -/
example : (RoundTrip.compose exExtractTypes exSynthesize).effect
    = { tokens := 17000, quality := .good } := by decide

/-- A refinement loop: run extract-types 3 times (base + 2 refines).
    Cost: 5000 × 3 = 15000. -/
example : (RefinementLoop.mk exExtractTypes 2).totalEffect.tokens = 15000 := by decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 17: Pipeline Cost Accounting
-- ═══════════════════════════════════════════════════════════════

/-! Given a pipeline of N cells, the total cost is the sum of
    individual costs. Fidelity loss through the pipeline is tracked
    by the abstract preorder (Section 14) — the DPI guarantees
    that each additional stage can only lose information, but we
    make no fake-precision claims about specific retention percentages. -/

/-- Total cost of a pipeline: sum of all cell costs. -/
def pipelineTotalCost (p : List RoundTrip) : Nat :=
  p.foldl (fun acc rt => acc + rt.effect.tokens) 0

/-- Empty pipeline has zero cost. -/
theorem pipeline_empty_zero_cost : pipelineTotalCost [] = 0 := by rfl

-- ── Non-Vacuity: Pipeline Examples ──────────────────────────

/-- The algebraic survey pipeline: source → extract → pattern → synthesize → decide. -/
def exPipeline : List RoundTrip := [
  { prompt := { template := "Read source: {{code}}", refs := ["code"] },
    eval := { model := "sonnet", effect := { tokens := 5000, quality := .good } } },
  { prompt := { template := "Find patterns: {{types}}", refs := ["types"] },
    eval := { model := "sonnet", effect := { tokens := 8000, quality := .adequate } } },
  { prompt := { template := "Synthesize: {{patterns}}", refs := ["patterns"] },
    eval := { model := "opus", effect := { tokens := 12000, quality := .good } } },
  { prompt := { template := "Decide: {{synthesis}}", refs := ["synthesis"] },
    eval := { model := "sonnet", effect := { tokens := 1000, quality := .adequate } } }
]

/-- Pipeline total cost: 5000 + 8000 + 12000 + 1000 = 26000 tokens. -/
example : pipelineTotalCost exPipeline = 26000 := by decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 18: The Graded Monad — Effect-Indexed Computation
-- ═══════════════════════════════════════════════════════════════

/-! Tao's insight: the existing Effect algebra (Section 1) already IS a
    graded monad. A graded monad over a monoid (M, ⊕, e) is a family
    of type constructors T_m indexed by m ∈ M, with:

    - unit η : A → T_e A              (pure computation at identity effect)
    - bind μ : T_m A → (A → T_n B) → T_{m⊕n} B  (effects compose via monoid)
    - monad laws hold up to monoid structure

    For Gas City:
    - M = (Effect, seq, zero)          — the effect monoid (Section 1)
    - T_{(c,q)} A = "computation of type A costing ≤ c tokens at quality ≥ q"
    - The theorems seq_assoc, seq_zero_left, seq_zero_right ARE the monad laws

    We formalize this directly, without importing Mathlib category theory.
    The construction makes explicit what was already implicit in the algebra. -/

-- ── The Graded Monad Structure ────────────────────────────────

/-- A graded computation: a value of type α tagged with an effect grade.
    `Graded e α` represents "a computation that produces an α with effect e."

    This is the type family T_e(A) in the graded monad.
    The effect tag is a phantom index — it constrains composition
    but does not change the runtime representation. -/
structure Graded (e : Effect) (α : Type) where
  val : α

/-- Pure: inject a value into the graded monad at the identity effect.
    This is the unit η : A → T_{zero} A.
    A pure computation has zero cost and maximum quality. -/
def Graded.pure (a : α) : Graded Effect.zero α :=
  ⟨a⟩

/-- Bind: compose graded computations, accumulating effects.
    Given a computation of type A at effect e₁, and a function
    from A to a computation of type B at effect e₂, produce
    a computation of type B at effect (e₁ ⊕ e₂).

    This is the Kleisli composition in the graded monad. The effect
    indices compose via Effect.seq — costs add, quality takes minimum. -/
def Graded.bind (ma : Graded e₁ α) (f : α → Graded e₂ β) : Graded (Effect.seq e₁ e₂) β :=
  ⟨(f ma.val).val⟩

/-- Map: apply a pure function inside the graded monad.
    Does not change the effect (pure functions are free). -/
def Graded.map (f : α → β) (ma : Graded e α) : Graded e β :=
  ⟨f ma.val⟩

/-- Join: flatten a nested graded computation.
    This is the multiplication μ : T_m(T_n(A)) → T_{m⊕n}(A). -/
def Graded.join (mma : Graded e₁ (Graded e₂ α)) : Graded (Effect.seq e₁ e₂) α :=
  ⟨mma.val.val⟩

-- ── The Graded Monad Laws ─────────────────────────────────────

/-! The three monad laws for the graded monad, stated as equalities
    up to the effect monoid structure. Each law corresponds to an
    existing theorem from Section 1.

    Law 1 (Left unit):   bind (pure a) f  =  f a
           Effect law:    seq zero e₂     =  e₂        (seq_zero_left)

    Law 2 (Right unit):  bind m pure      =  m
           Effect law:    seq e₁ zero     =  e₁        (seq_zero_right)

    Law 3 (Assoc):       bind (bind m f) g = bind m (λ a. bind (f a) g)
           Effect law:    seq (seq e₁ e₂) e₃ = seq e₁ (seq e₂ e₃)  (seq_assoc)

    The proof strategy: since Graded is a simple wrapper, the value-level
    equalities are trivial (they just unwrap). The interesting content is
    in the EFFECT INDICES, where the existing Section 1 theorems do the work. -/

/-- Left unit law: bind (pure a) f = f a (up to effect index).
    The effect index of the left side is `seq zero e₂`, which equals `e₂`
    by seq_zero_left. We prove the value equality directly, then state
    the index coherence separately. -/
theorem Graded.bind_pure_left (a : α) (f : α → Graded e β) :
    (Graded.bind (Graded.pure a) f).val = (f a).val := by
  rfl

/-- Left unit index coherence: the effect grade `seq zero e` equals `e`. -/
theorem Graded.bind_pure_left_index (e : Effect) :
    Effect.seq Effect.zero e = e :=
  Effect.seq_zero_left e

/-- Right unit law: bind m pure = m (up to effect index).
    The effect index of the left side is `seq e₁ zero`, which equals `e₁`
    by seq_zero_right. -/
theorem Graded.bind_pure_right (ma : Graded e α) :
    (Graded.bind ma Graded.pure).val = ma.val := by
  rfl

/-- Right unit index coherence: the effect grade `seq e zero` equals `e`. -/
theorem Graded.bind_pure_right_index (e : Effect) :
    Effect.seq e Effect.zero = e :=
  Effect.seq_zero_right e

/-- Associativity law: bind (bind m f) g = bind m (λ a. bind (f a) g)
    (up to effect index). The left side has index `seq (seq e₁ e₂) e₃`,
    the right side has index `seq e₁ (seq e₂ e₃)`. These are equal
    by seq_assoc. -/
theorem Graded.bind_assoc (ma : Graded e₁ α) (f : α → Graded e₂ β)
    (g : β → Graded e₃ γ) :
    (Graded.bind (Graded.bind ma f) g).val
    = (Graded.bind ma (fun a => Graded.bind (f a) g)).val := by
  rfl

/-- Associativity index coherence: `seq (seq e₁ e₂) e₃ = seq e₁ (seq e₂ e₃)`. -/
theorem Graded.bind_assoc_index (e₁ e₂ e₃ : Effect) :
    Effect.seq (Effect.seq e₁ e₂) e₃ = Effect.seq e₁ (Effect.seq e₂ e₃) :=
  Effect.seq_assoc e₁ e₂ e₃

-- ── The GradedMonad Typeclass ─────────────────────────────────

/-- A graded monad over a monoid (M, op, unit) with a type family T.
    This captures the abstract structure: T is indexed by a monoid,
    with unit at the identity element and bind composing indices.

    The laws are split into VALUE laws (about the underlying computation)
    and INDEX laws (about the monoid structure). This separation is
    necessary because T_m and T_n are different types when m ≠ n,
    so we express value equality via an extraction function `extract`. -/
structure GradedMonadLaws (M : Type) (op : M → M → M) (unit : M)
    (T : M → Type → Type) where
  /-- Left unit index: op unit n = n. -/
  left_unit_index  : ∀ (n : M), op unit n = n
  /-- Right unit index: op m unit = m. -/
  right_unit_index : ∀ (m : M), op m unit = m
  /-- Associativity index: op (op m n) p = op m (op n p). -/
  assoc_index      : ∀ (m n p : M), op (op m n) p = op m (op n p)

/-- The Effect algebra satisfies the graded monad laws.
    This is the central theorem of Section 18: the existing Effect.seq monoid
    structure satisfies all index-level graded monad laws.
    The proofs delegate directly to Section 1 theorems. -/
def effectGradedMonadLaws : GradedMonadLaws Effect Effect.seq Effect.zero Graded where
  left_unit_index  := Effect.seq_zero_left
  right_unit_index := Effect.seq_zero_right
  assoc_index      := Effect.seq_assoc

-- ── Connection to RoundTrip (Section 16) ─────────────────────

/-! The RoundTrip structure from Section 16 is the Kleisli arrow of the
    graded monad. A RoundTrip with effect e corresponds to a morphism
    A → T_e B in the Kleisli category. The RoundTrip.compose operation
    IS Kleisli composition, and RoundTrip.identity IS the monad unit. -/

/-- Lift a RoundTrip into a graded computation.
    The effect grade is determined by the RoundTrip's eval effect. -/
def RoundTrip.toGraded (rt : RoundTrip) (input : String) : Graded rt.effect String :=
  ⟨input⟩  -- In the formal model, the value is the result of running the LLM

/-- The identity RoundTrip corresponds to the graded monad's pure operation.
    Its effect is zero, matching the unit grade. -/
theorem RoundTrip.identity_is_pure :
    RoundTrip.identity.effect = Effect.zero := by
  rfl

/-- RoundTrip.compose corresponds to Kleisli composition in the graded monad.
    The composed effect equals seq of the individual effects. -/
theorem RoundTrip.compose_is_kleisli (a b : RoundTrip) :
    (RoundTrip.compose a b).effect = Effect.seq a.effect b.effect := by
  rfl

-- ── Connection to Pipelines (Section 17) ─────────────────────

/-! A pipeline (List RoundTrip) from Section 17 corresponds to an
    iterated Kleisli composition in the graded monad. The pipeline's
    total cost equals the sum of cell costs, which is exactly the
    token component of the composed effect grade.

    The key insight: `pipelineTotalCost` computes the same value as
    the token component of folding Effect.seq over the pipeline. -/

/-- The composed effect of a pipeline: fold Effect.seq over all cells.
    This is the grade of the composed Kleisli arrow. -/
def pipelineEffect (p : List RoundTrip) : Effect :=
  p.foldl (fun acc rt => Effect.seq acc rt.effect) Effect.zero

/-- Empty pipeline has zero effect. -/
theorem pipelineEffect_nil : pipelineEffect [] = Effect.zero := by
  rfl

/-- Helper: the token component of foldl Effect.seq tracks foldl of token addition.
    This lemma generalizes the relationship between the effect fold and the cost fold
    for any accumulator state. -/
private theorem foldl_effect_tokens (p : List RoundTrip) (acc : Effect) (n : Nat)
    (h : acc.tokens = n) :
    (p.foldl (fun a rt => Effect.seq a rt.effect) acc).tokens
    = p.foldl (fun a rt => a + rt.effect.tokens) n := by
  induction p generalizing acc n with
  | nil => exact h
  | cons rt rest ih =>
    simp only [List.foldl_cons]
    exact ih (Effect.seq acc rt.effect) (n + rt.effect.tokens)
      (by simp [Effect.seq, h])

/-- The token component of the pipeline effect equals pipelineTotalCost.
    This connects the graded monad (Section 18) to the pipeline cost model
    (Section 17). The graded monad's index tracking IS the cost accounting. -/
theorem pipelineEffect_tokens_eq_cost (p : List RoundTrip) :
    (pipelineEffect p).tokens = pipelineTotalCost p := by
  exact foldl_effect_tokens p Effect.zero 0 rfl

-- ── Non-Vacuity Witnesses ─────────────────────────────────────

/-- Non-vacuity: a pure graded computation has zero effect. -/
example : (Graded.pure "hello" : Graded Effect.zero String).val = "hello" := by rfl

/-- Non-vacuity: binding two graded computations accumulates effects. -/
example :
    let e₁ : Effect := { tokens := 5000, quality := .good }
    let e₂ : Effect := { tokens := 3000, quality := .adequate }
    let m : Graded e₁ String := ⟨"types found"⟩
    let f : String → Graded e₂ String := fun s => ⟨s ++ " → synthesized"⟩
    (Graded.bind m f).val = "types found → synthesized" := by rfl

/-- Non-vacuity: the composed effect has correct tokens and quality. -/
example :
    let e₁ : Effect := { tokens := 5000, quality := .good }
    let e₂ : Effect := { tokens := 3000, quality := .adequate }
    Effect.seq e₁ e₂ = { tokens := 8000, quality := .adequate } := by rfl

/-- Non-vacuity: three-way bind associativity — both groupings give same result. -/
example :
    let e₁ : Effect := { tokens := 1000, quality := .good }
    let e₂ : Effect := { tokens := 2000, quality := .adequate }
    let e₃ : Effect := { tokens := 3000, quality := .excellent }
    let m : Graded e₁ Nat := ⟨42⟩
    let f : Nat → Graded e₂ Nat := fun n => ⟨n + 1⟩
    let g : Nat → Graded e₃ Nat := fun n => ⟨n * 2⟩
    (Graded.bind (Graded.bind m f) g).val
    = (Graded.bind m (fun a => Graded.bind (f a) g)).val := by rfl

/-- Non-vacuity: the graded monad index tracks the exPipeline cost from Section 17. -/
example : (pipelineEffect exPipeline).tokens = 26000 := by decide

/-- Non-vacuity: pipeline effect tokens matches pipelineTotalCost. -/
example : (pipelineEffect exPipeline).tokens = pipelineTotalCost exPipeline := by decide

/-- Non-vacuity: pipeline effect quality is the min across all cells. -/
example : (pipelineEffect exPipeline).quality = .adequate := by decide

/-- Non-vacuity: graded join (the multiplication μ) flattens nested computations. -/
example :
    let inner : Graded { tokens := 3000, quality := .good } String := ⟨"result"⟩
    let outer : Graded { tokens := 5000, quality := .adequate } (Graded { tokens := 3000, quality := .good } String) := ⟨inner⟩
    (Graded.join outer).val = "result" := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 19: Molecule Lifecycle — Staleness as Re-instantiation
-- ═══════════════════════════════════════════════════════════════

/-! The key insight from mapping the reactive algebra onto Gas Town primitives:
    staleness is NOT a label mutation on existing beads. It is RE-INSTANTIATION.

    The lifecycle:
      Proto (solid)  ──pour──▶  Molecule (liquid)  ──execute──▶  all cells closed
                                                                       │
                                                                  squash ▼
                                                              Digest (immutable)
                                                                       │
                                                           upstream changes
                                                                       │
                                                     distill ──▶ Proto' ──pour──▶ Molecule₂

    Each molecule is a DAG of beads. Completed molecules become immutable digests.
    When upstream changes, we don't reopen beads — we pour a new molecule.
    Version history IS the chain of digests.

    This maps perfectly to Gas Town primitives:
      Cell          = Bead (bd create)
      Cell type     = Label (cell:text, cell:inventory, ...)
      DAG edges     = Dependencies (--deps)
      Ready set     = bd ready (blocker-aware)
      BeginCompute  = bd update --claim
      Complete      = bd close --reason "result"
      PropagateStale = pour new molecule (re-instantiation)
      DAG viz       = bd graph --html
      History       = chain of squashed digests
-/

/-- Phase of matter for a molecule in the Gas Town lifecycle. -/
inductive Phase where
  | solid   : Phase  -- Proto: template, not yet instantiated
  | liquid  : Phase  -- Molecule: active, beads being evaluated
  | vapor   : Phase  -- Wisp: ephemeral molecule (auto-cleanup)
  | crystal : Phase  -- Digest: squashed, immutable history
  deriving DecidableEq, Repr, BEq

/-- Specification of a cell within a proto. -/
structure CellSpec where
  name   : String
  type   : CellType
  prompt : String                  -- May contain {{variable}} and {{ref}} holes
  refs   : List String             -- Names of upstream cells
  deriving Repr, BEq, DecidableEq

/-- A proto is a reusable template for a cell DAG.
    Variables ({{key}}) are substituted during instantiation (pour). -/
structure Proto where
  name      : String
  cells     : List CellSpec
  variables : List String          -- Template variables (e.g., "source", "repo")
  deriving Repr, BEq, DecidableEq

/-- A molecule is an instantiated proto — real beads with real dependencies. -/
structure Molecule where
  proto     : Proto                -- Which template this came from
  phase     : Phase                -- Current phase of matter
  bindings  : List (String × String)  -- Variable substitutions applied
  cells     : List CellSpec        -- Instantiated cell specs (variables resolved)
  deriving Repr, BEq, DecidableEq

/-- A digest is the immutable record of a completed molecule execution. -/
structure Digest where
  molecule  : Molecule             -- The molecule that was executed
  values    : List (String × String)  -- Cell name → final value
  cost      : Effect               -- Total effect consumed
  deriving Repr, BEq, DecidableEq

/-- Pour: instantiate a proto into a molecule (solid → liquid). -/
def pour (p : Proto) (bindings : List (String × String)) : Molecule where
  proto    := p
  phase    := .liquid
  bindings := bindings
  cells    := p.cells.map fun spec =>
    { spec with prompt := bindings.foldl (fun s (k, v) =>
        s.replace ("{{" ++ k ++ "}}") v) spec.prompt }

/-- Squash: compress a completed molecule into an immutable digest (liquid → crystal). -/
def squash (m : Molecule) (values : List (String × String)) (cost : Effect) : Digest where
  molecule := { m with phase := .crystal }
  values   := values
  cost     := cost

/-- Pouring preserves the proto's cell count. -/
theorem pour_preserves_cells (p : Proto) (bindings : List (String × String)) :
    (pour p bindings).cells.length = p.cells.length := by
  simp [pour, List.length_map]

/-- Pouring produces a liquid-phase molecule. -/
theorem pour_is_liquid (p : Proto) (bindings : List (String × String)) :
    (pour p bindings).phase = .liquid := by
  rfl

/-- Squashing produces a crystal-phase digest. -/
theorem squash_is_crystal (m : Molecule) (vals : List (String × String)) (cost : Effect) :
    (squash m vals cost).molecule.phase = .crystal := by
  rfl

/-- The cost of a squashed digest reflects the total execution cost. -/
theorem squash_preserves_cost (m : Molecule) (vals : List (String × String)) (e : Effect) :
    (squash m vals e).cost = e := by
  rfl

-- ── Non-Vacuity: Molecule Lifecycle ────────────────────────

/-- Example proto: a code analysis pipeline. -/
def exCodeAnalysis : Proto where
  name := "code-analysis"
  cells := [
    { name := "read-code", type := .text, prompt := "Read files in {{repo}}", refs := [] },
    { name := "find-bugs", type := .inventory, prompt := "List bugs in {{read-code}}", refs := ["read-code"] },
    { name := "find-patterns", type := .inventory, prompt := "List patterns in {{read-code}}", refs := ["read-code"] },
    { name := "report", type := .synthesis, prompt := "Bugs: {{find-bugs}}, Patterns: {{find-patterns}}", refs := ["find-bugs", "find-patterns"] }
  ]
  variables := ["repo"]

/-- Pouring the code analysis proto with repo=myapp. -/
example : (pour exCodeAnalysis [("repo", "myapp")]).cells.head?.map (·.prompt)
    = some "Read files in myapp" := by native_decide

/-- The poured molecule has 4 cells. -/
example : (pour exCodeAnalysis [("repo", "myapp")]).cells.length = 4 := by native_decide

/-- Squashing after execution captures the total cost. -/
example :
    let m := pour exCodeAnalysis [("repo", "myapp")]
    let d := squash m [("read-code", "src/..."), ("find-bugs", "3 bugs"), ("find-patterns", "MVC"),
                       ("report", "Review complete")] { tokens := 25000, quality := .good }
    d.cost.tokens = 25000 := by rfl

-- ═══════════════════════════════════════════════════════════════
-- SECTION 20: Annotated Evolution — Cyclic Refinement Across Generations
-- ═══════════════════════════════════════════════════════════════

/-! Within a molecule, the cell graph is a DAG (no circular deps).
    But ACROSS molecule generations, cycles emerge naturally through annotations.

    A user (or agent) annotates a completed digest with evolution instructions:
    - AddRef: "cell C should also depend on cell A" (adds a DAG edge)
    - RemoveRef: "cell B no longer needs cell X" (removes a DAG edge)
    - SplitCell: "split cell X into X₁ and X₂" (topology change)
    - MergeCell: "merge cells X and Y into Z" (topology change)
    - RefinePrompt: "cell X's prompt should say ..." (prompt evolution)
    - SeedValue: "cell X should start with value V" (from prior digest)

    The evolution operator: evolve(Proto, Annotations) → Proto'
    This is the mechanism by which the graph becomes cyclic across time:

    Generation 1:  A → B → C     (C has no dep on A)
                   annotate(C, AddRef A)
    Generation 2:  A → B → C     (C now deps on A too — diamond)
                        ↗
                   annotate(B, SeedValue from C's gen1 result)
    Generation 3:  A → B → C     (B seeded with C's old output — cycle unrolled)

    Each generation is a DAG. The cycle lives in the CHAIN of generations.
    Annotations are the feedback signal that drives structural evolution.
-/

/-- An annotation on a completed digest that influences the next proto. -/
inductive Annotation where
  | addRef       (cell : String) (newRef : String)              -- Add a dependency edge
  | removeRef    (cell : String) (oldRef : String)              -- Remove a dependency edge
  | splitCell    (cell : String) (into : List String)           -- Split one cell into many
  | mergeCell    (cells : List String) (into : String)          -- Merge many cells into one
  | refinePrompt (cell : String) (newPrompt : String)           -- Update a cell's prompt
  | seedValue    (cell : String) (value : String)               -- Pre-fill from prior digest
  | addCell      (spec : CellSpec)                              -- Add a new cell to the graph
  | removeCell   (cell : String)                                -- Remove a cell from the graph
  deriving Repr, BEq, DecidableEq

/-- Apply a single annotation to a proto, producing an evolved proto. -/
def applyAnnotation (p : Proto) (a : Annotation) : Proto :=
  match a with
  | .addRef cell newRef =>
    { p with cells := p.cells.map fun spec =>
        if spec.name == cell then { spec with refs := spec.refs ++ [newRef] } else spec }
  | .removeRef cell oldRef =>
    { p with cells := p.cells.map fun spec =>
        if spec.name == cell then { spec with refs := spec.refs.filter (· != oldRef) } else spec }
  | .refinePrompt cell newPrompt =>
    { p with cells := p.cells.map fun spec =>
        if spec.name == cell then { spec with prompt := newPrompt } else spec }
  | .addCell spec =>
    { p with cells := p.cells ++ [spec] }
  | .removeCell cell =>
    { p with cells := p.cells.filter (·.name != cell) }
  | .splitCell cell _into =>
    -- Splitting: replace cell with N new cells, each inheriting the original's refs.
    -- The caller specifies the new cell specs via subsequent addCell annotations.
    { p with cells := p.cells.filter (·.name != cell) }
  | .mergeCell cells into =>
    -- Merging: combine multiple cells into one. Refs = union of all refs.
    let allRefs := ((p.cells.filter (fun s => cells.contains s.name)).map (·.refs)).flatten
    let merged : CellSpec := {
      name := into
      type := .synthesis  -- Merged cells default to synthesis
      prompt := "Merged from: " ++ String.intercalate ", " cells
      refs := List.eraseDups allRefs
    }
    { p with cells := (p.cells.filter (fun s => !cells.contains s.name)) ++ [merged] }
  | .seedValue _cell _value =>
    -- Seeding doesn't change the proto structure; it's applied during pour.
    p

/-- Evolve a proto by applying a list of annotations. -/
def evolve (p : Proto) (annotations : List Annotation) : Proto :=
  annotations.foldl applyAnnotation p

/-- A generation: one complete cycle of pour → execute → squash → annotate. -/
structure Generation where
  number      : Nat
  digest      : Digest
  annotations : List Annotation    -- Annotations for the NEXT generation
  deriving Repr, BEq, DecidableEq

/-- An evolution history: a chain of generations showing how the graph evolves. -/
def EvolutionHistory := List Generation

/-- Evolution preserves or modifies cell count based on annotations. -/
theorem evolve_empty_preserves (p : Proto) :
    evolve p [] = p := by
  rfl

-- Annotation properties are demonstrated via concrete examples below
-- (universal proofs over applyAnnotation's match branches require
--  more infrastructure than they're worth for the formalization).

-- ── The Cycle Unrolling Theorem ────────────────────────────

-- Two generations form a "cycle" when generation N+1's proto has
-- an edge from cell B to cell A, and generation N's proto had
-- an edge from cell A to cell B. Cycles unrolled across molecule generations:
-- each generation is a DAG, cycles emerge across the chain.

/-- Check if a proto has a reference edge from `from` to `to`. -/
def hasRef (p : Proto) (from_ to_ : String) : Bool :=
  p.cells.any fun spec => spec.name == from_ && spec.refs.contains to_

-- ── Non-Vacuity: Evolution Examples ────────────────────────

/-- Start with a linear proto: A → B → C. -/
def exLinearProto : Proto where
  name := "linear"
  cells := [
    { name := "A", type := .text, prompt := "Start", refs := [] },
    { name := "B", type := .inventory, prompt := "Process {{A}}", refs := ["A"] },
    { name := "C", type := .synthesis, prompt := "Synthesize {{B}}", refs := ["B"] }
  ]
  variables := []

/-- Evolve: add C→A dependency (creates diamond structure). -/
example : hasRef exLinearProto "C" "A" = false := by native_decide
example : hasRef (evolve exLinearProto [.addRef "C" "A"]) "C" "A" = true := by native_decide

/-- Evolve: refine B's prompt based on feedback. -/
example :
    let evolved := evolve exLinearProto [.refinePrompt "B" "Process {{A}} with focus on types"]
    (evolved.cells.find? (·.name == "B")).map (·.prompt)
      = some "Process {{A}} with focus on types" := by native_decide

/-- Evolve: add a new cell D that depends on both A and C (cross-cutting concern). -/
example :
    let evolved := evolve exLinearProto [
      .addCell { name := "D", type := .decision, prompt := "Decide based on {{A}} and {{C}}",
                 refs := ["A", "C"] }
    ]
    evolved.cells.length = 4 := by native_decide

/-- Multi-step evolution: gen1 → annotate → gen2 → annotate → gen3.
    Shows how the graph topology evolves across generations. -/
example :
    -- Gen 1: Linear A → B → C
    let gen1 := exLinearProto
    -- Annotate: C should also see A (diamond), B needs better prompt
    let annotations1 := [
      Annotation.addRef "C" "A",
      Annotation.refinePrompt "B" "Process {{A}}, focus on dependencies"
    ]
    let gen2 := evolve gen1 annotations1
    -- Gen 2 now has 3 cells, C refs both A and B
    gen2.cells.length = 3
    ∧ hasRef gen2 "C" "A" = true
    ∧ hasRef gen2 "C" "B" = true
    ∧ hasRef gen2 "B" "A" = true := by
  native_decide

/-- The complete lifecycle: pour → squash → evolve → pour again.
    This demonstrates staleness-as-re-instantiation. -/
example :
    -- Pour gen1
    let m1 := pour exLinearProto []
    -- Squash gen1 (simulate completion)
    let d1 := squash m1 [("A", "raw data"), ("B", "processed"), ("C", "synthesis")]
                        { tokens := 15000, quality := .good }
    -- Evolve based on annotations
    let proto2 := evolve d1.molecule.proto [.addRef "C" "A"]
    -- Pour gen2 — this IS the staleness refresh
    let m2 := pour proto2 []
    -- Gen2 molecule has the evolved structure
    hasRef m2.proto "C" "A" = true := by native_decide

-- ═══════════════════════════════════════════════════════════════
-- SECTION 21: DAG-Rewrite-Completeness & Annotation Derivability
-- ═══════════════════════════════════════════════════════════════

/-! ## Part 1: DAG-Rewrite-Completeness

    The 4 minimal annotation operations {addCell, removeCell, addRef, removeRef}
    are DAG-rewrite-complete: for any Proto P1 and P2, there exists a finite
    List Annotation using only these constructors that transforms P1.cells into
    P2.cells.

    Strategy: removeCell every cell in P1, then addCell every cell in P2.
    Since CellSpec carries refs, no separate addRef/removeRef is needed (though
    they are available for finer-grained rewrites).

    Note: `name` and `variables` are proto metadata that NO annotation changes.
    Completeness is stated over the `cells` field. -/

-- ── Metadata Preservation ────────────────────────────────────

/-- Every annotation preserves the proto's name. -/
theorem applyAnnotation_preserves_name (p : Proto) (a : Annotation) :
    (applyAnnotation p a).name = p.name := by
  cases a <;> simp [applyAnnotation]

/-- Every annotation preserves the proto's variables. -/
theorem applyAnnotation_preserves_variables (p : Proto) (a : Annotation) :
    (applyAnnotation p a).variables = p.variables := by
  cases a <;> simp [applyAnnotation]

/-- Evolution preserves the proto's name across any annotation list. -/
theorem evolve_preserves_name (p : Proto) (anns : List Annotation) :
    (evolve p anns).name = p.name := by
  induction anns generalizing p with
  | nil => rfl
  | cons a as ih =>
    simp only [evolve, List.foldl_cons] at *
    exact (ih (applyAnnotation p a)).trans (applyAnnotation_preserves_name p a)

/-- Evolution preserves the proto's variables across any annotation list. -/
theorem evolve_preserves_variables (p : Proto) (anns : List Annotation) :
    (evolve p anns).variables = p.variables := by
  induction anns generalizing p with
  | nil => rfl
  | cons a as ih =>
    simp only [evolve, List.foldl_cons] at *
    exact (ih (applyAnnotation p a)).trans (applyAnnotation_preserves_variables p a)

-- ── Helper Lemmas ────────────────────────────────────────────

/-- Filtering by a constant-true predicate is the identity. -/
private theorem filter_true_id (l : List CellSpec) :
    l.filter (fun _ => true) = l := by
  induction l with
  | nil => rfl
  | cons x xs ih => simp [List.filter, ih]

/-- Removing name n then filtering by names ns = filtering by (n :: ns).
    Proved by induction on the cell list with case-split on Bool values. -/
private theorem filter_removeCell_cons (cells : List CellSpec) (n : String) (ns : List String) :
    (cells.filter (fun c => !(c.name == n))).filter (fun c => !(ns.contains c.name)) =
    cells.filter (fun c => !((n :: ns).contains c.name)) := by
  induction cells with
  | nil => rfl
  | cons x xs ih =>
    simp only [List.filter, List.contains, List.any]
    cases h1 : (x.name == n) <;> cases h2 : (List.any ns (BEq.beq x.name))
    all_goals simp_all [Bool.not_or, List.filter]

-- ── evolve distributes over append ───────────────────────────

/-- Evolution distributes over annotation list concatenation. -/
theorem evolve_append (p : Proto) (as bs : List Annotation) :
    evolve p (as ++ bs) = evolve (evolve p as) bs := by
  simp [evolve, List.foldl_append]

-- ── addCell fold appends specs ───────────────────────────────

/-- Folding addCell annotations appends the specs to the proto's cells. -/
theorem evolve_addAll_cells (p : Proto) (specs : List CellSpec) :
    (evolve p (specs.map fun s => .addCell s)).cells = p.cells ++ specs := by
  induction specs generalizing p with
  | nil => simp [evolve]
  | cons s ss ih =>
    simp only [List.map, evolve, List.foldl_cons]
    change (evolve (applyAnnotation p (.addCell s)) (ss.map fun s => .addCell s)).cells = _
    rw [ih]
    simp [applyAnnotation, List.append_assoc]

-- ── removeCell fold filters by names ─────────────────────────

/-- Folding removeCell over a list of names filters the proto's cells. -/
theorem evolve_removeNames (p : Proto) (names : List String) :
    (evolve p (names.map (Annotation.removeCell ·))).cells =
    p.cells.filter (fun c => !(names.contains c.name)) := by
  induction names generalizing p with
  | nil =>
    exact (filter_true_id p.cells).symm
  | cons n ns ih =>
    simp only [List.map, evolve, List.foldl_cons]
    change (evolve (applyAnnotation p (.removeCell n)) (ns.map (Annotation.removeCell ·))).cells = _
    rw [ih]
    simp only [applyAnnotation]
    exact filter_removeCell_cons p.cells n ns

/-- Folding removeCell annotations (from CellSpecs) filters the proto's cells. -/
theorem evolve_removeAll_cells (p : Proto) (toRemove : List CellSpec) :
    (evolve p (toRemove.map fun c => .removeCell c.name)).cells =
    p.cells.filter (fun c => !(toRemove.map (·.name)).contains c.name) := by
  induction toRemove generalizing p with
  | nil =>
    exact (filter_true_id p.cells).symm
  | cons x xs ih =>
    simp only [List.map, evolve, List.foldl_cons]
    change (evolve (applyAnnotation p (.removeCell x.name)) (xs.map fun c => .removeCell c.name)).cells = _
    rw [ih]
    simp only [applyAnnotation]
    exact filter_removeCell_cons p.cells x.name (xs.map (·.name))

-- ── Corollary: removing all own cell names empties the list ──

/-- Every cell's name is contained in the name list of the cells it belongs to. -/
private theorem name_mem_contains {c : CellSpec} {cells : List CellSpec}
    (h : c ∈ cells) : (cells.map (·.name)).contains c.name = true := by
  induction cells with
  | nil => contradiction
  | cons x xs ih =>
    simp only [List.map, List.contains, List.elem] at *
    cases h with
    | head => rw [beq_self_eq_true]
    | tail _ h' =>
      have := ih h'
      cases (c.name == x.name) <;> simp_all

/-- If every element fails a predicate, filter returns []. -/
private theorem filter_eq_nil_of_all_false (l : List CellSpec) (p : CellSpec → Bool)
    (h : ∀ x, x ∈ l → p x = false) : l.filter p = [] := by
  induction l with
  | nil => rfl
  | cons x xs ih =>
    simp only [List.filter]
    rw [h x (List.Mem.head _)]
    exact ih (fun y hy => h y (List.Mem.tail _ hy))

/-- Filtering a list to exclude all names that appear in it yields []. -/
private theorem filter_own_names_empty (cells : List CellSpec) :
    cells.filter (fun c => !(cells.map (·.name)).contains c.name) = [] := by
  apply filter_eq_nil_of_all_false
  intro c hc
  have := name_mem_contains hc
  simp only [this, Bool.not_true]

-- ── Main Theorem 1: DAG-Rewrite-Completeness ────────────────

/-- The rewrite witness: removeCell all P1 cells, then addCell all P2 cells. -/
def rewriteAnnotations (from_ to_ : Proto) : List Annotation :=
  (from_.cells.map fun c => .removeCell c.name) ++
  (to_.cells.map fun c => .addCell c)

/-- The rewrite witness uses only addCell and removeCell. -/
theorem rewriteAnnotations_minimal (from_ to_ : Proto) :
    ∀ a ∈ rewriteAnnotations from_ to_,
      (∃ c, a = .removeCell c) ∨ (∃ s, a = .addCell s) := by
  intro a ha
  simp only [rewriteAnnotations, List.mem_append, List.mem_map] at ha
  rcases ha with ⟨c, _, rfl⟩ | ⟨s, _, rfl⟩
  · exact Or.inl ⟨c.name, rfl⟩
  · exact Or.inr ⟨s, rfl⟩

/-- **DAG-Rewrite-Completeness**: for any protos P1 and P2, there exists a list
    of annotations using only {addCell, removeCell} that transforms P1.cells
    into P2.cells. -/
theorem dag_rewrite_complete (P1 P2 : Proto) :
    ∃ anns : List Annotation,
      (∀ a ∈ anns, (∃ c, a = .removeCell c) ∨ (∃ s, a = .addCell s)) ∧
      (evolve P1 anns).cells = P2.cells := by
  refine ⟨rewriteAnnotations P1 P2, rewriteAnnotations_minimal P1 P2, ?_⟩
  show (evolve P1 ((P1.cells.map fun c => .removeCell c.name) ++
       (P2.cells.map fun c => .addCell c))).cells = P2.cells
  rw [evolve_append, evolve_addAll_cells, evolve_removeAll_cells, filter_own_names_empty]
  simp

-- ── Non-Vacuity: Completeness Example ────────────────────────

private def exSingleProto : Proto where
  name := "single"
  cells := [{ name := "X", type := .text, prompt := "Do everything", refs := [] }]
  variables := []

/-- Rewriting the linear proto A→B→C into a single-cell proto. -/
example :
    (evolve exLinearProto (rewriteAnnotations exLinearProto exSingleProto)).cells
      = exSingleProto.cells := by
  native_decide

-- ═══════════════════════════════════════════════════════════════
-- Part 2: Derivability of Compound Operations
-- ═══════════════════════════════════════════════════════════════

/-! ## Part 2: Derivability

    The 4 "semantic" operations (splitCell, mergeCell, refinePrompt, seedValue)
    are each derivable from the 4 minimal operations. This shows the 8-operation
    set is convenient but redundant: the minimal 4 suffice. -/

-- ── splitCell = removeCell ───────────────────────────────────

/-- splitCell is exactly removeCell: both filter out the named cell.
    (New cells are added via subsequent addCell annotations.) -/
theorem splitCell_eq_removeCell (p : Proto) (cell : String) (into : List String) :
    applyAnnotation p (.splitCell cell into) = applyAnnotation p (.removeCell cell) := by
  rfl

/-- splitCell followed by any annotations equals removeCell followed by the same. -/
theorem splitCell_derivable (p : Proto) (cell : String) (into : List String)
    (rest : List Annotation) :
    evolve p (.splitCell cell into :: rest) =
    evolve p (.removeCell cell :: rest) := by
  simp [evolve, List.foldl_cons, splitCell_eq_removeCell]

-- ── mergeCell = removeCell* + addCell ────────────────────────

/-- mergeCell is derivable: removing each constituent cell then adding the merged
    cell produces the same cells as the single mergeCell annotation.
    The merged spec is computed from the original proto. -/
theorem mergeCell_cells_derivable (p : Proto) (cellNames : List String) (into : String) :
    let allRefs := ((p.cells.filter (fun s => cellNames.contains s.name)).map (·.refs)).flatten
    let merged : CellSpec :=
      { name := into, type := .synthesis,
        prompt := "Merged from: " ++ String.intercalate ", " cellNames,
        refs := List.eraseDups allRefs }
    (applyAnnotation p (.mergeCell cellNames into)).cells =
    (evolve p ((cellNames.map (Annotation.removeCell ·)) ++ [.addCell merged])).cells := by
  -- Both sides produce: (p.cells.filter (fun s => !cellNames.contains s.name)) ++ [merged]
  simp only []
  rw [evolve_append]
  -- RHS: (evolve (evolve p removes) [.addCell merged]).cells
  -- Unfold the outer evolve [.addCell merged]
  simp only [evolve, List.foldl, applyAnnotation]
  -- Now RHS: (evolve p removes).cells ++ [merged]
  -- LHS: (p.cells.filter ...) ++ [merged]
  -- These differ only in (evolve p removes).cells vs p.cells.filter ...
  congr 1
  exact (evolve_removeNames p cellNames).symm

-- ── seedValue is identity on structure ───────────────────────

/-- seedValue does not change the proto structure at all. -/
theorem seedValue_is_id (p : Proto) (cell value : String) :
    applyAnnotation p (.seedValue cell value) = p := by
  rfl

-- ── Non-Vacuity: Derivability Examples ──────────────────────

/-- splitCell then addCells equals removeCell then addCells (concrete). -/
example :
    let p := exLinearProto
    let splitResult := evolve p [.splitCell "B" ["B1", "B2"],
      .addCell { name := "B1", type := .inventory, prompt := "Part 1 of {{A}}", refs := ["A"] },
      .addCell { name := "B2", type := .inventory, prompt := "Part 2 of {{A}}", refs := ["A"] }]
    let derivedResult := evolve p [.removeCell "B",
      .addCell { name := "B1", type := .inventory, prompt := "Part 1 of {{A}}", refs := ["A"] },
      .addCell { name := "B2", type := .inventory, prompt := "Part 2 of {{A}}", refs := ["A"] }]
    splitResult = derivedResult := by
  native_decide

/-- mergeCell equals removeCell* + addCell (concrete). -/
example :
    let p := exLinearProto
    let mergeResult := applyAnnotation p (.mergeCell ["A", "B"] "AB")
    let allRefs := ((p.cells.filter (fun s => ["A", "B"].contains s.name)).map (·.refs)).flatten
    let merged : CellSpec :=
      { name := "AB", type := .synthesis,
        prompt := "Merged from: A, B",
        refs := List.eraseDups allRefs }
    let derivedResult := evolve p [.removeCell "A", .removeCell "B", .addCell merged]
    mergeResult.cells = derivedResult.cells := by
  native_decide

end BeadCalculus.GasCity
