package client

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/coder/acp-go-sdk"
)

func MustCwd() string {
	wd, _ := os.Getwd()
	if wd == "" {
		return "/"
	}
	return wd
}

type InputCommand struct {
	Type int
	Text string
}

const (
	CmdPrompt = iota
	CmdInterrupt
)

type OutputEvent struct {
	Kind   string
	Update *acp.SessionNotification
	Error  error
}

// ACPClient implements acp.Client for handling agent callbacks.
type ACPClient struct {
	Events chan<- OutputEvent
}

func NewACPClient(events chan<- OutputEvent) *ACPClient {
	return &ACPClient{Events: events}
}

func (c *ACPClient) ReadTextFile(_ context.Context, params acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return acp.ReadTextFileResponse{}, err
	}
	return acp.ReadTextFileResponse{Content: string(data)}, nil
}

func (c *ACPClient) WriteTextFile(_ context.Context, params acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
		return acp.WriteTextFileResponse{}, err
	}
	return acp.WriteTextFileResponse{}, nil
}

func (c *ACPClient) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{}, fmt.Errorf("permission denied")
}

func (c *ACPClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	c.Events <- OutputEvent{Kind: "update", Update: &params}
	return nil
}

func (c *ACPClient) CreateTerminal(ctx context.Context, params acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, fmt.Errorf("not implemented")
}

func (c *ACPClient) KillTerminal(ctx context.Context, params acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, fmt.Errorf("not implemented")
}

func (c *ACPClient) TerminalOutput(ctx context.Context, params acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, fmt.Errorf("not implemented")
}

func (c *ACPClient) ReleaseTerminal(ctx context.Context, params acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("not implemented")
}

func (c *ACPClient) WaitForTerminalExit(ctx context.Context, params acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("not implemented")
}

// PromptRunner sends prompts and manages the prompt lifecycle.
type PromptRunner struct {
	Input  <-chan InputCommand
	conn   *acp.ClientSideConnection
	events chan<- OutputEvent
}

func NewClient(input <-chan InputCommand, conn *acp.ClientSideConnection, events chan<- OutputEvent) *PromptRunner {
	return &PromptRunner{Input: input, conn: conn, events: events}
}

func (c *PromptRunner) Run(ctx context.Context, sessionId acp.SessionId) {
	for {
		select {
		case cmd, ok := <-c.Input:
			if !ok {
				return
			}
			switch cmd.Type {
			case CmdPrompt:
				_, err := c.conn.Prompt(ctx, acp.PromptRequest{
					SessionId: sessionId,
					Prompt:    []acp.ContentBlock{acp.TextBlock(cmd.Text)},
				})
				if err != nil {
					c.events <- OutputEvent{Kind: "error", Error: err}
				} else {
					c.events <- OutputEvent{Kind: "done"}
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

type Connection struct {
	cmd    *exec.Cmd
	conn   *acp.ClientSideConnection
	logger *slog.Logger
}

func NewConnection(cmd *exec.Cmd, client acp.Client, logger *slog.Logger) (*Connection, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	conn := acp.NewClientSideConnection(client, stdin, stdout)
	conn.SetLogger(logger)

	return &Connection{
		cmd:    cmd,
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *Connection) Initialize(ctx context.Context, req acp.InitializeRequest) (acp.InitializeResponse, error) {
	return c.conn.Initialize(ctx, req)
}

func (c *Connection) NewSession(ctx context.Context, req acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return c.conn.NewSession(ctx, req)
}

func (c *Connection) ClientConn() *acp.ClientSideConnection {
	return c.conn
}

func ptr[T any](v T) *T {
	return &v
}
