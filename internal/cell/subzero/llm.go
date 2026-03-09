package subzero

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// LLMExecutor calls the Claude API to execute llm cells.
type LLMExecutor struct {
	APIKey    string // defaults to ANTHROPIC_API_KEY env var
	Model     string // defaults to claude-sonnet-4-6
	MaxTokens int    // defaults to 4096
	client    *http.Client
}

// claudeRequest is the request body for the Claude Messages API.
type claudeRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []claudeMessage  `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (l *LLMExecutor) Execute(ctx context.Context, cell *CellExec) (*CellResult, error) {
	if cell.Type != "llm" && cell.Type != "decision" {
		return nil, fmt.Errorf("LLMExecutor only handles llm/decision cells, got %q", cell.Type)
	}

	apiKey := l.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set — cannot execute llm cells")
	}

	model := l.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	maxTokens := l.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Assemble messages from prompt sections
	var systemPrompt string
	var messages []claudeMessage
	var userParts []string

	for _, p := range cell.Prompts {
		switch p.Role {
		case "system":
			systemPrompt += p.Content
		case "context", "examples", "think":
			// Context, examples, think go into the user message
			userParts = append(userParts, p.Content)
		case "user":
			userParts = append(userParts, p.Content)
		}
	}

	if len(userParts) == 0 {
		userParts = append(userParts, "Execute this cell.")
	}

	messages = append(messages, claudeMessage{
		Role:    "user",
		Content: strings.Join(userParts, "\n"),
	})

	// Add format instruction if present
	if cell.FormatJSON != "" {
		messages[len(messages)-1].Content += "\n\nRespond with JSON matching this schema: " + cell.FormatJSON
	}

	req := claudeRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  messages,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if l.client == nil {
		l.client = &http.Client{Timeout: 120 * time.Second}
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := l.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if claudeResp.Error != nil {
		return nil, fmt.Errorf("API error: %s: %s", claudeResp.Error.Type, claudeResp.Error.Message)
	}

	// Extract text from response
	var output string
	for _, c := range claudeResp.Content {
		if c.Type == "text" {
			output += c.Text
		}
	}

	result := &CellResult{
		Output: output,
		Fields: parseJSONFields(output),
	}

	return result, nil
}
