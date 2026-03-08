# Paradigm A/B/C Test: YAML-first vs Bead-first vs Molecule Lifecycle Reactive Sheets

## Context

Gas City reactive sheets are DAGs of typed cells with staleness propagation.
We need to decide: is the sheet YAML the source of truth that stamps into beads,
or are beads themselves the cells (no YAML intermediary)?

## Paradigm A: YAML-first

The sheet is defined in a YAML file. A command (`gt sheet bind`) creates beads
from it, wiring dependencies and setting labels. The YAML is the schema;
beads are the runtime instances.

```
┌──────────────┐     gt sheet bind     ┌──────────────┐
│ formula.yaml │ ───────────────────▶  │ beads + labels│
│  (schema)    │                       │  (runtime)    │
└──────────────┘                       └──────────────┘
```

### How it works

1. Author writes `formula.yaml` defining cells, types, refs, prompts
2. `gt sheet bind formula.yaml` creates one bead per cell:
   - Bead title = cell name
   - Labels: `cell:empty`, `cell:text` (or `cell:inventory`, etc.)
   - Dependencies: `--deps` matching `refs:`
3. `gt sheet ready formula.yaml` queries bead labels to find ready cells
4. `gt sheet eval formula.yaml extract "result"` → sets bead value, updates labels:
   - Remove `cell:empty`/`cell:stale`, add `cell:fresh`
   - `bd label propagate` pushes `cell:stale` to downstream beads
5. The YAML file is the canonical definition; beads are derived

### Example: Code Review Pipeline

```yaml
name: code-review
cells:
  - name: read-code
    type: text
    prompt: "Read the source files in src/"

  - name: find-bugs
    type: inventory
    prompt: "Given this code: {{read-code}}, list potential bugs."
    refs: [read-code]

  - name: find-patterns
    type: inventory
    prompt: "Given this code: {{read-code}}, list design patterns used."
    refs: [read-code]

  - name: review-report
    type: synthesis
    prompt: |
      Bugs found: {{find-bugs}}
      Patterns found: {{find-patterns}}
      Write a comprehensive code review.
    refs: [find-bugs, find-patterns]
```

Usage:
```bash
gt sheet bind code-review.yaml          # Creates 4 beads
gt sheet ready code-review.yaml         # → [read-code]
gt sheet eval code-review.yaml read-code "function foo() { ... }"
gt sheet ready code-review.yaml         # → [find-bugs, find-patterns]
```

### Example: Onboarding Analysis

```yaml
name: onboarding
cells:
  - name: codebase
    type: text
    prompt: "Read the repository structure and key files."

  - name: docs
    type: text
    prompt: "Read README, CONTRIBUTING, and architecture docs."

  - name: architecture
    type: synthesis
    prompt: "Codebase: {{codebase}}. Docs: {{docs}}. Describe the architecture."
    refs: [codebase, docs]

  - name: getting-started
    type: decision
    prompt: "Given architecture: {{architecture}}, write a getting-started guide."
    refs: [architecture]
```

---

## Paradigm B: Bead-first

There is no YAML file. Beads ARE the cells. You create beads with labels
and dependencies directly. The DAG emerges from bead relationships.

```
┌──────────────┐
│ bd create    │ ──▶ bead with cell:empty, cell:text labels
│ bd update    │ ──▶ deps wired via --deps
│ bd label     │ ──▶ state transitions via label add/remove
└──────────────┘
```

### How it works

1. Create beads directly with cell labels:
   ```bash
   bd create "read-code" --description="Read the source files" \
     -t task -p 2 --json
   bd label add bd-XX cell:empty cell:text
   ```
2. Wire dependencies:
   ```bash
   bd create "find-bugs" --description="List potential bugs from {{read-code}}" \
     -t task -p 2 --deps bd-XX --json
   bd label add bd-YY cell:empty cell:inventory
   ```
3. Ready = `bd ready` (already checks deps) + filter for `cell:empty` or `cell:stale`
4. Evaluate = close bead with result, transition labels:
   ```bash
   bd label remove bd-XX cell:empty
   bd label add bd-XX cell:fresh
   bd label propagate bd-XX cell:stale  # downstream gets stale
   ```
5. The beads ARE the source of truth; no external schema

### Example: Code Review Pipeline

```bash
# Create cells as beads
bd create "read-code" --description="Read the source files in src/" \
  -t task -p 2 --json
# → bd-a1

bd label add bd-a1 cell:empty cell:text

bd create "find-bugs" \
  --description="Given code from read-code, list potential bugs." \
  -t task -p 2 --deps bd-a1 --json
# → bd-b2

bd label add bd-b2 cell:empty cell:inventory

bd create "find-patterns" \
  --description="Given code from read-code, list design patterns used." \
  -t task -p 2 --deps bd-a1 --json
# → bd-c3

bd label add bd-c3 cell:empty cell:inventory

bd create "review-report" \
  --description="Combine bug list and pattern list into a review." \
  -t task -p 2 --deps bd-b2,bd-c3 --json
# → bd-d4

bd label add bd-d4 cell:empty cell:synthesis

# Check what's ready
bd ready --json  # → bd-a1 (read-code)

# Evaluate
bd label remove bd-a1 cell:empty
bd label add bd-a1 cell:fresh
bd label propagate bd-a1 cell:stale

bd ready --json  # → bd-b2, bd-c3
```

