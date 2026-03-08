# Algebraic Survey: Spatial Structure

**Subsystem**: Spatial Structure
**Packages**: `workspace`, `rig`, `polecat`, `beads` (routing/redirect), `formula` (embedding),
`session` (identity/registry), `git` (worktree operations), `config` (path resolution)
**Purpose**: Defines WHERE things live — the filesystem hierarchy, worktree topology,
beads routing, formula search paths, and the relationship between git topology and agent topology.

## 1. Subsystem Overview

The Spatial Structure subsystem answers: given a multi-agent system with multiple
repositories, multiple worktrees per repository, and shared issue databases, how is
the filesystem organized and how are paths resolved?

```
Wasteland (federation)
  └── Town (workspace root)
        ├── Mayor (identity + config)
        ├── Rig₁ (managed repository)
        │     ├── Mayor/Rig (canonical clone)
        │     ├── Polecats (worker worktrees)
        │     ├── Crew (user worktrees)
        │     └── Refinery/Witness (service clones)
        └── Rig₂ ...
```

### Package Dependency Graph (within subsystem)

```
workspace   ←── config (leaf-ish: only needs config.LoadTownConfig)
rig         ←── config, beads, wisp, git, style
polecat     ←── rig, git, beads, tmux, config, session
beads       ←── config (routing: leaf package)
formula     ←── (no internal deps, leaf package)
session     ←── config (identity: prefix registry)
git         ←── (no internal deps, leaf package)
config      ←── scheduler/capacity (leaf-ish: types only)
```

