# The Power User's Case for Gas City

**Author**: A power user coordinating 10+ agents across multiple rigs daily.

---

## 1. What I WISH I Had Right Now

### The coordination tax is killing me

I run a typical day with 3 rigs, 8 polecats, and maybe 15 beads in flight. Here's what actually happens:

**Yesterday I had four polecats working on related changes across the gastown codebase.** Polecat A was refactoring the dispatch system. Polecat B was adding a new formula type. Polecat C was fixing a bug in the witness. Polecat D was writing tests for the refinery. These are all touching overlapping code. What do I have to coordinate this? Mail. Literal mail messages between agents that create Dolt commits and have zero structure beyond "subject" and "body."

I needed to express: "B depends on A's dispatch refactor landing before its new formula type can wire up. C's witness fix might conflict with A's refactor. D should wait for both A and B before writing integration tests." In Gas Town today, I do this by:

1. Creating beads with `bd dep add` — but dependencies are binary (blocked/unblocked) with no type information. I can't say "B needs the `DispatchResult` type that A is defining." I can only say "B needs A."
2. Sending mail to polecats telling them about each other's work — which they promptly forget when their context compacts.
3. Manually checking `bd blocked` to see what unblocked after A merged — then manually dispatching B.
4. Hoping C doesn't create a merge conflict with A because I have no way to express "these two touch the same file" in the dependency graph.

**What I WISH I had:** A sheet where cell A is "dispatch refactor", cell B is "new formula type", and there's a typed wire from A to B carrying `DispatchResult`. When A completes, B's inputs automatically refresh. I can SEE the whole computation at a glance — not by running five `bd show` commands and mentally reconstructing the DAG.

### Staleness blindness

Last week, a polecat completed an analysis of the codebase structure. Three hours later, another polecat merged a major refactor. The analysis was now stale — but nothing told anyone. The synthesis polecat downstream consumed the stale analysis and produced garbage. I only caught it because I manually noticed the timestamps didn't add up.

**In a spreadsheet, the stale cell would be highlighted red.** I wouldn't need to manually cross-reference timestamps. The system would KNOW that the upstream changed and the downstream is stale.

### The "what's actually happening" problem

Right now, to understand the state of my multi-agent computation, I run:

```bash
bd ready --json          # What's unblocked?
gt peek                  # What are polecats doing?
bd show <id>             # What's the status of this specific thing?
git log --oneline -5     # What actually landed?
```

Four commands, four different views, zero integration. I'm the human join operation. I'm the one mentally merging these four result sets into a coherent picture of "where are we." This is EXACTLY what a spreadsheet does — it shows you all the cells, their values, their staleness, their dependencies, in ONE view.

---

## 2. The Spreadsheet Equivalent for Agent Coordination

### Cells = Work Units (Beads with Typed Outputs)

In Excel, a cell has a value and a formula. In Gas City, a cell is a bead with:
- A **prompt template** (the formula)
- **Typed inputs** (references to other cells, with types — not just "depends on")
- A **computed value** (the LLM output)
- **Staleness state** (fresh/stale/empty)
- **Effect metadata** (cost, quality, provenance)

The killer feature over beads: **cells have typed wires.** Not just "A depends on B" but "A consumes B's `TypeInventory` output as its `types` input." This is the difference between a TODO list and a spreadsheet.

### Hotkeys: The Operations I'd Use Every Day

**Ctrl+R: Recompute Cell** — Force re-evaluation of a stale cell. Today I do this by creating a new bead, assigning it to a polecat, waiting for dispatch. In Gas City, I'd just hit recompute and the engine finds a capable agent and dispatches.

**Ctrl+Shift+R: Recompute Stale Subtree** — Find all stale cells downstream of a change and recompute them in topological order. Today this is MANUAL. I have to trace the dependency graph myself, figure out the right order, dispatch each one. Gas City's reactive engine does this automatically because it knows the DAG and the staleness state.

