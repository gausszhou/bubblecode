package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type LLMClient struct {
	apiBase    string
	apiKey     string
	model      string
	maxTokens  int
	temp       float64
	httpClient *http.Client
}

func NewLLMClient(cfg *Config) *LLMClient {
	base := strings.TrimRight(cfg.APIBase, "/")
	return &LLMClient{
		apiBase:    base,
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		maxTokens:  cfg.MaxTokens,
		temp:       cfg.Temperature,
		httpClient: &http.Client{},
	}
}

type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type deltaFunc func(StreamEvent)

type StreamEvent struct {
	Type      string
	Text      string
	ToolCalls []ToolCall
	Done      bool
	Err       error
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []ChatMessage    `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
}

type streamChoice struct {
	Index int             `json:"index"`
	Delta json.RawMessage `json:"delta"`
}

type streamResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Choices []streamChoice `json:"choices"`
}

type deltaContent struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

func (c *LLMClient) ChatStream(ctx context.Context, conversation *Conversation, tools []ToolDefinition, fn deltaFunc) error {
	body := chatRequest{
		Model:       c.model,
		Messages:    conversation.ToLLMMessages(),
		MaxTokens:   c.maxTokens,
		Temperature: c.temp,
		Stream:      true,
		Tools:       tools,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return parseSSE(ctx, resp.Body, fn)
}

func (c *LLMClient) Chat(ctx context.Context, conversation *Conversation, tools []ToolDefinition) (string, []ToolCall, error) {
	var text strings.Builder
	var toolCalls []ToolCall

	err := c.ChatStream(ctx, conversation, tools, func(ev StreamEvent) {
		if ev.Err != nil {
			return
		}
		text.WriteString(ev.Text)
		if len(ev.ToolCalls) > 0 {
			toolCalls = mergeToolCalls(toolCalls, ev.ToolCalls)
		}
	})
	if err != nil {
		return "", nil, err
	}

	return text.String(), toolCalls, nil
}

func parseSSE(ctx context.Context, r io.Reader, fn deltaFunc) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			fn(StreamEvent{Type: "done", Done: true})
			return nil
		}

		var sr streamResponse
		if err := json.Unmarshal([]byte(data), &sr); err != nil {
			continue
		}

		for _, ch := range sr.Choices {
			var dc deltaContent
			if err := json.Unmarshal(ch.Delta, &dc); err != nil {
				continue
			}

			if dc.Content != "" {
				fn(StreamEvent{Type: "text", Text: dc.Content})
			}

			for _, tc := range dc.ToolCalls {
				fn(StreamEvent{Type: "tool_call", ToolCalls: []ToolCall{tc}})
			}

			finishReason, _ := getFinishReason(ch.Delta)
			if finishReason == "tool_calls" {
				fn(StreamEvent{Type: "tool_choice", Done: true})
			}
		}
	}

	return scanner.Err()
}

func getFinishReason(delta json.RawMessage) (string, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(delta, &raw); err != nil {
		return "", false
	}
	fr, ok := raw["finish_reason"]
	if !ok {
		return "", false
	}
	var s string
	if err := json.Unmarshal(fr, &s); err != nil {
		return "", false
	}
	return s, true
}

func mergeToolCalls(existing, incoming []ToolCall) []ToolCall {
	for _, inc := range incoming {
		found := false
		for i, ex := range existing {
			if ex.Index == inc.Index {
				if inc.ID != "" {
					existing[i].ID = inc.ID
				}
				if inc.Function.Name != "" {
					existing[i].Function.Name = inc.Function.Name
				}
				existing[i].Function.Arguments += inc.Function.Arguments
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, inc)
		}
	}
	return existing
}
