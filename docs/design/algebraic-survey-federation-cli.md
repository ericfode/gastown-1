# Algebraic Survey: Federation & CLI Subsystem

**Subsystem**: Federation & CLI
**Packages**: `cmd`, `rig`, `wasteland`, `polecat`, `boot`, `hooks`, `hookutil`, `beads`, `wisp`, `mail`
**Purpose**: Maps user intent to system operations — the CLI command tree defines the
operator algebra, rigs federate into towns, beads provide the data plane, and the
prime/hook/done lifecycle governs agent session state.

## 1. Subsystem Overview

The Federation & CLI subsystem answers: given an agent (human or polecat) with a role
in a rig, how does intent flow from command invocation through federation routing to
state transitions across the distributed system?

```
CLI (intent) → Role (identity) → Rig (scope) → Beads (state) → Session (execution)
                                    ↓
                              Federation (cross-rig routing)
```

### Package Dependency Graph (within subsystem)

```
cmd       ←── rig, beads, polecat, mail, session, hooks (orchestrator, imports everything)
rig       ←── config, beads, git (leaf for federation)
wasteland ←── config (leaf for town-to-town)
polecat   ←── rig, beads, session, tmux, hooks (agent lifecycle)
boot      ←── tmux, session (watchdog, minimal deps)
hooks     ←── config (hook resolution, no subsystem deps)
hookutil  ←── hooks (utilities for hook config)
beads     ←── (external: bd CLI wrapper, no internal Go deps)
wisp      ←── (thin: directory utilities for ephemeral beads)
mail      ←── beads (message routing over beads)
```

**Key finding**: The `cmd` package is the hub — it imports everything else and wires
the command tree. Below it, packages are loosely coupled: `rig`, `polecat`, `beads`,
and `mail` share no direct imports and communicate only through the beads data plane
and environment variables. This is a **star topology** with `cmd` at center.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| cmd | cmd/gt (entry point) |
| rig | cmd, daemon, witness, refinery, crew, polecat |
| wasteland | cmd/wasteland (join/sync commands) |
| polecat | cmd, witness, daemon |
| boot | deacon (watchdog subprocess) |
| hooks | session, polecat, crew |
| beads | cmd, rig, polecat, mail, convoy, witness, refinery |
| mail | cmd, polecat, witness, mayor, nudge |

---

## 2. Package: cmd (CLI Command Tree)

**Files**: root.go, prime.go, hook.go, done.go, role.go, molecule_status.go, + ~60 command files

### Type Algebra

The CLI defines a **command algebra** — a rooted tree of operations dispatched by name:

```
Command = Root | Parent Command* | Leaf
Root = { Use: "gt", Groups: List Group, PersistentPreRun: Hook }
Parent = { Use: String, Subcommands: List Command }
Leaf = { Use: String, Run: Args → Error, Flags: List Flag }
```

Seven semantic groups partition the command space:

```
Group = Work | Agents | Comm | Services | Workspace | Config | Diag
```

### Command Tree Structure (~97 top-level commands)

| Group | Commands | Purpose |
|-------|----------|---------|
| Work (14) | ready, done, hook, sling, unsling, close, checkpoint, handoff, ... | Work lifecycle |
| Agents (20) | crew, polecat, witness, refinery, mayor, session, role, ... | Agent management |
| Comm (4) | mail, broadcast, notify, nudge | Inter-agent communication |
| Services (7) | daemon, deacon, dolt, dog | Background services |
| Workspace (10) | rig, town, convoy, worktree, orphans, ... | Rig & workspace ops |
| Config (6) | config, krc, theme, shell, plugin, namepool | Configuration |
| Diag (36+) | prime, doctor, version, status, health, metrics, ... | Diagnostics |

### Lifecycle Commands (prime → hook → done)

These three commands implement a **session state monad**:

```
prime : (Env, CWD, TownRoot) → (RoleInfo, SessionContext, InjectedEnv)
hook  : AgentID × BeadID? → HookState
done  : (Branch, GitState, IssueID, ExitStatus) → (MRSubmitted | Escalated | Deferred)
```

### Role Detection Algebra

