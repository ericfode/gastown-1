# Gas City: Visualization & Interaction Design

## How Gas City Maps to Gas Town (Zero New Infrastructure)

```mermaid
graph TB
    subgraph "Gas City (new layer)"
        GRID["Living Grid UI"]
        SANKEY["Sankey View"]
        TRACE["Provenance Trace"]
        POLICY["Recompute Policy Engine"]
        EVAL["gt eval / gt map"]
    end

    subgraph "Gas Town (existing)"
        BD["bd (beads)"]
        SLING["gt sling"]
        POLECAT["polecats"]
        DOLT["Dolt"]
        FORMULA["formulas (TOML)"]
    end

    subgraph "What Changes in Gas Town"
        STALE["+ stale:bool on beads"]
        SNAP["+ input_snapshot on beads"]
        CDEPTH["+ compression_depth:int"]
        TMPL["+ prompt_template field"]
    end

    GRID --> BD
    GRID --> DOLT
    SANKEY --> BD
    TRACE --> DOLT
    POLICY --> SLING
    EVAL --> SLING
    EVAL --> POLECAT

    BD --- STALE
    BD --- SNAP
    BD --- CDEPTH
    FORMULA --- TMPL

    style GRID fill:#4a9eff,color:#fff
    style SANKEY fill:#4a9eff,color:#fff
    style TRACE fill:#4a9eff,color:#fff
    style POLICY fill:#ff9f43,color:#000
    style EVAL fill:#ff9f43,color:#000
    style STALE fill:#10ac84,color:#fff
    style SNAP fill:#10ac84,color:#fff
    style CDEPTH fill:#10ac84,color:#fff
    style TMPL fill:#10ac84,color:#fff
```

**Key point:** Gas City is a LENS on Gas Town, not a replacement. The
existing bead/formula/polecat infrastructure stays. We add 4 fields to
beads, a template field to formulas, and build the visualization +
policy layer on top.

## Iteration Model: Bounded Loops via Convergence

DAGs can't represent loops. But LLM workflows need iteration:
draft → review → revise → review → approve. Gas City handles this
via **bounded unrolling with convergence detection**.

```mermaid
graph LR
    subgraph "Iteration: max 3 rounds"
        D["draft₁"] -->|"output"| R1["review₁"]
        R1 -->|"feedback"| REV1["revise₁"]
        REV1 -->|"output"| R2["review₂"]
        R2 -->|"feedback"| REV2["revise₂"]
        REV2 -->|"output"| R3["review₃"]
        R3 -->|"approved ✓"| FINAL["final"]
    end

    subgraph "Convergence check"
        CC["diff(revise₁, revise₂)<br/>< threshold?"]
    end

    REV1 -.-> CC
    REV2 -.-> CC
    CC -.->|"converged"| FINAL

    style D fill:#4a9eff,color:#fff
    style R1 fill:#ff9f43,color:#000
    style REV1 fill:#4a9eff,color:#fff
    style R2 fill:#ff9f43,color:#000
    style REV2 fill:#4a9eff,color:#fff
    style R3 fill:#ff9f43,color:#000
    style FINAL fill:#10ac84,color:#fff
```

The loop is unrolled into a DAG with convergence gates. Each review
cell checks if the revision materially changed. If not → converged,
skip remaining rounds. The `convergent(maxRounds)` recomputation
policy enforces the bound.

## Architecture Overview

```mermaid
graph TB
    subgraph "Gas City Layers"
        L4["Layer 4: Visualization<br/>CellView, Sankey, Provenance"]
        L3["Layer 3: Operations<br/>Pin, Snapshot, Policy"]
        L2["Layer 2: Effects<br/>Cost, Quality, Compression"]
        L1["Layer 1: Reactive DAG<br/>Staleness, Readiness"]
        L0["Layer 0: Beads<br/>Dependencies, Values"]
    end

    L4 --> L3
    L3 --> L2
    L2 --> L1
    L1 --> L0

    style L4 fill:#4a9eff,color:#fff
    style L3 fill:#ff9f43,color:#fff
    style L2 fill:#10ac84,color:#fff
    style L1 fill:#ee5253,color:#fff
    style L0 fill:#576574,color:#fff
```

## The Seven Visualizations

### 1. The Living Grid

