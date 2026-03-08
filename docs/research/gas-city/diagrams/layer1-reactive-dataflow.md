# Layer 1: Reactive Dataflow Substrate — Diagrams

**Source**: S3 Architecture Sketch §2.1 (Reactive Cells), §2.2 (Computation DAGs)

---

## 1. Reactive Cell Schema Extension

How Gas City extends the existing beads schema to support reactive computation.

```mermaid
erDiagram
    BEADS {
        varchar id PK "Existing"
        varchar title "Existing"
        varchar status "Existing"
        boolean dirty "NEW — marks cell as stale"
        text cell_value "NEW — cached output"
        text cell_evaluator "NEW — prompt template or command"
        varchar cutoff_mode "NEW — exact|structural|semantic"
    }
    DEPENDENCIES {
        varchar child_id FK "Existing"
        varchar parent_id FK "Existing"
        boolean reactive "NEW — participates in dirty propagation"
    }
    BEADS ||--o{ DEPENDENCIES : "has reactive deps"
```

---

## 2. Dirty-Marking Propagation Through a DAG

When a source cell changes, dirty flags propagate eagerly through the entire
downstream graph. This is cheap (O(edges), no LLM calls).

```mermaid
graph TD
    subgraph "Initial State: All Clean"
        A1["Cell A<br/>📊 Source Data<br/>dirty: false"]
        B1["Cell B<br/>📝 Analysis<br/>dirty: false"]
        C1["Cell C<br/>📝 Summary<br/>dirty: false"]
        D1["Cell D<br/>📊 Other Source<br/>dirty: false"]
        E1["Cell E<br/>📋 Report<br/>dirty: false"]

        A1 -->|reactive dep| B1
        B1 -->|reactive dep| C1
        B1 -->|reactive dep| E1
        D1 -->|reactive dep| E1
    end

    style A1 fill:#c8e6c9,stroke:#388e3c
    style B1 fill:#c8e6c9,stroke:#388e3c
    style C1 fill:#c8e6c9,stroke:#388e3c
    style D1 fill:#c8e6c9,stroke:#388e3c
    style E1 fill:#c8e6c9,stroke:#388e3c
```

```mermaid
graph TD
    subgraph "After mark_stale(A): Dirty Propagates Downstream"
        A2["Cell A<br/>📊 Source Data<br/>dirty: TRUE ①"]
        B2["Cell B<br/>📝 Analysis<br/>dirty: TRUE ②"]
        C2["Cell C<br/>📝 Summary<br/>dirty: TRUE ③"]
        D2["Cell D<br/>📊 Other Source<br/>dirty: false"]
        E2["Cell E<br/>📋 Report<br/>dirty: TRUE ④"]

        A2 -->|"propagate"| B2
        B2 -->|"propagate"| C2
        B2 -->|"propagate"| E2
        D2 -.->|"no propagation"| E2
    end

    style A2 fill:#ffcdd2,stroke:#c62828
    style B2 fill:#ffcdd2,stroke:#c62828
    style C2 fill:#ffcdd2,stroke:#c62828
    style D2 fill:#c8e6c9,stroke:#388e3c
    style E2 fill:#ffcdd2,stroke:#c62828
```

```mermaid
graph LR
    subgraph "Dirty-Marking Algorithm"
        Start["mark_stale(cell_id)"] --> Check{"cell.dirty?"}
        Check -->|"Already dirty"| Stop["Return<br/>(stop propagation)"]
        Check -->|"Clean"| SetDirty["cell.dirty = true"]
        SetDirty --> Iterate["For each downstream<br/>in reactive_dependents"]
        Iterate --> Recurse["mark_stale(downstream)"]
        Recurse --> Iterate
        Iterate -->|"No more dependents"| Done["Done"]
    end
```

---

## 3. Lazy Evaluation (Stabilization) Flow

Recomputation only happens when an observer demands a fresh value via
`bd stabilize`. This is expensive (LLM calls) and follows the inverted
cost model.

