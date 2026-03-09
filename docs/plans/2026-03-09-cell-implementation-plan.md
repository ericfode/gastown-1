# Cell Language — Implementation Plan

**Date**: 2026-03-09
**Epic**: hq-7vk
**Status**: Phase 1 (bootstrap)

---

## Implementation Phases

### Phase 1: Bootstrap (Current)

**Goal**: Execute Cell files using existing Gas Town infrastructure.

| Deliverable | Status | Bead |
|-------------|--------|------|
| Cell language spec | DONE | hq-606 |
| EBNF grammar | DONE | hq-98i (furiosa) |
| Formula survey (44/44) | DONE | in spec |
| Cell pour formula (TOML) | DONE | — |
| DAG visualization mockup | DONE | — |
| Parser validation | IN PROGRESS | hq-qjx (slit) |
| CLI tool (parse + validate) | IN PROGRESS | hq-vsf (nux) |

**What works now**: A crew member can read a `.cell` file and execute it by
manually creating beads, wiring deps, and evaluating cells using `bd` commands.
The `cell-pour.formula.toml` documents this process.

### Phase 2: CLI Tool (~2000 LOC Go)

**Goal**: `cell pour file.cell` automates what the formula describes.

Components:
1. **Cell parser** — PEG grammar → AST (tree-sitter or Go PEG library)
2. **DAG builder** — AST → dependency graph, cycle detection
3. **Oracle runtime** — Execute oracle blocks against cell outputs
4. **Sheet↔Beads bridge** — Map Cell DAG to bd commands
5. **Prompt assembler** — Section interpolation, typed hole validation

Architecture:
```
cell pour hello.cell --preset production
  │
  ├─ parse(file) → AST
  ├─ validate(AST) → errors[]
  ├─ build_dag(AST) → DAG
  ├─ create_beads(DAG) → bead_map
  └─ loop:
       ├─ ready = bd ready (filtered to epic)
       ├─ for each ready cell:
       │    ├─ assemble_prompt(cell, outputs)
       │    ├─ evaluate(cell) → output
       │    ├─ run_oracle(cell, output) → verdict
       │    └─ close_bead(cell, output)
       └─ until done | fail | deadlock
```

### Phase 3: Native Runtime

**Goal**: Cell runtime integrated into Gas Town daemon.

- Hot-reload `.cell` files on filesystem change
- Live state tracking (fresh/stale/computing/empty)
- Incremental re-evaluation (only dirty cells)
- Visualization connects to runtime via WebSocket
- Content-addressed cell cache (blake3 hashing)

---

## Go Package Layout

```
gt/cmd/cell/           — CLI entry point
gt/cell/
  ├─ parser/           — PEG parser, AST types
  ├─ dag/              — DAG builder, cycle detection, topo sort
  ├─ oracle/           — Oracle statement evaluator
  ├─ prompt/           — Section assembly, ref interpolation
  ├─ bridge/           — bd command integration
  └─ content/          — Content addressing (blake3)
```

## Key Design Decisions

1. **No embedded language** — Oracles at the seams. Shell for computation.
   If wrong, add `expr` cells later.

2. **bd as execution engine** — Cell doesn't need its own scheduler.
   `bd ready` IS the ready set. `bd close` IS cell completion.

3. **Formula-first bootstrap** — Agents can pour Cell files TODAY using the
   TOML formula, before any Go code ships.

4. **Content addressing deferred** — Phase 3 concern. Phase 2 uses bead IDs.

5. **Visualization decoupled** — `cell-graph.html` works standalone.
   Phase 3 wires it to the runtime.

---

## Work Breakdown

### Remaining Phase 1 tasks
- [ ] Collect furiosa's formal grammar and merge into spec
- [ ] Complete parser validation (slit, hq-qjx)
- [ ] Complete CLI tool MVP (nux, hq-vsf)
- [ ] Test-pour a real formula (e.g., code-review) through the bootstrap formula

### Phase 2 tasks (future epic)
- [ ] Go PEG parser for Cell grammar
- [ ] DAG builder with cycle detection
- [ ] Oracle runtime (json_parse, keys_present, assert, score)
- [ ] Prompt assembler (section interpolation, typed holes)
- [ ] Sheet↔Beads bridge (bd command layer)
- [ ] `cell pour` CLI command
- [ ] `cell validate` CLI command
- [ ] `cell graph` CLI command (output DOT/Mermaid)
- [ ] Integration tests with real formulas

### Phase 3 tasks (future epic)
- [ ] Live runtime daemon mode
- [ ] WebSocket state feed for visualization
- [ ] Content-addressed cell cache
- [ ] Incremental re-evaluation
- [ ] Hot-reload on file change
