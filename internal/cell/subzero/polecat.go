package subzero

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PolecatExecutor delegates mol() cells to polecats via gt sling.
// Safety: max_depth=1 — a polecat spawned by Sub-Zero cannot spawn another polecat.
// This executor is ONLY used for mol() cell types.
type PolecatExecutor struct {
	// Rig is the target rig for polecat dispatch. Defaults to "gastown".
	Rig string
	// TimeoutMin is the max time to wait for a polecat. Defaults to 5 minutes.
	TimeoutMin int
	// Depth tracks recursion depth. If > 0, mol() cells are BLOCKED.
	Depth int
}

func (p *PolecatExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	if cell.Type != "mol" {
		return nil, fmt.Errorf("PolecatExecutor only handles mol cells, got %q", cell.Type)
	}

	// Hard safety: no recursive polecat spawning
	if p.Depth > 0 {
		return nil, fmt.Errorf("BLOCKED: polecat recursion depth %d > 0 — Sub-Zero prevents recursive agent spawning", p.Depth)
	}

	rig := p.Rig
	if rig == "" {
		rig = "gastown"
	}

	timeout := time.Duration(p.TimeoutMin) * time.Minute
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	// Assemble the task description from prompts
	var taskParts []string
	for _, pm := range cell.Prompts {
		taskParts = append(taskParts, pm.Content)
	}
	task := strings.Join(taskParts, "\n")
	if task == "" {
		task = fmt.Sprintf("Execute cell %q", cell.Name)
	}

	// Create a bead for tracking
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use bd create + gt sling
	createCmd := exec.CommandContext(ctx, "bd", "create",
		fmt.Sprintf("SubZero polecat: %s", cell.Name),
		"--description", task,
		"-t", "task", "-p", "2", "--json")
	createOut, err := createCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd create failed: %w\noutput: %s", err, string(createOut))
	}

	// Extract bead ID from JSON output
	beadID := extractBeadID(string(createOut))
	if beadID == "" {
		return nil, fmt.Errorf("could not extract bead ID from: %s", string(createOut))
	}

	// Sling to polecat with depth marker
	slingTask := fmt.Sprintf("%s\n\n--- CELL SUB-ZERO DEPTH=1 --- Do NOT spawn additional polecats or use gt sling.", task)
	slingCmd := exec.CommandContext(ctx, "gt", "sling", beadID, rig,
		"-m", slingTask, "--no-merge")
	slingOut, err := slingCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gt sling failed: %w\noutput: %s", err, string(slingOut))
	}

	// Poll for completion
	return p.waitForCompletion(ctx, beadID)
}

func (p *PolecatExecutor) waitForCompletion(ctx context.Context, beadID string) (*CellResult, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("polecat timed out waiting for %s", beadID)
		case <-ticker.C:
			cmd := exec.CommandContext(ctx, "bd", "show", beadID, "--json")
			out, err := cmd.CombinedOutput()
			if err != nil {
				continue // retry
			}
			if strings.Contains(string(out), `"status": "closed"`) {
				return &CellResult{Output: string(out)}, nil
			}
		}
	}
}

// extractBeadID extracts the bead ID from bd create JSON output.
func extractBeadID(jsonOut string) string {
	// Simple extraction — look for "id": "gt-xxx"
	idx := strings.Index(jsonOut, `"id"`)
	if idx < 0 {
		return ""
	}
	rest := jsonOut[idx:]
	start := strings.Index(rest, `"gt-`)
	if start < 0 {
		return ""
	}
	end := strings.Index(rest[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return rest[start+1 : start+1+end]
}
