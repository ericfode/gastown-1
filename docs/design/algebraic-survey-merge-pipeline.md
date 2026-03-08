# Algebraic Survey: Merge Pipeline (Refinery)

**Subsystem**: Merge Pipeline
**Packages**: `refinery`, `mq`, `protocol`, `beads` (MR/merge-slot), `cmd/mq_*`
**Purpose**: Takes completed polecat work branches, validates them via quality gates,
and merges them into target branches using a batch-then-bisect algorithm with
serialized push access via merge slots.

## 1. Subsystem Overview

The Merge Pipeline answers: given a set of completed work branches (merge requests)
and a target branch, how are they validated, ordered, merged, and cleaned up?

```
gt done (polecat) → MR bead created → Refinery polls → Engineer processes
                                                            ↓
                              Score → Batch → Stack → Gate → Push → PostMerge
```

### Package Dependency Graph (within subsystem)

```
mq          ←── crypto/rand, crypto/sha256 (leaf package, ID generation only)
refinery    ←── beads, crew, events, git, mail, rig (core processing)
protocol    ←── mail (inter-agent message types)
cmd/mq_*    ←── beads, config, git, refinery, rig, workspace (CLI orchestration)
```

**Key finding**: The `mq` package is a pure leaf (ID generation). The `refinery`
package contains ALL merge logic (Engineer, Manager, scoring, batching). The
`protocol` package defines message types but contains no processing logic.
The `cmd/mq_*` files provide CLI entry points that wire everything together.

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| refinery.Manager | cmd/refinery, witness (health checks) |
| refinery.Engineer | Refinery agent (via gt prime → patrol cycle) |
| protocol | witness, refinery (inter-agent messages) |
| mq | cmd/done (MR ID generation at submit time) |
| cmd/mq_* | CLI users, polecats (gt done → gt mq submit) |

---

## 2. Package: refinery/types

**Files**: types.go

### Type Algebra

The MR lifecycle is modeled as two parallel state machines:

**MRStatus (beads-compatible, 3 states):**
```
MRStatus = MROpen | MRInProgress | MRClosed
```

**MRPhase (extended, 8 states):**
```
MRPhase = MRPhaseReady | MRPhaseClaimed | MRPhasePreparing | MRPhasePrepared
        | MRPhaseMerging | MRPhaseMerged | MRPhaseRejected | MRPhaseFailed
```

### MR Status State Machine

```
                ┌──────────┐
                │   Open   │
                └──┬───┬───┘
     Claim (valid) │   │ Manual reject
                   ▼   ▼
         ┌─────────────────┐     ┌──────────┐
         │  In Progress    │────→│  Closed  │
         └────────┬────────┘     └──────────┘
                  │ Failure                ↑
                  │ (reassign)             │
                  ▼                        │
         ┌──────────┐    Manual reject     │
         │   Open   │─────────────────────→┘
         └──────────┘
```

Closed is a **terminal absorbing state**: once entered, no transitions out are possible.

### MR Phase State Machine (Extended)

```
ValidPhaseTransitions:
  Ready     → {Claimed}
  Claimed   → {Preparing, Ready}
  Preparing → {Prepared, Failed}
  Prepared  → {Merging, Rejected, Ready}
  Merging   → {Merged, Failed}
  Failed    → {Ready}
  Merged    → ∅  (terminal)
  Rejected  → ∅  (terminal)
```

**Key algebraic properties:**
1. **Two terminal states**: Merged and Rejected are absorbing (no outgoing transitions)
2. **Recovery path**: Failed → Ready allows retry
3. **Demotion**: Claimed → Ready and Prepared → Ready allow releasing back to queue
4. **Self-transition**: `from = to` is always valid (no-op)
5. **Deterministic validation**: `ValidatePhaseTransition(from, to)` is a total function

### Close Reasons (Tagged Union)

```
CloseReason = CloseReasonMerged | CloseReasonRejected
            | CloseReasonConflict | CloseReasonSuperseded
```

### Failure Types (Sum Type with Routing)

```
FailureType = FailureNone | FailureConflict | FailureTestsFail | FailureBuildFail
            | FailureFlakyTest | FailurePushFail | FailureFetch | FailureCheckout
```

Each failure type maps to a **label** and a **routing decision**:

| FailureType | Label | AssignToWorker |
|-------------|-------|----------------|
| Conflict | needs-rebase | true |
| TestsFail | needs-fix | true |
| BuildFail | needs-fix | true |
| FlakyTest | needs-fix | true |
| PushFail | needs-retry | false |
| Fetch | (none) | false |
| Checkout | (none) | false |

