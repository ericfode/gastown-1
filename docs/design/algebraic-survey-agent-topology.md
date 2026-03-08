# Algebraic Survey: Agent Topology Subsystem

**Subsystem**: Agent Topology
**Packages**: `config` (roles, agents), `polecat`, `refinery`, `protocol`, `dog`, `deacon`, `witness`, `session`, `cmd` (agents)
**Purpose**: Defines WHAT agents exist, WHERE they run, HOW they transition between
lifecycle states, and WHICH invariants the agent graph must satisfy.

## 1. Subsystem Overview

The Agent Topology subsystem answers: given a town with N rigs, what agents exist,
how are they organized hierarchically, what states can they be in, and what
operations transform the agent graph?

```
Town (Singleton)
├── Mayor     (1)    ← Global coordinator
├── Deacon    (1)    ← Daemon beacon, heartbeat/monitoring
├── Dogs      (0..N) ← Deacon's cross-rig infrastructure workers
└── Rigs      (1..M)
    ├── Witness   (0..1) ← Worker monitor, cleanup
    ├── Refinery  (0..1) ← Merge queue processor
    ├── Polecats  (0..N) ← Persistent workers, ephemeral sessions
    └── Crew      (0..N) ← User-managed persistent workspaces
```

### Package Dependency Graph (within subsystem)

```
config/roles     ←── (leaf: defines RoleDefinition, loads TOML)
config/agents    ←── (leaf: defines AgentPresetInfo, LLM runtime registry)
beads/status     ←── (leaf: defines AgentState, IssueStatus enums)
polecat/types    ←── (leaf: defines polecat State, CleanupStatus)
dog/types        ←── (leaf: defines dog State)
refinery/types   ←── (leaf: defines MRStatus, MRPhase, transition validation)
protocol/types   ←── (leaf: defines inter-agent MessageType, payloads)
polecat/manager  ←── polecat/types, beads, config, git, rig, session, tmux, workspace
witness/handlers ←── beads, session, protocol
cmd/agents       ←── config, constants (display layer, AgentType enum)
```

**Key finding**: The type definitions form a layered architecture. Five leaf
packages (`config/roles`, `config/agents`, `beads/status`, `polecat/types`,
`dog/types`, `refinery/types`) define the core algebraic types with no
cross-dependencies. Higher-level packages (`polecat/manager`, `witness/handlers`)
compose these types into lifecycle operations.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| config/roles | session, polecat/manager, cmd/spawn, cmd/agents |
| config/agents | session, polecat/manager, hooks installer |
| beads/status | witness, polecat, refinery, sling, cmd |
| polecat/types | polecat/manager, witness, cmd |
| protocol/types | witness, refinery, protocol/handlers |
| dog/types | deacon, dog/kennel |

---

## 2. Type: Role Hierarchy

**Files**: `internal/config/roles.go`, `internal/config/roles/*.toml`

### Type Algebra

The role system is a **fixed enumeration** with a two-level scope partition:

```
Role = Mayor | Deacon | Dog | Witness | Refinery | Polecat | Crew
Scope = Town | Rig

partition : Role → Scope
partition Mayor   = Town
partition Deacon  = Town
partition Dog     = Town
partition Witness = Rig
partition Refinery = Rig
partition Polecat  = Rig
partition Crew     = Rig
```

This is enforced in code:

```go
// config/roles.go
func AllRoles() []string {
    return []string{"mayor", "deacon", "dog", "witness", "refinery", "polecat", "crew"}
}
func TownRoles() []string { return []string{"mayor", "deacon", "dog"} }
func RigRoles() []string  { return []string{"witness", "refinery", "polecat", "crew"} }
```

### Core Type: RoleDefinition

```
RoleDefinition {
  Role: String,             -- role identifier
  Scope: "town" | "rig",    -- scope partition
  Session: RoleSessionConfig,
  Env: Map String String,   -- environment variables
  Health: RoleHealthConfig,  -- monitoring thresholds
  Nudge: String,             -- startup prompt
  PromptTemplate: String     -- template file name
}

RoleSessionConfig {
  Pattern: String,       -- tmux session name pattern (supports {rig}, {name}, {role}, {prefix})
  WorkDir: String,       -- working directory pattern (supports {town}, {rig}, {name}, {role})
  NeedsPreSync: Bool,    -- whether workspace needs git sync before starting
  StartCommand: String   -- command to run (default: "exec claude --dangerously-skip-permissions")
}

RoleHealthConfig {
  PingTimeout: Duration,
  ConsecutiveFailures: Nat,
  KillCooldown: Duration,
  StuckThreshold: Duration,
  HungSessionThreshold: Duration
}
```

