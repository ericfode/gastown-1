# Layer 2: Learning Pipeline — Mermaid Diagrams

**Source**: S3 Architecture Sketch, Sections 3.1 (Reflection Cycles) and 3.2 (Skill Crystals)

---

## 1. Reflection Generation Flow

How a polecat generates a structured reflection after completing work.

```mermaid
flowchart TD
    A[Polecat completes implementation] --> B{Trivial task?<br/>< 2 commits, no blockers}
    B -- Yes --> SKIP[Skip reflection step]
    B -- No --> C[Review commits:<br/>git log origin/main..HEAD]
    C --> D[Review original bead:<br/>bd show issue-id]
    D --> E[Generate structured reflection]

    E --> F["Reflection JSON:
    • what_worked
    • what_failed
    • would_do_differently
    • patterns_discovered
    • difficulty (trivial→novel)
    • blockers_hit"]

    F --> G["Persist: bd update issue-id --reflection JSON"]
    G --> H[(Dolt: beads table<br/>reflection JSON column<br/>difficulty VARCHAR column)]

    SKIP --> I[Proceed to submit step]
    H --> I

    style H fill:#2d5a27,stroke:#4a8,color:#fff
    style SKIP fill:#555,stroke:#888,color:#fff
    style F fill:#1a3a5c,stroke:#4a8,color:#fff
```

---

## 2. Reflection Retrieval During `gt prime`

How past reflections are surfaced to a new polecat at session start.

```mermaid
flowchart TD
    A[Polecat starts: gt prime] --> B[Read current bead title + description]
    B --> C[Extract keywords from bead]
    C --> D[SQL query against beads table]

    D --> E["SELECT reflection FROM beads
    WHERE reflection IS NOT NULL
      AND status = 'closed'
      AND difficulty IN ('challenging','novel')
      AND (title LIKE '%keyword%'
           OR labels LIKE '%label%')
    ORDER BY updated_at DESC
    LIMIT 5"]

    E --> F{Results found?}
    F -- Yes --> G[Inject up to 5 reflections<br/>into polecat context]
    F -- No --> H[No reflections injected]

    G --> I[Polecat begins work<br/>with historical context]
    H --> I

    style E fill:#1a3a5c,stroke:#4a8,color:#fff
    style G fill:#2d5a27,stroke:#4a8,color:#fff
```

---

## 3. Reflection Consolidation

Periodic synthesis of raw reflections into higher-level meta-reflections.

```mermaid
flowchart TD
    A["Scheduled: bd reflection consolidate --since 7d"] --> B[Query raw reflections<br/>from past 7 days]
    B --> C[Group by patterns_discovered<br/>and category]
    C --> D[LLM synthesizes cross-reflection<br/>patterns into meta-reflection]
    D --> E[Create meta-reflection bead]

    E --> F[(Dolt: meta-reflection bead<br/>aggregated patterns<br/>higher retrieval priority)]

    G[Future gt prime retrieval] --> H{Meta-reflection<br/>available?}
    H -- Yes --> I[Prioritize meta-reflection<br/>over raw reflections]
    H -- No --> J[Fall back to raw reflections]

    style F fill:#2d5a27,stroke:#4a8,color:#fff
    style D fill:#1a3a5c,stroke:#4a8,color:#fff
```

---

## 4. Skill Crystal Extraction and Lifecycle

How crystals are born from completions, matched during prime, and garbage-collected.

```mermaid
flowchart TD
    subgraph Extraction
        A[Closed bead with reflection] --> B["bd crystal extract bead-id"]
        B --> C[Examine reflection +<br/>commit diff + description]
        C --> D[LLM identifies reusable pattern]
        D --> E[Create crystal record]
    end

    E --> F[("crystals table:
    • id, name
    • trigger_description
    • solution_template
    • provenance_bead
    • category
    • times_used
    • last_used")]

    subgraph Matching
        G[gt prime with new bead] --> H[Extract keywords from bead]
        H --> I[SQL match on trigger_description]
        I --> J[Rank by times_used]
        J --> K[Inject matched crystals<br/>into polecat context]
        K --> L[Increment times_used<br/>and update last_used]
    end

    subgraph Garbage Collection
        M[Crystal unused 90 days] --> N[Mark stale<br/>excluded from matching]
        N --> O{Used again<br/>within 90 days?}
        O -- Yes --> P[Remove stale flag]
        O -- No --> Q[180 days total staleness]
        Q --> R[Archive crystal<br/>Dolt history preserves it]
    end

    F --> G
    F --> M

    style F fill:#2d5a27,stroke:#4a8,color:#fff
    style E fill:#1a3a5c,stroke:#4a8,color:#fff
    style R fill:#555,stroke:#888,color:#fff
```

---

## 5. Full Learning Pipeline — End-to-End

The complete data flow from polecat completion through reflection, crystal extraction, consolidation, and retrieval.

```mermaid
flowchart LR
    subgraph Completion
        P1[Polecat completes work]
        P1 --> REF[Generate reflection]
        REF --> STORE["Store in bead<br/>(reflection JSON)"]
    end

    subgraph Learning
        STORE --> CONS[Consolidation<br/>weekly batch]
        CONS --> META[Meta-reflections]
        STORE --> XTAL["Crystal extraction<br/>(bd crystal extract)"]
        XTAL --> CTAB[("crystals table")]
    end

    subgraph Retrieval
        PRIME[New polecat: gt prime]
        PRIME --> RMATCH[Keyword match<br/>on reflections]
        PRIME --> CMATCH[Keyword match<br/>on crystals]
        META --> RMATCH
        STORE --> RMATCH
        CTAB --> CMATCH
        RMATCH --> CTX[Polecat context<br/>enriched with<br/>past learnings]
        CMATCH --> CTX
    end

    subgraph GC [Garbage Collection]
        CTAB --> STALE[90d unused → stale]
        STALE --> ARCHIVE[180d → archived]
    end

    style META fill:#2d5a27,stroke:#4a8,color:#fff
    style CTAB fill:#2d5a27,stroke:#4a8,color:#fff
    style CTX fill:#1a3a5c,stroke:#4a8,color:#fff
```

---

## 6. Schema Changes Summary

```mermaid
erDiagram
    BEADS {
        varchar id PK
        varchar title
        json reflection "NEW — structured reflection JSON"
        varchar difficulty "NEW — trivial|routine|challenging|novel"
        varchar status
        timestamp updated_at
        text labels
    }

    CRYSTALS {
        varchar id PK
        varchar name
        text trigger_description
        text solution_template
        varchar provenance_bead FK
        varchar category
        int times_used
        timestamp last_used
        timestamp created_at
    }

    BEADS ||--o{ CRYSTALS : "provenance_bead"
```

---

## 7. CLI Command Flow

```mermaid
flowchart TD
    subgraph "Reflection Commands"
        A1["bd update id --reflection JSON"] --> D1[(Dolt: beads.reflection)]
        A2["bd reflection consolidate --since 7d"] --> D2[(Dolt: meta-reflection bead)]
    end

    subgraph "Crystal Commands"
        B1["bd crystal extract bead-id"] --> D3[(Dolt: crystals table)]
        B2["bd crystal match bead-id"] --> D3
    end

    subgraph "Retrieval (automatic)"
        C1[gt prime] --> R1[Query reflections]
        C1 --> R2[Query crystals]
        R1 --> D1
        R2 --> D3
    end

    style D1 fill:#2d5a27,stroke:#4a8,color:#fff
    style D2 fill:#2d5a27,stroke:#4a8,color:#fff
    style D3 fill:#2d5a27,stroke:#4a8,color:#fff
```
