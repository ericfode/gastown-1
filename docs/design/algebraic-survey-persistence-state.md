# Algebraic Survey: Persistence & State Subsystem

**Subsystem**: Persistence & State
**Packages**: `doltserver`, `beads` (routing/redirect/types/status), `rig` (config/manager), `state`, `connection`
**Purpose**: Manages HOW data is stored and accessed — Dolt server lifecycle, database
routing (which agent sees which DB), the redirect/metadata.json mechanism,
beads_prefix mapping, and state consistency invariants across agent boundaries.

## 1. Subsystem Overview

The Persistence & State subsystem answers: given a set of agents in different
worktrees and rigs, how do they all share a consistent view of the beads database?

```
Config (identity)    Rig (registration)    Connection (abstraction)
       ↓                    ↓                       ↓
   DoltServer ←──── Routing ←──── Redirect ←──── Agent
   (lifecycle)     (prefix→DB)    (.beads/)      (workdir)
       ↓
    State (XDG)
```

### Package Dependency Graph (within subsystem)

```
doltserver ←── beads, style, util (core server management)
beads      ←── config, constants  (routing, redirect, types)
rig        ←── config, beads, wisp, git, doltserver (rig lifecycle)
state      ←── util              (XDG state persistence)
connection ←── (no internal deps, leaf package)
```