**Routing partition**: Failures partition into {worker-fixable, infrastructure} where
`ShouldAssignToWorker()` is `true` iff the failure requires human/agent intervention
on the source branch.

### Core Types

```
MergeRequest {
  ID Branch Worker IssueID SwarmID TargetBranch: String,
  CreatedAt: Time, Status: MRStatus, CloseReason: CloseReason, Error: String
}

MRInfo {
  ID Branch Target SourceIssue Worker Rig Title AgentBead: String,
  Priority RetryCount: Int, ConvoyID: String, ConvoyCreatedAt: Option Time,
  CreatedAt UpdatedAt: Time, BlockedBy Assignee: String,
  PreVerified: Bool, PreVerifiedAt: Time, PreVerifiedBase: String,
  BranchExistsLocal BranchExistsRemote: Bool
}

QueueItem { Position: Int, MR: Ptr MergeRequest, Age: String }

MRAnomaly { ID Branch Type Assignee Detail: String, Age: Duration }
```

### Key Operations on MergeRequest

| Operation | Signature | Algebraic Property |
|-----------|-----------|-------------------|
| SetStatus | MRStatus → Error | Validated transition (total on valid, error on invalid) |
| Close | CloseReason → Error | Idempotent guard: closed → closed = error |
| Reopen | () → Error | Partial: only from InProgress, clears CloseReason |
| Claim | () → Error | Partial: only from Open |
| ValidateTransition | MRStatus × MRStatus → Error | Reflexive (same-state = nil), absorbing at Closed |
| ValidatePhaseTransition | MRPhase × MRPhase → Error | Reflexive, two terminal states |

### Lean 4 Formalization Candidates

1. **State machine well-formedness**: All reachable states have defined transitions or are terminal
2. **Absorbing state**: From Closed, no valid transition exists (∀ s ≠ Closed. ValidateTransition(Closed, s) = error)
3. **Reflexivity**: ValidateTransition(s, s) = nil for all s
4. **Phase reachability**: Every non-terminal phase can reach Merged via some path
5. **Failure routing partition**: {worker-fixable} ∩ {infrastructure} = ∅

---

## 3. Package: refinery/score

**Files**: score.go

### Core Abstraction

The scoring function maps MR metadata to a single real number for priority ordering.
Higher scores mean higher priority (process first).

### Type Algebra

```
ScoreConfig {
  BaseScore ConvoyAgeWeight PriorityWeight RetryPenalty MRAgeWeight MaxRetryPenalty: Float64
}

ScoreInput {
  Priority RetryCount: Int,
  MRCreatedAt: Time, ConvoyCreatedAt: Option Time, Now: Time
}
```

### Scoring Formula

```
ScoreMR(input, config) =
    config.BaseScore
  + config.ConvoyAgeWeight * max(0, hours(now - convoyCreatedAt))
  + config.PriorityWeight * clamp(4 - priority, 0, 4)
  - min(config.RetryPenalty * retryCount, config.MaxRetryPenalty)
  + config.MRAgeWeight * max(0, hours(now - mrCreatedAt))
```

With defaults:
```
DefaultScoreConfig = { BaseScore=1000, ConvoyAgeWeight=10, PriorityWeight=100,
                       RetryPenalty=50, MRAgeWeight=1, MaxRetryPenalty=300 }
```

### Algebraic Properties

1. **Monotonicity in time**: Score increases with age (convoy and MR age terms are non-negative)
2. **Anti-monotonicity in retries**: Score decreases with retry count (bounded by MaxRetryPenalty)
3. **Priority ordering preserved**: P0 gets +400, P4 gets +0 — priority dominates age for fresh MRs
4. **Bounded penalty**: retryPenalty is capped at MaxRetryPenalty (300), preventing permanent deprioritization
5. **Starvation prevention**: ConvoyAgeWeight (10 pts/hour = 240 pts/day) ensures old convoys eventually
   surpass fresher, higher-priority work
6. **Priority clamping**: Invalid priorities (< 0 or > 4) are clamped to P4 (lowest), never crash

### Ordering

```
compareScoredIssues(a, b) =
  if a.score ≠ b.score then a.score > b.score    -- Higher score wins
  else a.issue.ID < b.issue.ID                    -- Lexicographic tiebreak
```

**Determinism**: Given identical inputs, ordering is fully deterministic (score + ID).

### Lean 4 Formalization Candidates

1. **Score monotonicity in time**: ∀ t₁ < t₂. ScoreMR(input, t₁) ≤ ScoreMR(input, t₂)
2. **Priority dominance**: For fresh MRs (age ≈ 0), P0 always scores higher than P4
3. **Retry bound**: retryPenalty ≤ MaxRetryPenalty (never exceeds cap)
4. **Starvation-freedom**: ∀ priority difference, ∃ convoy age that overcomes it
5. **Total ordering**: compareScoredIssues is a total order (reflexive, antisymmetric, transitive)