**Drag-to-fill: Formula Templates** — In Excel, you drag a formula across a row and it adjusts references. In Gas City, you take a formula like "analyze module X" and instantiate it across a list of modules. Today I do this with `bd mol pour` — but molecules are heavyweight (each step is a full bead with Dolt storage). I want lightweight parameterized instantiation: "run this cell type across these 10 inputs" without creating 10 beads.

**Pivot tables: Aggregation Views** — "Show me all cells grouped by quality level." "Show me total token cost by formula." "Show me which agents are computing the most cells." Today I have zero aggregation capability. I can list beads. I can show individual beads. There's no `bd pivot` command that gives me a multi-dimensional view of my computation.

**Conditional formatting: Visual Staleness** — Every stale cell red. Every fresh cell green. Every empty cell gray. Every in-progress cell yellow. Today, staleness is invisible unless I manually check. I want the SHEET to show me at a glance: "60% fresh, 30% stale, 10% empty — you need to recompute these 4 cells."

### Named Ranges: Formula Scopes

In Excel, you name a range so formulas can reference it semantically. In Gas City, a formula is a named range — a composition of cells with a semantic name. "The type synthesis formula" isn't cell B7 — it's `synthesis.types`, and it has a type, and you can reference it from other formulas.

---

## 3. What Would Make Me 10x More Productive

### 10x Factor #1: Cost Visibility and Budgets

Right now, dispatching work is a GAMBLE. I create a molecule with 8 steps, pour it, and hope it doesn't burn 500K tokens. I have zero visibility into cost until AFTER execution. The effect system in Gas City gives me `seq_cost` and `par_cost` as the molecule runs — I can watch token spend accumulate in real time and set hard caps. After a molecule completes, I see exactly what it cost. When I pour the same formula again, I have historical cost data from prior digests.

This is the `VLOOKUP` equivalent: **given a formula's digest chain, LOOK UP what it actually cost last time.** Today I'm flying blind. Gas City gives me instrumentation — not prediction, but measurement and history.

### 10x Factor #2: Automatic Dispatch via Type Matching

Today's dispatch is manual sling. I look at a bead, decide which polecat should handle it, and dispatch. Gas City has `AgentCapability` — agents declare what cell types they can handle and at what quality level. The engine matches cells to agents automatically: "This cell needs an `inventory` computation. Agents A, C, and E can do inventories. A is cheapest, E is highest quality. Budget says use A."

This eliminates the entire sling ceremony. I define the computation graph (the sheet), annotate quality requirements, and the engine dispatches. I go from "manually assigning 15 beads to 8 polecats" to "defining one sheet and pressing go."

### 10x Factor #3: Composition That Actually Composes

Today's formulas are TOML checklists. They don't compose. You can't take the output of formula A and feed it as input to formula B. You can BOND molecules together, but bonding is structural — it doesn't carry data. The steps in a bonded molecule don't know what the previous molecule produced.

Gas City's cell wires carry data. Cell B's prompt template literally contains `{{A.output}}`. When A computes, its value flows into B's prompt. This is function composition, not just dependency ordering. The difference is:

- **Today**: "Step 2 runs after Step 1" (temporal ordering)
- **Gas City**: "Step 2 consumes Step 1's output" (data flow)

Data flow means I can REASON about what information flows where. I can see that a synthesis cell aggregates three analysis cells. I can see that changing one analysis invalidates the synthesis but not the other analyses. I can see the shape of the computation, not just its schedule.

### 10x Factor #4: Incremental Recomputation

Today, if something goes wrong in step 5 of an 8-step molecule, I have two options: (a) create a new bead for step 5 and manually dispatch it, or (b) re-pour the entire molecule from scratch. There's no "recompute just this cell and propagate."

Gas City's staleness propagation gives me surgical recomputation. Mark cell 5 as stale. The engine knows cells 6, 7, 8 depend on it. Recompute 5, then 6, then 7, then 8 — but DON'T recompute 1-4 because they're still fresh. This is the reactive spreadsheet model, and it's transformative for long-running multi-step agent workflows.

---

## 4. What SUCKS About Coordinating Agents Today

