# Algebraic Survey: Communication Subsystem

**Subsystem**: Communication
**Packages**: `mail`, `nudge`, `events`, `hooks`, escalation (in `cmd/`, `beads/`), beacon (in `session/`)
**Purpose**: How agents TALK — mail delivers persistent bead-backed messages, nudge sends
ephemeral signals, escalation routes severity-graded alerts, hooks attach work to agents,
beacons announce sessions, and events log the audit trail.

## 1. Subsystem Overview

The Communication subsystem answers: given a set of agents (polecats, witnesses,
refineries, mayor, deacon) distributed across rigs, how do they exchange information
with appropriate durability, routing, and urgency guarantees?

```
Nudge (ephemeral)  ──→  Queue (file)  ──→  Hook drain  ──→  Agent context
                   └──→  Tmux (direct) ──→  Agent input

Mail (persistent)  ──→  Beads/Dolt    ──→  Mailbox     ──→  Agent inbox
                   └──→  Notification  ──→  Nudge path

Escalation         ──→  Bead + Route  ──→  Mail/Email/SMS ──→ Human/Agent

Hook               ──→  Bead status   ──→  SessionStart  ──→  Autonomous mode

Beacon             ──→  Tmux inject   ──→  Session picker ──→  Identity
```

### Package Dependency Graph (within subsystem)

```
nudge   ←── (no internal deps, leaf package; uses only stdlib + time)
mail    ←── nudge (queued fallback), tmux (delivery), beads (persistence), config
events  ←── config (no subsystem deps)
hooks   ←── config, tmux (no subsystem deps)
escalation (cmd/) ←── mail (routing), beads (storage), config (thresholds)
beacon (session/) ←── config (no subsystem deps)
```

**Key finding**: The subsystem has a clear layering: `nudge` is the ephemeral leaf,
`mail` builds on nudge + beads for persistence, `escalation` builds on mail for
severity routing. The `hooks` and `events` packages are independent leaves that
integrate at the orchestration layer (cmd/, daemon/).

### External Consumers

| Package | Primary Consumers |
|---------|-------------------|
| mail | cmd/mail*, witness, refinery, crew, mayor, deacon, polecat |
| nudge | cmd/nudge, mail/router (fallback), cmd/mail_check (hook drain) |
| events | daemon, witness, refinery, feed/curator |
| hooks | cmd/hooks*, session (lifecycle), daemon |
| escalation | cmd/escalate, witness (auto-bump), deacon |
| beacon | session/startup, cmd/prime |

---

## 2. Package: mail

**Files**: types.go (~1100 lines), router.go (~1650 lines), mailbox.go (~1000 lines),
delivery.go (~160 lines), resolve.go (~500 lines), bd.go (~105 lines)

### Type Algebra

The mail system is built on a **tagged union** of routing modes:

```
RoutingMode = Direct(to: Address) | Queue(name: String) | Channel(name: String)
            | List(name: String) | Announce(name: String) | Group(pattern: String)
```

Routing modes are mutually exclusive — a message uses exactly one:

```
Validate(msg) ≡ exactlyOne(msg.To ≠ "", msg.Queue ≠ "", msg.Channel ≠ "")
              ∧ msg.From ≠ "" ∧ msg.Subject ≠ ""
```

### Core Types

```
Priority = Low | Normal | High | Urgent

MessageType = Task | Scavenge | Notification | Reply

DeliveryMode = DeliveryQueue | DeliveryInterrupt

DeliveryState = Pending | Acked

Message {
  ID From To Subject Body: String, Timestamp: Time, Read: Bool,
  Priority: Priority, Type: MessageType, Delivery: DeliveryMode,
  ThreadID ReplyTo: String, Pinned Wisp: Bool,
  CC: List String,
  -- Queue-specific
  Queue Channel ClaimedBy: String, ClaimedAt: Time,
  -- Delivery tracking
  DeliveryState: DeliveryState, DeliveryAckedBy: String, DeliveryAckedAt: Time,
  SuppressNotify: Bool
}

BeadsMessage {
  ID Title Description Assignee: String, Priority: Nat,
  Status: String, CreatedAt: Time, Labels: List String,
  Pinned Wisp: Bool
}
```

