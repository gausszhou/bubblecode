package tui

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"

	"github.com/gausszhou/bubblecode/agent"
	"github.com/gausszhou/bubblecode/client"
	"github.com/gausszhou/bubblecode/tui/component"
	"github.com/gausszhou/bubblecode/tui/layout"
	"github.com/gausszhou/bubblecode/tui/theme"
)

const (
	roleUser    = "user"
	roleAgent   = "agent"
	roleThought = "thought"
	roleTool    = "tool"
	rolePlan    = "plan"
)

type Selection struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

type Model struct {
	logger    *slog.Logger
	changeLog *slog.Logger
	inputCh   chan client.InputCommand
	outputCh  chan client.OutputEvent
	cmd       *exec.Cmd
	ctx       context.Context
	cancel    context.CancelFunc

	width  int
	height int

	textarea     textarea.Model
	chatViewport viewport.Model

	messages []component.Message

	chars int
	times int64

	promptRunning bool
	loading       bool
	statusText    string
	spinner       component.Loading

	showCommands bool

	pendingEvents []client.OutputEvent
	mu            sync.Mutex

	dirty bool

	dragging       bool
	needAutoScroll bool

	selecting bool
	selection *Selection

	modelName string
}

func newChangeLog() *slog.Logger {
	home, err := os.UserHomeDir()
	if err != nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	path := filepath.Join(home, ".gausszhou", "bubblecode", "logs", "change.log")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func NewModel(logger *slog.Logger, cmd *exec.Cmd, _ string, ctx context.Context, cancel context.CancelFunc, inputCh chan client.InputCommand, outputCh chan client.OutputEvent) *Model {
	ta := newTextarea()
	vp := viewport.New(viewport.WithWidth(layout.GetChatWidth(layout.InitWidth)-1), viewport.WithHeight(layout.InitHeight))

	modelName := "unknown"
	if p, err := agent.ConfigPath(); err == nil {
		if cfg, err := agent.LoadConfig(p); err == nil {
			modelName = cfg.Model
		}
	}

	m := &Model{
		logger:       logger,
		changeLog:    newChangeLog(),
		inputCh:      inputCh,
		outputCh:     outputCh,
		cmd:          cmd,
		ctx:          ctx,
		cancel:       cancel,
		width:        layout.InitWidth,
		height:       layout.InitHeight,
		textarea:     ta,
		chatViewport: vp,
		statusText:   "Ready",
		spinner:      component.NewLoading(theme.LoadingSpinner()),
		modelName:    modelName,
	}
	m.changeLog.Info("model created", "status", "ready")
	return m
}

func (m *Model) Init() tea.Cmd {
	go m.startEventCollector()
	return tea.Batch(textarea.Blink, spinnerTick(), pollResize(), drainEventsCmd(), renderCmd())
}

func (m *Model) startEventCollector() {
	m.changeLog.Info("event collector started")
	for {
		select {
		case ev, ok := <-m.outputCh:
			if !ok {
				m.changeLog.Info("event collector: output channel closed")
				return
			}
			m.mu.Lock()
			m.pendingEvents = append(m.pendingEvents, ev)
			n := len(m.pendingEvents)
			m.mu.Unlock()
			if ev.Update != nil {
				m.changeLog.Info("event collected", "kind", "update", "pending", n)
			} else {
				m.changeLog.Info("event collected", "kind", ev.Kind, "pending", n)
			}
		case <-m.ctx.Done():
			m.changeLog.Info("event collector: context cancelled")
			return
		}
	}
}

func (m *Model) drainEvents() {
	m.mu.Lock()
	events := m.pendingEvents
	m.pendingEvents = nil
	m.mu.Unlock()
	for _, ev := range events {
		m.handleOutputEvent(ev)
	}
}

func (m *Model) refreshChat() {
	t0 := time.Now()
	content := m.renderMessages()
	t1 := time.Now()
	m.chatViewport.SetContent(content)
	t2 := time.Now()

	if m.needAutoScroll {
		m.chatViewport.GotoBottom()
	}
	renderMs := t1.Sub(t0).Milliseconds()
	setMs := t2.Sub(t1).Milliseconds()
	m.times = t2.Sub(t0).Milliseconds()
	chars := len(content)
	m.chars = chars
	if m.times > 50 || chars > 100_0000 {
		m.changeLog.Info("refresh chat",
			"chars", chars,
			"render_ms", renderMs,
			"set_ms", setMs,
		)
	}
}

func (m *Model) updateSizes() {
	m.chatViewport.SetWidth(layout.GetChatWidth(m.width) - 1)
	m.chatViewport.SetHeight(layout.GetChatHeight(m.height))
	m.textarea.SetWidth(layout.GetInputWidth(m.width))
	m.textarea.SetHeight(layout.InputHeight)
}

func (m *Model) cleanup() {
	m.cancel()
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
	}
}

