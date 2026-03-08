import BeadCalculus

open BeadCalculus

-- Example: a tiny formula with two cells
def exampleFormula : Formula where
  name := "tiny-survey"
  cells := [
    { name := "extract-types"
      sig := { inputs := [], output := { name := "types", type := .inventory } }
      prompt := "Read the Go source and list all algebraic types." },
    { name := "synthesize"
      sig := { inputs := [{ name := "types", type := .inventory }],
               output := { name := "synthesis", type := .synthesis } }
      prompt := "Given the type inventory, what algebra is this?" }
  ]
  wires := [
    { source := "extract-types", sourcePort := "types",
      target := "synthesize", targetPort := "types" }
  ]

def main : IO Unit := do
  IO.println s!"Formula: {exampleFormula.name}"
  IO.println s!"Cells: {exampleFormula.cells.length}"
  IO.println s!"Wires: {exampleFormula.wires.length}"
  let state := exampleFormula.initState
  IO.println "Initial state created."
  let state' := state.evaluate "extract-types" "Found 5 types: Status, Role, Bead, Agent, Wire"
  IO.println "After evaluating extract-types."
  let state'' := state'.evaluate "synthesize" "This is a typed actor system."
  IO.println "After evaluating synthesize."
  IO.println "Done."
