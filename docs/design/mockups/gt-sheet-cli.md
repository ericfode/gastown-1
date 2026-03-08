# gt sheet CLI — Agent-Facing Spreadsheet Interface

**Bead**: hq-utl
**Date**: 2026-03-08
**Audience**: AI agents (Claude Code) running as Gas Town polecats/crew

---

This document mocks up the `gt sheet` family of CLI commands from an **agent's
perspective**. Every command has `--json` output. The examples show realistic
terminal sessions with timing, ANSI color descriptions, and error cases.

The running example is `mol-algebraic-survey`, a 5-cell molecule that analyzes
a codebase's algebraic structure.

---

## Table of Contents

1. [gt sheet status](#1-gt-sheet-status)
2. [gt eval](#2-gt-eval)
3. [gt trace](#3-gt-trace)
4. [gt stale](#4-gt-stale)
5. [gt pin / gt unpin](#5-gt-pin--gt-unpin)
6. [gt map](#6-gt-map)
7. [gt diff](#7-gt-diff)
8. [gt sankey](#8-gt-sankey)
9. [gt sheet recompute](#9-gt-sheet-recompute)
10. [gt sheet snapshot](#10-gt-sheet-snapshot)
11. [Debugging Session](#debugging-session-trace--pin--recompute--verify)
12. [Map Session](#map-session-template--map--aggregate)
13. [Error Cases](#error-cases)

---

## 1. gt sheet status

Show all cells in a sheet with their states — a text-mode spreadsheet view.

### Human-readable output

```
$ gt sheet status mol-algebraic-survey
mol-algebraic-survey (5 cells, 2 stale, 1 computing)
Budget: 45,000 tok remaining | Quality floor: adequate
Last recompute: 12m ago | Snapshot: v7

  CELL               STATE       TOKENS  DEPTH  QUALITY   VER  NOTE
  extract-types      🟢 fresh      5,240  d:0    good      v4
  find-patterns      🟡 stale      8,102  d:1    adequate  v2   upstream changed
  synthesize         🟡 stale     12,003  d:2    good      v3   upstream changed
  write-report       🔵 computing   ~15k  d:3    good      —    ETA 8s
  decision           ⬜ empty       ~1k   d:4    draft     —

  Wires:
    extract-types → find-patterns, synthesize
    find-patterns → synthesize
    synthesize → write-report, decision

  Staleness cascade: extract-types (v4 changed) → find-patterns → synthesize
```

### JSON output

```
$ gt sheet status mol-algebraic-survey --json
{
  "sheet": "mol-algebraic-survey",
  "cell_count": 5,
  "summary": {
    "fresh": 1,
    "stale": 2,
    "computing": 1,
    "empty": 1
  },
  "budget": {
    "remaining_tokens": 45000,
    "used_tokens": 25345,
    "total_tokens": 70345
  },
  "quality_floor": "adequate",
  "last_recompute": "2026-03-08T17:04:22Z",
  "snapshot_version": "v7",
  "cells": [
    {
      "name": "extract-types",
      "state": "fresh",
      "tokens": 5240,
      "depth": 0,
      "quality": "good",
      "version": 4,
      "upstream": [],
      "downstream": ["find-patterns", "synthesize"],
      "note": null
    },
    {
      "name": "find-patterns",
      "state": "stale",
      "tokens": 8102,
      "depth": 1,
      "quality": "adequate",
      "version": 2,
      "upstream": ["extract-types"],
      "downstream": ["synthesize"],
      "note": "upstream changed",
      "stale_reason": {
        "type": "upstream_changed",
        "source": "extract-types",
        "source_version": 4,
        "last_saw_version": 3
      }
    },
    {
      "name": "synthesize",
      "state": "stale",
      "tokens": 12003,
      "depth": 2,
      "quality": "good",
      "version": 3,
      "upstream": ["extract-types", "find-patterns"],
      "downstream": ["write-report", "decision"],
      "note": "upstream changed",
      "stale_reason": {
        "type": "upstream_changed",
        "source": "find-patterns",
        "source_version": 2,
        "last_saw_version": 1
      }
    },
    {
      "name": "write-report",
      "state": "computing",
      "tokens": null,
      "tokens_estimate": 15000,
      "depth": 3,
      "quality": "good",
      "version": null,
      "upstream": ["synthesize"],
      "downstream": [],
      "note": "ETA 8s",
      "compute": {
        "started_at": "2026-03-08T17:16:10Z",
        "model": "claude-sonnet-4-6",
        "eta_seconds": 8
      }
    },
    {
      "name": "decision",
      "state": "empty",
      "tokens": null,
      "tokens_estimate": 1000,
      "depth": 4,
      "quality": "draft",
      "version": null,
      "upstream": ["synthesize"],
      "downstream": [],
      "note": null
    }
  ],
  "wires": [
    {"from": "extract-types", "to": "find-patterns"},
    {"from": "extract-types", "to": "synthesize"},
    {"from": "find-patterns", "to": "synthesize"},
    {"from": "synthesize", "to": "write-report"},
    {"from": "synthesize", "to": "decision"}
  ]
}
```

---

## 2. gt eval

Evaluate a cell: fill its prompt template with upstream values, dispatch to LLM.

### Dry run (show filled prompt without executing)

```
$ gt eval find-patterns --dry-run
gt eval: dry run for find-patterns (depth:1)

Template: prompts/find-patterns.md
Model: claude-sonnet-4-6
Quality: adequate
Estimated tokens: ~8,000

── Filled Prompt (2,847 tokens) ──────────────────────────────────

You are analyzing a codebase for algebraic patterns.

## Extracted Types (from extract-types v4)

The following types were found in the codebase:
- Monoid: `WorkQueue`, `TokenBudget`, `MergeStrategy`
- Functor: `Pipeline[T]`, `Cell[T]`, `Wire[T]`
- Monad: `Effect[T]`, `Result[T]`
- Semiring: `Cost`, `Priority`

## Task

Identify structural patterns across these types:
1. Which types compose naturally?
2. Where are the homomorphisms?
3. What universal properties emerge?

Output a structured analysis, max 2000 tokens.

── End Prompt ────────────────────────────────────────────────────

Would consume ~8,000 tokens from budget (45,000 remaining).
Run without --dry-run to execute.
```

### Dry run JSON

```
$ gt eval find-patterns --dry-run --json
{
  "cell": "find-patterns",
  "action": "dry_run",
  "template": "prompts/find-patterns.md",
  "model": "claude-sonnet-4-6",
  "quality": "adequate",
  "filled_prompt": "You are analyzing a codebase for algebraic patterns.\n\n## Extracted Types (from extract-types v4)\n\nThe following types were found in the codebase:\n- Monoid: `WorkQueue`, `TokenBudget`, `MergeStrategy`\n- Functor: `Pipeline[T]`, `Cell[T]`, `Wire[T]`\n- Monad: `Effect[T]`, `Result[T]`\n- Semiring: `Cost`, `Priority`\n\n## Task\n\nIdentify structural patterns across these types:\n1. Which types compose naturally?\n2. Where are the homomorphisms?\n3. What universal properties emerge?\n\nOutput a structured analysis, max 2000 tokens.",
  "prompt_tokens": 2847,
  "estimated_output_tokens": 8000,
  "budget_remaining": 45000,
  "inputs": [
    {
      "ref": "extract-types",
      "version": 4,
      "state": "fresh",
      "tokens_injected": 1203
    }
  ]
}
```

### Execute evaluation

```
$ gt eval find-patterns
gt eval: find-patterns (depth:1)
  Model: claude-sonnet-4-6 | Quality: adequate
  Inputs: extract-types v4 (fresh, 1,203 tok)
  Dispatching... ████████████████████████████████ done (6.2s)

  Output: 1,847 tokens (v3)
  Quality: adequate ✓ (meets floor)
  Budget: 45,000 → 36,898 remaining

  Downstream affected:
    synthesize  — now stale (will see find-patterns v3, had v2)
```

### Execute JSON

```
$ gt eval find-patterns --json
{
  "cell": "find-patterns",
  "action": "eval",
  "status": "success",
  "model": "claude-sonnet-4-6",
  "duration_seconds": 6.2,
  "input_tokens": 2847,
  "output_tokens": 1847,
  "total_tokens": 4694,
  "version": 3,
  "quality": "adequate",
  "quality_meets_floor": true,
  "budget_before": 45000,
  "budget_after": 36898,
  "inputs_consumed": [
    {"ref": "extract-types", "version": 4, "tokens": 1203}
  ],
  "downstream_affected": [
    {
      "cell": "synthesize",
      "new_state": "stale",
      "reason": "upstream find-patterns changed from v2 to v3"
    }
  ],
  "value_preview": "## Pattern Analysis\n\n### 1. Natural Composition\n\nThe Monoid types (`WorkQueue`, `TokenBudget`, `MergeStrategy`) compose via their binary operation..."
}
```

---

## 3. gt trace

Show provenance chain: what inputs did this cell see, how compressed.

### Human-readable

```
$ gt trace synthesize
gt trace: synthesize (depth:2, v3)

Provenance Chain:
  synthesize v3 (12,003 tok)
  ├─ extract-types v3 (saw v3, current v4 ← STALE INPUT)
  │  └─ [source] codebase scan of src/**/*.go
  │     Raw: 142,000 tok → Compressed to 5,240 tok (ratio 27:1)
  └─ find-patterns v2 (saw v2, current v3 ← STALE INPUT)
     └─ extract-types v3 (was fresh at eval time)
        └─ [source] codebase scan of src/**/*.go

Compression Summary:
  Source → synthesize: 142,000 tok → 12,003 tok (11.8:1)
  Information path length: 3 hops
  Staleness: 2 inputs stale (extract-types, find-patterns)

Warning: synthesize v3 was computed with stale inputs.
  extract-types has advanced v3 → v4 since synthesize last ran.
  find-patterns has advanced v2 → v3 since synthesize last ran.
  Recommend: gt eval synthesize (or gt sheet recompute)
```

### JSON

```
$ gt trace synthesize --json
{
  "cell": "synthesize",
  "version": 3,
  "depth": 2,
  "tokens": 12003,
  "provenance": [
    {
      "input": "extract-types",
      "version_seen": 3,
      "version_current": 4,
      "is_stale": true,
      "tokens_at_eval": 4980,
      "source": {
        "type": "codebase_scan",
        "glob": "src/**/*.go",
        "raw_tokens": 142000,
        "compressed_tokens": 5240,
        "compression_ratio": 27.1
      }
    },
    {
      "input": "find-patterns",
      "version_seen": 2,
      "version_current": 3,
      "is_stale": true,
      "tokens_at_eval": 7200,
      "provenance": [
        {
          "input": "extract-types",
          "version_seen": 3,
          "version_current": 4,
          "is_stale": true,
          "tokens_at_eval": 4980
        }
      ]
    }
  ],
  "compression_summary": {
    "source_tokens": 142000,
    "output_tokens": 12003,
    "overall_ratio": 11.83,
    "path_length": 3,
    "stale_inputs": 2
  },
  "warnings": [
    "synthesize v3 was computed with stale inputs",
    "extract-types advanced v3 → v4 since last eval",
    "find-patterns advanced v2 → v3 since last eval"
  ]
}
```

---

## 4. gt stale

Show all stale cells with WHY they're stale and recommended action.

### Human-readable

```
$ gt stale mol-algebraic-survey
mol-algebraic-survey: 2 stale cells

  CELL            STALE SINCE  REASON                          ACTION
  find-patterns   12m ago      extract-types v3→v4             gt eval find-patterns
  synthesize      12m ago      extract-types v3→v4 (transitive) gt eval synthesize
                               find-patterns v1→v2 (direct)

  Recompute order (topological):
    1. find-patterns  (~8k tok, depth:1)
    2. synthesize     (~12k tok, depth:2)

  Last run cost: ~20,000 tokens (from prior digest)
  Budget remaining: 36,898 tokens  ✓ cap not exceeded

  Tip: gt sheet recompute mol-algebraic-survey --policy budgeted
```

### JSON

```
$ gt stale mol-algebraic-survey --json
{
  "sheet": "mol-algebraic-survey",
  "stale_count": 2,
  "cells": [
    {
      "name": "find-patterns",
      "stale_since": "2026-03-08T17:04:22Z",
      "stale_duration_seconds": 720,
      "reasons": [
        {
          "type": "upstream_changed",
          "upstream": "extract-types",
          "version_seen": 3,
          "version_current": 4,
          "is_transitive": false
        }
      ],
      "recommended_action": "gt eval find-patterns",
      "estimated_tokens": 8000,
      "depth": 1
    },
    {
      "name": "synthesize",
      "stale_since": "2026-03-08T17:04:22Z",
      "stale_duration_seconds": 720,
      "reasons": [
        {
          "type": "upstream_changed",
          "upstream": "extract-types",
          "version_seen": 3,
          "version_current": 4,
          "is_transitive": true
        },
        {
          "type": "upstream_changed",
          "upstream": "find-patterns",
          "version_seen": 1,
          "version_current": 2,
          "is_transitive": false
        }
      ],
      "recommended_action": "gt eval synthesize",
      "estimated_tokens": 12000,
      "depth": 2
    }
  ],
  "recompute_order": ["find-patterns", "synthesize"],
  "estimated_total_tokens": 20000,
  "budget_remaining": 36898,
  "budget_sufficient": true
}
```

---

## 5. gt pin / gt unpin

Pin a cell value for debugging. Downstream cells see the pinned value instead
of the computed one.

### Pin a value

```
$ gt pin extract-types '{"types": ["Monoid: WorkQueue", "Functor: Pipeline[T]"]}'
gt pin: extract-types pinned to manual value
  Pinned value: 47 tokens (was 5,240 tokens at v4)
  Version: v4-pinned

  Downstream effects:
    find-patterns  — now stale (upstream pinned, value changed)
    synthesize     — now stale (upstream pinned, value changed)

  ⚠ Pin active. Computed value preserved as v4-shadow.
  Run gt unpin extract-types to restore.
```

### Pin JSON

```
$ gt pin extract-types '{"types": ["Monoid: WorkQueue", "Functor: Pipeline[T]"]}' --json
{
  "cell": "extract-types",
  "action": "pin",
  "status": "success",
  "pinned_value": "{\"types\": [\"Monoid: WorkQueue\", \"Functor: Pipeline[T]\"]}",
  "pinned_tokens": 47,
  "previous_tokens": 5240,
  "previous_version": 4,
  "pinned_version": "v4-pinned",
  "shadow_version": "v4-shadow",
  "downstream_affected": [
    {"cell": "find-patterns", "new_state": "stale", "reason": "upstream pinned"},
    {"cell": "synthesize", "new_state": "stale", "reason": "upstream pinned"}
  ]
}
```

### Unpin

```
$ gt unpin extract-types
gt unpin: extract-types restored to computed value (v4)
  Restored: 5,240 tokens
  Shadow v4-shadow removed.

  Downstream effects:
    find-patterns  — now stale (upstream restored from pin)
    synthesize     — now stale (upstream restored from pin)
```

### Unpin JSON

```
$ gt unpin extract-types --json
{
  "cell": "extract-types",
  "action": "unpin",
  "status": "success",
  "restored_version": 4,
  "restored_tokens": 5240,
  "downstream_affected": [
    {"cell": "find-patterns", "new_state": "stale", "reason": "upstream restored from pin"},
    {"cell": "synthesize", "new_state": "stale", "reason": "upstream restored from pin"}
  ]
}
```

---

## 6. gt map

Apply a template across a parameter list. Instantiates one sheet per parameter set.

### Map a template over repositories

```
$ cat repos.jsonl
{"repo": "gastown", "glob": "**/*.go", "focus": "agent coordination"}
{"repo": "beads", "glob": "**/*.go", "focus": "issue tracking"}
{"repo": "longeye", "glob": "**/*.rs", "focus": "monitoring"}

$ gt map mol-algebraic-survey --over repos.jsonl
gt map: instantiating mol-algebraic-survey × 3 repos

  Creating sheets:
    mol-algebraic-survey@gastown    5 cells  budget: 70,000 tok
    mol-algebraic-survey@beads      5 cells  budget: 70,000 tok
    mol-algebraic-survey@longeye    5 cells  budget: 70,000 tok

  Total: 15 cells across 3 sheets
  Combined budget: 210,000 tokens

  Parameter bindings:
    @gastown:  repo=gastown,  glob=**/*.go, focus="agent coordination"
    @beads:    repo=beads,    glob=**/*.go, focus="issue tracking"
    @longeye:  repo=longeye,  glob=**/*.rs, focus="monitoring"

  Sheets created. Run gt sheet recompute --all to evaluate.
```

### Map JSON

```
$ gt map mol-algebraic-survey --over repos.jsonl --json
{
  "template": "mol-algebraic-survey",
  "params_file": "repos.jsonl",
  "param_count": 3,
  "sheets_created": [
    {
      "name": "mol-algebraic-survey@gastown",
      "cells": 5,
      "budget_tokens": 70000,
      "params": {"repo": "gastown", "glob": "**/*.go", "focus": "agent coordination"}
    },
    {
      "name": "mol-algebraic-survey@beads",
      "cells": 5,
      "budget_tokens": 70000,
      "params": {"repo": "beads", "glob": "**/*.go", "focus": "issue tracking"}
    },
    {
      "name": "mol-algebraic-survey@longeye",
      "cells": 5,
      "budget_tokens": 70000,
      "params": {"repo": "longeye", "glob": "**/*.rs", "focus": "monitoring"}
    }
  ],
  "total_cells": 15,
  "total_budget_tokens": 210000
}
```

---

## 7. gt diff

Diff two versions of a cell output. Shows semantic changes.

### Human-readable

```
$ gt diff synthesize v2 v3
gt diff: synthesize v2 → v3

  Version  Tokens  Quality   Computed
  v2       11,204  good      2026-03-08T16:30:00Z
  v3       12,003  good      2026-03-08T17:04:22Z

  Semantic diff:
  ┌─────────────────────────────────────────────────────────────┐
  │ ADDED in v3:                                                │
  │  + Semiring structure identified in Cost × Priority         │
  │  + Homomorphism: Pipeline[T] → Effect[T] via .run()        │
  │  + Universal property: WorkQueue is free monoid on Task     │
  │                                                             │
  │ CHANGED in v3:                                              │
  │  ~ Functor count: 2 → 3 (added Wire[T])                    │
  │  ~ Composition analysis expanded with naturality proof      │
  │                                                             │
  │ REMOVED in v3:                                              │
  │  - Tentative group structure on MergeStrategy (disproved)   │
  └─────────────────────────────────────────────────────────────┘

  Token delta: +799 (+7.1%)
  Structural similarity: 0.82 (moderate change)
```

### JSON

```
$ gt diff synthesize v2 v3 --json
{
  "cell": "synthesize",
  "version_a": 2,
  "version_b": 3,
  "version_a_tokens": 11204,
  "version_b_tokens": 12003,
  "token_delta": 799,
  "token_delta_pct": 7.1,
  "structural_similarity": 0.82,
  "version_a_computed": "2026-03-08T16:30:00Z",
  "version_b_computed": "2026-03-08T17:04:22Z",
  "changes": {
    "added": [
      "Semiring structure identified in Cost × Priority",
      "Homomorphism: Pipeline[T] → Effect[T] via .run()",
      "Universal property: WorkQueue is free monoid on Task"
    ],
    "changed": [
      {"field": "functor_count", "from": 2, "to": 3, "detail": "added Wire[T]"},
      {"field": "composition_analysis", "detail": "expanded with naturality proof"}
    ],
    "removed": [
      "Tentative group structure on MergeStrategy (disproved)"
    ]
  }
}
```

### Diff latest vs previous (shorthand)

```
$ gt diff synthesize
gt diff: synthesize v2 → v3 (latest vs previous)
  ...same output...
```

---

## 8. gt sankey

ASCII-art Sankey diagram showing information flow and compression ratios.

### Human-readable

```
$ gt sankey mol-algebraic-survey
mol-algebraic-survey — Information Flow

  [source: codebase]     142,000 tok
        │
        ▼
  ┌─────────────────┐
  │  extract-types   │  142k → 5.2k  (27:1 compression)
  │  🟢 v4  good     │
  └────────┬────────┘
           │ 5,240 tok
      ┌────┴──────────────┐
      │                   │
      ▼                   ▼
  ┌──────────────┐   (direct wire)
  │ find-patterns │        │
  │ 🟡 v2 adeq.  │        │
  └──────┬───────┘        │
         │ 8,102 tok      │
         │    ┌───────────┘
         ▼    ▼
  ┌─────────────────┐
  │   synthesize     │  13.3k → 12k  (1.1:1 — aggregation)
  │   🟡 v3  good    │
  └────────┬────────┘
           │ 12,003 tok
      ┌────┴────┐
      ▼         ▼
  ┌────────┐ ┌──────────┐
  │ write- │ │ decision │
  │ report │ │ ⬜ empty  │
  │ 🔵 ... │ │ ~1k est  │
  └────────┘ └──────────┘

  End-to-end: 142,000 → ~28,000 tok (5:1 overall compression)
  Active compute: 3 cells | Waiting: 2 cells
```

### JSON

```
$ gt sankey mol-algebraic-survey --json
{
  "sheet": "mol-algebraic-survey",
  "source_tokens": 142000,
  "total_output_tokens": 27345,
  "overall_compression_ratio": 5.19,
  "nodes": [
    {
      "name": "source:codebase",
      "type": "source",
      "tokens": 142000
    },
    {
      "name": "extract-types",
      "type": "cell",
      "input_tokens": 142000,
      "output_tokens": 5240,
      "compression_ratio": 27.1,
      "state": "fresh",
      "version": 4
    },
    {
      "name": "find-patterns",
      "type": "cell",
      "input_tokens": 5240,
      "output_tokens": 8102,
      "compression_ratio": 0.65,
      "state": "stale",
      "version": 2
    },
    {
      "name": "synthesize",
      "type": "cell",
      "input_tokens": 13342,
      "output_tokens": 12003,
      "compression_ratio": 1.11,
      "state": "stale",
      "version": 3
    },
    {
      "name": "write-report",
      "type": "cell",
      "input_tokens": 12003,
      "output_tokens": null,
      "state": "computing"
    },
    {
      "name": "decision",
      "type": "cell",
      "input_tokens": 12003,
      "output_tokens": null,
      "output_tokens_estimate": 1000,
      "state": "empty"
    }
  ],
  "edges": [
    {"from": "source:codebase", "to": "extract-types", "tokens": 142000},
    {"from": "extract-types", "to": "find-patterns", "tokens": 5240},
    {"from": "extract-types", "to": "synthesize", "tokens": 5240},
    {"from": "find-patterns", "to": "synthesize", "tokens": 8102},
    {"from": "synthesize", "to": "write-report", "tokens": 12003},
    {"from": "synthesize", "to": "decision", "tokens": 12003}
  ]
}
```

---

## 9. gt sheet recompute

Recompute all stale cells according to a recomputation policy.

### Eager policy (recompute everything)

```
$ gt sheet recompute mol-algebraic-survey --policy eager
gt sheet recompute: mol-algebraic-survey (policy: eager)

  Recompute plan (topological order):
    1. find-patterns  (stale, depth:1)  est ~8k tok
    2. synthesize     (stale, depth:2)  est ~12k tok
  Skip: extract-types (fresh), write-report (computing), decision (blocked)

  Executing...
    [1/2] find-patterns ████████████████████████████████ done (6.2s, 8,340 tok → v3)
    [2/2] synthesize    ████████████████████████████████ done (9.1s, 11,720 tok → v4)

  Recompute complete.
    Cells refreshed: 2
    Tokens consumed: 20,060
    Budget: 36,898 → 16,838 remaining
    Duration: 15.3s

    New downstream state:
      write-report — stale (synthesize changed v3→v4)
      decision     — stale (synthesize changed v3→v4)
```

### Budgeted policy

```
$ gt sheet recompute mol-algebraic-survey --policy budgeted --budget 10000
gt sheet recompute: mol-algebraic-survey (policy: budgeted, cap: 10,000 tok)

  Recompute plan (budget-constrained):
    1. find-patterns  (stale, depth:1)  est ~8k tok  ✓ within budget
    2. synthesize     (stale, depth:2)  est ~12k tok ✗ exceeds remaining budget

  Executing...
    [1/1] find-patterns ████████████████████████████████ done (6.2s, 8,340 tok → v3)

  Recompute complete (budget exhausted).
    Cells refreshed: 1 of 2 stale
    Tokens consumed: 8,340
    Budget cap: 10,000 → 1,660 remaining in cap
    Skipped: synthesize (would exceed budget)

  ⚠ 1 cell still stale. Increase budget or run gt eval synthesize.
```

### Convergent policy

```
$ gt sheet recompute mol-algebraic-survey --policy convergent
gt sheet recompute: mol-algebraic-survey (policy: convergent)

  Convergent recompute: iterate until no cell changes by more than threshold.
  Threshold: structural_similarity > 0.95
  Max iterations: 3

  Iteration 1:
    find-patterns ████████ done (8,340 tok → v3, similarity to v2: 0.78)
    synthesize    ████████ done (11,720 tok → v4, similarity to v3: 0.71)

  Iteration 2:
    find-patterns ████████ done (8,102 tok → v4, similarity to v3: 0.94)
    synthesize    ████████ done (12,003 tok → v5, similarity to v4: 0.89)

  Iteration 3:
    find-patterns ████████ done (8,050 tok → v5, similarity to v4: 0.98) ✓ converged
    synthesize    ████████ done (11,980 tok → v6, similarity to v5: 0.97) ✓ converged

  Converged after 3 iterations.
    Total tokens consumed: 60,195
    All cells above similarity threshold.
```

### Recompute JSON

```
$ gt sheet recompute mol-algebraic-survey --policy eager --json
{
  "sheet": "mol-algebraic-survey",
  "policy": "eager",
  "plan": [
    {"cell": "find-patterns", "reason": "stale", "estimated_tokens": 8000},
    {"cell": "synthesize", "reason": "stale", "estimated_tokens": 12000}
  ],
  "results": [
    {
      "cell": "find-patterns",
      "status": "success",
      "duration_seconds": 6.2,
      "tokens": 8340,
      "new_version": 3,
      "quality": "adequate"
    },
    {
      "cell": "synthesize",
      "status": "success",
      "duration_seconds": 9.1,
      "tokens": 11720,
      "new_version": 4,
      "quality": "good"
    }
  ],
  "summary": {
    "cells_refreshed": 2,
    "tokens_consumed": 20060,
    "budget_before": 36898,
    "budget_after": 16838,
    "duration_seconds": 15.3,
    "downstream_newly_stale": ["write-report", "decision"]
  }
}
```

---

## 10. gt sheet snapshot

Capture a full snapshot of the sheet state for later replay or comparison.

### Human-readable

```
$ gt sheet snapshot mol-algebraic-survey
gt sheet snapshot: mol-algebraic-survey

  Snapshot saved: snap-mol-algebraic-survey-v8
  Timestamp: 2026-03-08T17:20:15Z

  Contents:
    5 cells captured (values, versions, metadata)
    5 wires captured
    Budget state: 16,838 tok remaining
    Quality floor: adequate

  Cell versions at snapshot:
    extract-types   v4   (5,240 tok)
    find-patterns   v3   (8,340 tok)
    synthesize      v4   (11,720 tok)
    write-report    v1   (14,200 tok)
    decision        ⬜   (empty)

  Replay: gt sheet restore snap-mol-algebraic-survey-v8
  Compare: gt sheet snapshot --diff snap-v7 snap-v8
```

### JSON

```
$ gt sheet snapshot mol-algebraic-survey --json
{
  "sheet": "mol-algebraic-survey",
  "snapshot_id": "snap-mol-algebraic-survey-v8",
  "timestamp": "2026-03-08T17:20:15Z",
  "cells": [
    {
      "name": "extract-types",
      "version": 4,
      "state": "fresh",
      "tokens": 5240,
      "quality": "good",
      "value_hash": "a3f2c1d8e5b7...",
      "computed_at": "2026-03-08T16:45:00Z"
    },
    {
      "name": "find-patterns",
      "version": 3,
      "state": "fresh",
      "tokens": 8340,
      "quality": "adequate",
      "value_hash": "b7d4e2f1a9c3...",
      "computed_at": "2026-03-08T17:16:22Z"
    },
    {
      "name": "synthesize",
      "version": 4,
      "state": "fresh",
      "tokens": 11720,
      "quality": "good",
      "value_hash": "c1a5d3b8f2e7...",
      "computed_at": "2026-03-08T17:16:31Z"
    },
    {
      "name": "write-report",
      "version": 1,
      "state": "fresh",
      "tokens": 14200,
      "quality": "good",
      "value_hash": "d8f2e1c3a5b9...",
      "computed_at": "2026-03-08T17:18:00Z"
    },
    {
      "name": "decision",
      "version": null,
      "state": "empty",
      "tokens": null,
      "quality": null,
      "value_hash": null,
      "computed_at": null
    }
  ],
  "wires": [
    {"from": "extract-types", "to": "find-patterns"},
    {"from": "extract-types", "to": "synthesize"},
    {"from": "find-patterns", "to": "synthesize"},
    {"from": "synthesize", "to": "write-report"},
    {"from": "synthesize", "to": "decision"}
  ],
  "budget": {
    "remaining_tokens": 16838,
    "used_tokens": 53507,
    "total_tokens": 70345
  },
  "quality_floor": "adequate"
}
```

---

## Debugging Session: trace → pin → recompute → verify

A complete session showing how an agent debugs a wrong output.

```
# Agent notices the synthesize output looks wrong — it claims there are no
# monoid structures, but extract-types clearly found some.

$ gt sheet status mol-algebraic-survey --json | jq '.cells[] | select(.name=="synthesize")'
{
  "name": "synthesize",
  "state": "fresh",
  "tokens": 12003,
  "depth": 2,
  "quality": "good",
  "version": 3
}

# Step 1: Trace the provenance — what did synthesize actually see?

$ gt trace synthesize --json | jq '.provenance'
[
  {
    "input": "extract-types",
    "version_seen": 3,
    "version_current": 4,
    "is_stale": true,
    "tokens_at_eval": 4980
  },
  {
    "input": "find-patterns",
    "version_seen": 2,
    "version_current": 3,
    "is_stale": true,
    "tokens_at_eval": 7200
  }
]

# Aha — synthesize v3 was computed against extract-types v3, but we're now on v4.
# The monoid structures were only found in v4. Let's verify by diffing.

$ gt diff extract-types v3 v4 --json | jq '.changes.added'
[
  "Monoid: WorkQueue (binary op: enqueue-merge)",
  "Monoid: TokenBudget (binary op: budget-sum)",
  "Monoid: MergeStrategy (binary op: strategy-compose)"
]

# Confirmed: v3 didn't have the monoids. synthesize was working with old data.
# Before recomputing the whole chain, let's pin extract-types to test our theory.

$ gt pin extract-types '{"types": ["Monoid: WorkQueue", "Monoid: TokenBudget"]}' --json | jq '.status'
"success"

# Now recompute just synthesize to see if the monoid detection works.

$ gt eval synthesize --json | jq '{version, quality, value_preview}'
{
  "version": 4,
  "quality": "good",
  "value_preview": "## Algebraic Structure Analysis\n\n### Monoid Structures\n\nTwo monoid structures identified:\n- WorkQueue: free monoid on Task with enqueue-merge..."
}

# It works with pinned data. Now unpin and do a proper recompute.

$ gt unpin extract-types --json | jq '.status'
"success"

$ gt sheet recompute mol-algebraic-survey --policy eager --json | jq '.summary'
{
  "cells_refreshed": 2,
  "tokens_consumed": 20060,
  "budget_before": 36898,
  "budget_after": 16838,
  "duration_seconds": 15.3,
  "downstream_newly_stale": ["write-report", "decision"]
}

# Verify the fix propagated.

$ gt trace synthesize --json | jq '[.provenance[] | {input, is_stale}]'
[
  {"input": "extract-types", "is_stale": false},
  {"input": "find-patterns", "is_stale": false}
]

# All inputs fresh. Synthesize now correctly identifies monoid structures.
```

---

## Map Session: template → map → aggregate

A complete session showing how to apply an analysis template across multiple repos.

```
# Step 1: Check the template sheet exists and understand its structure.

$ gt sheet status mol-algebraic-survey --json | jq '{cells: [.cells[].name], wires: .wires}'
{
  "cells": ["extract-types", "find-patterns", "synthesize", "write-report", "decision"],
  "wires": [
    {"from": "extract-types", "to": "find-patterns"},
    {"from": "extract-types", "to": "synthesize"},
    {"from": "find-patterns", "to": "synthesize"},
    {"from": "synthesize", "to": "write-report"},
    {"from": "synthesize", "to": "decision"}
  ]
}

# Step 2: Create the parameter file for the repos we want to analyze.

$ cat > /tmp/repos.jsonl << 'EOF'
{"repo": "gastown", "glob": "**/*.go", "focus": "agent coordination patterns"}
{"repo": "beads", "glob": "**/*.go", "focus": "issue tracking data structures"}
{"repo": "longeye", "glob": "**/*.rs", "focus": "monitoring and observability"}
EOF

# Step 3: Map the template.

$ gt map mol-algebraic-survey --over /tmp/repos.jsonl --json
{
  "template": "mol-algebraic-survey",
  "params_file": "/tmp/repos.jsonl",
  "param_count": 3,
  "sheets_created": [
    {"name": "mol-algebraic-survey@gastown", "cells": 5, "budget_tokens": 70000},
    {"name": "mol-algebraic-survey@beads", "cells": 5, "budget_tokens": 70000},
    {"name": "mol-algebraic-survey@longeye", "cells": 5, "budget_tokens": 70000}
  ],
  "total_cells": 15,
  "total_budget_tokens": 210000
}

# Step 4: Recompute all sheets (depth-first within each sheet).

$ for sheet in gastown beads longeye; do
    echo "=== $sheet ==="
    gt sheet recompute "mol-algebraic-survey@$sheet" --policy budgeted --budget 50000 --json \
      | jq '.summary | {cells_refreshed, tokens_consumed, duration_seconds}'
  done
=== gastown ===
{
  "cells_refreshed": 5,
  "tokens_consumed": 42300,
  "duration_seconds": 38.2
}
=== beads ===
{
  "cells_refreshed": 5,
  "tokens_consumed": 38700,
  "duration_seconds": 31.4
}
=== longeye ===
{
  "cells_refreshed": 5,
  "tokens_consumed": 47100,
  "duration_seconds": 42.8
}

# Step 5: Compare synthesis results across repos.

$ for sheet in gastown beads longeye; do
    echo "--- $sheet ---"
    gt sheet status "mol-algebraic-survey@$sheet" --json \
      | jq '.cells[] | select(.name=="decision") | {state, quality, version}'
  done
--- gastown ---
{"state": "fresh", "quality": "good", "version": 1}
--- beads ---
{"state": "fresh", "quality": "adequate", "version": 1}
--- longeye ---
{"state": "fresh", "quality": "good", "version": 1}

# Step 6: Snapshot all sheets for the record.

$ for sheet in gastown beads longeye; do
    gt sheet snapshot "mol-algebraic-survey@$sheet" --json | jq '.snapshot_id'
  done
"snap-mol-algebraic-survey@gastown-v1"
"snap-mol-algebraic-survey@beads-v1"
"snap-mol-algebraic-survey@longeye-v1"
```

---

## Error Cases

### Evaluating a cell whose inputs aren't fresh

```
$ gt eval synthesize
Error: cannot eval synthesize — upstream inputs are stale.

  Stale inputs:
    find-patterns v2 (current: v3)

  Options:
    gt eval synthesize --force    Eval anyway with stale inputs (not recommended)
    gt eval find-patterns         Refresh the stale input first
    gt sheet recompute            Refresh all stale cells in topological order

  Use --force to override this safety check.
```

```
$ gt eval synthesize --json
{
  "cell": "synthesize",
  "action": "eval",
  "status": "error",
  "error": "upstream_stale",
  "message": "cannot eval synthesize — upstream inputs are stale",
  "stale_inputs": [
    {"cell": "find-patterns", "version_seen": 2, "version_current": 3}
  ],
  "suggestions": [
    "gt eval synthesize --force",
    "gt eval find-patterns",
    "gt sheet recompute"
  ]
}
```

### Budget exhausted

```
$ gt eval write-report
Error: insufficient budget for write-report.

  Last run cost: ~15,000 tokens (from prior digest)
  Budget remaining: 3,200 tokens

  Options:
    gt sheet status --json | jq '.budget'    Check budget details
    gt eval write-report --budget-override   Override budget cap (use with caution)
```

```
$ gt eval write-report --json
{
  "cell": "write-report",
  "action": "eval",
  "status": "error",
  "error": "budget_exhausted",
  "message": "insufficient budget for write-report",
  "estimated_tokens": 15000,
  "budget_remaining": 3200
}
```

### Cell not found

```
$ gt eval nonexistent-cell
Error: cell 'nonexistent-cell' not found in any active sheet.

  Did you mean:
    extract-types    (mol-algebraic-survey)
    find-patterns    (mol-algebraic-survey)

  List all cells: gt sheet status
```

```
$ gt eval nonexistent-cell --json
{
  "cell": "nonexistent-cell",
  "action": "eval",
  "status": "error",
  "error": "cell_not_found",
  "message": "cell 'nonexistent-cell' not found in any active sheet",
  "suggestions": ["extract-types", "find-patterns"]
}
```

### Circular dependency detected

```
$ gt sheet recompute mol-bad-sheet --policy eager
Error: circular dependency detected in mol-bad-sheet.

  Cycle: analyze → summarize → validate → analyze

  This sheet cannot be recomputed. Fix the wiring:
    gt sheet edit mol-bad-sheet    Open sheet definition
```

```
$ gt sheet recompute mol-bad-sheet --policy eager --json
{
  "sheet": "mol-bad-sheet",
  "policy": "eager",
  "status": "error",
  "error": "circular_dependency",
  "cycle": ["analyze", "summarize", "validate", "analyze"]
}
```

### Concurrent eval conflict

```
$ gt eval write-report
Error: write-report is already being computed.

  Started: 8s ago by gastown/polecats/nux
  Model: claude-sonnet-4-6
  ETA: ~7s remaining

  Options:
    gt eval write-report --wait    Wait for current computation to finish
    gt eval write-report --cancel  Cancel current computation and restart
```

```
$ gt eval write-report --json
{
  "cell": "write-report",
  "action": "eval",
  "status": "error",
  "error": "already_computing",
  "message": "write-report is already being computed",
  "compute": {
    "started_at": "2026-03-08T17:16:10Z",
    "started_by": "gastown/polecats/nux",
    "model": "claude-sonnet-4-6",
    "eta_seconds": 7
  }
}
```

### Quality below floor

```
$ gt eval find-patterns
gt eval: find-patterns (depth:1)
  Dispatching... ████████████████████████████████ done (5.8s)

  ⚠ Quality below floor.
    Output quality: draft
    Quality floor: adequate
    Version: v3 (marked draft)

  The output was saved but flagged. Options:
    gt eval find-patterns --quality excellent    Re-eval with higher quality model
    gt eval find-patterns --accept-draft         Accept draft quality
    gt sheet recompute --quality-override draft   Lower the floor
```

```
$ gt eval find-patterns --json
{
  "cell": "find-patterns",
  "action": "eval",
  "status": "warning",
  "warning": "quality_below_floor",
  "output_quality": "draft",
  "quality_floor": "adequate",
  "version": 3,
  "version_flagged": true,
  "tokens": 6200,
  "suggestions": [
    "gt eval find-patterns --quality excellent",
    "gt eval find-patterns --accept-draft"
  ]
}
```
