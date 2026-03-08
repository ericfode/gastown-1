# Algebraic Survey: Work Distribution Subsystem

**Subsystem**: Work Distribution
**Packages**: `formula`, `convoy`, `feed`, `scheduler/capacity`, `session`
**Purpose**: Decides WHO does WHAT — formulas define workflow templates, convoys batch
work, feed curates dispatch events, scheduler manages capacity, sessions execute work.

## 1. Subsystem Overview

The Work Distribution subsystem answers: given a set of work items (beads) and a set
of execution contexts (polecats in rigs), how does work get defined, dispatched,
tracked, and reported?

```
Formula (template) → Convoy (batch) → Scheduler (capacity) → Session (execution)
                                          ↓
                                        Feed (observation)
```

### Package Dependency Graph (within subsystem)

```
formula ←── (no internal deps, leaf package)
convoy  ←── beads, beadsdk, util (no subsystem deps)
feed    ←── events, config (no subsystem deps)
scheduler/capacity ←── config (no subsystem deps)
session ←── config, tmux, telemetry (no subsystem deps)
```

**Key finding**: These five packages share NO direct import dependencies within the
subsystem. They are coupled only through higher-level orchestration (cmd/, daemon/,
polecat/, witness/) and shared domain types (beads, events, config). This is a
**hub-and-spoke** architecture, not a pipeline.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| formula | cmd/formula, cmd/patrol_report, wisp |
| convoy | daemon/convoy_manager, cmd/convoy_stage |
| feed | daemon/daemon (Curator lifecycle) |
| scheduler/capacity | cmd/capacity_dispatch, cmd/sling_schedule |
| session | polecat, refinery, witness, crew, mayor, deacon, daemon, dog, mail |

---

## 2. Package: formula

**Files**: types.go, parser.go, embed.go, variable_validation.go, doc.go

### Type Algebra

The formula package defines a **tagged union** (sum type) with four variants:

```
FormulaType = TypeConvoy | TypeWorkflow | TypeExpansion | TypeAspect
```

Each variant populates different fields of the `Formula` struct:

| Variant | Active Fields | Execution Model |
|---------|--------------|-----------------|
| TypeConvoy | Legs, Synthesis, Inputs, Output | Parallel legs → synthesis |
| TypeWorkflow | Steps, Vars | DAG-scheduled sequential/parallel steps |
| TypeExpansion | Template | Generative DAG (template instantiation) |
| TypeAspect | Aspects | Parallel analysis (no synthesis) |

### Core Types

