# Algebraic Survey: Work Distribution Subsystem

**Subsystem**: Work Distribution
**Packages**: `formula`, `convoy`, `feed`, `scheduler/capacity`, `session`
**Orchestration**: `cmd/sling*`, `cmd/capacity_dispatch`
**Purpose**: Decides WHO does WHAT — formulas define workflow templates, convoys batch
work, feed curates dispatch events, scheduler manages capacity, sessions execute work.
The sling command is the universal dispatcher that orchestrates all these packages.

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

## 7. The Sling Command: Universal Dispatcher

**Files**: sling.go, sling_target.go, sling_dispatch.go, sling_formula.go,
sling_convoy.go, sling_batch.go, sling_schedule.go, sling_validate.go,
sling_helpers.go, sling_idempotency.go, sling_dog.go

The sling command (`gt sling`) is the **single entry point** for all work dispatch
in Gas Town. It unifies bead-to-agent assignment, formula instantiation, convoy
creation, and capacity scheduling into one command with multiple dispatch paths.

### 7.1 Dispatch Mode Algebra

The first-arg determines the dispatch mode via a **priority-ordered type test**:

```
ClassifyFirstArg : String → DispatchMode
ClassifyFirstArg(arg) =
  if verifyBeadExists(arg)    then BeadMode(arg)
  else if verifyFormulaExists(arg) then FormulaMode(arg)
  else if looksLikeBeadID(arg)     then BeadMode(arg)  -- routing fallback
  else Error("not a valid bead or formula")
```

The full dispatch decision tree (within `runSling`) branches on:

```
(numArgs, isDeferred, hasOnFlag, lastArgIsRig, allArgsAreBeadIDs, idType)
```

```
                          runSling(args)
                               │
                    ┌──────────┴──────────┐
                 len > 2              len ≤ 2
                    │                     │
          ┌────────┴────────┐     ┌───────┴───────┐
       lastIsRig         allBeads  len=2         len=1
          │                │         │              │
    ┌─────┴─────┐     autoResolve  ┌─┴─┐      ┌────┴────┐
  deferred   direct     Rig       rig  2-bead  epic/    task
    │           │                   │   auto    convoy
 batchSched batchSling           ┌─┴─┐         │        │
                               def  dir      auto     ┌──┴──┐
                                │    │       detect   def   dir
                             schedule │               │      │
                                   singleSling     sched  self/formula
```

Dispatch modes:

| Mode | Condition | Path |
|------|-----------|------|
| Batch sling | `len > 2 ∧ lastIsRig ∧ ¬deferred` | `runBatchSling` |
| Batch schedule | `len > 2 ∧ lastIsRig ∧ deferred` | `runBatchSchedule` |
| Batch auto-resolve | `len > 2 ∧ allBeadIDs ∧ ¬lastIsRig` | `resolveRigFromBeadIDs → runBatchSling` |
| Deferred single | `len=2 ∧ lastIsRig ∧ deferred` | `scheduleBead` |
| Deferred formula-on-bead | `deferred ∧ --on ∧ lastIsRig` | `scheduleBead` |
| Epic/convoy auto-detect | `len=1 ∧ idType ∈ {epic, convoy}` | `runConvoySling/runEpicSling` |
| Standalone formula | `len≥1 ∧ verifyFormula(arg)` | `runSlingFormula` |
| Single bead | default | `runSling` (inline) |

### 7.2 Target Resolution Algebra

Target resolution converts a user-provided string to `(agentID, pane, workDir)`.
It is a **sum type dispatch** with fallback:

```
TargetSpec = Self | Dog DogName | Rig RigName | ExistingAgent AgentPath

resolveTarget : String → ResolvedTarget × Error
resolveTarget(target) =
  case target of
    "" | "." → resolveSelfTarget()          -- Self
    _ | IsDogTarget(target) → DispatchToDog(...)    -- Dog
    _ | IsRigName(target)   → spawnPolecatForSling(...)  -- Rig (spawn)
    _  → resolveTargetAgent(target)         -- Existing agent
         |> on error, if isPolecatTarget → spawnPolecatForSling  -- Dead polecat fallback
```

The target types form a **lattice of specificity**:

