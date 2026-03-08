/-
  BeadCalculus.Formula — The formula language.

  A formula is a typed DAG of cells. Each cell has:
  - A name (unique within the formula)
  - A cell signature (typed inputs and output)
  - A prompt (the instruction an LLM receives to compute this cell)
  - Dependencies (which other cells' outputs feed into this cell's inputs)

  Formulas compose: you can plug one formula's output cells into another's
  input cells, creating a larger formula. This is the key operation that
  makes the bead calculus more than a workflow engine.
-/

import BeadCalculus.CellType
import BeadCalculus.DAG

namespace BeadCalculus

/-- A cell in a formula. The fundamental unit of computation.
    A cell is evaluated by giving an LLM the prompt along with
    the values of all input ports. -/
structure Cell where
  name   : String
  sig    : CellSig
  prompt : String  -- The LLM instruction for computing this cell
  deriving Repr

/-- A wire connects an output port of one cell to an input port of another.
    Wires carry typed values through the DAG. -/
structure Wire where
  source     : String  -- source cell name
  sourcePort : String  -- source output port name (usually just the cell's output)
  target     : String  -- target cell name
  targetPort : String  -- target input port name
  deriving Repr

/-- A formula is a collection of cells and wires forming a typed DAG.
    This is the "spreadsheet definition" — it says what cells exist,
    how they're connected, and what each cell computes. -/
structure Formula where
  name  : String
  cells : List Cell
  wires : List Wire
  deriving Repr

/-- A cell value — the result of evaluating a cell. -/
inductive CellValue where
  | pending   : CellValue                    -- Not yet computed
  | computing : CellValue                    -- Being computed by an LLM
  | computed  : (content : String) → CellValue  -- Successfully computed
  | failed    : (error : String) → CellValue    -- Failed to compute
  deriving Repr

/-- A formula state tracks the current value of each cell. -/
structure FormulaState where
  formula : Formula
  values  : String → CellValue  -- cell name → current value

/-- A cell is ready to evaluate when all its input wires have computed values. -/
def FormulaState.cellReady (state : FormulaState) (cellName : String) : Prop :=
  -- The cell exists
  (∃ c ∈ state.formula.cells, c.name = cellName) ∧
  -- The cell is pending
  state.values cellName = CellValue.pending ∧
  -- All input wires to this cell have computed source values
  ∀ w ∈ state.formula.wires,
    w.target = cellName →
    ∃ content, state.values w.source = CellValue.computed content

/-- Initial state: all cells pending. -/
def Formula.initState (f : Formula) : FormulaState where
  formula := f
  values := fun _ => CellValue.pending

/-- Evaluate a cell: transition from pending to computed.
    This models the effect of an LLM computing the cell's value. -/
def FormulaState.evaluate (state : FormulaState) (cellName : String)
    (result : String) : FormulaState where
  formula := state.formula
  values := fun n => if n = cellName then CellValue.computed result else state.values n

/-- Mark a cell as failed. -/
def FormulaState.fail (state : FormulaState) (cellName : String)
    (error : String) : FormulaState where
  formula := state.formula
  values := fun n => if n = cellName then CellValue.failed error else state.values n

/-- A formula is complete when all cells have non-pending values. -/
def FormulaState.complete (state : FormulaState) : Prop :=
  ∀ c ∈ state.formula.cells,
    state.values c.name ≠ CellValue.pending ∧
    state.values c.name ≠ CellValue.computing

/-- Formula composition: connect one formula's outputs to another's inputs.
    This is the key operation. Given formulas A and B and a wiring that connects
    A's output cells to B's input cells, produce a combined formula A ⊗ B. -/
def Formula.compose (a b : Formula) (connections : List Wire) : Formula where
  name := s!"{a.name} ⊗ {b.name}"
  cells := a.cells ++ b.cells
  wires := a.wires ++ b.wires ++ connections

/-- A formula is well-typed if all wires connect compatible ports. -/
def Formula.wellTyped (f : Formula) : Prop :=
  ∀ w ∈ f.wires,
    -- Source cell exists and has the named output port
    (∃ sc ∈ f.cells, sc.name = w.source ∧ sc.sig.output.name = w.sourcePort) ∧
    -- Target cell exists and has the named input port
    (∃ tc ∈ f.cells, tc.name = w.target ∧ ∃ p ∈ tc.sig.inputs, p.name = w.targetPort) ∧
    -- The types are compatible
    (∃ sc ∈ f.cells, ∃ tc ∈ f.cells,
      sc.name = w.source ∧ tc.name = w.target ∧
      ∃ tp ∈ tc.sig.inputs, tp.name = w.targetPort ∧
      sc.sig.output.type = tp.type)

end BeadCalculus
