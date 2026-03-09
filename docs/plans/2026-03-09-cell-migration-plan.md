# Cell Migration Plan — Replace All Gas Town Formulas

**Date**: 2026-03-09
**Epic**: hq-7vk (Cell Language — Formula Engine v2 DSL)
**Strategy**: Staged migration as test case for language development

---

## Philosophy

Each migration stage tests a new layer of Cell language complexity. We don't
just port formulas — we **prove the language works** by running increasingly
complex programs through it. Failures at each stage feed back into the grammar
and runtime.

## Inventory

44 TOML formulas total. Classified by Cell complexity:

| Tier | Count | Description |
|------|-------|-------------|
| T1 Simple | 7 | Linear molecules, Dog infra tasks |
| T2 Convoy | 7 | Parallel-leg reviews + Hanoi proofs |
| T3 Molecule (simple) | 16 | Release pipelines, basic orchestration |
| T4 Molecule (complex) | 10 | Patrol loops, state machines, sub-mol |
| T5 Expansion/Meta | 4 | AOP aspects, composition |

Migration order: T1 → T5 → T2 → T3 → T4
(T5 early because it defines composition primitives others need)

---

## Stage 1: Sequential Workflows (Language Core)
**Tests**: cells, wires, refs, prompt sections, format, oracle, input params

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| shiny.formula.toml | 1.8K | Sequential deps, accept blocks |
| rule-of-five.formula.toml | 1.2K | Simple evaluation |
| security-audit.formula.toml | 1.2K | Single-purpose analysis |
| towers-of-hanoi.formula.toml | 2.7K | Recursive structure |

**Gate**: All 4 parse cleanly, execute via `cell pour`, produce equivalent output.

**Language features validated**:
- `# name : type ... #/` cell declarations
- `- ref` dependency declarations
- `system>`, `user>`, `format>`, `accept>` prompt sections
- `input param.X : type` declarations
- `{{ref}}` interpolation
- Oracle blocks (json_parse, keys_present, assert)

---

## Stage 2: Parallel Convoys (Map + Synthesis)
**Tests**: map combinator, prompt fragments, parallel execution, synthesis

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| code-review.formula.toml | 12.7K | 10 parallel legs + synthesis |
| design.formula.toml | 8.6K | 6 parallel legs + synthesis |
| mol-polecat-code-review.formula.toml | 8.3K | Polecat convoy variant |
| mol-prd-review.formula.toml | 11.2K | PRD analysis convoy |
| mol-plan-review.formula.toml | 9.7K | Plan analysis convoy |

**Gate**: All 5 parse, map cells spawn correctly, synthesis collects all legs.

**Language features validated**:
- `map # name : llm over {{ref}} as var`
- `prompt@ name` fragments and `{{@fragment}}` refs
- `preset name { ... }` parameter sets
- `each> var in {{ref}}` iteration in prompts
- Parallel execution semantics (no deps between legs)

---

## Stage 3: Simple Molecules (Shell + Script Cells)
**Tests**: script cells, shell execution, conditional logic

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| mol-gastown-boot.formula.toml | 4.2K | Boot sequence, shell commands |
| mol-dog-backup.formula.toml | 3.8K | File operations |
| mol-polecat-lease.formula.toml | 4.9K | Lease management |
| mol-town-shutdown.formula.toml | 4.5K | Graceful shutdown |
| mol-session-gc.formula.toml | 6.7K | Cleanup operations |
| mol-dog-phantom-db.formula.toml | 4.6K | Database operations |

**Gate**: Script cells execute shell, outputs captured, wired to downstream.

