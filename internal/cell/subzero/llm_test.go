package subzero

import (
	"context"
	"os"
	"testing"
)

func TestLLMExecutorRequiresAPIKey(t *testing.T) {
	// Temporarily clear the API key
	saved := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", saved)

	exec := &LLMExecutor{}
	_, err := exec.Execute(context.Background(), &CellExec{
		Name: "test", Type: "llm",
		Prompts: []PromptMsg{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error without API key")
	}
}

func TestLLMExecutorLive(t *testing.T) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	exec := &LLMExecutor{Model: "claude-haiku-4-5-20251001", MaxTokens: 100}
	result, err := exec.Execute(context.Background(), &CellExec{
		Name: "test", Type: "llm",
		Prompts: []PromptMsg{
			{Role: "system", Content: "You are a test. Respond with exactly: {\"ok\": true}"},
			{Role: "user", Content: "Respond now."},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output == "" {
		t.Fatal("expected non-empty output")
	}
	t.Logf("LLM output: %s", result.Output)
}
