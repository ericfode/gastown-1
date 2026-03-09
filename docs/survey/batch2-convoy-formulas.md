# Batch 2: Convoy Formula Survey -- TOML to Cell Translation

**Date**: 2026-03-08
**Surveyor**: Morpheus
**Scope**: 8 convoy/workflow/molecule formulas

---

## Summary

| # | Formula | Type | Category | Notes |
|---|---------|------|----------|-------|
| 1 | code-review | convoy | EXTENDED | Needs `map`, `preset`, `prompt@` |
| 2 | design | convoy | EXTENDED | Needs `map`, `prompt@` |
| 3 | mol-algebraic-survey | molecule (sequential) | DIRECT | Linear step chain maps cleanly |
| 4 | mol-survey-dispatch | molecule (orchestrator) | EXTENDED | Needs `map` over subsystem list + meta-dispatch |
| 5 | mol-idea-to-plan | workflow | EXTENDED | Human gates need `accept>` + oracle-gated wires |
| 6 | mol-prd-review | convoy | EXTENDED | Needs `map`, `prompt@` |
| 7 | mol-plan-review | convoy | EXTENDED | Needs `map`, `prompt@` |
| 8 | mol-polecat-code-review | molecule (sequential) | DIRECT | Linear step chain, similar to #3 |

**Verdict**: 2 DIRECT, 6 EXTENDED. 0 GAP. All formulas are expressible in Cell.

---

## 1. code-review.formula.toml

**Type**: Convoy (parallel legs + synthesis)
**Category**: EXTENDED

Requires: `map` (parallel leg dispatch), `preset` (gate/full/refactor/security-focused), `prompt@` (shared base prompt), `input` declarations.

### Cell Translation

```cell
## code-review {

  -- ================================================================
  -- INPUTS
  -- ================================================================
  input param.pr : number required_unless(param.files, param.branch)
  input param.files : str required_unless(param.pr, param.branch)
  input param.branch : str required_unless(param.pr, param.files)

  -- ================================================================
  -- PRESETS (named leg selections)
  -- ================================================================
  preset gate {
    aspects = ["wiring", "security", "smells", "test-quality"]
  }
  preset full {
    aspects = ["correctness", "performance", "security", "elegance",
               "resilience", "style", "smells", "wiring",
               "commit-discipline", "test-quality"]
  }
  preset security-focused {
    aspects = ["security", "resilience", "correctness", "wiring"]
  }
  preset refactor {
    aspects = ["elegance", "smells", "style", "commit-discipline"]
  }

  -- ================================================================
  -- REUSABLE PROMPT FRAGMENT
  -- ================================================================
  prompt@ review-base
    > You are a specialized code reviewer participating in a convoy review.
    >
    > ## Context
    > - **Review target**: {{param.pr}}{{param.files}}{{param.branch}}
    > - **Your focus**: {{aspect.focus}}
    > - **Leg ID**: {{aspect.id}}
    >
    > ## Output Requirements
    > Structure findings as:
    > - Critical Issues (P0 - must fix before merge)
    > - Major Issues (P1 - should fix before merge)
    > - Minor Issues (P2 - nice to fix)
    > - Observations (non-blocking)
    >
    > Use specific file:line references. Be actionable. Prioritize impact.

  -- ================================================================
  -- ASPECT DEFINITIONS (data for map)
  -- ================================================================
  -- Each aspect has: id, focus, description
  -- The map construct iterates over aspects from the active preset.

  -- ================================================================
  -- PARALLEL LEGS via map
  -- ================================================================
  map # leg : llm over aspects as aspect
    @ cost(max: 8000) @ quality(min: good) @ model(claude-sonnet)
    system>
      {{@review-base}}
    user>
      > {{aspect.description}}
    format> json {
      summary: str,
      critical_issues: [{ file: str, line: number, description: str,
                          impact: str, suggestion: str }],
      major_issues: [{ file: str, line: number, description: str,
                       impact: str, suggestion: str }],
      minor_issues: [{ file: str, line: number, description: str }],
      observations: [str]
    }
  #/

  -- ================================================================
  -- SYNTHESIS (depends on all legs)
  -- ================================================================
  # synthesis : llm
    - leg.*
    @ cost(max: 12000) @ quality(min: excellent)
    system>
      > You are a senior reviewer synthesizing findings from
      > {{leg.* | length}} parallel review legs.
    user>
      > Combine all leg findings into a unified, prioritized review.
      >
      each> finding in {{leg.*}}
        > ## {{finding.aspect.id}} Findings
        > {{finding}}
      >
      > ## Instructions
      > 1. Executive Summary: overall assessment, merge recommendation
      > 2. Critical Issues: P0 items from all legs, deduplicated
      > 3. Major Issues: P1 items, grouped by theme
      > 4. Minor Issues: P2 items, briefly listed
      > 5. Wiring Gaps: dependencies added but not used
      > 6. Commit Quality: notes on commit discipline
      > 7. Test Quality: assessment of test meaningfulness
      > 8. Positive Observations: what's done well
      > 9. Recommendations: actionable next steps
      >
      > Deduplicate issues found by multiple legs. Note which legs found them.
    format> json {
      verdict: "approve" | "request-changes" | "comment",
      executive_summary: str,
      critical_issues: [{ description: str, found_by: [str] }],
      major_issues: [{ description: str, theme: str, found_by: [str] }],
      minor_issues: [str],
      positive_observations: [str],
      recommendations: [str]
    }
    ``` oracle
    json_parse(v);
    assert v.verdict in ["approve", "request-changes", "comment"];
    if v.verdict == "request-changes" { assert len(v.critical_issues) > 0; }
    ```
  #/

  -- TOPOLOGY
  leg.* -> synthesis

##/
```