```mermaid
sequenceDiagram
    participant Observer as Observer<br/>(gt prime / polecat)
    participant CellC as Cell C<br/>(dirty)
    participant CellB as Cell B<br/>(dirty)
    participant CellA as Cell A<br/>(dirty, source)
    participant LLM as LLM / Evaluator

    Observer->>CellC: stabilize(C)
    Note over CellC: C is dirty — must stabilize inputs first

    CellC->>CellB: stabilize(B)
    Note over CellB: B is dirty — must stabilize inputs first

    CellB->>CellA: stabilize(A)
    Note over CellA: A is dirty (source cell)
    CellA->>LLM: evaluate(A.evaluator, inputs)
    LLM-->>CellA: new_value_A
    Note over CellA: cutoff(old, new)?
    alt Value changed
        CellA-->>CellA: cell.value = new_value_A<br/>cell.dirty = false
    else Value unchanged (cutoff)
        CellA-->>CellA: cell.dirty = false<br/>keep old value
    end
    CellA-->>CellB: return value_A

    CellB->>LLM: evaluate(B.evaluator, {A: value_A})
    LLM-->>CellB: new_value_B
    Note over CellB: cutoff(old, new)?
    alt Value changed
        CellB-->>CellB: cell.value = new_value_B<br/>cell.dirty = false
    else Value unchanged (cutoff!)
        Note over CellB: CUTOFF — downstream<br/>recomputation stops here
        CellB-->>CellB: cell.dirty = false<br/>keep old value
    end
    CellB-->>CellC: return value_B

    alt B's value changed
        CellC->>LLM: evaluate(C.evaluator, {B: value_B})
        LLM-->>CellC: new_value_C
        CellC-->>CellC: cell.value = new_value_C<br/>cell.dirty = false
    else B's value unchanged (cutoff propagated)
        CellC-->>CellC: cell.dirty = false<br/>keep old value<br/>NO LLM CALL SAVED
    end

    CellC-->>Observer: return value_C
```

```mermaid
graph TD
    subgraph "Stabilization Decision Tree"
        S["stabilize(cell)"] --> D{"cell.dirty?"}
        D -->|"No"| Ret["Return cached<br/>cell.value<br/>💰 FREE"]
        D -->|"Yes"| Ups["Stabilize all<br/>upstream deps first"]
        Ups --> Eval["evaluate(cell.evaluator,<br/>fresh inputs)<br/>💸 EXPENSIVE"]
        Eval --> Cut{"cutoff(old, new)?"}
        Cut -->|"Values match"| Keep["Keep old value<br/>dirty = false<br/>🛑 CUTOFF"]
        Cut -->|"Values differ"| Update["Update cell.value<br/>dirty = false<br/>✅ Propagate"]
    end

    style Ret fill:#c8e6c9,stroke:#388e3c
    style Keep fill:#fff9c4,stroke:#f9a825
    style Update fill:#bbdefb,stroke:#1565c0
    style Eval fill:#ffcdd2,stroke:#c62828
```

---

## 4. Cutoff Predicates

Three modes prevent unnecessary downstream recomputation. The choice trades
comparison cost against recomputation savings.

```mermaid
graph LR
    subgraph "Cutoff Mode Selection"
        Input["New value<br/>computed"] --> Mode{"cutoff_mode?"}

        Mode -->|"exact"| Exact["String equality<br/>old == new"]
        Mode -->|"structural"| Struct["Normalized JSON/YAML<br/>comparison"]
        Mode -->|"semantic"| Sem["LLM-judged<br/>equivalence"]

        Exact --> ExCost["Cost: FREE<br/>Use: deterministic outputs<br/>(builds, tests)"]
        Struct --> StCost["Cost: CHEAP<br/>Use: structured data<br/>(configs, schemas)"]
        Sem --> SeCost["Cost: EXPENSIVE<br/>Use: natural language<br/>(research, analysis)"]
    end

    style ExCost fill:#c8e6c9,stroke:#388e3c
    style StCost fill:#fff9c4,stroke:#f9a825
    style SeCost fill:#ffcdd2,stroke:#c62828
```

```mermaid
graph TD
    subgraph "Cutoff in Action: 5-Cell DAG"
        Source["Source Cell<br/>value changed"] -->|dirty| B["Cell B<br/>Recompute"]
        B -->|"exact cutoff:<br/>same output!"| C["Cell C<br/>🛑 SKIP"]
        B -->|"value changed"| D["Cell D<br/>Recompute"]
        C -.->|"not reached"| F["Cell F<br/>🛑 SKIP"]
        D -->|"structural cutoff:<br/>equivalent JSON"| E["Cell E<br/>🛑 SKIP"]
    end

    style Source fill:#ffcdd2,stroke:#c62828
    style B fill:#bbdefb,stroke:#1565c0
    style C fill:#c8e6c9,stroke:#388e3c
    style D fill:#bbdefb,stroke:#1565c0
    style E fill:#c8e6c9,stroke:#388e3c
    style F fill:#c8e6c9,stroke:#388e3c
```

---

## 5. Computation DAGs — Formula Steps With Inputs/Outputs

