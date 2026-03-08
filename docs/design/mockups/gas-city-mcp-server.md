# Gas City MCP Server

**Native tool interface for AI agents interacting with Gas City sheets.**

> Instead of shelling out to CLI commands and parsing text, agents get structured
> JSON tool calls with typed inputs and outputs. No shell escaping. No parsing
> ambiguity. Rich error types. Streaming updates.

---

## Server Configuration

### Claude Desktop (`claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "gas-city": {
      "command": "gt",
      "args": ["mcp", "serve"],
      "env": {
        "GAS_CITY_SHEETS_DIR": "/path/to/sheets",
        "GAS_CITY_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Claude Code (`.claude/settings.json`)

```json
{
  "mcpServers": {
    "gas-city": {
      "command": "gt",
      "args": ["mcp", "serve"],
      "env": {
        "GAS_CITY_SHEETS_DIR": "/path/to/sheets"
      }
    }
  }
}
```

### Programmatic (stdio transport)

```bash
gt mcp serve                    # stdio transport (default)
gt mcp serve --transport sse    # SSE transport on :8371
gt mcp serve --transport sse --port 9000
```

---

## Tools

### 1. `gas_city_sheet_status`

Get the current state of a sheet: all cells, wires, budget, and staleness.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": {
      "type": "string",
      "description": "Name of the sheet to inspect"
    },
    "includeValues": {
      "type": "boolean",
      "default": false,
      "description": "Include cell value previews (first 200 chars). Omit for cheaper overview."
    }
  },
  "required": ["sheetName"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cells": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "status": { "enum": ["fresh", "stale", "empty", "computing", "error"] },
          "tokens": { "type": "integer", "description": "Tokens used in last computation" },
          "depth": { "type": "integer", "description": "Compression depth (0 = leaf)" },
          "quality": { "enum": ["draft", "adequate", "refined", "verified"] },
          "version": { "type": "integer" },
          "valuePreview": { "type": "string", "description": "First 200 chars if includeValues=true" }
        }
      }
    },
    "wires": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "from": { "type": "string" },
          "to": { "type": "string" },
          "type": { "type": "string", "description": "Wire type annotation (e.g. TypeInventory, Summary)" }
        }
      }
    },
    "budget": {
      "type": "object",
      "properties": {
        "remaining": { "type": "integer" },
        "total": { "type": "integer" }
      }
    },
    "staleCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Names of cells in stale state"
    }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_sheet_status",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "includeValues": true
  }
}
```

**Example Response**

```json
{
  "sheetName": "mol-algebraic-survey",
  "cells": [
    {
      "name": "spatial-structure",
      "status": "fresh",
      "tokens": 12840,
      "depth": 0,
      "quality": "refined",
      "version": 3,
      "valuePreview": "# Spatial Structure Analysis\n\nGas Town organizes work through a two-level hierarchy: Town (cross-rig coordination) and Rig (project implementation)..."
    },
    {
      "name": "work-distribution",
      "status": "stale",
      "tokens": 18200,
      "depth": 0,
      "quality": "adequate",
      "version": 2,
      "valuePreview": "# Work Distribution Patterns\n\nThe dispatch system routes beads to polecats through a priority-weighted queue..."
    },
    {
      "name": "synthesis",
      "status": "stale",
      "tokens": 8500,
      "depth": 1,
      "quality": "draft",
      "version": 1,
      "valuePreview": "# Algebraic Survey Synthesis\n\nCombining structural, communication, and work-distribution analyses reveals..."
    }
  ],
  "wires": [
    { "from": "spatial-structure", "to": "synthesis", "type": "StructuralAnalysis" },
    { "from": "work-distribution", "to": "synthesis", "type": "DistributionAnalysis" },
    { "from": "communication", "to": "synthesis", "type": "CommAnalysis" }
  ],
  "budget": { "remaining": 145000, "total": 200000 },
  "staleCells": ["work-distribution", "synthesis"]
}
```

**Error Responses**

| Code | Condition | Body |
|------|-----------|------|
| `sheet_not_found` | Sheet name doesn't match any loaded sheet | `{ "error": "sheet_not_found", "sheetName": "bad-name", "available": ["mol-algebraic-survey", ...] }` |
| `storage_error` | Dolt query failed | `{ "error": "storage_error", "detail": "connection refused on port 3307" }` |

---

### 2. `gas_city_eval`

Evaluate (recompute) a single cell. Returns the filled prompt, the result, and
provenance metadata.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cellName": { "type": "string", "description": "Cell to evaluate" },
    "dryRun": {
      "type": "boolean",
      "default": false,
      "description": "If true, return the filled prompt without executing. Useful for cost estimation."
    }
  },
  "required": ["sheetName", "cellName"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "cellName": { "type": "string" },
    "filledPrompt": { "type": "string", "description": "The prompt after template substitution with input values" },
    "result": { "type": "string", "description": "LLM output. Null if dryRun=true." },
    "inputSnapshot": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "cell": { "type": "string" },
          "version": { "type": "integer" }
        }
      },
      "description": "Versions of all input cells consumed during this evaluation"
    },
    "newVersion": { "type": "integer", "description": "Version number after evaluation (unchanged if dryRun)" },
    "tokensUsed": { "type": "integer", "description": "Total tokens (input + output). 0 if dryRun." },
    "model": { "type": "string", "description": "Model used for evaluation" },
    "durationMs": { "type": "integer" }
  }
}
```