### Role Configuration Resolution (3-layer merge)

```
LoadRoleDefinition : (townRoot, rigPath, roleName) → RoleDefinition

resolve(role) = merge(
  loadBuiltin(role),                    -- 1. Embedded defaults
  loadOverride(town/roles/role.toml),   -- 2. Town-level overrides
  loadOverride(rig/roles/role.toml)     -- 3. Rig-level overrides
)
```

**Merge semantics**: Non-zero override fields replace base values. Role and Scope
are **immutable** — cannot be changed via override. `NeedsPreSync` can only be
**enabled** via override, never disabled (monotonic strengthening).

### Session Name Patterns (from TOML)

| Role | Pattern | Example |
|------|---------|---------|
| Mayor | `hq-mayor` | `hq-mayor` |
| Deacon | `hq-deacon` | `hq-deacon` |
| Dog | `gt-dog-{name}` | `gt-dog-alpha` |
| Witness | `{prefix}-witness` | `gt-witness` |
| Refinery | `{prefix}-refinery` | `gt-refinery` |
| Polecat | `{prefix}-{name}` | `gt-dementus` |
| Crew | `{prefix}-crew-{name}` | `gt-crew-morpheus` |

### Key Operations

- `AllRoles() → [String]` — enumerates the universe
- `isValidRoleName(name) → Bool` — membership predicate
- `ExpandPattern(pattern, town, rig, name, role, prefix) → String` — template instantiation
- `mergeRoleDefinition(base, override)` — partial update with monotonicity constraints

---

## 3. Type: Agent State Machines

**Files**: `internal/beads/status.go`, `internal/polecat/types.go`, `internal/dog/types.go`

### 3.1 AgentState (beads-level, all agents)

A **9-element enumeration** with semantic predicates:

```
AgentState = Spawning | Working | Done | Stuck | Escalated
           | Idle | Running | Nuked | AwaitingGate

ProtectsFromCleanup : AgentState → Bool
ProtectsFromCleanup Stuck        = True
ProtectsFromCleanup AwaitingGate = True
ProtectsFromCleanup _            = False

IsActive : AgentState → Bool
IsActive Working  = True
IsActive Running  = True
IsActive Spawning = True
IsActive _        = False
```

**State graph** (observed transitions in manager code):

```
        ┌──────────────────────────────┐
        ↓                              │
   Spawning → Working → Done → Idle → Working  (happy path cycle)
        │         │                │
        │         ↓                ↓
        │      Stuck            Nuked
        │         │
        ↓         ↓
      (rollback)  Escalated
```

Note: `Running` is used for ZFC-compliant derivation from tmux state, not stored.

### 3.2 Polecat State (polecat-specific lifecycle)

A **5-element enumeration** with behavioral semantics:

```
State = Working | Idle | Done | Stuck | Zombie

-- Working: Session active, doing assigned work
-- Idle: Work completed, session killed, sandbox PRESERVED for reuse
-- Done: Called 'gt done' — transient state before cleanup
-- Stuck: Explicit request for help (self-reported)
-- Zombie: tmux session exists but no corresponding worktree (detected)
```

**Key insight (gt-4ac)**: Polecats are **persistent**. The happy-path cycle is
`Working → Idle → Working`, not `Working → Nuked`. Idle polecats keep their
worktree for reuse, eliminating spawn overhead.

### 3.3 Polecat CleanupStatus (removal safety)

A **5-element enumeration** with safety predicates:

```
CleanupStatus = Clean | Uncommitted | Stash | Unpushed | Unknown

IsSafe : CleanupStatus → Bool
IsSafe Clean = True
IsSafe _     = False

RequiresRecovery : CleanupStatus → Bool
RequiresRecovery Uncommitted = True
RequiresRecovery Stash       = True
RequiresRecovery Unpushed    = True
RequiresRecovery _           = False

CanForceRemove : CleanupStatus → Bool
CanForceRemove Clean       = True
CanForceRemove Uncommitted = True
CanForceRemove _           = False
```