---

## 4. Package: refinery/batch

**Files**: batch.go

### Core Abstraction

The batch-then-bisect algorithm processes multiple MRs simultaneously for throughput,
then uses binary search to isolate failures. This is the key algorithmic innovation
of the merge pipeline.

### Type Algebra

```
BatchConfig {
  MaxBatchSize: Int, BatchWaitTime: Duration, RetryBatchOnFlaky: Bool
}

BatchResult {
  Merged Culprits Conflicts: List (Ptr MRInfo),
  MergeCommit: String, Error: Option Error
}
```

### The Batch-Then-Bisect Algorithm

```
ProcessBatch(batch, target, config) =
  -- Degenerate case
  if |batch| = 1 → processSingleMR(batch[0], target)

  -- Step 1: Build rebase stack
  (stacked, conflicts) ← BuildRebaseStack(batch, target)
  if |stacked| = 0 → return {Conflicts = conflicts}
  if |stacked| = 1 → verifyAndPush(stacked, target)

  -- Step 2: Run gates on stack tip
  gateResult ← runBatchGates()

  -- Step 3: Happy path — all green
  if gateResult.Success → fastForwardBatch(stacked, target)

  -- Step 4: Flaky test retry (optional)
  if config.RetryBatchOnFlaky:
    resetAndRebuildStack(stacked, target)
    retryResult ← runBatchGates()
    if retryResult.Success → fastForwardBatch(stacked, target)

  -- Step 5: Bisect to isolate culprit
  (good, culprits) ← bisectBatch(stacked, target)

  -- Step 6: Merge good MRs
  if |good| > 0:
    resetAndRebuildStack(good, target)
    verifyResult ← runBatchGates()
    if verifyResult.Success → fastForwardBatch(good, target)
    else → error("good subset also failed")

  return {Merged = good, Culprits = culprits, Conflicts = conflicts}
```

### Stack Construction

`BuildRebaseStack` creates a linear sequence of squash-merges:

```
target ← MR₁ ← MR₂ ← ... ← MRₙ
```

When an MR conflicts, it is **removed** and the stack is **rebuilt** from the base
SHA with the remaining MRs. This ensures the stack is always clean.

```
BuildRebaseStack(batch, target) =
  baseSHA ← git.Rev("HEAD")
  stacked, conflicts ← [], []
  for mr ∈ batch:
    if hasConflicts(mr, target):
      conflicts ← conflicts ++ [mr]
      git.ResetHard(baseSHA)
      for prev ∈ stacked: git.MergeSquash(prev)    -- Rebuild
    else:
      git.MergeSquash(mr)
      stacked ← stacked ++ [mr]
  return (stacked, conflicts)
```

### Bisection Algorithm

The bisection follows a modified binary search that handles **interaction effects**
(MR A works alone, MR B works alone, but A+B fails together):

```
bisectBatch(batch, target) =
  if |batch| ≤ 1 → return ([], batch)    -- Base case: single MR is culprit

  mid ← |batch| / 2
  left, right ← batch[:mid], batch[mid:]

  resetAndRebuildStack(left, target)
  leftResult ← runBatchGates()

  if leftResult.Success:
    -- Culprit is in right half
    (rightGood, rightCulprits) ← bisectRight(left, right, target)
    return (left ++ rightGood, rightCulprits)
  else:
    -- Culprit is in left half
    (leftGood, leftCulprits) ← bisectBatch(left, target)
    -- Test right half with leftGood context
    ...recurse for right with leftGood as known-good prefix
```

The `bisectRight` variant tests sub-batches of the right half in the **context**
of known-good MRs (cumulative merge), which correctly handles interaction effects.

### Batch Assembly

```
AssembleBatch(readyMRs, config) =
  batch ← []
  for mr ∈ readyMRs:
    if |batch| ≥ config.MaxBatchSize → break
    if mr.BlockedBy ≠ "" ∧ mr.BlockedBy ∉ batch.IDs → continue
    batch ← batch ++ [mr]
  return batch
```

**Key property**: MRs blocked by something not in the batch are excluded, but
MRs blocked by something *in* the batch are included (preserves dependency order).

### Key Invariants

1. **Conflict isolation**: Conflicting MRs are removed from the stack and tracked separately
2. **Stack rebuild consistency**: After removing a conflicting MR, the stack is rebuilt
   from the base SHA with all previously-stacked MRs
3. **Bisection soundness**: Every MR in `culprits` caused a gate failure; every MR
   in `good` passed gates in at least one valid combination