### Notes

- **Easy**: The convoy pattern (parallel legs + synthesis) maps perfectly to Cell's `map` + synthesis cell with `- leg.*`.
- **Hard**: The 10 individual leg descriptions are long prose blocks. In TOML they are `[[legs]]` array entries; in Cell they become aspect data fed to the `map`. The exact mechanism for defining the aspect data inline (id, focus, description) is the main extension point -- Cell needs a way to define the collection that `map` iterates over, either inline or via a preset.
- **Presets** are a direct match to Cell's `preset` construct.
- **Go template syntax** (`{{.pr_number}}`, `{{range .files}}`) translates to Cell's `{{param.X}}` and `each>` constructs.

---

## 2. design.formula.toml

**Type**: Convoy (parallel legs + synthesis)
**Category**: EXTENDED

Requires: `map`, `prompt@`, `input` declarations. Structurally identical pattern to code-review.

### Cell Translation

```cell
## design {

  input param.problem : str required
  input param.context : str
  input param.scope : str = "medium"

  prompt@ design-base
    > You are a specialized design analyst participating in a convoy
    > design exploration.
    >
    > ## Context
    > - **Problem**: {{param.problem}}
    > - **Your dimension**: {{aspect.focus}}
    > - **Scope**: {{param.scope}}
    >
    > {{param.context}}
    >
    > ## Output Structure
    > - Summary (1-2 paragraphs)
    > - Key Considerations (bulleted)
    > - Options Explored (with pros/cons/effort for each)
    > - Recommendation
    > - Constraints Identified
    > - Open Questions (needing human input)
    > - Integration Points (cross-dimension connections)

  -- ================================================================
  -- PARALLEL LEGS: api, data, ux, scale, security, integration
  -- ================================================================
  map # leg : llm over aspects as aspect
    @ cost(max: 10000) @ quality(min: good)
    system>
      {{@design-base}}
    user>
      > {{aspect.description}}
    format> json {
      summary: str,
      key_considerations: [str],
      options: [{ name: str, description: str,
                  pros: [str], cons: [str], effort: "low"|"medium"|"high" }],
      recommendation: str,
      constraints: [str],
      open_questions: [str],
      integration_points: [str]
    }
  #/

  -- ================================================================
  -- SYNTHESIS
  -- ================================================================
  # synthesis : llm
    - leg.*
    @ cost(max: 15000) @ quality(min: excellent)
    user>
      > Combine all dimension analyses into a unified design document
      > for: {{param.problem}}
      >
      each> dim in {{leg.*}}
        > ## {{dim.aspect.id}} Analysis
        > {{dim}}
      >
      > Produce:
      > 1. Executive Summary (2-3 paragraphs)
      > 2. Problem Statement
      > 3. Proposed Design (overview, key components, interface, data model)
      > 4. Trade-offs and Decisions (decisions made, open questions, trade-offs)
      > 5. Risks and Mitigations
      > 6. Implementation Plan (Phase 1 MVP, Phase 2 Polish, Phase 3 Future)
      >
      > Identify conflicts between dimensions. Flag decisions needing human input.
    format> json {
      executive_summary: str,
      problem_statement: str,
      proposed_design: { overview: str, components: [str],
                         interface_summary: str, data_model_summary: str },
      decisions: [{ decision: str, rationale: str }],
      open_questions: [str],
      risks: [{ risk: str, mitigation: str }],
      implementation_phases: [{ phase: str, items: [str] }]
    }
  #/

  leg.* -> synthesis

##/
```

### Notes

- **Easy**: Identical structural pattern to code-review. Convoy = `map` + synthesis.
- **Hard**: Same issue -- the 6 aspect definitions (api, data, ux, scale, security, integration) are long prose. The `map` construct needs a way to carry per-aspect descriptions.
- The `scope` input with a default value maps cleanly to `input param.scope : string = "medium"`.

---

## 3. mol-algebraic-survey.formula.toml

**Type**: Molecule (sequential steps with dependencies)
**Category**: DIRECT

This is a linear-ish step chain: orient -> {extract-types, extract-operations, extract-state-machines} -> {extract-invariants, map-boundaries} -> synthesize -> submit-and-exit. Steps 2-4 have parallel branches that converge.

### Cell Translation

