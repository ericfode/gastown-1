# Algebraic Survey: Work Items (Beads)

**Subsystem**: Work Items (Beads)
**Packages**: `types`, `storage`, `storage/dolt`, `beads`, `cmd/bd`
**Purpose**: Defines WHAT work exists — issue types, statuses, priorities, dependencies,
the claim/release/close lifecycle, convoy tracking, and wisp (ephemeral) scaffolding.

## 1. Subsystem Overview

The Work Items subsystem is the data model and persistence layer for all tracked work
in Gas Town. Every bead (issue) has a type, status, priority, and optional dependency
edges that form a DAG. The `bd` CLI provides CRUD operations with transactional
guarantees backed by Dolt (git-for-data).

```
Issue (data) → Dependencies (DAG) → Ready Work (predicate) → Dispatch (scheduler)
                     ↓
              Convoy (tracking)
              Wisp (ephemeral)
```

### Package Dependency Graph (within subsystem)

```
types      ←── (no internal deps, leaf package — all type definitions)
storage    ←── types (Storage interface, Transaction interface)
storage/dolt ←── types, storage (Dolt-backed implementation)
beads      ←── types, storage (context resolution, redirect handling)
cmd/bd     ←── types, storage, beads (CLI commands, validation)
```

**Key finding**: The `types` package is a pure leaf — it defines all domain types with
zero internal imports. The `storage` package defines the interface; `storage/dolt` is the
sole implementation. This is a **layered architecture** with strict dependency direction.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| types | Every package in the system (universal domain language) |
| storage | cmd/bd, internal/beads, integrations, tests |
| storage/dolt | cmd/bd (via storage interface), tests |
| beads | cmd/bd (context resolution), gastown (convoy operations) |
| cmd/bd | End users, agent automation |

---

## 2. Package: types (Core Type Algebra)

**Files**: types.go, id_generator.go

### The Issue Type (Product Type)

The `Issue` struct is the central product type — a large record with ~60 fields
organized into semantic groups:

```
Issue {
  -- Identification
  ID ContentHash: String,

  -- Content
  Title Description Design AcceptanceCriteria Notes SpecID: String,

  -- Workflow
  Status: Status, Priority: Int, IssueType: IssueType,

  -- Assignment
  Assignee Owner: String, EstimatedMinutes: Option Int,

  -- Timestamps
  CreatedAt UpdatedAt: Time, CreatedBy: String,
  ClosedAt: Option Time, CloseReason ClosedBySession: String,

  -- Time-Based Scheduling
  DueAt DeferUntil: Option Time,

  -- External Integration
  ExternalRef: Option String, SourceSystem: String,

  -- Custom Metadata
  Metadata: JSON,

  -- Relational (for export/import)
  Labels: List String, Dependencies: List Dependency, Comments: List Comment,

  -- Messaging
  Sender: String, Ephemeral: Bool, WispType: WispType,

  -- Context Markers
  Pinned IsTemplate: Bool,

  -- Bonding (compound molecules)
  BondedFrom: List BondRef,

  -- HOP Entity Tracking
  Creator: Option EntityRef, Validations: List Validation,
  QualityScore: Option Float32, Crystallizes: Bool,

  -- Gate (async coordination)
  AwaitType AwaitID: String, Timeout: Duration, Waiters: List String,

  -- Slot (exclusive access)
  Holder: String,

  -- Agent Identity
  HookBead RoleBead: String, AgentState: AgentState,
  LastActivity: Option Time, RoleType Rig: String,

  -- Molecule Coordination
  MolType: MolType, WorkType: WorkType,

  -- Source Tracing
  SourceFormula SourceLocation: String,

  -- Event Fields
  EventKind Actor Target Payload: String
}
```

### Status (Sum Type)

```
Status = Open | InProgress | Blocked | Deferred | Closed | Pinned | Hooked
```

| Status | Semantics |
|--------|-----------|
| Open | Default, unstarted work |
| InProgress | Currently being worked on (auto-set on claim) |
| Blocked | Cannot proceed (dependency blocker or external) |
| Deferred | Deliberately on ice; hidden from ready until `defer_until` |
| Closed | Completed or rejected (terminal for most flows) |
| Pinned | Persistent context marker; stays open indefinitely |
| Hooked | Work attached to agent's hook (GUPP coordination) |

Custom statuses are supported via configuration: `bd config set status.custom "review,testing"`.

### Issue Type (Sum Type)