```
RoleInfo {
  Role: Role, Source: String, Home Rig Polecat: String,
  EnvRole: String, CwdRole: Role, Mismatch: Bool,
  EnvIncomplete: Bool, TownRoot WorkDir: String
}

Role = Mayor | Deacon | Boot | Witness | Refinery | Polecat | Crew | Dog | Unknown
```

Detection is a **meet operation** on two partial information sources:

```
detectRole : (GT_ROLE_env, cwd_path) → RoleInfo

-- Path pattern matching:
<town>/mayor/rig/                    → Mayor
<town>/<rig>/witness/                → Witness (Rig = rig)
<town>/<rig>/refinery/rig/           → Refinery (Rig = rig)
<town>/<rig>/polecats/<name>/        → Polecat (Rig = rig, Polecat = name)
<town>/<rig>/crew/<name>/            → Crew (Rig = rig, Polecat = name)

-- When env and cwd disagree: Mismatch = true (warning, not error)
-- When env is incomplete: fill from cwd, EnvIncomplete = true
```

### Hook Status Types

```
HookStatus = Working | Naked | Complete | Blocked
```

### Done Exit Algebra

```
ExitStatus = Completed | Escalated | Deferred
CleanupStatus = Clean | Uncommitted | Unpushed | Stash | Unknown

done(status, cleanup) =
  match status with
  | Completed → validate(branch ≠ main ∧ work_exists ∧ clean) → submit_MR → notify_witness
  | Escalated → skip_MR → notify_witness(blocker)
  | Deferred  → skip_MR → notify_witness(incomplete)
  → transition polecat to Idle (session stays alive)
```

**Done checkpoint** (crash recovery): tracks completed stages, resumes from last.

### Persistent Pre-Run Hook

Every command runs through `persistentPreRun`:

```
persistentPreRun : Command × Args → Error
  1. Initialize theme
  2. Log telemetry
  3. Initialize session registry
  4. Check stale binary (unless beads-exempt)
  5. Check beads version (non-blocking)
  6. Touch polecat heartbeat
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| Execute | () → Error | Total (Cobra dispatches or errors) |
| persistentPreRun | Cmd × Args → Error | Idempotent (safe to run repeatedly) |
| runPrime | Cmd × Args → Error | Effect: injects env, no mutation |
| runHook | Cmd × Args → Error | At-most-one: one bead per hook |
| runDone | Cmd × Args → Error | Checkpoint-resumable, at-most-once MR |

### Command Annotation System

```
Annotations = Map String String
AnnotationPolecatSafe = "true"  -- Safe for polecat to call
```

Commands annotated `PolecatSafe`: prime, hook, done, close, sling, unsling,
checkpoint, cycle, mail, nudge. These cannot interfere with polecat state.

### Lean 4 Formalization Candidates

1. **Role detection determinism**: Same (env, cwd) → same RoleInfo
2. **Hook at-most-one**: hook(agent, bead) fails if agent already hooked
3. **Done checkpoint idempotency**: Resuming from checkpoint never re-executes completed stages
4. **Command tree well-formedness**: No two commands share the same path

---

## 3. Package: rig (Federation)

**Files**: types.go, manager.go, config.go, overlay.go

### Core Abstraction

A rig is a **container** (directory) hosting a git repository and its agents. Rigs
federate into a town via a flat registry. Cross-rig coordination uses prefix-based
routing through the beads data plane.

### Type Algebra

```
Rig {
  Name: String, Path: String, GitURL PushURL LocalRepo: String,
  Config: Option BeadsConfig, Polecats Crew: List String,
  HasWitness HasRefinery HasMayor: Bool
}

-- Invariant: Name is unique within town, contains no hyphens/dots/spaces
-- Invariant: Path = Join(townRoot, Name)

RigEntry { GitURL PushURL UpstreamURL LocalRepo: String, AddedAt: Time, BeadsConfig: Option BeadsConfig }
RigsConfig { Version: Nat, Rigs: Map String RigEntry }

