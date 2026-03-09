# Batch 3: Molecule/Operational Formula Survey

**Date**: 2026-03-08
**Surveyor**: Morpheus
**Scope**: 14 molecule formulas (boot, shutdown, polecat, dog workflows)

---

## Summary

| # | Formula | Category | Notes |
|---|---------|----------|-------|
| 1 | mol-gastown-boot | EXTENDED | Parallel containers need `map` or nested molecules |
| 2 | mol-town-shutdown | DIRECT | Linear pipeline of script cells |
| 3 | mol-shutdown-dance | EXTENDED | Conditional gates, state machine branching |
| 4 | mol-polecat-work | DIRECT | Linear pipeline with script cells and vars |
| 5 | mol-polecat-lease | DIRECT | Linear pipeline, monitoring workflow |
| 6 | mol-polecat-review-pr | DIRECT | Linear pipeline of mixed script/llm cells |
| 7 | mol-polecat-conflict-resolve | DIRECT | Linear pipeline with script cells |
| 8 | mol-dog-reaper | EXTENDED | Iteration over databases, conditional dry-run |
| 9 | mol-dog-backup | EXTENDED | Iteration over databases |
| 10 | mol-dog-compactor | EXTENDED | SQL-heavy, ZFC-exempt (Go executor) |
| 11 | mol-dog-doctor | DIRECT | Linear probe-inspect-report |
| 12 | mol-dog-jsonl | EXTENDED | Iteration over databases, spike detection |
| 13 | mol-dog-phantom-db | DIRECT | Linear scan-quarantine-report |
| 14 | mol-dog-stale-db | EXTENDED | Conditional cleanup vs escalation branching |

**Totals**: 7 DIRECT, 7 EXTENDED, 0 GAP

---

## 1. mol-gastown-boot

**Type**: Lifecycle/bootstrap workflow
**Category**: EXTENDED

The boot molecule has parallel containers (`ensure-witnesses`, `ensure-refineries`) with children that execute concurrently. Cell language lacks a first-class "parallel container with children" construct -- the closest is wiring independent cells to the same dependency and merging at a downstream cell.

```cell
## gastown-boot {

  # ensure-daemon : script
    @ cost(max: 0)
    ``` sh
    gt daemon status || gt daemon start
    # Verify
    kill -0 $(cat $GT_TOWN_ROOT/daemon/daemon.pid) 2>/dev/null
    gt daemon status
    ```
  #/

  # ensure-deacon : script
    - ensure-daemon
    @ cost(max: 0)
    ``` sh
    gt deacon start
    # Verify: session exists, not stalled, heartbeat fresh
    tmux has-session -t hq-deacon 2>/dev/null
    output=$(gt peek deacon/)
    if echo "$output" | grep -q "> Try"; then
      gt nudge deacon/ "Start patrol."
      sleep 30
    fi
    ```
  #/

  -- Parallel witness cells (both depend on ensure-deacon, run concurrently)

  # ensure-gastown-witness : script
    - ensure-deacon
    @ cost(max: 0)
    ``` sh
    gt witness start gastown
    tmux has-session -t gastown-witness 2>/dev/null
    ```
  #/

  # ensure-beads-witness : script
    - ensure-deacon
    @ cost(max: 0)
    ``` sh
    gt witness start beads
    tmux has-session -t beads-witness 2>/dev/null
    ```
  #/

  -- Parallel refinery cells (both depend on ensure-deacon, run concurrently)

  # ensure-gastown-refinery : script
    - ensure-deacon
    @ cost(max: 0)
    ``` sh
    gt refinery start gastown
    tmux has-session -t gastown-refinery 2>/dev/null
    ```
  #/

  # ensure-beads-refinery : script
    - ensure-deacon
    @ cost(max: 0)
    ``` sh
    gt refinery start beads
    tmux has-session -t beads-refinery 2>/dev/null
    ```
  #/

  # verify-town-health : script
    - ensure-gastown-witness
    - ensure-beads-witness
    - ensure-gastown-refinery
    - ensure-beads-refinery
    @ cost(max: 0)
    ``` sh
    gt status
    ```
  #/

  -- Topology
  ensure-daemon -> ensure-deacon
  ensure-deacon -> ensure-gastown-witness
  ensure-deacon -> ensure-beads-witness
  ensure-deacon -> ensure-gastown-refinery
  ensure-deacon -> ensure-beads-refinery
  ensure-gastown-witness -> verify-town-health
  ensure-beads-witness -> verify-town-health
  ensure-gastown-refinery -> verify-town-health
  ensure-beads-refinery -> verify-town-health

  squash> on_complete

##/
```

**Notes**: The TOML formula uses `type = "parallel"` containers with `children` arrays -- a structural grouping concept. Cell flattens this: children become independent cells wired to the same upstream, achieving the same parallel execution via DAG topology. The "parallel container" is implicit rather than explicit. This is correct semantically but loses the grouping annotation. Classified EXTENDED because the `children` sub-step pattern has no direct Cell equivalent and would benefit from a `group` or `parallel` construct for clarity.

---

## 2. mol-town-shutdown

**Type**: Operational workflow
**Category**: DIRECT

A strictly linear pipeline of 8 steps with one input variable. Maps cleanly to sequential script cells.

