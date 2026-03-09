# Batch 1: Workflow Formula Survey -- TOML to Cell Translation

**Date**: 2026-03-08
**Surveyor**: Morpheus (Cell language design)
**Source**: `/home/nixos/wasteland/.beads/formulas/`

---

## Summary Table

| # | Formula | Type | Category | Notes |
|---|---------|------|----------|-------|
| 1 | shiny | workflow | DIRECT | Linear chain with vars; maps cleanly |
| 2 | shiny-secure | workflow (composed) | GAP | AOP `aspects` / `advice.around` has no Cell equivalent |
| 3 | shiny-enterprise | workflow (composed) | GAP | `expand` (step explosion via expansion template) has no Cell equivalent |
| 4 | security-audit | aspect | GAP | Aspect-oriented pointcuts/advice are outside Cell's paradigm |
| 5 | gastown-release | workflow | EXTENDED | Long chain; needs `input` for vars, embedded shell via script cells |
| 6 | beads-release | workflow | EXTENDED | Similar to gastown-release; fan-in at verify step needs parallel wires |
| 7 | rule-of-five | expansion | GAP | Expansion templates with `{target}` interpolation have no Cell equivalent |
| 8 | towers-of-hanoi | workflow | DIRECT | Pure linear chain of trivial steps |

**Totals**: 2 DIRECT, 2 EXTENDED, 4 GAP

---

## 1. shiny.formula.toml

**Type**: workflow (5 steps, linear chain)
**Category**: DIRECT

The canonical engineering workflow. Linear dependency chain with one input variable (`feature`) and an optional `assignee`. Each step has a title, description, and acceptance criteria. This maps directly to a Cell molecule with LLM cells wired in sequence.

### Cell Translation

```cell
## shiny {

  input param.feature : string required
  input param.assignee : string

  # design : llm
    @ cost(max: 8000) @ quality(min: good)
    > Think carefully about architecture before writing code.
    > Consider: How does this fit into the existing system?
    > What are the edge cases? What could go wrong?
    > Is there a simpler approach?
    >
    > Feature: {{param.feature}}
    >
    > Produce a design doc covering approach, trade-offs,
    > and files to change.
    accept> Design doc committed covering approach, trade-offs, and files to change
  #/

  # implement : llm
    - design
    @ cost(max: 12000) @ quality(min: good)
    > Write the code for {{param.feature}}.
    > Follow the design: {{design}}
    > Keep it simple. Don't gold-plate.
    accept> All files from the design doc are modified/created and committed
  #/

  # review : llm
    - implement
    @ cost(max: 8000) @ quality(min: good)
    > Review the implementation: {{implement}}
    > Check for: Does it match the design? Are there obvious bugs?
    > Is it readable and maintainable? Are there security concerns?
    accept> Self-review complete, no obvious bugs, code is readable and secure
  #/

  # test : llm
    - review
    @ cost(max: 10000) @ quality(min: good)
    > Write and run tests for {{param.feature}}.
    > Unit tests for new code, integration tests if needed,
    > run the full test suite, fix any regressions.
    > Review feedback: {{review}}
    accept> All tests pass, no regressions, test coverage for new code
  #/

  # submit : llm
    - test
    @ cost(max: 4000) @ quality(min: adequate)
    > Submit for merge. Final check: git status, git diff.
    > Commit with clear message. Follow your role's git workflow.
    > Test results: {{test}}
    accept> Clean git status, clear commit message, code pushed to feature branch
  #/

  design -> implement -> review -> test -> submit

##/
```

### Notes

- **Easy**: Linear dependency chain maps 1:1 to Cell wires. `{{feature}}` becomes `{{param.feature}}`.
- **Easy**: Acceptance criteria map to `accept>` blocks.
- **Minor gap**: TOML `vars.assignee` (optional, non-interpolated) has no obvious Cell placement; stored as `input param.assignee` but nothing references it. In TOML it is metadata for the runtime, not the prompt.

---

## 2. shiny-secure.formula.toml

**Type**: workflow (composed: extends shiny, applies security-audit aspect)
**Category**: GAP