Manager { townRoot: String, config: RigsConfig, git: Git }
```

### Physical Layout

```
<townRoot>/<rigName>/               -- Container (NOT a git repo itself)
  config.json                       -- RigEntry metadata
  .beads/                           -- Issue tracking (Dolt-backed)
  .repo.git/                        -- Bare git repo (shared object store)
  mayor/rig/                        -- Mayor's working clone
  refinery/rig/                     -- Refinery clone (all branches)
  witness/                          -- Witness metadata (no clone)
  polecats/<name>/                  -- Ephemeral worker worktrees
  crew/<name>/                      -- Persistent human workspaces
  .runtime/overlay/                 -- Gitignored files for agent startup
```

### Agent ID Algebra

Agent IDs encode a **complete hierarchical address**:

```
AgentID = <prefix>-[<rig>-]<role>[-<name>]

-- Town-level singletons:    gt-mayor, gt-deacon
-- Per-rig singletons:       gt-gastown-witness, gt-gastown-refinery
-- Per-rig named:            gt-gastown-polecat-Toast, gt-gastown-crew-max

-- Collapse rule: if prefix = rig name, omit rig
--   Rig "ff" with prefix "ff": ff-witness (not ff-ff-witness)

ParseAgentBeadID : String → (Rig × Role × Name × Bool)
-- Parsing is RIGHT-TO-LEFT to handle hyphenated rig names
```

### Prefix Registry (Bidirectional Map)

```
PrefixRegistry {
  prefixToRig : Map String String   -- "gt" → "gastown"
  rigToPrefix : Map String String   -- "gastown" → "gt"
}

-- Invariant: prefixToRig and rigToPrefix are inverses on the registered domain
-- Ordering: longest-prefix-first prevents ambiguous matches
```

### Beads Routing

```
Route { Prefix: String, Path: String }
-- Stored in: ${townRoot}/.beads/routes.jsonl

ResolveRoutingTarget : TownRoot × BeadID × FallbackDir → BeadsDir
  1. ExtractPrefix(beadID)        -- "gt-abc" → "gt"
  2. Lookup routes.jsonl          -- "gt" → "gastown/.beads"
  3. Follow redirect chain        -- .beads/redirect file
  4. Return resolved directory

-- External references: "external:<prefix>:<issue-id>"
-- Lazy-resolved by bd CLI at query time
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| AddRig | AddRigOptions → Rig × Error | Name-unique, 10-step orchestration |
| LoadRig | String × RigEntry → Rig × Error | Idempotent, handles missing components |
| DiscoverRigs | () → List Rig | Total (reads registry) |
| GetRig | String → Rig × Error | Partial (rig may not exist) |
| AgentBeadID | Rig × Role × Name → String | Deterministic, prefix-collapse aware |
| ParseAgentBeadID | String → (Rig × Role × Name) | Right-to-left, handles hyphenated rigs |
| AppendRoute | TownRoot × Route → Error | Injective (one prefix per rig) |

### Configuration Layering

```
GetConfig(key) : lookup order (first non-nil wins)
  1. Wisp layer (ephemeral overrides)
  2. Bead labels (rig identity bead)
  3. Town defaults
  4. System defaults

-- Special: "priority_adjustment" uses additive stacking (not first-wins)
```

### Lean 4 Formalization Candidates

1. **Name uniqueness**: ∀ r₁ r₂ ∈ Town, r₁.Name = r₂.Name → r₁ = r₂
2. **Prefix bijectivity**: PrefixToRig and RigToPrefix are inverses
3. **Agent ID round-trip**: ParseAgentBeadID(AgentBeadID(rig, role, name)) = (rig, role, name)
4. **Route injectivity**: Each prefix maps to exactly one rig path
5. **Config layering determinism**: Same key always resolves through same precedence chain

---

## 4. Package: wasteland (Town-to-Town Federation)

**Files**: wasteland.go

### Core Abstraction

Wasteland enables town-to-town federation via DoltHub fork/PR mechanics. Each town
maintains a sovereign fork of a shared commons database.

### Type Algebra