```
Formula {
  Name: String, Description: String, Type: FormulaType,
  Version: Nat, Pour: Bool, Agent: String,
  -- Convoy
  Inputs: Map String Input, Prompts: Map String String,
  Output: Option Output, Legs: List Leg, Synthesis: Option Synthesis,
  -- Workflow
  Steps: List Step, Vars: Map String Var,
  -- Expansion
  Template: List Template,
  -- Aspect
  Aspects: List Aspect
}

Step { ID Title Description: String, Needs: List String, Parallel: Bool, Acceptance: String }
Leg { ID Title Focus Description Agent: String }
Template { ID Title Description: String, Needs: List String }
Aspect { ID Title Focus Description: String }
Input { Description Type: String, Required: Bool, RequiredUnless: List String, Default: String }
Output { Directory LegPattern Synthesis: String }
Synthesis { Title Description: String, DependsOn: List String }
Var { Description: String, Required: Bool, Default: String }
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| Parse | Bytes → Formula × Error | Total (all TOML maps to Formula or error) |
| inferType | Formula → Formula | Idempotent (f.inferType().inferType() = f.inferType()) |
| Validate | Formula → Error | Type-dispatched (convoy/workflow/expansion/aspect) |
| TopologicalSort | Formula → List String × Error | Kahn's algorithm; identity for convoy/aspect |
| ReadySteps | Formula × Set String → List String | Monotone (more completed → fewer ready OR same) |
| ParallelReadySteps | Formula × Set String → List String × String | Partitions ready into {parallel, sequential} |
| GetDependencies | Formula × String → List String | Projection from dependency graph |
| checkCycles | Formula → Error | DFS cycle detection on dependency DAG |
| ExtractTemplateVariables | String → List String | Regex extraction, deduplicated, sorted |
| ValidateTemplateVariables | Formula → Error | Cross-reference: extracted vars ⊆ defined vars ∪ inputs |

### Embedding & Health System

```
InstalledRecord { Formulas: Map String String }  -- filename → SHA256
FormulaStatus { Name Status EmbeddedHash InstalledHash CurrentHash: String }
HealthReport { Formulas: List FormulaStatus, OK Outdated Modified Missing New Untracked Error: Nat }
```

Status classification: `(embeddedHash, installedHash, currentHash) → Status`

| Condition | Status |
|-----------|--------|
| all equal | ok |
| embedded ≠ installed, installed = current | outdated |
| installed ≠ current | modified |
| no current file | missing |
| no installed record | new |
| current file, no embedded | untracked |

### Lean 4 Formalization Candidates

1. **DAG properties**: TopologicalSort correctness (Kahn's terminates iff acyclic)
2. **ReadySteps monotonicity**: Adding to completed set never adds new dependencies
3. **Cycle detection soundness**: checkCycles returns error iff dependency graph has cycle
4. **Type inference determinism**: inferType assigns unique type based on field presence
5. **Variable validation completeness**: ValidateTemplateVariables catches all undefined vars

---

## 3. Package: convoy

**Files**: operations.go (469 lines, no subpackages)

### Core Abstraction

A convoy is a bead that **tracks** other beads via `tracks` dependency type. When a
tracked issue closes, the convoy reactor checks if more work should be dispatched.

### Type Algebra

```
trackedIssue { ID Status Assignee: String, Priority: Int, IssueType: String }

slingableTypes : Set IssueType = {task, bug, feature, chore, ""}
blockingDepTypes : Set DepType = {blocks, conditional-blocks, waits-for, merge-blocks}
```

### Readiness Predicate

The core algebraic object is the **readiness predicate** for dispatch:

```
Ready(issue) ≡ issue.Status = "open"
             ∧ issue.Assignee = ""
             ∧ issue.IssueType ∈ slingableTypes
             ∧ ¬Blocked(issue)
             ∧ rigForIssue(issue) ≠ ""
             ∧ ¬isRigParked(rig)
```

### Blocking Relation

```
Blocked(issue) ≡ ∃ dep ∈ Dependencies(issue).
    dep.Type ∈ blockingDepTypes
  ∧ dep.Status ∉ {closed, tombstone}
  ∧ (dep.Type = merge-blocks → ¬(dep.CloseReason.startsWith("Merged in ")))
```

Note the special handling of `merge-blocks`: a closed issue still blocks if it hasn't
been confirmed merged. This prevents premature dispatch.

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| CheckConvoysForIssue | Context × Storage × ... → List String | Idempotent, fail-open |
| feedNextReadyIssue | Context × Storage × ... → () | Dispatches 0 or 1 issue |
| getTrackingConvoys | Context × Storage × String → List String | Reverse dependency lookup |
| isIssueBlocked | Context × Storage × String → Bool | Monotone in closed deps |
| getConvoyTrackedIssues | Context × Storage × String → List trackedIssue | Cross-rig aware |

### Dispatch Ordering

```
sort(issues) = sortBy (Priority ASC, ID ASC) issues
dispatch(sorted) = first(filter(Ready, sorted))
```

Priority is the primary key; ID breaks ties deterministically.

### Fail-Open Pattern

Every function in convoy follows fail-open semantics:
- Nil store → false/empty (no dispatch)
- Missing data → skip (continue to next)
- Error from store → assume not blocked

This ensures convoy never blocks the system due to storage failures.

### Lean 4 Formalization Candidates

1. **Readiness is a conjunction**: Ready(x) ∧ ¬Ready(x) is decidable
2. **Blocking is monotone**: Closing a dependency can only unblock, never block
3. **Dispatch determinism**: Given same state, feedNextReadyIssue always dispatches same issue
4. **At-most-one dispatch**: Each call dispatches exactly 0 or 1 issues
5. **Fail-open safety**: Store errors never cause false positives (spurious dispatch)

---

## 4. Package: feed

**Files**: curator.go (single file, ~544 lines)

### Core Abstraction

The Feed Curator is an **event stream processor** that transforms raw system events
into a curated, human-readable feed. It is a **stateless processor** — all
deduplication and aggregation state is derived from files, not memory.

### Type Algebra

```
FeedEvent { Timestamp Source Type Actor Summary: String, Payload: Map String Any, Count: Nat }