```cell
## algebraic-survey {

  input param.issue : str required
  input param.subsystem : str required
  input param.packages : str required
  input param.boundary_notes : str required

  -- ================================================================
  -- STEP 1: Orient
  -- ================================================================
  # orient : llm
    @ cost(max: 8000) @ quality(min: good)
    user>
      > You are an algebraic archaeologist surveying **{{param.subsystem}}**.
      > Packages: {{param.packages}}
      >
      > ## Boundary Notes
      > {{param.boundary_notes}}
      >
      > Smart sample the packages:
      > 1. Read doc.go (developer intent)
      > 2. Read ALL type definitions (carrier sets)
      > 3. Read ALL function signatures (operations)
      > 4. Note package sizes for selective body reading later
      >
      > Output: package inventory with file counts, types seen, signatures seen,
      > and initial observations about subsystem boundaries.
    format> json {
      packages: [{ name: str, files: number, types: [str],
                   functions: [str], notes: str }],
      boundary_observations: [str]
    }
  #/

  -- ================================================================
  -- STEP 2a: Extract Types (parallel with 2b, 2c)
  -- ================================================================
  # extract-types : llm
    - orient
    @ cost(max: 15000) @ quality(min: good)
    user>
      > Given package inventory: {{orient}}
      >
      > For each significant type, determine:
      > - Carrier set (what values can it hold?)
      > - Kind: product | sum | state | identity | wrapper | alias
      > - Parameters (type dependencies)
      > - Significant fields with algebraic meaning
      > - Invariants you can infer
      >
      > Look for implicit types: constants/enums (sum types), interfaces
      > (abstract operations), map keys (equivalence classes).
    format> json {
      types: [{ name: str, kind: str, carrier: str,
                params: [str], fields: [str],
                invariants: [str], notes: str }]
    }
  #/

  -- ================================================================
  -- STEP 2b: Extract Operations (parallel with 2a, 2c)
  -- ================================================================
  # extract-operations : llm
    - orient
    @ cost(max: 15000) @ quality(min: good)
    user>
      > Given package inventory: {{orient}}
      >
      > Classify each exported function/method as:
      > constructor | transformer | projection | combinator |
      > predicate | effect | observer
      >
      > For key operations determine: domain/codomain, composition,
      > idempotency, commutativity, partiality.
      > Look for hidden operations in goroutines, channels, context.
    format> json {
      operations: [{ name: str, signature: str,
                     class: str, composes_with: [str],
                     properties: [str], notes: str }]
    }
  #/

  -- ================================================================
  -- STEP 2c: Extract State Machines (parallel with 2a, 2b)
  -- ================================================================
  # extract-state-machines : llm
    - orient
    @ cost(max: 12000) @ quality(min: good)
    user>
      > Given package inventory: {{orient}}
      >
      > Every status field is a state machine someone forgot to draw. Draw them.
      > For each: states, initial state, terminal states, transitions
      > (event + guard + effect), invariants.
      > Look for implicit state machines: boolean gates, mutex, retry loops,
      > channel open/closed.
    format> json {
      state_machines: [{ name: str, states: [str],
                         initial: str, terminal: [str],
                         transitions: [{ from: str, event: str,
                                         to: str, guard: str, effect: str }],
                         invariants: [str] }]
    }
  #/

  -- ================================================================
  -- STEP 3a: Extract Invariants (needs types + ops + state machines)
  -- ================================================================
  # extract-invariants : llm
    - extract-types
    - extract-operations
    - extract-state-machines
    @ cost(max: 15000) @ quality(min: excellent)
    user>
      > Types: {{extract-types}}
      > Operations: {{extract-operations}}
      > State Machines: {{extract-state-machines}}
      >
      > Extract invariants -- the laws of this subsystem's algebra.
      > Sources: validation functions, test assertions, error checks,
      > mutex patterns, defer statements, comments.
      >
      > For each invariant provide:
      > - Informal statement (English)
      > - Formal statement (use quantifiers: forall, exists, implies)
      > - Evidence (where in code)
      > - Confidence (high/medium/low)
      >
      > Look for compositional laws:
      > f(f(x)) = f(x) idempotency, f(g(x)) = g(f(x)) commutativity,
      > f(x, f(y,z)) = f(f(x,y), z) associativity,
      > f(x, e) = x identity element.
    format> json {
      invariants: [{ name: str, informal: str, formal: str,
                     evidence: str, confidence: str, notes: str }],
      algebraic_structure: str
    }
  #/

  -- ================================================================
  -- STEP 3b: Map Boundaries (parallel with 3a)
  -- ================================================================
  # map-boundaries : llm
    - extract-types
    - extract-operations
    - extract-state-machines
    @ cost(max: 12000) @ quality(min: good)
    user>
      > Types: {{extract-types}}
      > Operations: {{extract-operations}}
      > State Machines: {{extract-state-machines}}
      >
      > Map subsystem boundaries for **{{param.subsystem}}**:
      > - Trace imports both directions
      > - Characterize each boundary: direction, interface, coupling, protocol
      > - Look for structure-preserving maps (functors, isomorphisms, embeddings)
      > - Look for monadic patterns (context threading, error returns, builders)
    format> json {
      boundaries: [{ subsystem: str, direction: str,
                     interface: [str], coupling: str,
                     protocol: str, notes: str }],
      structural_maps: [str],
      monadic_patterns: [str]
    }
  #/

  -- ================================================================
  -- STEP 4: Synthesize
  -- ================================================================
  # synthesize : llm
    - extract-invariants
    - map-boundaries
    @ cost(max: 20000) @ quality(min: excellent)
    user>
      > Invariants and Laws: {{extract-invariants}}
      > Boundaries: {{map-boundaries}}
      >
      > Step back. You have types, operations, state machines, invariants,
      > and boundaries. Answer the big question:
      > **What algebra IS this?**
      >
      > Channel Terence Tao:
      > 1. Don't force a name. Describe the structure you actually see.
      > 2. Find analogies (database? process calculus? type system? board game?)
      > 3. Find the simplest non-trivial example (the toy model)
      > 4. Ask what's preserved under transformation
      > 5. Be honest about what you don't know
      >
      > Output: best-fit algebra, confidence, analogy, argument,
      > toy model, open questions, surprises, suggested Lean approach.
    format> json {
      subsystem: str,
      best_fit_algebra: str,
      confidence: str,
      analogy: str,
      argument: str,
      toy_model: str,
      open_questions: [str],
      surprises: [str],
      suggested_lean_approach: str
    }
  #/

  -- ================================================================
  -- TOPOLOGY
  -- ================================================================
  orient -> extract-types
  orient -> extract-operations
  orient -> extract-state-machines

  extract-types -> extract-invariants
  extract-operations -> extract-invariants
  extract-state-machines -> extract-invariants

  extract-types -> map-boundaries
  extract-operations -> map-boundaries
  extract-state-machines -> map-boundaries

  extract-invariants -> synthesize
  map-boundaries -> synthesize

##/
```

