package subzero

import (
	"context"
	"strings"
	"testing"

	"github.com/steveyegge/gastown/internal/cell/parser"
)

// mockHumanIO returns a canned response for testing.
type mockHumanIO struct {
	response string
}

func (m *mockHumanIO) Prompt(ctx context.Context, prompt string, format *parser.FormatSpec) (string, error) {
	return m.response, nil
}

func TestHumanExecutorBasic(t *testing.T) {
	exec := &HumanExecutor{IO: &mockHumanIO{response: "yes"}}
	result, err := exec.Execute(context.Background(), &CellExec{
		Name:    "approve",
		Type:    "human",
		Prompts: []PromptMsg{{Role: "user", Content: "Deploy to prod?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "yes" {
		t.Errorf("got %q, want %q", result.Output, "yes")
	}
}

func TestHumanExecutorJSONResponse(t *testing.T) {
	exec := &HumanExecutor{IO: &mockHumanIO{response: `{"approved": true, "reason": "looks good"}`}}
	result, err := exec.Execute(context.Background(), &CellExec{
		Name:    "approve",
		Type:    "human",
		Prompts: []PromptMsg{{Role: "user", Content: "Approve?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Fields["approved"] != "true" {
		t.Errorf("expected approved=true, got %v", result.Fields["approved"])
	}
	if result.Fields["reason"] != "looks good" {
		t.Errorf("expected reason='looks good', got %v", result.Fields["reason"])
	}
}

func TestHumanExecutorNoIO(t *testing.T) {
	exec := &HumanExecutor{IO: nil}
	_, err := exec.Execute(context.Background(), &CellExec{
		Name:    "approve",
		Type:    "human",
		Prompts: []PromptMsg{{Role: "user", Content: "Approve?"}},
	})
	if err == nil {
		t.Fatal("expected error with nil IO")
	}
	if !strings.Contains(err.Error(), "no HumanIO configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHumanExecutorNoPrompt(t *testing.T) {
	exec := &HumanExecutor{IO: &mockHumanIO{response: "yes"}}
	_, err := exec.Execute(context.Background(), &CellExec{
		Name: "approve",
		Type: "human",
	})
	if err == nil {
		t.Fatal("expected error with no prompt")
	}
	if !strings.Contains(err.Error(), "no user> prompt") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatchExecutorRoutesHuman(t *testing.T) {
	dispatch := &DispatchExecutor{
		Human: &HumanExecutor{IO: &mockHumanIO{response: "approved"}},
	}
	result, err := dispatch.Execute(context.Background(), &CellExec{
		Name:    "gate",
		Type:    "human",
		Prompts: []PromptMsg{{Role: "user", Content: "Approve?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output != "approved" {
		t.Errorf("got %q, want %q", result.Output, "approved")
	}
}

func TestDispatchExecutorBlocksHumanWithoutExecutor(t *testing.T) {
	dispatch := &DispatchExecutor{}
	_, err := dispatch.Execute(context.Background(), &CellExec{
		Name: "gate",
		Type: "human",
	})
	if err == nil {
		t.Fatal("expected error without human executor")
	}
	if !strings.Contains(err.Error(), "BLOCKED") {
		t.Errorf("unexpected error: %v", err)
	}
}
