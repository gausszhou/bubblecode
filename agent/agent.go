package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"
)

type sessionState struct {
	conversation *Conversation
	executor     *ToolExecutor
	cwd          string
	model        string
	toolCounter  int
}

type LLMAgent struct {
	conn     *acp.AgentSideConnection
	cfg      *Config
	logger   *slog.Logger
	sessions map[string]*sessionState
	mu       sync.Mutex
}

func NewLLMAgent(cfg *Config, logger *slog.Logger) *LLMAgent {
	return &LLMAgent{
		cfg:      cfg,
		logger:   logger,
		sessions: make(map[string]*sessionState),
	}
}

func (a *LLMAgent) SetAgentConnection(conn *acp.AgentSideConnection) {
	a.conn = conn
}

func (a *LLMAgent) Authenticate(_ context.Context, _ acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *LLMAgent) Initialize(_ context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{
			PromptCapabilities:  acp.PromptCapabilities{},
			SessionCapabilities: acp.SessionCapabilities{},
		},
	}, nil
}

func (a *LLMAgent) NewSession(_ context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	sid := string(generateID())
	cwd := params.Cwd
	if cwd == "" {
		cwd = "."
	}

	systemPrompt := fmt.Sprintf(`You are Bubblecode, an AI coding assistant integrated into a terminal user interface.
You help with software engineering tasks by reading and writing files, running shell commands, and searching code.

When you need to work with files or run commands, use the available tools.
Always explain what you're doing before making changes.
Be concise and clear in your responses.

Working directory: %s`, cwd)

	ss := &sessionState{
		conversation: NewConversation(systemPrompt),
		executor:     NewToolExecutor(cwd),
		cwd:          cwd,
		model:        a.cfg.DefaultModel,
	}

	a.mu.Lock()
	a.sessions[sid] = ss
	a.mu.Unlock()

	a.logger.Info("session created", "session_id", sid)
	return acp.NewSessionResponse{SessionId: acp.SessionId(sid)}, nil
}

func (a *LLMAgent) Cancel(_ context.Context, params acp.CancelNotification) error {
	a.mu.Lock()
	delete(a.sessions, string(params.SessionId))
	a.mu.Unlock()
	a.logger.Info("session cancelled", "session_id", params.SessionId)
	return nil
}