```
WastelandConfig {
  Upstream: String,    -- DoltHub commons path (e.g., "steveyegge/wl-commons")
  ForkOrg: String,     -- DoltHub org for fork
  ForkDB: String,      -- Fork database name
  LocalDir: String,    -- Absolute path to local clone
  RigHandle: String,   -- Unique handle in registry
  JoinedAt: Time
}
```

### Join Protocol

```
Join(upstream, forkOrg, token, handle, ...) =
  1. Fork commons from upstream → forkOrg/wl-commons
  2. Clone fork locally → ~/.wasteland/forkOrg/wl-commons
  3. Add "upstream" remote → upstream commons
  4. RegisterRig: INSERT into rigs table (ON DUPLICATE KEY UPDATE)
  5. Push registration to fork
  6. Save config to ${townRoot}/mayor/wasteland.json
```

### Key Properties

1. **Idempotent registration**: ON DUPLICATE KEY UPDATE → safe to re-join
2. **Sovereign fork**: All contributions go through fork, never direct to upstream
3. **Single upstream**: Town can join only one upstream commons

### Lean 4 Formalization Candidates

1. **Join idempotency**: Join(x); Join(x) ≡ Join(x) (same final state)
2. **Fork sovereignty**: All mutations go to fork, upstream is read-only

---

## 5. Package: polecat (Agent Lifecycle)

**Files**: types.go, manager.go, session_manager.go, heartbeat.go, namepool.go, + ~10 files

### Core Abstraction

A polecat is a **persistent disposable agent** — it has durable identity and state
that survives work completion, but its session can be killed and restarted. The
lifecycle is a state machine with well-defined transitions.

### Type Algebra

```
Polecat { Name Rig: String, State: State, ClonePath Branch Issue: String, CreatedAt UpdatedAt: Time }

State = Working | Idle | Done | Stuck | Zombie

HeartbeatState = Working | Idle | Exiting | Stuck

SessionHeartbeat { Timestamp: Time, State: HeartbeatState, Context Bead: String }

CleanupStatus = Clean | Uncommitted | Stash | Unpushed | Unknown
```

### Agent Bead Fields

```
AgentFields {
  RoleType: String,       -- "polecat"
  Rig: String,
  AgentState: String,     -- spawning | working | done | stuck | idle | nuked
  HookBead: String,       -- Pinned work bead ID
  ActiveMR: String,       -- Open merge request
  CleanupStatus: String   -- Git state at completion
}
```

### Lifecycle State Machine

```
                  ┌─────────────┐
                  │  Not Exists │
                  └──────┬──────┘
                         │ AllocateAndAdd
                         │ (namepool.Allocate, create worktree,
                         │  create agent bead state=spawning)
                         ▼
              ┌────────────────────┐
              │  Working           │◄─────────────────────┐
              │  (tmux session     │                      │
              │   running,         │                      │ Reuse
              │   heartbeat        │                      │ (SessionStart
              │   active)          │                      │  with new issue)
              └────────┬───────────┘                      │
                       │                                  │
                       │ gt done                          │
                       ▼                                  │
              ┌────────────────────┐                      │
              │  Idle              │──────────────────────┘
              │  (session dead,    │
              │   sandbox preserved,│
              │   identity durable)│
              └────────┬───────────┘
                       │
                       │ Remove (nuke)
                       ▼
              ┌────────────────────┐
              │  Nuked             │
              │  (worktree removed,│
              │   name released,   │
              │   bead reset)      │
              └────────────────────┘
```

### Name Pool Algebra

```
NamePool { RigName Theme: String, CustomNames: List String,
           InUse: Map String Bool, OverflowNext MaxSize: Nat }

Theme = MadMax | Minerals | Wasteland | Custom
-- Each theme provides ~50 names

Allocate : NamePool → String × NamePool
  -- Returns first unused name from theme; overflow to numeric if exhausted

Release : NamePool × String → NamePool
  -- Mark name as available

Reconcile : NamePool × FileSystem → NamePool
  -- Derive InUse from polecats/ directory listing (read reality from disk)

Reserved = {witness, mayor, deacon, refinery, crew, polecats}
Available = Theme \ Reserved
```

### Session Manager