**Lattice structure**: `Clean ⊑ Uncommitted ⊑ Stash ≈ Unpushed`. Safety
decreases monotonically as work state increases. `CanForceRemove` is a
coarsening of `IsSafe`.

### 3.4 Dog State

A **2-element enumeration** (simplest of all agents):

```
DogState = Idle | Working
```

Dogs are simpler than polecats — they have no `Done/Stuck/Zombie` states because
their lifecycle is managed entirely by the Deacon.

### 3.5 IssueStatus (beads-level)

```
IssueStatus = Open | Closed | InProgress | Tombstone | Blocked | Pinned | Hooked

BlocksRemoval : IssueStatus → Bool
BlocksRemoval Open = True
BlocksRemoval _    = False

IsTerminal : IssueStatus → Bool
IsTerminal Closed    = True
IsTerminal Tombstone = True
IsTerminal _         = False

IsAssigned : IssueStatus → Bool
IsAssigned Hooked     = True
IsAssigned InProgress = True
IsAssigned _          = False
```

---

## 4. Type: Merge Request State Machine

**File**: `internal/refinery/types.go`

### 4.1 MRStatus (coarse, beads-compatible)

```
MRStatus = Open | InProgress | Closed

-- Valid transitions:
-- Open → InProgress   (Engineer claims MR)
-- Open → Closed       (manual rejection)
-- InProgress → Closed (merge success or rejection)
-- InProgress → Open   (failure, reassign to worker)
-- Closed → ∅          (immutable — terminal state)
```

### 4.2 MRPhase (fine-grained, v2 state machine)

```
MRPhase = Ready | Claimed | Preparing | Prepared | Merging | Merged | Rejected | Failed

ValidPhaseTransitions : MRPhase → Set MRPhase
Ready     → {Claimed}
Claimed   → {Preparing, Ready}
Preparing → {Prepared, Failed}
Prepared  → {Merging, Rejected, Ready}
Merging   → {Merged, Failed}
Failed    → {Ready}
Merged    → ∅   (terminal)
Rejected  → ∅   (terminal)
```

**Graph properties**:
- **Two terminal states**: `Merged` and `Rejected` (no outgoing edges)
- **One recovery cycle**: `Failed → Ready` allows retry
- **Rework cycle**: `Prepared → Ready` when quality gates suggest fixable issues
- **Monotonic progress**: the primary path `Ready → Claimed → Preparing → Prepared → Merging → Merged` is strictly forward
- **Validated transitions**: `ValidatePhaseTransition(from, to)` enforces the graph at runtime

### 4.3 FailureType

```
FailureType = None | Conflict | TestsFail | BuildFail | FlakyTest
            | PushFail | Fetch | Checkout

ShouldAssignToWorker : FailureType → Bool
ShouldAssignToWorker Conflict  = True
ShouldAssignToWorker TestsFail = True
ShouldAssignToWorker BuildFail = True
ShouldAssignToWorker FlakyTest = True
ShouldAssignToWorker _         = False

FailureLabel : FailureType → String
FailureLabel Conflict               = "needs-rebase"
FailureLabel TestsFail | BuildFail  = "needs-fix"
FailureLabel PushFail               = "needs-retry"
```

---

## 5. Spawn and Despawn Operations

**File**: `internal/polecat/manager.go`

### 5.1 Polecat Spawn (Manager.AddWithOptions)

The spawn operation is a **multi-phase transaction with rollback**:

```
Spawn(name, opts) → (Polecat, Error)

Phase 1: Admission Control
  1. CheckDoltHealth()        -- 10 retries, exponential backoff with ±25% jitter
  2. CheckDoltServerCapacity() -- fail-closed admission gate (gt-lfc0d)
  3. lockPolecat(name)        -- file-based mutual exclusion
  4. exists(name) → reject ErrPolecatExists

Phase 2: Resource Allocation
  5. AllocateName()           -- from themed name pool
  6. MkdirAll(polecatDir)     -- create polecats/<name>/
  7. Fetch("origin")          -- sync remote refs
  8. WorktreeAddFromRef()     -- create git worktree on new branch

Phase 3: Configuration
  9. setupSharedBeads()       -- link .beads/ to rig canonical location
  10. ProvisionPrimeMD()      -- instructions for agent
  11. CopyOverlay()           -- rig overlay files
  12. EnsureGitignorePatterns()
  13. EnsureSettingsForRole()  -- hooks, slash commands
  14. RunSetupHooks()          -- rig-specific setup

Phase 4: Registration
  15. createAgentBeadWithRetry() -- non-ephemeral agent bead (state="spawning")
  16. Return Polecat{State: Working}
```

