/-
  BeadCalculus.ProcessModel — Model 2: Process calculus with channels.

  Instead of a DAG of cells, model formulas as communicating processes.
  Each step is a process. Channels connect processes. Composition is
  parallel composition with channel binding (like the pi-calculus).

  This model is more dynamic than the dataflow model — processes can
  create new channels, fork, and synchronize. But it may be overkill
  for what we need.
-/

namespace BeadCalculus.Process

/-- A channel carries values of a given type between processes. -/
structure Channel where
  name : String
  type : String  -- simplified; would be CellType in full model
  deriving DecidableEq, Repr

/-- A process is the basic unit of computation. -/
inductive Proc where
  | nil     : Proc                                    -- Terminated
  | send    : Channel → String → Proc → Proc          -- Send value on channel, then continue
  | recv    : Channel → (String → Proc) → Proc        -- Receive value from channel, continue with it
  | par     : Proc → Proc → Proc                      -- Parallel composition
  | choice  : Proc → Proc → Proc                      -- Non-deterministic choice
  | new     : String → (Channel → Proc) → Proc        -- Create fresh channel

/-- A formula in the process model is a top-level parallel composition
    of named processes with channel bindings. -/
structure ProcessFormula where
  name     : String
  channels : List Channel
  procs    : List (String × Proc)  -- named processes

/-- Composition of process formulas: parallel compose with channel linking. -/
def ProcessFormula.compose (a b : ProcessFormula)
    (links : List (Channel × Channel)) : ProcessFormula where
  name := s!"{a.name} | {b.name}"
  channels := a.channels ++ b.channels  -- TODO: rename to avoid conflicts
  procs := a.procs ++ b.procs

/-
  Observation after writing this:

  The process model is more expressive than we need. Gas Town formulas
  don't need dynamic channel creation (new) or non-deterministic choice.
  They need:
  1. Typed data flow from step to step (dataflow model handles this)
  2. Parallel execution of independent steps (parallel composition)
  3. Synchronization at join points (channel recv)

  The process model CAN do all of this, but it's like using lambda calculus
  to model a spreadsheet — technically correct, too powerful to be useful.

  The pi-calculus gives us bisimulation as an equivalence notion, which is
  interesting: two formulas are equivalent if they produce the same observable
  behavior on all channels. But for LLM orchestration, "same behavior" is
  hard to define because LLM outputs are non-deterministic.

  Verdict: The process model is the RIGHT theoretical foundation but the
  WRONG user-facing model. Keep it as the semantic backend, use the
  dataflow model as the surface language.
-/

end BeadCalculus.Process