This formula extends `shiny` and composes in the `security-audit` aspect. The aspect uses AOP-style `advice.around` to inject pre-scan and post-scan steps around `implement` and `submit`.

### Cell Translation (best effort)

Cell has no aspect-oriented programming model. There is no `advice`, `pointcut`, or `around` mechanism. The closest approximation is to manually inline the aspect's steps:

```cell
-- GAP: Cell cannot express aspect composition.
-- The following is a MANUAL INLINING of what the TOML runtime produces
-- after applying the security-audit aspect to shiny.

## shiny-secure {

  input param.feature : string required
  input param.assignee : string

  # design : llm
    @ cost(max: 8000) @ quality(min: good)
    > Think carefully about architecture for {{param.feature}}.
    > Cover approach, trade-offs, and files to change.
    accept> Design doc committed
  #/

  # implement-security-prescan : llm
    - design
    @ cost(max: 4000) @ quality(min: good)
    > Pre-implementation security check.
    > Review for secrets/credentials in scope.
    > Check dependencies for known vulnerabilities.
    > Design: {{design}}
    accept> No pre-existing security issues in scope
  #/

  # implement : llm
    - implement-security-prescan
    @ cost(max: 12000) @ quality(min: good)
    > Write the code for {{param.feature}}.
    > Follow the design: {{design}}
    > Security prescan clear: {{implement-security-prescan}}
    accept> All files modified/created and committed
  #/

  # implement-security-postscan : llm
    - implement
    @ cost(max: 6000) @ quality(min: excellent)
    > Post-implementation security scan.
    > Scan new code for vulnerabilities (SAST).
    > Check for hardcoded secrets.
    > Review for OWASP Top 10 issues.
    > Implementation: {{implement}}
    accept> No vulnerabilities found in new code
  #/

  # review : llm
    - implement-security-postscan
    @ cost(max: 8000) @ quality(min: good)
    > Review the implementation: {{implement}}
    > Security postscan: {{implement-security-postscan}}
    accept> Self-review complete, no obvious bugs
  #/

  # test : llm
    - review
    @ cost(max: 10000) @ quality(min: good)
    > Write and run tests for {{param.feature}}.
    > Review: {{review}}
    accept> All tests pass, no regressions
  #/

  # submit-security-prescan : llm
    - test
    @ cost(max: 4000) @ quality(min: good)
    > Pre-submission security check.
    > Final vulnerability scan before merge.
    > Test results: {{test}}
    accept> Final security scan clean
  #/

  # submit : llm
    - submit-security-prescan
    @ cost(max: 4000) @ quality(min: adequate)
    > Submit for merge. Final check: git status, git diff.
    > Security prescan: {{submit-security-prescan}}
    accept> Clean git status, code pushed
  #/

  # submit-security-postscan : llm
    - submit
    @ cost(max: 4000) @ quality(min: good)
    > Post-submission security verification.
    > Confirm no new vulnerabilities introduced.
    > Submission: {{submit}}
    accept> No new vulnerabilities post-submission
  #/

  design -> implement-security-prescan -> implement
  implement -> implement-security-postscan -> review
  review -> test -> submit-security-prescan -> submit
  submit -> submit-security-postscan

##/
```

### Notes

- **GAP**: Cell has no `extends`, no `compose.aspects`, no `advice.around`. The AOP model (pointcuts + before/after advice) is a fundamentally different composition paradigm from Cell's DAG-wiring model.
- **Workaround**: Manual inlining works but loses the composability. A recipe could partially automate the insertion pattern, but cannot express "for every step matching this glob, wrap with before/after."
- **Possible extension**: A `recipe wrap-step(target, before-spec, after-spec)` could handle one step at a time, but the glob-based pointcut matching would need a `map` over matched steps.

---

## 3. shiny-enterprise.formula.toml

**Type**: workflow (composed: extends shiny, expands implement with rule-of-five)
**Category**: GAP

This formula extends `shiny` and replaces the `implement` step with the `rule-of-five` expansion (draft + 4 refinement passes).

### Cell Translation (best effort)

Cell has no `extends` or `compose.expand` mechanism. The expansion template in `rule-of-five` uses `{target}` interpolation to generate step IDs dynamically. The closest Cell equivalent is manual inlining:

```cell
-- GAP: Cell cannot express expansion composition.
-- The following manually inlines rule-of-five into the implement step.

## shiny-enterprise {

  input param.feature : string required
  input param.assignee : string

  # design : llm
    @ cost(max: 8000) @ quality(min: good)
    > Think carefully about architecture for {{param.feature}}.
    > Cover approach, trade-offs, and files to change.
    accept> Design doc committed
  #/

  -- Rule of Five expansion replaces the single "implement" step

  # implement-draft : llm
    - design
    @ cost(max: 12000) @ quality(min: adequate)
    > Initial attempt at implementing {{param.feature}}.
    > Design: {{design}}
    > Don't aim for perfection. Get the shape right. Breadth over depth.
    accept> Draft implementation complete
  #/

  # implement-refine-1 : llm
    - implement-draft
    @ cost(max: 10000) @ quality(min: good)
    > First refinement pass. Focus: CORRECTNESS.
    > Fix errors, bugs, mistakes. Is the logic sound?
    > Draft: {{implement-draft}}
    accept> Correctness pass complete
  #/

  # implement-refine-2 : llm
    - implement-refine-1
    @ cost(max: 8000) @ quality(min: good)
    > Second refinement pass. Focus: CLARITY.
    > Can someone else understand this? Simplify. Remove jargon.
    > After correctness: {{implement-refine-1}}
    accept> Clarity pass complete
  #/

  # implement-refine-3 : llm
    - implement-refine-2
    @ cost(max: 8000) @ quality(min: good)
    > Third refinement pass. Focus: EDGE CASES.
    > What could go wrong? What's missing? Handle the unusual.
    > After clarity: {{implement-refine-2}}
    accept> Edge cases pass complete
  #/

  # implement-refine-4 : llm
    - implement-refine-3
    @ cost(max: 8000) @ quality(min: excellent)
    > Final polish. Focus: EXCELLENCE.
    > This is the last pass. Make it shine.
    > Is this something you'd be proud to ship?
    > After edge cases: {{implement-refine-3}}
    accept> Final polish complete
  #/

  # review : llm
    - implement-refine-4
    @ cost(max: 8000) @ quality(min: good)
    > Review the implementation: {{implement-refine-4}}
    > Does it match the design? Obvious bugs? Readable?
    accept> Self-review complete
  #/

  # test : llm
    - review
    @ cost(max: 10000) @ quality(min: good)
    > Write and run tests for {{param.feature}}.
    > Review: {{review}}
    accept> All tests pass
  #/

  # submit : llm
    - test
    @ cost(max: 4000) @ quality(min: adequate)
    > Submit for merge. Final check.
    > Test results: {{test}}
    accept> Clean git status, code pushed
  #/

  design -> implement-draft -> implement-refine-1 -> implement-refine-2
  implement-refine-2 -> implement-refine-3 -> implement-refine-4
  implement-refine-4 -> review -> test -> submit

##/
```

### Notes

- **GAP**: Cell has no `extends` or `expand` composition. The expansion template's `{target}` interpolation to generate dynamic step IDs is a meta-programming pattern Cell does not support.
- **Workaround**: Manual inlining works but loses reusability. A `recipe` could define the 5-step pattern, but Cell recipes operate on existing graphs via `!split`/`!refine` -- they don't replace a cell with a subgraph the way TOML expansion does.
- **Possible extension**: A `recipe expand-to-five(target)` using `!split target => [draft, refine-1, refine-2, refine-3, refine-4]` plus `!refine` for each could work, but this stretches the recipe semantics beyond their current definition.

---

## 4. security-audit.formula.toml

**Type**: aspect
**Category**: GAP

This is an aspect-oriented cross-cutting concern. It defines pointcuts (glob patterns matching step IDs) and advice (before/after steps injected around matched steps). The `{step.id}` interpolation generates dynamic IDs based on what step the advice wraps.

### Cell Translation

There is no Cell translation. The aspect type is fundamentally incompatible with Cell's paradigm.

