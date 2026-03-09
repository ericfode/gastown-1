# Cell Parser — Implementation Plan

**Poured from**: idea-to-plan.cell (generate-plan cell)
**Date**: 2026-03-09
**PRD**: docs/plans/2026-03-09-cell-parser-prd.md

---

## Phase 1: Corpus Validation (Pour cell-parse by hand)

**Goal**: Execute cell-parse against every .cell file manually. Find grammar bugs.

### Steps

1. **Pour cell-parse.lex against hello.cell** (simplest file)
   - Hand-execute the lexer: tokenize hello.cell
   - Check: does every token fit the enumerated types?
   - Record: token count, any ambiguities

2. **Pour cell-parse.lex against cell-parse.cell** (metacircular test)
   - The parser must parse itself
   - This exercises: mol() invocations, decision cells, format blocks, oracle blocks

3. **Pour cell-parse.lex against deacon-patrol.cell** (hardest file)
   - 26 cells, parallel groups, all cell types
   - Exercises: every grammar production

4. **Pour cell-parse.parse against the token streams**
   - Build AST from each tokenization
   - Check: does every AST node fit the spec grammar?
   - Record: which productions are used, which are untested

5. **Document grammar bugs** — file as beads
   - Every ambiguity or gap found during pouring

### Deliverables
- Lex results for 3 representative .cell files
- Parse trees for 3 representative .cell files
- Bug reports as beads for any grammar issues found

### Distillation Gate
- Lex cell: token type distribution stabilizes across files
- If lex produces consistent tokenizations, it's a distillation candidate

---

## Phase 2: Batch Pouring (all 34 files)

**Goal**: Pour cell-parse against every .cell file. Track consistency.

### Steps

1. **Create test harness script** — loops over docs/examples/*.cell
   - For each: record input hash, token count, cell count, valid/invalid
   - Output: JSON summary of all parses

2. **Pour in batches**
   - Batch 1: T1 files (simple — shiny, rule-of-five, security-audit, etc.)
   - Batch 2: T2 files (convoy — code-review, design, prd-review, etc.)
   - Batch 3: T3 files (molecule — dog-*, gastown-boot, session-gc, etc.)
   - Batch 4: T4-T5 files (complex — boot-triage, deacon-patrol, cell-zero, etc.)

3. **Track distillation candidates**
   - Which lex patterns repeat identically?
   - Which parse rules produce identical AST shapes?
   - Which validate checks always pass/always fail?

4. **Fix grammar spec for any bugs found**
   - Patch docs/plans/2026-03-08-cell-language-spec.md
   - Update cell-parse.cell oracle blocks if needed

### Deliverables
- Parse results for all 34 files
- Distillation candidate list (cells with >95% consistency)
- Updated grammar spec if bugs found

### Distillation Gate
- Lex cell: >90% of token patterns repeat across files → candidate
- Parse cell: >80% of AST shapes are predictable → candidate
- Validate cell: all 34 files pass or fail for documented reasons

---

## Phase 3: First Distillation (Lex)

**Goal**: Distill the lex cell into deterministic token rules.

### Steps

1. **Run Cell Zero against cell-parse**
   - Cell Zero observes cell-parse execution history
   - Analyzes the lex cell's consistency
   - Proposes a distill> block for lex

2. **Build the lex distill> block**
   - Map regex patterns to token types
   - Example: `"^--.*$" -> COMMENT`
   - Example: `"^##\s+\w+" -> MOL_OPEN`
   - Example: `"^#\s+\w+\s*:\s*\w+" -> CELL_OPEN`
   - Cover all token types in the enum
   - Fallback: llm (for edge cases)

3. **Validate distilled lex**
   - Re-pour with distilled lex against all 34 files
   - Compare token streams: distilled vs LLM
   - Match rate must be >95%

4. **Freeze lex**
   - Change lex cell type from llm to distilled
   - Write cell-parse-v1.cell with frozen lex

### Deliverables
- Distilled lex cell with token regex rules
- cell-parse-v1.cell (lex frozen, rest still LLM)
- Match rate report: distilled vs LLM lex

### Distillation Gate
- Distilled lex matches LLM lex on >95% of tokens across all 34 files
- Zero false positives (distilled never produces wrong token type)

---

## Phase 4: Parser Distillation

**Goal**: Distill the parse cell into deterministic grammar rules.

### Steps

1. **Analyze parse patterns** — which grammar productions are deterministic?
   - Molecule structure: always `MOL_OPEN cells* MOL_CLOSE` → deterministic
   - Cell structure: `CELL_OPEN deps* body CELL_CLOSE` → deterministic
   - Prompt sections: `SECTION_TAG content*` → deterministic
   - Oracle/script blocks: `CODE_OPEN body CODE_CLOSE` → deterministic
   - Format blocks: more complex, may need LLM longer

2. **Build parse distill> block**
   - Token sequence patterns → AST node types
   - Hierarchical: molecule → cell → body elements
   - Recursive: format_type can nest

3. **Validate and freeze**
   - Same protocol as Phase 3

### Deliverables
- Distilled parse cell
- cell-parse-v2.cell (lex + parse frozen)
- AST comparison report

---

## Phase 5: Full Parser (validate + emit distill)

**Goal**: Distill remaining cells. The parser is now fully deterministic.

### Steps

1. Distill validate cell (semantic checks are rule-based)
2. Distill emit cell (canonical formatting is deterministic)
3. Fingerprint is already a script cell (no distillation needed)
4. Report cell may stay LLM (summaries vary — acceptable)

### Deliverables
- cell-parse-final.cell (fully distilled except report)
- This IS the Cell parser

---

## Tracking

All work tracked as beads. Parser bugs → bd create with deps discovered-from parent.
Each phase requires distillation gate pass before advancing to next phase.