**Example Call — Dry Run**

```json
{
  "tool": "gas_city_eval",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "synthesis",
    "dryRun": true
  }
}
```

**Example Response — Dry Run**

```json
{
  "cellName": "synthesis",
  "filledPrompt": "You are analyzing the algebraic structure of Gas Town.\n\n## Spatial Structure\n\n<spatial-structure version=3>\nGas Town organizes work through a two-level hierarchy...\n</spatial-structure>\n\n## Work Distribution\n\n<work-distribution version=2>\nThe dispatch system routes beads to polecats...\n</work-distribution>\n\n## Communication\n\n<communication version=4>\nAgents communicate through a layered mail system...\n</communication>\n\nSynthesize these analyses into a unified algebraic model.",
  "result": null,
  "inputSnapshot": [
    { "cell": "spatial-structure", "version": 3 },
    { "cell": "work-distribution", "version": 2 },
    { "cell": "communication", "version": 4 }
  ],
  "newVersion": 1,
  "tokensUsed": 0,
  "model": null,
  "durationMs": 12
}
```

**Example Call — Full Evaluation**

```json
{
  "tool": "gas_city_eval",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "synthesis"
  }
}
```

**Example Response — Full Evaluation**

```json
{
  "cellName": "synthesis",
  "filledPrompt": "You are analyzing the algebraic structure of Gas Town...",
  "result": "# Algebraic Survey Synthesis\n\nGas Town exhibits a layered algebraic structure that maps cleanly to category-theoretic constructions...\n\n## Core Findings\n\n1. **Spatial Functor**: The Town→Rig→Polecat hierarchy forms a presheaf over a small category of organizational scopes...",
  "inputSnapshot": [
    { "cell": "spatial-structure", "version": 3 },
    { "cell": "work-distribution", "version": 2 },
    { "cell": "communication", "version": 4 }
  ],
  "newVersion": 2,
  "tokensUsed": 8540,
  "model": "claude-sonnet-4-6",
  "durationMs": 14200
}
```

**Error Responses**

| Code | Condition | Body |
|------|-----------|------|
| `cell_not_found` | Cell doesn't exist in sheet | `{ "error": "cell_not_found", "cellName": "bad", "available": [...] }` |
| `inputs_empty` | One or more required input cells have no value | `{ "error": "inputs_empty", "emptyCells": ["communication"] }` |
| `budget_exceeded` | Token spend has hit the budget cap | `{ "error": "budget_exceeded", "spent": 12000, "cap": 5000 }` |
| `eval_failed` | LLM call failed | `{ "error": "eval_failed", "detail": "rate_limited", "retryAfterMs": 30000 }` |

