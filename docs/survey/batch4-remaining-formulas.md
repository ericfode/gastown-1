# Batch 4: Remaining Operational Formulas -- Cell Language Survey

**Date**: 2026-03-08
**Surveyor**: morpheus
**Batch**: 4 of 4 -- remaining operational formulas

---

## Summary

| # | Formula | Type | Steps | Category | Key Challenge |
|---|---------|------|-------|----------|---------------|
| 1 | mol-boot-triage | Infra/watchdog | 5 | DIRECT | Script-heavy sequential chain |
| 2 | mol-convoy-cleanup | Infra/dog | 5 | DIRECT | Variables map to `input` params |
| 3 | mol-convoy-feed | Infra/dog | 5 | EXTENDED | Iteration over ready issues needs `each>` |
| 4 | mol-deacon-patrol | Patrol loop | 26 | EXTENDED | Complex DAG with parallel branches, loop-back |
| 5 | mol-dep-propagate | Infra/dog | 5 | EXTENDED | Cross-rig iteration over dependents |
| 6 | mol-digest-generate | Infra/dog | 5 | EXTENDED | Multi-rig data collection needs `map` |
| 7 | mol-orphan-scan | Infra/dog | 8 | EXTENDED | Parallel scans (issues/molecules/wisps) + triage |
| 8 | mol-refinery-patrol | Patrol loop | 14 | EXTENDED | Conditional config, loop-back, merge protocol |
| 9 | mol-session-gc | Infra/dog | 6 | DIRECT | Sequential script cells |
| 10 | mol-sync-workspace | Infra/sync | 10 | DIRECT | Sequential script cells with config vars |
| 11 | mol-witness-patrol | Patrol loop | 9 | EXTENDED | Complex survey logic, Task-tool parallelism |
| 12 | towers-of-hanoi-7 | Durability test | 129 | DIRECT | Pure linear chain, 127 pre-computed moves |
| 13 | towers-of-hanoi-9 | Durability test | 513 | DIRECT | Same pattern, 511 moves |
| 14 | towers-of-hanoi-10 | Durability test | 1025 | DIRECT | Same pattern, 1023 moves |

**Totals**: DIRECT: 7, EXTENDED: 7, GAP: 0

---

## 1. mol-boot-triage

**Type**: Infrastructure / watchdog (ephemeral per-tick)
**Category**: DIRECT
**Steps**: observe -> decide -> act -> cleanup -> exit (linear chain)

### Cell Translation