```
       Self (.)
         │
    ┌────┴────┐
   Dog      Rig (auto-spawn)
    │         │
  Named    Named Polecat (rig/polecats/name)
  Dog        │
           ┌─┴──┐
         Crew  Witness/Refinery
```

#### Self-Sling Identity

Self-sling builds the agent ID from the current role environment:

```
resolveSelfTarget : () → (AgentID × Pane × WorkDir) × Error
resolveSelfTarget() =
  let role = GetRole()
  let agentID = case role.Role of
    Mayor    → "mayor/"
    Deacon   → "deacon/"
    Boot     → "deacon/boot"
    Witness  → "{rig}/witness"
    Refinery → "{rig}/refinery"
    Polecat  → "{rig}/polecats/{name}"
    Crew     → "{rig}/crew/{name}"
  (agentID, $TMUX_PANE, role.Home)
```

#### Dog Target Resolution

```
IsDogTarget : String → (DogName × Bool)
IsDogTarget(t) =
  case lowercase(t) of
    "deacon/dogs" | "dog:" → ("", true)     -- Pool dispatch
    "dog:{name}"           → (name, true)   -- Named dispatch
    "deacon/dogs/{name}"   → (name, true)   -- Path dispatch
    _                      → ("", false)
```

Dog dispatch has **pool semantics**: when no name is given, find an idle dog
or auto-create one up to `maxDogPoolSize = 4`.

#### Rig Target Guard Algebra

Before spawning into a rig, three guards are checked:

```
RigDispatchGuard(target, beadID, force) ≡
    ¬isRigParked(target)           -- Rig must not be parked
  ∧ ¬isRigDocked(target)           -- Rig must not be docked
  ∧ (force ∨ prefixMatchesRig(beadID, target))  -- Cross-rig guard
```

The cross-rig guard ensures bead prefixes route to the correct rig:

```
checkCrossRigGuard : BeadID × TargetPath × TownRoot → Error
checkCrossRigGuard(beadID, target, townRoot) =
  let prefix = ExtractPrefix(beadID)
  let beadRig = GetRigNameForPrefix(townRoot, prefix)
  let targetRig = extractRig(target)
  if beadRig ≠ "" ∧ beadRig ≠ targetRig then Error("prefix mismatch")
  else nil
```

### 7.3 Target Validation (Pre-Resolution)

`ValidateTarget` is a **syntactic pre-filter** that catches malformed targets
before `resolveTarget` can trigger side-effects (like polecat spawning):

```
ValidateTarget : String → Error
ValidateTarget(target) =
  if target = "" ∨ target = "." then nil                 -- Self always valid
  if ¬contains(target, "/") then nil                      -- No slash = rig/shortcut
  let parts = split(target, "/")
  reject if ∃ i. parts[i] = ""                           -- Empty segments
  if parts[0] = "deacon" then nil                         -- Dog paths valid
  if parts[0] = "mayor" ∧ len > 1 then Error             -- Mayor has no sub-agents
  if parts[1] ∈ knownRoles then roleSpecificCheck(parts)  -- Role constraints
  if len > 2 ∧ parts[1] ∉ knownRoles then Error           -- Unknown deep path
  else nil                                                -- Shorthand, let resolver handle
```

Role-specific constraints:

| Role | Max Depth | Requires Name? |
|------|-----------|----------------|
| witness | 2 | No (singleton) |
| refinery | 2 | No (singleton) |
| polecats | 3 | Yes |
| crew | 3 | Yes |

### 7.4 Idempotent Sling Detection

```
matchesSlingTarget : Target × Assignee × SelfAgent → Bool
matchesSlingTarget(target, assignee, self) =
  let a = normalize(assignee)
  if a = "" then false
  case target of
    "" | "." → normalize(self) = a                        -- Self-match
    t       → normalize(t) = a                            -- Exact match
            ∨ (¬contains(t, "/") ∧ startsWith(a, t+"/polecats/"))  -- Rig subsumption

  -- NOT matched (intentionally):
  --   Two-segment shorthand (ambiguous: crew vs polecat)
  --   Pool targets like "deacon/dogs" (means "any idle", not "keep current")
```

The idempotency check runs **after** dead-agent detection but **before** the
"already hooked" error, so a dead agent with matching target re-slings rather
than no-oping.

