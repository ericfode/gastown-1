package reactive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseAllTestdata ensures every YAML file in testdata/ parses successfully
// and produces a valid sheet with correct cell types and ref wiring.
func TestParseAllTestdata(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatalf("read testdata dir: %v", err)
	}

	var count int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		count++
		t.Run(e.Name(), func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", e.Name()))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			s, err := ParseYAML(data)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			validateSheet(t, s)
		})
	}
	if count == 0 {
		t.Fatal("no YAML files found in testdata/")
	}
	t.Logf("validated %d testdata files", count)
}

// TestParseSpecExamples parses every Cell YAML example from the spec and design
// docs. These are the "survey translations" — real-world use cases translated
// into Cell YAML.
func TestParseSpecExamples(t *testing.T) {
	examples := []struct {
		name string
		yaml string
	}{
		{
			name: "incident-postmortem",
			yaml: `
name: incident-postmortem
cells:
  - name: timeline
    type: inventory
    prompt: |
      Reconstruct the incident timeline from the following log excerpts,
      alert history, and chat transcripts. List events chronologically
      with timestamps, actors, and systems affected.

      Logs: (attached at eval time)

  - name: blast-radius
    type: boundaries
    prompt: |
      From the incident timeline, determine the blast radius:
      - Which services were directly affected?
      - Which downstream consumers experienced degradation?
      - What was the user-facing impact (error rates, latency, data loss)?
      - Geographic and tenant scope.

      Timeline:
      {{timeline}}
    refs: [timeline]

  - name: root-cause
    type: synthesis
    prompt: |
      Analyze the incident timeline for root cause. Apply the "5 Whys"
      method. Distinguish between:
      - Proximate cause (the trigger)
      - Contributing factors (what made it worse)
      - Systemic cause (why the system was vulnerable)

      Timeline:
      {{timeline}}
    refs: [timeline]

  - name: detection-gap
    type: laws
    refs: [timeline, blast-radius]
    prompt: |
      Compare the incident timeline against the blast radius assessment.
      Identify detection gaps:
      - How long between first symptom and first alert?
      - Were any affected services missing monitoring?
      - What signal would have caught this earlier?

      Timeline:
      {{timeline}}

      Blast radius:
      {{blast-radius}}

  - name: remediation-plan
    type: code
    refs: [root-cause, detection-gap]
    prompt: |
      Given the root cause analysis and detection gaps, produce a
      remediation plan with concrete action items. Each item must have:
      - Owner (team, not individual)
      - Priority (P0-P3)
      - Estimated effort (hours/days/weeks)
      - Verification criteria

      Root cause:
      {{root-cause}}

      Detection gaps:
      {{detection-gap}}

  - name: executive-summary
    type: decision
    refs: [blast-radius, root-cause, remediation-plan]
    prompt: |
      Write a 1-page executive summary of this incident for leadership.
      Include: what happened (2 sentences), impact (metrics), root cause
      (1 sentence), and top 3 remediation items with owners and timelines.
      No jargon. No blame.

      Blast radius:
      {{blast-radius}}

      Root cause:
      {{root-cause}}

      Remediation plan:
      {{remediation-plan}}
`,
		},
		{
			name: "api-migration-assessment",
			yaml: `
name: api-migration-assessment
cells:
  - name: old-api-surface
    type: inventory
    prompt: |
      Catalog the current (v2) API surface. For each endpoint, list:
      - HTTP method and path
      - Request/response schema (key fields only)
      - Rate limits and auth requirements
      - Known consumers (services and teams)

      API spec: (attached at eval time)

  - name: new-api-surface
    type: inventory
    prompt: |
      Catalog the proposed (v3) API surface. For each endpoint, list:
      - HTTP method and path
      - Request/response schema (key fields only)
      - Breaking vs non-breaking changes from v2
      - New capabilities not present in v2

      API spec: (attached at eval time)

  - name: breaking-changes
    type: boundaries
    refs: [old-api-surface, new-api-surface]
    prompt: |
      Diff the v2 and v3 API surfaces. Produce a table of breaking changes:
      | v2 Endpoint | Change Type | v3 Equivalent | Migration Effort |
      Classify each as: removed, renamed, schema-changed, auth-changed,
      or semantics-changed.

      v2 API:
      {{old-api-surface}}

      v3 API:
      {{new-api-surface}}

  - name: consumer-impact
    type: synthesis
    refs: [old-api-surface, breaking-changes]
    prompt: |
      For each known consumer from the v2 API inventory, assess impact:
      - Which breaking changes affect this consumer?
      - Estimated migration effort (trivial/moderate/significant)
      - Can they run dual-stack (v2+v3) during transition?
      - Risk level if they don't migrate by deprecation date

      v2 consumers:
      {{old-api-surface}}

      Breaking changes:
      {{breaking-changes}}

  - name: compatibility-shim
    type: code
    refs: [breaking-changes]
    prompt: |
      Design a compatibility shim layer that translates v2 requests into
      v3 calls. For each breaking change, specify:
      - Translation logic (pseudocode)
      - Edge cases the shim cannot handle
      - Performance overhead estimate
      - Sunset timeline recommendation

      Breaking changes:
      {{breaking-changes}}

  - name: migration-runbook
    type: code
    refs: [consumer-impact, compatibility-shim]
    prompt: |
      Write a migration runbook for platform consumers. Structure as phases:
      Phase 1: Deploy compatibility shim (zero consumer changes)
      Phase 2: Early adopter migration (willing teams)
      Phase 3: Bulk migration (remaining consumers)
      Phase 4: Shim sunset and v2 decommission

      For each phase, specify: entry criteria, steps, rollback plan,
      success metrics.

      Consumer impact:
      {{consumer-impact}}

      Compatibility shim:
      {{compatibility-shim}}

  - name: go-no-go
    type: decision
    refs: [consumer-impact, migration-runbook]
    prompt: |
      Render a go/no-go decision for the v2 to v3 migration. Consider:
      - Total consumer count and high-risk consumer count
      - Shim coverage (what percentage of breaking changes are shimmable?)
      - Aggregate migration effort across all consumers
      - Risk of NOT migrating (v2 tech debt, security, performance)

      Recommendation must be: GO, CONDITIONAL GO (with conditions), or NO-GO.

      Consumer impact:
      {{consumer-impact}}

      Migration runbook:
      {{migration-runbook}}
`,
		},
		{
			name: "supply-chain-vuln-triage",
			yaml: `
name: supply-chain-vuln-triage
cells:
  - name: vuln-feed
    type: inventory
    prompt: |
      Parse the vulnerability advisory feed. For each CVE, extract:
      - CVE ID, CVSS score, attack vector
      - Affected package and version range
      - Patch availability (yes/no, which version)
      - Exploit maturity (POC, weaponized, in-the-wild)

      Feed data: (attached at eval time)

  - name: dep-graph
    type: inventory
    prompt: |
      Analyze the project's dependency graph. For each direct and
      transitive dependency, list:
      - Package name and locked version
      - Depth in dependency tree (direct=0, transitive=1+)
      - Number of reverse dependents in our codebase
      - Last update date

      Lock file: (attached at eval time)

  - name: runtime-profile
    type: boundaries
    prompt: |
      From production telemetry, determine which dependencies are
      actually loaded at runtime vs build-time-only:
      - Packages imported in hot paths (request handling)
      - Packages used only in tests, codegen, or CI
      - Packages imported but never called (dead code)

      Telemetry data: (attached at eval time)

  - name: exposure-map
    type: synthesis
    refs: [vuln-feed, dep-graph, runtime-profile]
    prompt: |
      Cross-reference the vulnerability feed against our dependency
      graph and runtime profile. For each CVE that affects us:
      - Is the vulnerable package a direct or transitive dep?
      - Is it loaded at runtime or build-time-only?
      - Is the vulnerable code path reachable from our code?
      - Effective CVSS (adjusted for our exposure)

      Vulnerabilities:
      {{vuln-feed}}

      Dependencies:
      {{dep-graph}}

      Runtime profile:
      {{runtime-profile}}

  - name: upgrade-paths
    type: code
    refs: [exposure-map, dep-graph]
    prompt: |
      For each exposed CVE, determine the upgrade path:
      - Direct dep: bump to patched version, check API compat
      - Transitive dep: which direct dep pulls it in? Can we bump?
      - If no patch exists: is there a workaround or alternative package?
      - Dependency conflicts (version constraints from other deps)

      Exposure map:
      {{exposure-map}}

      Full dependency graph:
      {{dep-graph}}

  - name: risk-matrix
    type: diagram
    refs: [exposure-map]
    prompt: |
      Build a risk matrix (likelihood x impact) for all exposed CVEs.
      Classify each into quadrants:
      - CRITICAL: high likelihood + high impact
      - HIGH: high in one dimension
      - MEDIUM: moderate in both
      - LOW: low in both

      Exposure map:
      {{exposure-map}}

  - name: patch-plan
    type: decision
    refs: [upgrade-paths, risk-matrix]
    prompt: |
      Produce a prioritized patch plan. Order by risk quadrant
      (CRITICAL first), then by upgrade complexity (easiest first
      within each quadrant).

      Upgrade paths:
      {{upgrade-paths}}

      Risk matrix:
      {{risk-matrix}}

  - name: stakeholder-brief
    type: text
    refs: [risk-matrix, patch-plan]
    prompt: |
      Write a stakeholder brief summarizing:
      - Total CVEs found / total affecting us / total critical
      - Key risk: the single scariest finding
      - Patch plan summary: how many patches, estimated total effort
      - Residual risk: what we are choosing NOT to patch and why

      Risk matrix:
      {{risk-matrix}}

      Patch plan:
      {{patch-plan}}
`,
		},
		{
			name: "code-review",
			yaml: `
name: code-review
cells:
  - name: read-code
    type: text
    prompt: "Read the source files in src/"

  - name: find-bugs
    type: inventory
    prompt: "Given this code: {{read-code}}, list potential bugs."
    refs: [read-code]

  - name: find-patterns
    type: inventory
    prompt: "Given this code: {{read-code}}, list design patterns used."
    refs: [read-code]

  - name: review-report
    type: synthesis
    prompt: |
      Bugs found: {{find-bugs}}
      Patterns found: {{find-patterns}}
      Write a comprehensive code review.
    refs: [find-bugs, find-patterns]
`,
		},
		{
			name: "onboarding",
			yaml: `
name: onboarding
cells:
  - name: codebase
    type: text
    prompt: "Read the repository structure and key files."

  - name: docs
    type: text
    prompt: "Read README, CONTRIBUTING, and architecture docs."

  - name: architecture
    type: synthesis
    prompt: "Codebase: {{codebase}}. Docs: {{docs}}. Describe the architecture."
    refs: [codebase, docs]

  - name: getting-started
    type: decision
    prompt: "Given architecture: {{architecture}}, write a getting-started guide."
    refs: [architecture]
`,
		},
		{
			name: "parse-test-inline-algebraic-survey",
			yaml: `
name: algebraic-survey
cells:
  - name: extract-types
    type: inventory
    prompt: |
      Read the Go source and list all algebraic types.
  - name: extract-laws
    type: laws
    prompt: |
      Read the Go source and identify invariants.
  - name: synthesize
    type: synthesis
    refs: [extract-types, extract-laws]
    prompt: |
      Given the type inventory: {{extract-types}}
      And the invariants: {{extract-laws}}
      What algebra is this?
`,
		},
		{
			name: "all-cell-types",
			yaml: `
name: all-cell-types
cells:
  - name: t-text
    type: text
    prompt: "A text cell."
  - name: t-inventory
    type: inventory
    prompt: "An inventory cell."
  - name: t-diagram
    type: diagram
    prompt: "A diagram cell."
  - name: t-laws
    type: laws
    prompt: "A laws cell."
  - name: t-boundaries
    type: boundaries
    prompt: "A boundaries cell."
  - name: t-synthesis
    type: synthesis
    prompt: "A synthesis cell."
  - name: t-code
    type: code
    prompt: "A code cell."
  - name: t-decision
    type: decision
    prompt: "A decision cell."
`,
		},
		{
			name: "field-ref-example",
			yaml: `
name: field-ref-test
cells:
  - name: source
    type: text
    prompt: "Read the data."
  - name: consumer
    type: synthesis
    refs: [source.summary]
    prompt: "Use {{source.summary}} to produce output."
`,
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			s, err := ParseYAML([]byte(ex.yaml))
			if err != nil {
				t.Fatalf("parse %q: %v", ex.name, err)
			}
			validateSheet(t, s)
		})
	}
	t.Logf("validated %d spec examples", len(examples))
}