**Rollback on error** (cleanupOnError): resets agent bead, removes worktree,
removes directory, releases name back to pool.

**Error classification for Dolt retries**:
- Optimistic lock errors → retry (transient write conflict)
- Config/init errors → fail fast (gt-2ra: no retry, waste prevention)
- Read-only errors → attempt server recovery (gt-chx92)

### 5.2 Polecat Removal (Manager.RemoveWithOptions)

```
Remove(name, force, nuclear, selfNuke) → Error

Phase 1: Safety Checks
  1. lockPolecat(name)
  2. Check CleanupStatus (ZFC path: trust self-report, fallback: git check)
  3. Check ActiveMR (refuse if MR is open in merge queue — gt-6a9d)

Phase 2: Bead Cleanup
  4. ResetAgentBeadForReuse() -- clear fields, set state="nuked"
  5. unassignWorkBeads()      -- release any assigned beads (gt-e4u1)

Phase 3: Filesystem Cleanup
  6. Check cwd-in-worktree (skip if selfNuke)
  7. KillSession()            -- tmux session
  8. DeleteBranch()           -- remote + local
  9. WorktreeRemove()         -- git worktree
  10. RemoveAll(polecatDir)   -- directory
  11. Release name to pool
```

**Safety hierarchy**: `nuclear > force > default`. Nuclear bypasses git status
checks (needed for self-nuke) but still checks MR status.

### 5.3 Witness Operations

```
NukePolecat(workDir, rig, name) → Error
  1. Check for pending MR (refuse if pending — gt-6a9d)
  2. Kill tmux session (Ctrl-C → brief delay → force kill)
  3. Run `gt polecat nuke` for full cleanup

RestartPolecatSession(workDir, rig, name) → Error
  -- Used for stuck/hung polecats with work worth preserving (gt-dsgp)
  1. Kill existing tmux session
  2. Start fresh session via `gt session restart`
  3. New session picks up existing hook and continues

AutoNukeIfClean → always returns Skipped (gt-4ac)
  -- Persistent polecat model: sandbox preserved for reuse
```

---

## 6. Inter-Agent Protocol

**File**: `internal/protocol/types.go`

### 6.1 Protocol Message Types

```
MessageType = MergeReady | Merged | MergeFailed | ReworkRequest | ConvoyNeedsFeeding

-- Direction and purpose:
MergeReady         : Witness → Refinery  (branch verified, ready for merge)
Merged             : Refinery → Witness  (merge succeeded)
MergeFailed        : Refinery → Witness  (merge failed — tests/build/push)
ReworkRequest      : Refinery → Witness  (needs rebase — conflicts)
ConvoyNeedsFeeding : Refinery → Deacon   (convoy may have newly-ready issues)
```

### 6.2 Protocol Payloads

Each message type has a structured payload (JSON-serializable):

```
MergeReadyPayload {Branch, Issue, Polecat, Rig, Verified, Timestamp}
MergedPayload {Branch, Issue, Polecat, Rig, MergedAt, MergeCommit, TargetBranch}
MergeFailedPayload {Branch, Issue, Polecat, Rig, FailedAt, FailureType, Error, TargetBranch}
ReworkRequestPayload {Branch, Issue, Polecat, Rig, RequestedAt, TargetBranch, ConflictFiles, Instructions}
ConvoyNeedsFeedingPayload {ConvoyID, SourceIssue, Rig, MergedAt}
```

### 6.3 Polecat Exit Protocol

```
PolecatDonePayload {
  Polecat, ExitType, Issue, Branch, MR,
  ConvoyID, ConvoyOwned, MergeStrategy, Errors
}

ExitType = "COMPLETED" | "ESCALATED" | "DEFERRED" | "PHASE_COMPLETE"

SkipMergeFlow : PolecatDonePayload → Bool
SkipMergeFlow p = p.ConvoyOwned ∧ p.MergeStrategy == "direct"
```