---

### 3. `gas_city_eval_stale`

Batch-recompute stale cells according to a policy. This is the "Ctrl+Shift+R"
operation — find everything that's stale and fix it.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "policy": {
      "enum": ["eager", "budgeted", "convergent"],
      "description": "eager: recompute all stale cells in topological order. budgeted: recompute until budget exhausted. convergent: iterate until no cells change (fixed-point)."
    },
    "budget": {
      "type": "integer",
      "description": "Token budget for this operation. Required for 'budgeted' policy. Optional cap for others."
    },
    "maxRounds": {
      "type": "integer",
      "default": 3,
      "description": "Maximum iteration rounds for 'convergent' policy."
    }
  },
  "required": ["sheetName", "policy"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "recomputed": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Cells that were successfully recomputed"
    },
    "skipped": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Stale cells that were NOT recomputed (budget, error, etc.)"
    },
    "rounds": { "type": "integer", "description": "Number of rounds executed (convergent policy)" },
    "converged": { "type": "boolean", "description": "True if no cells changed in the final round" },
    "budgetRemaining": { "type": "integer" },
    "totalTokensUsed": { "type": "integer" },
    "durationMs": { "type": "integer" }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_eval_stale",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "policy": "budgeted",
    "budget": 50000
  }
}
```

**Example Response**

```json
{
  "recomputed": ["work-distribution", "synthesis"],
  "skipped": [],
  "rounds": 1,
  "converged": true,
  "budgetRemaining": 23260,
  "totalTokensUsed": 26740,
  "durationMs": 31200
}
```

**Error Responses**

| Code | Condition | Body |
|------|-----------|------|
| `no_stale_cells` | Nothing to recompute | `{ "error": "no_stale_cells", "sheetName": "..." }` |
| `cycle_detected` | Sheet DAG has a cycle (convergent won't terminate) | `{ "error": "cycle_detected", "cycle": ["A", "B", "C", "A"] }` |

---

### 4. `gas_city_trace`

Trace the computation chain for a cell — how it was built, what compressed,
where the bottleneck is.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cellName": { "type": "string" }
  },
  "required": ["sheetName", "cellName"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "cellName": { "type": "string" },
    "chain": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "cell": { "type": "string" },
          "version": { "type": "integer" },
          "compressionPolicy": { "enum": ["none", "summary", "extract", "elide"] },
          "depth": { "type": "integer" },
          "tokensIn": { "type": "integer" },
          "tokensOut": { "type": "integer" }
        }
      },
      "description": "Ordered from leaves (depth 0) to the traced cell"
    },
    "totalCompression": {
      "type": "number",
      "description": "Ratio of raw input tokens to final output tokens across the chain"
    },
    "bottleneck": {
      "type": "string",
      "description": "Cell name with the highest token cost or longest latency"
    }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_trace",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "synthesis"
  }
}
```

**Example Response**

```json
{
  "cellName": "synthesis",
  "chain": [
    { "cell": "spatial-structure", "version": 3, "compressionPolicy": "summary", "depth": 0, "tokensIn": 38200, "tokensOut": 12840 },
    { "cell": "work-distribution", "version": 3, "compressionPolicy": "summary", "depth": 0, "tokensIn": 42100, "tokensOut": 18200 },
    { "cell": "communication", "version": 4, "compressionPolicy": "extract", "depth": 0, "tokensIn": 35600, "tokensOut": 9800 },
    { "cell": "synthesis", "version": 2, "compressionPolicy": "none", "depth": 1, "tokensIn": 40840, "tokensOut": 8540 }
  ],
  "totalCompression": 4.65,
  "bottleneck": "work-distribution"
}
```

---

### 5. `gas_city_pin`

