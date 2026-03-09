# Cell Formula Survey: TOML-to-Cell Translation Assessment

**Date**: 2026-03-08
**Scope**: All 44 Gas Town formulas across 4 survey batches
**Purpose**: Determine Cell language coverage of existing TOML formula corpus

---

## Executive Summary

- **Total formulas surveyed**: 44
- **DIRECT** (maps cleanly to Cell): 18 (41%)
- **EXTENDED** (needs planned Cell features like `map`, `preset`, `prompt@`): 22 (50%)
- **GAP** (no Cell equivalent exists): 4 (9%)

**Coverage today**: 41% expressible with current Cell syntax. With planned EXTENDED features (`map`, `preset`, `prompt@`, oracle gates), coverage rises to 91%. Four formulas (all in Batch 1) require composition paradigms Cell does not address: aspect-oriented programming and expansion templates.

---

## Formula Table

| # | Formula | Batch | Category | One-Line Note |
|---|---------|-------|----------|---------------|
| 1 | shiny | 1 | DIRECT | Linear 5-step workflow, canonical engineering chain |
| 2 | shiny-secure | 1 | GAP | AOP `advice.around` / aspect composition has no Cell equivalent |
| 3 | shiny-enterprise | 1 | GAP | `expand` (step explosion via template) has no Cell equivalent |
| 4 | security-audit | 1 | GAP | Aspect type: pointcuts, advice, dynamic `{step.id}` -- fundamentally incompatible |
| 5 | gastown-release | 1 | EXTENDED | 14-step release chain; shell-heavy, needs script cells |
| 6 | beads-release | 1 | EXTENDED | 17-step release; fan-in at verify, `pour` flag has no Cell equiv |
| 7 | rule-of-five | 1 | GAP | Expansion template with `{target}` interpolation -- meta-programming |
| 8 | towers-of-hanoi | 1 | DIRECT | 9-step linear chain, trivial deterministic steps |
| 9 | code-review | 2 | EXTENDED | Convoy; needs `map`, `preset`, `prompt@` |
| 10 | design | 2 | EXTENDED | Convoy; needs `map`, `prompt@` |
| 11 | mol-algebraic-survey | 2 | DIRECT | Linear sequential molecule |
| 12 | mol-survey-dispatch | 2 | EXTENDED | Orchestrator; needs `map` over subsystem list |
| 13 | mol-idea-to-plan | 2 | EXTENDED | Human gates need `accept>` + oracle-gated wires |
| 14 | mol-prd-review | 2 | EXTENDED | Convoy; needs `map`, `prompt@` |
| 15 | mol-plan-review | 2 | EXTENDED | Convoy; needs `map`, `prompt@` |
| 16 | mol-polecat-code-review | 2 | DIRECT | Linear sequential molecule |
| 17 | mol-gastown-boot | 3 | EXTENDED | Parallel containers need `map` or nested molecules |
| 18 | mol-town-shutdown | 3 | DIRECT | Linear 8-step pipeline of script cells |
| 19 | mol-shutdown-dance | 3 | EXTENDED | Conditional gates, state machine, OR-join semantics |
| 20 | mol-polecat-work | 3 | DIRECT | Linear 8-step worker lifecycle |
| 21 | mol-polecat-lease | 3 | DIRECT | Linear 5-step monitoring workflow |
| 22 | mol-polecat-review-pr | 3 | DIRECT | Linear 7-step PR review pipeline |
| 23 | mol-polecat-conflict-resolve | 3 | DIRECT | Linear 8-step merge conflict resolution |
| 24 | mol-dog-reaper | 3 | EXTENDED | Iteration over databases, conditional dry-run |
| 25 | mol-dog-backup | 3 | EXTENDED | Iteration over databases, rsync offsite |
| 26 | mol-dog-compactor | 3 | EXTENDED | ZFC-exempt; Go executor, Cell is observable skeleton |
| 27 | mol-dog-doctor | 3 | DIRECT | Linear 3-step probe-inspect-report |
| 28 | mol-dog-jsonl | 3 | EXTENDED | Iteration + spike detection, hybrid Go executor |
| 29 | mol-dog-phantom-db | 3 | DIRECT | Linear 3-step scan-quarantine-report |
| 30 | mol-dog-stale-db | 3 | EXTENDED | Conditional cleanup vs escalation branching |
| 31 | mol-boot-triage | 4 | DIRECT | Script-heavy 5-step sequential chain |
| 32 | mol-convoy-cleanup | 4 | DIRECT | Variables map to `input` params |
| 33 | mol-convoy-feed | 4 | EXTENDED | Iteration over ready issues needs `each>` |
| 34 | mol-deacon-patrol | 4 | EXTENDED | 26-step DAG with parallel branches and loop-back |
| 35 | mol-dep-propagate | 4 | EXTENDED | Cross-rig iteration over dependents |
| 36 | mol-digest-generate | 4 | EXTENDED | Multi-rig data collection needs `map` |
| 37 | mol-orphan-scan | 4 | EXTENDED | Parallel scans + triage logic |
| 38 | mol-refinery-patrol | 4 | EXTENDED | Conditional config, loop-back, merge protocol |
| 39 | mol-session-gc | 4 | DIRECT | Sequential script cells |
| 40 | mol-sync-workspace | 4 | DIRECT | Sequential script cells with config vars |
| 41 | mol-witness-patrol | 4 | EXTENDED | Complex survey logic, Task-tool parallelism |
| 42 | towers-of-hanoi-7 | 4 | DIRECT | 129 steps, pure linear chain |
| 43 | towers-of-hanoi-9 | 4 | DIRECT | 513 steps, pure linear chain |
| 44 | towers-of-hanoi-10 | 4 | DIRECT | 1025 steps, pure linear chain |