```
SessionManager { tmux: Tmux, rig: Rig }

Start : Polecat × Issue × Opts → Error
  1. Kill stale session
  2. Validate issue (not tombstoned)
  3. Resolve agent config
  4. Provision hooks/settings
  5. Build startup beacon
  6. Create tmux session with env:
     GT_RIG, GT_POLECAT, GT_ROLE, GT_POLECAT_PATH, GT_BRANCH, GT_RUN
  7. SessionStart hook fires → "gt prime --hook"

Stop : Polecat × Force → Error
  -- Kill tmux session, remove heartbeat

IsRunning : Polecat → Bool
  -- Check tmux session existence
```

### Heartbeat Protocol

```
TouchSessionHeartbeat : TownRoot × SessionName × State → ()
  -- Write heartbeat file with timestamp + state

IsSessionHeartbeatStale : TownRoot × SessionName × Threshold → Bool
  -- Stale if no heartbeat or age > threshold (default 3 min)

-- Witness uses heartbeat for:
-- 1. Detect dead polecats (stale heartbeat)
-- 2. Distinguish working vs idle (v2 state field)
-- 3. Detect exiting (gt done in progress)
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| AllocateAndAdd | Issue × Opts → Polecat × Error | Lock-protected, name-unique |
| Remove | Name × Force × Nuclear → Error | ZFC: reads agent bead first |
| SetState | Name × State → Error | Clears assignee on Done transition |
| SessionStart | Polecat × Issue → Error | Idempotent (kills stale first) |
| Reconcile | () → Error | Reads reality from disk |
| Heartbeat.Touch | SessionName × State → Error | Idempotent |
| Heartbeat.IsStale | SessionName × Duration → Bool | Monotone in time |

### Lean 4 Formalization Candidates

1. **State machine well-formedness**: Only valid transitions (Working→Idle, Idle→Working, Idle→Nuked)
2. **Name pool conservation**: |Allocated ∪ Available| = |Theme| (no names lost or duplicated)
3. **Heartbeat monotonicity**: Staleness is monotone (once stale, stays stale until touched)
4. **Session-at-most-one**: At most one tmux session per polecat name
5. **ZFC correctness**: Remove reads cleanup status before destroying state

---

## 6. Package: hooks (Session Hooks)

**Files**: hooks configuration, resolution, defaults

### Core Abstraction

Hooks are shell commands that fire at lifecycle events. They implement the
**prime-on-start** pattern: every new session runs `gt prime --hook` to inherit
hooked work.

### Type Algebra

```
HooksConfig {
  PreToolUse PostToolUse: List HookEntry,
  SessionStart Stop PreCompact: List HookEntry,
  UserPromptSubmit: List HookEntry,
  WorktreeCreate WorktreeRemove: List HookEntry
}

HookEntry { Matcher: String, Hooks: List Hook }
Hook { Type: String, Command: String }
-- Type is always "command"
```

### Hook Resolution (Layered Merge)

```
ComputeExpected(target) → HooksConfig
  1. DefaultBase()           -- gt prime --hook on SessionStart
  2. Merge role defaults     -- crew: handoff on PreCompact
  3. Merge on-disk base      -- user customizations
  4. Merge role overrides    -- witness/deacon: block patrol formulas
  5. Merge rig+role overrides
```

### Default Hooks

| Event | Default Command | Purpose |
|-------|----------------|---------|
| SessionStart | `gt prime --hook` | Inherit hooked work on session start |
| PreCompact | `gt prime --hook` | Re-inject context before compaction |
| UserPromptSubmit | `gt mail check --inject` | Check for incoming mail |
| PreCompact (crew) | `gt handoff --cycle --reason compaction` | Preserve crew session across compaction |

### Lean 4 Formalization Candidates

1. **Hook resolution is a fold**: ComputeExpected = foldl merge defaultBase [role, disk, roleOverride, rigRoleOverride]
2. **SessionStart always fires prime**: DefaultBase guarantees gt prime --hook on every session start

---

## 7. Package: beads (Issue Tracking Data Plane)

**Files**: beads.go, agent.go, molecule.go, routes.go, types.go, + ~10 files

### Core Abstraction

Beads wraps the `bd` CLI to provide typed Go operations on the issue tracking
database. Issues are the universal data structure — work items, agent identity,
molecules, mail messages, and configuration all live as beads issues.

### Type Algebra

```
Issue {
  ID Title Description: String,
  Status: IssueStatus, Priority: Nat, Type: String,
  Labels: List String,
  CreatedAt UpdatedAt ClosedAt CreatedBy: String,
  Assignee Parent: String,
  Children DependsOn Blocks BlockedBy: List String,
  Ephemeral: Bool, HookBead AgentState: String
}