### Notes

- **Easy**: The step DAG with `needs` arrays maps directly to Cell's dependency (`- ref`) and wire declarations. The parallel branching (steps 2a/2b/2c) and convergence (step 3a) are natural in Cell.
- **Easy**: Variables (`{{issue}}`, `{{subsystem}}`, `{{packages}}`, `{{boundary_notes}}`) map to `input param.X` and `{{param.X}}`.
- **Omitted**: The `submit-and-exit` step is an operational concern (running `gt done`), not a computation node. Cell describes the data flow graph, not the session lifecycle. This is appropriate -- lifecycle management belongs to the runtime.

---

## 4. mol-survey-dispatch.formula.toml

**Type**: Molecule (meta-orchestrator, dispatches 8 sub-formulas)
**Category**: EXTENDED

This is a meta-formula: it creates beads, slings them to polecats, monitors, and collects. It needs `map` over a subsystem collection and some way to express "invoke sub-formula."

### Cell Translation

```cell
## survey-dispatch {

  input param.issue : str required
  input param.rig : str = "gastown"

  -- ================================================================
  -- SUBSYSTEM DEFINITIONS (the collection to map over)
  -- ================================================================
  -- In Cell, this would be a typed collection or preset.
  -- Each entry carries the variables needed by mol-algebraic-survey.

  preset subsystems {
    items = [
      { name = "Agent Topology", packages = "polecat,crew,witness,mayor,rig",
        boundary_notes = "Connects to Work Distribution, Communication, Spatial Structure. The agent graph IS the topology." },
      { name = "Work Items (Beads)", packages = "beads",
        boundary_notes = "Connects to Communication, Work Distribution, Persistence, Agent Topology. The bead is the universal work unit." },
      { name = "Communication", packages = "mail,nudge,escalation,hook",
        boundary_notes = "Two channels: mail (persistent, heavy) vs nudge (ephemeral, light)." },
      { name = "Work Distribution", packages = "cmd/sling,polecat,convoy",
        boundary_notes = "Sling is the universal dispatcher." },
      { name = "Merge Pipeline", packages = "refinery,mergerequest",
        boundary_notes = "The refinery owns main branch health." },
      { name = "Spatial Structure", packages = "workspace,rig,config,paths",
        boundary_notes = "The filesystem IS the deployment topology." },
      { name = "Persistence & State", packages = "beads/dolt,dolt,metadata",
        boundary_notes = "Dolt is the single source of truth -- and a single point of failure." },
      { name = "Federation & CLI", packages = "cmd,prime,hook,done,federation",
        boundary_notes = "CLI is the user-facing surface of all subsystems." }
    ]
  }

  -- ================================================================
  -- PARALLEL DISPATCH: one algebraic-survey per subsystem
  -- ================================================================
  map # survey : llm over subsystems.items as sub
    -- Each leg is itself a molecule invocation.
    -- This requires Cell to support sub-molecule references.
    @ cost(max: 100000) @ quality(min: good)
    user>
      > Run mol-algebraic-survey for subsystem "{{sub.name}}"
      > with packages={{sub.packages}}
      > and boundary_notes="{{sub.boundary_notes}}"
      > and issue={{param.issue}}
  #/

  -- ================================================================
  -- COLLECTION: verify all surveys complete
  -- ================================================================
  # collect : llm
    - survey.*
    @ cost(max: 5000) @ quality(min: adequate)
    user>
      > Verify all 8 algebraic surveys completed.
      >
      each> result in {{survey.*}}
        > Survey: {{result.sub.name}} -- status: {{result.status}}
      >
      > Report: completion count, any missing sections, readiness for synthesis.
    format> json {
      surveys_complete: number,
      total_surveys: 8,
      missing: [{ subsystem: str, missing_sections: [str] }],
      ready_for_synthesis: boolean
    }
  #/

  survey.* -> collect

##/
```

### Notes

- **Hard**: This is a meta-formula that invokes other formulas. Cell doesn't have a direct "invoke sub-molecule" primitive. The `map` construct handles parallel dispatch, but each leg would need to reference `mol-algebraic-survey` as a sub-graph. This could be expressed as a molecule import or a nested molecule declaration.
- **Hard**: The monitoring step (`monitor`) is an operational concern -- polling polecat status over time. This is inherently imperative and doesn't map to Cell's declarative DAG model. The runtime would handle this.
- **Extended**: The `preset` construct carrying structured items (objects with multiple fields) extends beyond simple string lists.