Pin a cell to a fixed value, overriding its computed output. Downstream cells
see the pinned value. Useful for debugging ("what if this cell said X?") and
for injecting human-authored content.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cellName": { "type": "string" },
    "value": { "type": "string", "description": "The value to pin. Replaces the cell's computed output." }
  },
  "required": ["sheetName", "cellName", "value"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "pinned": { "type": "boolean", "const": true },
    "cellName": { "type": "string" },
    "previousVersion": { "type": "integer" },
    "affectedCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "All downstream cells (transitive)"
    },
    "newStaleCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Downstream cells that became stale due to this pin"
    }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_pin",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "spatial-structure",
    "value": "# Spatial Structure (Override)\n\nFor debugging: assume a flat hierarchy with no rig separation."
  }
}
```

**Example Response**

```json
{
  "pinned": true,
  "cellName": "spatial-structure",
  "previousVersion": 3,
  "affectedCells": ["synthesis"],
  "newStaleCells": ["synthesis"]
}
```

---

### 6. `gas_city_unpin`

Remove a pin from a cell, restoring it to its last computed value.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cellName": { "type": "string" }
  },
  "required": ["sheetName", "cellName"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "unpinned": { "type": "boolean", "const": true },
    "cellName": { "type": "string" },
    "restoredState": {
      "enum": ["fresh", "stale", "empty"],
      "description": "State of the cell after removing the pin"
    },
    "restoredVersion": { "type": "integer" },
    "affectedCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Downstream cells affected by the restoration"
    }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_unpin",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "spatial-structure"
  }
}
```

**Example Response**

```json
{
  "unpinned": true,
  "cellName": "spatial-structure",
  "restoredState": "fresh",
  "restoredVersion": 3,
  "affectedCells": ["synthesis"]
}
```

**Error Responses**

| Code | Condition | Body |
|------|-----------|------|
| `not_pinned` | Cell is not currently pinned | `{ "error": "not_pinned", "cellName": "spatial-structure" }` |

---

### 7. `gas_city_diff`

Compare two versions of a cell's output. Useful for understanding what changed
across recomputations.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "cellName": { "type": "string" },
    "versionA": { "type": "integer", "description": "Older version" },
    "versionB": { "type": "integer", "description": "Newer version" }
  },
  "required": ["sheetName", "cellName", "versionA", "versionB"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "cellName": { "type": "string" },
    "versionA": { "type": "integer" },
    "versionB": { "type": "integer" },
    "stable": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Sections/headings present in both versions with no semantic change"
    },
    "added": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Sections/content present only in versionB"
    },
    "removed": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Sections/content present only in versionA"
    },
    "changed": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "section": { "type": "string" },
          "old": { "type": "string" },
          "new": { "type": "string" }
        }
      },
      "description": "Sections present in both but with different content"
    }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_diff",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "cellName": "work-distribution",
    "versionA": 2,
    "versionB": 3
  }
}
```

**Example Response**

```json
{
  "cellName": "work-distribution",
  "versionA": 2,
  "versionB": 3,
  "stable": ["Priority Queue", "Budget Allocation"],
  "added": ["Convoy Batching"],
  "removed": [],
  "changed": [
    {
      "section": "Dispatch Algorithm",
      "old": "Round-robin dispatch across available polecats with priority weighting.",
      "new": "Weighted dispatch with affinity scoring — polecats who previously worked on related issues get priority. Affinity decays over 48h."
    }
  ]
}
```

**Error Responses**

| Code | Condition | Body |
|------|-----------|------|
| `version_not_found` | Requested version doesn't exist | `{ "error": "version_not_found", "cellName": "...", "version": 5, "maxVersion": 3 }` |

---

### 8. `gas_city_snapshot`

Capture a point-in-time snapshot of an entire sheet. Snapshots are immutable
references — useful for before/after comparisons around major changes.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "label": {
      "type": "string",
      "description": "Optional human-readable label for this snapshot"
    }
  },
  "required": ["sheetName"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "snapshotId": { "type": "string" },
    "label": { "type": "string" },
    "timestamp": { "type": "string", "format": "date-time" },
    "sheetName": { "type": "string" },
    "cells": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": { "type": "string" },
          "value": { "type": "string" },
          "version": { "type": "integer" },
          "status": { "enum": ["fresh", "stale", "empty", "pinned"] },
          "inputs": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "cell": { "type": "string" },
                "version": { "type": "integer" }
              }
            }
          }
        }
      }
    },
    "totalTokens": { "type": "integer", "description": "Sum of all cell token costs in snapshot" }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_snapshot",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "label": "before-refactor"
  }
}
```