```cell
## town-shutdown {

  input param.shutdown_reason : str

  # preflight-check : script
    @ cost(max: 0)
    ``` sh
    gt shutdown preflight
    ```
  #/

  # stop-sessions : script
    - preflight-check
    @ cost(max: 0)
    ``` sh
    gt stop --all --preserve-sandbox
    gt polecats --all --status
    ```
  #/

  # clear-inboxes : script
    - stop-sessions
    @ cost(max: 0)
    ``` sh
    for rig in $(gt rigs --names); do
      gt mail clear $rig/witness --archive
      gt mail clear $rig/refinery --archive
    done
    gt mail clear mayor --archive
    ```
  #/

  # stop-daemon : script
    - clear-inboxes
    @ cost(max: 0)
    ``` sh
    gt daemon stop
    ```
  #/

  # rotate-logs : script
    - stop-daemon
    @ cost(max: 0)
    ``` sh
    gt daemon rotate-logs
    gt doctor --fix
    ```
  #/

  # sync-state : script
    - rotate-logs
    @ cost(max: 0)
    ``` sh
    bd sync
    ```
  #/

  # handoff-mayor : script
    - sync-state
    @ cost(max: 0)
    ``` sh
    gt mail send mayor -s "HANDOFF: Town shutdown complete" -m "
    Town shutdown completed. State preserved.
    Polecat sandboxes: PRESERVED (will resume from hooks)
    Inboxes: ARCHIVED and cleared
    Daemon: STOPPED
    Shutdown reason: {{param.shutdown_reason}}
    "
    ```
  #/

  # restart-daemon : script
    - handoff-mayor
    @ cost(max: 0)
    ``` sh
    gt daemon start
    ```
  #/

  -- Topology: strict linear chain
  preflight-check -> stop-sessions -> clear-inboxes -> stop-daemon
  stop-daemon -> rotate-logs -> sync-state -> handoff-mayor -> restart-daemon

##/
```

**Notes**: Clean one-to-one mapping. The `[vars]` section maps to `input param.X`. Shell loops in `clear-inboxes` work naturally inside a script cell.

---

## 3. mol-shutdown-dance

**Type**: State machine specification
**Category**: EXTENDED

This is the most complex formula in the batch. It defines a multi-attempt interrogation state machine with conditional gates (`gate = { type = "conditional", condition = "..." }`), branching paths (pardon vs execute), and a fan-in at the epitaph step. Cell's `-> ? oracle ->` gates can express the conditional branching, but the retry loop (3 attempts with escalating timeouts) must be unrolled into explicit cells since Cell has no loop construct.