---

## 5. mol-idea-to-plan.formula.toml

**Type**: Workflow (sequential steps with human gates)
**Category**: EXTENDED

This is the most complex formula: a 7-step pipeline with two human approval gates, three sub-formula invocations (mol-prd-review, design, mol-plan-review), and conditional loopback.

### Cell Translation

```cell
## idea-to-plan {

  input param.problem : str required
  input param.context : str = ""

  -- ================================================================
  -- STEP 1: Intake -- structure idea into draft PRD
  -- ================================================================
  # intake : llm
    @ cost(max: 10000) @ quality(min: good)
    user>
      > You are structuring a raw idea into a draft PRD.
      >
      > **Idea**: {{param.problem}}
      > **Context**: {{param.context}}
      >
      > Write a structured PRD with:
      > - Problem Statement (what, for whom, why now)
      > - Goals (specific, measurable)
      > - Non-Goals (explicitly out of scope)
      > - User Stories / Scenarios
      > - Constraints
      > - Open Questions
      > - Rough Approach (direction, not a plan)
      >
      > Don't polish. Breadth over depth. A draft that exposes uncertainty
      > is more useful than one that hides it.
    format> json {
      review_id: str,
      prd: {
        problem_statement: str,
        goals: [str],
        non_goals: [str],
        user_stories: [str],
        constraints: [str],
        open_questions: [str],
        rough_approach: str
      }
    }
  #/

  -- ================================================================
  -- STEP 2: PRD Review (sub-molecule: mol-prd-review convoy)
  -- 6 parallel legs: requirements, gaps, ambiguity, feasibility,
  --                  scope, stakeholders
  -- ================================================================
  # prd-review : mol(mol-prd-review)
    - intake
    @ cost(max: 80000)
    user>
      > problem = {{param.problem}}
      > context = "See PRD draft: {{intake}}"
  #/

  -- ================================================================
  -- STEP 3: Human Clarification Gate
  -- ================================================================
  # human-clarify : llm
    - prd-review
    @ cost(max: 5000)
    user>
      > PRD review synthesis: {{prd-review}}
      >
      > Present the critical questions to the human.
      > Extract the "Before You Build: Critical Questions" section.
    accept>
      > Human must answer all critical questions.
      > Wait for human reply with numbered answers.
  #/

  -- Oracle gate: human must respond before proceeding
  prd-review -> ? human-responded -> human-clarify

  -- ================================================================
  -- STEP 4: Generate Implementation Plan (sub-molecule: design convoy)
  -- 6 parallel legs: api, data, ux, scale, security, integration
  -- ================================================================
  # generate-plan : mol(design)
    - human-clarify
    @ cost(max: 80000)
    user>
      > problem = {{param.problem}}
      > context = "PRD with clarifications: {{human-clarify}}"
  #/

  -- ================================================================
  -- STEP 5: Plan Review (sub-molecule: mol-plan-review convoy)
  -- 5 parallel legs: completeness, sequencing, risk, scope-creep,
  --                  testability
  -- ================================================================
  # plan-review : mol(mol-plan-review)
    - generate-plan
    @ cost(max: 60000)
    user>
      > plan = {{generate-plan}}
      > problem = {{param.problem}}
      > prd_review = {{prd-review}}
  #/

  -- ================================================================
  -- STEP 6: Human Approval Gate
  -- ================================================================
  # human-approve : llm
    - plan-review
    @ cost(max: 5000)
    user>
      > Plan review verdict: {{plan-review}}
      >
      > Present verdict to human with:
      > - Overall verdict (GO / GO WITH FIXES / NO-GO)
      > - Must-Fix Items
      > - Links to all reports
      >
      > Human replies: APPROVE, APPROVE WITH NOTES, or REVISE.
    accept>
      > Human must reply APPROVE or APPROVE WITH NOTES.
      > If REVISE: loop back to generate-plan with revision notes.
  #/

  plan-review -> ? verdict-acceptable -> human-approve

  -- ================================================================
  -- STEP 7: Create Beads from Approved Plan
  -- ================================================================
  # create-beads : llm
    - human-approve
    - generate-plan
    @ cost(max: 10000) @ quality(min: good)
    user>
      > Approved plan: {{generate-plan}}
      > Approval notes: {{human-approve}}
      >
      > Convert the approved implementation plan into beads:
      > 1. One bead per task/phase
      > 2. Wire dependencies based on sequencing
      > 3. Create tracking epic
      > 4. Verify dependency graph is acyclic
    format> json {
      epic: { title: str, description: str },
      beads: [{ title: str, type: str, description: str,
                acceptance_criteria: [str],
                depends_on: [str] }]
    }
  #/

  -- ================================================================
  -- TOPOLOGY
  -- ================================================================
  intake -> prd-review
  prd-review -> human-clarify
  human-clarify -> generate-plan
  generate-plan -> plan-review
  plan-review -> human-approve
  human-approve -> create-beads
  generate-plan -> create-beads

  -- ================================================================
  -- ORACLE GATES
  -- ================================================================
  # human-responded : oracle
    ``` oracle
    -- Human has provided answers to critical questions
    assert v.answers != null;
    assert len(v.answers) > 0;
    ```
  #/

  # verdict-acceptable : oracle
    ``` oracle
    json_parse(v);
    assert v.verdict in ["GO", "GO WITH FIXES"];
    ```
  #/

##/
```