Formula steps gain `inputs` and `outputs` declarations, enabling topological
scheduling and parallel dispatch.

```mermaid
graph TD
    subgraph "Linear Formula (Gas Town — unchanged)"
        L1["1. load-context"] --> L2["2. setup-branch"]
        L2 --> L3["3. implement"]
        L3 --> L4["4. self-review"]
        L4 --> L5["5. build-check"]
        L5 --> L6["6. commit"]
        L6 --> L7["7. submit"]
    end
```

```mermaid
graph TD
    subgraph "DAG Formula (Gas City — parallel scheduling)"
        D1["load-context<br/>outputs: [context]"]
        D2["setup-branch<br/>inputs: [context]<br/>outputs: [branch]"]
        D3["implement<br/>inputs: [context, branch]<br/>outputs: [code_changes]"]
        D4["self-review<br/>inputs: [code_changes]<br/>outputs: [review_findings]"]
        D5["build-check<br/>inputs: [code_changes]<br/>outputs: [build_result]"]
        D6["commit<br/>inputs: [code_changes,<br/>review_findings,<br/>build_result]<br/>outputs: [commits]"]
        D7["submit<br/>inputs: [commits]"]

        D1 --> D2
        D2 --> D3
        D3 --> D4
        D3 --> D5
        D4 --> D6
        D5 --> D6
        D6 --> D7
    end

    style D4 fill:#bbdefb,stroke:#1565c0
    style D5 fill:#bbdefb,stroke:#1565c0
```

> Steps `self-review` and `build-check` (blue) execute in **parallel** — they
> share the same input (`code_changes`) but are independent of each other.

---

## 6. Parallel Scheduling Rules

How the DAG scheduler decides which steps to dispatch.

```mermaid
stateDiagram-v2
    [*] --> Pending: Step created

    Pending --> Eligible: All inputs satisfied
    Pending --> Blocked: Missing inputs

    Blocked --> Eligible: Upstream step completes

    Eligible --> Running: Polecat dispatched
    Eligible --> Skipped: Inputs unchanged (cutoff)

    Running --> Completed: Step succeeds
    Running --> Failed: Step errors

    Completed --> [*]: Output available for downstream
    Skipped --> [*]: Cached output reused
    Failed --> [*]: Downstream steps NOT dispatched
```

```mermaid
graph TD
    subgraph "Scheduler: Dispatch Decision"
        Start["Step becomes eligible"] --> Check{"Inputs changed<br/>since last run?"}
        Check -->|"No"| Skip["SKIP step<br/>reuse cached output<br/>💰 Saves LLM call"]
        Check -->|"Yes"| Avail{"Polecat<br/>available?"}
        Avail -->|"Yes"| Dispatch["DISPATCH to polecat<br/>in its own worktree"]
        Avail -->|"No"| Queue["QUEUE until<br/>polecat frees up"]
    end

    style Skip fill:#c8e6c9,stroke:#388e3c
    style Dispatch fill:#bbdefb,stroke:#1565c0
```

---

## 7. Bounded Cycle Unrolling

DAGs cannot express true cycles. Reflection loops (implement -> review -> revise)
are handled by bounded unrolling with a convergence predicate.

```mermaid
graph TD
    subgraph "Bounded Cycle: implement-review-revise (max_iterations: 3)"
        I1["implement ①<br/>outputs: [code]"] --> R1["review ①<br/>inputs: [code]<br/>outputs: [findings]"]
        R1 --> Conv1{"findings.severity<br/>== 'none'?"}
        Conv1 -->|"No"| Rev1["revise ①<br/>inputs: [findings]<br/>outputs: [code]"]

        Rev1 --> R2["review ②<br/>inputs: [code]<br/>outputs: [findings]"]
        R2 --> Conv2{"findings.severity<br/>== 'none'?"}
        Conv2 -->|"No"| Rev2["revise ②<br/>inputs: [findings]<br/>outputs: [code]"]

        Rev2 --> R3["review ③<br/>inputs: [code]<br/>outputs: [findings]"]
        R3 --> Conv3{"max_iterations<br/>reached"}

        Conv1 -->|"Yes ✓"| Exit1["Exit cycle<br/>→ downstream steps"]
        Conv2 -->|"Yes ✓"| Exit2["Exit cycle<br/>→ downstream steps"]
        Conv3 --> Exit3["Exit cycle<br/>(forced termination)<br/>→ downstream steps"]
    end

    style Exit1 fill:#c8e6c9,stroke:#388e3c
    style Exit2 fill:#c8e6c9,stroke:#388e3c
    style Exit3 fill:#fff9c4,stroke:#f9a825
```