**Language features validated**:
- `# name : script` with ```bash blocks
- Shell output capture as cell output
- Mixed LLM + script cell pipelines
- `squash>` molecule completion directives

---

## Stage 4: Complex Molecules (Loops, Conditions, Sub-molecules)
**Tests**: conditional wires, OR-join, sub-molecule invocation, reduce

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| mol-boot-triage.formula.toml | 5.5K | Decision tree, conditional actions |
| mol-convoy-cleanup.formula.toml | 5.1K | Cleanup with conditions |
| mol-convoy-feed.formula.toml | 7.2K | Feed processing |
| mol-dep-propagate.formula.toml | 5.9K | Dependency propagation |
| mol-digest-generate.formula.toml | 5.7K | Content generation |
| mol-dog-compactor.formula.toml | 6.4K | Compaction with conditions |
| mol-dog-doctor.formula.toml | 5.5K | Diagnostic + repair |
| mol-dog-jsonl.formula.toml | 5.9K | JSON-L processing |
| mol-dog-reaper.formula.toml | 6.9K | Resource reaping |
| mol-dog-stale-db.formula.toml | 5.8K | Stale detection + cleanup |
| mol-orphan-scan.formula.toml | 9.8K | Scan + triage |
| mol-sync-workspace.formula.toml | 13.3K | Complex workspace sync |
| mol-polecat-conflict-resolve.formula.toml | 10.0K | Conflict resolution |
| mol-polecat-review-pr.formula.toml | 7.9K | PR review workflow |

**Gate**: Conditional wires fire correctly, sub-molecules complete, loops bounded.

**Language features validated**:
- `wire -> ?oracle -> target` conditional wires
- `- ref (or)` OR-join semantics
- `# name : mol(other)` sub-molecule invocation
- `reduce # name over ref as item with acc = init`
- `# name : decision` cells with routing
- `@retry(max: N)` bounded retry

---

## Stage 5: Patrols + Orchestration (Full Complexity)
**Tests**: import/apply, recipes, metacircular cells, full orchestration

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| mol-deacon-patrol.formula.toml | 36.1K | Complex multi-phase patrol |
| mol-witness-patrol.formula.toml | 31.6K | Observation + reporting |
| mol-refinery-patrol.formula.toml | 28.2K | Quality enforcement |
| mol-polecat-work.formula.toml | 15.1K | Generic work execution |
| mol-shutdown-dance.formula.toml | 15.2K | Orchestrated shutdown |
| mol-idea-to-plan.formula.toml | 11.9K | Multi-stage planning |
| mol-survey-dispatch.formula.toml | 10.4K | Survey orchestration |
| mol-algebraic-survey.formula.toml | 20.5K | Deep analysis workflow |

**Gate**: Full formula parity. Every TOML formula has a Cell equivalent that
produces identical behavior.

**Language features validated**:
- `import name` + `apply name() where selector`
- `recipe name() { !add, !drop, !wire, !cut }` graph operations
- `meta # ... meta #/` metacircular cells
- Content addressing (blake3 hashing)
- Full Gas Town API integration

---

## Stage 6: Expansion Formulas (AOP Patterns)
**Tests**: import/apply/selectors close the AOP gap

| Formula | Size | Cell Features Exercised |
|---------|------|------------------------|
| shiny-secure.formula.toml | 182B | Extends shiny with security |
| shiny-enterprise.formula.toml | 258B | Extends shiny with enterprise gates |
| beads-release.formula.toml | 6.3K | Release with multi-stage gates |
| gastown-release.formula.toml | 9.4K | Full release pipeline |
| towers-of-hanoi-7/9/10.formula.toml | 19-145K | Generated recursive structures |

**Gate**: Import/apply correctly compose base formulas with extensions.

**Language features validated**:
- `import shiny` base formula import
- `apply security-gate(review, test) where type == "llm"`
- Selector predicates (type, depth, tag, name)
- Expansion-by-composition (not expansion-by-template)

---

## Execution Timeline

```
Stage 1 ──────────── Stage 2 ──────── Stage 3
 (4 formulas)          (5 formulas)     (6 formulas)
 Week 1                Week 2           Week 3
 Parser MVP            Map combinator   Script cells
 Oracle runtime        Prompt frags     Shell execution
 Basic execution       Parallel exec    squash>

Stage 4 ────────────── Stage 5 ──────── Stage 6
 (14 formulas)           (8 formulas)     (8 formulas)
 Week 4-5                Week 6-7         Week 8
 Conditional wires       Import/apply     Composition
 Sub-molecules           Recipes          AOP patterns
 Decision cells          Metacircular     Full parity
```

## Success Criteria

1. **Parse parity**: Every TOML formula has a `.cell` equivalent that parses
2. **Execution parity**: `cell pour X.cell` produces equivalent behavior to
   `gt formula run X`
3. **Regression suite**: All `.cell` examples become the parser's test corpus
4. **Performance**: Cell execution is no slower than TOML formula execution
5. **Readability**: Cell versions are shorter and clearer than TOML equivalents

## Risk Mitigation

- **Stage gates are hard**: Don't advance until the gate passes
- **Dual-run period**: Run TOML and Cell in parallel, compare outputs
- **Rollback**: TOML formulas stay as fallback until Stage 5 gate passes
- **Language changes**: Grammar modifications allowed in Stages 1-3, frozen at 4
