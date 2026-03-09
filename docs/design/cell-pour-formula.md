# Cell Pour — Bootstrap Execution Strategy

**Date**: 2026-03-09
**Epic**: hq-7vk (Cell Language — Formula Engine v2 DSL)

---

## Problem

We need to execute `.cell` files. Rather than building a parser/runtime from
scratch first, we bootstrap: a **formula teaches crew members to pour Cell
files** using existing Gas Town APIs.

## Approach: Formula-as-Interpreter

A TOML formula (`cell-pour`) that any crew member can run. The formula's steps
walk the agent through:

1. **Parse** — Read the `.cell` file, identify molecules, cells, wires
2. **Plan** — Build a dependency graph, identify ready cells
3. **Execute** — For each ready cell, create a bead and evaluate it
4. **Wire** — Connect outputs as dependencies, propagate results
5. **Gate** — Run oracle blocks, accept/reject, retry if needed
6. **Report** — Summarize execution, show outputs

## Cell → Gas Town API Mapping

| Cell Construct | Gas Town API | Command |
|---------------|-------------|---------|
| Molecule `## name` | Epic | `bd create "name" -t epic` |
| Cell `# name : llm` | Bead | `bd create "name" -t task` |
| Cell `# name : script` | Shell task | `bd create "name" -t task` + shell |
| Wire `a -> b` | Dependency | `bd update <b> --deps <a>` |
| Ref `- upstream` | Dependency | `bd update <cell> --deps <upstream>` |
| `map # name over ref` | N beads | Create one bead per item |
| `reduce # name` | Accumulator bead | Sequential reduce via deps |
| Oracle block | Validation step | Agent validates output |
| `preset name` | Parameter set | Inject as bead metadata |
| `input param.x` | Formula parameter | Map to formula input |
| `mol(other)` | Sub-epic | Nested epic execution |
| Conditional wire `?oracle` | Gated dep | Oracle check before wiring |

## Ready-Set Execution Model

Cell's evaluation is isomorphic to `bd ready`:

```
1. Parse .cell → build DAG
2. Create beads for all cells (state: blocked)
3. Wire dependencies
4. Loop:
   a. bd ready → get unblocked beads
   b. For each ready bead:
      - Assemble prompt (interpolate refs from completed deps)
      - If LLM cell: send to LLM, capture output
      - If script cell: execute shell block, capture output
      - Run oracle if present
      - If oracle PASS: bd close <id>
      - If oracle FAIL: retry (up to @retry limit)
   c. Repeat until all cells complete or error
```

## Bootstrap Sequence

**Phase 1: Agent-interpreted** (now)
- Formula teaches agents to manually pour Cell files
- Agent reads `.cell`, creates beads, wires deps, evaluates

**Phase 2: CLI tool** (next)
- `cell pour file.cell` — automated parser + executor
- Calls `bd` commands under the hood
- ~2000 LOC Go (parser, oracle runtime, Sheet↔Beads bridge)

**Phase 3: Native runtime** (later)
- Cell runtime integrated into Gas Town daemon
- Hot-reload, live state, incremental re-evaluation
- The visualization (cell-graph.html) connects to this

## Example Pour Session

Given `hello.cell`:
```cell
## hello-world
  # greet : llm
    system>
      You are friendly.
    user>
      Say hello to {{param.name}}.
    format> greeting
      { message: str, emoji: str }

  input param.name : str required
##/
```

Agent execution:
```bash
# 1. Create epic
bd create "hello-world" -t epic --json
# → bd-abc

# 2. Create cell bead
bd create "greet" -t task -p 2 --deps parent:bd-abc --json
# → bd-def

# 3. No upstream deps → cell is ready immediately

# 4. Assemble prompt
# system: "You are friendly."
# user: "Say hello to Alice."  (param.name = "Alice")

# 5. LLM call (agent does this naturally)
# Output: {"message": "Hello Alice!", "emoji": "👋"}

# 6. Validate format oracle
# json_parse? ✓  keys_present(message, emoji)? ✓

# 7. Close bead with output
bd close bd-def --reason '{"message":"Hello Alice!","emoji":"👋"}' --json

# 8. All cells complete → molecule done
bd close bd-abc --reason "All cells evaluated" --json
```
