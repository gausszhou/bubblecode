package client

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"
)

func TestMustCwd(t *testing.T) {
	cwd := MustCwd()
	if cwd == "" {
		t.Error("Expected non-empty CWD")
	}
}

func TestInputCommand(t *testing.T) {
	cmd := InputCommand{Type: CmdPrompt, Text: "test message"}
	if cmd.Type != CmdPrompt {
		t.Errorf("Expected Type to be CmdPrompt, got %d", cmd.Type)
	}
	if cmd.Text != "test message" {
		t.Errorf("Expected Text to be 'test message', got %s", cmd.Text)
	}
}

func TestOutputEvent(t *testing.T) {
	ev := OutputEvent{Kind: "test", Error: nil}
	if ev.Kind != "test" {
		t.Errorf("Expected Kind to be 'test', got %s", ev.Kind)
	}
}

func TestNewACPClient(t *testing.T) {
	events := make(chan OutputEvent, 100)
	c := NewACPClient(events)
	if c.Events != events {
		t.Error("Expected Events channel to match")
	}
}

func TestNewPromptRunner(t *testing.T) {
	inputCh := make(chan InputCommand, 1)
	events := make(chan OutputEvent, 100)

	r := NewClient(inputCh, nil, events)
	if r == nil {
		t.Error("Expected non-nil PromptRunner")
	}
}

func TestClientRequestMarshal(t *testing.T) {
	req := acp.ClientRequest{
		Id:     acp.RequestId{Number: ptr(acp.RequestIdNumber(1))},
		Method: "prompt",
		Params: acp.PromptRequest{
			SessionId: "mock-session",
			Prompt:    []acp.ContentBlock{{Text: &acp.ContentBlockText{Text: "hello"}}},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Method != "prompt" {
		t.Errorf("Expected method 'prompt', got %s", decoded.Method)
	}
}

func TestInitializeRequest(t *testing.T) {
	req := acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs:       acp.FileSystemCapabilities{ReadTextFile: true, WriteTextFile: true},
			Terminal: true,
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded acp.InitializeRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ProtocolVersion != acp.ProtocolVersionNumber {
		t.Errorf("Expected ProtocolVersion %d, got %d", acp.ProtocolVersionNumber, decoded.ProtocolVersion)
	}
}

func TestNewSessionRequest(t *testing.T) {
	req := acp.NewSessionRequest{
		Cwd:        "/test",
		McpServers: []acp.McpServer{},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded acp.NewSessionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Cwd != "/test" {
		t.Errorf("Expected Cwd '/test', got %s", decoded.Cwd)
	}
}

func TestPromptRequest(t *testing.T) {
	req := acp.PromptRequest{
		SessionId: "mock-session",
		Prompt:    []acp.ContentBlock{{Text: &acp.ContentBlockText{Text: "Hello, World!"}}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded acp.PromptRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Prompt) != 1 {
		t.Errorf("Expected 1 prompt block, got %d", len(decoded.Prompt))
	}
}

func TestSessionUpdateNotification(t *testing.T) {
	notif := acp.SessionNotification{
		SessionId: "mock-session",
		Update:    acp.SessionUpdate{AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{Content: acp.ContentBlock{Text: &acp.ContentBlockText{Text: "test"}}}},
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded acp.SessionNotification
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.SessionId != "mock-session" {
		t.Errorf("Expected 'mock-session', got %s", decoded.SessionId)
	}
}
