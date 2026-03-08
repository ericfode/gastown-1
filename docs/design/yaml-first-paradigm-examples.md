# Paradigm A: YAML-First Reactive Sheets — Example Use Cases

**Bead**: gt-x6e
**Date**: 2026-03-08

---

In the YAML-first paradigm, a YAML file defines the sheet schema: cell names, types,
`{{ref}}` template wiring, and prompts. A command (`gt sheet status <file.yaml>`)
stamps this into the reactive engine. The YAML is the schema; beads are runtime instances.

Each example below provides:
1. The YAML sheet definition
2. A narrative walkthrough of evaluation order and data flow
3. Why the topology suits the task

Cell types: `text`, `inventory`, `diagram`, `laws`, `boundaries`, `synthesis`, `code`, `decision`

---

## Example 1: Incident Post-Mortem Analysis

**Domain**: Site reliability / operations
**Cells**: 6 | **Topology**: Diamond with fan-out to dual sinks

### YAML Definition

```yaml
name: incident-postmortem
cells:
  - name: timeline
    type: inventory
    prompt: |
      Reconstruct the incident timeline from the following log excerpts,
      alert history, and chat transcripts. List events chronologically
      with timestamps, actors, and systems affected.

      Logs: (attached at eval time)

  - name: blast-radius
    type: boundaries
    prompt: |
      From the incident timeline, determine the blast radius:
      - Which services were directly affected?
      - Which downstream consumers experienced degradation?
      - What was the user-facing impact (error rates, latency, data loss)?
      - Geographic and tenant scope.

      Timeline:
      {{timeline}}

  - name: root-cause
    type: synthesis
    prompt: |
      Analyze the incident timeline for root cause. Apply the "5 Whys"
      method. Distinguish between:
      - Proximate cause (the trigger)
      - Contributing factors (what made it worse)
      - Systemic cause (why the system was vulnerable)

      Timeline:
      {{timeline}}

  - name: detection-gap
    type: laws
    refs: [timeline, blast-radius]
    prompt: |
      Compare the incident timeline against the blast radius assessment.
      Identify detection gaps:
      - How long between first symptom and first alert?
      - Were any affected services missing monitoring?
      - What signal would have caught this earlier?

      Timeline:
      {{timeline}}

      Blast radius:
      {{blast-radius}}

  - name: remediation-plan
    type: code
    refs: [root-cause, detection-gap]
    prompt: |
      Given the root cause analysis and detection gaps, produce a
      remediation plan with concrete action items. Each item must have:
      - Owner (team, not individual)
      - Priority (P0-P3)
      - Estimated effort (hours/days/weeks)
      - Verification criteria

      Root cause:
      {{root-cause}}

      Detection gaps:
      {{detection-gap}}

  - name: executive-summary
    type: decision
    refs: [blast-radius, root-cause, remediation-plan]
    prompt: |
      Write a 1-page executive summary of this incident for leadership.
      Include: what happened (2 sentences), impact (metrics), root cause
      (1 sentence), and top 3 remediation items with owners and timelines.
      No jargon. No blame.

      Blast radius:
      {{blast-radius}}

      Root cause:
      {{root-cause}}

      Remediation plan:
      {{remediation-plan}}
```

### DAG Structure

```
  timeline (inventory, depth 0)
  ├──→ blast-radius (boundaries, depth 1)
  │    ├──→ detection-gap (laws, depth 2) ←── timeline [diamond]
  │    └──→ executive-summary (decision, depth 3)
  └──→ root-cause (synthesis, depth 1)
       ├──→ remediation-plan (code, depth 2) ←── detection-gap
       └──→ executive-summary ←── remediation-plan
```

### Narrative Walkthrough

**Step 0 — Initial state.** All 6 cells are `empty`. The ready set is `{timeline}`
because it has no upstream refs.