### 7.5 Single-Bead Dispatch (runSling)

The single-bead path is the most complex, handling all target types and
formula-on-bead scenarios. The 12-step execution for `executeSling`:

```
executeSling : SlingParams → SlingResult × Error
executeSling(p) =
  0. Guard: rig not parked/docked
  1. Fetch bead info; guard closed/tombstone/deferred
  2. Auto-force if hooked agent is dead; send LIFECYCLE:Shutdown if force-stealing
  3. Burn stale molecules (if formula + force/stale)
  4. Spawn polecat: spawnPolecatForSling(rig, opts)
  5. Auto-convoy: createAutoConvoy(beadID, title, owned, merge) if ¬NoConvoy
  6. Cook formula: CookFormula(name, workDir, townRoot) if ¬SkipCook
  7. Instantiate formula on bead: wisp + bond
  8. Hook bead with retry: hookBeadWithRetry(beadToHook, agent, hookDir)
  9. Log sling event to feed
  10. Update agent hook_bead state
  11. Store fields in bead (dispatcher, args, attached_molecule, no_merge, mode)
  12. Start polecat session: spawnInfo.StartSession()
```

**Rollback semantics**: If any step after spawn (4) fails, the spawned polecat
is cleaned up via `rollbackSlingArtifactsFn` or `cleanupSpawnedPolecat`, which
removes the worktree, agent bead, git branch, and auto-convoy.

### 7.6 Formula Instantiation

Two formula dispatch paths exist:

**Standalone formula** (`runSlingFormula`):
```
1. resolveTarget(target)        → agent, pane, workDir
2. CookFormula(name)            → ensure proto exists
3. bd mol wisp <formula> --json → create wisp instance (ephemeral)
4. hookBeadWithRetry(wispID)    → attach wisp to hook
5. storeFieldsInBead(wispID)    → dispatcher, args, attached_formula
6. StartSession / NudgePane     → activate agent
```

**Formula-on-bead** (`InstantiateFormulaOnBead` via `executeSling`):
```
1. CookFormula(name)
2. bd mol wisp <formula> --on <beadID> --json → wisp bonded to work bead
3. beadToHook = wispRootID (not the original bead)
4. attachedMoleculeID stored in bead fields
```

The formula resolution function:

```
resolveFormula : ExplicitFlag × HookRawBead → FormulaName
resolveFormula(explicit, raw) =
  if raw  then ""                    -- No formula
  if explicit ≠ "" then explicit     -- User override
  else "mol-polecat-work"            -- Default
```

### 7.7 Convoy Tracking Algebra

#### Auto-Convoy Creation

```
createAutoConvoy : BeadID × Title × Owned × MergeStrategy → ConvoyID × Error
createAutoConvoy(beadID, title, owned, merge) =
  guard ¬IsFlagLikeTitle(title)                         -- Safety
  let convoyID = "hq-cv-" ++ randomShortID()            -- 5-char random
  let convoyTitle = "Work: " ++ title
  let description = SetConvoyFields(prose, {merge})
  bd create --type=convoy --id=convoyID ...              -- Create convoy bead
  bd dep add convoyID beadID --type=tracks               -- Track relation
  if tracksFailed then bd close convoyID                 -- Cleanup orphan
```

#### Convoy Lookup (Triple Strategy)

```
isTrackedByConvoy : BeadID → ConvoyID
isTrackedByConvoy(beadID) =
  -- Strategy 1: Dependency lookup (authoritative when cross-rig works)
  deps ← bd dep list beadID --direction=up --type=tracks
  if ∃ d ∈ deps. d.IssueType = "convoy" ∧ d.Status = "open" then d.ID

  -- Strategy 2: Description pattern matching (robust fallback)
  convoys ← bd list --type=convoy --status=open
  if ∃ c ∈ convoys. c.Description contains "tracking {beadID}" then c.ID

  -- Strategy 3: Forward dependency scan (manual convoy fallback)
  for c ∈ convoys:
    tracked ← bd dep list c.ID --direction=down --type=tracks
    if beadID ∈ tracked.IDs ∨ "external:*:{beadID}" ∈ tracked.IDs then c.ID
```

#### Batch Convoy