4. **Conservation**: `|merged| + |culprits| + |conflicts| ≤ |batch|`
5. **Batch size bound**: `|batch| ≤ min(MaxBatchSize, |readyMRs|)`
6. **Bisection complexity**: O(log N) gate runs in the worst case (binary search)
7. **Flaky test guard**: Optional full-batch retry before bisection avoids blaming
   an innocent MR for a flaky test

### Lean 4 Formalization Candidates

1. **Bisection termination**: bisectBatch terminates (strictly decreasing batch size)
2. **Conservation law**: merged ∪ culprits ∪ conflicts ⊆ batch (partition)
3. **Bisection correctness**: Every MR in culprits caused a gate failure
4. **Stack rebuild idempotency**: Rebuilding twice gives same result as rebuilding once
5. **Batch monotonicity**: Removing an MR from the batch can only improve (or maintain) gate results

---

## 5. Package: refinery/engineer

**Files**: engineer.go (1753 lines, largest file)

### Core Abstraction

The Engineer is the central processing unit of the merge pipeline. It polls for
ready MRs, claims them, runs quality gates, performs merges, and handles success/failure.

### Type Algebra

```
Engineer {
  rig: Ptr Rig, beads: Ptr Beads, git: Ptr Git,
  config: Ptr MergeQueueConfig, workDir: String,
  output: Writer, router: Ptr Router,
  mergeSlotEnsureExists: () → (String, Error),
  mergeSlotAcquire: (String, Bool) → (Ptr MergeSlotStatus, Error),
  mergeSlotRelease: (String) → Error,
  mergeSlotMaxRetries: Int, mergeSlotRetryBackoff: Duration
}

MergeQueueConfig {
  Enabled: Bool, OnConflict TestCommand: String,
  RunTests DeleteMergedBranches GatesParallel: Bool,
  RetryFlakyTests MaxConcurrent MaxRetryCount: Int,
  PollInterval StaleClaimTimeout: Duration,
  StaleClaimWarningAfter StaleClaimCriticalAfter: Duration,
  Gates: Map String (Ptr GateConfig),
  Batch: Option (Ptr BatchConfig)
}

GateConfig { Cmd: String, Timeout: Duration }
GateResult { Name: String, Success: Bool, Error: String, Elapsed: Duration }
ProcessResult { Success Conflict TestsFailed SlotTimeout: Bool, MergeCommit Error: String }
```

### The doMerge Algorithm (Single MR)

```
doMerge(branch, target, sourceIssue, skipGates) =
  -- Step 1: Verify source branch exists
  if ¬branchExists(branch) → fail(fetch)

  -- Step 2: Checkout target, pull latest
  git.Checkout(target); git.Pull("origin", target)

  -- Step 3: Check for conflicts
  conflicts ← git.CheckConflicts(branch, target)
  if |conflicts| > 0 → fail(conflict)

  -- Step 3.5: Push submodule commits (if branch changes submodule pointers)
  for sc ∈ submoduleChanges(target, branch):
    git.PushSubmoduleCommit(sc.Path, sc.NewSHA)

  -- Step 4: Run quality gates (unless pre-verified)
  if ¬skipGates:
    if |gates| > 0 → runGates()
    elif runTests ∧ testCommand ≠ "" → runTests()

  -- Step 5: Squash merge
  msg ← getBranchCommitMessage(branch) ?? fallbackMessage
  git.MergeSquash(branch, msg)

  -- Step 6: Get merge commit SHA
  mergeCommit ← git.Rev("HEAD")

  -- Step 7: Acquire merge slot (for default branch pushes only)
  if target = rig.DefaultBranch():
    holder ← acquireMainPushSlot()

  -- Step 8: Push to origin
  git.Push("origin", target)

  return ProcessResult{Success=true, MergeCommit=mergeCommit}
```

### Pre-Verification Fast-Path (Phase 3)

When a polecat runs full quality gates after rebasing onto the target branch,
it records `PreVerified=true`, `PreVerifiedBase=<target-HEAD-SHA>`.

```
FastPathCondition(mr) ≡ mr.PreVerified
                      ∧ mr.PreVerifiedBase ≠ ""
                      ∧ git.Rev("origin/" + mr.Target) = mr.PreVerifiedBase
```

When the fast-path fires, the Engineer **skips all gates**, reducing merge time
from O(gate-duration) to O(git-operations) (~5s).

**Staleness detection**: If the target moved since pre-verification (another MR
merged), the fast-path is invalidated and full gates run.

### Gate Execution

