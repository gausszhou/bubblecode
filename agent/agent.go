package agent

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/coder/acp-go-sdk"
)

//go:embed examples/markdown_example.md
var exampleMD string

//go:embed examples/go_best_practices.md
var exampleGo string

//go:embed examples/api_design_guide.md
var exampleAPI string

//go:embed examples/sql_optimization.md
var exampleSQL string

var examples = []string{
	exampleMD,
	exampleGo,
	exampleAPI,
	exampleSQL,
}

type mockAgent struct {
	conn     *acp.AgentSideConnection
	sessions map[string]context.CancelFunc
	mu       sync.Mutex
	logger   *slog.Logger
}

func NewMockAgent(logger *slog.Logger) *mockAgent {
	return &mockAgent{sessions: make(map[string]context.CancelFunc), logger: logger}
}

func (a *mockAgent) SetAgentConnection(conn *acp.AgentSideConnection) {
	a.conn = conn
}

func (a *mockAgent) Authenticate(ctx context.Context, params acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *mockAgent) Initialize(ctx context.Context, params acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{
			PromptCapabilities:  acp.PromptCapabilities{},
			SessionCapabilities: acp.SessionCapabilities{},
		},
	}, nil
}

func (a *mockAgent) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{SessionId: "mock-session"}, nil
}

func (a *mockAgent) Cancel(ctx context.Context, params acp.CancelNotification) error {
	a.mu.Lock()
	cancel, ok := a.sessions[string(params.SessionId)]
	a.mu.Unlock()
	if ok && cancel != nil {
		cancel()
	}
	return nil
}

func (a *mockAgent) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	sid := string(params.SessionId)
	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.sessions[sid] = cancel
	a.mu.Unlock()

	a.simulateTurn(ctx, params.SessionId, params.Prompt)

	a.mu.Lock()
	delete(a.sessions, sid)
	a.mu.Unlock()
	return acp.PromptResponse{}, nil
}

func RandomSplit(s string) []string {
	runes := []rune(s)
	if len(runes) == 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var parts []string
	for i := 0; i < len(runes); {
		maxLen := len(runes) - i
		if maxLen > 100 {
			maxLen = 100
		}
		l := rng.Intn(maxLen) + 1
		parts = append(parts, string(runes[i:i+l]))
		i += l
	}
	return parts
}

func (a *mockAgent) simulateTurn(ctx context.Context, sid acp.SessionId, prompt []acp.ContentBlock) {
	text := ""
	for _, block := range prompt {
		if block.Text != nil {
			text = block.Text.Text
		}
	}

	n := 10 + rand.Intn(11)
	steps := makeActions(n)
	callCounter := 1

	for _, s := range steps {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(time.Duration(100+rand.Intn(300)) * time.Millisecond)

		switch s.typ {
		case "think":
			for _, char := range RandomSplit(s.thought) {
				time.Sleep(time.Duration(10+rand.Intn(30)) * time.Millisecond)
				a.sendUpdate(ctx, sid, acp.SessionNotification{
					SessionId: sid,
					Update:    acp.UpdateAgentThoughtText(char),
				})
			}

		case "plan":
			a.sendUpdate(ctx, sid, acp.SessionNotification{
				SessionId: sid,
				Update: acp.SessionUpdate{
					Plan: &acp.SessionUpdatePlan{
						Entries: []acp.PlanEntry{
							{Content: "Analyze: " + text, Status: acp.PlanEntryStatusCompleted},
							{Content: "Explore codebase", Status: acp.PlanEntryStatusInProgress},
							{Content: "Make changes", Status: acp.PlanEntryStatusPending},
							{Content: "Verify results", Status: acp.PlanEntryStatusPending},
						},
					},
				},
			})

		case "tool":
			callId := acp.ToolCallId(fmt.Sprintf("call-%d", callCounter))
			callCounter++

			a.sendUpdate(ctx, sid, acp.SessionNotification{
				SessionId: sid,
				Update: acp.StartToolCall(
					callId,
					s.label,
					acp.WithStartKind(acp.ToolKind(s.kind)),
					acp.WithStartStatus(acp.ToolCallStatusPending),
					acp.WithStartRawInput(map[string]any{"tool": s.name}),
				),
			})

			time.Sleep(200 * time.Millisecond)

			a.sendUpdate(ctx, sid, acp.SessionNotification{
				SessionId: sid,
				Update: acp.UpdateToolCall(
					callId,
					acp.WithUpdateStatus(acp.ToolCallStatusCompleted),
					acp.WithUpdateRawOutput(mockToolOutput(s.name)),
				),
			})

		case "text":
			content := examples[rand.Intn(len(examples))]
			for len(content) < 100_000 {
				remaining := 100_000 - len(content)
				if len(content) <= remaining {
					content += content
				} else {
					content += content[:remaining]
				}
			}
			for _, char := range RandomSplit(content) {
				time.Sleep(time.Duration(10+rand.Intn(30)) * time.Millisecond)
				a.sendUpdate(ctx, sid, acp.SessionNotification{
					SessionId: sid,
					Update:    acp.UpdateAgentMessageText(char),
				})
			}
		}
	}
}

func (a *mockAgent) sendUpdate(ctx context.Context, sid acp.SessionId, notif acp.SessionNotification) {
	if err := a.conn.SessionUpdate(ctx, notif); err != nil {
		a.logger.Error("send update error", "error", err)
	}
}

type action struct {
	typ     string
	thought string
	name    string
	label   string
	kind    string
}

func makeActions(n int) []action {
	actions := make([]action, 0, n)
	actions = append(actions, action{typ: "think", thought: thoughtPhrases[rand.Intn(len(thoughtPhrases))]})

	middle := n - 2
	for i := 0; i < middle; {
		if rand.Intn(5) == 0 {
			actions = append(actions, action{typ: "plan"})
			i++
			continue
		}
		if rand.Intn(3) == 0 {
			actions = append(actions, action{typ: "think", thought: thoughtPhrases[rand.Intn(len(thoughtPhrases))]})
			i++
			continue
		}
		t := toolTemplates[rand.Intn(len(toolTemplates))]
		actions = append(actions, action{typ: "tool", name: t.name, label: t.desc, kind: t.kind})
		i++
	}
	actions = append(actions, action{typ: "text"})
	return actions
}

type toolTpl struct {
	name string
	kind string
	desc string
}

var toolTemplates = []toolTpl{
	{name: "read_dir", kind: "read", desc: "List directory"},
	{name: "read_file", kind: "read", desc: "Read file"},
	{name: "edit_file", kind: "edit", desc: "Edit file"},
	{name: "bash", kind: "run", desc: "Run command"},
}

var thoughtPhrases = []string{
	"Let me analyze this step by step.",
	"Let me check the relevant files first.",
	"Looking at the code, I can see what needs to change.",
	"Let me trace through the logic carefully.",
	"I should verify the impact of this change.",
	"Now I understand the issue clearly.",
	"Let me plan the implementation approach.",
	"Time to make the actual changes.",
	"Let me verify the changes compile correctly.",
}

func mockToolOutput(name string) string {
	switch name {
	case "read_dir":
		return "main.go\nclient/\ntui/\nagent/\nexamples/\nMakefile"
	case "read_file":
		return "package main\n\nfunc main() {\n    println(\"hello\")\n}"
	case "edit_file":
		return "File updated successfully"
	case "bash":
		return "Build succeeded"
	}
	return "done"
}

func (a *mockAgent) ListSessions(ctx context.Context, params acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

func (a *mockAgent) SetSessionConfigOption(ctx context.Context, params acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (a *mockAgent) SetSessionMode(ctx context.Context, params acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}