**Key finding**: The spatial packages form a clear layered architecture:
- **Layer 0** (no deps): `git`, `formula`, `beads` (routing)
- **Layer 1** (config only): `workspace`, `session`, `config`
- **Layer 2** (multi-dep): `rig`
- **Layer 3** (orchestrator): `polecat`

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| workspace | cmd/*, daemon, boot, doctor, all agents |
| rig | cmd/rig, polecat, witness, refinery, crew, daemon |
| polecat | cmd/sling, witness, scheduler, daemon |
| beads (routing) | cmd/bd, beads CLI, all agents |
| formula (embed) | cmd/formula, rig.AddRig (provisioning) |
| session (identity) | polecat, witness, mail, all agents |
| git (worktree) | rig, polecat, refinery, witness |
| config | everything |

---

## 2. Package: workspace

**Files**: find.go (193 lines)

### Core Abstraction

The workspace package answers a single question: "Where is the town root?"
It implements upward directory traversal with marker detection.

### Type Algebra

The workspace has no named types — its algebra is expressed through path predicates:

```
IsTown(dir) ≡ ∃ file at dir/mayor/town.json
            ∨ ∃ dir at dir/mayor/

InWorktree(path) ≡ "/polecats/" ⊂ path ∨ "/crew/" ⊂ path
```

### Markers (Priority-Ordered)

```
PrimaryMarker   = "mayor/town.json"   -- Authoritative
SecondaryMarker = "mayor"              -- Fallback (directory)
```

### Key Operations

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| Find | Path → Option Path | Idempotent: Find(Find(p)) = Find(p) |
| FindOrError | Path → Path × Error | Total on valid filesystems |
| FindFromCwd | () → Path × Error | Find ∘ Getwd |
| FindFromCwdOrError | () → Path × Error | Fallback: Getwd ∥ GT_TOWN_ROOT |
| FindFromCwdWithFallback | () → Path × Path × Error | Recovers from deleted cwd |
| IsWorkspace | Path → Bool | PrimaryMarker ∨ SecondaryMarker |
| GetTownName | Path → String × Error | LoadTownConfig ∘ Join(_, PrimaryMarker) |

### Find Algorithm

```
Find(startDir) =
  let inWorktree = InWorktree(startDir)
  let primary = None, secondary = None
  for dir ∈ ancestors(startDir):
    if IsTown_primary(dir):
      if ¬inWorktree: return dir    -- Early return for non-worktree
      primary ← dir                  -- Record but continue (find outermost)
    if IsTown_secondary(dir):
      secondary ← dir               -- Always update (outermost wins)
  return primary ∥ secondary ∥ None
```

**Critical invariant**: When `InWorktree(path)` is true, Find continues past the
first primary match to find the outermost workspace. This prevents a rig's internal
`mayor/` directory from being mistaken for the town root.

### Fallback Chain

```
FindFromCwdWithFallback =
  Getwd() → Find()
  ∥ (Getwd fails) → GT_TOWN_ROOT env → Stat(PrimaryMarker)
  ∥ Error
```

This three-level fallback handles the case where a polecat's worktree has been
deleted while the agent is still running (e.g., nuked by Witness).

### Lean 4 Formalization Candidates

1. **Find idempotency**: Find(Find(p)) = Find(p) when Find(p) ≠ None
2. **Worktree override**: InWorktree ⇒ Find returns outermost workspace, not inner rig
3. **Primary dominance**: If PrimaryMarker and SecondaryMarker both match, Primary wins
4. **Ancestry monotonicity**: If Find(p) = Some(r), then r is an ancestor of p

---

## 3. Package: rig

**Files**: types.go (99 lines), manager.go (61K), config.go (219 lines),
overlay.go (203 lines), setuphooks.go (3.7K)

### Core Abstraction

A **Rig** is a managed git repository in the workspace. It provides the organizational
unit that binds: a git clone, a set of agent worktrees, a beads database, and a
configuration layer stack.

### Type Algebra

```
Rig {
  Name: String,                    -- Directory name (identifier)
  Path: String,                    -- Absolute path
  GitURL: String,                  -- Fetch URL
  PushURL: Option String,          -- Fork push URL (when GitURL is read-only)
  LocalRepo: Option String,        -- Reference clone path
  Config: Option BeadsConfig,      -- Rig-level beads config
  Polecats: List String,           -- Polecat names
  Crew: List String,               -- Crew worker names
  HasWitness: Bool,                -- Has witness agent
  HasRefinery: Bool,               -- Has refinery agent
  HasMayor: Bool                   -- Has mayor clone
}

RigSummary { Name: String, PolecatCount CrewCount: Nat, HasWitness HasRefinery: Bool }
```

### Agent Directory Structure (Fixed Topology)

```
AgentDirs = ["polecats", "crew", "refinery/rig", "witness", "mayor/rig"]
```

This is the canonical set of agent directories within a rig. Note the asymmetry:
- `refinery/rig` and `mayor/rig` have a `/rig` subdirectory (git clone lives there)
- `witness` has no `/rig` subdirectory (no clone needed, uses rig root)
- `polecats` and `crew` are containers for multiple worktrees

### BeadsPath Resolution

```
Rig.BeadsPath() = Rig.Path    -- Always returns rig root
```

The rig root's `.beads/` contains either:
1. A local beads database (when repo doesn't track .beads/)
2. A redirect file pointing to `mayor/rig/.beads` (when repo tracks .beads/)

This ensures beads operations never write to the user's repo clone (`mayor/rig/`).

### Configuration Layer Stack

```
ConfigSource = Wisp | Bead | Town | System | Blocked | None

GetConfig(key) : Rig → ConfigSource × Value
  Layer 1: Wisp    (.beads-wisp/config/)     -- Transient, local
  Layer 2: Bead    (rig identity bead labels) -- Persistent, key:value labels
  Layer 3: Town    (settings/config.json)     -- Town-wide defaults
  Layer 4: System  (compiled-in defaults)     -- Hardcoded fallback

Override semantics: first non-nil value wins (short-circuit)
Stacking semantics (priority_adjustment): values from all layers sum
Blocked semantics: wisp can explicitly block a key (returns nil regardless of lower layers)
```

### System Defaults

```
SystemDefaults = {
  "status"                  → "operational",
  "auto_restart"            → true,
  "auto_start_on_up"        → false,
  "max_polecats"            → 10,
  "priority_adjustment"     → 0,
  "dnd"                     → false,
  "polecat_branch_template" → ""
}
```

### Overlay System

```
CopyOverlay : RigPath × DestPath → Error
  Source: <rig>/.runtime/overlay/*
  Target: <destPath>/*
  Semantics: Non-recursive file copy, preserving permissions
  Failure: Logged as warnings, never fatal
```

The overlay system copies gitignored runtime files (`.env`, credentials) from
the rig's `.runtime/overlay/` to agent worktrees at spawn time.

### Gitignore Management

```
RequiredPatterns = [".runtime/", ".claude/", ".logs/", "__pycache__/"]

EnsureGitignorePatterns : WorktreePath → Error
  -- Appends missing patterns to .gitignore
  -- matchesGitignorePattern handles variants: "x", "x/", "/x", "/x/"
  -- Broader patterns cover narrower: ".claude/" covers ".claude/commands/"
```

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| BeadsPath | Rig → Path | Always returns Rig.Path (invariant) |
| DefaultBranch | Rig → String | Fallback: "main" |
| GetConfig | Rig × String → Value | Layer cascade: Wisp → Bead → Town → System |
| GetIntConfig | Rig × String → Int | Stacking for StackingKeys, override for others |
| GetBoolConfig | Rig × String → Bool | Parses string "true"/"1"/"yes" |
| Summary | Rig → RigSummary | Pure projection |
| CopyOverlay | Path × Path → Error | Fail-soft (warnings only) |
| EnsureGitignorePatterns | Path → Error | Idempotent (checks before appending) |

### Lean 4 Formalization Candidates

1. **Config layer precedence**: Wisp always overrides Bead overrides Town overrides System
2. **Blocked semantics**: IsBlocked(k) ⇒ GetConfig(k) = None ∀ layers
3. **Stacking commutativity**: base + beadAdj + wispAdj = base + wispAdj + beadAdj
4. **Overlay idempotency**: CopyOverlay called twice produces same result as once
5. **Gitignore idempotency**: EnsureGitignorePatterns is idempotent

---

## 4. Package: polecat

**Files**: types.go (153 lines), manager.go (84K), session_manager.go (31K),
namepool.go (21K), heartbeat.go (4.8K)

### Core Abstraction

A **Polecat** is a persistent worker agent sandbox. It survives work completion and
can be reused. The key insight is that the polecat's **identity** (CV chain, mailbox,
work history) and **sandbox** (git worktree) persist across sessions.

### Type Algebra

```
State = Working | Idle | Done | Stuck | Zombie

Polecat {
  Name: String,              -- Identifier
  Rig: String,               -- Parent rig
  State: State,              -- Lifecycle state
  ClonePath: String,         -- Worktree directory path
  Branch: String,            -- Current git branch
  Issue: Option String,      -- Assigned issue ID
  CreatedAt UpdatedAt: Time
}

CleanupStatus = Clean | Uncommitted | Stash | Unpushed | Unknown
```

### State Machine

```
                ┌──────────────┐
                │  Not Exists  │
                └──────┬───────┘
                       │ Add(name)
                       ▼
  ┌─────────────────────────────────────────┐
  │               Idle                      │
  │  (sandbox exists, no session, reusable) │
  └─────────┬──────────────────┬────────────┘
            │ Sling(issue)     ↑ cleanup/reuse
            ▼                  │
  ┌─────────────────────────────┐
  │            Working          │
  │  (tmux session active)      │
  └──────┬──────────┬───────────┘
         │          │
  gt done│          │ session dies
         ▼          ▼
  ┌───────────┐  ┌──────────┐
  │   Done    │  │ Stalled  │ (detected by Witness)
  └─────┬─────┘  └──────────┘
        │ cleanup succeeds
        ▼
  ┌───────────┐
  │   Idle    │ (recycled, ready for new assignment)
  └───────────┘
```

### Stalled vs Zombie (Detected States)

```
Stalled(p) ≡ p.State = Working ∧ ¬TmuxSessionExists(p.SessionName)
Zombie(p)  ≡ TmuxSessionExists(p.SessionName) ∧ ¬WorktreeExists(p.ClonePath)
```

These are not stored states but detected conditions — the Witness infers them
from the divergence between expected and actual system state.

### CleanupStatus Algebra

```
IsSafe : CleanupStatus → Bool
  IsSafe(s) ≡ s = Clean

RequiresRecovery : CleanupStatus → Bool
  RequiresRecovery(s) ≡ s ∈ {Uncommitted, Stash, Unpushed}

CanForceRemove : CleanupStatus → Bool
  CanForceRemove(s) ≡ s ∈ {Clean, Uncommitted}
```

The lattice of cleanup safety:
```
Clean < Uncommitted < {Stash, Unpushed} < Unknown
(safer)                                   (less safe)
```

### Worktree Path Convention

```
PolecatPath(rig, name) = rig.Path + "/polecats/" + name + "/" + repoName
  where repoName = basename(rig.GitURL, ".git")
```

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| Add | Manager × String × Opts → Error | Creates worktree + agent bead |
| Remove | Manager × String × Bool → Error | Removes worktree (force flag) |
| Sling | Manager × String × String → Error | Assigns issue, starts session |
| List | Manager → List Polecat | Snapshot of all polecats |
| Get | Manager × String → Polecat × Error | Lookup by name |
| StartSession | Manager × String × Opts → Error | Starts tmux session |
| StopSession | Manager × String → Error | Kills tmux session |
| RepairWorktree | Manager × String → Error | Fixes stale worktree state |

### Lean 4 Formalization Candidates

1. **State machine validity**: Only valid state transitions are permitted
2. **Cleanup safety lattice**: IsSafe ⇒ CanForceRemove ⇒ ¬RequiresRecovery⁻¹
3. **Idle reuse**: After successful cleanup, polecat returns to Idle (not destroyed)
4. **Worktree-session correlation**: Working ⇒ TmuxSessionExists ∧ WorktreeExists
5. **Name uniqueness**: ∀ p₁ p₂ ∈ Rig.Polecats. p₁.Name = p₂.Name ⇒ p₁ = p₂

---

## 5. Package: beads (routing & redirect)

**Files**: routes.go (302 lines), beads_redirect.go (346 lines)

### Core Abstraction

The beads routing system provides **location transparency** for issue databases.
Agents can create, read, and update beads without knowing where the database
physically lives — routing resolves bead ID prefixes to filesystem paths, and
redirect files chain worktrees to shared databases.

### Type Algebra

```
Route { Prefix: String, Path: String }
  -- Prefix includes trailing hyphen: "gt-", "bd-", "hq-"
  -- Path is relative to town root: "gastown/mayor/rig", "."

Routes = List Route
  -- Stored in .beads/routes.jsonl (JSONL format, one route per line)
  -- Invariant: at most one route per prefix (enforced by AppendRoute)
```

### Prefix Extraction

```
ExtractPrefix : String → String
  ExtractPrefix("") = ""
  ExtractPrefix(s) = if idx = firstIndexOf(s, '-'), idx > 0
                     then s[0..idx+1]
                     else ""

-- Examples:
ExtractPrefix("gt-df2")     = "gt-"
ExtractPrefix("hq-cv-abc")  = "hq-"
ExtractPrefix("no-hyphen")  = "no-"
ExtractPrefix("-leading")   = ""     -- idx = 0, invalid
ExtractPrefix("")           = ""
```

### Routing Resolution

```
GetRigPathForPrefix : TownRoot × Prefix → Option Path
  GetRigPathForPrefix(t, p) =
    let routes = LoadRoutes(t/.beads)
    match find(r. r.Prefix = p, routes) with
    | Some r → if r.Path = "." then t else t/r.Path
    | None → None

GetRigNameForPrefix : TownRoot × Prefix → Option String
  GetRigNameForPrefix(t, p) =
    let r = lookup(p, routes)
    if r.Path = "." then None    -- Town-level, no rig
    else firstComponent(r.Path)  -- "gastown/mayor/rig" → "gastown"

GetPrefixForRig : TownRoot × RigName → String
  -- Reverse lookup: searches routes where path starts with rigName
  -- Fallback: config.GetRigPrefix(townRoot, rigName)
```

### Route CRUD

```
LoadRoutes : BeadsDir → List Route × Error
  -- JSONL parsing: skip empty lines, comments (#), malformed entries
  -- Missing file → empty list (not an error)

AppendRoute : TownRoot × Route → Error
  -- Upsert: if prefix exists, update path; else append
  -- Write via atomic temp file + rename

RemoveRoute : TownRoot × Prefix → Error
  -- Filter + rewrite

WriteRoutes : BeadsDir × List Route → Error
  -- Atomic write: temp file → sync → rename
```

### Redirect Resolution

```
ResolveBeadsDir : WorkDir → BeadsDir
  ResolveBeadsDir(w) =
    let beadsDir = w/.beads
    let target = readFile(beadsDir/redirect)
    if target = None then beadsDir                       -- No redirect
    else let resolved = resolve(target, w)               -- Relative to workDir
         if resolved = beadsDir then beadsDir             -- Circular → ignore
         else resolveBeadsDirWithDepth(resolved, 3)       -- Follow chain

resolveBeadsDirWithDepth : BeadsDir × Nat → BeadsDir
  resolveBeadsDirWithDepth(d, 0) = d                      -- Depth limit
  resolveBeadsDirWithDepth(d, n) =
    let target = readFile(d/redirect)
    if target = None then d                               -- Terminal
    else let resolved = resolve(target, parent(d))
         if resolved = d then d                           -- Circular
         else resolveBeadsDirWithDepth(resolved, n-1)     -- Recurse
```

### Redirect Target Computation

```
ComputeRedirectTarget : TownRoot × WorktreePath → String × Error
  -- Input validation:
  --   worktree must be ≥ 2 levels deep from town root
  --   worktree must NOT be in mayor/ (prevents circular redirect)

  let parts = split(relPath(townRoot, worktreePath), "/")
  let rigRoot = townRoot/parts[0]
  let depth = len(parts) - 1
  let upPath = "../" × depth

  -- Database location resolution:
  if rigRoot/.beads/dolt/ exists:
    redirectPath = upPath + ".beads"                      -- Direct to rig beads
  elif rigRoot/.beads/redirect exists:
    redirectPath = upPath + ".beads"                      -- Rig has redirect (tracked beads)
    -- Optimization: skip intermediate hop, point directly to final target
    -- Validate: target must stay within town root and exist on disk
  elif rigRoot/mayor/rig/.beads exists:
    redirectPath = upPath + "mayor/rig/.beads"            -- Mayor fallback
  else:
    Error("no beads found")
```

### Redirect Architecture Diagram

```
Town Root
├── .beads/                           ← Town beads (hq-* prefix, canonical)
│   └── routes.jsonl                  ← Prefix → path mapping
├── gastown/                          ← Rig
│   ├── .beads/
│   │   └── redirect → mayor/rig/.beads   ← Points to canonical
│   ├── mayor/rig/.beads/             ← Canonical beads database
│   │   ├── dolt/                     ← Dolt database engine
│   │   ├── formulas/                 ← Installed formulas
│   │   └── routes.jsonl
│   ├── polecats/toast/.beads/
│   │   └── redirect → ../../mayor/rig/.beads  ← Skips intermediate hop
│   ├── crew/max/.beads/
│   │   └── redirect → ../../mayor/rig/.beads
│   └── refinery/rig/.beads/
│       └── redirect → ../../mayor/rig/.beads
```

### Safety Properties

1. **Anti-circular**: `ComputeRedirectTarget` refuses to create redirects in `mayor/rig`
2. **Depth-bounded**: Redirect chains limited to depth 3
3. **Self-healing**: Circular redirects detected and removed at resolution time
4. **Town-bounded**: Redirect targets validated to stay within town root
5. **Existence-checked**: Redirect targets verified to exist on disk

### Lean 4 Formalization Candidates

1. **Routing is a partial function**: At most one path per prefix (enforced by AppendRoute upsert)
2. **Prefix extraction well-defined**: ExtractPrefix never panics, returns "" for invalid input
3. **Redirect termination**: resolveBeadsDirWithDepth always terminates (depth bound)
4. **Anti-circular safety**: ComputeRedirectTarget rejects mayor/ paths
5. **Redirect chain optimization**: Direct-to-final-target produces same resolved path as following chain
6. **Route round-trip**: GetPrefixForRig(t, GetRigNameForPrefix(t, p)) = p (on registered domain)

---

## 6. Package: formula (embedding & provisioning)

**Files**: embed.go (358 lines), types.go (6.2K), parser.go (11.1K)

### Core Abstraction

Formulas are embedded in the `gt` binary at compile time and provisioned to the
filesystem at install time. The embedding system provides version tracking via
SHA256 checksums to detect user modifications vs. upstream updates.

### Type Algebra

```
InstalledRecord { Formulas: Map String String }  -- filename → SHA256

FormulaStatus { Name Status EmbeddedHash InstalledHash CurrentHash: String }

HealthReport {
  Formulas: List FormulaStatus,
  OK Outdated Modified Missing New Untracked Error: Nat
}

Status = OK | Outdated | Modified | Missing | New | Untracked | Error
```

### Formula Locations

```
Source of truth:  internal/formula/formulas/*.formula.toml  (compiled into binary)
Installation:    <rig>/.beads/formulas/*.formula.toml       (filesystem)
Install record:  <rig>/.beads/formulas/.installed.json      (checksums)
```

### Status Classification (Three-Hash Comparison)

```
classify(embeddedHash, installedHash, currentHash, fileExists, wasInstalled) =
  if ¬fileExists ∧ wasInstalled  then Missing     -- Installed, user deleted
  if ¬fileExists ∧ ¬wasInstalled then New          -- Never installed
  if fileError                   then Error        -- Can't read
  if currentHash = embeddedHash  then OK           -- Matches source
  if wasInstalled ∧ currentHash = installedHash
                                 then Outdated     -- Embedded changed, user didn't touch
  if wasInstalled                then Modified     -- User changed it
  else                                Untracked    -- Exists but not tracked
```

### Update Safety

```
shouldUpdate(status) = status ∈ {Outdated, Missing, New, Untracked}
shouldSkip(status)   = status ∈ {Modified}
shouldIgnore(status) = status ∈ {OK, Error}
```

**Key invariant**: User modifications are never overwritten. The three-hash system
distinguishes "embedded changed" from "user changed" from "both changed".

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| GetEmbeddedFormulaContent | String → Bytes × Error | Pure (compiled-in data) |
| ProvisionFormulas | BeadsPath → Nat × Error | Idempotent (skip existing) |
| CheckFormulaHealth | BeadsPath → HealthReport × Error | Pure observation |
| UpdateFormulas | BeadsPath → (Nat × Nat × Nat) × Error | Safe update (respects modifications) |

### Conservation Law

```
∀ report: HealthReport.
  report.OK + report.Outdated + report.Modified + report.Missing
  + report.New + report.Untracked + report.Error = |embeddedFormulas|
```

### Lean 4 Formalization Candidates

1. **Status classification is total**: Every (embedded, installed, current) triple maps to exactly one status
2. **Update safety**: Modified formulas are never overwritten
3. **Provisioning idempotency**: ProvisionFormulas called twice installs each formula at most once
4. **Conservation**: Sum of all status counts = total embedded formulas
5. **Hash determinism**: Same content always produces same SHA256

---

## 7. Package: session (identity & registry)

**Files**: identity.go, registry.go, names.go, lifecycle.go, pidtrack.go,
startup.go, town.go, stale.go

### Core Abstraction

The session package provides the **naming algebra** that maps between four equivalent
representations of an agent identity. The PrefixRegistry provides the bidirectional
map between rig names and beads prefixes.

### Type Algebra

```
Role = Mayor | Deacon | Overseer | Witness | Refinery | Crew | Polecat

AgentIdentity {
  Role: Role,
  Rig: String,      -- Empty for town-level roles (Mayor, Deacon)
  Name: String,     -- Empty for singleton roles (Mayor, Deacon, Witness, Refinery)
  Prefix: String    -- Beads prefix for rig-level agents
}
```

### Identity Isomorphisms

An agent identity has four equivalent representations connected by invertible maps:

```
                    Address
                   ╱        ╲
          ParseAddress   format
                 ╱            ╲
AgentIdentity ←────────────────→ "gastown/polecats/Toast"
                 ╲            ╱
        ParseSessionName  format
                   ╲        ╱
                  SessionName

                "gt-Toast"
```

| Representation | Example (Polecat Toast in gastown) |
|---------------|-----------------------------------|
| Address | `gastown/polecats/Toast` |
| SessionName | `gt-Toast` |
| GTRole | `gastown/polecats/Toast` |
| BeaconAddress | `polecat Toast (rig: gastown)` |

### Address Parsing Grammar

```
Address = RigName "/" RoleName
        | RigName "/" RoleName "/" AgentName
        | "mayor" | "mayor/"
        | "deacon" | "deacon/"

-- Where:
RoleName = "witness" | "refinery" | "crew" | "polecats" | <polecat-name>
AgentName = <string>

-- Shorthand: "gastown/Toast" → Polecat Toast in gastown
-- Full form: "gastown/polecats/Toast" → Same
```

### Session Name Parsing

```
SessionName = "hq-mayor"
            | "hq-deacon"
            | "hq-boot"
            | Prefix "-witness"
            | Prefix "-refinery"
            | Prefix "-crew-" Name
            | Prefix "-" Name         -- Polecat (default)

-- Prefix resolved via PrefixRegistry: "gt" → "gastown"
```

### PrefixRegistry (Bidirectional Map)

```
PrefixRegistry {
  prefixToRig: Map String String,    -- "gt" → "gastown"
  rigToPrefix: Map String String     -- "gastown" → "gt"
}

Register : PrefixRegistry × String × String → ()
  post: RigForPrefix(p) = r ∧ PrefixForRig(r) = p

RigForPrefix : String → Option String
PrefixForRig : String → Option String
```

**Invariant**: `prefixToRig` and `rigToPrefix` are inverses on the registered domain:
```
∀ (p, r) ∈ registered.
  prefixToRig[p] = r ⟺ rigToPrefix[r] = p
```

### Stale Detection

```
StaleReasonForTimes(messageTime, sessionCreated) =
  if messageTime < sessionCreated then (true, "message predates session")
  else (false, "")
```

This prevents agents from processing messages from previous sessions.

### Lean 4 Formalization Candidates

1. **Identity round-trip**: ParseAddress(id.Address()) = id
2. **Session name round-trip**: ParseSessionName(id.SessionName()) = id
3. **Registry bijectivity**: Register(p, r) ⇒ RigForPrefix(p) = r ∧ PrefixForRig(r) = p
4. **Address parsing totality**: Every valid address string has exactly one parse
5. **Stale detection monotonicity**: message before session ⇒ always stale

---

## 8. Package: git (worktree operations)

**Files**: git.go (main logic), copy_unix.go, copy_windows.go

### Core Abstraction

The git package wraps git subprocess operations with particular focus on **worktree
management** — the mechanism by which multiple agent sandboxes share a single repository.

### Type Algebra

```
Git { workDir: String, gitDir: Option String }

Worktree { Path Branch Commit: String }

GitError { Command: String, Args: List String, Stdout Stderr: String, Err: Error }
```

### Worktree Operations

| Operation | Signature | Git Command |
|-----------|-----------|-------------|
| WorktreeAdd | Path × Branch → Error | `git worktree add -b <branch> <path>` |
| WorktreeAddFromRef | Path × Branch × Ref → Error | `git worktree add -b <branch> <path> <ref>` |
| WorktreeAddDetached | Path × Ref → Error | `git worktree add --detach <path> <ref>` |
| WorktreeAddExisting | Path × Branch → Error | `git worktree add <path> <branch>` |
| WorktreeAddExistingForce | Path × Branch → Error | `git worktree add --force <path> <branch>` |
| WorktreeRemove | Path × Bool → Error | `git worktree remove [--force] <path>` |
| WorktreeMove | OldPath × NewPath → Error | `git worktree move <old> <new>` |
| WorktreePrune | () → Error | `git worktree prune` |
| WorktreeList | () → List Worktree × Error | `git worktree list --porcelain` |

### Worktree Algebra

Git worktrees form a **tree of shared references**:

```
          Bare Repo (mayor/rig/.git)
         ╱    │    │    ╲
        ╱     │    │     ╲
Polecat₁  Polecat₂  Crew₁  Refinery
  (branch₁) (branch₂) (main)  (main)
```

**Key properties**:
1. All worktrees share the same `.git` objects (space-efficient)
2. Each worktree has its own branch (no branch collision, except with `--force`)
3. LFS files are pointer-only at checkout time (`GIT_LFS_SKIP_SMUDGE=1`)
4. Submodules are initialized per-worktree (`InitSubmodules`)
5. Cross-filesystem moves use copy+delete (handles EXDEV errors)

### Branch Naming Convention

```
PolecatBranch(rig, name, issue) =
  "polecat/" ++ name ++ "/" ++ rig.Prefix ++ "-" ++ issueID ++ "@" ++ shortHash
  -- Example: "polecat/rictus/gt-df2@mmhfhadc"
```

### Lean 4 Formalization Candidates

1. **Worktree list completeness**: WorktreeList returns all paths registered by WorktreeAdd
2. **Branch uniqueness**: Without `--force`, no two worktrees share a branch
3. **Remove inverse**: WorktreeRemove(WorktreeAdd(p, b)) restores prior state
4. **Move preservation**: WorktreeMove preserves all git references
5. **Prune safety**: WorktreePrune only removes entries for non-existent paths

---

## 9. Package: config (path resolution)

**Files**: types.go (56K), loader.go (77K), env.go (16K), roles.go (8.7K),
agents.go (25.8K), operational.go (21.8K), cost_tier.go (7.4K)

### Core Abstraction

The config package defines the **configuration topology** of a Gas Town workspace —
which files live where, how they're loaded, and how they cascade.

### Type Algebra

```
TownConfig {
  Type: "town", Version: Nat,
  Name: String,                   -- Internal identifier
  Owner: Option String,           -- Entity identity (email)
  PublicName: Option String,      -- Display name
  CreatedAt: Time
}

MayorConfig {
  Type: "mayor-config", Version: Nat,
  Theme: Option TownThemeConfig,
  Daemon: Option DaemonConfig,
  Deacon: Option DeaconConfig,
  DefaultCrewName: Option String
}

TownSettings {
  Type: "town-settings", Version: Nat,
  CLITheme: String,               -- "dark" | "light" | "auto"
  DefaultAgent: String,           -- Agent preset name
  Agents: Map String RuntimeConfig,
  RoleAgents: Map String String,  -- Role → agent mapping
  AgentEmailDomain: String,       -- Default: "gastown.local"
  ...
}

RigsConfig { Version: Nat, Rigs: Map String RigEntry }

RigEntry {
  GitURL: String, PushURL: Option String,
  UpstreamURL: Option String, LocalRepo: Option String,
  AddedAt: Time, BeadsConfig: Option BeadsConfig
}

BeadsConfig { Repo: String, Prefix: String }
```

### Configuration File Topology

```
TownRoot/
├── mayor/
│   ├── town.json        ← TownConfig (identity, immutable after creation)
│   ├── config.json      ← MayorConfig (behavioral settings)
│   └── rigs.json        ← RigsConfig (rig registry)
├── settings/
│   └── config.json      ← TownSettings (agent config, cost tiers)
└── <rig>/
    └── config.json      ← RigConfig (rig-specific settings)
```

### Role → Agent Resolution Chain

```
ResolveRoleAgent(townRoot, role) =
  1. TownSettings.RoleAgents[role]        -- Per-role mapping
  2. TownSettings.DefaultAgent             -- Town-wide default
  3. "claude"                              -- System default

ResolveAgentConfig(townRoot, agentName) =
  1. TownSettings.Agents[agentName]       -- Custom config
  2. BuiltInPresets[agentName]             -- Built-in preset
  3. Error
```

### Lean 4 Formalization Candidates

1. **Config loading totality**: LoadTownConfig either succeeds with valid config or returns error
2. **Version compatibility**: Config loaders handle all versions ≤ CurrentVersion
3. **Role resolution completeness**: Every role resolves to exactly one agent config
4. **Rig registry uniqueness**: RigsConfig.Rigs is keyed by name (at most one entry per rig)

---

## 10. Cross-Package Algebraic Structure

### The Spatial Hierarchy (Containment Algebra)

The system defines a strict containment hierarchy with five levels:

```
Level 0: Wasteland (federation)       -- Many towns
Level 1: Town (workspace root)        -- One per machine (typically)
Level 2: Rig (managed repository)     -- Many per town
Level 3: Role (agent directory)       -- Fixed set per rig
Level 4: Agent (worktree instance)    -- Many per role (polecats, crew)
```

Each level introduces its own marker and configuration:

| Level | Marker | Config |
|-------|--------|--------|
| Town | `mayor/town.json` | `mayor/config.json`, `settings/config.json` |
| Rig | `config.json` (in rig root) | Rig config.json |
| Role | Directory existence (`polecats/`, `witness/`) | (None — roles are structural) |
| Agent | Worktree + agent bead | `.beads/redirect` |

### The Three Topologies

The system maintains three parallel topologies that must stay synchronized:

**1. Filesystem Topology** (physical):
```
town/rig/polecats/name/repo/  -- Path on disk
```

**2. Git Topology** (version control):
```
bare-repo → worktree₁, worktree₂, ..., worktreeₙ
  (mayor/rig)  (polecats/*)    (crew/*)     (refinery/rig)
```

**3. Agent Topology** (logical identity):
```
Role × Rig × Name → AgentIdentity
  → Address ("gastown/polecats/Toast")
  → SessionName ("gt-Toast")
  → BeadID ("gt-gastown-polecat-Toast")
```

The **PrefixRegistry** is the bridge between git topology (rig names) and agent
topology (beads prefixes). The **redirect system** is the bridge between filesystem
topology (worktree paths) and beads topology (database locations).

### Shared Algebraic Patterns

**1. Path Resolution Chains** (workspace, beads redirect, formula, config):
```
workspace:  Walk up ancestors until marker found
beads:      Follow redirect chain until terminal (depth ≤ 3)
formula:    Embedded → .beads/formulas/ (two-level)
config:     Wisp → Bead → Town → System (four-level cascade)
```

All four implement the same abstract pattern: a search through ordered locations,
returning the first match.

**2. Idempotent Provisioning** (formula, overlay, gitignore):
```
ProvisionFormulas:        Skip existing files
CopyOverlay:              Overwrite (idempotent for same content)
EnsureGitignorePatterns:  Check before appending
```

**3. Fail-Soft Semantics** (overlay, workspace, routing):
```
CopyOverlay:      Individual file failures are warnings, not errors
Find:             Secondary marker is fallback for missing primary
LoadRoutes:       Missing file → empty list (not error)
GetPrefixForRig:  Fallback to config.GetRigPrefix
```

**4. Atomic Writes** (routes, formulas):
```
WriteRoutes:           temp file → sync → rename
saveInstalledRecord:   direct write (acceptable: single consumer)
```

**5. Anti-Escape Validation** (beads redirect):
```
ComputeRedirectTarget validates:
  - Target stays within town root (relToTown doesn't start with "..")
  - Target exists on disk
  - Target is not in mayor/ (anti-circular)
```

### The Beads Location Problem

The most complex spatial problem in the system: given a worktree at any depth in
the hierarchy, find the correct beads database. The solution involves three mechanisms:

```
1. Prefix → Path (routes.jsonl)
   "gt-df2" → prefix "gt-" → path "gastown/mayor/rig" → database

2. Worktree → Database (redirect files)
   polecats/toast/.beads/redirect → ../../mayor/rig/.beads

3. Rig → Database (rig-level redirect)
   gastown/.beads/redirect → mayor/rig/.beads
```

These three mechanisms compose:
```
ResolveBeadForAgent(workDir, beadID) =
  let prefix = ExtractPrefix(beadID)
  let rigPath = GetRigPathForPrefix(townRoot, prefix)    -- Mechanism 1
  let beadsDir = ResolveBeadsDir(rigPath)                 -- Mechanisms 2+3
  return beadsDir
```

### Conservation Laws

1. **Rig ↔ Prefix bijection**: Each rig has exactly one prefix, each prefix maps to exactly one rig
2. **Worktree ↔ Branch bijection**: Each worktree has exactly one branch (without --force)
3. **Agent ↔ Identity bijection**: Each running agent has exactly one AgentIdentity
4. **Redirect convergence**: All redirects within a rig converge to the same beads directory
5. **Formula conservation**: |embedded| = OK + Outdated + Modified + Missing + New + Untracked + Error

### Category-Theoretic View

The spatial subsystem can be viewed as a category where:
- **Objects**: Filesystem locations (paths)
- **Morphisms**: Resolution operations (Find, Resolve, Lookup)

Key functors:
```
WorkDir → TownRoot     (workspace.Find — upward traversal)
TownRoot → RigPath     (routing — prefix resolution)
RigPath → BeadsDir     (redirect — chain following)
BeadsDir → FormulaDir  (formula — provisioned subdirectory)
AgentName → SessionName (identity — prefix registry)
```

The composition `WorkDir → ... → BeadsDir` is the full spatial resolution pipeline.

---

## 11. Summary: Formalization Priority

### High Priority (complex resolution logic, safety-critical invariants)

| Package | Target | Why |
|---------|--------|-----|
| beads (redirect) | Redirect chain resolution, anti-circular safety | Termination proof, escape prevention |
| beads (routing) | Prefix → path bijection, route CRUD | Uniqueness invariant, atomic writes |
| workspace | Find algorithm with worktree override | Correctness for nested hierarchies |

### Medium Priority (clear structure, important but simpler)

| Package | Target | Why |
|---------|--------|-----|
| session (identity) | Four-way identity isomorphism | Round-trip proofs |
| formula (embed) | Three-hash status classification | Totality, conservation law |
| rig (config) | Layer cascade with blocking and stacking | Override/stacking semantics |

### Lower Priority (operational, fewer invariants)

| Package | Target | Why |
|---------|--------|-----|
| polecat | State machine, cleanup safety lattice | Lifecycle correctness |
| git (worktree) | Worktree ↔ branch bijection | Git's own invariants, mostly verified upstream |
| config | File topology, loading/saving | Serialization correctness |

### Cross-Cutting Theorems

1. **Spatial resolution is confluent**: All paths to finding a beads database yield the same result
2. **Redirect chains terminate**: Bounded depth + anti-circular checks guarantee termination
3. **Identity is a bijection**: ParseAddress and ParseSessionName are mutual inverses (on valid domain)
4. **Configuration cascade is deterministic**: Same (key, rig) always produces same (value, source)
5. **Provisioning is monotone**: Formulas accumulate on disk, never removed by provisioning
6. **Worktree paths are injective**: No two agents share the same filesystem path
