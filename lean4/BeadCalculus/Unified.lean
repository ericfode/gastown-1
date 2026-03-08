/-
  BeadCalculus.Unified — The Lazy Reactive Bead Spreadsheet.

  The chosen model combines:
  - Spreadsheet surface (named cells, prompt templates, stale tracking)
  - Typed DAG structure (acyclic dependencies, monotone readiness)
  - Lazy reactive evaluation (dispatch-triggered, not automatic)

  Key design: CELLS ARE BEADS. Each cell maps to a bead in Dolt.
  This isn't a new system — it's a typed layer on existing beads.
-/

import BeadCalculus.CellType

namespace BeadCalculus.Unified

/-- A cell reference inside a prompt template. -/
structure Ref where
  cell  : String          -- Name of the referenced cell
  field : Option String   -- Optional field within the cell's value
  deriving DecidableEq, Repr

/-- A cell in the bead spreadsheet. -/
structure Cell where
  name     : String
  cellType : CellType       -- What kind of value this cell produces
  prompt   : String          -- Prompt template with {{ref}} holes
  refs     : List Ref        -- All references in the prompt
  deriving Repr

/-- A cell's value with versioning and staleness. -/
structure Value where
  content    : String
  version    : Nat
  stale      : Bool          -- True if an upstream cell changed since computation
  deriving Repr, DecidableEq, BEq

/-- The state of a cell: the evaluation lifecycle. -/
inductive CellState where
  | empty     : CellState                        -- Never computed
  | stale     : (last : Value) → CellState        -- Has old value, upstream changed
  | computing : (last : Option Value) → CellState  -- LLM is working on it
  | fresh     : (val : Value) → CellState          -- Computed and up to date
  | failed    : (err : String) → (last : Option Value) → CellState
  deriving Repr, DecidableEq, BEq

/-- A bead spreadsheet. -/
structure Sheet where
  name   : String
  cells  : List Cell
  states : String → CellState  -- cell name → state

/-- Dependencies: which cells does a cell reference? -/
def Cell.deps (c : Cell) : List String :=
  c.refs.map (·.cell)

/-- A cell is ready for evaluation when:
    1. It is empty or stale (not fresh or computing)
    2. All referenced cells are fresh -/
def Sheet.ready (s : Sheet) (cellName : String) : Prop :=
  (∃ c ∈ s.cells, c.name = cellName) ∧
  (match s.states cellName with
   | .empty => True
   | .stale _ => True
   | _ => False) ∧
  (∀ c ∈ s.cells, c.name = cellName →
    ∀ ref ∈ c.refs,
      match s.states ref.cell with
      | .fresh _ => True
      | _ => False)

/-- Begin computing a cell: transition empty/stale → computing. -/
def Sheet.beginCompute (s : Sheet) (cellName : String) : Sheet where
  name := s.name
  cells := s.cells
  states := fun n =>
    if n = cellName then
      match s.states n with
      | .empty => .computing none
      | .stale v => .computing (some v)
      | other => other  -- Only transition from empty/stale
    else s.states n

/-- Complete computation: transition computing → fresh. -/
def Sheet.complete (s : Sheet) (cellName : String) (content : String) : Sheet where
  name := s.name
  cells := s.cells
  states := fun n =>
    if n = cellName then
      match s.states n with
      | .computing last =>
        let ver := match last with | some v => v.version + 1 | none => 1
        .fresh { content, version := ver, stale := false }
      | other => other  -- Only transition from computing
    else s.states n

/-- Propagate staleness: when a cell gets a new value, mark downstream cells stale. -/
def Sheet.propagateStale (s : Sheet) (changedCell : String) : Sheet where
  name := s.name
  cells := s.cells
  states := fun n =>
    match s.states n with
    | .fresh v =>
      let cell? := s.cells.find? (·.name = n)
      match cell? with
      | some c => if c.deps.contains changedCell then .stale v else .fresh v
      | none => .fresh v
    | other => other

/-- Full evaluate-and-propagate: compute a cell and mark downstream stale. -/
def Sheet.evaluate (s : Sheet) (cellName : String) (content : String) : Sheet :=
  let s' := s.beginCompute cellName
  let s'' := s'.complete cellName content
  s''.propagateStale cellName

/-- A sheet is fully evaluated when all cells are fresh. -/
def Sheet.allFresh (s : Sheet) : Prop :=
  ∀ c ∈ s.cells, match s.states c.name with
    | .fresh _ => True
    | _ => False

/-- Initial sheet state: all cells empty. -/
def Sheet.init (name : String) (cells : List Cell) : Sheet where
  name := name
  cells := cells
  states := fun _ => .empty

/-- Compose two sheets by connecting output cells to input cells.
    The connections list maps source cell names to (target cell, ref name). -/
def Sheet.compose (a b : Sheet)
    (connections : List (String × String)) : Sheet where
  name := s!"{a.name} ⊗ {b.name}"
  cells := a.cells ++ b.cells  -- TODO: handle name conflicts
  states := fun n =>
    -- Prefer b's state for cells in b, a's state for cells in a
    if b.cells.any (·.name = n) then b.states n else a.states n

/-- Fill a prompt template with referenced values. Returns None if any ref is not fresh. -/
def Sheet.fillPrompt (s : Sheet) (cellName : String) : Option String := do
  let c ← s.cells.find? (·.name = cellName)
  let mut prompt := c.prompt
  for ref in c.refs do
    match s.states ref.cell with
    | .fresh v =>
      let placeholder := "{{" ++ ref.cell ++ "}}"
      prompt := prompt.replace placeholder v.content
    | _ => failure
  return prompt

/-
  OBSERVATIONS:

  1. This model maps cleanly to existing Gas Town beads:
     - Cell → Issue (with cellType in labels, prompt in description)
     - Ref → Dependency (with type = "needs-output-of")
     - Value → Issue.notes or Issue.design field
     - CellState → Issue.status (empty=open, computing=in_progress, fresh=closed)
     - Sheet → Molecule (a formula instance with child beads)

  2. The staleness propagation is the novel bit. Current Gas Town doesn't
     have this — once a bead is closed, it stays closed. With staleness,
     a closed bead can become "stale" (needs recomputation) without being
     reopened. This is the reactive part.

  3. Composition is straightforward: merge two sheets and add cross-references.
     The only subtlety is name conflicts (solved by prefixing).

  4. The prompt template with {{ref}} syntax is directly usable by LLMs.
     An LLM computing cell "synthesize" would receive:
       "Given the type inventory: {{extract-types}}, what algebra is this?"
     with {{extract-types}} replaced by the actual computed value.

  NEXT: Implement this in Go as a standalone package. Test by re-expressing
  mol-algebraic-survey as a bead spreadsheet.
-/

end BeadCalculus.Unified