Curator { townRoot: String, maxFeedFileSize: Int64, ctx: Context,
          doneDedupeWindow slingAggregateWindow: Duration, minAggregateCount: Nat }
```

### Pipeline

```
events.Event → processLine → (shouldDedupe? → drop | writeFeedEvent → .feed.jsonl)
```

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| processLine | String → () | Filter by visibility, dedupe, write |
| shouldDedupe | Event → Bool | Only for TypeDone events; file-derived |
| writeFeedEvent | Event → () | Transform, aggregate, lock, append |
| generateSummary | Event → String | Pure function (type → template) |
| truncateFeedFile | String × Int64 → () | Keeps newest half |
| readRecentFeedEvents | Duration → List FeedEvent | Temporal filter with shared lock |
| countRecentSlings | String × Duration → Nat | Aggregate count from file |

### Algebraic Properties

1. **Visibility filter**: `processLine` admits only events with visibility ∈ {Feed, Both}
2. **Dedup window**: `shouldDedupe(e) ≡ e.Type = Done ∧ ∃ f ∈ recentFeed. f.Type = Done ∧ f.Actor = e.Actor ∧ f.ts ≥ now - window`
3. **Aggregation**: `count(slings(actor, window)) ≥ minAggregateCount → aggregate`
4. **Truncation invariant**: After truncation, fileSize ≈ maxFeedFileSize / 2
5. **Zero-knowledge design**: No in-process state between events; all derived from files

### Synchronization

```
feedMu (sync.Mutex)     — in-process feed file protection
flock.Flock (shared)    — cross-process read lock
flock.Flock (exclusive) — cross-process write lock
sync.Once               — idempotent Start()
```

### Lean 4 Formalization Candidates

1. **Dedup correctness**: shouldDedupe never drops non-duplicate events
2. **Truncation preserves newest**: After truncation, all events with ts > cutoff are retained
3. **Aggregation threshold**: Aggregation only occurs when count ≥ minAggregateCount
4. **Visibility filter is a projection**: Only admits Feed ∪ Both ⊂ AllVisibilities

---

## 5. Package: scheduler/capacity

**Files**: config.go, state.go, pipeline.go, dispatch.go

### Core Abstraction

The capacity scheduler manages a **dispatch pipeline**: pending beads are filtered by
readiness and capacity, then dispatched with retry/circuit-breaker policies.

### Type Algebra

```
SchedulerConfig { MaxPolecats BatchSize: Option Int, SpawnDelay: String }
SchedulerState { Paused: Bool, PausedBy PausedAt LastDispatchAt: String, LastDispatchCount: Nat }

PendingBead { ID WorkBeadID Title TargetRig Description: String, Labels: List String,
              Context: Option SlingContextFields }

SlingContextFields { Version: Nat, WorkBeadID TargetRig Formula Args Vars EnqueuedAt
                     Merge Convoy BaseBranch Account Agent Mode LastFailure: String,
                     NoMerge HookRawBead Owned: Bool, DispatchFailures: Nat }

DispatchPlan { ToDispatch: List PendingBead, Skipped: Nat, Reason: String }
DispatchParams { BeadID FormulaName RigName Args Merge BaseBranch Account Agent Mode: String,
                 Vars: List String, NoMerge HookRawBead: Bool }