IssueStatus = Open | InProgress | Hooked | Pinned | Closed | Tombstone | Blocked
-- Invariant: IsTerminal(s) ↔ s ∈ {Closed, Tombstone}
-- Invariant: IsAssigned(s) ↔ s ∈ {Hooked, InProgress}

AgentState = Spawning | Working | Running | Idle | Done | Stuck | Escalated | AwaitingGate | Nuked
-- Invariant: IsActive(s) ↔ s ∈ {Working, Running, Spawning}
-- Invariant: ProtectsFromCleanup(s) ↔ s ∈ {Stuck, AwaitingGate}
```

### Issue as Universal Data Type

All Gas Town entities are encoded as issues with type-discriminating labels:

| Entity | Label | Distinguishing Fields |
|--------|-------|----------------------|
| Work item | `gt:task` / `gt:bug` / `gt:feature` | Standard issue fields |
| Agent identity | `gt:agent` | AgentState, HookBead, CleanupStatus in description |
| Molecule template | `gt:molecule` | Steps in description or children |
| Mail message | `gt:message` | From/To/Thread in labels |
| Merge request | `gt:merge-request` | Branch, MR metadata |
| Rig identity | (custom) | Rig config in labels |

### Core Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| Create | CreateOptions → Issue × Error | Auto-generates ID |
| CreateWithID | String × CreateOptions → Issue × Error | Deterministic ID (for agents, rigs) |
| Show | String → Issue × Error | Partial (may not exist) |
| Update | String × UpdateOptions → Error | Partial update (preserves unset) |
| Close | List String → Error | Respects dependency blocks |
| ForceClose | String × List String → Error | Bypasses dependency checks |
| Release | String → Error | InProgress → Open, clears assignee |
| List | ListOptions → List Issue | Filtered, sorted |
| Ready | () → List Issue | Unblocked issues only |
| Blocked | () → List Issue | Issues with unsatisfied deps |
| AddDependency | String × String → Error | issue depends-on target |
| Sync | () → Error | Merge with remote |

### Molecule Algebra

```
MoleculeStep {
  Ref Title Instructions: String,
  Needs: List String, WaitsFor: List String,
  Tier: String, Type: String,
  Backoff: Option BackoffConfig
}

BackoffConfig { Base: String, Multiplier: Nat, Max: String }