### Message ↔ BeadsMessage Isomorphism

Messages are persisted as beads issues. The conversion is a **partial isomorphism**
mediated by labels:

```
toBeads : Message → BeadsMessage
  ID        → ID
  Subject   → Title
  Body      → Description
  To        → Assignee (identity form)
  Priority  → PriorityToBeads(p) : {Low→3, Normal→2, High→1, Urgent→0}
  From      → label "from:<address>"
  ThreadID  → label "thread:<id>"
  ReplyTo   → label "reply-to:<id>"
  CC        → labels "cc:<address>" (one per CC)
  Type      → label "msg-type:<type>"
  Queue     → label "queue:<name>"
  Channel   → label "channel:<name>"
  ClaimedBy → label "claimed-by:<address>"
  ClaimedAt → label "claimed-at:<rfc3339>"
  Delivery  → label "delivery:pending" | "delivery:acked"
  Read      → Status: "open" = unread, "closed" = read

fromBeads : BeadsMessage → Message
  (inverse of above, parsing labels back to fields)
```

**PriorityToBeads** is an order-reversing injection:
```
PriorityToBeads : Priority → Nat
  Urgent → 0, High → 1, Normal → 2, Low → 3

PriorityFromInt : Nat → Priority
  (left inverse: PriorityFromInt ∘ PriorityToBeads = id)
```

### Address Algebra

An address is a structured path identifying an agent:

```
Address = TownAgent(role: Role)
        | RigAgent(rig: String, role: Role, name: String)
        | Singleton(name: String)

Role = Mayor | Deacon | Overseer | Witness | Refinery | Crew | Polecat

-- Canonical forms:
  "mayor/"                    → TownAgent(Mayor)
  "gastown/witness"           → RigAgent("gastown", Witness, "")
  "gastown/polecats/Toast"    → RigAgent("gastown", Polecat, "Toast")
  "overseer"                  → Singleton("overseer")
```