```
createBatchConvoy : List BeadID × RigName × Owned × Merge → (ConvoyID × List BeadID) × Error
createBatchConvoy(beadIDs, rig, owned, merge) =
  let convoyTitle = "Batch: {n} beads to {rig}"
  create convoy bead
  for beadID ∈ beadIDs:
    result ← bd dep add convoyID beadID --type=tracks
    if success then tracked ← tracked ++ [beadID]       -- Partial success OK
  return (convoyID, tracked)
```

#### Convoy Info Model

```
ConvoyInfo { ID: String, Owned: Bool, MergeStrategy: String }

IsOwnedDirect(c) ≡ c ≠ nil ∧ c.Owned ∧ c.MergeStrategy = "direct"

MergeStrategy = "direct" | "mr" | "local" | ""
-- "direct": push branch to main (skip refinery)
-- "mr":     merge queue via refinery (default)
-- "local":  keep on feature branch
```

Two lookup methods, with **issue-embedded fields as primary**:
1. `getConvoyInfoFromIssue(issueID)` — reads convoy_id/merge_strategy from bead description
2. `getConvoyInfoForIssue(issueID)` — dependency-based lookup (cross-rig, slower)

### 7.8 Batch Dispatch

```
runBatchSling : List BeadID × RigName × BeadsDir → Error
runBatchSling(beadIDs, rig, _) =
  -- Phase 1: Validate all beads exist (fail-fast before any spawn)
  for id ∈ beadIDs: verifyBeadExists(id) or Error

  -- Phase 2: Cross-rig guard (all beads must match target rig)
  for id ∈ beadIDs: checkCrossRigGuard(id, rig) or Error

  -- Phase 3: Pre-cook formula once (batch optimization)
  formulaCooked ← CookFormula(formula, workDir, townRoot) succeeds

  -- Phase 4: Sequential dispatch with throttling
  for (i, beadID) ∈ enumerate(beadIDs):
    if maxConcurrent > 0 ∧ active ≥ maxConcurrent then wait(cooldown)
    result ← executeSling({beadID, rig, formulaCooked, ...})
    sleep(2s) between spawns                             -- Dolt contention avoidance

  -- Phase 5: Wake rig agents (once, post-loop)
  wakeRigAgents(rig)
```

**Batch auto-resolve** (no explicit rig):

```
resolveRigFromBeadIDs : List BeadID × TownRoot → RigName × Error
resolveRigFromBeadIDs(ids, root) =
  for id ∈ ids:
    let prefix = ExtractPrefix(id)
    let rig = GetRigNameForPrefix(root, prefix)
    guard prefix ≠ "" ∧ rig ≠ ""                        -- Must resolve
    guard rig = resolvedRig ∨ resolvedRig = ""           -- Must all agree
  return resolvedRig
```

**Invariant**: All beads in a batch must resolve to the **same** rig.

### 7.9 Deferred Dispatch (Scheduler Integration)

When `scheduler.max_polecats > 0`, sling enqueues rather than dispatching directly.

```
shouldDeferDispatch : () → Bool × Error
shouldDeferDispatch() =
  let settings = LoadTownSettings()
  let maxPol = settings.Scheduler.GetMaxPolecats()
  maxPol > 0

scheduleBead : BeadID × RigName × ScheduleOptions → Error
scheduleBead(beadID, rig, opts) =
  -- Idempotency: check for existing open sling context
  if FindOpenSlingContext(beadID) ≠ nil then no-op

  -- Guard: status, force, formula
  guard status ∉ {closed, tombstone, deferred}
  guard ¬(hooked ∨ in_progress) ∨ force

  -- Create sling context bead (atomic, no two-step write)
  fields = SlingContextFields{version=1, workBeadID, targetRig, formula, args, ...}
  ctxBead ← CreateSlingContext(title, beadID, fields)

  -- Auto-convoy (same as direct path)
  createAutoConvoy(beadID, ...) unless --no-convoy
```

**Sling context lifecycle**:

```
              ┌───────────┐
              │   open    │ ← CreateSlingContext
              └─────┬─────┘
                    │ dispatch cycle
              ┌─────┴─────┐
              │ dispatched │ ← CloseSlingContext("dispatched")
              └───────────┘

  Error paths:
    open → circuit-broken   (DispatchFailures ≥ 3)
    open → stale-work-bead  (work bead hooked/closed/tombstone)
    open → invalid-context  (unparseable description)
```