---

## Gap Analysis: Missing Cell Features

### Hard Gaps (4 formulas blocked)

| Missing Feature | Formulas Affected | Description |
|----------------|-------------------|-------------|
| Aspect-oriented composition | shiny-secure, security-audit | `advice.around`, pointcuts, `{step.id}` interpolation. AOP is a composition paradigm; Cell is a wiring paradigm. |
| Expansion templates | shiny-enterprise, rule-of-five | `{target}` interpolation generates dynamic step IDs. Meta-programming that replaces one step with a subgraph. |
| Formula inheritance | shiny-secure, shiny-enterprise | `extends: [base]` -- molecules cannot extend other molecules. |

### Soft Gaps (workaround exists, but inelegant)

| Missing Feature | Formulas Affected | Current Workaround |
|----------------|-------------------|-------------------|
| `map` construct | 10+ formulas (convoys, dogs, patrols) | Shell `for` loops inside script cells |
| `preset` (named configurations) | code-review | Hardcode aspect lists per molecule variant |
| `prompt@` (reusable prompt fragments) | code-review, design, prd-review, plan-review | Copy-paste prompt text into each cell |
| OR-join wire semantics | mol-shutdown-dance | Oracle gates that check "any predecessor completed" |
| Loop-back / iteration | deacon-patrol, refinery-patrol, witness-patrol | Unroll loop iterations into explicit cells |
| Default parameter values | towers-of-hanoi, others | No `input param.X : type = default` syntax |
| `pour` flag (auto-execute) | beads-release | Runtime annotation, not language concern |
| Template conditionals (`{{#if}}`) | dog-reaper, dog-stale-db | Shell `if`/`case` inside script cells |
| Computed runtime variables | dog-backup, dog-jsonl, dog-stale-db | Variables live inside shell, not Cell-level |
| Parallel container grouping | mol-gastown-boot | DAG topology achieves same result, loses grouping annotation |
| Cross-cell field projection (`{{scan.orphan_count}}`) | dog-stale-db, boot-triage | Requires structured output parsing at runtime |

---

## Coverage Assessment

| Category | Count | % | Meaning |
|----------|-------|---|---------|
| DIRECT | 18 | 41% | Current Cell syntax handles these |
| EXTENDED | 22 | 50% | Needs `map`, `preset`, `prompt@`, oracle gates, field projection |
| GAP | 4 | 9% | Requires AOP or expansion templates -- new paradigms |

**With EXTENDED features implemented**: 40/44 = **91% coverage**.

The 4 GAP formulas (shiny-secure, shiny-enterprise, security-audit, rule-of-five) can be manually inlined but lose their composability. In practice, these are rarely instantiated directly -- shiny-secure and shiny-enterprise are composed variants of shiny, and rule-of-five/security-audit are reusable templates applied by composition.

---

## Recommendations

### Priority 1: Unlock the 22 EXTENDED formulas

1. **`map` construct** -- Highest impact. Enables convoys (parallel legs), database iteration, multi-rig operations. At least 10 formulas need this.
   ```
   map # leg over aspects as aspect { ... }
   ```

2. **`prompt@` (reusable prompt fragments)** -- All 4 convoy formulas share base prompts. Without this, prompts are copy-pasted across cells.

3. **Oracle-gated wires (`-> ? oracle ->`)** -- Needed for conditional branching in shutdown-dance, patrol loops, and dog formulas with triage logic.

4. **Cross-cell field projection (`{{cell.field}}`)** -- Several formulas pass structured data between cells (boot-triage, stale-db). Requires runtime to parse JSON output from predecessor cells.

### Priority 2: Quality-of-life

5. **`preset` (named configurations)** -- Only code-review needs this today, but it's a clean pattern for parameterized molecule variants.

6. **Default parameter values** -- `input param.X : type = "default"`. Minor syntax addition, broad utility.

7. **`squash>` directive** -- Already sketched in survey translations. Formalizes molecule-level completion behavior.

### Priority 3: Close the 4 GAP formulas (optional)

8. **Recipe enhancement for expansion** -- Allow `!split` to preserve upstream/downstream wiring. Gets rule-of-five ~80% expressible.

9. **Pattern-matching in recipes** -- `recipe name(target matching "glob*")` enables aspect-like before/after injection without full AOP.

10. **Molecule inheritance** -- `## name extends base { ... }` for formula composition. Lower priority: manual inlining works, and inheritance adds complexity.

### Not recommended

- **Full AOP in Cell** -- The 2 aspect-blocked formulas (shiny-secure, security-audit) can be manually inlined. Adding pointcuts/advice to a wiring language would be a paradigm mismatch. Keep AOP in the TOML runtime if needed.

---

## Key Insight

The TOML formula corpus splits cleanly into two populations:

1. **Operational molecules** (dogs, polecats, patrols, releases) -- 36 formulas, overwhelmingly script-heavy linear chains. Cell handles these well today or with `map`.

2. **Composition formulas** (aspects, expansions, convoy orchestrators) -- 8 formulas using higher-order patterns. These need either EXTENDED features (`map`, `preset`) or represent genuine paradigm gaps (AOP).

Cell's strength is in population 1, which is 82% of the corpus and growing. The composition formulas are high-value but low-volume -- addressing them with targeted recipe enhancements is more pragmatic than adding new paradigms.