---

## 7. Agent LLM Runtime Registry

**File**: `internal/config/agents.go`

### Type Algebra

The agent registry maps agent preset names to runtime configurations:

```
AgentPreset = Claude | Gemini | Codex | Cursor | Auggie | Amp | OpenCode | Copilot | Pi | Omp

AgentPresetInfo {
  Name: AgentPreset,
  Command: String,            -- CLI binary
  Args: [String],             -- autonomous mode flags
  Env: Map String String,     -- agent-specific env vars
  ProcessNames: [String],     -- for tmux liveness detection
  SessionIDEnv: String,       -- env var for session ID
  ResumeFlag: String,         -- flag for resuming sessions
  ContinueFlag: String,       -- flag for auto-resume
  ResumeStyle: "flag" | "subcommand",
  SupportsHooks: Bool,
  SupportsForkSession: Bool,
  NonInteractive: Option NonInteractiveConfig,
  -- Runtime configuration
  PromptMode: "arg" | "none",
  ConfigDir: String,
  HooksProvider: String,
  HooksDir: String,
  HooksSettingsFile: String,
  HooksInformational: Bool,
  ReadyPromptPrefix: String,
  ReadyDelayMs: Nat,
  InstructionsFile: String,
  EmitsPermissionWarning: Bool
}
```

**Registry resolution** (3-layer, same pattern as roles):

```
ResolveAgent(name) = merge(
  builtinPresets[name],                -- 1. Compiled-in defaults
  loadJSON(town/settings/agents.json), -- 2. Town-level overrides
  loadJSON(rig/settings/agents.json)   -- 3. Rig-level overrides
)
```

**Key properties**:
- Registry is **thread-safe** (`sync.RWMutex` + `registryInitialized` flag)
- Loading is **idempotent** (cached via `loadedPaths`)
- User agents **override** built-in presets with the same name
- Default preset is `Claude` (`DefaultAgentPreset()`)

---

## 8. Topology Invariants

### Cardinality Invariants

| Agent | Scope | Cardinality | Enforcement Mechanism |
|-------|-------|-------------|----------------------|
| Mayor | Town | Exactly 1 | Fixed session name `hq-mayor` |
| Deacon | Town | Exactly 1 | Fixed session name `hq-deacon` |
| Dogs | Town | 0..N | Named pool, managed by Deacon |
| Witness | Per-Rig | At most 1 | Session pattern `{prefix}-witness` |
| Refinery | Per-Rig | At most 1 | Session pattern `{prefix}-refinery` |
| Polecats | Per-Rig | 0..N | Pool-based naming, file locks |
| Crew | Per-Rig | 0..N | Per-user directories |

**Singleton enforcement**: Mayor and Deacon are enforced by fixed tmux session
names — creating a second session with the same name fails. Witness and Refinery
use per-rig session patterns, producing at-most-one per rig.

### Structural Invariants

1. **Scope partition is total**: Every role maps to exactly one scope (Town or Rig).
   `AllRoles() = TownRoles() ∪ RigRoles()` and `TownRoles() ∩ RigRoles() = ∅`.

2. **Role immutability under override**: `mergeRoleDefinition` never changes
   `Role` or `Scope` fields. An override cannot transform a witness into a mayor.

3. **NeedsPreSync monotonicity**: Override can enable pre-sync but never disable it.
   If a role's builtin requires pre-sync (refinery, crew), overrides cannot weaken
   this requirement. `∀ override: base.NeedsPreSync ⇒ merged.NeedsPreSync`.

4. **Work assignment exclusivity**: Each polecat has at most one `hook_bead` at
   any time. Hook assignment is set atomically during spawn or sling.

5. **MR protection invariant** (gt-6a9d): A polecat with an open MR in the
   refinery queue CANNOT be nuked. The nuke operation checks
   `hasPendingMR(...)` and refuses if true.

6. **Persistent identity** (gt-4ac): Polecat identity (name, CV chain, mailbox)
   and sandbox (worktree) persist across sessions. `AutoNukeIfClean` always
   returns `Skipped`. The `Working → Idle → Working` cycle is the expected path.

7. **ZFC compliance**: Running state is derived from tmux sessions, not stored
   in beads. `AgentStateRunning` is a derived value, not a persisted state.