```
IssueType = Bug | Feature | Task | Epic | Chore | Decision | Message | Molecule | Event
```

| Type | Category | Purpose |
|------|----------|---------|
| Bug | Core work | Something broken |
| Feature | Core work | New functionality |
| Task | Core work | Work item (default) |
| Epic | Core work | Large feature with subtasks |
| Chore | Core work | Maintenance (deps, tooling) |
| Decision | Core work | Architectural decision record |
| Message | Core work | Inter-agent communication |
| Molecule | Internal | Swarm coordination |
| Event | Internal | Operational state changes |

Normalization aliases: `"enhancement" → Feature`, `"adr" → Decision`.

### Priority (Total Order)

```
Priority : Fin 5    -- {0, 1, 2, 3, 4}

P0 = 0  (Critical)     -- security, data loss, broken builds
P1 = 1  (High)         -- major features, important bugs
P2 = 2  (Medium)       -- default, nice-to-have
P3 = 3  (Low)          -- polish, optimization
P4 = 4  (Backlog)      -- future ideas

Ordering: P0 < P1 < P2 < P3 < P4  (lower number = higher priority)
Sort key: (priority ASC, created_at DESC, id ASC)
```

Priority is a total order with deterministic tiebreaking. No inheritance from
parent to child. Default is P2 on creation.

### Agent State (Sum Type)

```
AgentState = Idle | Spawning | Running | Working | Stuck | Done | Stopped | Dead
```

State machine for agent lifecycle tracking within bead metadata.

### Molecule Type (Sum Type)

```
MolType = Swarm | Patrol | Work
```

| Type | Semantics |
|------|-----------|
| Swarm | Coordinated multi-polecat work |
| Patrol | Recurring operational work (Witness, Deacon) |
| Work | Regular polecat assignment (default) |

### Work Type (Sum Type)

```
WorkType = Mutex | OpenCompetition
```

| Type | Semantics |
|------|-----------|
| Mutex | One worker, exclusive assignment (default) |
| OpenCompetition | Many submit, buyer picks best |

### Wisp Type (Sum Type with TTL Stratification)

```
WispType = Heartbeat | Ping | Patrol | GCReport | Recovery | Error | Escalation

TTL : WispType → Duration
TTL(Heartbeat) = TTL(Ping) = 6h          -- high-churn, low forensic value
TTL(Patrol) = TTL(GCReport) = 24h         -- operational state
TTL(Recovery) = TTL(Error) = TTL(Escalation) = 7d  -- significant events
```

### ID Generation

```
GenerateHashID : (prefix, title, description, created, workspaceID) → String
  -- Deterministic SHA256-based, 6-8 char hex suffix
  -- Progressive extension on collision

GenerateChildID : (parentID, childNumber) → String
  -- Format: parent.N (e.g., "gt-af78e9.1", "gt-af78e9.1.2")

MaxHierarchyDepth = 3

ParseHierarchicalID : String → (rootID, parentID, depth)
```

### Content Hash

```
ComputeContentHash : Issue → String
  -- Deterministic SHA256 of canonical content fields
  -- Used for: collision detection, content equivalence, federation trust
```

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| SetDefaults | Issue → Issue | Idempotent (Status=Open, Type=Task) |
| Validate | Issue → Error | Checks all field constraints |
| ValidateWithCustom | Issue × CustomStatuses × CustomTypes → Error | Extension point |
| ValidateForImport | Issue → Error | Federation trust model |
| ComputeContentHash | Issue → String | Deterministic, content-addressed |
| Normalize | IssueType → IssueType | Alias resolution (enhancement→feature) |
| AffectsReadyWork | DependencyType → Bool | Predicate over 4 blocking types |

### Lean 4 Formalization Candidates

1. **Priority total order**: (Fin 5, ≤) is a decidable total order
2. **Status transition well-formedness**: Claim always produces InProgress
3. **ID generation determinism**: Same inputs → same hash ID
4. **Hierarchy depth bound**: GenerateChildID respects MaxHierarchyDepth = 3
5. **Content hash stability**: ComputeContentHash is a pure function of content fields

---

## 3. Dependency DAG

**Files**: types/types.go (type definitions), storage/dolt/dependencies.go (CRUD + cycle
detection), storage/dolt/queries.go (blocked computation)

### Dependency Type (Sum Type)