**Example Response**

```json
{
  "snapshotId": "snap-a4f2c",
  "label": "before-refactor",
  "timestamp": "2026-03-08T14:22:00Z",
  "sheetName": "mol-algebraic-survey",
  "cells": [
    {
      "name": "spatial-structure",
      "value": "# Spatial Structure Analysis\n\nGas Town organizes work through...",
      "version": 3,
      "status": "fresh",
      "inputs": []
    },
    {
      "name": "synthesis",
      "value": "# Algebraic Survey Synthesis\n\nCombining structural...",
      "version": 2,
      "status": "fresh",
      "inputs": [
        { "cell": "spatial-structure", "version": 3 },
        { "cell": "work-distribution", "version": 3 },
        { "cell": "communication", "version": 4 }
      ]
    }
  ],
  "totalTokens": 67580
}
```

---

### 9. `gas_city_map`

Instantiate a formula template across multiple parameter sets. This is the
"drag-to-fill" operation — take one cell's formula and stamp it out across
a list of inputs.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "templateName": {
      "type": "string",
      "description": "Name of an existing cell whose prompt template will be reused"
    },
    "sheetName": {
      "type": "string",
      "description": "Sheet containing the template. Also where new cells are created."
    },
    "params": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "name": {
            "type": "string",
            "description": "Name for the new cell"
          },
          "values": {
            "type": "object",
            "additionalProperties": { "type": "string" },
            "description": "Template variable substitutions"
          }
        },
        "required": ["name", "values"]
      },
      "description": "One entry per instantiation"
    }
  },
  "required": ["templateName", "sheetName", "params"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "templateName": { "type": "string" },
    "sheets": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "cellName": { "type": "string" },
          "status": { "enum": ["created", "already_exists"] },
          "budgetCap": { "type": "integer", "description": "Token budget cap for this cell" }
        }
      }
    },
    "totalBudgetCap": { "type": "integer" }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_map",
  "arguments": {
    "templateName": "analyze-module",
    "sheetName": "mol-algebraic-survey",
    "params": [
      { "name": "analyze-dispatch", "values": { "module": "internal/dispatch", "focus": "routing logic" } },
      { "name": "analyze-beads", "values": { "module": "internal/beads", "focus": "storage layer" } },
      { "name": "analyze-witness", "values": { "module": "internal/witness", "focus": "health monitoring" } }
    ]
  }
}
```

**Example Response**

```json
{
  "templateName": "analyze-module",
  "sheets": [
    { "cellName": "analyze-dispatch", "status": "created", "budgetCap": 15000 },
    { "cellName": "analyze-beads", "status": "created", "budgetCap": 12000 },
    { "cellName": "analyze-witness", "status": "created", "budgetCap": 9000 }
  ],
  "totalBudgetCap": 36000
}
```

---

### 10. `gas_city_aggregate`

Combine outputs from multiple cells (or all cells matching a pattern) into a
single aggregated result. This is the pivot table / reduce operation.

**Input Schema**

```json
{
  "type": "object",
  "properties": {
    "sheetName": { "type": "string" },
    "sourceCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Cell names to aggregate. Supports glob patterns (e.g. 'analyze-*')."
    },
    "policy": {
      "enum": ["summarize", "extract", "classify", "decide"],
      "description": "summarize: produce a condensed overview. extract: pull structured data. classify: categorize each cell. decide: make a recommendation based on all inputs."
    },
    "prompt": {
      "type": "string",
      "description": "Optional custom prompt to guide aggregation. If omitted, uses policy default."
    }
  },
  "required": ["sheetName", "sourceCells", "policy"]
}
```

**Output Schema**

```json
{
  "type": "object",
  "properties": {
    "aggregation": { "type": "string", "description": "The aggregated result" },
    "inputCount": { "type": "integer" },
    "inputCells": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Resolved cell names (after glob expansion)"
    },
    "compressionRatio": {
      "type": "number",
      "description": "Ratio of total input tokens to output tokens"
    },
    "tokensUsed": { "type": "integer" },
    "model": { "type": "string" }
  }
}
```

**Example Call**

```json
{
  "tool": "gas_city_aggregate",
  "arguments": {
    "sheetName": "mol-algebraic-survey",
    "sourceCells": ["spatial-structure", "work-distribution", "communication", "persistence-state"],
    "policy": "extract",
    "prompt": "Extract all named algebraic structures (monoids, categories, functors, etc.) mentioned across these analyses. Return as a typed inventory."
  }
}
```

**Example Response**

```json
{
  "aggregation": "## Algebraic Structure Inventory\n\n| Structure | Type | Source Cell | Description |\n|-----------|------|------------|-------------|\n| Town→Rig→Polecat | Presheaf (Cat^op → Set) | spatial-structure | Organizational hierarchy as functor |\n| Dispatch queue | Priority monoid (ℕ, min, ∞) | work-distribution | Priority-based merge of work items |\n| Mail system | Free category on agent graph | communication | Composable message routing |\n| Dolt history | Monad (T, η, μ) | persistence-state | Commit/branch/merge as monadic operations |",
  "inputCount": 4,
  "inputCells": ["spatial-structure", "work-distribution", "communication", "persistence-state"],
  "compressionRatio": 6.2,
  "tokensUsed": 9400,
  "model": "claude-sonnet-4-6"
}
```

---

## MCP Resources

Read-only data the agent can reference without making tool calls. Resources are
accessed via `gas-city://` URIs.