func (m *Model) getSelectedText() string {
	if m.selection == nil {
		return ""
	}
	s := m.selection
	startLine, endLine := s.StartLine, s.EndLine
	startCol, endCol := s.StartCol, s.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	content := m.chatViewport.GetContent()
	lines := strings.Split(content, "\n")

	var result []string
	for i := startLine; i <= endLine && i < len(lines); i++ {
		clean := ansi.Strip(lines[i])

		colStart := 0
		colEnd := len(clean)
		if i == startLine {
			colStart = startCol
		}
		if i == endLine {
			colEnd = endCol
		}
		if colStart < 0 {
			colStart = 0
		}
		if colEnd > len(clean) {
			colEnd = len(clean)
		}
		if colStart >= colEnd {
			continue
		}

		part := clean[colStart:colEnd]
		if len(part) >= 2 {
			part = part[2:]
		}
		part = strings.TrimRight(part, " ")
		part = strings.TrimRight(part, "\x00")
		if part != "" {
			result = append(result, part)
		}
	}
	return strings.Join(result, "\n")
}

func (m *Model) copyToClipboard(text string) {
	if text == "" {
		m.statusText = "nothing selected"
		return
	}
	err := clipboard.WriteAll(text)
	if err != nil {
		m.statusText = "copy failed"
	} else {
		preview := text
		if len([]rune(preview)) > 40 {
			preview = string([]rune(preview)[:40]) + "..."
		}
		m.statusText = "copied"
	}
}

func (m *Model) applySelectionHighlight(content string) string {
	if !m.selecting || m.selection == nil {
		return content
	}

	s := m.selection
	startLine, endLine := s.StartLine, s.EndLine
	startCol, endCol := s.StartCol, s.EndCol

	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}

	lines := strings.Split(content, "\n")
	for i := startLine; i <= endLine && i < len(lines); i++ {
		lineWidth := ansi.StringWidth(lines[i])
		colStart := 0
		colEnd := lineWidth
		if i == startLine {
			colStart = startCol
		}
		if i == endLine {
			colEnd = endCol
		}
		if colStart >= colEnd || colStart >= lineWidth {
			continue
		}
		if colEnd > lineWidth {
			colEnd = lineWidth
		}

		before := ansi.Cut(lines[i], 0, colStart)
		selected := ansi.Cut(lines[i], colStart, colEnd)
		after := ansi.Cut(lines[i], colEnd, lineWidth)

		selected = strings.ReplaceAll(selected, "\x1b[0m", "\x1b[0m\x1b[7m")

		lines[i] = before + "\x1b[7m" + selected + "\x1b[27m" + after
	}
	return strings.Join(lines, "\n")
}

func newTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message... (Enter to Send)"
	ta.SetWidth(layout.GetInputWidth(layout.InitWidth))
	ta.SetHeight(layout.InputHeight)
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Prompt = theme.AccentStyle().Render("┃ ")
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("shift+enter", "enter"))

	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	s.Blurred.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	return ta
}

type drainEventsMsg struct{}

type renderMsg struct{}

type loadingTickMsg struct{}

type resizePollMsg struct{}

func pollResize() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return resizePollMsg{}
	})
}

func drainEventsCmd() tea.Cmd {
	return tea.Tick(16*time.Millisecond, func(t time.Time) tea.Msg {
		return drainEventsMsg{}
	})
}

func sendInput(ch chan client.InputCommand, cmd client.InputCommand) tea.Cmd {
	return func() tea.Msg {
		ch <- cmd
		return nil
	}
}

func renderCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return renderMsg{}
	})
}

func spinnerTick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return loadingTickMsg{}
	})
}

var _ tea.Model = (*Model)(nil)