```cell
## shutdown-dance {

  input param.warrant_id : str required
  input param.target : str required
  input param.reason : str required
  input param.requester : str

  # warrant-received : script
    @ cost(max: 0)
    ``` sh
    tmux has-session -t {{param.target}} 2>/dev/null
    # Initialize state file, set attempt=1
    echo '{"warrant_id":"{{param.warrant_id}}","target":"{{param.target}}","state":"interrogating","attempt":1}' \
      > $GT_ROOT/deacon/dogs/active/$(uuidgen).json
    ```
  #/

  # interrogation-1 : script
    - warrant-received
    @ cost(max: 0)
    ``` sh
    tmux send-keys -t {{param.target}} \
      "[DOG] HEALTH CHECK: Session {{param.target}}, respond ALIVE within 60s or face termination. Warrant reason: {{param.reason}} Filed by: {{param.requester}} Attempt: 1/3" Enter
    sleep 60
    ```
  #/

  # evaluate-1 : script
    - interrogation-1
    @ cost(max: 0)
    ``` sh
    tmux capture-pane -t {{param.target}} -p | tail -50 | grep -q ALIVE
    ```
  #/

  # interrogation-2 : script
    - evaluate-1
    @ cost(max: 0)
    ``` sh
    tmux send-keys -t {{param.target}} \
      "[DOG] HEALTH CHECK: Session {{param.target}}, respond ALIVE within 120s or face termination. Warrant reason: {{param.reason}} Filed by: {{param.requester}} Attempt: 2/3" Enter
    sleep 120
    ```
  #/

  # evaluate-2 : script
    - interrogation-2
    @ cost(max: 0)
    ``` sh
    tmux capture-pane -t {{param.target}} -p | tail -50 | grep -q ALIVE
    ```
  #/

  # interrogation-3 : script
    - evaluate-2
    @ cost(max: 0)
    ``` sh
    tmux send-keys -t {{param.target}} \
      "[DOG] HEALTH CHECK: Session {{param.target}}, respond ALIVE within 240s or face termination. Warrant reason: {{param.reason}} Filed by: {{param.requester}} Attempt: 3/3" Enter
    sleep 240
    ```
  #/

  # evaluate-3 : script
    - interrogation-3
    @ cost(max: 0)
    ``` sh
    tmux capture-pane -t {{param.target}} -p | tail -50 | grep -q ALIVE
    ```
  #/

  # pardon : script
    @ cost(max: 0)
    ``` sh
    bd close {{param.warrant_id}} --reason "Session responded"
    gt mail send {{param.requester}}/ -s "PARDON: {{param.target}}" \
      -m "Death warrant cancelled. Session responded."
    ```
  #/

  # execute-warrant : script
    - evaluate-3
    @ cost(max: 0)
    ``` sh
    tmux kill-session -t {{param.target}}
    # Verify kill
    ! tmux has-session -t {{param.target}} 2>/dev/null
    ```
  #/

  # epitaph : script
    @ cost(max: 0)
    ``` sh
    bd close {{param.warrant_id}} --reason "Warrant processed"
    mv $GT_ROOT/deacon/dogs/active/*.json $GT_ROOT/deacon/dogs/completed/ 2>/dev/null
    ```
  #/

  -- Topology: unrolled state machine
  -- The conditional gates control whether we proceed to next interrogation
  -- or short-circuit to pardon.

  warrant-received -> interrogation-1
  interrogation-1 -> evaluate-1

  evaluate-1 -> ? alive-detected -> pardon
  evaluate-1 -> ? no-response -> interrogation-2

  interrogation-2 -> evaluate-2
  evaluate-2 -> ? alive-detected -> pardon
  evaluate-2 -> ? no-response -> interrogation-3

  interrogation-3 -> evaluate-3
  evaluate-3 -> ? alive-detected -> pardon
  evaluate-3 -> ? no-response-final -> execute-warrant

  pardon -> epitaph
  execute-warrant -> epitaph

  -- Gate oracles

  # alive-detected : oracle
    ``` oracle
    -- Script cell exit code 0 means ALIVE was found
    assert v.exit_code == 0;
    ```
  #/

  # no-response : oracle
    ``` oracle
    assert v.exit_code != 0;
    ```
  #/

  # no-response-final : oracle
    ``` oracle
    assert v.exit_code != 0;
    ```
  #/

  squash> on_complete

##/
```

**Notes**: Several translation challenges:
1. **Conditional gates**: The TOML uses `gate = { type = "conditional", condition = "no_response_1" }` for step-level gating. Cell uses `-> ? oracle ->` wire-level gating, which is equivalent but requires defining oracle cells for each condition.
2. **State machine semantics**: The original describes a Go goroutine state machine, not an LLM workflow. Cell can express the DAG structure, but the runtime semantics (timeout gates, tmux polling) live in the script cells.
3. **Fan-in with OR semantics**: The `pardon` step can be reached from any of the three `evaluate-*` cells. Cell wires typically imply AND (all deps must complete). OR-join semantics (any one path triggers) would need runtime extension.
4. **The `sleep` for timeout gates**: Embedded in script cells as a workaround; the TOML envisions proper timer gates in Go.

Classified EXTENDED because the conditional branching and OR-join at `pardon` push beyond Cell's current AND-join wire semantics.

---

## 4. mol-polecat-work

**Type**: Worker lifecycle workflow
**Category**: DIRECT

Linear 8-step pipeline with configurable gate commands (build, test, lint, typecheck). All steps are script cells with conditional execution of tool commands.

```cell
## polecat-work {

  input param.issue : str required
  input param.base_branch : str
  input param.setup_command : str
  input param.typecheck_command : str
  input param.test_command : str
  input param.lint_command : str
  input param.build_command : str

  # load-context : script
    @ cost(max: 0)
    ``` sh
    gt prime
    bd prime
    gt hook
    bd show {{param.issue}}
    gt mail inbox
    ```
  #/

  # branch-setup : script
    - load-context
    @ cost(max: 0)
    ``` sh
    git fetch origin
    branch=$(git branch --show-current)
    if [ -z "$branch" ] || [ "$branch" = "{{param.base_branch}}" ]; then
      git checkout -b polecat/$(whoami) origin/{{param.base_branch}}
    fi
    git rebase origin/{{param.base_branch}}
    [ -n "{{param.setup_command}}" ] && {{param.setup_command}}
    ```
  #/

  # implement : llm
    - branch-setup
    @ cost(max: 100000) @ quality(min: good)
    > You are a polecat worker implementing issue {{param.issue}}.
    > Work through the implementation following existing codebase conventions.
    > Make atomic, focused commits. Persist findings to the bead as you go:
    >   bd update {{param.issue}} --notes "Findings: ..."
    > Commit frequently: git add <files> && git commit -m "<type>: <desc> ({{param.issue}})"
  #/

  # self-review : llm
    - implement
    @ cost(max: 20000) @ quality(min: good)
    > Review your own changes before build check.
    > git diff origin/{{param.base_branch}}...HEAD
    > Check for: bugs, security, style, completeness, cruft.
    > Fix any issues found -- do not just note them.
  #/

  # build-check : script
    - self-review
    @ cost(max: 0)
    ``` sh
    [ -n "{{param.build_command}}" ] && {{param.build_command}}
    [ -n "{{param.setup_command}}" ] && {{param.setup_command}}
    [ -n "{{param.typecheck_command}}" ] && {{param.typecheck_command}}
    [ -n "{{param.lint_command}}" ] && {{param.lint_command}}
    true  # Don't fail if all commands are empty
    ```
  #/

  # commit-changes : script
    - build-check
    @ cost(max: 0)
    ``` sh
    git status
    git add -A && git commit -m "chore: final commit ({{param.issue}})" 2>/dev/null || true
    # Verify commits exist
    count=$(git log origin/{{param.base_branch}}..HEAD --oneline | wc -l)
    [ "$count" -gt 0 ] || { echo "ERROR: no commits"; exit 1; }
    ```
  #/

  # pre-verify : script
    - commit-changes
    @ cost(max: 0)
    ``` sh
    git fetch origin {{param.base_branch}}
    git rebase origin/{{param.base_branch}}
    [ -n "{{param.build_command}}" ] && {{param.build_command}}
    [ -n "{{param.typecheck_command}}" ] && {{param.typecheck_command}}
    [ -n "{{param.lint_command}}" ] && {{param.lint_command}}
    [ -n "{{param.test_command}}" ] && {{param.test_command}}
    true
    ```
  #/

  # submit-and-exit : script
    - pre-verify
    @ cost(max: 0)
    ``` sh
    git log origin/{{param.base_branch}}..HEAD --oneline
    gt done --pre-verified
    ```
  #/

  -- Topology: strict linear chain
  load-context -> branch-setup -> implement -> self-review
  self-review -> build-check -> commit-changes -> pre-verify -> submit-and-exit

##/
```

**Notes**: The `implement` and `self-review` steps are LLM-driven in practice (the polecat is a Claude session), but the formula describes them as human-readable instructions, not prompt templates. The Cell translation models them as `llm` cells with prompt guidance. The rest are pure script cells. All parameters map cleanly to `input param.X`.

---

## 5. mol-polecat-lease

**Type**: Monitoring/tracking workflow (Witness side)
**Category**: DIRECT

Linear 5-step lifecycle tracker. Each step is a monitoring/verification action.

```cell
## polecat-lease {

  input param.polecat : str required
  input param.issue : str required
  input param.rig : str required

  # boot : script
    @ cost(max: 0)
    ``` sh
    tmux capture-pane -t gt-{{param.rig}}-{{param.polecat}} -p | tail -20
    # Look for Claude prompt, gt prime output, issue reading
    # If idle >60s, nudge
    gt nudge {{param.rig}}/polecats/{{param.polecat}} "Begin work on {{param.issue}}."
    ```
  #/

  # working : script
    - boot
    @ cost(max: 0)
    ``` sh
    # Monitor for completion signal
    # Watch for: git commits, file changes, active tool usage
    # Flag if idle >15min, repeated errors, explicit stuck messages
    gt peek {{param.rig}}/polecats/{{param.polecat}}
    ```
  #/

  # verifying : script
    - working
    @ cost(max: 0)
    ``` sh
    cd polecats/{{param.polecat}}
    git status
    git stash list
    git log origin/main..HEAD
    git log origin/$(git branch --show-current)..HEAD
    bd show {{param.issue}}
    ```
  #/

  # merge-requested : script
    - verifying
    @ cost(max: 0)
    ``` sh
    gt mail send {{param.rig}}/refinery -s "MERGE_READY {{param.polecat}}" \
      -m "Branch: $(cd polecats/{{param.polecat}} && git branch --show-current)
    Issue: {{param.issue}}
    Polecat: {{param.polecat}}
    Verified: clean git state, issue closed"
    ```
  #/

  # done : script
    - merge-requested
    @ cost(max: 0)
    ``` sh
    gt session kill {{param.rig}}/polecats/{{param.polecat}}
    git worktree remove polecats/{{param.polecat}} --force
    git branch -D polecat/{{param.polecat}} 2>/dev/null || true
    ```
  #/

  -- Topology
  boot -> working -> verifying -> merge-requested -> done

##/
```

**Notes**: Clean mapping. The TOML step descriptions include rich monitoring behavior (nudge protocols, peek patterns) that are guidance for the Witness agent, not executable code. The Cell script cells contain the key commands; the behavioral guidance would live in the agent's prompt context, not the molecule definition.

---

## 6. mol-polecat-review-pr

**Type**: PR review workflow
**Category**: DIRECT

Linear 7-step pipeline mixing script and LLM cells for PR review.

```cell
## polecat-review-pr {

  input param.pr_url : str required
  input param.issue : str required
  input param.rig : str required

  # load-context : script
    @ cost(max: 0)
    ``` sh
    gt prime
    bd prime
    gt hook
    bd show {{param.issue}}
    gh pr view {{param.pr_url}} --json title,body,author,files,commits
    gh pr diff {{param.pr_url}}
    gh pr checks {{param.pr_url}}
    ```
  #/

  # review-code : llm
    - load-context
    @ cost(max: 30000) @ quality(min: good)
    > Review the PR diff systematically.
    > PR: {{param.pr_url}}
    > Context: {{load-context}}
    >
    > Check for: correctness, security, style, tests, docs, scope.
    > For each file changed, assess consistency with existing patterns.
    > Note blocking issues, suggestions, and questions.
  #/

  # check-tests : script
    - review-code
    @ cost(max: 0)
    ``` sh
    gh pr checks {{param.pr_url}}
    ```
  #/

  # make-decision : llm
    - review-code
    - check-tests
    @ cost(max: 10000) @ quality(min: good)
    > Based on code review and test status, decide:
    > - APPROVE: Clean code, tests pass, good scope
    > - REQUEST_CHANGES: Issues that need fixing
    > - NEEDS_DISCUSSION: Unclear requirements
    > - BLOCK: Security concern
    >
    > Review findings: {{review-code}}
    > CI status: {{check-tests}}
  #/

  # submit-review : script
    - make-decision
    @ cost(max: 0)
    ``` sh
    # Decision determines which gh pr review variant to call
    # APPROVE: gh pr review {{param.pr_url}} --approve --body "..."
    # REQUEST_CHANGES: gh pr review {{param.pr_url}} --request-changes --body "..."
    # COMMENT: gh pr review {{param.pr_url}} --comment --body "..."
    echo "Review submitted for {{param.pr_url}}"
    ```
  #/

  # file-followups : script
    - submit-review
    @ cost(max: 0)
    ``` sh
    bd update {{param.issue}} --notes "Review complete."
    bd sync
    ```
  #/

  # complete-and-exit : script
    - file-followups
    @ cost(max: 0)
    ``` sh
    bd sync
    gt done
    ```
  #/

  -- Topology
  load-context -> review-code -> check-tests
  review-code -> make-decision
  check-tests -> make-decision
  make-decision -> submit-review -> file-followups -> complete-and-exit

##/
```

**Notes**: The `submit-review` step has conditional behavior (different `gh pr review` flags based on decision). In the TOML formula, this is described as prose for the agent to interpret. In Cell, the script cell would need to read the `make-decision` output to choose the right command, or this could be modeled as an LLM cell that generates the command.

---

## 7. mol-polecat-conflict-resolve

**Type**: Merge conflict resolution workflow
**Category**: DIRECT

Linear 8-step pipeline for conflict resolution with merge-slot serialization.

```cell
## polecat-conflict-resolve {

  input param.task : str required
  input param.original_mr : str required
  input param.branch : str required
  input param.base_branch : str

  # load-task : script
    @ cost(max: 0)
    ``` sh
    gt prime
    bd prime
    gt hook
    bd show {{param.task}}
    ```
  #/

  # acquire-slot : script
    - load-task
    @ cost(max: 0)
    ``` sh
    bd merge-slot acquire --holder=$(whoami) --wait --json
    ```
  #/

  # checkout-branch : script
    - acquire-slot
    @ cost(max: 0)
    ``` sh
    git fetch origin
    git fetch origin {{param.branch}}:refs/remotes/origin/{{param.branch}}
    git checkout -b temp-resolve origin/{{param.branch}}
    git log --oneline -5
    ```
  #/

  # rebase-resolve : llm
    - checkout-branch
    @ cost(max: 50000) @ quality(min: good)
    > Rebase onto {{param.base_branch}} and resolve any conflicts.
    > git rebase origin/{{param.base_branch}}
    > For each conflict, read both versions, consider original intent.
    > After resolving: git add <file> && git rebase --continue
  #/

  # run-tests : script
    - rebase-resolve
    @ cost(max: 0)
    ``` sh
    go test ./...
    go build ./...
    ```
  #/

  # push-to-main : script
    - run-tests
    @ cost(max: 0)
    ``` sh
    git fetch origin
    git rebase origin/{{param.base_branch}}
    git push origin temp-resolve:{{param.base_branch}}
    git log origin/{{param.base_branch}} --oneline -3
    ```
  #/

  # close-beads : script
    - push-to-main
    @ cost(max: 0)
    ``` sh
    bd close {{param.original_mr}} --reason="merged after conflict resolution"
    bd sync
    ```
  #/

  # release-slot : script
    - close-beads
    @ cost(max: 0)
    ``` sh
    bd merge-slot release --holder=$(whoami) --json
    ```
  #/

  # cleanup-and-exit : script
    - release-slot
    @ cost(max: 0)
    ``` sh
    git checkout {{param.base_branch}}
    git branch -D temp-resolve
    bd close {{param.task}} --reason="Conflicts resolved and merged to {{param.base_branch}}"
    gt done
    ```
  #/

  -- Topology
  load-task -> acquire-slot -> checkout-branch -> rebase-resolve
  rebase-resolve -> run-tests -> push-to-main -> close-beads
  close-beads -> release-slot -> cleanup-and-exit

##/
```

**Notes**: Clean mapping. The merge-slot acquire/release pattern is just CLI calls in script cells. The `rebase-resolve` step is LLM-driven (agent uses judgment to resolve conflicts), while all others are deterministic scripts.

---

## 8. mol-dog-reaper

**Type**: Infrastructure cleanup (Dog)
**Category**: EXTENDED

Iterates over multiple databases with scan/reap/purge/auto-close operations. Uses Handlebars-style `{{#if dry_run}}` conditionals and `{{#each}}` iteration in the TOML formula, which Cell's `each>` and `map` constructs would need to handle.

```cell
## dog-reaper {

  input param.max_age : str
  input param.purge_age : str
  input param.stale_issue_age : str
  input param.mail_delete_age : str
  input param.alert_threshold : str
  input param.dry_run : str
  input param.databases : str
  input param.dolt_port : str

  # scan : script
    @ cost(max: 0)
    ``` sh
    dbs=$(gt reaper databases --json)
    for db in $(echo "$dbs" | jq -r '.[].name'); do
      gt reaper scan --db="$db" --port={{param.dolt_port}} \
        --max-age={{param.max_age}} --purge-age={{param.purge_age}} \
        --mail-age={{param.mail_delete_age}} --stale-age={{param.stale_issue_age}} \
        --json
    done
    ```
  #/

  # reap : script
    - scan
    @ cost(max: 0)
    ``` sh
    for db in $(gt reaper databases --json | jq -r '.[].name'); do
      gt reaper reap --db="$db" --port={{param.dolt_port}} \
        --max-age={{param.max_age}} \
        $([ "{{param.dry_run}}" = "true" ] && echo "--dry-run") \
        --json
    done
    ```
  #/

  # purge : script
    - reap
    @ cost(max: 0)
    ``` sh
    for db in $(gt reaper databases --json | jq -r '.[].name'); do
      gt reaper purge --db="$db" --port={{param.dolt_port}} \
        --purge-age={{param.purge_age}} --mail-age={{param.mail_delete_age}} \
        $([ "{{param.dry_run}}" = "true" ] && echo "--dry-run") \
        --json
    done
    ```
  #/

  # auto-close : script
    - purge
    @ cost(max: 0)
    ``` sh
    for db in $(gt reaper databases --json | jq -r '.[].name'); do
      gt reaper auto-close --db="$db" --port={{param.dolt_port}} \
        --stale-age={{param.stale_issue_age}} \
        $([ "{{param.dry_run}}" = "true" ] && echo "--dry-run") \
        --json
    done
    ```
  #/

  # report : llm
    - auto-close
    @ cost(max: 5000) @ quality(min: adequate)
    > Generate a Reaper Dog Report summarizing:
    > - Databases scanned, wisps reaped, wisps purged, mail purged
    > - Issues auto-closed, open wisps remaining, anomalies
    >
    > Scan results: {{scan}}
    > Reap results: {{reap}}
    > Purge results: {{purge}}
    > Auto-close results: {{auto-close}}
    >
    > If anomalies found, escalate via:
    > gt escalate "Reaper anomalies detected" -s MEDIUM -m "<details>"
  #/

  -- Topology
  scan -> reap -> purge -> auto-close -> report

  squash> on_complete

##/
```

**Notes**: The TOML formula uses Handlebars `{{#if dry_run}}` and `{{#each}}` which are template-level iteration/conditionals. Cell has no template conditionals -- the `dry_run` flag must be handled inside the shell script. Database iteration is done via shell `for` loops inside script cells. A native Cell `map # reap over databases as db` would be cleaner but requires the EXTENDED `map` construct. The report step uses an LLM to synthesize results, replacing the Handlebars template in the TOML.

---

## 9. mol-dog-backup

**Type**: Infrastructure backup (Dog)
**Category**: EXTENDED

Iterates over databases for backup sync, then rsyncs to offsite storage.

```cell
## dog-backup {

  input param.databases : str

  # sync : script
    @ cost(max: 0)
    ``` sh
    # Discover or use configured databases
    for db in $(echo "{{param.databases}}" | tr ',' ' '); do
      cd $GT_ROOT/.dolt-data/$db
      dolt backup sync ${db}-backup
    done
    ```
  #/

  # offsite : script
    - sync
    @ cost(max: 0)
    ``` sh
    if [ -d .dolt-backup/ ] && [ -d ~/Library/Mobile\ Documents/com~apple~CloudDocs/ ]; then
      rsync -a --delete .dolt-backup/ \
        ~/Library/Mobile\ Documents/com~apple~CloudDocs/gt-dolt-backup/
    else
      echo "Offsite sync skipped: prerequisites not met"
    fi
    ```
  #/

  # report : script
    - offsite
    @ cost(max: 0)
    ``` sh
    gt mail send deacon/ -s "DOG_DONE: backup" -m "Task: dolt-backup
    Status: COMPLETE"
    ```
  #/

  -- Topology
  sync -> offsite -> report

  squash> on_complete

##/
```

**Notes**: The TOML formula has many computed variables (`synced_count`, `total_count`, etc.) that are populated during execution -- these are runtime state, not input parameters. Cell doesn't model runtime-computed variables; the script cells would compute and use them inline. The database iteration (`{{#each db}}`) in the report template would ideally use `map`, but shell loops suffice. Classified EXTENDED for the iteration pattern.

---

## 10. mol-dog-compactor

**Type**: Infrastructure compaction (Dog, ZFC-exempt)
**Category**: EXTENDED

This formula is explicitly ZFC-exempt -- it defines the observable structure but the Go code in `compactor_dog.go` is the actual executor. The daemon "pours" the molecule, runs Go, and closes/fails each step for observability. Cell can express the DAG structure, but the SQL operations and transactional semantics require a Go executor.

```cell
## dog-compactor {

  input param.commit_threshold : str
  input param.databases : str
  input param.mode : str
  input param.keep_recent : str

  # inspect : script
    @ cost(max: 0)
    ``` sh
    -- ZFC-exempt: Go executor queries each database
    -- SELECT COUNT(*) FROM <database>.dolt_log;
    -- Records: database name, commit count, exceeds threshold
    echo "inspect: delegated to Go executor"
    ```
  #/

  # compact : script
    - inspect
    @ cost(max: 0)
    ``` sh
    -- ZFC-exempt: Go executor runs flatten or surgical compaction
    -- Flatten: DOLT_RESET('--soft', root) + DOLT_COMMIT
    -- Surgical: DOLT_REBASE('--interactive', 'compact-base')
    -- Post: CALL dolt_gc()
    echo "compact: delegated to Go executor"
    ```
  #/

  # verify : script
    - compact
    @ cost(max: 0)
    ``` sh
    -- ZFC-exempt: Go executor compares pre/post row counts
    echo "verify: delegated to Go executor"
    ```
  #/

  # report : script
    - verify
    @ cost(max: 0)
    ``` sh
    -- ZFC-exempt: Go executor generates report
    echo "report: delegated to Go executor"
    ```
  #/

  -- Topology
  inspect -> compact -> verify -> report

  squash> on_complete

##/
```

**Notes**: This is a fundamentally hybrid formula: Cell defines the structure, Go executes the logic. The Cell translation preserves the DAG shape for observability, but the script cell bodies are stubs -- the real work happens in Go. This pattern (formula-as-observable-skeleton with external executor) works in Cell but the formula is not self-contained. Classified EXTENDED because the Go delegation pattern needs runtime support beyond Cell's native execution model.

---

## 11. mol-dog-doctor

**Type**: Infrastructure health check (Dog)
**Category**: DIRECT

Linear 3-step probe-inspect-report pipeline. All steps are deterministic checks.

```cell
## dog-doctor {

  input param.port : str
  input param.latency_threshold : str

  # probe : script
    @ cost(max: 0)
    ``` sh
    start=$(date +%s%N)
    dolt sql -q "SELECT active_branch()" 2>/dev/null
    rc=$?
    end=$(date +%s%N)
    latency=$(( (end - start) / 1000000 ))
    if [ $rc -ne 0 ]; then
      echo "ERROR: Dolt server unreachable on port {{param.port}}"
      exit 1
    fi
    echo "latency_ms=$latency"
    ```
  #/

  # inspect : script
    - probe
    @ cost(max: 0)
    ``` sh
    # Connection count
    dolt sql -q "SELECT COUNT(*) FROM information_schema.PROCESSLIST;"
    # Disk usage
    du -sh $GT_ROOT/.dolt-data/
    # Daemon dir
    du -sh $GT_ROOT/daemon/
    # Orphan detection
    dolt sql -q "SHOW DATABASES;" | grep -E '^(testdb_|beads_t|beads_pt|doctest_|doctortest_|beads_vr)' | wc -l
    # Backup freshness
    ls -lt $GT_ROOT/.dolt-backup/ 2>/dev/null | head -5
    ```
  #/

  # report : script
    - inspect
    @ cost(max: 0)
    ``` sh
    gt mail send deacon/ -s "DOG_DONE: doctor" -m "Task: dolt-doctor
    Status: COMPLETE"
    ```
  #/

  -- Topology
  probe -> inspect -> report

  squash> on_complete

##/
```

**Notes**: Clean one-to-one mapping. All three steps are pure shell. The report step in the TOML uses Handlebars templates with computed variables, but in Cell, the mail body is assembled from the script's own output. The computed variables (`server_status`, `latency`, `conn_count`, etc.) are runtime artifacts, not Cell inputs.

---

## 12. mol-dog-jsonl

**Type**: Infrastructure backup/export (Dog)
**Category**: EXTENDED

Iterates over databases for JSONL export, has spike detection verification, and conditional halt on anomalies.

```cell
## dog-jsonl {

  input param.databases : str
  input param.scrub : str
  input param.spike_threshold : str
  input param.max_push_failures : str

  # export : script
    @ cost(max: 0)
    ``` sh
    for db in $(echo "{{param.databases}}" | tr ',' ' '); do
      dolt sql -r json -q "SELECT * FROM ${db}.issues" > $GIT_REPO/${db}/issues.jsonl
      # Export supplemental tables: comments, config, dependencies, labels, metadata
      for table in comments config dependencies labels metadata; do
        dolt sql -r json -q "SELECT * FROM ${db}.${table}" > $GIT_REPO/${db}/${table}.jsonl 2>/dev/null
      done
    done
    ```
  #/

  # verify : script
    - export
    @ cost(max: 0)
    ``` sh
    # Filter test pollution from exported JSONL
    # Spike detection: compare current counts against previous commit
    # Halt if delta > spike_threshold for any database
    echo "verify: spike detection delegated to Go executor"
    ```
  #/

  # push : script
    - verify
    @ cost(max: 0)
    ``` sh
    cd $GIT_REPO
    git add -A *.jsonl */
    if ! git diff --cached --quiet; then
      git commit -m "backup $(date +%Y-%m-%d-%H%M): export" \
        --author="Gas Town Daemon <daemon@gastown.local>"
      git push origin main
    else
      echo "No changes to commit"
    fi
    ```
  #/

  # report : script
    - push
    @ cost(max: 0)
    ``` sh
    gt mail send deacon/ -s "DOG_DONE: jsonl" -m "Task: jsonl-git-backup
    Status: COMPLETE"
    ```
  #/

  -- Topology
  export -> verify -> push -> report

  squash> on_complete

##/
```