### `gas-city://sheets`

List all available sheets.

```json
{
  "uri": "gas-city://sheets",
  "name": "Sheet Index",
  "mimeType": "application/json"
}
```

**Content**

```json
[
  {
    "name": "mol-algebraic-survey",
    "cellCount": 8,
    "freshCount": 5,
    "staleCount": 2,
    "emptyCount": 1,
    "totalTokens": 67580,
    "lastModified": "2026-03-08T14:22:00Z"
  },
  {
    "name": "mol-codebase-audit",
    "cellCount": 12,
    "freshCount": 12,
    "staleCount": 0,
    "emptyCount": 0,
    "totalTokens": 124300,
    "lastModified": "2026-03-07T09:15:00Z"
  }
]
```

### `gas-city://sheets/{name}/cells`

All cells in a sheet with their metadata (no values — use `gas_city_sheet_status`
with `includeValues` for that).

```json
{
  "uri": "gas-city://sheets/mol-algebraic-survey/cells",
  "name": "mol-algebraic-survey cells",
  "mimeType": "application/json"
}
```

**Content**

```json
[
  {
    "name": "spatial-structure",
    "status": "fresh",
    "version": 3,
    "depth": 0,
    "quality": "refined",
    "inputs": [],
    "outputs": ["synthesis"],
    "lastEval": "2026-03-08T12:00:00Z"
  },
  {
    "name": "synthesis",
    "status": "stale",
    "version": 2,
    "depth": 1,
    "quality": "draft",
    "inputs": ["spatial-structure", "work-distribution", "communication"],
    "outputs": [],
    "lastEval": "2026-03-07T18:30:00Z"
  }
]
```

### `gas-city://sheets/{name}/sankey`

Information flow data for visualization — token volumes along each wire.

```json
{
  "uri": "gas-city://sheets/mol-algebraic-survey/sankey",
  "name": "mol-algebraic-survey information flow",
  "mimeType": "application/json"
}
```

**Content**

```json
{
  "nodes": [
    { "id": "spatial-structure", "tokens": 12840, "depth": 0 },
    { "id": "work-distribution", "tokens": 18200, "depth": 0 },
    { "id": "communication", "tokens": 9800, "depth": 0 },
    { "id": "synthesis", "tokens": 8540, "depth": 1 }
  ],
  "links": [
    { "source": "spatial-structure", "target": "synthesis", "tokensTransferred": 12840, "compressionRatio": 1.0 },
    { "source": "work-distribution", "target": "synthesis", "tokensTransferred": 18200, "compressionRatio": 1.0 },
    { "source": "communication", "target": "synthesis", "tokensTransferred": 9800, "compressionRatio": 1.0 }
  ],
  "totalInputTokens": 40840,
  "totalOutputTokens": 8540,
  "overallCompression": 4.78
}
```