func (a *LLMAgent) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	sid := string(params.SessionId)

	a.mu.Lock()
	ss, ok := a.sessions[sid]
	a.mu.Unlock()
	if !ok {
		return acp.PromptResponse{}, fmt.Errorf("session not found: %s", sid)
	}

	promptText := extractText(params.Prompt)
	if promptText == "" {
		return acp.PromptResponse{}, nil
	}

	a.logger.Info("prompt received", "session_id", sid, "text_length", len(promptText))

	p := a.cfg.GetDefaultProvider()
	if p == nil || p.APIKey == "" {
		errMsg := "API key not configured. Set BUBBLECODE_API_KEY env var or create ~/.config/bubblecode/config.json"
		a.sendText(ctx, params.SessionId, errMsg)
		return acp.PromptResponse{}, nil
	}

	ss.conversation.AddUserMessage(promptText)

	if isModelCommand(promptText) {
		modelName := extractModelName(promptText)
		if modelName != "" {
			ss.model = modelName
			a.logger.Info("model changed via command", "session_id", sid, "model", modelName)
			if err := a.sendText(ctx, params.SessionId, fmt.Sprintf("Model changed to: %s", modelName)); err != nil {
				a.logger.Error("send model change text failed", "error", err)
			}
		}
		return acp.PromptResponse{}, nil
	}

	if isProviderCommand(promptText) {
		providerName := extractProviderName(promptText)
		if providerName != "" {
			configPath, cfgErr := ConfigPath()
			if cfgErr != nil {
				a.logger.Error("get config path", "error", cfgErr)
			} else {
				if a.cfg.SwitchProvider(providerName) {
					if saveErr := SaveConfig(configPath, a.cfg); saveErr == nil {
						a.logger.Info("provider switched", "provider", providerName, "model", a.cfg.DefaultModel)
						if err := a.sendText(ctx, params.SessionId, fmt.Sprintf("Switched to provider: %s (model: %s)", providerName, a.cfg.DefaultModel)); err != nil {
							a.logger.Error("send provider switch text failed", "error", err)
						}
					}
				} else {
					if err := a.sendText(ctx, params.SessionId, fmt.Sprintf("Provider '%s' not found in config", providerName)); err != nil {
						a.logger.Error("send provider not found text failed", "error", err)
					}
				}
			}
		}
		return acp.PromptResponse{}, nil
	}

	p = a.cfg.GetDefaultProvider()
	llm := NewLLMClient(p.APIBase, p.APIKey, ss.model, p.MaxTokensVal(ss.model, a.cfg.MaxTokens), 0)
	tools := DefaultTools()

	for {
		select {
		case <-ctx.Done():
			return acp.PromptResponse{}, nil
		default:
		}

		var streamText strings.Builder
		var toolCalls []ToolCall

		err := llm.ChatStream(ctx, ss.conversation, tools, func(ev StreamEvent) {
			switch ev.Type {
			case "text":
				streamText.WriteString(ev.Text)
				if err := a.sendText(ctx, params.SessionId, ev.Text); err != nil {
					a.logger.Error("send text chunk failed", "error", err)
				}
			case "tool_call":
				toolCalls = mergeToolCalls(toolCalls, ev.ToolCalls)
			}
		})
		if err != nil {
			a.logger.Error("llm call failed", "error", err)
			return acp.PromptResponse{}, fmt.Errorf("llm error: %w", err)
		}

		finalText := streamText.String()
		if finalText != "" {
			ss.conversation.AddAssistantMessage(finalText, nil)
		}

		if len(toolCalls) == 0 {
			break
		}

		ss.conversation.AddAssistantMessage("", toolCalls)

		for _, tc := range toolCalls {
			select {
			case <-ctx.Done():
				return acp.PromptResponse{}, nil
			default:
			}

			if tc.ID == "" || tc.Function.Name == "" {
				continue
			}

			ss.toolCounter++
			callID := acp.ToolCallId(fmt.Sprintf("call-%d", ss.toolCounter))

			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				a.logger.Error("parse tool args", "error", err)
				args = map[string]any{}
			}

			kind := toolKindFor(tc.Function.Name)

			if err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
				SessionId: params.SessionId,
				Update: acp.StartToolCall(
					callID,
					tc.Function.Name,
					acp.WithStartKind(kind),
					acp.WithStartRawInput(args),
				),
			}); err != nil {
				a.logger.Error("send tool start failed", "error", err)
			}

			start := time.Now()
			result := ss.executor.Execute(ctx, tc.Function.Name, args)
			elapsed := time.Since(start)

			output := result.Output
			if result.Error != "" {
				output = result.Error
			}

			status := acp.ToolCallStatusCompleted
			if result.Error != "" {
				status = acp.ToolCallStatusFailed
			}

			if err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
				SessionId: params.SessionId,
				Update: acp.UpdateToolCall(
					callID,
					acp.WithUpdateTitle(tc.Function.Name),
					acp.WithUpdateKind(kind),
					acp.WithUpdateStatus(status),
					acp.WithUpdateRawOutput(output),
				),
			}); err != nil {
				a.logger.Error("send tool update failed", "error", err)
			}

			a.logger.Info("tool executed",
				"tool", tc.Function.Name,
				"elapsed_ms", elapsed.Milliseconds(),
				"status", status,
			)

			ss.conversation.AddToolResult(tc.ID, tc.Function.Name, output)
		}
	}

	a.logger.Info("prompt complete", "session_id", sid)
	return acp.PromptResponse{}, nil
}

func (a *LLMAgent) sendText(ctx context.Context, sid acp.SessionId, text string) error {
	return a.conn.SessionUpdate(ctx, acp.SessionNotification{
		SessionId: sid,
		Update:    acp.UpdateAgentMessageText(text),
	})
}

func (a *LLMAgent) ListSessions(_ context.Context, _ acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

func (a *LLMAgent) SetSessionConfigOption(_ context.Context, _ acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (a *LLMAgent) SetSessionMode(_ context.Context, _ acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

func isModelCommand(text string) bool {
	return len(text) > 7 && text[:7] == "/model "
}

func extractModelName(text string) string {
	return text[7:]
}

func isProviderCommand(text string) bool {
	return len(text) > 10 && text[:10] == "/provider "
}

func extractProviderName(text string) string {
	return text[10:]
}

func extractText(blocks []acp.ContentBlock) string {
	for _, b := range blocks {
		if b.Text != nil {
			return b.Text.Text
		}
	}
	return ""
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func toolKindFor(name string) acp.ToolKind {
	switch name {
	case "read_file":
		return acp.ToolKindRead
	case "write_file":
		return acp.ToolKindEdit
	case "bash":
		return acp.ToolKindExecute
	case "glob":
		return acp.ToolKindSearch
	default:
		return acp.ToolKindExecute
	}
}
