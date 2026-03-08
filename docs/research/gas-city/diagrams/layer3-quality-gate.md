# Layer 3: Quality Gate — Critic Lens Diagrams

**Source**: S3 Architecture Sketch §4.1 (Critic Lens)

---

## 1. Refinery Pipeline: Current vs Gas City

Shows how the Critic Lens inserts before expensive gates.

```mermaid
flowchart LR
    subgraph current["Current Pipeline"]
        direction LR
        A1[MR Arrives] --> B1[Run Gates\nbuild / test / lint]
        B1 --> C1{Gates Pass?}
        C1 -->|Yes| D1[Merge]
        C1 -->|No| E1[Bisect]
    end

    subgraph gascity["Gas City Pipeline"]
        direction LR
        A2[MR Arrives] --> B2[Critic Review]
        B2 --> C2{Verdict?}
        C2 -->|PASS| D2[Run Gates\nbuild / test / lint]
        C2 -->|CONCERNS| D2
        C2 -->|BLOCK| F2[Quarantine\nfor Review]
        D2 --> E2{Gates Pass?}
        E2 -->|Yes| G2[Merge]
        E2 -->|No| H2[Bisect]
    end

    current ~~~ gascity
```

---

## 2. Critic Review Process

End-to-end flow of a single Critic review — inputs, evaluation, outputs.

```mermaid
flowchart TD
    MR[MR Submitted to\nMerge Queue] --> GATHER[Gather Inputs]

    GATHER --> DIFF["git diff origin/main...HEAD\n(full diff)"]
    GATHER --> DESC["bd show &lt;issue&gt;\n(bead description)"]
    GATHER --> REFL["Reflection JSON\n(if available)"]

    DIFF --> PROMPT[Compose Critic Prompt]
    DESC --> PROMPT
    REFL --> PROMPT

    PROMPT --> LLM["LLM Call\n(different model/prompt\nthan implementer)"]

    LLM --> OUTPUT["Structured Output\n{verdict, findings[], confidence}"]

    OUTPUT --> REVIEW_BEAD["Create Review Bead\n(attached to MR)"]

    OUTPUT --> DECIDE{Route by Verdict}

    DECIDE -->|PASS| GATES[Proceed to Gates]
    DECIDE -->|CONCERNS| LOG[Log Findings]
    LOG --> GATES
    DECIDE -->|BLOCK| QUARANTINE[Quarantine MR]

    style LLM fill:#f9e2af,stroke:#f5c211
    style QUARANTINE fill:#f38ba8,stroke:#d62d4a
    style GATES fill:#a6e3a1,stroke:#40a02b
```

---

## 3. Verdict Threshold Decision Matrix

How findings map to verdicts based on severity and confidence.

```mermaid
flowchart TD
    START[Critic Findings] --> SEV{Max Finding\nSeverity?}

    SEV -->|info only| PASS["✓ PASS\n(no action needed)"]

    SEV -->|warning| CONF_W{Confidence?}
    CONF_W -->|"< 0.7"| CONCERNS_LOG["⚠ CONCERNS\n(logged, doesn't block)"]
    CONF_W -->|"≥ 0.7"| CONCERNS_REF["⚠ CONCERNS\n(logged, Refinery decides)"]

    SEV -->|error| CONF_E{Confidence?}
    CONF_E -->|"< 0.8"| CONCERNS_REF
    CONF_E -->|"≥ 0.8"| BLOCK["✗ BLOCK\n(quarantine for review)"]

    PASS --> GATES[Run Gates]
    CONCERNS_LOG --> GATES
    CONCERNS_REF --> REF_DECIDE{Refinery\nOverride?}
    REF_DECIDE -->|Proceed| GATES
    REF_DECIDE -->|Uphold| BLOCK

    BLOCK --> QUARANTINE[MR Quarantined]

    style PASS fill:#a6e3a1,stroke:#40a02b
    style CONCERNS_LOG fill:#f9e2af,stroke:#f5c211
    style CONCERNS_REF fill:#fab387,stroke:#fe640b
    style BLOCK fill:#f38ba8,stroke:#d62d4a
    style QUARANTINE fill:#f38ba8,stroke:#d62d4a
```