### Notes

- **Hard**: Human gates are the novel element. Cell's `accept>` construct expresses the acceptance criteria, and `-> ? oracle ->` gated wires enforce the gate. But the actual mechanism for "pause and wait for human reply" is a runtime concern that Cell's syntax can declare but not implement.
- **Hard**: Sub-molecule invocation (`molecule(mol-prd-review)`) is an extension. Cell needs a way to reference and instantiate other molecules as cells within a parent molecule.
- **Hard**: Conditional loopback ("if REVISE, go back to generate-plan") is not expressible in a DAG -- it requires a cycle or a retry mechanism. This would need to be handled by the runtime or expressed as a bounded retry recipe.
- **Extended**: The `required_unless` pattern on inputs (from code-review) would also need to be defined for the sub-molecules' inputs.

---

## 6. mol-prd-review.formula.toml

**Type**: Convoy (parallel legs + synthesis)
**Category**: EXTENDED

Structurally identical to code-review and design convoys. 6 legs: requirements, gaps, ambiguity, feasibility, scope, stakeholders.

### Cell Translation

```cell
## prd-review {

  input param.problem : str required
  input param.context : str

  prompt@ prd-review-base
    > You are a specialized analyst participating in a parallel PRD review convoy.
    >
    > ## Context
    > - **Problem / Feature**: {{param.problem}}
    > - **Your dimension**: {{aspect.focus}}
    >
    > {{param.context}}
    >
    > ## Output Structure
    > - Critical Gaps / Questions (must answer before implementation)
    > - Important Considerations (should address but not blockers)
    > - Observations (non-blocking notes)
    > - Confidence Assessment (High/Medium/Low with rationale)
    >
    > Be specific. Flag unknowns explicitly. Surface what's missing or unclear.

  -- ================================================================
  -- PARALLEL LEGS
  -- ================================================================
  map # leg : llm over aspects as aspect
    @ cost(max: 8000) @ quality(min: good)
    system>
      {{@prd-review-base}}
    user>
      > {{aspect.description}}
    format> json {
      summary: str,
      critical_gaps: [{ question: str, why_it_matters: str,
                        suggested_clarification: str }],
      important_considerations: [str],
      observations: [str],
      confidence: { score: "high" | "medium" | "low", rationale: str }
    }
  #/

  -- ================================================================
  -- SYNTHESIS
  -- ================================================================
  # synthesis : llm
    - leg.*
    @ cost(max: 12000) @ quality(min: excellent)
    user>
      > Combine all leg analyses into a consolidated PRD review.
      >
      each> finding in {{leg.*}}
        > ## {{finding.aspect.id}} Findings
        > {{finding}}
      >
      > Produce:
      > 1. Executive Summary (PRD health, biggest risks, confidence)
      > 2. Before You Build: Critical Questions (must answer before implementation)
      > 3. Important But Non-Blocking
      > 4. Observations and Suggestions
      > 5. Confidence Assessment table (per dimension: H/M/L)
    format> json {
      executive_summary: str,
      critical_questions: [{ category: str, question: str,
                             impact: str, found_by: [str],
                             suggested_options: [str] }],
      non_blocking: [str],
      observations: [str],
      confidence: [{ dimension: str, score: str, notes: str }],
      overall_readiness: str
    }
    ``` oracle
    json_parse(v);
    assert len(v.critical_questions) >= 0;
    assert v.overall_readiness in ["high", "medium", "low"];
    ```
  #/

  leg.* -> synthesis

##/
```

### Notes

- **Easy**: Same convoy pattern as code-review and design. `map` + synthesis.
- **Easy**: The synthesis step includes a mail action (`gt mail send --human`). In Cell, the output format captures the data; the runtime handles delivery.

---

## 7. mol-plan-review.formula.toml

**Type**: Convoy (parallel legs + synthesis)
**Category**: EXTENDED

5 legs: completeness, sequencing, risk, scope-creep, testability. Same convoy pattern.

### Cell Translation