**Step 1 — Eval `timeline`.** The agent reads raw logs and chat transcripts (attached
at eval time as the prompt's input). Output: a structured chronological event list.
State: `timeline` → `fresh`. Downstream `blast-radius` and `root-cause` become ready.

**Step 2 — Eval `blast-radius` and `root-cause` (parallel).** Both depend only on
`timeline` (now fresh). They can run concurrently. `blast-radius` maps the impact
surface. `root-cause` digs into causality. Neither sees the other's output.

**Step 3 — Eval `detection-gap`.** Now ready: it needs `timeline` (fresh) and
`blast-radius` (fresh). This is the diamond join — `timeline` flows into
`detection-gap` both directly and via `blast-radius`. The direct wire gives the raw
events; the `blast-radius` wire gives the interpreted impact. The agent correlates
the two to find monitoring blind spots.

**Step 4 — Eval `remediation-plan`.** Needs `root-cause` (fresh) and `detection-gap`
(fresh). Produces actionable items grounded in both the systemic cause and the
observability gaps.

**Step 5 — Eval `executive-summary`.** The final sink. Needs `blast-radius`,
`root-cause`, and `remediation-plan`. Compresses the full analysis into a leadership-
readable page. This is a 3-input fan-in — the highest-cardinality join in the sheet.

**Staleness scenario.** If new log data arrives and `timeline` is re-evaluated,
staleness propagates through the entire DAG. The topological recompute order ensures
no cell runs with stale inputs.

### Why This Topology

The diamond (`timeline` → `blast-radius` → `detection-gap` ← `timeline`) lets the
detection-gap analysis see both the raw events AND the interpreted impact, enabling
cross-referencing. The dual-sink pattern (`remediation-plan` and `executive-summary`
both consuming different subsets) means the remediation plan can be detailed/technical
while the executive summary stays high-level — they don't compete for the same prompt
budget. The fan-out at depth 1 (`blast-radius` || `root-cause`) allows parallel
evaluation, reducing wall-clock time.

---

## Example 2: API Migration Impact Assessment

**Domain**: Platform engineering / developer experience
**Cells**: 7 | **Topology**: Two-source fan-in with diamond and secondary fan-out

### YAML Definition

```yaml
name: api-migration-assessment
cells:
  - name: old-api-surface
    type: inventory
    prompt: |
      Catalog the current (v2) API surface. For each endpoint, list:
      - HTTP method and path
      - Request/response schema (key fields only)
      - Rate limits and auth requirements
      - Known consumers (services and teams)

      API spec: (attached at eval time)

  - name: new-api-surface
    type: inventory
    prompt: |
      Catalog the proposed (v3) API surface. For each endpoint, list:
      - HTTP method and path
      - Request/response schema (key fields only)
      - Breaking vs non-breaking changes from v2
      - New capabilities not present in v2

      API spec: (attached at eval time)

  - name: breaking-changes
    type: boundaries
    refs: [old-api-surface, new-api-surface]
    prompt: |
      Diff the v2 and v3 API surfaces. Produce a table of breaking changes:
      | v2 Endpoint | Change Type | v3 Equivalent | Migration Effort |
      Classify each as: removed, renamed, schema-changed, auth-changed,
      or semantics-changed.

      v2 API:
      {{old-api-surface}}

      v3 API:
      {{new-api-surface}}

  - name: consumer-impact
    type: synthesis
    refs: [old-api-surface, breaking-changes]
    prompt: |
      For each known consumer from the v2 API inventory, assess impact:
      - Which breaking changes affect this consumer?
      - Estimated migration effort (trivial/moderate/significant)
      - Can they run dual-stack (v2+v3) during transition?
      - Risk level if they don't migrate by deprecation date

      v2 consumers:
      {{old-api-surface}}

      Breaking changes:
      {{breaking-changes}}

  - name: compatibility-shim
    type: code
    refs: [breaking-changes]
    prompt: |
      Design a compatibility shim layer that translates v2 requests into
      v3 calls. For each breaking change, specify:
      - Translation logic (pseudocode)
      - Edge cases the shim cannot handle
      - Performance overhead estimate
      - Sunset timeline recommendation

      Breaking changes:
      {{breaking-changes}}

  - name: migration-runbook
    type: code
    refs: [consumer-impact, compatibility-shim]
    prompt: |
      Write a migration runbook for platform consumers. Structure as phases:
      Phase 1: Deploy compatibility shim (zero consumer changes)
      Phase 2: Early adopter migration (willing teams)
      Phase 3: Bulk migration (remaining consumers)
      Phase 4: Shim sunset and v2 decommission

      For each phase, specify: entry criteria, steps, rollback plan,
      success metrics.

      Consumer impact:
      {{consumer-impact}}

      Compatibility shim:
      {{compatibility-shim}}

  - name: go-no-go
    type: decision
    refs: [consumer-impact, migration-runbook]
    prompt: |
      Render a go/no-go decision for the v2→v3 migration. Consider:
      - Total consumer count and high-risk consumer count
      - Shim coverage (what % of breaking changes are shimmable?)
      - Aggregate migration effort across all consumers
      - Risk of NOT migrating (v2 tech debt, security, performance)

      Recommendation must be: GO, CONDITIONAL GO (with conditions), or NO-GO.

      Consumer impact:
      {{consumer-impact}}

      Migration runbook:
      {{migration-runbook}}
```

### DAG Structure

```
  old-api-surface (inventory, depth 0)     new-api-surface (inventory, depth 0)
       │                                          │
       └──────────┐                    ┌──────────┘
                  ▼                    ▼
             breaking-changes (boundaries, depth 1)
              ├──→ consumer-impact (synthesis, depth 2) ←── old-api-surface [diamond]
              └──→ compatibility-shim (code, depth 2)
                         │                    │
                         ▼                    ▼
                    migration-runbook (code, depth 3)
                         │
                         ▼
                    go-no-go (decision, depth 4) ←── consumer-impact [diamond]
```

### Narrative Walkthrough

**Step 0 — Initial state.** 7 cells, all `empty`. Ready set: `{old-api-surface,
new-api-surface}` — two independent leaf cells.

**Step 1 — Eval `old-api-surface` and `new-api-surface` (parallel).** Two independent
inventories. Can run simultaneously. Each reads its respective API specification
document.

**Step 2 — Eval `breaking-changes`.** Now ready: both inventories are fresh. This is
the first fan-in — it joins two independent data sources to produce a structured diff.
This cell is the critical bottleneck: everything downstream depends on its output.

**Step 3 — Eval `consumer-impact` and `compatibility-shim` (parallel).** Both depend
on `breaking-changes` (fresh). `consumer-impact` also pulls from `old-api-surface`
directly (the diamond: `old-api-surface` → `breaking-changes` → `consumer-impact`
← `old-api-surface`). This lets the impact assessment cross-reference the raw consumer
list with the breaking change analysis. Meanwhile, `compatibility-shim` focuses purely
on the technical translation layer.

**Step 4 — Eval `migration-runbook`.** Needs both `consumer-impact` and
`compatibility-shim`. Synthesizes the "who's affected" with the "how to bridge it"
into a phased execution plan.

**Step 5 — Eval `go-no-go`.** Final decision cell. Needs `consumer-impact` (directly,
for the risk profile) and `migration-runbook` (for feasibility). The second diamond:
`consumer-impact` flows into `go-no-go` both directly and through `migration-runbook`.

**Staleness scenario.** If the v3 API spec changes (common during design), only
`new-api-surface` is re-evaluated. Staleness cascades through `breaking-changes` →
everything downstream. But `old-api-surface` stays fresh — the v2 spec hasn't changed.
The recompute only touches cells affected by the changed input, saving tokens.

### Why This Topology

The two-source design mirrors reality: a migration inherently compares two systems.
The diamond on `old-api-surface` ensures the consumer-impact cell has both the raw
consumer list AND the breaking-change context — it can say "team X uses endpoint Y,
which is affected by breaking change Z." The secondary fan-out at depth 2
(`consumer-impact` || `compatibility-shim`) separates concerns: people impact from
technical solution. The second diamond on `consumer-impact` into `go-no-go` means
the decision-maker sees both the raw risk profile and the mitigated risk (after the
runbook accounts for the shim). This prevents the decision from being unduly
optimistic about migration feasibility.

---

## Example 3: Supply Chain Vulnerability Triage

**Domain**: Security engineering
**Cells**: 8 | **Topology**: Three-source fan-in, diamond lattice, dual decision sinks

### YAML Definition

```yaml
name: supply-chain-vuln-triage
cells:
  - name: vuln-feed
    type: inventory
    prompt: |
      Parse the vulnerability advisory feed. For each CVE, extract:
      - CVE ID, CVSS score, attack vector
      - Affected package and version range
      - Patch availability (yes/no, which version)
      - Exploit maturity (POC, weaponized, in-the-wild)

      Feed data: (attached at eval time)

  - name: dep-graph
    type: inventory
    prompt: |
      Analyze the project's dependency graph. For each direct and
      transitive dependency, list:
      - Package name and locked version
      - Depth in dependency tree (direct=0, transitive=1+)
      - Number of reverse dependents in our codebase
      - Last update date

      Lock file: (attached at eval time)

  - name: runtime-profile
    type: boundaries
    prompt: |
      From production telemetry, determine which dependencies are
      actually loaded at runtime vs build-time-only:
      - Packages imported in hot paths (request handling)
      - Packages used only in tests, codegen, or CI
      - Packages imported but never called (dead code)

      Telemetry data: (attached at eval time)

  - name: exposure-map
    type: synthesis
    refs: [vuln-feed, dep-graph, runtime-profile]
    prompt: |
      Cross-reference the vulnerability feed against our dependency
      graph and runtime profile. For each CVE that affects us:
      - Is the vulnerable package a direct or transitive dep?
      - Is it loaded at runtime or build-time-only?
      - Is the vulnerable code path reachable from our code?
      - Effective CVSS (adjusted for our exposure)

      Vulnerabilities:
      {{vuln-feed}}

      Dependencies:
      {{dep-graph}}

      Runtime profile:
      {{runtime-profile}}

  - name: upgrade-paths
    type: code
    refs: [exposure-map, dep-graph]
    prompt: |
      For each exposed CVE, determine the upgrade path:
      - Direct dep: bump to patched version, check API compat
      - Transitive dep: which direct dep pulls it in? Can we bump?
      - If no patch exists: is there a workaround or alternative package?
      - Dependency conflicts (version constraints from other deps)

      Exposure map:
      {{exposure-map}}

      Full dependency graph:
      {{dep-graph}}

  - name: risk-matrix
    type: diagram
    refs: [exposure-map]
    prompt: |
      Build a risk matrix (likelihood × impact) for all exposed CVEs.
      Classify each into quadrants:
      - CRITICAL: high likelihood + high impact → patch now
      - HIGH: high in one dimension → patch this sprint
      - MEDIUM: moderate in both → schedule for next cycle
      - LOW: low in both → accept risk, monitor

      Use the effective CVSS scores and exploit maturity from the
      exposure map. Output as a structured table.

      Exposure map:
      {{exposure-map}}

  - name: patch-plan
    type: decision
    refs: [upgrade-paths, risk-matrix]
    prompt: |
      Produce a prioritized patch plan. Order by risk quadrant
      (CRITICAL first), then by upgrade complexity (easiest first
      within each quadrant). For each item:
      - CVE ID and affected package
      - Risk quadrant
      - Upgrade action (bump version / replace package / apply workaround)
      - Estimated CI breakage risk (low/medium/high)
      - Recommended branch strategy (hotfix / feature branch / batch PR)

      Upgrade paths:
      {{upgrade-paths}}

      Risk matrix:
      {{risk-matrix}}

  - name: stakeholder-brief
    type: text
    refs: [risk-matrix, patch-plan]
    prompt: |
      Write a stakeholder brief (for engineering leadership and security
      team) summarizing:
      - Total CVEs found / total affecting us / total critical
      - Key risk: the single scariest finding, in plain language
      - Patch plan summary: how many patches, estimated total effort
      - Residual risk: what we're choosing NOT to patch and why

      Risk matrix:
      {{risk-matrix}}

      Patch plan:
      {{patch-plan}}
```

### DAG Structure

```
  vuln-feed (inventory, d0)   dep-graph (inventory, d0)   runtime-profile (boundaries, d0)
       │                           │         │                      │
       └───────────┐               │         │           ┌─────────┘
                   ▼               ▼         │           ▼
              exposure-map (synthesis, d1) ←─┼───────────┘
               ├──→ upgrade-paths (code, d2) ←── dep-graph [diamond]
               └──→ risk-matrix (diagram, d2)
                         │              │
                         ▼              ▼
                    patch-plan (decision, d3)
                         │
                         ▼
                    stakeholder-brief (text, d4) ←── risk-matrix [diamond]
```

### Narrative Walkthrough

**Step 0 — Initial state.** 8 cells, all `empty`. Ready set: `{vuln-feed, dep-graph,
runtime-profile}` — three independent leaf cells.

**Step 1 — Eval `vuln-feed`, `dep-graph`, and `runtime-profile` (parallel).** Three
fully independent data ingestion cells. Maximum parallelism at the leaves. Each
processes a different data source: CVE advisories, the lock file, and production
telemetry.

**Step 2 — Eval `exposure-map`.** The critical three-way fan-in. This cell joins
vulnerability data with dependency structure with runtime behavior. The three-input
join is what makes this analysis valuable — a CVE in a build-only dependency that's
never loaded at runtime is a different beast than one in a hot-path package. Only the
exposure-map cell has all three perspectives.

**Step 3 — Eval `upgrade-paths` and `risk-matrix` (parallel).** Both depend on
`exposure-map`. `upgrade-paths` also pulls `dep-graph` directly (the diamond:
`dep-graph` → `exposure-map` → `upgrade-paths` ← `dep-graph`). This diamond ensures
the upgrade analysis has both the filtered exposure (which CVEs matter?) AND the full
dependency tree (what are the transitive constraint chains?). Meanwhile, `risk-matrix`
focuses purely on severity classification.

**Step 4 — Eval `patch-plan`.** Joins `upgrade-paths` (the how) with `risk-matrix`
(the priority). This produces the actionable ordered list.

**Step 5 — Eval `stakeholder-brief`.** The final sink. Needs `risk-matrix` (directly,
for the severity overview) and `patch-plan` (for the action items). The diamond on
`risk-matrix` means the brief can present both the raw risk picture and the mitigation
plan — preventing the stakeholder communication from being either alarmist (showing
only risk) or dismissive (showing only the plan).

**Staleness scenario.** Vulnerability feeds update frequently. When `vuln-feed` gets
new data, only the left branch of the DAG is initially stale: `exposure-map` →
everything downstream. But `dep-graph` and `runtime-profile` remain fresh — the
project's dependency tree and runtime behavior haven't changed. The recompute touches
5 cells (exposure-map, upgrade-paths, risk-matrix, patch-plan, stakeholder-brief),
skipping the 2 unaffected leaf cells. If instead the project ships a new release
(changing `dep-graph`), a different staleness pattern emerges: `dep-graph` →
`exposure-map` AND `upgrade-paths` (direct wire) → downstream. The `runtime-profile`
may also need re-evaluation if the release changes import paths.

### Why This Topology

The three-source fan-in at `exposure-map` is the defining feature. Security triage
requires correlating three independent signals: what's vulnerable (vuln-feed), what we
use (dep-graph), and what's actually loaded (runtime-profile). No single source gives
the full picture. The diamond on `dep-graph` into `upgrade-paths` ensures the upgrade
analysis can reason about the full dependency constraint graph, not just the filtered
exposure list. The dual-sink pattern (`patch-plan` for the engineering team,
`stakeholder-brief` for leadership) produces appropriately-scoped outputs from the
same underlying analysis. The `risk-matrix` diamond into `stakeholder-brief` prevents
the classic security communication failure where leadership sees only a remediation
plan without understanding the underlying risk landscape.

---

## Summary: Topology Comparison

| Sheet | Cells | Depth | Diamonds | Max Parallel | Sinks |
|-------|-------|-------|----------|-------------|-------|
| incident-postmortem | 6 | 4 | 1 (timeline) | 2 (blast-radius ‖ root-cause) | 1 |
| api-migration | 7 | 5 | 2 (old-api, consumer-impact) | 2 at depths 0, 1, 2 | 1 |
| supply-chain-vuln | 8 | 5 | 2 (dep-graph, risk-matrix) | 3 at depth 0, 2 at depth 2 | 1 |

Each example demonstrates a different fan-in/fan-out pattern:
- **Incident postmortem**: Classic diamond — one source feeds two parallel analyses that
  rejoin at a synthesis cell.
- **API migration**: Two independent sources merge at a bottleneck, then fan out again
  before rejoining at a decision.
- **Supply chain vuln**: Three independent sources merge at a critical join, then
  fan out with a diamond bypass for constraint-aware reasoning.

All three are valid `gt sheet` YAML files. Run `gt sheet status <file>.yaml` to see
cell states, `gt sheet ready <file>.yaml` to see what's evaluable, and
`gt sheet dag <file>.yaml` to visualize the dependency graph.