**Sling context TTL**: Contexts older than 30 minutes are ignored by
`areScheduled()` to prevent orphaned contexts from permanently blocking tasks.

### 7.10 Capacity Dispatch Cycle (Orchestrator)

The daemon/CLI dispatch loop wires up the generic `DispatchCycle`:

```
dispatchScheduledWork : TownRoot × Actor × BatchOverride × DryRun → Nat × Error
dispatchScheduledWork(townRoot, actor, batch, dry) =
  -- Serialize: flock("scheduler-dispatch.lock")
  -- Check: scheduler not paused, max_polecats > 0

  -- Clean stale contexts BEFORE querying ready
  cleanupStaleContexts(townRoot)

  -- Wire dispatch cycle callbacks
  cycle = DispatchCycle {
    AvailableCapacity = maxPolecats - countActivePolecats()
    QueryPending      = getReadySlingContexts(townRoot)
    Execute           = dispatchSingleBead → executeSling
    OnSuccess         = CloseSlingContext("dispatched")
    OnFailure         = recordDispatchFailure (increment counter, circuit-break at 3)
    BatchSize, SpawnDelay
  }

  report ← cycle.Run()
  wakeRigAgents(each successful rig)
  updateSchedulerState(report.Dispatched)
```

**Ready context query** (`getReadySlingContexts`):

```
getReadySlingContexts : TownRoot → List PendingBead × Error
getReadySlingContexts(root) =
  1. contexts ← ListOpenSlingContexts() from HQ       -- Authoritative
  2. readyIDs ← bd ready across all rig dirs           -- Cross-rig readiness
  3. sort(contexts, by=EnqueuedAt ASC, ID ASC)         -- Deterministic order
  4. deduplicate by WorkBeadID (oldest context wins)
  5. filter: fields ≠ nil ∧ failures < 3 ∧ workBead ∈ readyIDs
```

### 7.11 Concurrency Control

The sling command uses a **three-layer locking strategy**:

```
Layer 1: Per-bead flock     — tryAcquireSlingBeadLock(townRoot, beadID)
  Prevents concurrent sling of the same bead (TOCTOU races).
  Both runSling and executeSling acquire this lock independently.

Layer 2: Dispatch lock      — flock("scheduler-dispatch.lock")
  Prevents concurrent dispatch cycles from the daemon and CLI.

Layer 3: Dolt auto-commit   — BD_DOLT_AUTO_COMMIT=off during sling
  Prevents manifest contention under concurrent load.
  Individual BdCmd calls use WithAutoCommit() for critical writes
  (convoy creation, dep adds) that must persist.
```

### 7.12 Safety Invariants

**1. Status guard (non-bypassable)**:
```
¬Slingable(bead) ≡ bead.Status ∈ {closed, tombstone}
-- Cannot be overridden with --force. Must reopen first.
```

**2. Deferred guard (force-gated)**:
```
DeferredGuard(bead, force) ≡ isDeferredBead(bead) ∧ ¬force
-- Prevents deferred work from consuming polecat slots.
```

**3. Dead-agent auto-force**:
```
AutoForce(bead) ≡ bead.Status ∈ {hooked, in_progress}
                 ∧ bead.Assignee ≠ ""
                 ∧ isHookedAgentDead(bead.Assignee)
-- Automatically force-slings when the current assignee's session is dead.
-- Sends LIFECYCLE:Shutdown to witness for zombie cleanup.
```

**4. Explicit force vs auto-force distinction**:
```
-- executeSling tracks explicitForce separately from params.Force.
-- Dead-agent auto-force sets params.Force=true but NOT explicitForce.
-- DeferredGuard checks explicitForce, preventing auto-force from bypassing it.
```

**5. Flag-like title guard**:
```
¬IsFlagLikeTitle(title)
-- Rejects garbage beads created by flag-parsing bugs to prevent dispatch loops.
```