```
DependencyType =
  -- Workflow (affect ready work)
  | Blocks              -- A must complete before B can start
  | ParentChild         -- Hierarchical relationship
  | ConditionalBlocks   -- B runs only if A fails
  | WaitsFor            -- Fanout gate: wait for dynamic children

  -- Association (non-blocking)
  | Related | DiscoveredFrom

  -- Graph links (non-blocking)
  | RepliesTo | RelatesTo | Duplicates | Supersedes

  -- Entity links (HOP)
  | AuthoredBy | AssignedTo | ApprovedBy | Attests

  -- Convoy (non-blocking tracking)
  | Tracks

  -- Reference (non-blocking)
  | Until | CausedBy | Validates

  -- Delegation
  | DelegatedFrom
```

### Blocking Predicate

The set of blocking dependency types:

```
blockingTypes : Set DependencyType = {Blocks, ParentChild, ConditionalBlocks, WaitsFor}

AffectsReadyWork(t) ≡ t ∈ blockingTypes
```

### Dependency Structure

```
Dependency {
  IssueID     : String,        -- the issue that depends
  DependsOnID : String,        -- the issue depended upon
  Type        : DependencyType,
  CreatedAt   : Time,
  CreatedBy   : String,
  Metadata    : String,        -- type-specific JSON
  ThreadID    : String         -- conversation root (for RepliesTo)
}
```

### WaitsFor Gate Semantics

```
WaitsForMeta { Gate: String, SpawnerID: String }

Gate = AllChildren | AnyChildren

Blocked_WaitsFor(issue, gate, children) ≡
  gate = AllChildren  → ∃ child ∈ children. child.Status ∉ {Closed, Pinned}
  gate = AnyChildren  → children ≠ ∅ ∧ (∀ child ∈ children. child.Status ≠ Closed)
                         ∧ (∃ child ∈ children. child.Status ∉ {Closed, Pinned})
```