DispatchReport { Dispatched Failed Skipped: Nat, Reason: String }

FailureAction = FailureRetry | FailureQuarantine
```

### Pipeline Operations

```
PlanDispatch : Nat × Nat × List PendingBead → DispatchPlan
  PlanDispatch(capacity, batchSize, ready) =
    let n = min(capacity, batchSize, len(ready))
    { ToDispatch = take(n, ready), Skipped = len(ready) - n,
      Reason = if capacity = 0 then "capacity"
               else if n < len(ready) then "batch"
               else if len(ready) > 0 then "ready"
               else "none" }
```

### Readiness Filters (Higher-Order)

```
ReadinessFilter : List PendingBead → List PendingBead

AllReady : ReadinessFilter = id
BlockerAware(readyIDs) : ReadinessFilter = filter(λ b. b.WorkBeadID ∈ readyIDs)
```

### Failure Policies (Higher-Order)

```
FailurePolicy : Nat → FailureAction

NoRetryPolicy : FailurePolicy = λ _. FailureQuarantine
CircuitBreakerPolicy(max) : FailurePolicy = λ n. if n ≥ max then Quarantine else Retry
```

### Dispatch Cycle (Orchestration)

```
DispatchCycle {
  AvailableCapacity : () → Nat × Error
  QueryPending : () → List PendingBead × Error
  Execute : PendingBead → Error
  OnSuccess : PendingBead → Error
  OnFailure : PendingBead × Error → ()
  BatchSize : Nat
  SpawnDelay : Duration
}

Run : DispatchCycle → DispatchReport × Error
Run(cycle) =
  plan ← Plan(cycle)
  for bead ∈ plan.ToDispatch:
    err ← Execute(bead)
    if err = nil:
      err' ← OnSuccess(bead) with retries(2, backoff=500ms)
      if err' ≠ nil: OnFailure(bead, ErrOnSuccessFailed{err'})
      else: report.Dispatched++; sleep(SpawnDelay)
    else: OnFailure(bead, err); report.Failed++
  return report
```

### Key Invariants

1. **Capacity constraint**: `|ToDispatch| ≤ min(capacity, batchSize, |ready|)`
2. **Conservation**: `dispatched + failed + skipped = |ready|`
3. **Reason determinism**: Reason uniquely determined by (capacity, batchSize, |ready|)
4. **OnSuccess atomicity**: OnSuccess failure → Failed++ (prevents double-dispatch)
5. **Retry bound**: OnSuccess retried at most 3 times total
6. **Circuit breaker**: `DispatchFailures ≥ maxFailures → Quarantine`

### Lean 4 Formalization Candidates

1. **PlanDispatch correctness**: Capacity/batch/ready constraints all satisfied
2. **Conservation law**: dispatched + failed + skipped = total ready
3. **Reason classification**: Exhaustive and mutually exclusive
4. **Circuit breaker threshold**: Eventually quarantines after max failures
5. **Retry convergence**: OnSuccess retry loop always terminates

---

## 6. Package: session

**Files**: identity.go, names.go, lifecycle.go, pidtrack.go, registry.go,
startup.go, town.go, stale.go, agent_logging_unix.go, agent_logging_windows.go

### Core Abstraction

The session package manages **agent identity and lifecycle** — how agents are named,
started, stopped, and tracked. It is the largest package in the subsystem.

### Type Algebra

```
Role = Mayor | Deacon | Overseer | Witness | Refinery | Crew | Polecat

AgentIdentity { Role: Role, Rig Name Prefix: String }

PrefixRegistry {
  prefixToRig : Map String String    -- "gt" → "gastown"
  rigToPrefix : Map String String    -- "gastown" → "gt"
}
-- Invariant: prefixToRig and rigToPrefix are inverses