```
runGates(ctx) =
  names ← sort(keys(gates))    -- Deterministic ordering
  if config.GatesParallel:
    results ← parallelMap(names, λ n. runGate(ctx, n, gates[n]))
  else:
    results ← sequentialMap(names, λ n. runGate(ctx, n, gates[n]))
    -- Stop on first failure in sequential mode
  if any(¬r.Success for r ∈ results):
    return ProcessResult{TestsFailed=true, Error=join(failures)}
  return ProcessResult{Success=true}

runGate(ctx, name, gate) =
  gateCtx ← if gate.Timeout > 0 then withTimeout(ctx, gate.Timeout) else ctx
  run("sh", "-c", gate.Cmd)
  -- Timeout → "timed out after <duration>"
  -- Failure → include stderr (capped at 500 chars)
```

### Merge Slot Serialization

The merge slot is a **distributed mutex** implemented via beads that serializes
pushes to the default branch. This prevents race conditions when multiple
merge operations could push simultaneously.

```
MergeSlotStatus { ID: String, Available: Bool, Holder: String, Waiters: List String }

acquireMainPushSlot(ctx) =
  slotID ← mergeSlotEnsureExists()
  holder ← rigName + "/refinery/push/" + timestamp + "-" + seq
  selfConflictHolder ← rigName + "/refinery"

  for attempt ∈ [0..maxRetries]:
    if attempt > 0: sleep(backoff); backoff ← min(backoff * 2, 10s)
    status ← mergeSlotAcquire(holder, false)
    if status.Available ∨ status.Holder = holder → return holder
    if status.Holder = selfConflictHolder → return ""  -- Self-conflict bypass
  return error(errMergeSlotTimeout)
```

**Self-conflict bypass**: When the same refinery holds the slot for conflict
resolution and then needs to push, it detects its own holder string and
proceeds without re-acquisition (single-threaded guarantee).

### MR Lifecycle Management

**Claim/Release pattern:**
```
ListReadyMRs() = filter(openMRs, λ mr.
    mr.Status = "open"
  ∧ ¬hasOpenBlocker(mr)
  ∧ ¬hasLabel(mr, "gt:owned-direct")
  ∧ (mr.Assignee = "" ∨ isClaimStale(mr.UpdatedAt, StaleClaimTimeout)))

ClaimMR(mrID, workerID) = beads.Update(mrID, {Assignee: workerID})
ReleaseMR(mrID) = beads.Update(mrID, {Assignee: ""})
```

**Stale claim detection:**
```
isClaimStale(updatedAt, timeout) =
  if updatedAt = "" → false    -- No timestamp = valid claim
  t ← parse(updatedAt)
  return time.Since(t) ≥ timeout
```

Default stale claim timeout: 30 minutes. Conservative to avoid re-claiming MRs
with long-running test suites. Only one refinery runs per rig (enforced by
`ErrAlreadyRunning`), so concurrent re-claim races are impossible.

### Post-Merge Operations

After a successful merge, the Engineer performs a multi-step cleanup:

```
HandleMRInfoSuccess(mr, result) =
  1. Release merge slot (best-effort)
  2. Update MR bead with merge_commit SHA
  3. Close MR bead (reason: "merged")
  4. Close source issue (ForceClose to bypass molecule dependencies)
  5. Clear agent bead's active_mr reference
  6. Delete source branch (local + remote, if configured)
  7. Check and auto-close completed convoys
  8. Notify deacon of convoy-eligible merges
  9. Prune stale remote tracking refs
```

### Failure Handling

```
HandleMRInfoFailure(mr, result) =
  if result.SlotTimeout:
    -- Transient contention: MR stays in queue, no notification
    return

  -- Nudge polecat about failure (not mail — no permanent Dolt commits)
  nudge(polecat, "MERGE_FAILED: ...")

  if result.Conflict:
    -- Create conflict resolution task → dispatch to fresh polecat
    taskID ← createConflictResolutionTask(mr)
    -- Block MR on task via beads dependency
    beads.AddDependency(mr.ID, taskID)
    -- MR auto-unblocks when task closes
```

**Non-blocking delegation**: Conflict MRs don't stall the queue. A resolution
task is created and the MR is blocked on it. The queue continues to the next MR.

### Queue Anomaly Detection

```
ListQueueAnomalies(now) = detectQueueAnomalies(issues, now, warningAfter, branchExistsFn)

detectQueueAnomalies(issues, now, warningAfter, branchExistsFn) =
  for issue ∈ openMRs:
    -- Stale claim: assigned but not progressing
    if issue.Assignee ≠ "" ∧ age(issue.UpdatedAt) ≥ warningAfter:
      emit MRAnomaly{Type="stale-claim", Age=age}

    -- Orphaned branch: MR exists but branch is missing
    if ¬localExists ∧ ¬remoteTrackingExists:
      emit MRAnomaly{Type="orphaned-branch"}
```