**Key finding**: The subsystem has a layered architecture: `connection` is the abstract
interface, `doltserver` is the concrete implementation, `beads` provides the routing
layer, `rig` orchestrates setup, and `state` handles XDG machine-level persistence.
The critical coupling is between `rig/manager.go` and `doltserver/doltserver.go` —
both contain database resolution logic (FindRigBeadsDir, bdDatabaseExists) that must
stay in sync.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| doltserver | cmd/dolt_*, daemon, doctor, dog/compactor, dog/reaper, rig/manager |
| beads (routing) | cmd/*, polecat, witness, refinery, crew, mail, convoy |
| rig | cmd/rig_*, cmd/install, cmd/up, daemon, witness, polecat |
| state | cmd/install, cmd/up, cmd/down, cmd/status |
| connection | rig/manager, witness, polecat (local + remote execution) |

---

## 2. Package: doltserver

**Files**: doltserver.go (3214 lines), sync.go, rollback.go, wisps_migrate.go, wl_commons.go, dolthub.go

### Type Algebra

The doltserver package manages a **single shared Dolt SQL server** per town serving
multiple databases. The core types form a configuration-state-health triple:

```
Config {
  TownRoot DataDir LogFile PidFile Host User Password LogLevel: String,
  Port MaxConnections ReadTimeoutMs WriteTimeoutMs: Nat
}

State {
  Running: Bool, PID Port: Nat,
  StartedAt: Time, DataDir: String,
  Databases: List String
}

HealthMetrics {
  Connections MaxConnections: Nat, ConnectionPct: Float,
  DiskUsageBytes: Int, DiskUsageHuman: String,
  QueryLatency: Duration, ReadOnly Healthy: Bool,
  Warnings: List String
}
```

### Server State Machine

```
              ┌──────────────┐
              │   Stopped    │
              └──────┬───────┘
                     │ Start (multi-step)
                     ▼
              ┌──────────────┐
              │   Starting   │──── WaitForReady (poll TCP + SQL)
              └──────┬───────┘
                     │ Ready (SQL responds)
                     ▼
              ┌──────────────┐
              │   Running    │──── HealthMetrics (continuous)
              │              │──── IsRunning (PID + port + data-dir)
              └──────┬───────┘
                     │ Stop (SIGTERM → SIGKILL after 10s)
                     ▼
              ┌──────────────┐
              │   Stopped    │
              └──────────────┘
```

**Start is a 15-step orchestrated operation:**
1. Check port availability (or kill imposters)
2. Ensure data directory exists
3. Ensure Dolt identity (git config → dolt config)
4. Write server config.yaml
5. Build dolt sql-server command
6. Start process (background)
7. Write PID file
8. Wait for TCP reachability
9. Wait for SQL readiness (SELECT 1)
10. Write state file
11. Verify all expected databases are served
12. Create missing databases
13. Wait for catalog registration
14. Update state with database list
15. Save final state

### IsRunning Detection (4-layer verification)

```
IsRunning(townRoot) =
  if remote → TCP reachability test
  else
    1. PID file → process alive → port match → data-dir match → process args match
    2. Port scan → findDoltServerOnPort → data-dir verify
    3. TCP reachability (non-default port only)
    4. Fall-through: not running
```

**Invariant**: `IsRunning` never claims another town's Dolt server. It verifies
data-dir from both state file AND process args to guard against PID reuse.

### Config Resolution Priority

```
Port:  config.yaml > GT_DOLT_PORT env > DefaultPort (3307)
Host:  GT_DOLT_HOST env > "" (localhost)
User:  GT_DOLT_USER env > "root"
Pass:  GT_DOLT_PASSWORD env > "" (none)
Level: GT_DOLT_LOGLEVEL env > "warning"
```

### Database Management

```
RigDatabaseDir : TownRoot × RigName → Path
  -- ~/gt/.dolt-data/<rigName>/

InitRig : TownRoot × RigName → (ServerWasRunning × Created × Error)
  -- Creates database directory, runs dolt init, registers in catalog

DatabaseExists : TownRoot × RigName → Bool
  -- Checks .dolt-data/<rigName>/ exists

ListDatabases : TownRoot → List String × Error
  -- SHOW DATABASES filtered by !IsSystemDatabase

VerifyDatabases : TownRoot → (Served × Missing) × Error
  -- Cross-references SQL catalog with filesystem databases

RemoveDatabase : TownRoot × DBName × Force → Error
  -- DROP DATABASE (requires force for non-empty DBs with user tables)
```

**System databases (excluded from listings):**
```
systemDatabases = {information_schema, mysql, dolt_cluster, wl_commons}
```

### Metadata Management

```
MetadataJSON = {
  database: "dolt",
  backend: "dolt",
  dolt_mode: "server",
  dolt_database: String,        -- rig name (e.g., "gastown")
  dolt_server_host: String,     -- from Config.EffectiveHost()
  dolt_server_port: Nat         -- from Config.Port
}

EnsureMetadata : TownRoot × RigName → Error
  -- Idempotent: reads existing, patches only changed fields, atomic write
  -- Thread-safe via per-path mutex (flock for inter-process, sync.Mutex for goroutines)

EnsureAllMetadata : TownRoot → (Updated × Errors)
  -- Iterates all databases, calls EnsureMetadata for each
```

**Split-brain invariant**: EnsureAllMetadata ensures all worktrees point to the
SAME Dolt server instance. Without it, worktrees can each have their own isolated
database, breaking the all-on-main concurrency model.

### Sync (Remote Push)

```
SyncOptions { Force DryRun: Bool, Filter: String }

SyncResult { Database Remote: String, Pushed Skipped DryRun: Bool, Error }

SyncDatabases : TownRoot × SyncOptions → List SyncResult
  -- Iterates databases, finds remotes, runs dolt push
  -- Server must be stopped during push (exclusive access to data dir)

CommitWorkingSet : DBDir → Error
  -- DOLT_ADD + DOLT_COMMIT for pending changes before push
```

### Health Monitoring

```
Healthy(metrics) ≡ ¬metrics.ReadOnly
                  ∧ metrics.ConnectionPct < 80
                  ∧ metrics.QueryLatency < 1s

GetHealthMetrics : TownRoot → HealthMetrics
  -- Partial: returns what it can even if some checks fail

CheckReadOnly : TownRoot → (Bool × Error)
  -- Probes by attempting a test write (CREATE TABLE, INSERT, DROP)

RecoverReadOnly : TownRoot → Error
  -- Restart server to recover from read-only state
```

### Orphan Detection

```
OrphanedDatabase { Name DatabaseDir: String, HasUserTables: Bool, SizeBytes: Int }

FindOrphanedDatabases : TownRoot → List OrphanedDatabase × Error
  -- Compares filesystem databases against collectReferencedDatabases
  -- Referenced = union of:
  --   1. metadata.json dolt_database fields from all beads dirs
  --   2. rigs.json rig names
  --   3. "hq" (always referenced)

BrokenWorkspace { RigName BeadsDir Problem Suggestion: String }

FindBrokenWorkspaces : TownRoot → List BrokenWorkspace
  -- Detects: missing metadata.json, wrong dolt_database, stale server config

RepairWorkspace : TownRoot × BrokenWorkspace → (Action × Error)
  -- Rewrites metadata.json with correct values
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| Start | TownRoot → Error | Not idempotent (fails if already running) |
| Stop | TownRoot → Error | Idempotent (no-op if not running) |
| IsRunning | TownRoot → (Bool × PID × Error) | Pure observation (no side effects) |
| EnsureMetadata | TownRoot × RigName → Error | Idempotent (no-op if already correct) |
| InitRig | TownRoot × RigName → ... | Idempotent (returns created=false if exists) |
| ListDatabases | TownRoot → List String | Deterministic snapshot of SQL catalog |
| FindOrphanedDatabases | TownRoot → List Orphan | Monotone in databases (more DBs → more potential orphans) |
| doltSQLWithRetry | TownRoot × DB × Query → Error | At-most-3 attempts with exponential backoff |
| KillImposters | TownRoot → Error | Idempotent (kills stale dolt processes on our port) |

### Lean 4 Formalization Candidates

1. **Start precondition**: Start succeeds iff port is available AND data dir exists
2. **IsRunning soundness**: Returns true only when PID, port, AND data-dir all match our town
3. **EnsureMetadata idempotency**: EnsureMetadata(t,r); EnsureMetadata(t,r) = EnsureMetadata(t,r)
4. **Orphan detection completeness**: Every unreferenced database is found by FindOrphanedDatabases
5. **Health monotonicity**: ReadOnly state cannot self-recover (requires explicit restart)

---

## 3. Package: beads (routing subsystem)

**Files**: routes.go, beads_redirect.go, beads_types.go, status.go, beads_rig.go

### Core Abstraction

The routing subsystem provides a **two-level indirection** that maps bead IDs to
database connections. Level 1 (prefix routing) maps bead ID prefixes to rig paths.
Level 2 (redirect files) maps worktree `.beads/` directories to shared databases.

### Type Algebra

```
Route { Prefix Path: String }
  -- Prefix includes trailing hyphen: "gt-", "bd-", "hq-"
  -- Path is relative to town root: "gastown/mayor/rig", ".", "beads/mayor/rig"

AgentState = Spawning | Working | Done | Stuck | Escalated | Idle | Running | Nuked | AwaitingGate

IssueStatus = Open | Closed | InProgress | Tombstone | Blocked | Pinned | Hooked
```

### Routing Resolution Chain

The full path from bead ID to database connection:

```
BeadID → ExtractPrefix → GetRigPathForPrefix → ResolveBeadsDir → metadata.json → Dolt DB
 "gt-bow"  → "gt-"      → "/home/.../gastown"  → follow redirect  → dolt_database  → gastown
```

**ExtractPrefix** (total function with fallback):
```
ExtractPrefix(beadID) =
  let idx = firstIndexOf(beadID, '-')
  if idx ≤ 0 then ""
  else beadID[0..idx+1]
```

### Prefix Registry (JSONL-based)

```
Routes = List Route  -- stored in .beads/routes.jsonl (one JSON per line)

LoadRoutes : BeadsDir → List Route × Error
  -- Skips empty lines and comments (#)
  -- Tolerates malformed lines (warns and skips)

AppendRoute : TownRoot × Route → Error
  -- Upsert: updates path if prefix exists, appends otherwise
  -- Atomic write via temp file + rename

RemoveRoute : TownRoot × Prefix → Error
  -- Filter and rewrite

FindConflictingPrefixes : BeadsDir → Map Prefix (List Path) × Error
  -- Returns prefixes with more than one path
```

**Invariant**: Each prefix maps to exactly one path.
`|FindConflictingPrefixes(dir)| = 0` for a well-formed routes file.

### Prefix ↔ Rig Bijection

```
GetRigPathForPrefix : TownRoot × Prefix → Path
GetPrefixForRig     : TownRoot × RigName → Prefix
GetRigNameForPrefix : TownRoot × Prefix → RigName
```

These form a **partial bijection** on the registered domain:
```
∀ r ∈ registeredRigs:
  GetRigNameForPrefix(GetPrefixForRig(r)) = r
  GetPrefixForRig(GetRigNameForPrefix(p)) = p
```

### Redirect Resolution

The redirect mechanism allows worktrees (polecats, crew, refinery) to share a
rig's beads database without each having their own:

```
ResolveBeadsDir : WorkDir → BeadsDir
  1. Check workDir/.beads/redirect
  2. If absent → return workDir/.beads (local database)
  3. If present → read target path
  4. Resolve relative to workDir (not .beads/)
  5. Detect circular redirects (resolved = original → remove errant file)
  6. Follow chains with depth limit (max 3)

ComputeRedirectTarget : TownRoot × WorktreePath → RedirectPath × Error
  -- Canonical function for computing expected redirects
  -- Safety: refuses to create redirect in mayor/rig (prevents circular chains)
  -- Validates: redirect stays within town root
  -- Validates: redirect target directory exists
```

**Redirect chain resolution** (bounded recursion):
```
resolveBeadsDirWithDepth(beadsDir, maxDepth) =
  if maxDepth ≤ 0 → beadsDir (warn: chain too deep)
  if no redirect file → beadsDir (terminal)
  else → resolveBeadsDirWithDepth(resolvedTarget, maxDepth - 1)
```

**Invariant**: Redirect chains terminate within 3 hops.
`resolveBeadsDirWithDepth(_, 3)` always returns a concrete directory.

### Redirect Topology

```
Town Root
├── .beads/                    ← Town beads (hq-* prefix, no redirect)
│   └── routes.jsonl           ← Prefix→rig mapping
├── gastown/
│   ├── .beads/redirect → "mayor/rig/.beads"   ← Rig-root redirect
│   ├── mayor/rig/.beads/      ← Canonical database location (tracked in git)
│   ├── polecats/slit/.beads/redirect → "../../.beads"  ← Worker redirect
│   ├── crew/max/.beads/redirect → "../../.beads"       ← Crew redirect
│   └── refinery/rig/.beads/redirect → "../../.beads"   ← Refinery redirect
```

All worker redirects resolve through the rig-root redirect to the canonical
`mayor/rig/.beads` location. The `bd` CLI doesn't support redirect chains,
so `ComputeRedirectTarget` shortcuts directly to the final destination.

### Agent State Machine

```
         ┌────────────┐
         │  Spawning   │───────────────────┐
         └──────┬──────┘                   │
                │                          │
                ▼                          ▼
         ┌────────────┐             ┌────────────┐
         │  Working   │────────────▶│   Done     │
         │  / Running │             └────────────┘
         └──────┬──────┘                   ▲
                │                          │
         ┌──────┼──────┐                   │
         ▼      ▼      ▼                   │
    ┌────────┐ ┌────────┐ ┌──────────────┐ │
    │ Stuck  │ │ Idle   │ │AwaitingGate  │ │
    └───┬────┘ └───┬────┘ └──────┬───────┘ │
        │          │             │          │
        ▼          └─────────────┴──────────┘
    ┌────────────┐
    │ Escalated  │
    └──────┬─────┘
           │
           ▼
    ┌────────────┐
    │   Nuked    │
    └────────────┘
```

**State predicates:**
```
ProtectsFromCleanup(s) ≡ s ∈ {Stuck, AwaitingGate}
IsActive(s) ≡ s ∈ {Working, Running, Spawning}
```

### Issue Status Machine

```
         ┌────────┐
         │  Open  │◀──────────── (created)
         └───┬────┘
             │
     ┌───────┼───────────┐
     ▼       ▼           ▼
┌─────────┐ ┌──────┐ ┌──────────┐
│InProgress│ │Hooked│ │ Blocked  │
└────┬────┘ └──┬───┘ └────┬─────┘
     │         │          │
     └────┬────┘          │
          ▼               │
     ┌─────────┐          │
     │ Closed  │◀─────────┘
     └────┬────┘
          │
          ▼
     ┌──────────┐
     │Tombstone │
     └──────────┘

     ┌────────┐
     │ Pinned │ (permanent, no transitions out)
     └────────┘
```

**Status predicates:**
```
BlocksRemoval(s) ≡ s = Open
IsTerminal(s) ≡ s ∈ {Closed, Tombstone}
IsAssigned(s) ≡ s ∈ {Hooked, InProgress}
```

### Custom Types & Statuses Sentinel System

```
EnsureCustomTypes : BeadsDir → Error
EnsureCustomStatuses : BeadsDir → Error
```

Both use a **two-level caching strategy**:
```
Cache = InMemory (map[string]bool, per-process) × OnDisk (sentinel file, cross-process)

Lookup(beadsDir) =
  1. In-memory cache hit → return (fast path)
  2. Sentinel file matches current list → cache + return (fast path)
  3. Sentinel stale or missing → configure via bd CLI → write sentinel → cache

Staleness: sentinel stores configured types/statuses as a string.
When the types list changes (e.g., new release), sentinel won't match → re-configure.
```

**Invariant**: EnsureCustomTypes is idempotent and thread-safe (mutex-protected).

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| ExtractPrefix | String → String | Total (empty string for invalid input) |
| ResolveBeadsDir | WorkDir → BeadsDir | Terminates within depth 3 |
| ComputeRedirectTarget | TownRoot × Path → String × Error | Pure (deterministic from filesystem state) |
| SetupRedirect | TownRoot × Path → Error | Idempotent (same redirect created) |
| LoadRoutes | BeadsDir → List Route | Tolerant (skips malformed lines) |
| AppendRoute | TownRoot × Route → Error | Upsert (idempotent on prefix) |
| EnsureCustomTypes | BeadsDir → Error | Idempotent, thread-safe, auto-healing |
| IsTerminal | IssueStatus → Bool | Pure predicate |
| IsActive | AgentState → Bool | Pure predicate |

### Lean 4 Formalization Candidates

1. **Redirect termination**: resolveBeadsDirWithDepth always terminates (bounded by maxDepth)
2. **Circular redirect safety**: Self-referencing redirects are detected and removed
3. **Prefix bijection**: GetPrefixForRig and GetRigNameForPrefix are partial inverses
4. **Route uniqueness**: After AppendRoute, each prefix maps to exactly one path
5. **Sentinel idempotency**: EnsureCustomTypes(dir); EnsureCustomTypes(dir) = EnsureCustomTypes(dir)

---

## 4. Package: rig (persistence setup)

**Files**: manager.go (1581 lines), types.go, config.go, overlay.go, setuphooks.go

### Type Algebra

```
Rig {
  Name Path GitURL PushURL LocalRepo: String,
  Config: Option BeadsConfig,
  Polecats Crew: List String,
  HasWitness HasRefinery HasMayor: Bool
}

RigConfig {
  Version: Nat,
  DefaultBranch PolecatBranchTemplate AgentCommand: String,
  Beads: Option BeadsConfig
}

BeadsConfig {
  Prefix: String  -- e.g., "gt", "bd"
}

ConfigSource = Wisp | Bead | Town | System | Blocked | None

ConfigResult { Value: Any, Source: ConfigSource }
```

### Property Layer Lookup (Config Resolution)

The config system implements a **layered override** pattern:

```
GetConfigWithSource(key) =
  Layer 1: Wisp (transient, local) → if blocked → SourceBlocked
  Layer 2: Bead (rig identity labels) → SourceBead
  Layer 3: Town (settings/config.json) → SourceTown [planned]
  Layer 4: System (compiled-in defaults) → SourceSystem
  None found → SourceNone
```

**Override semantics** (default): First non-nil wins.
**Stacking semantics** (opt-in): Values from all layers are summed.

```
StackingKeys = {priority_adjustment}

GetIntConfig(key) =
  if key ∉ StackingKeys → first non-nil value
  else → SystemDefault + beadAdjustment + wispAdjustment
```

### Beads Prefix Derivation

When adding a new rig, the prefix is derived from the rig name:

```
deriveBeadsPrefix : RigName → Prefix
  -- Split compound word (snake_case, kebab-case, camelCase)
  -- Take first letter of each part
  -- Validate against prefixRe: ^[a-zA-Z][a-zA-Z0-9-]{0,19}$
  -- Examples: "my-project" → "mp", "gastown" → "gt", "pixelforge" → "pf"
```

**Validation invariant**: Both `beads.prefixRe` (in beads package) and
`rig.beadsPrefixRegexp` (in rig package) enforce the same pattern.
These are intentionally duplicated to avoid a circular import.

### InitBeads Decision Tree

```
InitBeads(rigPath, prefix, rigName):
  if mayor/rig/.beads exists:
    → Create redirect file: .beads/redirect → "mayor/rig/.beads"
    → Return (tracked beads mode)
  else:
    → Create .beads/ directory
    → Run bd init --server --prefix <prefix>
    → Configure custom types
    → Set issue_prefix config
    → Remove orphan database (beads_<prefix> → should be <rigName>)
    → Ensure config.yaml
    → Run bd migrate --update-repo-id
```

**Orphan prevention**: `bd init --prefix` creates a database named `beads_<prefix>`,
but Gas Town uses `<rigName>` as the database name. InitBeads removes the orphan
immediately after init.

### Rig Registration

```
AddRig : Manager × AddRigOptions → Rig × Error
  -- 20+ step orchestrated operation including:
  -- 1. Clone repo
  -- 2. Create agent directories (polecats/, crew/, refinery/rig, witness/, mayor/rig)
  -- 3. InitBeads
  -- 4. InitRig (doltserver database)
  -- 5. Register in rigs.json
  -- 6. Register prefix in routes.jsonl
  -- 7. EnsureMetadata
  -- 8. Setup redirects for all agent worktrees
  -- 9. Create agent beads (witness, refinery)
  -- 10. Seed patrol molecules
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| DiscoverRigs | Manager → List Rig | Deterministic from rigs.json |
| AddRig | Manager × Opts → Rig × Error | Not idempotent (fails if exists) |
| InitBeads | Manager × Path × Prefix × Name → Error | Branching (tracked vs local) |
| GetConfig | Rig × Key → Value | Layered override, deterministic |
| GetIntConfig | Rig × Key → Int | Stacking or override per key |
| BeadsPath | Rig → Path | Always returns rig root (not mayor) |
| DefaultBranch | Rig → String | Fallback "main" |

### Lean 4 Formalization Candidates

1. **Config layer ordering**: Wisp overrides Bead overrides Town overrides System
2. **Stacking additivity**: GetIntConfig for stacking keys = base + bead + wisp
3. **Prefix derivation determinism**: Same rig name always produces same prefix
4. **InitBeads branching**: Tracked mode iff mayor/rig/.beads exists

---

## 5. Package: state

**Files**: state.go (181 lines)

### Type Algebra

```
State {
  Enabled: Bool, Version MachineID ShellIntegration: String,
  InstalledAt UpdatedAt LastDoctorRun: Time
}
```

### XDG Path Resolution

```
StateDir  = XDG_STATE_HOME/gastown  | ~/.local/state/gastown
ConfigDir = XDG_CONFIG_HOME/gastown | ~/.config/gastown
CacheDir  = XDG_CACHE_HOME/gastown  | ~/.cache/gastown
```

### Enable/Disable State Machine

```
IsEnabled =
  if GASTOWN_DISABLED=1 → false (env override)
  if GASTOWN_ENABLED=1  → true  (env override)
  else → state.Enabled from disk (default false)
```

**Priority**: Environment overrides > state file > default (false).

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| Load | () → State × Error | Pure read (no side effects) |
| Save | State → Error | Atomic write (util.AtomicWriteJSONWithPerm, 0600) |
| Enable | Version → Error | Create-or-update, sets Enabled=true |
| Disable | () → Error | Create-or-update, sets Enabled=false |
| IsEnabled | () → Bool | Pure function of (env, disk state) |
| GetMachineID | () → String | Idempotent (generates if missing) |

### Lean 4 Formalization Candidates

1. **Enable/Disable duality**: Enable; Disable; IsEnabled = false
2. **Environment override priority**: env vars always win over disk state
3. **MachineID stability**: GetMachineID returns same value after first Save

---

## 6. Package: connection

**Files**: connection.go (172 lines)

### Type Algebra

```
Connection = Interface {
  -- Identification
  Name: () → String,
  IsLocal: () → Bool,
  -- File ops (8 methods)
  ReadFile WriteFile MkdirAll Remove RemoveAll Stat Glob Exists,
  -- Command execution (3 methods)
  Exec ExecDir ExecEnv,
  -- Tmux management (5 methods)
  TmuxNewSession TmuxKillSession TmuxSendKeys TmuxCapturePane TmuxHasSession TmuxListSessions
}

FileInfo = Interface { Name Size Mode ModTime IsDir }

BasicFileInfo { FileName FileSize FileMode FileModTime FileIsDir }
  -- implements FileInfo (value type, JSON-serializable)

ConnectionError { Op Machine: String, Err: Error }
NotFoundError { Path: String }
PermissionError { Path Op: String }
```

### Error Algebra

```
Error = ConnectionError | NotFoundError | PermissionError | Other

ConnectionError wraps an inner error (Unwrap pattern)
NotFoundError and PermissionError are terminal (no wrapping)
```

### Key Properties

- `BasicFileInfo` is the **universal representation** of file info across local/remote
- `Connection` is a **free interface** — implementations exist in `connection/local.go`
  (local filesystem + process execution) and `connection/registry.go` (SSH remote)
- The interface abstracts **exactly three concerns**: filesystem, command execution, tmux

### Lean 4 Formalization Candidates

1. **FileInfo isomorphism**: FromOSFileInfo ∘ ToOSFileInfo = id (round-trip)
2. **Error hierarchy**: ConnectionError wraps exactly one inner error

---

## 7. Cross-Package Algebraic Structure

### The Persistence Pipeline

Though the packages don't form a linear pipeline, they compose into a resolution
chain orchestrated by `rig/manager.go`:

```
1. AddRig(opts)                    → Rig (identity + filesystem)
2. InitBeads(rigPath, prefix)      → .beads/ (redirect or database)
3. InitRig(townRoot, rigName)      → Dolt database in .dolt-data/
4. EnsureMetadata(townRoot, rig)   → metadata.json (server connection info)
5. AppendRoute(townRoot, route)    → routes.jsonl (prefix mapping)
6. SetupRedirect(townRoot, wt)     → .beads/redirect (worker indirection)
7. EnsureCustomTypes(beadsDir)     → sentinel file (type configuration)
```

Steps 2-7 are idempotent and can be re-run (e.g., by `bd doctor`).

### The Resolution Chain (Read Path)

```
Agent wants to access bead "gt-bow":
  1. ExtractPrefix("gt-bow")           → "gt-"
  2. LoadRoutes(townRoot/.beads)       → [{prefix:"gt-", path:"gastown/mayor/rig"}]
  3. GetRigPathForPrefix(town, "gt-")  → "/home/.../gastown/mayor/rig"
  4. ResolveBeadsDir(rigPath)          → follow redirect chain → final .beads/
  5. Read metadata.json                → {dolt_database: "gastown", port: 3307}
  6. Connect to Dolt SQL server        → USE gastown; SELECT ...
```

### Shared Algebraic Patterns

**1. Idempotent Ensure Pattern** (doltserver, beads, rig):
```
EnsureMetadata(t, r)      → idempotent, convergent
EnsureCustomTypes(dir)     → idempotent, cached, convergent
EnsureDoltIdentity()       → idempotent
SetupRedirect(t, wt)      → idempotent (overwrites with same content)
```

All "Ensure" functions converge to the correct state regardless of initial state.
They are self-healing: stale data is detected and corrected.

**2. Two-Level Caching** (beads.EnsureCustomTypes/Statuses):
```
Level 1: In-memory map (per-process, fast)
Level 2: Sentinel file (cross-process, persisted)
Invalidation: Content-addressed (sentinel stores expected value, not version)
```

**3. Atomic Write Pattern** (doltserver, state, beads.routes):
```
util.AtomicWriteJSON    → temp file + rename (crash-safe)
util.AtomicWriteFile    → temp file + rename (crash-safe)
WriteRoutes             → temp file + sync + rename (JSONL-specific)
```

All persistent writes use atomic replace to prevent partial writes from corrupting state.

**4. Fallback Resolution** (beads routing, rig config, doltserver config):
```
Routing:   routes.jsonl → config.GetRigPrefix → "gt" (default)
Config:    wisp → bead → town → system → none
DoltPort:  config.yaml → GT_DOLT_PORT → DefaultPort
BeadsDir:  mayor/rig/.beads → rig/.beads → create rig/.beads
```

All resolution chains terminate with a safe default, never an error.

**5. Filesystem as Database** (routes.jsonl, metadata.json, sentinel files):
```
routes.jsonl    : JSONL file → bidirectional prefix map
metadata.json   : JSON file → database connection info
sentinel files  : plain text → cache invalidation key
redirect files  : plain text → single path reference
state.json      : JSON file → global enable/disable toggle
```

These are the "shadow database" — configuration state that lives outside Dolt but
must stay consistent with it.

### Conservation Laws

1. **Route uniqueness**: After any sequence of AppendRoute calls, each prefix maps to exactly one path
2. **Redirect acyclicity**: Redirect chains are bounded by depth 3; circular redirects are auto-removed
3. **Metadata-server consistency**: EnsureMetadata guarantees metadata.json matches Config
4. **Database-filesystem correspondence**: VerifyDatabases detects any divergence between SQL catalog and .dolt-data/
5. **Agent state protection**: ProtectsFromCleanup(s) → s is not cleaned up by staleness patrol

### Category-Theoretic View

The subsystem can be viewed as a category where:
- **Objects**: Configuration states (filesystem layout, database catalog, metadata files)
- **Morphisms**: Setup/ensure operations that converge state

Key functors:
```
RigName → Prefix       (deriveBeadsPrefix: name algebra → prefix algebra)
Prefix → RigPath       (GetRigPathForPrefix: prefix algebra → filesystem)
RigPath → BeadsDir     (ResolveBeadsDir: filesystem → database reference)
BeadsDir → DoltDB      (metadata.json: database reference → SQL connection)
```

The composition `RigName → ... → DoltDB` is the full persistence resolution chain.

**Convergence property**: All "Ensure" morphisms are idempotent endomorphisms.
For any ensure function E and state s: E(E(s)) = E(s).
This makes the system self-healing: running all ensures from any state converges
to the canonical correct state.

---

## 8. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Package | Target | Why |
|---------|--------|-----|
| beads (routing) | Redirect resolution termination and acyclicity | Bounded recursion, circular detection |
| beads (routing) | Prefix↔Rig bijection correctness | Partial isomorphism on registered domain |
| doltserver | IsRunning soundness (4-layer verification) | Complex predicate with security implications |
| doltserver | EnsureMetadata convergence | Idempotent endomorphism, split-brain prevention |

### Medium Priority (simpler structure, useful invariants)

| Package | Target | Why |
|---------|--------|-----|
| rig | Config layer ordering and stacking | Layered override algebra |
| doltserver | Start preconditions and state machine | 15-step orchestration correctness |
| beads | Status/State machine transition validity | Finite state machine properties |
| state | Enable/Disable with environment overrides | Priority lattice |

### Cross-Cutting Theorems

1. **Convergence**: All Ensure operations are idempotent endomorphisms — the system self-heals
2. **Resolution chains terminate**: Every lookup (routing, config, redirect) has a bounded fallback chain
3. **Atomic persistence**: All state-changing writes use temp+rename — no partial states are observable
4. **Orphan detection is complete**: Every database not referenced by metadata.json or rigs.json is detected
5. **Split-brain prevention**: EnsureAllMetadata guarantees all worktrees point to the same Dolt server