SessionConfig { SessionID WorkDir Role TownRoot RigPath RigName AgentName Command
                Instructions AgentOverride RuntimeConfigDir: String,
                Beacon: BeaconConfig, ExtraEnv: Map String String,
                Theme: Option tmux.Theme,
                WaitForAgent WaitFatal AcceptBypass ReadyDelay AutoRespawn
                RemainOnExit TrackPID VerifySurvived: Bool }

StartResult { RuntimeConfig: RuntimeConfig, RunID: String }

BeaconConfig { Recipient Sender Topic MolID: String,
               IncludePrimeInstruction: Bool, ExcludeWorkInstructions: Bool }

TownSession { Name SessionID: String }
```

### Identity Representations (Isomorphisms)

An agent identity has four equivalent representations:

```
AgentIdentity ↔ Address ↔ SessionName ↔ GTRole ↔ BeaconAddress

Examples for polecat "Toast" in rig "gastown" (prefix "gt"):
  Address:       "gastown/polecats/Toast"
  SessionName:   "gt-polecat-Toast"
  GTRole:        "gastown/polecats/Toast"
  BeaconAddress: "polecat Toast (rig: gastown)"
```

The ParseAddress and ParseSessionName functions provide the inverse maps.

### PrefixRegistry (Bidirectional Map)

```
Register(p, r) : PrefixRegistry → PrefixRegistry
  post: RigForPrefix(p) = r ∧ PrefixForRig(r) = p

RigForPrefix : String → String
PrefixForRig : String → String
-- These form an isomorphism on the registered domain
```

Prefix matching uses longest-prefix-first ordering to prevent ambiguity.

### Session Lifecycle State Machine

```
                    ┌─────────────┐
                    │  Not Exists │
                    └──────┬──────┘
                           │ StartSession
                           ▼
              ┌────────────────────────┐
              │       Running          │
              │  (14-step startup)     │
              │  1. Config resolution  │
              │  2. Settings/plugins   │
              │  3. Command building   │
              │  4. Tmux creation      │
              │  5. Environment setup  │
              │  6. Theme application  │
              │  7. Agent startup      │
              │  8. Auto-respawn       │
              │  9. Dialog acceptance  │
              │  10. Ready delay       │
              │  11. Survival check    │
              │  12. PID tracking      │
              │  13. Agent logging     │
              │  14. Telemetry         │
              └────────┬──┬───────────┘
                       │  │
          StopSession  │  │ KillExistingSession
          (graceful)   │  │ (forced)
                       ▼  ▼
              ┌────────────────────────┐
              │      Terminated        │
              └────────────────────────┘
```

### Stale Detection

```
StaleReasonForTimes(messageTime, sessionCreated) =
  if messageTime < sessionCreated then (true, "message predates session")
  else (false, "")