### `gas-city://sheets/{name}/stale`

Stale cells with reasons — why they're stale and what would fix them.

```json
{
  "uri": "gas-city://sheets/mol-algebraic-survey/stale",
  "name": "mol-algebraic-survey stale cells",
  "mimeType": "application/json"
}
```

**Content**

```json
[
  {
    "cell": "work-distribution",
    "reason": "input_changed",
    "detail": "Upstream cell 'dispatch-code' was modified at version 4 (cell last evaluated against version 2)",
    "staleInputs": [
      { "cell": "dispatch-code", "cellVersion": 2, "currentVersion": 4 }
    ],
    "lastRunCost": 18200
  },
  {
    "cell": "synthesis",
    "reason": "transitive",
    "detail": "Input 'work-distribution' is stale (transitive staleness)",
    "staleInputs": [
      { "cell": "work-distribution", "cellVersion": 2, "currentVersion": 2, "note": "stale itself" }
    ],
    "lastRunCost": 8500
  }
]
```

---

## Agent Workflow Examples

### Debugging: "Why does the synthesis look wrong?"

An agent investigating a bad synthesis output would use the tools in this sequence:

```
1. gas_city_sheet_status("mol-algebraic-survey", includeValues=true)
   → See all cells. Notice synthesis is stale, quality=draft.

2. gas_city_trace("mol-algebraic-survey", "synthesis")
   → See the computation chain. Bottleneck is work-distribution.
   → Notice work-distribution was computed against dispatch-code v2, now v4.

3. gas_city_diff("mol-algebraic-survey", "work-distribution", 2, 3)
   → See what changed: "Convoy Batching" section was added.
   → The synthesis doesn't account for convoy batching — that's the bug.

4. gas_city_eval("mol-algebraic-survey", "synthesis", dryRun=true)
   → Preview the prompt. Confirm it now includes the updated work-distribution.
   → Last run cost: ~8500 tokens. Budget cap not exceeded.

5. gas_city_eval("mol-algebraic-survey", "synthesis")
   → Recompute. New synthesis includes convoy batching analysis.

6. gas_city_diff("mol-algebraic-survey", "synthesis", 2, 3)
   → Verify the fix. New section "Convoy Batching as Coproduct" appears.
```

### Debugging: "What if the spatial structure were different?"

Pin-based what-if analysis:

```
1. gas_city_snapshot("mol-algebraic-survey", label="before-experiment")
   → Save current state.

2. gas_city_pin("mol-algebraic-survey", "spatial-structure",
     "Assume a flat hierarchy — all agents report directly to Mayor.")
   → Pin to a hypothetical value. synthesis becomes stale.

3. gas_city_eval("mol-algebraic-survey", "synthesis")
   → Recompute synthesis against the pinned value.
   → Observe how the synthesis changes without the rig abstraction layer.

4. gas_city_unpin("mol-algebraic-survey", "spatial-structure")
   → Restore the real value. synthesis becomes stale again.

5. gas_city_eval_stale("mol-algebraic-survey", policy="eager")
   → Restore everything to the real computed state.
```

### Map/Aggregate: "Analyze all modules and synthesize"

Stamping a template across multiple inputs, then reducing:

```
1. gas_city_map(
     templateName="analyze-module",
     sheetName="mol-codebase-audit",
     params=[
       { name: "analyze-dispatch", values: { module: "internal/dispatch" } },
       { name: "analyze-beads", values: { module: "internal/beads" } },
       { name: "analyze-witness", values: { module: "internal/witness" } },
       { name: "analyze-refinery", values: { module: "internal/refinery" } },
       { name: "analyze-polecat", values: { module: "internal/polecat" } }
     ])
   → Creates 5 new cells. Budget cap: 80000 tokens.

2. gas_city_eval_stale("mol-codebase-audit", policy="eager")
   → Evaluates all 5 in topological order (they're independent, so parallel).
   → 62000 tokens consumed. All cells now fresh.

3. gas_city_aggregate(
     sheetName="mol-codebase-audit",
     sourceCells=["analyze-*"],
     policy="extract",
     prompt="Extract all public API surfaces, their callers, and breaking change risks.")
   → Aggregates 5 cell outputs into a structured API inventory.
   → Compression ratio: 8.3x.

4. gas_city_aggregate(
     sheetName="mol-codebase-audit",
     sourceCells=["analyze-*"],
     policy="decide",
     prompt="Which module is the highest risk for the upcoming dispatch refactor?")
   → "analyze-beads has the highest coupling to dispatch internals (14 direct imports).
   →  Recommend: refactor beads/dispatch interface first to reduce blast radius."
```

### Convergent Evaluation: "Keep recomputing until stable"

For sheets with feedback loops (cell A informs B, B's output changes what A
should say next):

```
1. gas_city_eval_stale("mol-iterative-design", policy="convergent", maxRounds=5)
   → Round 1: recompute A, B, C. B changes → A becomes stale.
   → Round 2: recompute A. A changes → B becomes stale.
   → Round 3: recompute B. No changes. Converged.
   → { rounds: 3, converged: true, totalTokensUsed: 34000 }
```

---

## Error Model

All errors follow the same envelope:

```json
{
  "error": "<error_code>",
  "detail": "<human-readable explanation>",
  ...additional fields specific to the error type
}
```

### Common Error Codes

| Code | HTTP Equiv | Meaning |
|------|-----------|---------|
| `sheet_not_found` | 404 | Sheet doesn't exist |
| `cell_not_found` | 404 | Cell doesn't exist in the specified sheet |
| `version_not_found` | 404 | Requested version number doesn't exist |
| `not_pinned` | 409 | Attempted to unpin a cell that isn't pinned |
| `inputs_empty` | 422 | One or more required input cells have no value |
| `budget_exceeded` | 429 | Operation would exceed the token budget |
| `cycle_detected` | 422 | Sheet DAG contains a cycle |
| `no_stale_cells` | 200 | Not really an error — nothing to recompute |
| `eval_failed` | 502 | Upstream LLM call failed |
| `storage_error` | 503 | Dolt database unreachable or query failed |
| `template_not_found` | 404 | Template cell for map operation doesn't exist |
| `invalid_glob` | 400 | Glob pattern in sourceCells is malformed |

---

## Implementation Notes

### Transport

The server implements the MCP specification over stdio (default) or SSE.
The `gt mcp serve` command starts the server process. Claude Code and Claude
Desktop both support stdio transport natively.

### Authentication

Local-only by default (stdio transport). For SSE transport, the server binds
to `127.0.0.1` and requires a bearer token generated at startup (printed to
stderr). Remote access requires an explicit `--bind 0.0.0.0` flag and token
configuration.

### Streaming

Long-running operations (`gas_city_eval`, `gas_city_eval_stale`) support MCP
progress notifications:

```json
{
  "method": "notifications/progress",
  "params": {
    "progressToken": "eval-synthesis-1709913720",
    "progress": 0.6,
    "total": 1.0,
    "message": "Evaluating synthesis (3/5 inputs resolved)"
  }
}
```

### Concurrency

The server handles multiple concurrent tool calls. Cell evaluations that share
no dependencies run in parallel. The server maintains a lock per cell to prevent
concurrent writes to the same cell.

### Budget Enforcement

Token budgets are enforced at two levels:
1. **Per-call**: `gas_city_eval` checks cumulative spend against the budget cap
   before calling the LLM. If the cap is already exceeded, the call is rejected.
2. **Per-sheet**: Each sheet has a configurable total budget. The server rejects
   evaluations that would exceed the sheet budget.

Budget is tracked in the Dolt database alongside cell metadata.
