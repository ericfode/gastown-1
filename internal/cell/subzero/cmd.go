package subzero

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// RunFile parses and executes a .cell file, printing results.
// mode: "mock" (echo prompts), "llm" (real Claude API calls)
func RunFile(path string, params map[string]string, mode string, maxCells int) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	prog, err := parser.Parse(string(src))
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if len(prog.Molecules) == 0 {
		return fmt.Errorf("no molecules in %s", path)
	}

	var executor Executor
	switch mode {
	case "mock":
		executor = &DispatchExecutor{
			LLM:    &MockExecutor{},
			Script: &ScriptExecutor{TimeoutSec: 30},
		}
	case "llm":
		executor = &DispatchExecutor{
			LLM:    &LLMExecutor{},
			Script: &ScriptExecutor{TimeoutSec: 30},
		}
	case "full":
		// Full mode: LLM + script + polecat (bounded depth=0)
		executor = &DispatchExecutor{
			LLM:     &LLMExecutor{},
			Script:  &ScriptExecutor{TimeoutSec: 30},
			Polecat: &PolecatExecutor{Depth: 0},
		}
	default:
		return fmt.Errorf("unknown mode %q (use 'mock', 'llm', or 'full')", mode)
	}

	if maxCells <= 0 {
		maxCells = 100
	}

	runner := &Runner{
		Executor: executor,
		Params:   params,
		MaxCells: maxCells,
	}

	mol := prog.Molecules[0]
	fmt.Fprintf(os.Stderr, "=== Cell Sub-Zero: %s (molecule: %s, mode: %s) ===\n", path, mol.Name, mode)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	results, err := runner.Run(ctx, mol)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		// Still print any partial results
	}

	fmt.Fprintf(os.Stderr, "\n=== Results (%d cells, %v) ===\n", len(results), elapsed.Round(time.Millisecond))
	for name, res := range results {
		fmt.Fprintf(os.Stderr, "\n--- %s ---\n", name)
		output := res.Output
		if len(output) > 500 {
			output = output[:500] + "..."
		}
		fmt.Println(output)
	}

	return err
}

// ParseParams parses "key=value,key=value" into a map.
func ParseParams(s string) map[string]string {
	params := make(map[string]string)
	if s == "" {
		return params
	}
	for _, pair := range strings.Split(s, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			params[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return params
}