```

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| ParseAddress | String → AgentIdentity × Error | Partial inverse of Address() |
| ParseSessionName | String → AgentIdentity × Error | Partial inverse of SessionName() |
| StartSession | Tmux × SessionConfig → StartResult × Error | 14-step orchestrated startup |
| StopSession | Tmux × String × Bool → Error | Graceful or forced |
| KillExistingSession | Tmux × String × Bool → Bool × Error | Idempotent kill |
| TrackSessionPID | String × String × Tmux → Error | Write PID file |
| KillTrackedPIDs | String → Nat × List String | Kill surviving processes |
| FormatStartupBeacon | BeaconConfig → String | Pure formatting |
| TownSessions | () → List TownSession | Ordered for shutdown |

### Lean 4 Formalization Candidates

1. **Identity isomorphism**: ParseAddress(id.Address()) = id (round-trip)
2. **Registry bijectivity**: Register then lookup returns registered value
3. **Prefix ordering**: Longest-first prevents ambiguous matches
4. **Stale detection**: Monotone in time (message before session → always stale)
5. **Shutdown ordering**: TownSessions returns a valid topological order for cleanup

---

## 7. Cross-Package Algebraic Structure

### The Work Distribution Pipeline

Though the packages don't import each other directly, they form a conceptual pipeline
orchestrated by cmd/, daemon/, and polecat/:

```
1. Formula.Parse(toml)           → Formula (workflow template)
2. Formula.TopologicalSort()     → ordered steps
3. Formula.ReadySteps(completed) → dispatchable steps
4. Scheduler.PlanDispatch(cap, batch, ready) → DispatchPlan
5. Scheduler.DispatchCycle.Run() → DispatchReport
6. Session.StartSession(config)  → running polecat
7. Convoy.CheckConvoysForIssue() → reactive dispatch on completion
8. Feed.Curator.processLine()    → curated event record
```

### Shared Algebraic Patterns

**1. Readiness Predicates** (appear in formula, convoy, scheduler):
```
formula.ReadySteps(completed)     : needs ⊆ completed
convoy.Ready(issue)               : open ∧ unassigned ∧ slingable ∧ ¬blocked
scheduler.BlockerAware(readyIDs)  : workBeadID ∈ readyIDs
```

All three are conjunctive predicates over different domains but share the pattern:
"an item is ready when all its prerequisites are satisfied."

**2. Fail-Open Semantics** (convoy, scheduler, feed):
- Convoy: Storage errors → assume not blocked
- Scheduler: Missing context → skip bead
- Feed: Parse errors → skip event

**3. DAG Scheduling** (formula, scheduler):
- Formula uses Kahn's algorithm for topological sort
- Scheduler uses capacity-bounded dispatch from ready set
- Both maintain the invariant: dispatched items have all deps satisfied

**4. Idempotency** (convoy, session, scheduler):
- Convoy.CheckConvoysForIssue: safe to call multiple times
- Session.KillExistingSession: safe to call on already-dead sessions
- Scheduler.PlanDispatch: pure function, same inputs → same plan

**5. At-Most-One Semantics** (convoy, scheduler):
- Convoy: feedNextReadyIssue dispatches 0 or 1
- Scheduler: DispatchCycle.Run dispatches ≤ min(capacity, batch, ready)

### Category-Theoretic View

The subsystem can be viewed as a category where:
- **Objects**: States of the work distribution system (beads, sessions, events)
- **Morphisms**: Operations that transform state

Key functors:
```
Formula → Graph     (extracting dependency DAG)
Graph → Schedule    (topological sort / readiness)
Schedule → Dispatch (capacity-bounded selection)
Dispatch → Session  (polecat lifecycle)
Session → Event     (telemetry / feed)
```

The composition `Formula → ... → Event` is the full work distribution pipeline.

### Conservation Laws

1. **Formula**: `|GetAllIDs()| = |Steps| + |Legs| + |Templates| + |Aspects|` (exactly one populated)
2. **Convoy**: Each `feedNextReadyIssue` call changes at most one issue's assignee
3. **Scheduler**: `dispatched + failed + skipped = |ready|`
4. **Feed**: Every visible event is written to feed exactly once (modulo dedup)

---

## 8. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Package | Target | Why |
|---------|--------|-----|
| formula | DAG scheduling (TopologicalSort, ReadySteps) | Classic graph theory, Kahn's algorithm correctness |
| scheduler | Capacity-bounded dispatch (PlanDispatch, Run) | Conservation law, capacity constraints |
| convoy | Readiness predicate and blocking relation | Monotonicity, fail-open safety |

### Medium Priority (simpler structure, fewer invariants)

| Package | Target | Why |
|---------|--------|-----|
| feed | Dedup/aggregation correctness | Temporal window properties |
| session | Identity isomorphisms, registry bijectivity | Algebraic data type round-trips |

### Cross-Cutting Theorems

1. **Readiness is conjunctive**: In all three readiness systems, Ready(x) is a conjunction of independent predicates
2. **Dispatch is monotone**: Closing a dependency can only increase the set of ready items
3. **Pipeline composition preserves ordering**: Topological order from formula is respected by scheduler dispatch
4. **Fail-open never causes false dispatch**: Storage errors lead to skipping, not spurious execution