### Example: Onboarding Analysis

```bash
bd create "codebase" --description="Read the repository structure" \
  -t task -p 2 --json
# → bd-e5
bd label add bd-e5 cell:empty cell:text

bd create "docs" --description="Read README and architecture docs" \
  -t task -p 2 --json
# → bd-f6
bd label add bd-f6 cell:empty cell:text

bd create "architecture" \
  --description="Synthesize codebase and docs into architecture description" \
  -t task -p 2 --deps bd-e5,bd-f6 --json
# → bd-g7
bd label add bd-g7 cell:empty cell:synthesis

bd create "getting-started" \
  --description="Write getting-started guide from architecture" \
  -t task -p 2 --deps bd-g7 --json
# → bd-h8
bd label add bd-h8 cell:empty cell:decision
```

---

## Paradigm C: Molecule Lifecycle

Staleness is handled by RE-INSTANTIATION, not mutation. Completed work becomes
immutable (digests). Fresh execution is a new molecule poured from the same proto.

```
Proto (solid) ──pour──▶ Molecule (liquid) ──execute──▶ All closed
                                                           │
                                                      squash
                                                           │
                                                      Digest (immutable)
                                                           │
                                               upstream changes
                                                           │
                                              distill ──▶ Proto'
                                                           │
                                                      pour ──▶ Molecule₂
```

### How it works

1. Define a formula template (proto) — the cell DAG with prompts and types
2. `bd mol pour my-analysis --var source=X` creates a molecule: one bead per cell, deps wired
3. Execute cells in dependency order using normal bead operations:
   - `bd update <id> --claim` → begin compute
   - Do work, then `bd close <id> --reason "result"` → cell complete
   - `bd ready` finds next unblocked cells automatically
4. When all cells are closed → `bd mol squash <mol-id>` → compressed digest (immutable)
5. When upstream changes:
   - `bd mol distill <mol-id> my-analysis` → extract proto from completed work
   - `bd mol pour my-analysis --var source=X-v2` → new molecule instance
   - Only changed cells need re-evaluation; unchanged cells can seed from prior digest
6. Version history = chain of digests. Each is a complete snapshot.

### Example: Code Review Pipeline

```bash
# Define proto (one time)
# formula: mol-code-review with cells: read-code → find-bugs, find-patterns → review-report

# First run
bd mol pour mol-code-review --var repo=myapp
# → Creates molecule with 4 beads, deps wired

bd ready                    # → read-code bead
bd update bd-XX --claim     # Claim it
# ... do work ...
bd close bd-XX --reason "function foo() { ... }"

bd ready                    # → find-bugs, find-patterns (parallel)
# ... evaluate both ...

bd ready                    # → review-report
# ... evaluate ...

bd mol squash bd-mol-YY     # → Digest: immutable record of this review

# Code changes → re-run
bd mol pour mol-code-review --var repo=myapp
# New molecule, fresh beads. Old digest is history.
```

### Why this is better

- **Immutability**: Completed work is never mutated. No "reopen" ambiguity.
- **Audit trail**: Each digest is a complete, timestamped snapshot.
- **Natural versioning**: Digest chain = version history (like git commits).
- **Clean separation**: Proto = schema, Molecule = execution, Digest = history.
- **Bead algebra native**: Uses pour/squash/distill — no new primitives needed.

---

## Key Differences

| Aspect | A: YAML-first | B: Bead-first | C: Molecule lifecycle |
|--------|---------------|---------------|----------------------|
| Schema location | YAML file | Implicit in bead graph | Proto (formula template) |
| Cell creation | `gt sheet bind` (batch) | `bd create` (one by one) | `bd mol pour` (batch from proto) |
| Prompt templates | `{{ref}}` in YAML | Description text | `{{var}}` in proto |
| State transitions | Sheet engine manages | Manual label ops | Bead open/closed lifecycle |
| Re-evaluation | `gt sheet eval` | Label remove/add | Pour new molecule |
| Staleness | Engine PropagateStale | `bd label propagate` | Re-instantiation (new molecule) |
| Discoverability | `gt sheet status` | `bd list` + labels | `bd mol show`, `bd mol progress` |
| Portability | YAML copyable | Instance-bound | Proto is portable formula |
| History | Version counter | Label audit events | Chain of squashed digests |
| Immutability | Mutable state | Mutable labels | Digests are immutable |