The primary view. Each cell shows its value, status, and metadata at a glance.

```mermaid
graph LR
    subgraph "Sheet: mol-algebraic-survey"
        A["📄 extract-types<br/>─────────────<br/><i>Found 12 types:<br/>Graph, DAG, Cell...</i><br/>─────────────<br/>🟢 fresh · 📊 5.2k tok<br/>🔍 depth:0 · ⚡ good"]
        B["📄 find-patterns<br/>─────────────<br/><i>3 patterns found:<br/>Monoidal, Reactive...</i><br/>─────────────<br/>🟡 stale · 📊 8.1k tok<br/>🔍 depth:1 · ⚡ adequate"]
        C["📄 synthesize<br/>─────────────<br/><i>The system forms a<br/>typed reactive...</i><br/>─────────────<br/>🟡 stale · 📊 12k tok<br/>🔍 depth:2 · ⚡ good"]
        D["📄 write-report<br/>─────────────<br/><i>(computing...)</i><br/>─────────────<br/>🔵 computing · 📊 —<br/>🔍 depth:3 · ⚡ —"]
    end

    A -->|"inventory"| B
    A -->|"inventory"| C
    B -->|"laws"| C
    C -->|"synthesis"| D
```

**Color key:**
- 🟢 Green border = fresh (up to date)
- 🟡 Amber border = stale (upstream changed)
- 🔵 Blue border = computing (LLM working)
- ⬜ Gray border = empty (never computed)
- 🔴 Red border = failed

### 2. The Information Sankey

Shows information narrowing through the pipeline. Width = token count.

```mermaid
sankey-beta
    Source Document, Extract Types, 10000
    Extract Types, Find Patterns, 5200
    Extract Types, Synthesize, 5200
    Find Patterns, Synthesize, 8100
    Synthesize, Write Report, 3000
    Synthesize, Decision, 500
```

**What you SEE:** The 10,000-token source document narrows to a 500-token
decision. Each node is a compression step. The Sankey makes the funnel
visceral — you can immediately spot where the biggest information drops
happen.

**Interactive:** Hover a flow to see compression ratio. Click a node to
see the full cell output. Double-click to see the prompt template.

### 3. The Staleness Wave

Animated propagation when an upstream cell changes.

```mermaid
sequenceDiagram
    participant U as User
    participant A as extract-types
    participant B as find-patterns
    participant C as synthesize
    participant D as write-report

    Note over A: 🟢 fresh
    Note over B: 🟢 fresh
    Note over C: 🟢 fresh
    Note over D: 🟢 fresh

    U->>A: gt eval extract-types
    Note over A: 🔵 computing...
    A->>A: LLM produces new output
    Note over A: 🟢 fresh (v2)

    A-->>B: staleness wave
    Note over B: 🟡 STALE
    A-->>C: staleness wave
    Note over C: 🟡 STALE
    B-->>D: (already stale via C)
    C-->>D: staleness wave
    Note over D: 🟡 STALE

    Note over U: User sees amber ripple<br/>through the grid
```

### 4. The Cost Flame Graph

Token expenditure as nested blocks. Width = tokens. Color = quality.

```mermaid
graph TB
    subgraph "Total: 26,300 tokens"
        subgraph "extract-types: 5,200 tok [good]"
            ET["████████████"]
        end
        subgraph "find-patterns: 8,100 tok [adequate]"
            FP["█████████████████"]
        end
        subgraph "synthesize: 12,000 tok [good]"
            SY["██████████████████████████"]
        end
        subgraph "decision: 1,000 tok [draft]"
            DE["██"]
        end
    end

    style ET fill:#10ac84
    style FP fill:#ff9f43
    style SY fill:#10ac84
    style DE fill:#4a9eff
```

**Interactive:** Click a block to expand into sub-costs (prompt tokens vs
output tokens). Drag the quality slider to see how cost changes:
"What if I run synthesize at draft instead of good?"

### 5. The Compression Depth Heatmap

Topological view colored by how far each cell is from raw data.