**Notes**: Database iteration and spike detection logic are the EXTENDED features. The verify step's Go-side pollution filter and count comparison aren't expressible purely in shell. Like the compactor, this is a hybrid formula where Go does the heavy lifting and the Cell molecule is the observable skeleton.

---

## 13. mol-dog-phantom-db

**Type**: Infrastructure hygiene (Dog)
**Category**: DIRECT

Linear 3-step scan-quarantine-report pipeline. Pure filesystem operations.

```cell
## dog-phantom-db {

  input param.data_dir : str

  # scan : script
    @ cost(max: 0)
    ``` sh
    phantom_count=0
    phantom_names=""
    for dir in ~/gt/{{param.data_dir}}/*/; do
      name=$(basename "$dir")
      if [ -d "$dir/.dolt" ] && [ ! -f "$dir/.dolt/noms/manifest" ]; then
        phantom_count=$((phantom_count + 1))
        phantom_names="$phantom_names $name"
        echo "PHANTOM: $name"
      fi
    done
    echo "scan_count=$(ls -d ~/gt/{{param.data_dir}}/*/ 2>/dev/null | wc -l)"
    echo "phantom_count=$phantom_count"
    ```
  #/

  # quarantine : script
    - scan
    @ cost(max: 0)
    ``` sh
    quarantined=0
    for dir in ~/gt/{{param.data_dir}}/*/; do
      name=$(basename "$dir")
      if [ -d "$dir/.dolt" ] && [ ! -f "$dir/.dolt/noms/manifest" ]; then
        rm -rf "$dir"
        quarantined=$((quarantined + 1))
        echo "QUARANTINED: $name"
      fi
    done
    if [ $quarantined -gt 0 ]; then
      gt escalate -s HIGH "Dolt: quarantined $quarantined phantom database(s)"
    fi
    ```
  #/

  # report : script
    - quarantine
    @ cost(max: 0)
    ``` sh
    gt mail send deacon/ -s "DOG_DONE: phantom-db" -m "Task: phantom-db-scan
    Status: COMPLETE"
    ```
  #/

  -- Topology
  scan -> quarantine -> report

  squash> on_complete

##/
```

**Notes**: Clean mapping. All operations are filesystem checks and `rm -rf` -- pure shell. The escalation is conditional but handled inside the script cell. No iteration constructs beyond shell loops needed.

---

## 14. mol-dog-stale-db

**Type**: Infrastructure hygiene (Dog)
**Category**: EXTENDED

Has conditional branching: SQL cleanup (if orphan count is small) vs escalation (if too many orphans). The decision logic is a runtime conditional that Cell can only express via oracle gates or embedding the logic in shell.

```cell
## dog-stale-db {

  input param.port : str
  input param.max_orphans_for_sql : str
  input param.warn_threshold : str

  # scan : script
    @ cost(max: 0)
    ``` sh
    all_dbs=$(dolt sql -q "SHOW DATABASES;" 2>/dev/null | tail -n +2)
    orphan_count=0
    orphan_names=""
    for db in $all_dbs; do
      case "$db" in
        testdb_*|beads_t*|beads_pt*|beads_vr*|doctest_*|doctortest_*)
          orphan_count=$((orphan_count + 1))
          orphan_names="$orphan_names $db"
          ;;
      esac
    done
    echo "total_count=$(echo "$all_dbs" | wc -l)"
    echo "orphan_count=$orphan_count"
    echo "orphan_names=$orphan_names"
    ```
  #/

  # cleanup : script
    - scan
    @ cost(max: 0)
    ``` sh
    orphan_count={{scan.orphan_count}}
    if [ "$orphan_count" -eq 0 ]; then
      echo "No orphans found"
      exit 0
    fi
    if [ "$orphan_count" -le {{param.max_orphans_for_sql}} ]; then
      for db in {{scan.orphan_names}}; do
        dolt sql -q "DROP DATABASE IF EXISTS \`$db\`;"
      done
    else
      gt escalate -s HIGH "Dolt: $orphan_count orphan databases detected (too many for SQL cleanup)"
    fi
    ```
  #/

  # report : script
    - cleanup
    @ cost(max: 0)
    ``` sh
    gt mail send deacon/ -s "DOG_DONE: stale-db" -m "Task: stale-db-scan
    Status: COMPLETE"

    orphan_count={{scan.orphan_count}}
    if [ "$orphan_count" -ge {{param.warn_threshold}} ]; then
      gt mail send deacon/ -s "WARN: $orphan_count orphan databases detected" -m "Early warning"
    fi
    ```
  #/

  -- Topology
  scan -> cleanup -> report

  squash> on_complete

##/
```

**Notes**: The conditional logic (SQL cleanup vs escalation based on orphan count) is embedded in shell. Cell's `{{scan.orphan_count}}` field projection is used to pass data between cells, but this requires the scan cell to output structured data that the runtime can parse -- a stretch beyond basic Cell semantics. Alternatively, all logic could live in a single script cell. The iteration over orphan databases for `DROP DATABASE` would benefit from `map`. Classified EXTENDED for the cross-cell data passing and conditional branching.

---

## Cross-Cutting Observations

### Patterns That Map Well

1. **Linear pipelines**: Most formulas (town-shutdown, polecat-work, polecat-lease, conflict-resolve, doctor, phantom-db) are strict linear chains. Cell's `->` wiring handles these trivially.
2. **Script-heavy molecules**: Operational formulas are dominated by shell commands. Cell's ``` sh ``` blocks are a natural fit.
3. **Input variables**: TOML `[vars]` maps directly to Cell `input param.X`.
4. **Squash configuration**: TOML `[squash]` maps to Cell `squash>`.

### Patterns That Need EXTENDED Features

1. **Database iteration**: Reaper, backup, JSONL, stale-db all iterate over a dynamic list of databases. Shell `for` loops work but Cell's `map # name over collection as var` would be cleaner.
2. **Conditional dry-run**: Reaper and others use `{{#if dry_run}}` template conditionals. Cell has no template-level conditionals; these must be shell conditionals.
3. **Computed variables**: Many TOML formulas define output variables (computed during execution) like `synced_count`, `orphan_count`. Cell doesn't model runtime-computed state -- these live inside script cells.
4. **Parallel containers with children**: The boot formula's `type = "parallel"` with `children` has no direct Cell equivalent. DAG topology achieves the same result but loses the grouping annotation.

### Patterns That Work Differently

1. **Conditional gates**: TOML's `gate = { type = "conditional", condition = "..." }` becomes `-> ? oracle ->` in Cell. Semantically equivalent but syntactically different.
2. **OR-join fan-in**: The shutdown-dance's `pardon` step (reachable from any evaluate step) requires OR-join semantics. Cell wires are AND-join by default.
3. **ZFC-exempt formulas**: Compactor and JSONL are executed by Go code, not agents. Cell molecules are the observable skeleton; the executor is external.
4. **Agent behavioral guidance**: TOML step descriptions include rich prose (nudge protocols, monitoring patterns, failure modes) that guide agent behavior. Cell cells contain executable content; the behavioral guidance belongs in the agent's system prompt, not the molecule.

### No GAPs Found

Every TOML formula can be expressed in Cell, either directly or with the EXTENDED features (map, conditional gates, field projection). The key insight is that operational molecules are predominantly script cells with shell-level control flow, and Cell's ``` sh ``` blocks handle this naturally. The remaining complexity (iteration, conditionals, computed state) is absorbed by the shell rather than requiring new Cell syntax.