---

## 4. Advisory vs Blocking Mode Lifecycle

The Critic starts advisory and graduates to blocking after calibration.

```mermaid
stateDiagram-v2
    [*] --> Advisory: Deploy Critic

    Advisory: Advisory Mode
    Advisory: All verdicts logged
    Advisory: Nothing blocks
    Advisory: Collecting calibration data

    Calibration: Calibration Check
    Calibration: After N=50 MRs
    Calibration: Evaluate false positive rate

    Blocking: Blocking Mode
    Blocking: BLOCK verdicts quarantine MRs
    Blocking: PASS/CONCERNS proceed
    Blocking: Refinery can override

    Fallback: Revert to Advisory
    Fallback: FP rate > 10%
    Fallback: Thresholds need adjustment

    Advisory --> Calibration: 50 MRs processed
    Calibration --> Blocking: FP rate ≤ 10%
    Calibration --> Advisory: FP rate > 10%\n(adjust thresholds)
    Blocking --> Fallback: FP rate exceeds 10%
    Fallback --> Advisory: Reset calibration counter
```

---

## 5. Full MR Lifecycle with Critic Lens

Complete flow from polecat submission through merge or rejection.

```mermaid
sequenceDiagram
    participant P as Polecat
    participant R as Refinery
    participant C as Critic
    participant G as Gates
    participant D as Dolt

    P->>R: gt done (submit MR)
    R->>R: Queue MR in batch

    R->>C: Request review (diff + bead + reflection)
    C->>C: Evaluate against checklist
    C->>D: Create review bead (audit trail)
    C->>R: Return verdict {PASS|CONCERNS|BLOCK}

    alt BLOCK (confidence ≥ 0.8)
        R->>D: Quarantine MR
        R->>P: Notify: MR blocked (findings attached)
    else PASS or CONCERNS
        R->>G: Run gates (build/test/lint)
        alt Gates pass
            G->>R: All green
            R->>R: Merge to main
            R->>D: Close bead
        else Gates fail
            G->>R: Failure
            R->>R: Bisect batch
            R->>R: Merge good MRs, reject culprit
        end
    end
```

---

## 6. Critic Independence Architecture

Why the Critic must be independent from the implementing polecat.

```mermaid
flowchart LR
    subgraph implement["Implementation"]
        P[Polecat] --> CODE[Code Changes]
        P --> SELF[Self-Review\nStep 4]
        P -.->|"Same model,\nsame persona"| SELF
    end

    subgraph critic["Critic Review"]
        CR[Critic] --> REVIEW[Adversarial Review]
        CR -.->|"Different prompt,\nideally different model\nor temperature"| REVIEW
    end

    CODE --> CR
    SELF -->|reflection| CR

    REVIEW --> V{Verdict}
    V --> PASS["✓ PASS"]
    V --> CONCERNS["⚠ CONCERNS"]
    V --> BLOCK["✗ BLOCK"]

    style implement fill:#cdd6f4,stroke:#7287bd
    style critic fill:#f2cdcd,stroke:#d62d4a
```

---

## 7. Cross-Layer Interaction: Critic ↔ Reactive Cells

When a Critic BLOCKs an MR, downstream reactive cells are affected.

```mermaid
flowchart TD
    MR[MR with cell\nevaluator code] --> CRITIC[Critic Review]
    CRITIC -->|BLOCK| STALE[Mark downstream\ncells stale]

    STALE --> C1["Cell A\n(depends on blocked code)"]
    STALE --> C2["Cell B\n(depends on Cell A)"]

    C1 -->|dirty=true| C1_STATE["Cached value\nuntrustworthy"]
    C2 -->|dirty=true| C2_STATE["Cached value\nuntrustworthy"]

    CRITIC -->|PASS| MERGE[Merge to main]
    MERGE --> EVAL["Cells re-evaluate\non next demand"]

    style STALE fill:#f38ba8,stroke:#d62d4a
    style C1_STATE fill:#fab387,stroke:#fe640b
    style C2_STATE fill:#fab387,stroke:#fe640b
    style MERGE fill:#a6e3a1,stroke:#40a02b
```