// validateSheet checks structural invariants on a parsed sheet.
func validateSheet(t *testing.T, s *Sheet) {
	t.Helper()

	if s.Name == "" {
		t.Error("sheet has empty name")
	}

	cells := s.Cells()
	if len(cells) == 0 {
		t.Error("sheet has no cells")
	}

	// Check for duplicate cell names.
	seen := make(map[string]bool)
	for _, c := range cells {
		if c.Name == "" {
			t.Error("cell with empty name")
		}
		if seen[c.Name] {
			t.Errorf("duplicate cell name: %q", c.Name)
		}
		seen[c.Name] = true
	}

	// Check that all refs point to cells that exist in this sheet.
	for _, c := range cells {
		for _, r := range c.Refs {
			if !seen[r.Cell] {
				t.Errorf("cell %q refs unknown cell %q", c.Name, r.Cell)
			}
		}
	}

	// Check that template {{ref}} placeholders reference existing cells.
	for _, c := range cells {
		for _, ref := range extractTemplateRefs(c.Prompt) {
			if !seen[ref] {
				t.Errorf("cell %q prompt references unknown cell {{%s}}", c.Name, ref)
			}
		}
	}

	// Verify all cells start in empty state.
	for _, c := range cells {
		st, err := s.State(c.Name)
		if err != nil {
			t.Errorf("State(%q): %v", c.Name, err)
		}
		if st != StateEmpty {
			t.Errorf("cell %q initial state = %s, want empty", c.Name, st)
		}
	}

	// Verify ready set is non-empty (at least leaf cells should be ready).
	ready := s.ReadySet()
	if len(ready) == 0 {
		t.Error("ready set is empty (no leaf cells?)")
	}

	// Verify leaf cells (no refs and no template refs) are in the ready set.
	for _, c := range cells {
		upstreams := s.upstreamRefs(c.Name)
		if len(upstreams) == 0 {
			found := false
			for _, r := range ready {
				if r == c.Name {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("leaf cell %q should be in ready set but isn't; ready=%v", c.Name, ready)
			}
		}
	}

	// Check for cycles: no cell should be its own upstream (transitively).
	if cycle := detectCycle(s); cycle != "" {
		t.Errorf("cycle detected: %s", cycle)
	}

	t.Logf("sheet %q: %d cells, %d ready", s.Name, len(cells), len(ready))
}

// detectCycle does a simple DFS to find cycles in the sheet's dependency graph.
func detectCycle(s *Sheet) string {
	type color int
	const (
		white color = iota
		gray
		black
	)
	colors := make(map[string]color)
	for _, c := range s.Cells() {
		colors[c.Name] = white
	}

	var dfs func(name string) string
	dfs = func(name string) string {
		colors[name] = gray
		for _, upstream := range s.upstreamRefs(name) {
			switch colors[upstream] {
			case gray:
				return fmt.Sprintf("%s -> %s", name, upstream)
			case white:
				if cycle := dfs(upstream); cycle != "" {
					return cycle
				}
			}
		}
		colors[name] = black
		return ""
	}

	for _, c := range s.Cells() {
		if colors[c.Name] == white {
			if cycle := dfs(c.Name); cycle != "" {
				return cycle
			}
		}
	}
	return ""
}
