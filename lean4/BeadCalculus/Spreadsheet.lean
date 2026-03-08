/-
  BeadCalculus.Spreadsheet — Model 3: Literal spreadsheet.

  The most concrete model. A spreadsheet where:
  - Cells are named (like A1, B2 — but with semantic names)
  - Each cell has a formula that references other cells
  - Values are computed by LLMs
  - Changing an upstream cell triggers recomputation downstream

  This is what a user would actually interact with. The question is:
  what's the right level of abstraction?
-/

namespace BeadCalculus.Sheet

/-- A cell reference — how one cell refers to another. -/
structure CellRef where
  cellName : String
  field    : Option String  -- optional field within the cell's value
  deriving DecidableEq, Repr

/-- A cell formula — the computation that produces a cell's value.
    This is where LLMs come in. The formula is a prompt template
    with holes that get filled by referenced cells' values. -/
structure CellFormula where
  prompt : String           -- The template. References like {{types.inventory}} get filled.
  refs   : List CellRef     -- All cell references in the prompt
  deriving Repr

/-- A spreadsheet cell. -/
structure Cell where
  name    : String
  formula : CellFormula
  deriving Repr

/-- A cell's computed value, with versioning. -/
structure CellValue where
  content   : String
  version   : Nat           -- Increments on each recomputation
  computedBy : String       -- Which LLM/agent computed this
  stale     : Bool          -- True if an upstream cell has changed since computation
  deriving Repr

/-- A spreadsheet. -/
structure Spreadsheet where
  name  : String
  cells : List Cell
  values : String → Option CellValue  -- cell name → current value (None = not computed)

/-- Dependency graph: which cells does a cell depend on? -/
def Cell.deps (c : Cell) : List String :=
  c.formula.refs.map (·.cellName)

/-- A cell is computable when all its dependencies have non-stale values. -/
def Spreadsheet.computable (s : Spreadsheet) (cellName : String) : Prop :=
  ∃ c ∈ s.cells, c.name = cellName ∧
    (∀ ref ∈ c.formula.refs,
      ∃ v, s.values ref.cellName = some v ∧ v.stale = false)

/-- Mark all downstream cells as stale after a cell is recomputed. -/
def Spreadsheet.propagateStale (s : Spreadsheet) (changedCell : String) : Spreadsheet where
  name := s.name
  cells := s.cells
  values := fun n =>
    match s.values n with
    | none => none
    | some v =>
      -- Check if this cell depends (transitively) on changedCell
      -- For now, simplified: mark as stale if direct dependency
      let cell? := s.cells.find? (·.name = n)
      match cell? with
      | none => some v
      | some c =>
        if c.deps.contains changedCell then
          some { v with stale := true }
        else
          some v

/-- Compute a cell: fill in the prompt template with referenced values.
    Returns the filled prompt that would be sent to an LLM. -/
def Spreadsheet.fillPrompt (s : Spreadsheet) (cellName : String) : Option String := do
  let c ← s.cells.find? (·.name = cellName)
  let mut prompt := c.formula.prompt
  for ref in c.formula.refs do
    let v ← s.values ref.cellName
    let placeholder := "{{" ++ ref.cellName ++ "}}"
    prompt := prompt.replace placeholder v.content
  return prompt

/-
  Observation after writing this:

  The spreadsheet model is the most intuitive. It maps directly to what
  users (and LLMs) understand. But it has a problem: VERSIONING.

  When an LLM recomputes a cell, it gets a new value. This invalidates
  downstream cells (they become stale). But the old values are still
  useful — they're the "last known good" state. We need:

  1. Version history per cell (like git for values)
  2. Stale detection (which cells need recomputation?)
  3. Minimal recomputation (only recompute what actually changed)

  This is exactly what a reactive spreadsheet does. Excel, Google Sheets,
  Observable — they all solve this problem. The twist is that our "formulas"
  are LLM calls, which are:
  - Non-deterministic (same inputs → different outputs)
  - Expensive (tokens cost money, time costs context)
  - Qualitative (we can't diff two LLM outputs automatically)

  This means reactive recomputation has to be DELIBERATE, not automatic.
  The user (or orchestrator) decides when to recompute a stale cell.
  The spreadsheet tracks what's stale, but doesn't auto-recompute.

  This is the key insight: the formula engine is a LAZY reactive spreadsheet
  where recomputation is triggered by dispatch (sling), not by value change.

  VERDICT: The spreadsheet model is the right USER-FACING abstraction.
  Combined with Model 1's typed DAG for the internal representation and
  Model 2's process semantics for composition, we get:

  Surface: Spreadsheet (named cells, prompt templates, stale tracking)
  Structure: Typed DAG (acyclic dependencies, monotone readiness)
  Semantics: Lazy reactive evaluation (dispatch-triggered, not auto)
-/

end BeadCalculus.Sheet