### Dependencies are untyped garbage

`bd dep add B A` says "B needs A." It doesn't say WHAT B needs from A. It doesn't say whether B needs A's code changes, A's analysis output, A's test results, or A's mere completion. Every dependency is the same: a boolean edge in a graph with no labels, no types, no data flow.

This means I can't distinguish between:
- "B needs A to finish so the branch is clean" (structural dependency)
- "B needs A's type inventory as input" (data dependency)
- "B and A touch the same file so B must merge after A" (conflict dependency)

These are fundamentally different relationships but they're all represented as the same arrow. Gas City's typed wires fix this. A wire from A to B carries a `CellType` — I know exactly what data flows through it.

### Mail is the worst IPC mechanism ever invented for agents

Agents communicate via mail: `gt mail send gastown/witness -s "HELP" -m "..."`. Every message creates a Dolt commit. Every message is permanent. Every message has zero structure beyond subject/body. There's no schema, no types, no machine-readable fields.

When a polecat needs to tell another polecat "here's the type inventory I found," it puts it in a mail body as unstructured text. The receiving polecat has to parse prose to extract structured data. This is INSANE. In a spreadsheet, the value is in the cell. The downstream cell references it directly. No serialization, no parsing, no "hope the LLM extracts the right fields from this wall of text."

### Molecule steps have no memory

Each step in a molecule is a fresh dispatch to a (potentially different) polecat. The new polecat gets the bead description and... that's it. It doesn't get the output of the previous step. It doesn't get the analysis that step 2 produced. It starts from scratch, re-reads the codebase, re-discovers things that the previous step already found.

Yes, you can use `bd update --notes` to persist findings. But this is OPT-IN and LOSSY. The polecat has to decide what to persist, serialize it into a text field, and hope the next polecat reads it. There's no structured handoff of computed values between steps.

In Gas City, cell outputs are first-class values. They persist in the sheet. They flow through typed wires. The next cell in the chain gets exactly the data it needs, automatically, without any manual serialization ceremony.

### No cost visibility, no quality tracking

I have ZERO idea how many tokens my polecats are burning. I have ZERO idea whether the output of step 3 was good enough for step 4 to consume. There's no quality signal, no cost accounting, no "this formula has cost 200K tokens and produced adequate-quality output."

The effect system in Gas City isn't academic — it's the instrumentation I desperately need. When I'm deciding whether to re-run a stale computation, I want to know: "This will cost ~15K tokens and produce good-quality output." Not "let me dispatch a polecat and find out in 20 minutes."

### The "spreadsheet view" doesn't exist

The most damning indictment: Gas Town has no overview. There's no single command that shows me "here's your entire computation graph, here's what's fresh, here's what's stale, here's what's in progress, here's what it's going to cost to finish." I'm assembling this picture from five different commands, every single time, multiple times per day.

A spreadsheet IS this view. It IS the overview. Every cell, every value, every dependency, every staleness state — visible at once. This is what makes spreadsheets the most successful end-user programming tool in history. And it's what Gas Town desperately needs.

---

## The Bottom Line

Gas Town is assembly language. It works. I can coordinate agents with beads and molecules and mail. But I'm hand-coding `MOV` and `ADD` instructions when I should be writing in a high-level language.

Gas City is that high-level language. Typed wires instead of untyped dependencies. Reactive staleness instead of manual timestamp checking. Cost tracking and budget caps instead of blind dispatch. Automatic type-directed dispatch instead of manual sling. Composition that carries data, not just ordering.

The Lean formalization isn't just math — it's a SPEC. It tells me exactly what properties the system guarantees: staleness propagates correctly (proven). Readiness is monotone (proven). Effects compose predictably (proven). Parallel execution is never slower than sequential (proven). These aren't aspirational — they're verified invariants.

I don't want 10% improvements to Gas Town. I want the paradigm shift that Gas City represents. Give me the spreadsheet. Give me the reactive engine. Give me the effect system. I'll coordinate 100 agents instead of 10.