```mermaid
graph LR
    subgraph "Depth 0 (raw)"
        S1["source-code"]
        S2["test-results"]
        S3["git-history"]
    end
    subgraph "Depth 1 (1x compressed)"
        E1["extract-types"]
        E2["extract-tests"]
    end
    subgraph "Depth 2 (2x compressed)"
        P1["find-patterns"]
        P2["test-coverage"]
    end
    subgraph "Depth 3 (3x compressed)"
        SY["synthesize"]
    end
    subgraph "Depth 4 (4x compressed)"
        DE["decision"]
    end

    S1 --> E1
    S2 --> E2
    S3 --> E1
    E1 --> P1
    E2 --> P2
    P1 --> SY
    P2 --> SY
    SY --> DE

    style S1 fill:#0a3d62,color:#fff
    style S2 fill:#0a3d62,color:#fff
    style S3 fill:#0a3d62,color:#fff
    style E1 fill:#1e6f5c,color:#fff
    style E2 fill:#1e6f5c,color:#fff
    style P1 fill:#ff9f43,color:#000
    style P2 fill:#ff9f43,color:#000
    style SY fill:#ee5253,color:#fff
    style DE fill:#b71540,color:#fff
```

**What you SEE:** Cells go from cool blue (raw data, depth 0) to hot red
(heavily compressed, depth 4). When cell `decision` gives a wrong answer,
you immediately see: it's 4 compressions from reality. Trace backwards
through the cooling colors to find where critical information was dropped.

### 6. The Provenance Trace (View Precedents)

Click any cell → see where its information came from.

```mermaid
graph RL
    subgraph "Tracing: synthesize"
        SY["🎯 synthesize<br/><i>'The system forms a<br/>typed reactive calculus'</i>"]
    end

    subgraph "Direct inputs"
        ET["extract-types (v3)<br/>compression: summarize<br/><i>'12 types: Graph, DAG,<br/>Cell, Wire, Formula...'</i>"]
        FP["find-patterns (v2)<br/>compression: extract<br/><i>'3 patterns: Monoidal,<br/>Reactive, Typed'</i>"]
    end

    subgraph "Transitive sources"
        SC["source-code (v1)<br/>compression: verbatim<br/><i>47 files, 12,000 LOC</i>"]
        GH["git-history (v1)<br/>compression: verbatim<br/><i>342 commits</i>"]
    end

    SY -->|"ref: extract-types<br/>depth: 2"| ET
    SY -->|"ref: find-patterns<br/>depth: 2"| FP
    ET -->|"ref: source-code<br/>depth: 1"| SC
    ET -->|"ref: git-history<br/>depth: 1"| GH
    FP -->|"ref: extract-types<br/>depth: 1"| ET
```

