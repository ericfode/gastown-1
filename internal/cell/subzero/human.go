package subzero

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// HumanIO is the interface for collecting human input.
// Implementations: CLIHumanIO (stdin), or custom (webhook, Slack, queue).
type HumanIO interface {
	Prompt(ctx context.Context, prompt string, format *parser.FormatSpec) (string, error)
}

// HumanExecutor handles human cell execution by presenting prompts
// and collecting input via a pluggable HumanIO interface.
type HumanExecutor struct {
	IO HumanIO
}

func (h *HumanExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	if h.IO == nil {
		return nil, fmt.Errorf("no HumanIO configured for human cell %q", cell.Name)
	}

	// Assemble the prompt from user> sections
	var prompt string
	for _, p := range cell.Prompts {
		if p.Role == "user" {
			prompt += p.Content
		}
	}
	if prompt == "" {
		return nil, fmt.Errorf("human cell %q has no user> prompt", cell.Name)
	}

	// Extract format spec if present (stored in FormatJSON as hint)
	// The actual FormatSpec validation happens at parse time; here we just
	// pass the prompt through to the HumanIO interface.
	response, err := h.IO.Prompt(ctx, strings.TrimSpace(prompt), nil)
	if err != nil {
		return nil, fmt.Errorf("human cell %q: %w", cell.Name, err)
	}

	result := &CellResult{
		Output: response,
		Fields: parseJSONFields(response),
	}
	return result, nil
}

// CLIHumanIO prompts on a writer and reads from a reader (defaults to stdout/stdin).
type CLIHumanIO struct {
	In  io.Reader
	Out io.Writer
}

func (c *CLIHumanIO) Prompt(ctx context.Context, prompt string, format *parser.FormatSpec) (string, error) {
	out := c.Out
	if out == nil {
		out = os.Stdout
	}
	in := c.In
	if in == nil {
		in = os.Stdin
	}

	fmt.Fprintf(out, "\n--- HUMAN INPUT REQUIRED ---\n%s\n> ", prompt)

	scanner := bufio.NewScanner(in)
	// Collect lines until empty line (double-enter) for multi-line, or single line for simple input
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && len(lines) > 0 {
			break
		}
		lines = append(lines, line)
		if format == nil {
			// No format spec = single line input
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading human input: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}