**AddressToIdentity** normalizes liberally (Postel's Law):
```
AddressToIdentity : String → String
  "gastown/crew/polecats/Toast" → "gastown/polecats/Toast"  -- strip "crew/"
  "gastown/Toast/"              → "gastown/Toast"            -- strip trailing /
  "mayor"                       → "mayor/"                   -- add trailing /
```

### Router: Send as a Coproduct Dispatch

The router's `Send` operation dispatches on routing mode:

```
Send : Router × Message → Error
Send(r, msg) = match routingMode(msg) with
  | Direct(to)       → sendToSingle(r, msg)
  | List(name)       → sendToList(r, msg)     -- fan-out: ∀ member. sendToSingle
  | Queue(name)      → sendToQueue(r, msg)    -- single bead, workers claim
  | Announce(name)   → sendToAnnounce(r, msg) -- bulletin board with retention
  | Channel(name)    → sendToChannel(r, msg)  -- pub/sub broadcast
  | Group(pattern)   → sendToGroup(r, msg)    -- resolve @pattern, fan-out
```

**Fan-out operations** (List, Group) satisfy:
```
sendToList(r, msg) = ∀ addr ∈ expandList(msg.To).
    sendToSingle(r, msg{To = addr, ID = fresh()})
```

Each recipient gets a unique copy with a fresh ID. This ensures independent
read/unread tracking per recipient.

### Mailbox Operations

```
Mailbox { identity: String, workDir: String, beadsDir: String }

List      : Mailbox → List Message        -- all open, sorted by (priority, timestamp)
ListUnread: Mailbox → List Message        -- filter to unread
Get       : Mailbox × String → Message    -- by ID
MarkRead  : Mailbox × String → Error      -- close in beads
MarkUnread: Mailbox × String → Error      -- reopen in beads
Delete    : Mailbox × String → Error      -- archive/acknowledge
Count     : Mailbox → Nat
CountUnread: Mailbox → Nat
Search    : Mailbox × Query → List Message

-- Sort order:
sort(messages) = sortBy (PriorityToBeads ASC, Timestamp ASC) messages
```

**Query filtering**: Mailbox queries use `--assignee` to scope to recipient identity,
reducing memory under concurrent agent load.

### Two-Phase Delivery Protocol

Delivery tracking is a **two-state machine** with crash-safe transitions:

```
                 SendLabels()           AckLabels(recipient, time)
  ┌──────────┐  ──────────→  ┌──────────┐  ──────────────────→  ┌────────┐
  │ (none)   │               │ Pending  │                       │ Acked  │
  └──────────┘               └──────────┘                       └────────┘
```

**Crash safety**: The ack label sequence is ordered for partial-write recovery:
```
AckLabelSequence(recipient, time) =
  [ "delivery-acked-by:<recipient>",   -- step 1: who
    "delivery-acked-at:<rfc3339>",     -- step 2: when
    "delivery:acked" ]                 -- step 3: state transition (commits)
```

If a crash occurs between steps 1-2, the message remains `Pending` (safe).
The final label is the commit point.

**Idempotency**: `AckLabelSequenceIdempotent` reuses existing timestamps on retry,
preventing duplicate labels. Multiple acks by the same recipient are no-ops.

### Queue Claiming (TOCTOU-Safe)

Queue messages use a **claim-then-verify** protocol:

```
Claim(mailbox, queueName) =
  messages ← list(queue:name, status:open, unclaimed)
  candidate ← first(sortBy(timestamp ASC, messages))
  addLabels(candidate, ["claimed-by:<identity>", "claimed-at:<now>"])
  -- Post-claim verification (TOCTOU defense):
  reread ← get(candidate.ID)
  if reread.ClaimedBy ≠ identity then
    Error("claimed by another worker")
  else
    AcknowledgeDeliveryBead(candidate.ID)
    return candidate
```

### Wisp Detection (Auto-Ephemeral)

Protocol messages are automatically classified as wisps:

```
shouldBeWisp(msg) = msg.Wisp ∨ msg.Subject.HasPrefix(protocolPrefix)

protocolPrefix ∈ { "POLECAT_", "NUDGE", "LIFECYCLE", "MERGED",
                   "MERGE_", "START_WORK", "WORK_DONE" }
```

Wisps are not synced to git, enabling auto-cleanup of lifecycle traffic.

### Group Resolution

```
ResolveGroupAddress : String → List Address

@town       → agents with rig = null (mayor, deacon)
@witnesses  → all witnesses across rigs
@rig/X      → all agents in rig X
@crew/X     → all crew in rig X
@polecats/X → all polecats in rig X
@dogs       → all deacon dogs
@refineries → all refineries
@overseer   → ["overseer"]
```

Resolution queries agent beads (both issues and wisps tables) filtered by
status ∈ {open, in_progress, hooked, pinned}.

### Lean 4 Formalization Candidates

1. **Routing mode exclusivity**: `Validate(msg)` ensures exactly one routing path
2. **Priority order reversal**: `PriorityFromInt ∘ PriorityToBeads = id` (round-trip)
3. **Fan-out cardinality**: `|sendToList results| = |expandList members|`
4. **Delivery state machine**: Pending → Acked is the only valid transition
5. **Claim linearizability**: Post-verification ensures exactly one claimer wins
6. **Ack idempotency**: Multiple acks produce identical label sets

---

## 3. Package: nudge

**Files**: queue.go (~317 lines)

### Core Abstraction

Nudges are **ephemeral, file-backed signals** with TTL-based expiry. They exist
outside Dolt — no beads, no commits, no persistence guarantees beyond the filesystem.
This makes them zero-cost for routine agent-to-agent chatter.

### Type Algebra

```
NudgePriority = Normal | Urgent

QueuedNudge {
  Sender Message Priority: String,
  Timestamp ExpiresAt: Time
}

-- TTL constants:
DefaultNormalTTL = 30 min
DefaultUrgentTTL = 2 hours
MaxQueueDepth   = 50
StaleClaimThreshold = 5 min
```

### Queue Storage Model

The queue is a **filesystem-based FIFO** with atomic operations:

```
Storage: <townRoot>/.runtime/nudge_queue/<session>/
Format:  <nanosecond-timestamp>-<hex-random>.json
Content: JSON(QueuedNudge)
```

Each nudge is an independent file. This enables:
- Atomic enqueue (single file write)
- Atomic claim (single file rename)
- Lock-free concurrent access
- FIFO ordering via filename sorting

### Key Operations

```
Enqueue : TownRoot × Session × QueuedNudge → Error
  pre:  Pending(townRoot, session) < MaxQueueDepth
  post: file created with defaults applied
  defaults:
    Timestamp ← now()       if zero
    Priority  ← Normal      if empty
    ExpiresAt ← Timestamp + TTL(Priority)

Drain : TownRoot × Session → List QueuedNudge × Error
  1. Sweep orphaned claims (rename .claimed.* → .json if age > StaleClaimThreshold)
  2. Sort entries by name (FIFO)
  3. Atomic claim: rename .json → .claimed.<unique-suffix>
     (only one drainer wins; losers get ENOENT and skip)
  4. Filter expired (ExpiresAt < now → discard)
  5. Remove .claimed files after successful processing
  return: ordered, non-expired nudges

Pending : TownRoot × Session → Nat
  count of .json files in queue directory (fast, no validation)

FormatForInjection : List QueuedNudge → String
  -- Separates urgent from normal, wraps in <system-reminder> tags
  urgent:  "[URGENT from <sender>] <message>" + "Handle urgent nudges before continuing"
  normal:  "[from <sender>] <message>" + "Continue current work unless higher priority"
```

### Concurrency Safety

**Double-delivery prevention** via atomic rename:
```
claim(file) = rename(file.json, file.claimed.<uuid>)
  -- Only one process can win the rename
  -- Losers receive ENOENT and skip
  -- UUID suffix prevents Windows MOVEFILE_REPLACE_EXISTING collisions
```

**Orphan recovery** via stale claim detection:
```
sweepOrphans(dir) = ∀ f ∈ dir.
  if f.name.HasSuffix(".claimed.*") ∧ f.modTime < now - StaleClaimThreshold then
    rename(f, f.withSuffix(".json"))  -- requeue for next drain
```

This ensures that crashed drainers cannot permanently lose nudges.

**Enqueue collision prevention** via randomized filenames:
```
filename = fmt.Sprintf("%019d-%s.json", time.Now().UnixNano(), randomHex(4))
```

### Delivery Modes (Three Strategies)

```
DeliveryMode = Immediate | Queue | WaitIdle

Immediate:  tmux.NudgeSession(session, message)
  -- Interrupts in-flight tool calls
  -- Use: emergency break-through

Queue:      nudge.Enqueue(townRoot, session, nudge)
  -- Zero interruption, cooperative pickup via hook
  -- Use: background notifications

WaitIdle:   (default, hybrid)
  err ← tmux.WaitForIdle(session, timeout)
  match err with
  | nil      → tmux.NudgeSession(session, message)     -- agent idle, safe to deliver
  | Timeout  → nudge.Enqueue(townRoot, session, nudge)  -- busy, queue for later
  | Terminal → error                                     -- session dead
  | QueueErr → tmux.NudgeSession(session, message)      -- queue failed, force-deliver
```

### Hook Integration (Turn-Boundary Drain)

```
UserPromptSubmit hook → gt mail check --inject →
  session ← tmux.CurrentSessionName()
  nudges  ← nudge.Drain(townRoot, session)
  if len(nudges) > 0:
    stdout ← nudge.FormatForInjection(nudges)
    -- Injected as <system-reminder> into agent context
```

This decouples sender from receiver: the sender writes to the queue, the
receiver's hook drains it at the natural turn boundary.

### Tmux Delivery Implementation

```
NudgeSession(session, message) =
  acquire(sessionNudgeLock[session], timeout=30s)
  exitCopyMode(session)
  sanitize(message)
  sendKeys(session, message, chunked=512)  -- split large messages
  wait(500ms)                               -- let tmux process
  sendEscape()                              -- exit vim mode
  wait(600ms)                               -- exceed readline keyseq-timeout (500ms)
  sendEnter(retries=3)                      -- submit with retries
  sigwinch(session)                         -- wake detached pane
  release(lock)
```

The per-session lock (channel-based semaphore) serializes concurrent nudges,
preventing garbled tmux input.

### Mail-Nudge Hybrid (Router Integration)

When mail is delivered, the router uses nudge as the notification channel:

```
notifyRecipient(msg) =
  if isMuted(msg.To) then return
  sessions ← sessionsFor(msg.To)
  ∀ session ∈ sessions.
    err ← tmux.WaitForIdle(session, IdleNotifyTimeout)
    match err with
    | nil     → tmux.NudgeSession(session, notification)
    | Timeout → nudge.Enqueue(townRoot, session, nudge{Sender: msg.From, Message: notification})
    | _       → tmux.NudgeSession(session, notification)  -- last resort
```

### Lean 4 Formalization Candidates

1. **FIFO ordering**: Drain returns nudges in enqueue order (filename sort = timestamp order)
2. **At-most-once delivery**: Atomic rename ensures each nudge delivered to exactly one drainer
3. **Orphan recovery**: Stale claims are requeued (no permanent loss from crashes)
4. **TTL monotonicity**: Expired nudges are never delivered (ExpiresAt < now → discard)
5. **Queue depth bound**: Pending count never exceeds MaxQueueDepth
6. **WaitIdle fallback chain**: Exactly one delivery path taken per nudge

---

## 4. Escalation Protocol

**Files**: cmd/escalate.go, cmd/escalate_ack.go, cmd/escalate_close.go,
cmd/escalate_stale.go, beads/beads_escalation.go (~1289 lines total)

### Type Algebra

```
Severity = Critical | High | Medium | Low
  -- Maps to bead priorities: Critical→0, High→1, Medium→2, Low→3

EscalationRoute {
  Severity: Severity,
  Actions: List Action
}

Action = BeadAction | MailAction(target: Address) | EmailAction(to: String)
       | SMSAction(to: String) | SlackAction(channel: String)

StaleConfig {
  Threshold: Duration,    -- default 4 hours
  MaxReescalations: Nat   -- default 2 (prevents infinite loops)
}
```

### Escalation State Machine

```
                Create(severity)        Ack(id)            Close(id, reason)
  ┌──────────┐  ──────────→  ┌──────────┐  ──────→  ┌──────────┐  ──────→  ┌────────┐
  │ (none)   │               │  Open    │           │  Acked   │          │ Closed │
  └──────────┘               └──────────┘           └──────────┘          └────────┘
                                   │                                           ▲
                                   │  Stale(threshold)                         │
                                   ├──────────→ BumpSeverity ─────────────────→│
                                   │            (if reescalations < max)  Close(auto)
                                   └───────────────────────────────────────────┘
                                                (if reescalations ≥ max)
```

### Key Operations

```
Escalate : Description × Severity × Message → Bead × Error
  1. Create bead with escalation type and severity-mapped priority
  2. Route to targets based on severity config
  3. Send mail/email/SMS/Slack per route actions

Ack : BeadID → Error
  -- Prevents re-escalation by stale detector
  -- Adds "escalation:acked" label

Close : BeadID × Reason → Error
  -- Marks resolved, adds close reason

Stale : Threshold × MaxReescalations → List Bead
  -- Finds unacked escalations older than threshold
  -- Bumps severity: Medium→High→Critical
  -- Guards against infinite loops (max reescalation count)
  -- Automatically closes if max reached
```

### Severity Routing

```
Route(severity) = match severity with
  | Critical → [bead, mail(mayor), mail(overseer), email(oncall), sms(oncall)]
  | High     → [bead, mail(mayor), mail(witness)]
  | Medium   → [bead, mail(witness)]
  | Low      → [bead]
```

Routes are configurable via `settings/escalation.json`.

### Lean 4 Formalization Candidates

1. **Severity ordering**: Critical < High < Medium < Low (total order, bump is predecessor)
2. **Stale convergence**: After max reescalations, escalation terminates (no infinite loop)
3. **Ack prevents bump**: Acked escalations are never re-escalated
4. **Route monotonicity**: Higher severity routes are supersets of lower severity routes

---

## 5. Hook Mechanism (Bead Attachment)

**Files**: cmd/hook.go, cmd/hooks_config.go, cmd/hooks_sync.go, hooks/ package

### Core Abstraction

A hook is a **durable attachment** of a bead to an agent. The hooked bead survives
session restarts, context compaction, handoffs, and agent failure. It is the
primary mechanism for assigning work.

### Type Algebra

```
HookState = Unhooked | Hooked(beadID: String)

-- Agent bead fields:
AgentBead {
  ...,
  HookBead: Option String,     -- currently hooked bead ID
  Status: AgentStatus           -- includes "hooked" as valid state
}

-- Hook operation:
Hook : Agent × BeadID → Error
  1. Set agent.HookBead = beadID
  2. Set bead.Status = "hooked"
  3. Set bead.Assignee = agent.Address
```

### Hook Lifecycle

```
  ┌──────────────┐     gt hook <id>      ┌──────────────┐
  │   Unhooked   │  ─────────────────→   │    Hooked    │
  │  (idle agent)│                       │ (working)    │
  └──────────────┘                       └──────┬───────┘
                                                │
                              ┌─────────────────┼─────────────────┐
                              │                 │                 │
                         gt done          gt handoff         session death
                              │                 │                 │
                         ┌────▼────┐      ┌─────▼─────┐    ┌─────▼─────┐
                         │ Refinery│      │ New session│    │  Witness  │
                         │  merge  │      │ picks up  │    │ recovery  │
                         └─────────┘      │ same hook │    └───────────┘
                                          └───────────┘
```

### Hook Config System (Three-Tier Merge)

```
EffectiveConfig = merge(BaseConfig, RoleOverride, AgentOverride)

BaseConfig: shared hooks for all agents
RoleOverride: role-specific additions (e.g., witness patrol hooks)
AgentOverride: per-agent customization

merge(base, override) = {
  ∀ hook ∈ base ∪ override:
    if hook ∈ override then override[hook]
    else base[hook]
}
```

### SessionStart Integration

```
SessionStart hook fires → gt prime --hook →
  hooked ← gt hook show (current agent)
  if hooked ≠ nil then
    inject AUTONOMOUS WORK MODE instructions
    inject bead details
    agent begins execution immediately
```

This is the propulsion mechanism: hook → prime → autonomous execution.

---

## 6. Beacon (Startup Announcements)

**Files**: session/startup.go (~131 lines)

### Type Algebra

```
BeaconConfig {
  Recipient Sender Topic MolID: String,
  IncludePrimeInstruction ExcludeWorkInstructions: Bool
}

BeaconFormat = "[GAS TOWN] <recipient> <- <sender> • <timestamp> • <topic[:mol-id]>"
```

### Beacon Topics

```
Topic = ColdStart | Handoff | Assigned | Attached | Patrol | MolID(String)

-- Each topic triggers different work instructions:
instructions(ColdStart) = "Run `gt prime --hook` and begin work on your hook."
instructions(Handoff)   = "Run `gt prime --hook` and begin work on your hook."
instructions(Assigned)  = "Run `gt prime --hook` and begin work on your hook."
instructions(Patrol)    = <patrol-specific instructions>
instructions(MolID(id)) = "Run `gt prime --hook` and begin work on your hook."
```

### BeaconAddress Format

Beacons use a non-path format to prevent LLM misinterpretation:

```
BeaconAddress : AgentIdentity → String
  Polecat("Toast", "gastown") → "polecat Toast (rig: gastown)"
  Witness("gastown")          → "witness (rig: gastown)"
  Mayor                       → "mayor"
```

The parenthetical format prevents agents from treating the address as a filesystem
path and attempting to `cd` to it.

### Key Operation

```
FormatStartupBeacon : BeaconConfig → String
  -- Pure function, no side effects
  -- Injected into tmux pane at session creation
  -- Visible in /resume session picker for identity
```

---

## 7. Package: events

**Files**: events/events.go (~371 lines)

### Core Abstraction

Events provide the **audit trail** — an append-only JSONL log of all system activity.
Events are the observation channel; they don't affect system behavior but enable
the feed curator and monitoring.

### Type Algebra

```
Event {
  Timestamp Source Type Actor: String,
  Message: String, Payload: Map String Any,
  Visibility: Visibility
}

Visibility = Internal | Feed | Both

EventType = Sling | Hook | Unhook | Handoff | Done
          | Spawn | Kill | Boot | Halt
          | SessionStart | SessionEnd | SessionDeath | MassDeath
          | PatrolStarted | PolecatChecked | PolecatNudged | PatrolComplete
          | EscalationSent | EscalationAcked | EscalationClosed
          | MergeStarted | Merged | MergeFailed | MergeSkipped
          | SchedulerEnqueue | SchedulerDispatch | SchedulerDispatchFailed
```

### Storage Model

```
Raw log:    <townRoot>/.events.jsonl    (all events, append-only, flock-synchronized)
Curated:    <townRoot>/.feed.jsonl      (visibility ∈ {Feed, Both}, processed by curator)
```

### Key Operations

```
Emit : Event → Error
  -- Append JSON line to .events.jsonl with flock
  -- Fire-and-forget (errors logged, not propagated)

-- Payload helpers (typed event constructors):
SlingPayload : BeadID × Rig × Formula → Map String Any
HookPayload  : BeadID × Agent → Map String Any
DonePayload  : BeadID × Agent × ExitType → Map String Any
MergePayload : BeadID × Branch × Result → Map String Any
```

### Visibility Filter (Feed Curator)

```
Curator.processLine(event) =
  if event.Visibility ∉ {Feed, Both} then drop
  if shouldDedupe(event) then drop
  writeFeedEvent(event)
```

This is the projection from the full event stream to the human-visible feed.

---

## 8. Cross-Package Algebraic Structure

### The Communication Duality: Persistent vs Ephemeral

The subsystem is organized around a fundamental **duality**:

```
                    Persistent (Dolt)           Ephemeral (Filesystem)
                    ─────────────────           ──────────────────────
Mechanism:          Mail                        Nudge
Storage:            Beads (Dolt commits)        JSON files (TTL-expiring)
Survives restart:   Yes                         No (but recovers crashes)
Cost per message:   1 Dolt commit               0 commits
Delivery:           Inbox query                 Hook drain / tmux inject
Addressing:         Identity-based              Session-based
Use case:           Protocol, handoff           Status, health, wake-up
```

**Design rule**: Default to nudge. Only use mail when the message must survive
the recipient's session death.

### The Notification Cascade

When mail arrives, the notification path cascades through nudge:

```
Mail.Send(msg)
  └→ Router.notifyRecipient(msg)
       └→ if idle:  Tmux.NudgeSession(direct)     ← immediate
          if busy:  Nudge.Enqueue(queue)           ← cooperative
          if dead:  (no notification, mail waits)  ← durable
```

This composition means mail guarantees delivery (via Dolt persistence) while
nudge optimizes latency (via immediate/queued delivery).

### Shared Algebraic Patterns

**1. Fail-Open Semantics** (mail routing, nudge delivery, escalation):
```
Mail:       bd command fails → skip notification (message still persisted)
Nudge:      queue full → try immediate delivery (graceful degradation)
Escalation: route action fails → continue with remaining actions
```

All communication subsystems degrade gracefully rather than blocking.

**2. Idempotency** (delivery ack, queue claim, escalation ack):
```
DeliveryAck:   multiple acks → same label set (timestamp reuse)
QueueClaim:    re-claim by same agent → no-op (post-verification)
EscalationAck: re-ack → no-op (label already present)
```

**3. At-Most-Once Delivery** (nudge queue, mail queue):
```
Nudge:  atomic rename claim → exactly one drainer wins
Mail:   post-claim verify → exactly one claimer wins
```

Both use optimistic concurrency with verification rather than locks.

**4. Severity/Priority Ordering** (mail, escalation):
```
Mail:       Urgent(0) > High(1) > Normal(2) > Low(3)
Escalation: Critical(0) > High(1) > Medium(2) > Low(3)

-- Same total order, same numeric mapping
-- Sort ascending = highest priority first
```

**5. TTL/Expiry Windows** (nudge, feed, escalation stale):
```
Nudge:      Normal=30min, Urgent=2hr    (discard stale signals)
Feed:       Dedup window (configurable)  (prevent duplicate events)
Escalation: Stale threshold=4hr          (auto-bump unacked)
```

All use temporal windows, but with different semantics:
- Nudge: TTL means "discard if too old" (ephemeral)
- Feed: window means "deduplicate within" (idempotent)
- Escalation: threshold means "re-escalate after" (progressive)

### The Routing Algebra

Messages are routed through a **coproduct of addressing modes**:

```
Target = Direct(Address)
       | List(Name)      -- expand to Set Address, fan-out
       | Queue(Name)     -- single message, first-claimer-wins
       | Channel(Name)   -- pub/sub, broadcast to subscribers
       | Announce(Name)  -- bulletin board, retain N messages
       | Group(Pattern)  -- @-pattern, resolve and fan-out

route : Target → Nat   -- cardinality of delivery
route(Direct(a))     = 1
route(List(n))       = |expandList(n)|
route(Queue(n))      = 1   -- single message, multiple potential claimers
route(Channel(n))    = |subscribers(n)|
route(Announce(n))   = 1   -- single bulletin entry
route(Group(p))      = |resolveGroup(p)|
```

Fan-out modes (List, Channel, Group) create independent copies.
Single-copy modes (Direct, Queue, Announce) share the original.

### The Hook-Mail-Nudge Triangle

These three mechanisms serve complementary roles in work coordination:

```
        Hook (durable assignment)
       /                         \
      /  "what to work on"        \  "work is done"
     /                             \
    ▼                               ▼
  Nudge ◄─────────────────────── Mail
  (ephemeral signal)         (persistent record)
     "wake up"                "protocol message"
```

- **Hook** assigns work (bead attachment, survives everything)
- **Mail** records events (POLECAT_DONE, MERGE_READY — must survive session death)
- **Nudge** coordinates in real-time (health checks, wake-ups — ephemeral)

### Conservation Laws

1. **Mail**: Every `Send` creates exactly `route(target)` bead entries
2. **Nudge**: Every `Enqueue` creates exactly 1 file; every successful `Drain` removes it
3. **Events**: Every `Emit` appends exactly 1 line to `.events.jsonl`
4. **Delivery**: `|pending| + |acked| = |sent|` (delivery states partition sent messages)
5. **Escalation**: `reescalation_count ≤ MaxReescalations` (bounded severity bumps)

### Category-Theoretic View

The communication subsystem forms a category where:
- **Objects**: Agent inboxes, queues, channels, nudge queues, event logs
- **Morphisms**: Message delivery operations

Key functors:
```
Message  → BeadsIssue    (persistence: mail type to bead representation)
Message  → Notification  (projection: mail to nudge for real-time alert)
Event    → FeedEvent     (visibility filter: raw to curated)
Severity → Priority      (escalation to bead priority mapping)
Address  → Session       (identity to tmux session resolution)
```

The composition `Message → BeadsIssue → Notification → Tmux` is the full
mail delivery pipeline. The `Message → BeadsIssue` functor is the persistence
layer; `Notification → Tmux` is the real-time notification layer.

---

## 9. Summary: Formalization Priority

### High Priority (rich algebraic structure, clear invariants)

| Package | Target | Why |
|---------|--------|-----|
| mail/delivery | Two-phase delivery (Pending → Acked) | Crash-safe state machine, idempotency proofs |
| mail/router | Routing coproduct dispatch | Fan-out cardinality, routing exclusivity |
| nudge/queue | Atomic claim + orphan recovery | At-most-once delivery, crash recovery correctness |

### Medium Priority (simpler structure, fewer invariants)

| Package | Target | Why |
|---------|--------|-----|
| mail/types | Priority/Address algebra | Order properties, round-trip isomorphisms |
| escalation | Severity state machine | Bounded re-escalation, stale convergence |
| events | Visibility filter projection | Event stream properties |

### Cross-Cutting Theorems

1. **Persistent-ephemeral duality**: Mail persists iff message must survive session death; nudge otherwise
2. **Notification cascade**: Mail delivery → nudge notification is a natural transformation
3. **At-most-once everywhere**: Both mail queue and nudge queue guarantee at-most-once via optimistic concurrency
4. **Fail-open is universal**: All communication channels degrade gracefully on infrastructure failure
5. **Priority is a total order**: The same ordering `Urgent > High > Normal > Low` is shared across mail and escalation
6. **Fan-out preserves identity**: Each recipient of a fan-out message gets an independent copy with fresh ID
