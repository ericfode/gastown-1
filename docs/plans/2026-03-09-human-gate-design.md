# Human Gate Design (gt-63f)

## Problem

Cell has no way to pause execution and wait for human input. TOML formulas
handle this naturally because the executing agent pauses. Cell needs a formal
mechanism for both approval gates (boolean accept/reject) and interactive input
(free-form or structured data from a human).

## Design: `human` Cell Type

A new cell type `human` that suspends execution and presents a prompt to a human
operator. The human's response becomes the cell's output.

### Syntax

```cell
-- Approval gate (boolean input)
# approve-deploy : human
  - run-tests
  user>
    Tests passed: {{run-tests.summary}}
    Deploy to production?
  format> approval
    { approved: bool, reason: str }
#/

-- Free-form input (no format = raw text)
# get-feedback : human
  - draft-report
  user>
    Here's the draft: {{draft-report.output}}
    Any changes?
#/

-- Structured input
# get-target : human
  user>
    Which branch for this PR?
  format> target
    { branch: str }
#/
```

### Rules

- `human` cells use `user>` as the prompt shown to the human
- `format>` is optional: if present, validates human input; if absent, output is raw text
- Dependencies (`- ref`), oracle blocks, and wires work normally
- `human` cells MUST have `user>` (what else would you show?)
- `human` cells MUST NOT have `system>`, `context>`, `think>`, `examples>` (LLM-specific)

### Execution Model

1. `DispatchExecutor` routes `human` cells to `HumanExecutor`
2. `HumanExecutor` assembles the `user>` prompt with interpolated refs
3. Emits the prompt to the human via a pluggable `HumanIO` interface
4. Blocks until human responds
5. Validates response against `format>` if present
6. Cell completes with human input as output

### HumanIO Interface

```go
type HumanIO interface {
    Prompt(ctx context.Context, prompt string, format *parser.FormatSpec) (string, error)
}
```

Initial implementation: CLI stdin. Future: webhook, Slack, queue.

### Parser Changes

- Add `TokenHuman` token type (keyword `human` in section tags context — no, as cell type identifier)
- No new token needed — `human` is just an identifier parsed by `parseCellType()`
- Validation: `human` cells must have `user>`, must not have `system>`/`context>`/`think>`/`examples>`

### Validation Rules

New in `validate.go`:
- Error: `human` cell without `user>` section
- Error: `human` cell with `system>`, `context>`, `think>`, or `examples>` section

### Executor Changes

- New `HumanExecutor` in `internal/cell/subzero/human.go`
- `DispatchExecutor` gets `Human Executor` field, routes `human` type to it
- `CellExec.Type` string gains `"human"` as a valid value
- `MockExecutor` returns `"human:mock_approved"` for human cells (testing)