**Interactive:** Click on a specific SENTENCE in synthesize's output →
highlight which upstream content contributed to it. This is the
"View Precedents" (Ctrl+[) of the agent spreadsheet.

### 7. The Multiverse Diff

Re-run a cell, compare outputs side by side.

```mermaid
graph TB
    subgraph "Run 1 (v2)"
        R1["synthesize v2<br/>─────────────<br/>🟢 'Monoidal category'<br/>🟢 'Typed DAG structure'<br/>🔴 'Uses pi-calculus'<br/>🟡 'Reactive staleness'"]
    end
    subgraph "Run 2 (v3)"
        R2["synthesize v3<br/>─────────────<br/>🟢 'Monoidal category'<br/>🟢 'Typed DAG structure'<br/>🔵 'Uses effect algebra'<br/>🟡 'Lazy evaluation'"]
    end

    R1 --- R2

    style R1 fill:#f8f9fa,color:#000
    style R2 fill:#f8f9fa,color:#000
```

**Legend:** 🟢 Stable across runs (signal) · 🔴 Lost (was in v2, not v3) ·
🔵 Added (new in v3) · 🟡 Changed (present in both, different wording)

---

## The Interaction Model: Spreadsheet Operations → Gas City

### Cell Selection & Navigation

```mermaid
graph TB
    subgraph "Keyboard Shortcuts"
        K1["Arrow keys: Navigate cells"]
        K2["Enter: Expand cell (full output)"]
        K3["Tab: Next stale cell"]
        K4["Shift+Tab: Previous stale cell"]
        K5["Ctrl+[: Trace precedents"]
        K6["Ctrl+]: Trace dependents"]
        K7["F9: Recompute selected cell"]
        K8["Ctrl+F9: Recompute all stale"]
        K9["F2: Edit prompt template"]
        K10["Ctrl+P: Pin/unpin cell value"]
    end
```

### The Pin & Rerun Workflow (Freeze Panes)

```mermaid
sequenceDiagram
    participant U as User
    participant G as Grid View
    participant E as Engine

    U->>G: Select cell 'extract-types'
    U->>G: Ctrl+P (pin)
    G->>G: Cell border becomes 📌 dashed
    U->>G: Type custom value into cell
    Note over G: extract-types is now PINNED<br/>with user-supplied value

    U->>G: Select cell 'synthesize'
    U->>G: F9 (recompute)
    G->>E: evaluate 'synthesize' with<br/>pinned upstream values
    E->>E: Fill prompt using pinned value<br/>for extract-types
    E->>G: New output for synthesize

    Note over U: "Aha! The synthesis is correct<br/>with my manual types.<br/>The problem is in extract-types."

    U->>G: Select cell 'extract-types'
    U->>G: Ctrl+P (unpin)
    G->>G: Cell returns to real value
    Note over G: Staleness re-propagates
```

### The Map Operation (Drag to Fill)

```mermaid
graph TB
    subgraph "Template: mol-analyze-repo"
        T1["scan<br/>{{param.repo_url}}"]
        T2["classify<br/>{{scan}}"]
        T1 --> T2
    end

    subgraph "Parameters (repos.csv)"
        P["alpha, github.com/org/alpha<br/>beta, github.com/org/beta<br/>gamma, github.com/org/gamma"]
    end

    subgraph "Result: 3 × 2 = 6 cells"
        subgraph "alpha"
            A1["alpha/scan"] --> A2["alpha/classify"]
        end
        subgraph "beta"
            B1["beta/scan"] --> B2["beta/classify"]
        end
        subgraph "gamma"
            G1["gamma/scan"] --> G2["gamma/classify"]
        end
    end

    subgraph "Aggregation (Pivot)"
        AGG["aggregate-classifications<br/>policy: summarize<br/>'Across 3 repos: 2 monoliths,<br/>1 microservice'"]
    end

    T1 -.->|"gt map"| A1
    T1 -.->|"gt map"| B1
    T1 -.->|"gt map"| G1
    A2 --> AGG
    B2 --> AGG
    G2 --> AGG
```

### Recomputation Policy Decision Tree

```mermaid
graph TD
    START["Cell is STALE"] --> POLICY{"What policy?"}

    POLICY -->|eager| RECOMPUTE["✅ Recompute now"]
    POLICY -->|lazy| WAIT["⏸️ Mark stale, wait for trigger"]
    POLICY -->|budgeted| BUDGET{"Cost ≤ budget?"}
    POLICY -->|convergent| ROUNDS{"Rounds < max?"}
    POLICY -->|gated| HUMAN["🧑 Ask human"]

    BUDGET -->|yes| RECOMPUTE
    BUDGET -->|no| EXCEEDED["❌ Budget exceeded, skip"]

    ROUNDS -->|yes| CHANGED{"Output changed<br/>from last run?"}
    ROUNDS -->|no| CONVERGED["✅ Converged, stop"]

    CHANGED -->|yes| RECOMPUTE
    CHANGED -->|no| CONVERGED

    style RECOMPUTE fill:#10ac84,color:#fff
    style WAIT fill:#ff9f43,color:#000
    style EXCEEDED fill:#ee5253,color:#fff
    style CONVERGED fill:#10ac84,color:#fff
    style HUMAN fill:#4a9eff,color:#fff
```

---

## Mockup: The Full Gas City UI

### Main View Layout

```
┌─────────────────────────────────────────────────────────────────┐
│ Gas City: mol-algebraic-survey                    [▶ Run All]   │
│ Budget: 45,000 tok remaining │ 4 fresh │ 2 stale │ 1 computing │
├────────────┬────────────────────────────────────────────────────┤
│            │                                                    │
│ FORMULA    │              LIVING GRID                           │
│ TREE       │                                                    │
│            │  ┌──────────┐    ┌──────────┐    ┌──────────┐     │
│ ▼ survey   │  │extract   │───▶│patterns  │───▶│synthesize│     │
│   extract  │  │──────────│    │──────────│    │──────────│     │
│   patterns │  │Found 12  │    │3 patterns│    │Typed     │     │
│   ▶synth.  │  │types:    │    │found:    │    │reactive  │     │
│   report   │  │Graph,DAG │    │Monoidal, │    │calculus  │     │
│   decision │  │Cell...   │    │Reactive..│    │with...   │     │
│            │  │──────────│    │──────────│    │──────────│     │
│ VIEWS      │  │🟢 5.2k  │    │🟡 8.1k  │    │🟡 12k   │     │
│ ○ Grid     │  │depth:0   │    │depth:1   │    │depth:2   │     │
│ ○ Sankey   │  └──────────┘    └──────────┘    └──────────┘     │
│ ○ Flame    │                        │                           │
│ ○ Heatmap  │                        ▼                           │
│ ○ Trace    │               ┌──────────────┐                    │
│            │               │write-report  │                    │
│ POLICY     │               │──────────────│                    │
│ ● lazy     │               │(computing...)│                    │
│ ○ eager    │               │──────────────│                    │
│ ○ budgeted │               │🔵 est: 15k  │                    │
│            │               │depth:3       │                    │
│            │               └──────────────┘                    │
├────────────┴────────────────────────────────────────────────────┤
│ PROMPT INSPECTOR (synthesize)                                   │
│ ┌──────────────────────────────────────────────────────────────┐│
│ │ Given the type inventory: [{{extract-types}}],               ││
│ │ and the patterns found: [{{find-patterns}}],                 ││
│ │ what algebraic structure does this system form?              ││
│ │                                                              ││
│ │ [{{extract-types}}] → links to cell, click to expand        ││
│ └──────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### Expanded Cell View

```
┌─────────────────────────────────────────────────────────────────┐
│ 📄 synthesize (v2)                          [📌 Pin] [🔄 Rerun]│
├─────────────────────────────────────────────────────────────────┤
│ STATUS: 🟡 stale (upstream extract-types changed)              │
│ COST: 12,000 tokens │ QUALITY: good │ DEPTH: 2                 │
│ AGENT: gastown/polecats/rictus │ MODEL: claude-sonnet-4-6      │
│ COMPUTED: 2026-03-08 14:23:07 UTC                              │
│ INPUT SNAPSHOT:                                                 │
│   extract-types v3 (current: v4 ⚠️)                            │
│   find-patterns v2 (current: v2 ✓)                             │
├─────────────────────────────────────────────────────────────────┤
│ OUTPUT:                                                         │
│                                                                 │
│ The system forms a typed reactive calculus with the following   │
│ algebraic structure:                                            │
│                                                                 │
│ 1. A monoidal category where cells are objects and typed       │
│    wires are morphisms. Composition is well-typed by the       │
│    port compatibility predicate.                                │
│                                                                 │
│ 2. An effect monoid tracking cost (tokens) and quality         │
│    (draft < adequate < good < excellent) with proven           │
│    composition laws.                                            │
│ [...]                                                           │
├─────────────────────────────────────────────────────────────────┤
│ HISTORY: v1 (draft) → v2 (good)                                │
│ COMPRESSION: summarize (from 13,300 tok input → 3,200 tok out) │
│ PROVENANCE: ← extract-types ← source-code, git-history        │
└─────────────────────────────────────────────────────────────────┘
```

### Sankey Detail View

```
┌─────────────────────────────────────────────────────────────────┐
│ INFORMATION FLOW: mol-algebraic-survey                          │
│                                                                 │
│ source-code ████████████████████████████████████████  47,000 tok│
│              ╲                                                  │
│               ╲ [extract: types]                                │
│                ╲                                                │
│ extract-types   ████████████  5,200 tok  (ratio: 9:1)          │
│                  ╲     ╲                                        │
│                   ╲     ╲ [extract: patterns]                   │
│                    ╲     ╲                                      │
│ find-patterns       ╲    ████████████████  8,100 tok            │
│                      ╲        ╱                                 │
│                       ╲      ╱ [summarize]                      │
│                        ╲    ╱                                   │
│ synthesize              ██████  3,200 tok  (ratio: 4:1)        │
│                           ╲                                     │
│                            ╲ [decide]                           │
│                             ╲                                   │
│ decision                     █  500 tok  (ratio: 6:1)          │
│                                                                 │
│ TOTAL: 47,000 → 500  (overall ratio: 94:1)                    │
│ BOTTLENECK: extract-types (9:1 — most information dropped here)│
└─────────────────────────────────────────────────────────────────┘
```

---

## Feynman Diagrams for Agent Computation

### Tree-Level: Simple Feed-Forward

```mermaid
graph LR
    IN(("input")) -->|"prompt"| V1["🔵 LLM call<br/>(extract)"]
    V1 -->|"output"| V2["🔵 LLM call<br/>(synthesize)"]
    V2 -->|"output"| OUT(("result"))

    style V1 fill:#4a9eff,color:#fff
    style V2 fill:#4a9eff,color:#fff
```

**Cost:** 2 LLM calls. No loops. This is the "Born approximation" —
the cheapest computation that produces any answer.

### One-Loop: Single Review Cycle

```mermaid
graph LR
    IN(("input")) -->|"prompt"| V1["🔵 draft"]
    V1 -->|"output"| V2["🟡 review"]
    V2 -->|"feedback"| V3["🔵 revise"]
    V3 -->|"output"| OUT(("result"))

    V2 -.->|"quality<br/>signal"| V1

    style V1 fill:#4a9eff,color:#fff
    style V2 fill:#ff9f43,color:#000
    style V3 fill:#4a9eff,color:#fff
```

**Cost:** 3 LLM calls. The loop adds a "correction" to the tree-level
answer. Like a one-loop Feynman diagram, this is the first perturbative
correction — more expensive but more accurate.

### Two-Loop: Draft → Review → Revise → Review → Final

```mermaid
graph LR
    IN(("input")) --> D["draft"]
    D --> R1["review₁"]
    R1 --> REV["revise"]
    REV --> R2["review₂"]
    R2 --> F["final"]
    F --> OUT(("result"))

    R1 -.->|"feedback"| D
    R2 -.->|"feedback"| REV

    style D fill:#4a9eff,color:#fff
    style R1 fill:#ff9f43,color:#000
    style REV fill:#4a9eff,color:#fff
    style R2 fill:#ff9f43,color:#000
    style F fill:#10ac84,color:#fff
```

**The perturbation series analogy:** Each loop improves quality but costs
more. The series converges — after enough review cycles, the answer
stabilizes (convergent recomputation policy). The art is knowing when to
truncate the series. In QED, higher-loop diagrams contribute less. In
agent computation, diminishing returns on review cycles.

### Fan-Out: Parallel Exploration

```mermaid
graph TB
    IN(("input")) --> A["agent A<br/>(opus)"]
    IN --> B["agent B<br/>(sonnet)"]
    IN --> C["agent C<br/>(haiku)"]
    A --> M["merge /<br/>synthesize"]
    B --> M
    C --> M
    M --> OUT(("result"))

    style A fill:#b71540,color:#fff
    style B fill:#10ac84,color:#fff
    style C fill:#4a9eff,color:#fff
    style M fill:#576574,color:#fff
```

**Cost:** par(A, B, C) + merge. The parallel diagram is cheaper in
wall-clock time than running sequentially (par_le_seq). The merge
vertex is where information from different "paths" recombines — like
a vertex in a Feynman diagram where particles interact.

---

## State Machine: Cell Lifecycle

```mermaid
stateDiagram-v2
    [*] --> empty: cell created

    empty --> computing: gt eval (dispatch)
    computing --> fresh: LLM completes
    computing --> failed: LLM errors

    fresh --> stale: upstream changed
    fresh --> computing: force recompute (F9)
    fresh --> pinned: Ctrl+P (pin value)

    stale --> computing: recompute (policy allows)
    stale --> pinned: Ctrl+P (pin value)

    failed --> computing: retry
    failed --> pinned: Ctrl+P (override)

    pinned --> fresh: Ctrl+P (unpin, value was fresh)
    pinned --> stale: Ctrl+P (unpin, upstream changed while pinned)

    note right of pinned
        Pinned cells resist recomputation.
        Used for debugging: "what if this
        cell had THIS value instead?"
    end note

    note right of stale
        Staleness propagates through
        the DAG but does NOT auto-trigger
        recomputation (lazy reactive).
    end note
```

## Data Flow: Recomputation with Input Snapshots

```mermaid
sequenceDiagram
    participant U as User
    participant E as Engine
    participant D as Dolt
    participant L as LLM

    U->>E: gt eval synthesize
    E->>E: Check policy → allowed

    E->>D: Read current values of<br/>extract-types, find-patterns
    D->>E: extract-types v4, find-patterns v2

    E->>E: Create InputSnapshot<br/>{extract-types: v4, find-patterns: v2}

    E->>E: Fill prompt template with values
    E->>L: Send filled prompt

    L->>E: Response (3,200 tokens)

    E->>D: Write new value (synthesize v3)
    E->>D: Write ComputationRecord<br/>{cell: synthesize, inputs: snapshot,<br/>output: v3, content: "..."}

    E->>E: Propagate staleness downstream
    E->>U: Cell updated, downstream marked stale
```

## The Complete Spreadsheet ↔ Gas City Operation Map

```mermaid
mindmap
    root((Gas City<br/>Operations))
        Direct Manipulation
            Click cell → expand
            Type value → pin
            Drag to connect → wire
            Delete wire → disconnect
        Navigation
            Arrow keys → move
            Tab → next stale
            Ctrl+[ → precedents
            Ctrl+] → dependents
            Ctrl+` → toggle formula/value
        Computation
            F9 → recompute cell
            Ctrl+F9 → recompute all stale
            Shift+F9 → recompute with budget
        Template
            F2 → edit prompt template
            gt map → fill down (parameterize)
            Copy cell → duplicate with new params
        Debugging
            Ctrl+P → pin/freeze cell
            Ctrl+Z → revert to previous version
            Diff view → compare runs
            Sankey view → see information flow
        Policy
            Set per-cell recompute policy
            Budget slider → cost control
            Quality dial → quality/cost tradeoff
            Convergence limit → max iterations
```

---

## Example Session: Debugging a Wrong Synthesis

A walkthrough of finding and fixing a problem using Gas City's tools.

### Step 1: Notice the Problem

```mermaid
graph LR
    A["extract-types<br/>🟢 v4"] --> C["synthesize<br/>🟢 v3<br/>❌ WRONG"]
    B["find-patterns<br/>🟢 v2"] --> C
    C --> D["decision<br/>🟢 v1"]

    style C fill:#ee5253,color:#fff
```

User sees synthesize has a wrong conclusion. Which input caused it?

### Step 2: Check Input Snapshot

Open the expanded cell view for synthesize. See:
- Input: extract-types **v4** (current: v4 ✓)
- Input: find-patterns **v2** (current: v2 ✓)

Both inputs are current. The problem is either in the inputs themselves
or in the synthesis prompt.

### Step 3: Pin and Rerun

```mermaid
sequenceDiagram
    participant U as User
    participant ET as extract-types
    participant SY as synthesize

    U->>ET: Ctrl+P → Pin with known-good value
    Note over ET: 📌 Pinned: "5 types: Graph, DAG,<br/>Cell, Wire, Formula"

    U->>SY: F9 → Recompute
    Note over SY: Uses pinned value from extract-types<br/>+ real value from find-patterns

    alt Synthesis is now correct
        Note over U: Problem was in extract-types v4!
        U->>ET: Ctrl+P → Unpin
        U->>ET: F9 → Recompute extract-types
    else Synthesis is still wrong
        Note over U: Problem is in the prompt template<br/>or in find-patterns
        U->>SY: F2 → Edit prompt template
    end
```

### Step 4: Trace the Sankey

Switch to Sankey view. See that extract-types compresses 47,000 tokens
to 5,200 (9:1 ratio). That's aggressive. The critical types might be
getting dropped in the compression.

**Fix:** Change extract-types' compression policy from `summarize` to
`extract` (structured data, higher fidelity). Recompute. Check if
synthesize now gets the right answer.

### Step 5: Verify with Multiverse Diff

Rerun synthesize three times. Compare outputs:
- Run 1: "monoidal category" ✅ stable
- Run 2: "effect algebra" ✅ stable
- Run 3: "pi-calculus model" ❌ volatile (appeared once, disappeared)

The volatile claim "pi-calculus model" is noise. The stable claims are
signal. The synthesis is now reliable.