```
-- GAP: No translation possible.
--
-- Cell has no concept of:
--   1. Aspects (cross-cutting concerns applied externally)
--   2. Pointcuts (glob-based step matching)
--   3. Advice (before/after/around injection)
--   4. {step.id} interpolation (dynamic ID generation from match context)
--
-- The closest Cell mechanism is a recipe, but recipes require
-- explicit target cell names -- they cannot glob-match.
--
-- To express this in Cell, aspects would need to be a new
-- first-class construct, or the recipe system would need
-- pattern-matching over cell names.
```

### Notes

- **GAP**: This is the clearest gap in the survey. AOP is a composition paradigm; Cell is a wiring paradigm. They solve different problems.
- **Design consideration**: If Cell wants to subsume TOML formulas, it needs either (a) an AOP extension, or (b) a recipe variant that can match cells by pattern and inject before/after cells.

---

## 5. gastown-release.formula.toml

**Type**: workflow (14 steps, linear chain with detailed shell scripts)
**Category**: EXTENDED

A long linear release workflow. Each step has extensive description with embedded shell scripts, error handling instructions for different agent types (crew vs polecat), and `{{version}}` interpolation throughout.

### Cell Translation

```cell
## gastown-release {

  input param.version : string required

  # preflight-workspaces : script
    @ cost(max: 0)
    > Before releasing, ensure no gastown workspaces have uncommitted work.
    ``` sh
    for dir in $GT_ROOT/gastown/crew/* $GT_ROOT/gastown/mayor; do
      if [ -d "$dir/.git" ] || [ -d "$dir" ]; then
        cd "$dir" 2>/dev/null || continue
        if ! git diff-index --quiet HEAD -- 2>/dev/null; then
          echo "UNCOMMITTED: $dir"
          git status --short
        fi
        stash_count=$(git stash list 2>/dev/null | wc -l | tr -d ' ')
        if [ "$stash_count" -gt 0 ]; then
          echo "STASHES: $dir ($stash_count)"
        fi
        current_branch=$(git branch --show-current 2>/dev/null)
        if [ -n "$current_branch" ] && [ "$current_branch" != "main" ]; then
          echo "BRANCH: $dir on $current_branch"
        fi
      fi
    done
    ```
    accept> All workspaces clean on main with no stashes
  #/

  # preflight-git : script
    - preflight-workspaces
    @ cost(max: 0)
    ``` sh
    git status --porcelain
    ```
    accept> Working tree is clean
  #/

  # preflight-pull : script
    - preflight-git
    @ cost(max: 0)
    ``` sh
    git pull --rebase
    ```
    accept> Up to date with origin, no conflicts
  #/

  # review-changes : script
    - preflight-pull
    @ cost(max: 0)
    ``` sh
    git log $(git describe --tags --abbrev=0)..HEAD --oneline
    ```
    accept> Changes reviewed and categorized
  #/

  # update-changelog : llm
    - review-changes
    @ cost(max: 8000) @ quality(min: good)
    > Write the CHANGELOG.md [Unreleased] section for version {{param.version}}.
    > Changes since last release: {{review-changes}}
    >
    > Format: Keep a Changelog (https://keepachangelog.com)
    > Sections: Added, Changed, Fixed, Deprecated, Removed.
    > Group related commits. The bump script stamps the date.
    accept> CHANGELOG.md updated with all changes for {{param.version}}
  #/

  # update-info-go : llm
    - update-changelog
    - review-changes
    @ cost(max: 6000) @ quality(min: good)
    > Add entry to versionChanges in internal/cmd/info.go for {{param.version}}.
    > This powers `gt info --whats-new`.
    > Changes: {{review-changes}}
    > Changelog: {{update-changelog}}
    >
    > Focus on agent-relevant and workflow-impacting changes.
    > Prefix with NEW:, CHANGED:, FIX:, or DEPRECATED:.
    accept> info.go versionChanges updated for {{param.version}}
  #/

  # run-bump-script : script
    - update-info-go
    @ cost(max: 0)
    ``` sh
    ./scripts/bump-version.sh {{param.version}}
    ```
    accept> All component versions bumped to {{param.version}}
  #/

  # verify-versions : script
    - run-bump-script
    @ cost(max: 0)
    ``` sh
    echo "version.go: $(grep 'Version = ' internal/cmd/version.go)"
    echo "package.json: $(grep '"version"' npm-package/package.json | head -1)"
    ```
    ``` oracle
    assert contains(v, "{{param.version}}");
    ```
    accept> All versions match {{param.version}}
  #/

  # commit-release : script
    - verify-versions
    @ cost(max: 0)
    ``` sh
    git add -A
    git commit -m "chore: Bump version to {{param.version}}"
    ```
    accept> Release commit created with all version files
  #/

  # create-tag : script
    - commit-release
    @ cost(max: 0)
    ``` sh
    git tag -a v{{param.version}} -m "Release v{{param.version}}"
    ```
    accept> Tag v{{param.version}} created
  #/

  # push-release : script
    - create-tag
    @ cost(max: 0)
    ``` sh
    git push origin main
    git push origin v{{param.version}}
    ```
    accept> Commit and tag pushed to origin
  #/

  # local-install : script
    - push-release
    @ cost(max: 0)
    ``` sh
    go build -o $(go env GOPATH)/bin/gt ./cmd/gt
    gt version
    ```
    accept> Local gt binary rebuilt, shows {{param.version}}
  #/

  # restart-daemons : script
    - local-install
    @ cost(max: 0)
    ``` sh
    gt daemon stop && gt daemon start
    gt daemon status
    ```
    accept> Daemon restarted with new version
  #/

  # release-complete : llm
    - restart-daemons
    @ cost(max: 2000) @ quality(min: adequate)
    > Release v{{param.version}} is complete.
    > Summarize: workspaces verified, versions updated, tag pushed,
    > local binary rebuilt, daemons restarted.
    accept> Release summary produced
  #/

  preflight-workspaces -> preflight-git -> preflight-pull -> review-changes
  review-changes -> update-changelog -> update-info-go -> run-bump-script
  run-bump-script -> verify-versions -> commit-release -> create-tag
  create-tag -> push-release -> local-install -> restart-daemons
  restart-daemons -> release-complete

##/
```

### Notes

- **Easy**: Linear chain maps directly. `{{version}}` becomes `{{param.version}}`.
- **Extended**: Most steps are `script` type (shell commands), not LLM cells. The TOML formula embeds shell in markdown descriptions; Cell formalizes this with ```` ``` sh ```` blocks.
- **Hard**: TOML descriptions contain conditional error-handling instructions ("For crew: do X / For polecat: do Y"). Cell has no mechanism for agent-type-conditional behavior. This context is lost in translation or would need to be embedded in prompts.
- **Hard**: The TOML `pour = true` flag (not present here but present in beads-release) has no Cell equivalent.

---

## 6. beads-release.formula.toml

**Type**: workflow (17 steps, mostly linear with one fan-in)
**Category**: EXTENDED

Similar to gastown-release but longer, with CI verification steps and a fan-in where `local-install` depends on both `verify-npm` and `verify-pypi` (which run in parallel after `verify-github-release`).

### Cell Translation

```cell
## beads-release {

  input param.version : string required

  # preflight-git : script
    @ cost(max: 0)
    ``` sh
    git status --porcelain
    ```
    accept> Working tree is clean
  #/

  # preflight-pull : script
    - preflight-git
    @ cost(max: 0)
    ``` sh
    git pull --rebase
    ```
    accept> Up to date with origin
  #/

  # review-changes : script
    - preflight-pull
    @ cost(max: 0)
    ``` sh
    git log $(git describe --tags --abbrev=0)..HEAD --oneline
    ```
    accept> Changes reviewed and categorized
  #/

  # update-changelog : llm
    - review-changes
    @ cost(max: 8000) @ quality(min: good)
    > Write the CHANGELOG.md [Unreleased] section for {{param.version}}.
    > Changes: {{review-changes}}
    > Format: Keep a Changelog. Sections: Added, Changed, Fixed, Documentation.
    accept> CHANGELOG.md updated
  #/

  # update-info-go : llm
    - update-changelog
    @ cost(max: 6000) @ quality(min: good)
    > Add entry to versionChanges in cmd/bd/info.go for {{param.version}}.
    > Focus on workflow-impacting changes agents need to know.
    > Changelog: {{update-changelog}}
    accept> info.go updated
  #/

  # run-bump-script : script
    - update-info-go
    @ cost(max: 0)
    ``` sh
    ./scripts/bump-version.sh {{param.version}}
    ```
    accept> All component versions bumped
  #/

  # verify-versions : script
    - run-bump-script
    @ cost(max: 0)
    ``` sh
    grep 'Version = ' cmd/bd/version.go
    jq -r '.version' .claude-plugin/plugin.json
    jq -r '.version' npm-package/package.json
    grep 'version = ' integrations/beads-mcp/pyproject.toml
    ```
    ``` oracle
    -- All lines should contain the target version
    for line in lines(v) {
      assert contains(line, "{{param.version}}") or line == "";
    }
    ```
    accept> All versions match {{param.version}}
  #/

  # commit-release : script
    - verify-versions
    @ cost(max: 0)
    ``` sh
    git add -A
    git commit -m "chore: Bump version to {{param.version}}"
    ```
    accept> Release commit created
  #/

  # create-tag : script
    - commit-release
    @ cost(max: 0)
    ``` sh
    git tag -a v{{param.version}} -m "Release v{{param.version}}"
    ```
    accept> Tag created
  #/

  # push-main : script
    - create-tag
    @ cost(max: 0)
    ``` sh
    git push origin main
    ```
    accept> Main branch pushed
  #/

  # push-tag : script
    - push-main
    @ cost(max: 0)
    ``` sh
    git push origin v{{param.version}}
    ```
    accept> Tag pushed, CI triggered
  #/

  # wait-ci : llm
    - push-tag
    @ cost(max: 4000) @ quality(min: adequate)
    > Monitor GitHub Actions for release completion.
    > https://github.com/steveyegge/beads/actions
    > Expected time: 5-10 minutes.
    > Watch for: build artifacts, test pass, npm publish, PyPI publish.
    accept> CI completed successfully
  #/

  # verify-github-release : script
    - wait-ci
    @ cost(max: 0)
    ``` sh
    gh release view v{{param.version}} --json tagName,assets 2>/dev/null
    ```
    accept> GitHub release exists with binaries and checksums
  #/

  -- Fan-out: npm and pypi verification run in parallel

  # verify-npm : script
    - verify-github-release
    @ cost(max: 0)
    ``` sh
    npm show @beads/bd version
    ```
    ``` oracle
    assert contains(v, "{{param.version}}");
    ```
    accept> npm package shows {{param.version}}
  #/

  # verify-pypi : script
    - verify-github-release
    @ cost(max: 0)
    ``` sh
    pip index versions beads-mcp 2>/dev/null | head -3
    ```
    accept> PyPI package shows {{param.version}}
  #/

  -- Fan-in: local-install depends on both verifications

  # local-install : script
    - verify-npm
    - verify-pypi
    @ cost(max: 0)
    ``` sh
    curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash
    bd --version
    ```
    ``` oracle
    assert contains(v, "{{param.version}}");
    ```
    accept> Local bd shows {{param.version}}
  #/

  # restart-daemons : script
    - local-install
    @ cost(max: 0)
    ``` sh
    bd daemons killall
    bd daemons list
    ```
    accept> Daemons restarted
  #/

  # release-complete : llm
    - restart-daemons
    @ cost(max: 2000) @ quality(min: adequate)
    > Release v{{param.version}} is complete.
    > All versions updated, tag pushed, CI passed,
    > npm and PyPI published, local install updated, daemons restarted.
    accept> Release summary produced
  #/

  -- Topology
  preflight-git -> preflight-pull -> review-changes
  review-changes -> update-changelog -> update-info-go -> run-bump-script
  run-bump-script -> verify-versions -> commit-release -> create-tag
  create-tag -> push-main -> push-tag -> wait-ci -> verify-github-release
  verify-github-release -> verify-npm
  verify-github-release -> verify-pypi
  verify-npm -> local-install
  verify-pypi -> local-install
  local-install -> restart-daemons -> release-complete

##/
```

### Notes

- **Easy**: Most of the chain is linear. The fan-out/fan-in at verify-npm/verify-pypi maps cleanly to Cell wires.
- **Extended**: The `pour = true` flag in TOML (auto-execute on creation) has no Cell equivalent. This is runtime behavior, not language structure.
- **Hard**: The `wait-ci` step is inherently time-based (polling). Cell has no sleep/retry mechanism; this would need to be an LLM cell that the agent interprets as "wait and check."

---

## 7. rule-of-five.formula.toml

**Type**: expansion
**Category**: GAP

An expansion template that replaces a target step with a 5-step chain (draft + 4 refinement passes). Uses `{target}` interpolation in step IDs and descriptions, and `{target.title}` / `{target.description}` for content. This is a meta-programming construct: it generates steps dynamically based on the target it is applied to.

### Cell Translation

Cell has no expansion template mechanism. The closest approximation is a recipe:

```cell
-- GAP: Cell cannot express expansion templates.
--
-- The TOML expansion template uses:
--   {target}             -> generates IDs like "implement.draft"
--   {target.title}       -> interpolates the original step's title
--   {target.description} -> interpolates the original step's description
--
-- A Cell recipe can approximate this for a KNOWN target:

recipe rule-of-five(target) {
  !split target => [draft, refine-1, refine-2, refine-3, refine-4]

  !refine draft {
    > Initial attempt. Don't aim for perfection.
    > Get the shape right. Breadth over depth.
  }

  !refine refine-1 {
    > First refinement pass. Focus: CORRECTNESS.
    > Fix errors, bugs, mistakes. Is the logic sound?
  }

  !refine refine-2 {
    > Second refinement pass. Focus: CLARITY.
    > Can someone else understand this? Simplify. Remove jargon.
  }

  !refine refine-3 {
    > Third refinement pass. Focus: EDGE CASES.
    > What could go wrong? What's missing? Handle the unusual.
  }

  !refine refine-4 {
    > Final polish. Focus: EXCELLENCE.
    > This is the last pass. Make it shine.
  }

  !wire draft -> refine-1
  !wire refine-1 -> refine-2
  !wire refine-2 -> refine-3
  !wire refine-3 -> refine-4
}
```

### Notes

- **GAP**: The recipe approximation is close but not equivalent. Key differences:
  1. TOML expansion generates step IDs from `{target}` (e.g., `implement.draft`). Cell `!split` names are static.
  2. TOML expansion interpolates `{target.description}` into the new steps. Cell `!refine` cannot reference the original cell's prompt.
  3. TOML expansion preserves the original step's `needs` on the first sub-step and wires the last sub-step to the original's dependents. Cell `!split` semantics are underspecified for this.
- **Design consideration**: If `!split` preserved upstream/downstream wires automatically (first sub-cell inherits inbound, last inherits outbound), this would be closer to expressible.

---

## 8. towers-of-hanoi.formula.toml

**Type**: workflow (9 steps, linear chain)
**Category**: DIRECT

A durability proof: 3-disk Towers of Hanoi with pre-computed moves. Each step is a trivial acknowledgment (the agent just closes it). Pure linear chain, no LLM reasoning needed.

### Cell Translation

```cell
## towers-of-hanoi {

  input param.source_peg : string   -- default "A"
  input param.target_peg : string   -- default "C"
  input param.auxiliary_peg : string -- default "B"

  # setup : script
    @ cost(max: 0)
    > Verify initial state: All 3 disks stacked on peg {{param.source_peg}}.
    > Largest on bottom.
    ``` sh
    echo "Initial state verified: 3 disks on peg {{param.source_peg}}"
    ```
    accept> All 3 disks stacked on peg A, largest on bottom
  #/

  # move-1 : script
    - setup
    @ cost(max: 0)
    ``` sh
    echo "Move disk 1: {{param.source_peg}} -> {{param.target_peg}}"
    ```
    accept> Disk 1 moved from A to C
  #/

  # move-2 : script
    - move-1
    @ cost(max: 0)
    ``` sh
    echo "Move disk 2: {{param.source_peg}} -> {{param.auxiliary_peg}}"
    ```
    accept> Disk 2 moved from A to B
  #/

  # move-3 : script
    - move-2
    @ cost(max: 0)
    ``` sh
    echo "Move disk 1: {{param.target_peg}} -> {{param.auxiliary_peg}}"
    ```
    accept> Disk 1 moved from C to B
  #/

  # move-4 : script
    - move-3
    @ cost(max: 0)
    ``` sh
    echo "Move disk 3: {{param.source_peg}} -> {{param.target_peg}}"
    ```
    accept> Disk 3 moved from A to C
  #/

  # move-5 : script
    - move-4
    @ cost(max: 0)
    ``` sh
    echo "Move disk 1: {{param.auxiliary_peg}} -> {{param.source_peg}}"
    ```
    accept> Disk 1 moved from B to A
  #/

  # move-6 : script
    - move-5
    @ cost(max: 0)
    ``` sh
    echo "Move disk 2: {{param.auxiliary_peg}} -> {{param.target_peg}}"
    ```
    accept> Disk 2 moved from B to C
  #/

  # move-7 : script
    - move-6
    @ cost(max: 0)
    ``` sh
    echo "Move disk 1: {{param.source_peg}} -> {{param.target_peg}}"
    ```
    accept> Disk 1 moved from A to C
  #/

  # verify : script
    - move-7
    @ cost(max: 0)
    ``` sh
    echo "All 3 disks now on peg {{param.target_peg}}. Tower intact."
    ```
    accept> All 3 disks on peg C, tower intact, all moves legal
  #/

  setup -> move-1 -> move-2 -> move-3 -> move-4
  move-4 -> move-5 -> move-6 -> move-7 -> verify

##/
```

### Notes

- **Easy**: Pure linear chain of trivial steps. Maps 1:1.
- **Minor**: TOML `vars` have `default` values. Cell `input param.X` syntax does not show a way to specify defaults. This is a minor gap -- the user prompt spec mentions `input param.X : type required` but not `default`.
- **Minor**: The TOML formula's extended description (agent execution protocol) is metadata/documentation, not executable content. Cell has no place for formula-level documentation beyond `--` comments.
- **Observation**: This formula demonstrates that Cell handles deterministic/mechanical workflows just as well as LLM-heavy ones. The `script` cell type with `@ cost(max: 0)` is a natural fit.

---

## Gap Analysis Summary

### What Cell handles well (DIRECT)

1. **Linear dependency chains** -- `needs: [prev]` maps to `- ref` + wires
2. **Variable interpolation** -- `{{version}}` maps to `{{param.version}}`
3. **Acceptance criteria** -- TOML `acceptance` maps to `accept>`
4. **Mixed cell types** -- Script and LLM steps both have natural Cell representations
5. **Fan-out / fan-in** -- Multiple deps and dependents wire cleanly

### What Cell can handle with extensions (EXTENDED)

1. **Long shell scripts in descriptions** -- Cell's ```` ``` sh ```` blocks formalize this
2. **Pour flag** -- Runtime behavior, not language concern (could be a molecule annotation)
3. **Default variable values** -- Minor syntax addition needed for `input param.X : type = default`

### What Cell cannot express (GAP)

1. **Aspect-oriented composition** (`aspects`, `advice.around`, `pointcuts`) -- No Cell equivalent. This is a fundamentally different composition model.
2. **Expansion templates** (`compose.expand`, `{target}` interpolation) -- No Cell equivalent. Step-explosion from a template requires meta-programming.
3. **Formula inheritance** (`extends: [base]`) -- No Cell equivalent. Molecules cannot extend other molecules.
4. **Dynamic ID generation** (`{step.id}-security-prescan`) -- Cell IDs are static.
5. **Agent-type-conditional behavior** (crew vs polecat instructions) -- No Cell equivalent, though this could live in prompts.

### Recommendations for Cell language evolution

1. **Recipe enhancement**: Allow `!split` to preserve upstream/downstream wiring, making expansion-like patterns expressible.
2. **Pattern-matching in recipes**: Allow `recipe name(target matching "glob")` to enable aspect-like before/after injection.
3. **Molecule inheritance**: Consider `## name extends base {` for formula composition.
4. **Default values**: Add `input param.X : type = default_value` syntax.