AllChildren blocks until ALL children close. AnyChildren blocks until at least
one child closes (unless no children exist, in which case it's unblocked).

### Blocking Resolution Algorithm: `computeBlockedIDs()`

The core algorithm for determining which issues are blocked:

```
computeBlockedIDs(includeWisps) → Set IssueID:

1. activeIDs ← { i.ID | i ∈ issues ∪ (wisps if includeWisps),
                         i.Status ∉ {Closed, Pinned} }

2. edges ← { (e.IssueID, e.DependsOnID, e.Type, e.Metadata)
            | e ∈ dependencies ∪ (wisp_dependencies if includeWisps),
              e.Type ∈ {Blocks, WaitsFor, ConditionalBlocks} }

3. blockedSet ← {}
   waitsForEdges ← []
   for (issueID, depID, type, meta) ∈ edges:
     if type ∈ {Blocks, ConditionalBlocks}:
       if issueID ∈ activeIDs ∧ depID ∈ activeIDs:
         blockedSet ← blockedSet ∪ {issueID}
     if type = WaitsFor:
       waitsForEdges.append((issueID, depID, meta))

4. for (issueID, spawnerID, meta) ∈ waitsForEdges:
     if issueID ∉ activeIDs: continue
     children ← { c | c has parent-child dep from spawnerID }
     gate ← parseGate(meta)  -- AllChildren or AnyChildren
     if Blocked_WaitsFor(issueID, gate, children):
       blockedSet ← blockedSet ∪ {issueID}

5. return blockedSet
```

### Cycle Detection

```
hasCycle(issueID, dependsOnID) : Bool
  -- Recursive CTE from dependsOnID following Blocks edges
  -- Returns true if issueID is reachable from dependsOnID
  -- Depth bounded at 100 for safety
  -- Spans both dependencies and wisp_dependencies tables
```

Only `Blocks` edges are checked for cycles. Other dependency types may form
cycles freely (e.g., RepliesTo conversation threads).

### Cross-Type Blocking Constraint

```
ValidBlocking(issue, blocker) ≡
  (issue.Type = Task ∧ blocker.Type = Task) ∨
  (issue.Type = Epic ∧ blocker.Type = Epic) ∨
  ... (same-type only)
```

Prevents task→epic blocking deadlocks.

### Caching

```
BlockedIDsCache {
  valid : Bool,
  ids   : List String,
  idMap : Map String Bool,    -- O(1) membership
  includesWisps : Bool,
  mu    : Mutex
}

-- Invariant: cache with includesWisps=true satisfies non-wisp queries
-- Invalidated on: dep add/remove, issue delete, child creation, status change
```

### Ready Work Predicate

```
Ready(issue) ≡ issue.Status ∈ {Open, InProgress, Blocked, Hooked}
             ∧ issue.Status ∉ {Closed, Pinned}
             ∧ issue.ID ∉ computeBlockedIDs()
             ∧ (issue.DeferUntil = nil ∨ issue.DeferUntil ≤ now)
             ∧ ¬DeferredParent(issue)
```

Where `DeferredParent(issue)` checks if any ancestor has a future `DeferUntil`.

### Newly Unblocked (on close)

```
NewlyUnblockedByClose(closedID) =
  { i | i ∈ issues,
        i.Status ∈ {Open, Blocked},
        ∃ dep(i, closedID, Blocks),
        ¬∃ dep(i, other, Blocks) where other.Status ∉ {Closed, Pinned} }
```

Issues that were blocked solely by the now-closed issue.

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| AddDependency | Dependency → Error | Validates: no cycles (Blocks), same-type blocking |
| RemoveDependency | (IssueID, DependsOnID) → Error | Invalidates blocked cache |
| computeBlockedIDs | Bool → Set String | Cached, monotone in closed deps |
| IsBlocked | IssueID → Bool | O(1) via cache map lookup |
| GetDependencyTree | (ID, maxDepth, showAllPaths, reverse) → List TreeNode | BFS/DFS traversal |
| NewlyUnblockedByClose | IssueID → List Issue | Batch query with queryBatchSize=50 |
| hasCycle | (IssueID, DependsOnID) → Bool | Recursive CTE, depth ≤ 100 |

### Lean 4 Formalization Candidates

1. **Blocking is monotone**: Closing a dependency can only shrink blockedSet
2. **Cycle detection soundness**: hasCycle returns true iff adding edge creates cycle
3. **WaitsFor gate completeness**: AllChildren blocks iff ∃ active child
4. **Ready work anti-monotonicity**: Adding a Blocks edge can only shrink ready set
5. **Cache coherence**: Invalidation after every mutation → cache always consistent
6. **Cross-type blocking prevents deadlock**: Same-type constraint + acyclicity → DAG

---

## 4. Claim/Release/Close Lifecycle

**Files**: storage/dolt/issues.go (atomic operations), cmd/bd/close.go (CLI guards),
cmd/bd/show_unit_helpers.go (validators)

### State Machine

```
                    ┌──────────┐
                    │   Open   │ ← SetDefaults()
                    └────┬─────┘
                         │ claim (atomic CAS)
                         ▼
              ┌────────────────────┐
              │    InProgress      │ ← auto-set by claim
              └───┬──────────┬─────┘
                  │          │ update --status=blocked
                  │          ▼
                  │   ┌──────────┐
                  │   │ Blocked  │
                  │   └──────┬───┘
                  │          │ (unblock via dep close)
                  │          │
                  ▼          ▼
              ┌────────────────────┐
              │      Closed        │ ← close (with guards)
              └────────────────────┘

Special states (orthogonal):
  Pinned   — persistent, protected from modification unless --force
  Hooked   — attached to agent's hook, protected unless --force
  Deferred — hidden from ready until defer_until
```

### Claim (Atomic Compare-and-Swap)

```
ClaimIssue(issueID, actor) → Error:
  -- Precondition: issue.Assignee = "" (empty)
  -- Postcondition: issue.Assignee = actor ∧ issue.Status = InProgress
  -- Failure: ErrAlreadyClaimed{currentAssignee}
  -- Implementation: conditional UPDATE ... WHERE assignee = '' OR assignee IS NULL
  -- Side effects: invalidates blocked cache, records "claimed" event
```

Claim is a CAS operation — it atomically checks that no one else has claimed the
issue and sets the assignee + status in a single transaction. This prevents
double-dispatch.

### Release

There is no explicit release operation. Release is modeled as:
```
Release(issueID) ≡ Update(issueID, {assignee: ""})
```

Clearing the assignee does NOT automatically change status. The issue remains
in its current status until explicitly transitioned.

### Close (Guarded Transition)

```
CloseIssue(issueID, reason, actor, session) → Error:

  Pre-close validation chain:
    1. NotTemplate()          -- templates are read-only
    2. NotPinned(force)       -- pinned issues protected unless --force
    3. ¬IsBlocked(issueID)    -- cannot close with open blockers (unless --force)
    4. GatesSatisfied(issueID) -- async coordination gates must be met

  Atomic effects:
    - status ← Closed
    - closed_at ← now (UTC)
    - close_reason ← reason (default: "Closed")
    - closed_by_session ← session (optional tracking)
    - Record EventClosed

  Cascading effects:
    - Auto-close parent molecule if all sibling steps are closed
    - Compute NewlyUnblockedByClose → return for UI feedback
```

### Failure Close Detection

```
FailureCloseKeywords = {"failed", "rejected", "wontfix", "won't fix",
                        "canceled", "cancelled", "abandoned", "blocked",
                        "error", "timeout", "aborted"}

IsFailureClose(reason) ≡ ∃ kw ∈ FailureCloseKeywords. kw ⊆ lowercase(reason)
```

Used by `ConditionalBlocks` to determine if a blocking issue failed (which
unblocks the conditional dependent).

### Reopen

```
ReopenIssue(issueID) → Error:
  Precondition: issue.Status = Closed
  Postcondition: issue.Status = Open, issue.ClosedAt = nil, issue.CloseReason = ""
  Side effects: records EventReopened
```

### Pre-Operation Validators

```
Validator = Issue → Error

NotTemplate : Validator     -- blocks all modifications to templates
NotPinned(force) : Validator -- blocks unless --force
NotClosed : Validator       -- blocks modifications to closed issues
NotHooked(force) : Validator -- blocks unless --force
Exists : Validator          -- issue must be non-nil
HasStatus(ss) : Validator   -- issue must have one of allowed statuses
HasType(ts) : Validator     -- issue must have one of allowed types

Chains:
  ForUpdate = Exists ∘ NotTemplate
  ForClose  = Exists ∘ NotTemplate ∘ NotPinned(force)
  ForDelete = Exists ∘ NotTemplate
  ForReopen = Exists ∘ NotTemplate ∘ HasStatus(Closed)
```

### Key Invariants

1. **Claim atomicity**: ClaimIssue is a CAS — no two actors can claim simultaneously
2. **Close guards**: Cannot close with open blocking deps (without --force)
3. **ClosedAt consistency**: `closed_at` is present iff `status = Closed`
4. **Auto-close propagation**: Molecule auto-closes when all steps close
5. **Event audit**: Every state transition records an Event with actor, old/new values
6. **Template immutability**: Templates cannot be modified through any operation

### Lean 4 Formalization Candidates

1. **CAS correctness**: ClaimIssue succeeds for at most one concurrent caller
2. **Close guard soundness**: Close with open blockers → Error (unless forced)
3. **ClosedAt invariant**: status = Closed ↔ closed_at ≠ nil
4. **Auto-close completeness**: All steps closed → parent molecule closes
5. **Validator chain composition**: ForClose = Exists ∧ NotTemplate ∧ NotPinned

---

## 5. Convoy Tracking

**Files**: types/types.go (DepTracks), storage/dolt/dependencies.go

### Core Abstraction

Convoy tracking uses the `tracks` dependency type for **non-blocking cross-project
references**. Unlike `blocks`, a `tracks` edge never prevents an issue from being ready.

```
DepTracks : DependencyType = "tracks"

AffectsReadyWork(Tracks) = false
```

### Convoy Pattern

```
Convoy(parent) = { child | ∃ dep(parent, child, Tracks) }

Ready(parent) does NOT depend on status of Convoy(parent)
```

A convoy parent can be ready and dispatched even if tracked children are still open.
This enables cross-rig awareness without coupling.

### Operations

| Operation | Semantics |
|-----------|-----------|
| `bd dep add A --type=tracks --blocked-by B` | A tracks B (non-blocking) |
| `bd dep list B --direction=up -t tracks` | Show what tracks B |
| External refs: `external:rig:id` | Cross-rig tracking |

### Algebraic Properties

1. **Non-interference**: Tracks edges never appear in blockedSet computation
2. **Transitivity-free**: Tracking is NOT transitive (A tracks B, B tracks C ⇏ A tracks C)
3. **Directionality**: Parent→child (parent tracks child); not symmetric
4. **Cross-rig**: Supports external references via routing resolution

---

## 6. Wisp (Ephemeral Issue) Scaffolding

**Files**: cmd/bd/wisp.go, storage/dolt/issues.go, types/types.go

### Core Abstraction

Wisps are ephemeral issues — stored in a parallel `wisps` table, excluded from
git sync, subject to TTL-based garbage collection. They represent transient
operational state (heartbeats, patrol reports, error records).

### Storage Partitioning

```
Issue.Ephemeral = true  →  wisps table
Issue.Ephemeral = false →  issues table

Routing: getIssuesByIDs() auto-routes based on ephemeral flag
Cross-table deps: dependencies ∪ wisp_dependencies (unified for cycle detection)
```

### Wisp Lifecycle (Sublimation Metaphor)

```
Proto (Solid)  →[create]→  Wisp (Vapor)  →[squash]→  Permanent (Solid)
                                          →[burn]→    Deleted (Gone)
                                          →[GC]→      Deleted (Gone)
```

| Operation | Effect |
|-----------|--------|
| Create | Formula/proto → ephemeral wisp in wisps table |
| Squash | Clears ephemeral flag → promotes to issues table |
| Burn | Deletes without audit digest |
| GC | TTL-based cleanup of abandoned wisps |

### Garbage Collection

```
WispGC(threshold) =
  candidates ← { w | w ∈ wisps,
                     w.UpdatedAt < now - threshold,
                     w.Status ∉ {Pinned},
                     w.IssueType ∉ infrastructureTypes }
  for w ∈ candidates:
    cascade_delete(w)  -- includes dependent step wisps
```

Infrastructure types (agents, roles, rigs, messages) are protected from GC.

### Formula Integration

```
CreateWisp(proto, vars) → Wisp:
  1. Load formula definition
  2. Apply variable defaults
  3. Validate required variables
  4. Create ephemeral issue with:
     - Ephemeral = true
     - ID prefix = IDPrefixWisp ("wisp")
     - SourceFormula = formula name
     - SourceLocation = step location
  5. Root-only: no child step materialization by default
```

### Key Invariants

1. **Ephemeral exclusion**: Wisps are never synced via git
2. **Cross-table integrity**: Cycle detection spans both tables
3. **GC safety**: Infrastructure types and pinned wisps are never collected
4. **Squash promotion**: Squash is the only path from ephemeral to permanent
5. **TTL stratification**: WispType determines compaction timeline (6h/24h/7d)

### Lean 4 Formalization Candidates

1. **Partition invariant**: Every issue is in exactly one table (wisps XOR issues)
2. **GC safety**: GC never deletes pinned or infrastructure wisps
3. **Cross-table cycle freedom**: No Blocks cycles across wisps ∪ issues
4. **TTL ordering**: Heartbeat TTL < Patrol TTL < Recovery TTL

---

## 7. Storage & Transactional Guarantees

**Files**: storage/storage.go (interface), storage/dolt/issues.go (implementation)

### Storage Interface

```
Storage {
  -- Issue CRUD
  CreateIssue  : Issue × Actor → Error
  GetIssue     : IssueID → Issue × Error
  UpdateIssue  : IssueID × Updates × Actor → Error
  CloseIssue   : IssueID × Reason × Actor × Session → Error
  DeleteIssue  : IssueID → Error

  -- Dependencies
  AddDependency    : Dependency × Actor → Error
  RemoveDependency : (IssueID, DependsOnID) × Actor → Error
  GetDependencies  : IssueID → List Issue × Error
  GetDependents    : IssueID → List Issue × Error
  GetDependencyTree : (IssueID, maxDepth, showAllPaths, reverse) → List TreeNode × Error

  -- Work queries
  GetReadyWork     : WorkFilter → List Issue × Error
  GetBlockedIssues : WorkFilter → List BlockedIssue × Error

  -- Labels, Comments, Events, Statistics, Config
  -- (CRUD operations with standard signatures)

  -- Transactions
  RunInTransaction : (commitMsg, fn) → Error
  Close : () → Error
}
```

### Transactional Model

Every write operation executes within a Dolt transaction:
```
BEGIN → mutations → CALL DOLT_COMMIT('message') → END
```

Rollback on any failure. Each operation produces exactly one Dolt commit
with a descriptive message (e.g., `bd: close gt-abc`).

### Audit Trail

```
Event { ID: Int64, IssueID: String, EventType: EventType,
        Actor: String, OldValue NewValue Comment: Option String,
        CreatedAt: Time }

EventType = Created | Updated | StatusChanged | Commented | Closed
          | Reopened | DependencyAdded | DependencyRemoved
          | LabelAdded | LabelRemoved | Compacted
```

Every mutation records an Event. The full history is accessible via Dolt's
git-like version control (branches, diffs, logs).

### Delete Semantics

```
DeleteIssue(id, cascade, force) → DeleteIssuesResult:
  cascade = true  → recursively include all dependents
  force = true    → delete, orphan remaining dependents
  default         → fail if dependents exist

  Cleanup order:
    1. Update text references → [deleted:ID]
    2. Remove outgoing dependencies
    3. Remove inbound dependencies
    4. Delete from issue table
    5. Cascade: labels, events, comments (FK-enforced)
```

---

## 8. Cross-Package Algebraic Structure

### The Ready Work Pipeline

```
1. Issue.Validate()              → well-formed issue
2. Dependencies.AddDependency()  → DAG edge (acyclic for Blocks)
3. computeBlockedIDs()           → blocked set (cached)
4. GetReadyWork(filter)          → unblocked, undeferred issues
5. Sort(priority ASC, created DESC) → dispatch-ordered list
```

### Shared Algebraic Patterns

**1. Guarded Transitions** (claim, close, delete):
```
claim : ¬Assigned → Assigned (CAS)
close : ¬Blocked ∧ ¬Pinned ∧ ¬Template → Closed
delete : ¬Template ∧ (¬HasDependents ∨ cascade ∨ force) → Deleted
```

All guarded transitions follow the pattern: check preconditions atomically,
apply effects, record audit event.

**2. Monotone Predicates** (blocking, ready work):
```
close(dep)  → blockedSet can only shrink
close(dep)  → readySet can only grow
add(dep)    → blockedSet can only grow
add(dep)    → readySet can only shrink
```

**3. TTL Stratification** (wisp compaction):
```
WispType → TTL → CompactionSchedule
High-churn (6h) < Operational (24h) < Significant (7d)
```

**4. Fail-Open in Storage** (consistent with convoy pattern in gastown):
```
Cache miss → recompute (never stale positive)
Storage error → skip (never false dispatch)
Missing issue → skip (graceful degradation)
```

**5. Content Addressing** (hash-based equivalence):
```
ContentHash : Issue → SHA256
issue₁ ≡ issue₂ ↔ ContentHash(issue₁) = ContentHash(issue₂)
```

### Conservation Laws

1. **Event conservation**: Every mutation produces exactly one Event record
2. **Dependency symmetry**: AddDependency(A→B) creates one edge; visible from both A's deps and B's dependents
3. **Status exclusivity**: An issue is in exactly one status at any time
4. **Table partitioning**: An issue is in exactly one table (issues XOR wisps)
5. **Claim exclusivity**: At most one actor holds a claim at any time

### Category-Theoretic View

```
Objects: Issue states (status × assignee × deps × ...)
Morphisms: Operations (create, claim, update, close, delete)

Key functors:
  Issue → Graph       (extracting dependency edges)
  Graph → BlockedSet  (computeBlockedIDs)
  BlockedSet → ReadySet (complement within active issues)
  ReadySet → SortedList (priority ordering)
```

The composition `Issue → ... → SortedList` is the ready work pipeline.

---

## 9. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Area | Target | Why |
|------|--------|-----|
| Dependency DAG | computeBlockedIDs correctness | Central to dispatch; monotonicity + caching |
| Claim lifecycle | CAS atomicity, exclusivity | Prevents double-dispatch in multi-agent |
| Close guards | Blocker check + auto-close propagation | Cascading effects, correctness-critical |

### Medium Priority (simpler structure, clear properties)

| Area | Target | Why |
|------|--------|-----|
| Priority ordering | Total order + sort determinism | Simple but foundational |
| Wisp partitioning | Table invariant, GC safety | Partition correctness |
| Validator chains | Composition of guard predicates | Algebraic composition |

### Cross-Cutting Theorems

1. **Blocking monotonicity**: Closing a dependency can only shrink the blocked set
2. **Ready work anti-monotonicity**: Adding a Blocks edge can only shrink the ready set
3. **Claim exclusivity**: ClaimIssue succeeds for at most one concurrent caller per issue
4. **Close-ClosedAt equivalence**: `status = Closed ↔ closed_at ≠ nil` is an invariant
5. **Event conservation**: `|events(issue)| ≥ |mutations(issue)|` (every mutation logged)
6. **DAG acyclicity**: The Blocks-edge subgraph is always acyclic
7. **Partition completeness**: `|issues| + |wisps| = |all_beads|` (no orphan records)