### Key Invariants

1. **Single writer**: Only one refinery per rig (enforced by Manager.Start → ErrAlreadyRunning)
2. **Merge slot serialization**: Default branch pushes go through merge slot acquire/release
3. **Squash merge**: All merges are squash merges (preserving original commit message)
4. **Fail-open on missing data**: Store errors → skip MR, don't block queue
5. **Belt-and-suspenders**: `gt:owned-direct` labeled MRs are filtered out (shouldn't exist, but guarded)
6. **Post-merge consistency**: Source issue closed with "Merged in <MR-ID>" reference
7. **No permanent mail for routine signals**: Merge failures → nudge (ephemeral), not mail (Dolt commit)

---

## 6. Package: refinery/manager

**Files**: manager.go

### Core Abstraction

The Manager handles refinery **lifecycle** (start/stop/status) and provides
the queue view for CLI display. It is the external API surface, while the
Engineer handles internal processing.

### Type Algebra

```
Manager { rig: Ptr Rig, workDir: String, output: Writer }
```

### Session Lifecycle

```
Start(foreground, agentOverride) =
  if foreground → error("deprecated")
  if session exists:
    if agent alive → ErrAlreadyRunning
    else → kill zombie, recreate
  -- Build startup config (resolve working directory, settings, agent command)
  -- Create tmux session with Claude agent
  -- Configure: env vars, theme, dialog acceptance, liveness verification
  -- Start agent logging (optional)
  -- Record telemetry event

Stop() =
  if ¬session exists → ErrNotRunning
  tmux.KillSession(sessionID)
```

**ZFC compliance**: Session state is derived from tmux (not files). "Running"
means tmux session exists AND agent process is alive (not zombie).

### Queue Operations

```
Queue() : List QueueItem =
  issues ← beads.List({Label="gt:merge-request", Status="open"})
  scored ← map(issues, λ i. (i, calculateIssueScore(i)))
  sorted ← sortBy(compareScoredIssues, scored)
  return zipWithIndex(sorted)

FindMR(idOrBranch) =
  for item ∈ Queue():
    if matches(item.MR, idOrBranch) → return item.MR
  return ErrMRNotFound
```

**Multi-match strategy**: FindMR matches by exact ID, exact branch, branch with
polecat prefix, or ID prefix (partial match for convenience).

### Key Operations

| Operation | Signature | Property |
|-----------|-----------|----------|
| Start | Bool × String → Error | Idempotent zombie handling |
| Stop | () → Error | Idempotent (ErrNotRunning if already stopped) |
| Queue | () → List QueueItem × Error | Score-sorted, beads as source of truth |
| FindMR | String → Ptr MergeRequest × Error | Multi-strategy matching |
| RejectMR | String × String × Bool → Error | Closes bead, optionally nudges worker |
| PostMerge | String → PostMergeResult × Error | Closes MR + source issue, no git |
| IsRunning | () → Bool × Error | tmux + agent liveness check |
| IsHealthy | Duration → ZombieStatus | Detects hung sessions (alive but inactive) |

---

## 7. Package: mq

**Files**: id.go

### Core Abstraction

The `mq` package provides exactly one function: deterministic MR ID generation.

### ID Generation

```
GenerateMRID(prefix, branch) =
  randomBytes ← crypto/rand.Read(8)
  input ← branch + ":" + unixNano + ":" + hex(randomBytes)
  hash ← SHA256(input)[:10]    -- 10 hex chars = 40 bits
  return prefix + "-mr-" + hash
```

**Collision analysis**: 10 hex chars = ~1.1 trillion values per namespace.
Birthday paradox: ~1% collision probability at ~150K IDs.

### Lean 4 Formalization Candidates

1. **ID format**: Output matches regex `[a-z]+-mr-[0-9a-f]{10}`
2. **Uniqueness (probabilistic)**: With fresh random bytes, collision probability is negligible

---

## 8. Package: protocol

**Files**: types.go, refinery_handlers.go

### Core Abstraction

The protocol package defines **inter-agent message types** for the Witness ↔ Refinery
communication channel. Messages are sent via the mail system.

### Message Type Algebra

```
MessageType = TypeMergeReady | TypeMerged | TypeMergeFailed
            | TypeReworkRequest | TypeConvoyNeedsFeeding
```

### Message Flow

```
Witness → Refinery:
  MERGE_READY(polecat, branch, issue)
    -- Polecat work verified, ready for merge queue

Refinery → Witness:
  MERGED(polecat, branch, issue, targetBranch, mergeCommit)
    -- Merge succeeded

  MERGE_FAILED(polecat, branch, issue, targetBranch, failureType, error)
    -- Merge failed (tests, build, push)

  REWORK_REQUEST(polecat, branch, issue, targetBranch, conflictFiles)
    -- Rebase needed

Refinery → Deacon:
  CONVOY_NEEDS_FEEDING(convoyID, sourceIssue)
    -- Trigger immediate convoy feeding after merge
```

### Protocol Payloads

```
MergeReadyPayload { Branch Issue Polecat Rig Verified: String, Timestamp: Time }
MergedPayload { Branch Issue Polecat Rig MergeCommit TargetBranch: String, MergedAt: Time }
MergeFailedPayload { Branch Issue Polecat Rig FailureType Error TargetBranch: String, FailedAt: Time }
ReworkRequestPayload { Branch Issue Polecat Rig TargetBranch Instructions: String, RequestedAt: Time, ConflictFiles: List String }
PolecatDonePayload { Polecat ExitType Issue Branch MR ConvoyID MergeStrategy Errors: String, ConvoyOwned: Bool }
ConvoyNeedsFeedingPayload { ConvoyID SourceIssue Rig: String, MergedAt: Time }
```

### SkipMergeFlow Predicate

```
SkipMergeFlow(p: PolecatDonePayload) ≡ p.ConvoyOwned ∧ p.MergeStrategy = "direct"
```

Owned convoys with direct merge strategy bypass the Refinery entirely (polecats
push directly to the target branch).

### Belt-and-Suspenders in HandleMergeReady

```
HandleMergeReady(payload) =
  if payload.Verified = "owned+direct: skip merge":
    -- This shouldn't happen (gt done skips MR creation), but guard anyway
    return nil
  -- MR bead already exists (created by gt done) — Refinery queries beads directly
  return nil
```

---

## 9. MR Submission Pipeline (cmd/mq_submit)

### Submission Flow

```
runMqSubmit() =
  1. Detect rig from working directory
  2. Get current branch (or --branch override)
  3. Parse branch name → (issue, worker)
  4. Determine target branch:
     --epic flag → integration branch
     auto-detect → parent epic's integration branch
     default → rig's default branch (main)
  5. Inherit priority from source issue (or --priority override)
  6. Check idempotency: FindMRForBranch(branch)
     if exists → skip creation (idempotent)
  7. Create MR bead (ephemeral, label="gt:merge-request")
  8. Nudge refinery
  9. Auto-cleanup: send shutdown request to witness (unless --no-cleanup)
```

### Branch Name Parsing

```
parseBranchName(branch) =
  if hasPrefix("polecat/"):
    parts ← split("/", 3)
    if |parts| = 3 → (worker=parts[1], issue=parts[2] without @timestamp)
    if |parts| = 2 → (worker from "worker-timestamp", issue="")
  else:
    issue ← regex match [a-z]+-[a-z0-9]+(\.[0-9]+)?
```

### Key Properties

1. **Idempotency**: If an MR bead for the branch already exists, skip creation
2. **Priority inheritance**: Default priority comes from source issue, overridable via CLI
3. **Ephemeral beads**: MR beads are marked ephemeral (cleaned up after merge)
4. **Auto-cleanup**: After submit, polecat sends shutdown request to witness

---

## 10. Integration Branch Subsystem (cmd/mq_integration)

### Operations

| Command | Signature | Description |
|---------|-----------|-------------|
| create | epicID → Error | Create integration branch from main, push to origin |
| land | epicID → Error | Merge integration branch to target, delete, close epic |
| status | epicID → Error | Show integration branch status |

### Integration Branch Resolution

```
resolveEpicBranch(epic, rigPath, checker) =
  1. Metadata: getIntegrationBranchField(epic.Description)
  2. Configured template: buildIntegrationBranchName(template, epicID, title)
  3. Legacy {epic} template fallback (with branch existence checking)
  4. Return primary name (caller handles "not found")
```

### Landing Algorithm

```
runMqIntegrationLand(epicID) =
  1. Verify epic exists and is type "epic"
  2. Fetch latest from origin
  3. Resolve integration branch name
  4. Verify all MRs targeting branch are merged (or --force)
  5. Verify all epic children are closed (or --force)
  6. Idempotency check: is branch already ancestor of target?
  7. Create temporary worktree (isolated from running agents)
  8. Merge integration branch → target (--no-ff for merge commit)
  9. Run tests (unless --skip-tests)
  10. Verify merge produced file changes (guard against empty merges)
  11. Push to origin (with GT_INTEGRATION_LAND=1 env for pre-push hook)
  12. Close epic
  13. Delete integration branch (local + remote)
```

### ReadyToLand Predicate

```
ReadyToLand(aheadCount, childrenTotal, childrenClosed, pendingMRCount) ≡
    aheadCount > 0
  ∧ childrenTotal > 0
  ∧ childrenTotal = childrenClosed
  ∧ pendingMRCount = 0
```

### Key Invariants

1. **Worktree isolation**: Land uses a temporary worktree to avoid disrupting running agents
2. **File lock serialization**: A global file lock serializes ALL land operations per rig
3. **Empty merge guard**: Verifies the merge produced file changes (prevents silent data loss)
4. **Epic close before branch delete**: Ensures retriable state on crash between steps
5. **Idempotent re-run**: If branch is already ancestor of target, skips to cleanup

---

## 11. Cross-Package Algebraic Structure

### The Merge Pipeline as a Composition

```
Submit → Score → Batch → Stack → Gate → Push → PostMerge → ConvoyCheck
  ↓        ↓       ↓       ↓       ↓      ↓        ↓           ↓
 beads   Float64  List   gitState  Bool  gitState  beads     beads
```

### Shared Algebraic Patterns

**1. State Machines** (types.go, protocol):
Both MRStatus and MRPhase are finite state machines with validated transitions.
The protocol messages correspond to FSM edges (MERGE_READY = open→claimed,
MERGED = merging→merged, etc.).

**2. Scoring as a Linear Functional** (score.go):
```
ScoreMR = BaseScore + Σ wᵢ · fᵢ(input)
```
Each factor fᵢ is independently monotone, making the composite score a
**monotone linear functional** on the product lattice of (time, priority, retries).

**3. Bisection as Binary Search on Boolean Predicate** (batch.go):
```
gates(batch[0..k]) = true ∧ gates(batch[0..k+1]) = false
  → culprit ∈ batch[k+1..n]
```
The batch-then-bisect algorithm is classic binary search applied to the
predicate "does this subset of MRs pass quality gates?"

**4. Merge Slot as Distributed Mutex** (engineer.go, beads_merge_slot.go):
```
acquire → (Available ∨ Self) → proceed
acquire → Held → retry with exponential backoff
acquire → MaxRetries → fail(errMergeSlotTimeout)
```

**5. Fail-Open Semantics** (engineer.go, manager.go):
- Missing branch → skip MR (don't block queue)
- Beads query failure → empty queue (don't crash)
- Slot release failure → log warning (don't fail merge)
- Submodule check failure → log warning (continue merge)

**6. Idempotency** (mq_submit.go, mq_integration.go):
- MR submission is idempotent (FindMRForBranch checks first)
- Integration land is idempotent (IsAncestor check)
- Merge slot ensure is idempotent (check then create)

### Conservation Laws

1. **Batch conservation**: `|merged| + |culprits| + |conflicts| ≤ |batch|`
2. **State conservation**: Every MR reaches exactly one terminal state (merged, rejected, or
   stays open indefinitely in degenerate cases)
3. **Score positivity**: With default config, BaseScore (1000) minus MaxRetryPenalty (300)
   ensures scores are always positive (≥ 700 for fresh MRs)
4. **One-writer guarantee**: Only one refinery per rig can process the queue

---

## 12. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Component | Target | Why |
|-----------|--------|-----|
| batch.bisectBatch | Termination + correctness | Classic algorithm with formal proofs available |
| types.ValidateTransition | State machine well-formedness | Small, self-contained, all transitions enumerable |
| score.ScoreMR | Monotonicity + starvation-freedom | Linear functional with clear algebraic properties |

### Medium Priority (useful but more complex)

| Component | Target | Why |
|-----------|--------|-----|
| engineer.doMerge | Step ordering + failure handling | Complex multi-step with branching error paths |
| batch.BuildRebaseStack | Conflict isolation invariant | Rebuild-on-conflict maintains stack consistency |
| engineer.acquireMainPushSlot | Liveness + mutual exclusion | Distributed mutex with timeout |

### Low Priority (mostly operational, less formal structure)

| Component | Target | Why |
|-----------|--------|-----|
| manager.Queue | Score ordering correctness | Thin wrapper over score + sort |
| mq.GenerateMRID | Collision bounds | Probabilistic, not algebraic |
| protocol | Message type completeness | Type definitions, no processing logic |

### Cross-Cutting Theorems

1. **Bisection terminates**: The batch size strictly decreases at each recursive call
2. **Merge slot serialization**: At most one push to the default branch at any time
3. **Pre-verification safety**: Fast-path only fires when target HEAD hasn't moved
4. **Non-blocking delegation**: Conflict resolution doesn't stall the queue
5. **Idempotent operations**: Submit, land, and slot-ensure are all safe to call multiple times
6. **Fail-open preserves liveness**: Infrastructure errors lead to skipping, not blocking