-- Two formats (format bridge):
-- Old: markdown in description (## Step: ref ... Needs: dep1, dep2)
-- New: child issues as templates (DependsOn edges)

ParseMoleculeSteps : String → List MoleculeStep × Error
InstantiateMolecule : Context × Issue × Issue × Opts → List Issue × Error
  -- Creates child issues from template, wires dependencies
  -- Atomic: all-or-nothing via bd CLI
  -- Provenance: each step tagged with instantiated_from metadata
```

### Agent Bead Operations

```
CreateOrReopenAgentBead : String × String × String → Issue × Error
  -- Create or reset agent bead (lock-protected)

ResetAgentBeadForReuse : String × String → Error
  -- Clear hook, reset state for next session

UpdateAgentHook : String × String → Error
  -- Pin work to agent's hook slot

ClearAgentHook : String → Error
  -- Unpin hook
```

### Lean 4 Formalization Candidates

1. **Issue as universal type**: All entities embed in Issue with label discrimination
2. **Close respects dependencies**: Close(x) fails if ∃ y. y depends-on x ∧ ¬IsTerminal(y)
3. **ForceClose bypasses**: ForceClose ignores dependency check (escape hatch)
4. **Release is left-inverse of claim**: Release(Claim(x)) = x (returns to Open)
5. **Molecule instantiation preserves DAG**: Step dependencies in template → issue dependencies in instance

---

## 8. Package: mail (Agent Communication)

**Files**: types.go, mailbox.go, router.go

### Core Abstraction

Mail is a communication system built atop beads — messages are issues with routing
metadata encoded in labels. Three routing patterns: direct, queue, and channel.

### Type Algebra

```
Message {
  ID From To Subject Body: String,
  Timestamp: Time, Read: Bool,
  Priority: Priority, Type: MessageType,
  Delivery: Delivery,
  ThreadID ReplyTo: String, Pinned Wisp: Bool,
  CC: List String,
  Queue Channel: String,
  ClaimedBy: String, ClaimedAt: Option Time,
  DeliveryState: String, DeliveryAckedBy: String, DeliveryAckedAt: Option Time
}

Priority = Low | Normal | High | Urgent
  -- Maps to beads: Low→3, Normal→2, High→1, Urgent→0

MessageType = Task | Scavenge | Notification | Reply

Delivery = Queue | Interrupt
  -- Queue: polled via gt mail check
  -- Interrupt: injected into session
```

### Routing Algebra

Three mutually exclusive routing modes:

```
Routing = Direct To | QueueRoute Queue | Broadcast Channel
-- Invariant: exactly one of {To, Queue, Channel} is non-empty
-- Validated by Message.Validate()
```

### Address Normalization (Postel's Law)

```
normalize : String → String
  overseer           → overseer
  mayor / mayor/     → mayor/
  gastown/polecats/x → gastown/x
  gastown/crew/x     → gastown/x
```

### Two-Phase Delivery Protocol

```
Phase 1 (Send):
  Create message in beads → add label "delivery:pending"

Phase 2 (Acknowledge):
  add "delivery-acked-by:<identity>"
  add "delivery-acked-at:<timestamp>"
  add "delivery:acked"

-- Labels added sequentially for crash safety
-- Idempotent retry: reuses timestamp if sole acker

State machine: [none] → pending → acked
```

### Mailbox Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| List | () → List Message × Error | Sorted: priority DESC, timestamp DESC |
| Get | String → Message × Error | Partial (may not exist) |
| Check | CheckOpts → List Message × Error | Filtered query |
| Send | Message → Error | Creates bead + routing labels |
| Claim | Message × String → Error | Queue messages only, at-most-one |
| Acknowledge | String × String → Error | Two-phase, idempotent |

### Identity Variants in Mailbox Queries

```
List() queries across identity variants:
  "gastown/polecats/Toast"    -- Exact match
  "gastown/Toast"             -- Normalized (without polecats/)
  "gastown"                   -- Rig-level broadcast
```

### Lean 4 Formalization Candidates

1. **Routing exclusivity**: Exactly one of {To, Queue, Channel} is non-empty
2. **Two-phase delivery**: pending → acked is the only valid transition
3. **Claim at-most-one**: Only one agent can claim a queue message
4. **Priority ordering**: Mailbox.List returns messages in priority × recency order
5. **Acknowledge idempotency**: Ack(x); Ack(x) ≡ Ack(x)

---

## 9. Cross-Package Algebraic Structure

### The Intent-to-Execution Pipeline

Though the packages don't import each other directly (except through cmd), they form
a conceptual pipeline:

```
1. CLI.Parse(args)               → Command (user intent)
2. Role.Detect(env, cwd)         → RoleInfo (agent identity)
3. Rig.Resolve(role.Rig)         → Rig (execution scope)
4. Beads.Route(prefix)           → BeadsDir (data plane target)
5. Hook.Attach(agent, bead)      → HookState (work binding)
6. Polecat.Start(issue)          → Session (execution context)
7. Prime.Inject(session)         → InjectedEnv (context delivery)
8. [Agent works]
9. Done.Submit(branch, status)   → MR | Escalation (completion)
10. Mail.Notify(witness, result) → Delivery (outcome reporting)
```

### Shared Algebraic Patterns

**1. Identity Isomorphisms** (appear in rig, session, beads, mail):
```
AgentID ↔ Address ↔ SessionName ↔ BeaconAddress
-- Four representations of the same entity
-- ParseAddress ∘ Address = id (round-trip)
-- ParseSessionName ∘ SessionName = id (round-trip)
```

**2. Prefix-Based Routing** (rig, beads, mail):
```
rig.PrefixToRig(prefix)     : prefix → rig name
beads.Route(prefix)          : prefix → beads directory
mail.normalize(address)      : address → canonical identity
-- All three use prefix extraction as the routing key
```

**3. At-Most-One Semantics** (hook, polecat, mail):
```
hook: one bead per agent hook
polecat: one tmux session per polecat name
mail.claim: one claimer per queue message
```

**4. Idempotent Operations** (across all packages):
```
prime: safe to call multiple times
hook.status: pure query
done checkpoint: resumes from last completed stage
heartbeat.touch: overwrites previous
wasteland.join: ON DUPLICATE KEY UPDATE
agent bead.create: CreateOrReopen handles existing
```

**5. ZFC (Zero-Failure Computing)** (polecat, beads):
```
polecat.Remove: reads cleanup status from bead before destroying worktree
beads.Ready: filters from full list (never assumes state)
namepool.Reconcile: reads reality from filesystem, not cached state
```

**6. Durability Boundaries** (hook, mail, beads):
```
hook: survives session death (stored in agent bead)
mail: two-phase delivery survives crash (label sequence)
beads: Dolt-backed (version-controlled state)
-- All critical state survives process death
```

### Category-Theoretic View

The subsystem can be viewed as a category where:
- **Objects**: Agent states (Naked, Hooked, Working, Idle, Nuked)
- **Morphisms**: Operations that transform agent state

Key functors:
```
CLI → Intent          (parsing user command)
Intent → Identity     (role detection)
Identity → Scope      (rig resolution)
Scope → State         (beads routing)
State → Session       (polecat lifecycle)
Session → Event       (mail notification)
```

The composition `CLI → ... → Event` is the full intent-to-execution pipeline.

### Conservation Laws

1. **Hook exclusivity**: Each agent has at most one hooked bead; each bead is hooked by at most one agent
2. **Name pool conservation**: |allocated| + |available| + |reserved| = |theme_size|
3. **Routing injectivity**: Each prefix maps to exactly one beads directory
4. **Delivery state monotonicity**: pending → acked (never reverses)
5. **Session uniqueness**: At most one active session per polecat name

---

## 10. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Package | Target | Why |
|---------|--------|-----|
| rig | Agent ID algebra (parse/format round-trips) | Isomorphism proofs, prefix bijectivity |
| polecat | Lifecycle state machine | Well-defined states, transition constraints |
| beads | Issue as universal type + dependency algebra | Subtyping via labels, DAG properties |
| cmd (done) | Checkpoint-resumable completion | Idempotency, at-most-once MR submission |

### Medium Priority (simpler structure, fewer invariants)

| Package | Target | Why |
|---------|--------|-----|
| mail | Two-phase delivery protocol | Crash recovery, label ordering |
| hooks | Resolution as layered fold | Merge algebra, deterministic computation |
| cmd (role) | Role detection meet operation | Information lattice, mismatch detection |

### Low Priority (mostly plumbing)

| Package | Target | Why |
|---------|--------|-----|
| wasteland | Join idempotency | Simple DoltHub protocol |
| boot | Watchdog triage | Straightforward decision tree |
| wisp | Ephemeral flag | Single boolean property |

### Cross-Cutting Theorems

1. **Identity isomorphism**: The four representations of agent identity (ID, Address, SessionName, BeaconAddress) are isomorphic — round-trip through any pair returns the original
2. **Prefix routing is injective**: No two rigs share a prefix; routing is deterministic
3. **Hook-done lifecycle**: prime(hook(done(x))) = prime(hook(x)) — completing work and re-priming returns to the same ready state
4. **Durability hierarchy**: Beads state ⊃ Hook state ⊃ Session state — each level survives strictly more failure modes
5. **Issue universality**: Every Gas Town entity can be encoded as an Issue with label-based type discrimination, and operations on Issue respect the type discipline