```cell
## plan-review {

  input param.plan : str required
  input param.problem : str required
  input param.prd_review : str

  prompt@ plan-review-base
    > You are a specialized reviewer participating in a parallel plan
    > review convoy.
    >
    > ## Context
    > - **Plan under review**: {{param.plan}}
    > - **Original problem**: {{param.problem}}
    > - **Your dimension**: {{aspect.focus}}
    >
    > {{param.prd_review}}
    >
    > ## Output Structure
    > - Verdict: PASS / PASS WITH NOTES / FAIL (one sentence rationale)
    > - Must Fix (blocks implementation)
    > - Should Fix (important but not blocking)
    > - Observations (non-blocking)
    >
    > Be specific. Reference plan sections where possible.

  -- ================================================================
  -- PARALLEL LEGS
  -- ================================================================
  map # leg : llm over aspects as aspect
    @ cost(max: 8000) @ quality(min: good)
    system>
      {{@plan-review-base}}
    user>
      > {{aspect.description}}
    format> json {
      verdict: "pass" | "pass-with-notes" | "fail",
      rationale: str,
      must_fix: [{ issue: str, why: str, resolution: str }],
      should_fix: [{ issue: str, suggestion: str }],
      observations: [str]
    }
    ``` oracle
    json_parse(v);
    assert v.verdict in ["pass", "pass-with-notes", "fail"];
    if v.verdict == "fail" { assert len(v.must_fix) > 0; }
    ```
  #/

  -- ================================================================
  -- SYNTHESIS
  -- ================================================================
  # synthesis : llm
    - leg.*
    @ cost(max: 10000) @ quality(min: excellent)
    user>
      > Combine all leg reviews into a consolidated verdict.
      >
      each> review in {{leg.*}}
        > ## {{review.aspect.id}} ({{review.verdict}})
        > {{review}}
      >
      > Determine overall verdict: GO / GO WITH FIXES / NO-GO
      > Rules:
      > - Any leg FAIL with unresolvable must-fix = NO-GO
      > - Any leg FAIL with fixable must-fix = GO WITH FIXES
      > - All legs PASS = GO
      >
      > Produce:
      > 1. Overall Verdict (GO / GO WITH FIXES / NO-GO) with rationale
      > 2. Leg Verdicts table
      > 3. Must Fix Before Creating Beads (blocking, deduplicated)
      > 4. Should Fix
      > 5. Observations
    format> json {
      overall_verdict: "GO" | "GO WITH FIXES" | "NO-GO",
      rationale: str,
      leg_verdicts: [{ dimension: str, verdict: str, key_finding: str }],
      must_fix: [{ title: str, found_by: [str],
                   problem: str, required_fix: str }],
      should_fix: [str],
      observations: [str]
    }
    ``` oracle
    json_parse(v);
    assert v.overall_verdict in ["GO", "GO WITH FIXES", "NO-GO"];
    if v.overall_verdict == "NO-GO" { assert len(v.must_fix) > 0; }
    ```
  #/

  leg.* -> synthesis

##/
```

### Notes

- **Easy**: Same pattern as all other convoys. The `depends_on` in TOML maps to `- leg.*` dependency.
- **Easy**: The per-leg oracle validation (verdict must match must_fix count) translates directly to Cell oracle blocks.

---

## 8. mol-polecat-code-review.formula.toml

**Type**: Molecule (sequential steps)
**Category**: DIRECT

A linear pipeline: load-context -> survey-code -> detailed-review -> prioritize-findings -> file-beads -> summarize-review -> complete-and-exit.

### Cell Translation

```cell
## polecat-code-review {

  input param.scope : str required
  input param.issue : str required
  input param.focus : str
  input param.rig : str required

  -- ================================================================
  -- STEP 1: Load Context
  -- ================================================================
  # load-context : llm
    @ cost(max: 5000) @ quality(min: adequate)
    user>
      > You are a code review polecat. Understand your review scope.
      >
      > **Scope**: {{param.scope}}
      > **Focus**: {{param.focus}}
      > **Tracking issue**: {{param.issue}}
      >
      > Locate the code, check recent changes, understand what
      > you're reviewing and why.
    format> json {
      scope_type: "file" | "directory",
      files: [str],
      recent_changes: [str],
      focus_area: str,
      context_notes: str
    }
  #/

  -- ================================================================
  -- STEP 2: Survey Code Structure
  -- ================================================================
  # survey-code : llm
    - load-context
    @ cost(max: 8000) @ quality(min: good)
    user>
      > Context: {{load-context}}
      >
      > Get a high-level understanding:
      > - Main types/structs
      > - Public functions
      > - Dependencies
      > - Tests (if any)
      > - Initial impressions: organization, patterns, risky areas
    format> json {
      components: [{ name: str, type: str, description: str }],
      dependencies: [str],
      has_tests: boolean,
      initial_impressions: [str],
      risky_areas: [str]
    }
  #/

  -- ================================================================
  -- STEP 3: Detailed Review
  -- ================================================================
  # detailed-review : llm
    - survey-code
    @ cost(max: 15000) @ quality(min: good)
    user>
      > Code survey: {{survey-code}}
      > Scope: {{param.scope}}
      > Focus: {{param.focus}}
      >
      > Systematic review checklist:
      > | Category | Look for |
      > |----------|----------|
      > | Correctness | Logic errors, off-by-one, nil handling, races |
      > | Security | Injection, auth bypass, secrets, unsafe ops |
      > | Error handling | Swallowed errors, missing checks |
      > | Performance | N+1 queries, unnecessary allocations |
      > | Maintainability | Dead code, unclear naming |
      > | Testing | Untested paths, missing edge cases |
      >
      > For each issue: file, line, category, severity, description, suggested fix.
    format> json {
      findings: [{ file: str, line: number,
                   category: "correctness" | "security" | "error-handling" |
                             "performance" | "maintainability" | "testing",
                   severity: "critical" | "high" | "medium" | "low",
                   description: str, suggestion: str }]
    }
  #/

  -- ================================================================
  -- STEP 4: Prioritize Findings
  -- ================================================================
  # prioritize-findings : llm
    - detailed-review
    @ cost(max: 5000) @ quality(min: good)
    user>
      > Raw findings: {{detailed-review}}
      >
      > Prioritize and categorize:
      > - P0: Security vulnerability, data loss risk
      > - P1: Bug affecting users, broken functionality
      > - P2: Code quality issue, potential future bug
      > - P3: Improvement opportunity, nice-to-have
      >
      > For P0 issues: flag for immediate escalation.
    format> json {
      p0: [{ file: str, line: number, description: str,
             suggestion: str }],
      p1: [{ file: str, line: number, description: str,
             suggestion: str }],
      p2: [{ file: str, line: number, description: str }],
      p3: [{ file: str, line: number, description: str }],
      has_critical: boolean
    }
    ``` oracle
    json_parse(v);
    if v.has_critical { assert len(v.p0) > 0; }
    ```
  #/

  -- ================================================================
  -- STEP 5: File Beads (output -- generates bead creation commands)
  -- ================================================================
  # file-beads : llm
    - prioritize-findings
    @ cost(max: 5000) @ quality(min: adequate)
    user>
      > Prioritized findings: {{prioritize-findings}}
      > Scope: {{param.scope}}
      > Issue: {{param.issue}}
      >
      > For each finding, prepare a bead specification:
      > - Type: bug (P0, P1) or task (P2, P3)
      > - Priority: 0-3 matching P-level
      > - Title: clear description
      > - Description: location, issue, impact, suggested fix
    format> json {
      beads: [{ type: "bug" | "task", priority: number,
                title: str, description: str }],
      total_findings: number
    }
  #/

  -- ================================================================
  -- STEP 6: Summarize Review
  -- ================================================================
  # summarize-review : llm
    - file-beads
    - prioritize-findings
    @ cost(max: 3000) @ quality(min: adequate)
    user>
      > Beads filed: {{file-beads}}
      > Findings: {{prioritize-findings}}
      > Scope: {{param.scope}}
      > Focus: {{param.focus}}
      >
      > Produce a review summary:
      > - P0 count, P1 count, P2 count, P3 count
      > - Overall assessment (healthy / needs attention / significant issues)
      > - Key themes across findings
    format> json {
      p0_count: number,
      p1_count: number,
      p2_count: number,
      p3_count: number,
      overall_assessment: "healthy" | "needs-attention" | "significant-issues",
      key_themes: [str],
      beads_filed: number
    }
  #/

  -- ================================================================
  -- TOPOLOGY
  -- ================================================================
  load-context -> survey-code
  survey-code -> detailed-review
  detailed-review -> prioritize-findings
  prioritize-findings -> file-beads
  file-beads -> summarize-review
  prioritize-findings -> summarize-review

##/
```