8. **Cleanup safety ordering**: `IsSafe(s) ⇒ CanForceRemove(s)` (safe implies
   force-removable). `RequiresRecovery(s) ⇒ ¬IsSafe(s)` (recovery needed means
   not safe). These predicates on `CleanupStatus` form a consistent safety lattice.

9. **Phase transition completeness**: Every non-terminal `MRPhase` has at least
   one outgoing transition. Terminal states (`Merged`, `Rejected`) have none.
   `Failed → Ready` ensures liveness (no permanent stuck state).

10. **Protocol message flow is acyclic**: Protocol messages flow
    `Polecat → Witness → Refinery → {Witness, Deacon}`. No message type creates
    a cycle in the communication graph.

---

## 9. Cross-Cutting Patterns

### Pattern: Two-Level State Machines

Both MR and Polecat lifecycle use a two-level state model:
- **Coarse level**: beads-compatible status (`open/in_progress/closed` for MR;
  `spawning/working/done/idle/nuked` for agents)
- **Fine level**: operational detail (`MRPhase` for MR processing steps;
  `polecat.State` for lifecycle nuance like `Zombie`)

The coarse level enables cross-system queries; the fine level drives operational
decisions. This is a standard **refinement** relationship.

### Pattern: Fail-Fast vs Retry Classification

The spawn operation classifies Dolt errors into three categories:
1. **Optimistic lock** → retry with exponential backoff (transient)
2. **Config/init error** → fail immediately (permanent, gt-2ra)
3. **Read-only error** → attempt server recovery, then retry (gt-chx92)

This classification prevents wasted retry loops (3 minutes saved on config errors)
while remaining resilient to transient failures.

### Pattern: Monotonic Safety Constraints

Multiple subsystems enforce one-way strengthening:
- `NeedsPreSync`: can only be enabled, never disabled via override
- `CleanupStatus`: safety predicates are monotonically ordered
- `IssueStatus.IsTerminal`: terminal states are absorbing (no exit transitions)
- `MRPhase`: the `Merged`/`Rejected` terminal states are irrevocable

### Pattern: File-Lock Mutual Exclusion

Polecat operations use per-resource file locks (`flock.Flock`):
- `lockPolecat(name)` — per-polecat operations (Add, Remove, Repair)
- `lockPool()` — name pool operations (AllocateName, ReconcilePool)

This prevents concurrent `gt` processes from racing on the same polecat's state.

---

## 10. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Component | Target | Why |
|-----------|--------|-----|
| MRPhase transitions | Transition graph correctness, terminal state reachability | Classic finite automaton verification |
| Spawn transaction | Rollback completeness, resource leak freedom | Multi-phase transaction with cleanup obligations |
| CleanupStatus lattice | Safety predicate consistency, ordering properties | Partial order with semantic predicates |

### Medium Priority (simpler structure, useful invariants)

| Component | Target | Why |
|-----------|--------|-----|
| Role partition | Scope totality, override monotonicity | Simple but foundational — errors here cascade |
| Cardinality constraints | Singleton enforcement correctness | Relies on tmux naming convention, not type system |
| Protocol message flow | Acyclicity, payload completeness | Communication graph properties |

### Low Priority (runtime behavior, harder to formalize)

| Component | Target | Why |
|-----------|--------|-----|
| Dolt retry classification | Correctness of error categorization | Heuristic string matching, not algebraic |
| Agent registry resolution | 3-layer merge correctness | Similar to role resolution but for runtime config |
| Name pool allocation | Uniqueness, exhaustion handling | Operational concern more than algebraic |

### Cross-Cutting Theorems

1. **Scope partition is a coproduct**: `AllRoles ≅ TownRoles + RigRoles` (disjoint union)
2. **Cardinality enforcement via naming**: Singleton agents are enforced by injective session naming, not by a type-level constraint
3. **Persistent polecats form a pool**: The `Working ↔ Idle` cycle is a reuse monad — no resource is created or destroyed in steady state
4. **MR phase transitions form a DAG with recovery edges**: Remove `Failed → Ready` and the graph is a strict DAG; the recovery edge adds bounded liveness
5. **Protocol messages are typed by direction**: Each `MessageType` has a fixed (sender, receiver) pair — the type determines the communication channel