---

## 8. Dynamic Dependency Discovery

A cell's true dependencies are discovered at evaluation time, not declared
statically. The evaluator records which cells it reads during execution.

```mermaid
sequenceDiagram
    participant Sched as Scheduler
    participant Cell as Cell B
    participant Tracker as DependencyTracker
    participant Store as Cell Store

    Sched->>Cell: evaluate(B)
    Cell->>Tracker: start tracking

    Cell->>Store: read(Cell A)
    Store-->>Cell: value_A
    Tracker->>Tracker: record read(A)

    Cell->>Store: read(Cell D)
    Store-->>Cell: value_D
    Tracker->>Tracker: record read(D)

    Note over Cell: Conditionally reads Cell E<br/>(only if value_D > threshold)
    alt value_D > threshold
        Cell->>Store: read(Cell E)
        Store-->>Cell: value_E
        Tracker->>Tracker: record read(E)
    end

    Cell-->>Cell: compute result
    Cell->>Tracker: stop tracking
    Tracker-->>Cell: recorded_reads = [A, D, E]
    Cell->>Cell: reactive_deps = [A, D, E]

    Note over Cell: Next evaluation may read<br/>different cells — deps are<br/>always up to date
```

---

## 9. Multi-Agent DAG Execution

Each step dispatches to exactly one polecat. Parallelism happens at the step
level across polecats, not within a single step.

```mermaid
graph LR
    subgraph "DAG Step Dispatch"
        Step1["load-context ✅"]
        Step2["setup-branch ✅"]
        Step3["implement ✅"]

        subgraph "Parallel Fan-Out"
            Step4["self-review"]
            Step5["build-check"]
        end

        Step6["commit"]
        Step7["submit"]
    end

    subgraph "Polecat Assignment"
        P1["🔧 Polecat A<br/>(worktree A)"]
        P2["🔧 Polecat B<br/>(worktree B)"]
        P3["🔧 Polecat C<br/>(worktree C)"]
    end

    Step3 --> Step4
    Step3 --> Step5
    Step4 --> Step6
    Step5 --> Step6
    Step6 --> Step7

    Step4 -.- P1
    Step5 -.- P2
    Step6 -.- P3

    style Step4 fill:#bbdefb,stroke:#1565c0
    style Step5 fill:#bbdefb,stroke:#1565c0
```

---

## 10. The Inverted Cost Model

Gas City's reactive computation operates under fundamentally different cost
assumptions than traditional reactive systems.

```mermaid
quadrantChart
    title Cost Model Comparison
    x-axis "Cheap Operation" --> "Expensive Operation"
    y-axis "Traditional Reactive" --> "Gas City"
    "Mark dirty (GC)": [0.1, 0.9]
    "Track deps (GC)": [0.2, 0.8]
    "Evaluate cell (GC)": [0.9, 0.85]
    "Over-compute (GC)": [0.95, 0.95]
    "Mark dirty (Trad)": [0.4, 0.2]
    "Track deps (Trad)": [0.7, 0.15]
    "Evaluate cell (Trad)": [0.1, 0.1]
    "Over-compute (Trad)": [0.3, 0.05]
```

```mermaid
graph TB
    subgraph "Design Consequence: Mark Aggressively, Evaluate Lazily"
        Change["Source cell changes"] -->|"EAGER — cheap"| Mark["Mark entire downstream<br/>graph dirty<br/>⏱️ milliseconds<br/>💰 zero LLM cost"]
        Mark -->|"LAZY — expensive"| Demand{"Observer demands<br/>a value?"}
        Demand -->|"No"| Skip["Do nothing<br/>💰 zero cost"]
        Demand -->|"Yes"| Eval["Stabilize on demand<br/>⏱️ seconds<br/>💸 LLM cost per cell"]
        Eval --> Cutoff{"Cutoff?"}
        Cutoff -->|"Match"| Stop["Stop propagation<br/>💰 save downstream cost"]
        Cutoff -->|"Differ"| Continue["Continue to<br/>dependent cells"]
    end

    style Mark fill:#c8e6c9,stroke:#388e3c
    style Skip fill:#c8e6c9,stroke:#388e3c
    style Stop fill:#fff9c4,stroke:#f9a825
    style Eval fill:#ffcdd2,stroke:#c62828
```

---

## Sources

- S3: Architecture Sketch — What Gas City Adds (§2.1 Reactive Cells, §2.2 Computation DAGs)
- R2: Reactive Dataflow and Incremental Computation
- Adapton: Composable, Demand-Driven Incremental Computation (Hammer et al.)
