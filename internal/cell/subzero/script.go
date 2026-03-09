package subzero

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// dangerousPatterns are shell commands that could cause fork-bombing
// or uncontrolled agent spawning.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bgt\s+sling\b`),
	regexp.MustCompile(`\bgt\s+mol\b`),
	regexp.MustCompile(`\bbd\s+create\b`),
	regexp.MustCompile(`\bgt\s+nudge\b`),
	regexp.MustCompile(`\bgt\s+mail\b`),
	regexp.MustCompile(`\bgt\s+escalate\b`),
}

// ScriptExecutor runs script cells as local bash subprocesses.
type ScriptExecutor struct {
	TimeoutSec int // default 30
}

func (s *ScriptExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	if cell.Type != "script" {
		return nil, fmt.Errorf("ScriptExecutor only handles script cells, got %q", cell.Type)
	}

	script := cell.Script

	// Resolve refs in script BEFORE safety check — inputs/params could
	// contain dangerous commands that bypass pattern matching on the raw template.
	for k, v := range cell.Params {
		script = strings.ReplaceAll(script, "{{param."+k+"}}", v)
	}
	for k, v := range cell.Inputs {
		script = strings.ReplaceAll(script, "{{"+k+"}}", v)
	}

	// Safety: reject dangerous commands AFTER substitution
	for _, pat := range dangerousPatterns {
		if pat.MatchString(script) {
			return nil, fmt.Errorf("BLOCKED: script contains dangerous pattern %q — Cell Sub-Zero does not allow agent spawning", pat.String())
		}
	}

	timeout := time.Duration(s.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("script %q timed out after %v", cell.Name, timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("script %q failed: %w\noutput: %s", cell.Name, err, string(out))
	}

	output := string(out)
	return &CellResult{Output: output, Fields: parseJSONFields(output)}, nil
}