```cell
## boot-triage {

  # observe : script
    ``` sh
    echo "=== Deacon State ==="
    tmux has-session -t hq-deacon 2>/dev/null && echo "alive" || echo "dead"
    gt peek deacon --lines 20

    echo "=== Agent Bead ==="
    bd show hq-deacon 2>/dev/null

    echo "=== Recent Activity ==="
    gt feed --since 10m --plain | head -20
    ls -lt $GT_ROOT/.beads-wisp/*.wisp.json 2>/dev/null | head -5

    echo "=== Deacon Mail ==="
    gt mail inbox deacon 2>/dev/null | head -10
    ```
  #/

  # decide : llm
    - observe
    @ cost(max: 4000) @ quality(min: adequate)
    > You are Boot, the Deacon's watchdog. Given these observations:
    >
    > {{observe}}
    >
    > Apply the decision matrix:
    > - Dead session -> START
    > - Alive + active output -> NOTHING
    > - Alive + idle < 5 min -> NOTHING
    > - Alive + idle 5-15 min + no mail -> NOTHING
    > - Alive + idle 5-15 min + has mail -> NUDGE
    > - Alive + idle > 15 min -> WAKE
    > - Alive + stuck (repeated errors) -> INTERRUPT
    >
    > Output exactly one of: NOTHING, NUDGE, WAKE, INTERRUPT, START
    format> json {
      "action": "NOTHING | NUDGE | WAKE | INTERRUPT | START",
      "reason": "string"
    }
  #/

  # act : script
    - decide
    ``` sh
    ACTION="{{decide.action}}"
    case "$ACTION" in
      NOTHING)
        echo "No action needed."
        ;;
      NUDGE)
        gt nudge --mode=queue deacon "Boot check-in: you have pending work"
        ;;
      WAKE)
        tmux send-keys -t hq-deacon Escape
        sleep 1
        gt nudge deacon "Boot wake: please check your inbox and pending work"
        ;;
      INTERRUPT)
        gt mail send deacon -s "INTERRUPT: Boot detected stuck state" \
          -m "Boot observed stuck state. Please check your context."
        ;;
      START)
        gt deacon start
        echo "Boot started Deacon session"
        ;;
    esac
    ```
  #/

  # cleanup : script
    - act
    ``` sh
    # Archive stale handoffs from Deacon inbox (> 1 hour, subject contains HANDOFF)
    gt mail inbox deacon --json 2>/dev/null | jq -r '.[] | select(.age > 3600 and (.subject | test("(?i)handoff"))) | .id' | while read id; do
      gt mail archive "$id"
    done

    # Archive Boot's own old mail
    gt mail inbox boot --json 2>/dev/null | jq -r '.[].id' | while read id; do
      gt mail archive "$id"
    done
    ```
  #/

  # exit : script
    - cleanup
    ``` sh
    if [ "$GT_DEGRADED" = "true" ]; then
      echo "Boot triage complete: {{decide.action}}"
      exit 0
    else
      gt mail send boot -s "Boot handoff" -m "Completed triage cycle.
    Action: {{decide.action}}
    Time: $(date)"
    fi
    ```
  #/

  observe -> decide -> act -> cleanup -> exit

##/
```

### Notes
- Linear chain maps directly. The `decide` step is the only LLM cell; the rest are pure scripts.
- The original embeds the decision matrix in prose; Cell version uses a proper LLM cell with format constraint.
- `{{decide.action}}` requires field projection from JSON output.

---

## 2. mol-convoy-cleanup

**Type**: Infrastructure / dog work
**Category**: DIRECT
**Steps**: load-convoy -> generate-summary -> archive-convoy -> notify-overseer -> return-to-kennel

### Cell Translation

```cell
## convoy-cleanup {

  input param.convoy : string required

  # load-convoy : script
    ``` sh
    gt hook
    bd show {{param.convoy}} --json
    # Verify status is closed or all tracked issues closed
    ```
  #/

  # generate-summary : llm
    - load-convoy
    @ cost(max: 8000) @ quality(min: adequate)
    > Given this convoy data:
    >
    > {{load-convoy}}
    >
    > Generate a summary including:
    > - Total duration (start to finish)
    > - Issues by type (task, bug, feature)
    > - Contributors (unique assignees)
    > - Key outcomes
    format> json {
      "summary_text": "string",
      "duration": "string",
      "issue_count": "number",
      "contributor_list": "string"
    }
  #/

  # archive-convoy : script
    - generate-summary
    ``` sh
    bd close {{param.convoy}} --reason="Convoy complete"
    bd show {{param.convoy}}
    bd sync
    ```
  #/

  # notify-overseer : script
    - archive-convoy
    ``` sh
    gt mail send mayor/ -s "Convoy complete: {{param.convoy}}" \
      -m "Convoy {{param.convoy}} has completed and been closed.

    {{generate-summary.summary_text}}

    Duration: {{generate-summary.duration}}
    Issues: {{generate-summary.issue_count}}
    Contributors: {{generate-summary.contributor_list}}"
    ```
  #/

  # return-to-kennel : script
    - notify-overseer
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: convoy-cleanup
    Convoy: {{param.convoy}}
    Status: COMPLETE

    Ready for next assignment."
    ```
  #/

  load-convoy -> generate-summary -> archive-convoy -> notify-overseer -> return-to-kennel

##/
```

### Notes
- `[vars]` map to `input param.X` declarations. Computed vars are just cell outputs.
- `generate-summary` is a natural LLM cell; the original expects the agent to reason about data.
- `[squash]` metadata has no Cell equivalent yet -- could be a molecule-level annotation.

---

## 3. mol-convoy-feed

**Type**: Infrastructure / dog work (dispatch)
**Category**: EXTENDED
**Steps**: load-convoy -> check-capacity -> dispatch-work -> report-results -> return-to-kennel

### Cell Translation

```cell
## convoy-feed {

  input param.convoy : string required

  # load-convoy : script
    ``` sh
    gt hook
    gt convoy status {{param.convoy}} --json
    bd blocked --json
    ```
  #/

  # check-capacity : script
    - load-convoy
    ``` sh
    # For each rig with ready issues, check polecat capacity
    gt polecats gastown --json 2>/dev/null
    gt polecats beads --json 2>/dev/null
    ```
  #/

  # dispatch-work : script
    - check-capacity
    ``` sh
    # Parse ready issues from load-convoy, dispatch each
    # This is a loop over a dynamic list -- shell handles iteration
    echo "{{load-convoy}}" | jq -r '.ready_issues[]?.id' | while read issue_id; do
      RIG=$(echo "$issue_id" | cut -d- -f1)
      gt sling "$issue_id" "$RIG" 2>&1 || echo "FAIL: $issue_id"
    done
    ```
  #/

  # report-results : script
    - dispatch-work
    ``` sh
    gt mail send deacon/ -s "Convoy fed: {{param.convoy}}" \
      -m "Convoy feeding complete. See dispatch output above."
    ```
  #/

  # return-to-kennel : script
    - report-results
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: convoy-feed
    Convoy: {{param.convoy}}
    Status: COMPLETE

    Ready for next assignment."
    ```
  #/

  load-convoy -> check-capacity -> dispatch-work -> report-results -> return-to-kennel

##/
```

### Notes
- The iteration over ready issues and per-rig capacity checks would ideally use Cell `each>` or `map`, but the dynamic list is only known at runtime. Shell-level iteration inside a script cell is the pragmatic solution.
- Cell `map # name over collection as var` could apply if load-convoy produced a structured list, but the dispatch logic includes conditional rig routing that keeps it in script territory.
- Categorized EXTENDED because a pure Cell expression of the per-issue fanout would need `map`.

---

## 4. mol-deacon-patrol

**Type**: Patrol loop (26 steps, complex DAG with parallel branches)
**Category**: EXTENDED
**Steps**: heartbeat -> inbox-check -> {orphan-process-cleanup, test-pollution-cleanup, gate-evaluation, check-convoy-completion} -> ... -> loop-or-exit

### Cell Translation

```cell
## deacon-patrol {

  -- Step 1: heartbeat (root)
  # heartbeat : script
    ``` sh
    gt deacon heartbeat "starting patrol cycle"
    ```
  #/

  -- Step 2: inbox-check
  # inbox-check : script
    - heartbeat
    ``` sh
    bd mol wisp gc --closed --force
    bd mol wisp gc --age 1h --force
    gt mail inbox
    # Process each message type: HELP, LIFECYCLE, DOG_DONE,
    # CONVOY_NEEDS_FEEDING, RECOVERED_BEAD
    # Archive after handling
    ```
  #/

  -- Steps 3-4: parallel branch after inbox-check
  # orphan-process-cleanup : script
    - inbox-check
    ``` sh
    gt deacon cleanup-orphans
    ```
  #/

  # test-pollution-cleanup : script
    - inbox-check
    ``` sh
    gt dolt kill-imposters 2>/dev/null || true
    TMPDIR="${TMPDIR:-/tmp}"
    for dir in "$TMPDIR"/beads-test-dolt-* "$TMPDIR"/beads-bd-tests-*; do
      [ -d "$dir" ] || continue
      if ! lsof +D "$dir" >/dev/null 2>&1; then
        chmod -R u+w "$dir" 2>/dev/null
        rm -rf "$dir" && echo "Cleaned: $(basename "$dir")"
      fi
    done
    # Clean stale PID files, dead dog worktrees (abbreviated)
    ```
  #/

  # gate-evaluation : script
    - inbox-check
    ``` sh
    bd gate list --json
    # Close timer gates where elapsed > timeout
    ```
  #/

  # dispatch-gated-molecules : script
    - gate-evaluation
    ``` sh
    bd ready --gated --json
    # For each ready molecule, dispatch to appropriate rig
    ```
  #/

  # check-convoy-completion : script
    - inbox-check
    ``` sh
    gt convoy list
    gt convoy check
    ```
  #/

  # resolve-external-deps : script
    - check-convoy-completion
    ``` sh
    gt feed --since 10m --plain | grep "done" || true
    # Check cross-rig dependents for each closure
    ```
  #/

  # fire-notifications : script
    - resolve-external-deps
    ``` sh
    # Notify mayor/ on convoy completions
    # Notify <rig>/witness on cross-rig dep resolution
    ```
  #/

  -- Mid-cycle heartbeat joins parallel branches
  # heartbeat-mid : script
    - orphan-process-cleanup
    - test-pollution-cleanup
    - dispatch-gated-molecules
    - fire-notifications
    ``` sh
    gt deacon heartbeat "mid-cycle checkpoint"
    ```
  #/

  # health-scan : script
    - heartbeat-mid
    ``` sh
    # For each active rig: check witness/refinery status
    # Skip DOCKED/PARKED rigs
    # Idle town protocol: skip health nudges if no active work
    ```
  #/

  # dolt-health : script
    - health-scan
    ``` sh
    gt health --json
    # Evaluate thresholds, dispatch compactor/backup dogs as needed
    ```
  #/

  # zombie-scan : script
    - dolt-health
    ``` sh
    gt deacon zombie-scan --dry-run
    # If zombies detected, file death warrants (NO kill authority)
    ```
  #/

  # plugin-run : script
    - zombie-scan
    ``` sh
    # Scan $GT_ROOT/plugins/ for plugin directories
    # Check gate conditions (cooldown, cron, condition, event)
    # Execute plugins whose gates are open
    ```
  #/

  # dog-pool-maintenance : script
    - health-scan
    ``` sh
    gt dog status
    # Ensure minimum idle dogs, retire stale dogs
    ```
  #/

  # dog-health-check : script
    - dog-pool-maintenance
    ``` sh
    gt dog list --json
    # Check each working dog's duration vs timeout
    # File warrants or force-clear stuck dogs
    ```
  #/

  # orphan-check : script
    - dog-health-check
    ``` sh
    bd list --status=in_progress --json | head -20
    # If orphans detected, dispatch mol-orphan-scan to dog
    ```
  #/

  # session-gc : script
    - orphan-check
    ``` sh
    gt doctor -v
    # If cleanup needed, dispatch mol-session-gc to dog
    ```
  #/

  # wisp-compact : script
    - session-gc
    ``` sh
    gt compact --dry-run --json
    gt compact --verbose
    ```
  #/

  # compact-report : script
    - wisp-compact
    ``` sh
    gt compact report
    # Weekly rollup on Mondays
    ```
  #/

  # costs-digest : script
    - compact-report
    ``` sh
    echo "DISABLED: cost tracking not available"
    ```
  #/

  # patrol-digest : script
    - costs-digest
    ``` sh
    gt patrol digest --yesterday --dry-run
    gt patrol digest --yesterday
    ```
  #/

  # log-maintenance : script
    - patrol-digest
    ``` sh
    gt daemon rotate-logs
    gt daemon status --json 2>/dev/null
    ```
  #/

  # patrol-cleanup : script
    - log-maintenance
    ``` sh
    gt mail inbox
    # Archive any remaining processed messages
    ```
  #/

  # context-check : script
    - patrol-cleanup
    ``` sh
    gt context --usage
    ```
  #/

  # loop-or-exit : script
    - context-check
    ``` sh
    # If context LOW: await-signal with backoff, then gt patrol report
    # If context HIGH: gt handoff and exit
    gt mol step await-signal --agent-bead hq-deacon \
      --backoff-base 60s --backoff-mult 2 --backoff-max 5m
    gt patrol report --summary "patrol cycle complete" \
      --steps "heartbeat:OK,inbox-check:OK,..."
    ```
  #/

  -- Wiring (DAG structure)
  heartbeat -> inbox-check
  inbox-check -> orphan-process-cleanup
  inbox-check -> test-pollution-cleanup
  inbox-check -> gate-evaluation -> dispatch-gated-molecules
  inbox-check -> check-convoy-completion -> resolve-external-deps -> fire-notifications
  orphan-process-cleanup -> heartbeat-mid
  test-pollution-cleanup -> heartbeat-mid
  dispatch-gated-molecules -> heartbeat-mid
  fire-notifications -> heartbeat-mid
  heartbeat-mid -> health-scan
  health-scan -> dolt-health -> zombie-scan -> plugin-run
  health-scan -> dog-pool-maintenance -> dog-health-check -> orphan-check -> session-gc
  session-gc -> wisp-compact -> compact-report -> costs-digest -> patrol-digest
  patrol-digest -> log-maintenance -> patrol-cleanup -> context-check -> loop-or-exit

##/
```

### Notes
- The 26-step DAG with parallel branches maps well to Cell's wiring syntax. The join at `heartbeat-mid` (4 dependencies) is naturally expressed.
- **Loop-back** (`loop-or-exit` returning to `heartbeat`) has no Cell primitive. The formula relies on the agent looping externally; Cell would need a `loop` construct or the agent runtime handles it.
- Many steps contain significant LLM judgment embedded in their description (e.g., health-scan thresholds, zombie assessment). These are modeled as script cells that call `gt` commands, with the judgment happening at the agent level rather than as explicit LLM cells. This is accurate to how the formula actually executes.
- `[vars]` only has `wisp_type` -- minimal input surface.
- Categorized EXTENDED due to the loop-back requirement and the implicit per-rig iteration in health-scan.

---

## 5. mol-dep-propagate

**Type**: Infrastructure / dog work (cross-rig dependency)
**Category**: EXTENDED
**Steps**: load-resolved-issue -> update-blocked-status -> notify-witnesses -> trigger-dispatch -> return-to-kennel

### Cell Translation

```cell
## dep-propagate {

  input param.resolved_issue : string required

  # load-resolved-issue : script
    ``` sh
    gt hook
    bd show {{param.resolved_issue}} --json
    # Extract 'blocks' field for cross-rig dependents
    ```
  #/

  # update-blocked-status : script
    - load-resolved-issue
    ``` sh
    # For each cross-rig dependent, verify unblock
    echo "{{load-resolved-issue}}" | jq -r '.blocks[]?' | while read dep; do
      bd blocked "$dep" || true
    done
    ```
  #/

  # notify-witnesses : script
    - update-blocked-status
    ``` sh
    # Group dependents by rig, send notification to each witness
    echo "{{load-resolved-issue}}" | jq -r '.blocks[]?' | while read dep; do
      RIG=$(echo "$dep" | cut -d- -f1)
      gt mail send "$RIG/witness" -s "Dependency resolved: {{param.resolved_issue}}" \
        -m "External dependency closed. Check bd ready for available work."
    done
    ```
  #/

  # trigger-dispatch : script
    - notify-witnesses
    ``` sh
    # For high-priority unblocked issues, notify mayor
    echo "{{load-resolved-issue}}" | jq -r '.blocks[]?' | while read dep; do
      PRIORITY=$(bd show "$dep" --json | jq -r '.priority')
      if [ "$PRIORITY" -le 1 ]; then
        gt mail send mayor/ -s "High-priority work unblocked: $dep" \
          -m "Issue $dep (P$PRIORITY) unblocked by {{param.resolved_issue}}"
      fi
    done
    ```
  #/

  # return-to-kennel : script
    - trigger-dispatch
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: dep-propagate
    Resolved: {{param.resolved_issue}}
    Status: COMPLETE

    Ready for next assignment."
    ```
  #/

  load-resolved-issue -> update-blocked-status -> notify-witnesses -> trigger-dispatch -> return-to-kennel

##/
```

### Notes
- Cross-rig iteration over dependents is the core challenge. Shell loops handle it, but Cell `each>` or `map` would be more idiomatic.
- Categorized EXTENDED because the per-dependent fanout with rig-based routing would benefit from `map # notify over dependents as dep`.

---

## 6. mol-digest-generate

**Type**: Infrastructure / dog work (periodic reporting)
**Category**: EXTENDED
**Steps**: determine-period -> collect-rig-data -> generate-digest -> send-digest -> return-to-kennel

### Cell Translation

```cell
## digest-generate {

  input param.period : string required  -- "daily" | "weekly" | "custom"

  # determine-period : script
    ``` sh
    PERIOD="{{param.period}}"
    case "$PERIOD" in
      daily)
        echo "{\"since\":\"$(date -d yesterday +%Y-%m-%dT00:00:00)\",\"until\":\"$(date +%Y-%m-%dT00:00:00)\"}"
        ;;
      weekly)
        echo "{\"since\":\"$(date -d 'last monday' +%Y-%m-%dT00:00:00)\",\"until\":\"$(date -d 'this monday' +%Y-%m-%dT00:00:00)\"}"
        ;;
    esac
    ```
  #/

  # collect-rig-data : script
    - determine-period
    ``` sh
    SINCE=$(echo '{{determine-period}}' | jq -r '.since')
    UNTIL=$(echo '{{determine-period}}' | jq -r '.until')
    for RIG in $(gt rigs --plain 2>/dev/null); do
      echo "=== $RIG ==="
      bd list --created-after="$SINCE" --created-before="$UNTIL" 2>/dev/null
      bd list --status=closed --updated-after="$SINCE" 2>/dev/null
      gt polecats "$RIG" 2>/dev/null
    done
    ```
  #/

  # generate-digest : llm
    - collect-rig-data
    - determine-period
    @ cost(max: 12000) @ quality(min: good)
    > You are generating a Gas Town digest for period {{param.period}}.
    >
    > Time range: {{determine-period}}
    >
    > Raw data from all rigs:
    > {{collect-rig-data}}
    >
    > Generate a formatted digest with:
    > - Summary statistics (filed, closed, net change, by type)
    > - Per-rig breakdown
    > - Highlights (big completions, incidents)
    > - Agent health metrics
    > - Trends
    format> json {
      "formatted_digest": "string",
      "date": "string"
    }
  #/

  # send-digest : script
    - generate-digest
    ``` sh
    gt mail send mayor/ -s "Gas Town Digest: {{generate-digest.date}}" \
      -m "{{generate-digest.formatted_digest}}"

    bd create --title="Digest: {{generate-digest.date}}" --type=digest \
      --description="{{generate-digest.formatted_digest}}" \
      --label=digest,{{param.period}}

    bd sync
    ```
  #/

  # return-to-kennel : script
    - send-digest
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: digest-generate
    Period: {{param.period}}
    Status: COMPLETE

    Digest sent to Mayor.
    Ready for next assignment."
    ```
  #/

  determine-period -> collect-rig-data -> generate-digest -> send-digest -> return-to-kennel

##/
```

### Notes
- Multi-rig data collection is the key EXTENDED feature. Cell `map # collect over rigs as rig` would parallelize the per-rig collection, but the dynamic rig list is only known at runtime.
- `generate-digest` is a natural LLM cell -- the original formula expects the agent to reason about raw data to produce a summary.

---

## 7. mol-orphan-scan

**Type**: Infrastructure / dog work (recovery)
**Category**: EXTENDED
**Steps**: determine-scope -> {scan-orphaned-issues, scan-orphaned-molecules, scan-orphaned-wisps} -> triage-orphans -> execute-recovery -> report-findings -> return-to-kennel

### Cell Translation

```cell
## orphan-scan {

  input param.scope : string required  -- "town" or rig name

  # determine-scope : script
    ``` sh
    SCOPE="{{param.scope}}"
    if [ "$SCOPE" = "town" ]; then
      gt rigs --plain
    else
      echo "$SCOPE"
    fi
    ```
  #/

  -- Three parallel scans
  # scan-orphaned-issues : script
    - determine-scope
    ``` sh
    for RIG in $(cat <<< '{{determine-scope}}'); do
      bd list --status=in_progress --json | while read -r line; do
        ASSIGNEE=$(echo "$line" | jq -r '.assignee // empty')
        [ -z "$ASSIGNEE" ] && echo "ORPHAN: $(echo "$line" | jq -r '.id') (no assignee)" && continue
        gt session status "$ASSIGNEE" --json 2>/dev/null | jq -r '.running' | grep -q true || \
          echo "ORPHAN: $(echo "$line" | jq -r '.id') (session dead: $ASSIGNEE)"
      done
    done
    ```
  #/

  # scan-orphaned-molecules : script
    - determine-scope
    ``` sh
    bd mol list --active --json 2>/dev/null | jq -c '.[]' | while read -r mol; do
      OWNER=$(echo "$mol" | jq -r '.agent // empty')
      [ -z "$OWNER" ] && continue
      gt session status "$OWNER" --json 2>/dev/null | jq -r '.running' | grep -q true || \
        echo "ORPHAN_MOL: $(echo "$mol" | jq -r '.id') (owner dead: $OWNER)"
    done
    ```
  #/

  # scan-orphaned-wisps : script
    - determine-scope
    ``` sh
    ls .beads-wisp/ 2>/dev/null
    # Check spawner session for each wisp > 1h old
    ```
  #/

  # triage-orphans : llm
    - scan-orphaned-issues
    - scan-orphaned-molecules
    - scan-orphaned-wisps
    @ cost(max: 6000) @ quality(min: adequate)
    > Classify each orphan found in the scans:
    >
    > Issues: {{scan-orphaned-issues}}
    > Molecules: {{scan-orphaned-molecules}}
    > Wisps: {{scan-orphaned-wisps}}
    >
    > For each orphan, assign an action:
    > RESET (return to open), REASSIGN, RECOVER, ESCALATE, or BURN
    format> json {
      "orphans": [{"id": "string", "type": "string", "action": "string", "reason": "string"}]
    }
  #/

  # execute-recovery : script
    - triage-orphans
    ``` sh
    echo '{{triage-orphans}}' | jq -c '.orphans[]' | while read -r item; do
      ACTION=$(echo "$item" | jq -r '.action')
      ID=$(echo "$item" | jq -r '.id')
      case "$ACTION" in
        RESET)    bd update "$ID" --status=open --assignee="" ;;
        ESCALATE) gt mail send mayor/ -s "Orphan escalation: $ID" -m "$(echo "$item" | jq -r '.reason')" ;;
        BURN)     rm -f ".beads-wisp/$ID" 2>/dev/null ;;
        *)        echo "Manual: $ACTION for $ID" ;;
      esac
    done
    ```
  #/

  # report-findings : script
    - execute-recovery
    ``` sh
    gt mail send deacon/ -s "Orphan scan complete" -m "{{triage-orphans}}"
    ```
  #/

  # return-to-kennel : script
    - report-findings
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: orphan-scan
    Scope: {{param.scope}}
    Status: COMPLETE

    Ready for next assignment."
    ```
  #/

  -- Wiring: parallel scans join at triage
  determine-scope -> scan-orphaned-issues
  determine-scope -> scan-orphaned-molecules
  determine-scope -> scan-orphaned-wisps
  scan-orphaned-issues -> triage-orphans
  scan-orphaned-molecules -> triage-orphans
  scan-orphaned-wisps -> triage-orphans
  triage-orphans -> execute-recovery -> report-findings -> return-to-kennel

##/
```

### Notes
- The parallel scan pattern (3 cells joining at triage) maps perfectly to Cell's DAG wiring.
- `triage-orphans` is a natural LLM cell -- classifying orphans by severity requires judgment.
- EXTENDED because the per-orphan iteration in `execute-recovery` ideally uses `each>`.

---

## 8. mol-refinery-patrol

**Type**: Patrol loop (14 steps, merge queue processor)
**Category**: EXTENDED
**Steps**: inbox-check -> queue-scan -> process-branch -> run-tests -> quality-review -> handle-failures -> merge-push -> loop-check -> generate-summary -> check-integration-branches -> context-check -> patrol-cleanup -> burn-or-loop

### Cell Translation

```cell
## refinery-patrol {

  input param.run_tests : string           -- "true" | "false"
  input param.test_command : string        -- e.g. "go test ./..."
  input param.setup_command : string
  input param.typecheck_command : string
  input param.lint_command : string
  input param.build_command : string
  input param.target_branch : string       -- default "main"
  input param.delete_merged_branches : string  -- "true" | "false"
  input param.judgment_enabled : string    -- "true" | "false"
  input param.review_depth : string        -- "quick" | "standard" | "deep"
  input param.integration_branch_refinery_enabled : string
  input param.integration_branch_auto_land : string

  # inbox-check : script
    ``` sh
    bd mol wisp gc --closed --force
    bd mol wisp gc --age 1h --force
    gt mail inbox
    # Process MERGE_READY, PATROL, HELP, HANDOFF messages
    # Track polecat name, MR bead ID, message ID for each MERGE_READY
    ```
  #/

  # queue-scan : script
    - inbox-check
    ``` sh
    git fetch --prune origin
    gt mq list $(gt rig name)
    # Verify each MR's branch still exists
    ```
  #/

  # process-branch : script
    - queue-scan
    ``` sh
    # Pick next branch, attempt mechanical rebase
    git checkout -b temp origin/<polecat-branch>
    git rebase origin/{{param.target_branch}}
    # On conflict: abort, create conflict-resolution task, skip
    ```
  #/

  # run-tests : script
    - process-branch
    ``` sh
    [ -n "{{param.setup_command}}" ] && {{param.setup_command}}
    [ -n "{{param.typecheck_command}}" ] && {{param.typecheck_command}}
    [ -n "{{param.lint_command}}" ] && {{param.lint_command}}
    [ -n "{{param.build_command}}" ] && {{param.build_command}}
    [ "{{param.run_tests}}" = "true" ] && {{param.test_command}}
    ```
  #/

  # quality-review : llm
    - run-tests
    @ cost(max: 15000) @ quality(min: good)
    > Review this merge diff for quality issues.
    > Judgment enabled: {{param.judgment_enabled}}
    > Review depth: {{param.review_depth}}
    >
    > {{run-tests}}
    >
    > Assess: correctness, security, clarity, style.
    > Score 0.0-1.0 and recommend approve or request_changes.
    format> json {
      "score": "number",
      "recommendation": "approve | request_changes",
      "issues": [{"category": "string", "description": "string"}]
    }
  #/

  # handle-failures : script
    - quality-review
    ``` sh
    # VERIFICATION GATE
    # If all passed: proceed
    # If branch caused failure: abort, reopen issue, send MERGE_FAILED, close MR
    # If pre-existing: file bug (check for duplicates first), proceed
    ```
  #/

  # merge-push : script
    - handle-failures
    ``` sh
    git checkout {{param.target_branch}}
    git merge --ff-only temp
    git push origin {{param.target_branch}}

    # Verify push: compare local vs remote SHA
    # Send MERGED to witness
    # Run gt mq post-merge
    # Archive MERGE_READY mail
    git branch -d temp
    ```
  #/

  # loop-check : script
    - merge-push
    ``` sh
    # More branches to process? Return to process-branch if yes.
    ```
  #/

  # generate-summary : script
    - loop-check
    ``` sh
    # Summarize: branches merged, MERGED mails sent, test results, conflicts
    ```
  #/

  # check-integration-branches : script
    - generate-summary
    ``` sh
    # If integration_branch_refinery_enabled AND auto_land:
    #   bd list --type=epic --status=open
    #   gt mq integration status <epic-id>
    #   gt mq integration land <epic-id> if ready_to_land
    ```
  #/

  # context-check : script
    - check-integration-branches
    ``` sh
    ps -o rss= -p $$
    # Assess session health: RSS, age, context consumed
    ```
  #/

  # patrol-cleanup : script
    - context-check
    ``` sh
    gt mail inbox
    # Archive stale messages, check for orphaned MR beads
    ```
  #/

  # burn-or-loop : script
    - patrol-cleanup
    ``` sh
    # If continuing: await-event, then gt patrol report + loop
    # If handing off: gt handoff
    gt mol step await-event --channel refinery --agent-bead gt-$(gt rig name)-refinery \
      --backoff-base 30s --backoff-mult 2 --backoff-max 5m --cleanup
    gt patrol report --summary "refinery patrol complete"
    ```
  #/

  -- Wiring
  inbox-check -> queue-scan -> process-branch -> run-tests -> quality-review
  quality-review -> handle-failures -> merge-push -> loop-check -> generate-summary
  generate-summary -> check-integration-branches -> context-check -> patrol-cleanup -> burn-or-loop

##/
```

### Notes
- The merge queue loop (`loop-check` returning to `process-branch`) has no Cell primitive. This is the same loop-back challenge as deacon-patrol.
- `quality-review` is a natural LLM cell when `judgment_enabled` is true; otherwise it would be skipped. Cell could express this with a conditional annotation or guard.
- The heavy use of configurable commands (`test_command`, `build_command`, etc.) maps to `input param.X` but the conditional execution (skip if empty) adds runtime logic.
- Categorized EXTENDED due to loop-back and conditional config execution.

---

## 9. mol-session-gc

**Type**: Infrastructure / dog work (garbage collection)
**Category**: DIRECT
**Steps**: determine-mode -> preview-cleanup -> execute-gc -> verify-cleanup -> report-gc -> return-to-kennel

### Cell Translation

```cell
## session-gc {

  input param.mode : string required  -- "conservative" | "aggressive"

  # determine-mode : script
    ``` sh
    echo "GC Mode: {{param.mode}}"
    ```
  #/

  # preview-cleanup : script
    - determine-mode
    ``` sh
    gt doctor -v
    ```
  #/

  # execute-gc : script
    - preview-cleanup
    ``` sh
    gt doctor --fix
    # If aggressive mode: also clean old branches, state files
    if [ "{{param.mode}}" = "aggressive" ]; then
      find "$GT_TOWN_ROOT" -path "*/.runtime/*.json" -mtime +7 -delete 2>/dev/null
    fi
    ```
  #/

  # verify-cleanup : script
    - execute-gc
    ``` sh
    gt doctor -v
    tmux list-sessions 2>/dev/null
    pgrep -f claude | wc -l
    ls .beads-wisp/ 2>/dev/null | wc -l
    ```
  #/

  # report-gc : script
    - verify-cleanup
    ``` sh
    gt mail send deacon/ -s "GC complete" -m "Mode: {{param.mode}}
    Verify output: {{verify-cleanup}}"
    ```
  #/

  # return-to-kennel : script
    - report-gc
    ``` sh
    gt mail send deacon/ -s "DOG_DONE $(hostname)" -m "Task: session-gc
    Mode: {{param.mode}}
    Status: COMPLETE

    Ready for next assignment."
    ```
  #/

  determine-mode -> preview-cleanup -> execute-gc -> verify-cleanup -> report-gc -> return-to-kennel

##/
```

### Notes
- Pure linear chain of script cells. No LLM judgment needed -- `gt doctor` handles the logic.
- Straightforward DIRECT translation.

---

## 10. mol-sync-workspace

**Type**: Infrastructure / sync (broadcast)
**Category**: DIRECT
**Steps**: assess-state -> handle-dirty-state -> cleanup-worktrees -> sync-git -> sync-beads -> run-doctor -> verify-build -> run-tests -> generate-report -> signal-ready

### Cell Translation

```cell
## sync-workspace {

  input param.setup_command : string
  input param.typecheck_command : string
  input param.lint_command : string
  input param.test_command : string
  input param.build_command : string

  # assess-state : script
    ``` sh
    gt prime
    bd prime
    git status --porcelain
    git stash list
    git branch --show-current
    bd sync --status
    ```
  #/

  # handle-dirty-state : script
    - assess-state
    ``` sh
    # Commit or stash uncommitted changes
    # Handle untracked files
    # Assess stash entries
    ```
  #/

  # cleanup-worktrees : script
    - handle-dirty-state
    ``` sh
    git worktree list
    git worktree prune
    ```
  #/

  # sync-git : script
    - cleanup-worktrees
    ``` sh
    git fetch origin
    git pull --rebase origin main
    # On conflict: preservation protocol (abort, file bead, reset)
    ```
  #/

  # sync-beads : script
    - sync-git
    ``` sh
    bd sync
    ```
  #/

  # run-doctor : script
    - sync-beads
    ``` sh
    bd doctor
    ```
  #/

  # verify-build : script
    - run-doctor
    ``` sh
    [ -n "{{param.setup_command}}" ] && {{param.setup_command}}
    [ -n "{{param.typecheck_command}}" ] && {{param.typecheck_command}}
    [ -n "{{param.lint_command}}" ] && {{param.lint_command}}
    [ -n "{{param.build_command}}" ] && {{param.build_command}}
    ```
  #/

  # run-tests : script
    - verify-build
    ``` sh
    [ -n "{{param.test_command}}" ] && {{param.test_command}} || echo "No test command configured"
    ```
  #/

  # generate-report : script
    - run-tests
    ``` sh
    BRANCH=$(git rev-parse --abbrev-ref HEAD)
    BEHIND=$(git rev-list HEAD..origin/main --count)
    AHEAD=$(git rev-list origin/main..HEAD --count)
    echo "SYNC COMPLETE
    Branch: $BRANCH
    Behind: $BEHIND
    Ahead: $AHEAD"
    ```
  #/

  # signal-ready : script
    - generate-report
    ``` sh
    # If responding to broadcast: mail coordinator
    # If autonomous: log report
    echo "Ready for work."
    ```
  #/

  assess-state -> handle-dirty-state -> cleanup-worktrees -> sync-git -> sync-beads
  sync-beads -> run-doctor -> verify-build -> run-tests -> generate-report -> signal-ready

##/
```

### Notes
- 10-step linear chain of pure script cells. No LLM needed.
- Config variables for quality checks mirror the refinery pattern.
- DIRECT -- straightforward sequential translation.

---

## 11. mol-witness-patrol

**Type**: Patrol loop (9 steps, worker monitor)
**Category**: EXTENDED
**Steps**: inbox-check -> process-cleanups -> check-refinery -> survey-workers -> check-timer-gates -> check-swarm-completion -> patrol-cleanup -> context-check -> loop-or-exit

### Cell Translation

```cell
## witness-patrol {

  # inbox-check : script
    ``` sh
    bd mol wisp gc --closed --force
    bd mol wisp gc --age 1h --force

    # Drain stale protocol messages
    gt mail drain --identity $(gt rig name)/witness --max-age 30m

    gt mail inbox
    # Process: POLECAT_STARTED, POLECAT_DONE, MERGED, HELP, HANDOFF, SWARM_START
    # Batch if > 10 messages
    ```
  #/

  # process-cleanups : script
    - inbox-check
    ``` sh
    bd list --label cleanup --status=open
    # For each cleanup wisp: diagnose dirty state, resolve or escalate
    ```
  #/

  # check-refinery : script
    - process-cleanups
    ``` sh
    gt session status $(gt rig name)/refinery
    gt refinery ready --all --json
    # If MRs waiting and refinery down: start it, emit event

    tmux has-session -t hq-deacon 2>/dev/null && echo "deacon alive" || \
      gt mail send mayor/ -s "ALERT: Deacon session hq-deacon is down" \
        -m "Detected during witness patrol."
    ```
  #/

  # survey-workers : script
    - check-refinery
    ``` sh
    # PRIMARY: Discover completions from agent bead metadata
    bd list --type=agent --json
    # For each polecat:
    #   - Check agent_state
    #   - ZOMBIE DETECTION: cross-reference tmux session
    #   - Progress assessment for running polecats
    #   - Stale spawn detection for spawning polecats
    #   - ORPHANED BEAD DETECTION from beads side
    ```
  #/

  # check-timer-gates : script
    - survey-workers
    ``` sh
    bd gate check --type=timer --escalate
    ```
  #/

  # check-swarm-completion : script
    - check-timer-gates
    ``` sh
    bd list --label swarm --status=open
    # If active swarm: count completed, notify mayor if all done
    ```
  #/

  # patrol-cleanup : script
    - check-swarm-completion
    ``` sh
    gt mail drain --identity $(gt rig name)/witness --max-age 30m
    gt mail inbox
    # Archive stale messages
    bd list --label cleanup --status=open
    ```
  #/

  # context-check : script
    - patrol-cleanup
    ``` sh
    gt context --usage
    ```
  #/

  # loop-or-exit : script
    - context-check
    ``` sh
    # If LOW: await-signal, then gt patrol report + loop
    # If HIGH: gt handoff
    gt mol step await-signal --agent-bead YOUR_AGENT_BEAD \
      --backoff-base 30s --backoff-mult 2 --backoff-max 5m
    gt patrol report --summary "witness patrol complete"
    ```
  #/

  inbox-check -> process-cleanups -> check-refinery -> survey-workers
  survey-workers -> check-timer-gates -> check-swarm-completion
  check-swarm-completion -> patrol-cleanup -> context-check -> loop-or-exit

##/
```

### Notes
- `survey-workers` is the most complex step -- it iterates over all polecat agent beads, performs zombie detection, progress checks, and orphan scanning. Ideally this would use `map # inspect over polecats as polecat` with Task-tool subagents.
- Loop-back is the same challenge as other patrol formulas.
- The swim lane rules and persistent polecat model are policy constraints that live in documentation, not language primitives.
- Categorized EXTENDED due to per-polecat iteration and loop-back.

---

## 12. towers-of-hanoi-7

**Type**: Durability test (pre-computed, 127 moves + setup + verify = 129 steps)
**Category**: DIRECT
**Steps**: setup -> move-1 -> move-2 -> ... -> move-127 -> verify (pure linear chain)

### Cell Translation

```cell
## towers-of-hanoi-7 {

  # setup : script
    ``` sh
    echo "All 7 disks stacked on peg A. Largest on bottom."
    ```
  #/

  # move-1 : script
    - setup
    ``` sh
    echo "Move disk 1: A -> C (1/127)"
    ```
  #/

  # move-2 : script
    - move-1
    ``` sh
    echo "Move disk 2: A -> B (2/127)"
    ```
  #/

  -- ... (moves 3-126 follow the same pattern) ...

  # move-127 : script
    - move-126
    ``` sh
    echo "Move disk 1: A -> C (127/127)"
    ```
  #/

  # verify : script
    - move-127
    ``` sh
    echo "All 7 disks now on peg C. Tower intact."
    ```
  #/

  setup -> move-1 -> move-2 -> ... -> move-127 -> verify

##/
```

### Notes
- Pure linear chain with 129 trivial script cells. Each move is a dependency on the previous.
- The Cell translation is mechanically identical to the TOML -- just different syntax.
- The purpose is crash-recovery durability testing, not computation. Cell handles this directly.
- The full 127-move wiring is omitted for brevity but follows `move-N -> move-(N+1)`.

---

## 13. towers-of-hanoi-9

**Type**: Durability test (511 moves + setup + verify = 513 steps)
**Category**: DIRECT

### Cell Translation

Same structure as towers-of-hanoi-7 but with 511 moves. Pure linear chain.

```cell
## towers-of-hanoi-9 {
  # setup : script
    ``` sh
    echo "All 9 disks stacked on peg A. Largest on bottom."
    ```
  #/

  # move-1 : script
    - setup
    ``` sh
    echo "Move disk 1: A -> C (1/511)"
    ```
  #/

  -- ... (moves 2-510) ...

  # move-511 : script
    - move-510
    ``` sh
    echo "Move disk 1: A -> C (511/511)"
    ```
  #/

  # verify : script
    - move-511
    ``` sh
    echo "All 9 disks now on peg C. Tower intact."
    ```
  #/

  setup -> move-1 -> ... -> move-511 -> verify
##/
```

### Notes
- Identical pattern to hanoi-7, just longer. Tests Cell's ability to handle 513-cell molecules.

---

## 14. towers-of-hanoi-10

**Type**: Durability test (1023 moves + setup + verify = 1025 steps)
**Category**: DIRECT

### Cell Translation

Same structure, 1023 moves. Pure linear chain.

```cell
## towers-of-hanoi-10 {
  # setup : script
    ``` sh
    echo "All 10 disks stacked on peg A. Largest on bottom."
    ```
  #/

  # move-1 : script
    - setup
    ``` sh
    echo "Move disk 1: A -> B (1/1023)"
    ```
  #/

  -- ... (moves 2-1022) ...

  # move-1023 : script
    - move-1022
    ``` sh
    echo "Move disk 1: A -> B (1023/1023)"
    ```
  #/

  # verify : script
    - move-1023
    ``` sh
    echo "All 10 disks now on peg C. Tower intact."
    ```
  #/

  setup -> move-1 -> ... -> move-1023 -> verify
##/
```

### Notes
- 1025-cell molecule. Tests scalability of the Cell parser and runtime.
- Note: hanoi-10 uses A->B for the first move (odd-disk Hanoi uses different start peg), while hanoi-7 and hanoi-9 use A->C. This is correct per the standard algorithm.

---

## Cross-Cutting Observations

### Patterns Seen Across All 14 Formulas

1. **Script-dominant**: 12 of 14 formulas are entirely or predominantly script cells calling `gt`/`bd` CLI tools. Only convoy-cleanup, digest-generate, and orphan-scan have natural LLM cells.

2. **Dog contract pattern**: 6 formulas (convoy-cleanup, convoy-feed, dep-propagate, digest-generate, orphan-scan, session-gc) share the same dog lifecycle: receive assignment -> do work -> report -> return-to-kennel. This is a candidate for a Cell `preset` or template.

3. **Patrol loop pattern**: 3 formulas (deacon-patrol, refinery-patrol, witness-patrol) share: inbox-check -> work -> cleanup -> context-check -> loop-or-exit. The loop-back has no Cell primitive.

4. **Durability test pattern**: 3 formulas (hanoi-7/9/10) are pure linear chains of trivial steps. Perfect for Cell, but the files are large (1025 steps for hanoi-10).

### Language Gaps Identified

| Gap | Affected Formulas | Severity |
|-----|-------------------|----------|
| **Loop-back** (patrol cycles) | deacon, refinery, witness | Medium -- runtime can handle externally |
| **`[squash]` metadata** | 6 dog formulas | Low -- annotation or molecule-level pragma |
| **Conditional step skip** (e.g., disabled cost-digest) | deacon-patrol | Low -- script cell can no-op |
| **`[vars]` with defaults** | All dog formulas | Low -- `input param.X` with default |
| **Dynamic iteration count** | convoy-feed, dep-propagate | Medium -- `map`/`each>` helps but list is runtime-determined |

### No True GAPs

All 14 formulas can be expressed in Cell, though some need the EXTENDED features (map/reduce for iteration, loop construct for patrols). The operational formulas are fundamentally script-driven -- the Cell language's script cell feature handles them directly. The few LLM judgment points (triage, quality review, digest generation) map naturally to LLM cells.