### Notes

- **Easy**: Linear pipeline with one fan-in at summarize-review. Maps directly to Cell.
- **Omitted**: The `complete-and-exit` step (`gt done`) is runtime lifecycle, not a computation node. Cell describes the data flow; the runtime handles cleanup.
- **Easy**: The `vars` section with `required = true/false` maps to `input param.X : type required` / `input param.X : type`.

---

## Cross-Cutting Observations

### Pattern 1: Convoy = map + synthesis

All four convoy formulas (code-review, design, mol-prd-review, mol-plan-review) follow the identical pattern:
1. Define parallel legs with shared base prompt
2. Map over aspects
3. Synthesize with `- leg.*` dependency

Cell's `map # leg over aspects as aspect` + synthesis cell with `- leg.*` is a direct match. The `prompt@` reusable fragment avoids repeating the base prompt.

### Pattern 2: Sequential molecule = cell chain

Both sequential molecules (mol-algebraic-survey, mol-polecat-code-review) are step chains with `needs` dependencies. These map directly to Cell's `- ref` dependencies and wire declarations.

### Pattern 3: Workflow with human gates

The mol-idea-to-plan workflow introduces human gates and sub-molecule invocation. These require Cell extensions:
- `accept>` for declaring acceptance criteria at human gates
- `-> ? oracle ->` gated wires for enforcing gate conditions
- `molecule(name)` cell type for sub-molecule invocation

### Pattern 4: Lifecycle steps don't translate (and shouldn't)

Steps like "submit-and-exit", "complete-and-exit", "run gt done" are operational lifecycle concerns. Cell correctly omits these -- they belong to the runtime, not the data flow graph.

### Gap Analysis

| Feature | Cell Support | Status |
|---------|-------------|--------|
| Parallel legs (convoy) | `map # name over collection` | EXTENDED (needs collection definition) |
| Synthesis with leg fan-in | `- leg.*` | EXTENDED |
| Named presets | `preset name { ... }` | EXTENDED |
| Reusable prompt fragments | `prompt@ name { ... }` | EXTENDED |
| Sequential step chains | `- ref` + wires | DIRECT |
| Input variables | `input param.X : type` | DIRECT |
| Human gates | `accept>` + `-> ? oracle ->` | EXTENDED |
| Sub-molecule invocation | `molecule(name)` cell type | EXTENDED (not yet in spec) |
| Conditional loopback | N/A | GAP (DAGs are acyclic; runtime concern) |
| Go template syntax | `{{param.X}}` + `each>` | DIRECT (different syntax, same semantics) |
| Operational lifecycle | N/A | Out of scope (correctly) |

### Key Extension Needed: Collection Definition for map

The biggest gap is how to define the collection that `map` iterates over. In TOML, each `[[legs]]` entry defines an aspect with id, title, focus, and description (often 20+ lines of prose). Cell needs a mechanism to:
1. Define a typed collection of aspect objects
2. Carry per-aspect prompt text (potentially long)
3. Allow presets to select subsets of the collection

Options:
- Inline collection literals in Cell syntax
- External aspect definition files referenced by Cell
- Presets that bundle both collection items and parameter sets
