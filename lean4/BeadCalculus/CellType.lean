/-
  BeadCalculus.CellType — The type system for reactive bead cells.

  A cell in the bead calculus has a type that describes what kind of value it holds.
  Cell types are the "column types" of our spreadsheet. They determine what an LLM
  is asked to produce and what downstream cells can expect as input.

  Design choice: cell types are a CLOSED enumeration, not an open universe.
  This is deliberate — we want to be able to pattern-match on types and prove
  exhaustiveness. New types require extending the enum, which is a conscious act.

  The type system is kept simple on purpose. We're not building a general-purpose
  type theory. We're building a vocabulary for describing structured LLM outputs.
-/

namespace BeadCalculus

/-- The kinds of values a cell can hold. Each corresponds to a structured
    output that an LLM can produce and that downstream cells can consume. -/
inductive CellType where
  | text       : CellType  -- Unstructured text (fallback)
  | inventory  : CellType  -- A named list of items with properties
  | diagram    : CellType  -- A state machine or graph description
  | laws       : CellType  -- A list of formal/semi-formal invariants
  | boundaries : CellType  -- A map of connections to other subsystems
  | synthesis  : CellType  -- A summary judgment combining multiple inputs
  | code       : CellType  -- Source code (Lean, Go, etc.)
  | decision   : CellType  -- A yes/no/conditional judgment with rationale
  deriving DecidableEq, Repr

/-- Cell types form a simple subtyping relation: every type can be projected to text. -/
def CellType.toText : CellType → CellType
  | _ => CellType.text

/-- A typed port — a named connection point with a cell type.
    Ports are how cells declare their inputs and outputs. -/
structure Port where
  name : String
  type : CellType
  deriving DecidableEq, Repr

/-- A cell signature — the interface of a cell.
    This is the "function type" of a cell: what it needs and what it produces.
    Think of it as: inputs → output, where the computation is done by an LLM. -/
structure CellSig where
  inputs  : List Port
  output  : Port
  deriving Repr

/-- Two ports are compatible if they have the same type.
    This is the basic type-checking rule for connecting cells. -/
def Port.compatible (p q : Port) : Prop :=
  p.type = q.type

instance : Decidable (Port.compatible p q) :=
  inferInstanceAs (Decidable (p.type = q.type))

/-- Check that an input port name exists in a signature. -/
def CellSig.hasInput (sig : CellSig) (name : String) : Prop :=
  ∃ p ∈ sig.inputs, p.name = name

end BeadCalculus