**6. Rollback guarantee**:
```
∀ step ∈ {5..12}: step fails → rollbackSlingArtifacts(spawnInfo, beadID, workDir, convoyID)
-- Cleans up: worktree, agent bead, git branch, auto-convoy.
-- No orphaned polecats from failed dispatch.
```

### 7.13 Lean 4 Formalization Candidates

1. **Target resolution exhaustiveness**: Every valid target string resolves to
   exactly one of {Self, Dog, Rig, ExistingAgent}
2. **Idempotent sling**: `matchesSlingTarget(t, a, s) → sling(t) is no-op`
3. **Cross-rig guard correctness**: `checkCrossRigGuard passes ↔ prefixOf(bead) routes to targetRig`
4. **Batch rig homogeneity**: `resolveRigFromBeadIDs succeeds → ∀ i j. rigOf(ids[i]) = rigOf(ids[j])`
5. **Sling context deduplication**: After dedup, at most one context per WorkBeadID
6. **Rollback completeness**: If executeSling fails after spawn, no artifacts remain
7. **Auto-force does not bypass deferred**: `AutoForce ∧ ¬explicitForce → DeferredGuard still holds`
8. **ValidateTarget is conservative**: `ValidateTarget(t) = nil → resolveTarget(t) may still fail,
   but ValidateTarget(t) = Error → resolveTarget(t) would have produced incorrect side-effects`
9. **Convoy lookup convergence**: All three lookup strategies return the same convoy
   (or a superset) when cross-rig routing is healthy
10. **Dispatch conservation**: In batch sling, `succeeded + failed = |beadIDs|`

---

## 8. Cross-Package Algebraic Structure

### The Work Distribution Pipeline

The five internal packages share no direct imports but are composed into a pipeline
by the sling command (§7) and daemon orchestration:

```
1. Sling.runSling(args)                  → dispatch mode selection
2. Sling.resolveTarget(target)           → agent + pane + workDir
3. Formula.Parse(toml)                   → Formula (workflow template)
4. Formula.TopologicalSort()             → ordered steps
5. Formula.ReadySteps(completed)         → dispatchable steps
6. Sling.executeSling(params)            → spawn + cook + wisp + hook
7. Scheduler.PlanDispatch(cap, batch, ready) → DispatchPlan
8. Scheduler.DispatchCycle.Run()         → DispatchReport
9. Session.StartSession(config)          → running polecat
10. Convoy.CheckConvoysForIssue()        → reactive dispatch on completion
11. Feed.Curator.processLine()           → curated event record
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

## 9. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Package | Target | Why |
|---------|--------|-----|
| formula | DAG scheduling (TopologicalSort, ReadySteps) | Classic graph theory, Kahn's algorithm correctness |
| scheduler | Capacity-bounded dispatch (PlanDispatch, Run) | Conservation law, capacity constraints |
| convoy | Readiness predicate and blocking relation | Monotonicity, fail-open safety |
| sling | Target resolution + rollback completeness | Sum-type exhaustiveness, transactional safety |

### Medium Priority (simpler structure, fewer invariants)

| Package | Target | Why |
|---------|--------|-----|
| sling | Idempotent sling detection | Equivalence classes on target strings |
| sling | Cross-rig guard correctness | Prefix-to-rig bijection preservation |
| feed | Dedup/aggregation correctness | Temporal window properties |
| session | Identity isomorphisms, registry bijectivity | Algebraic data type round-trips |

### Cross-Cutting Theorems

1. **Readiness is conjunctive**: In all three readiness systems, Ready(x) is a conjunction of independent predicates
2. **Dispatch is monotone**: Closing a dependency can only increase the set of ready items
3. **Pipeline composition preserves ordering**: Topological order from formula is respected by scheduler dispatch
4. **Fail-open never causes false dispatch**: Storage errors lead to skipping, not spurious execution
5. **Target resolution is total on valid inputs**: Every syntactically valid target (per ValidateTarget) resolves to exactly one dispatch path
6. **Rollback is complete**: Failed dispatch after polecat spawn leaves no orphaned artifacts
7. **Deferred dispatch subsumes direct**: scheduleBead stores the same SlingParams that executeSling consumes (via ReconstructFromContext), ensuring behavioral equivalence
8. **Auto-force is safe**: Dead-agent auto-force enables re-sling without bypassing deferred/status guards
