package tui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/cursor"
	tea "charm.land/bubbletea/v2"
	"github.com/coder/acp-go-sdk"

	"github.com/gausszhou/bubblecode/client"
	"github.com/gausszhou/bubblecode/tui/component"
	"github.com/gausszhou/bubblecode/tui/layout"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.KeyReleaseMsg:
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case drainEventsMsg:
		m.drainEvents()
		return m, drainEventsCmd()

	case renderMsg:
		if m.dirty {
			m.refreshChat()
			m.dirty = false
		}
		return m, renderCmd()

	case loadingTickMsg:
		m.spinner = m.spinner.Tick()
		return m, spinnerTick()

	case cursor.BlinkMsg:
		return m.handleBlink(msg)

	case tea.PasteMsg:
		m.textarea.Update(msg)
		return m, nil

	case resizePollMsg:
		return m, tea.Batch(tea.RequestWindowSize, pollResize())
	}
	return m, nil
}

func (m *Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	if msg.Width < layout.MinWidth || msg.Height < layout.MinHeight {
		return m, nil
	}
	if msg.Width == m.width && msg.Height == m.height {
		return m, nil
	}
	m.width = msg.Width
	m.height = msg.Height
	m.updateSizes()
	m.chatViewport.SetContent(m.renderMessages())
	if m.chatViewport.AtBottom() {
		m.needAutoScroll = true
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.cleanup()
		return m, tea.Quit
	}

	if m.showCommands {
		switch msg.String() {
		case "esc", "ctrl+p":
			m.showCommands = false
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+p":
		m.showCommands = true
		return m, nil

	case "enter":
		return m.sendPrompt()

	case "up", "k":
		m.chatViewport.ScrollUp(1)
		m.needAutoScroll = false
		return m, nil

	case "down", "j":
		m.chatViewport.ScrollDown(1)
		if m.chatViewport.AtBottom() {
			m.needAutoScroll = true
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	t0 := time.Now()

	switch e := msg.(type) {
	case tea.MouseWheelMsg:
		switch e.Button {
		case tea.MouseWheelUp:
			m.chatViewport.ScrollUp(3)
			m.needAutoScroll = false
		case tea.MouseWheelDown:
			m.chatViewport.ScrollDown(3)
			if m.chatViewport.AtBottom() {
				m.needAutoScroll = true
			}
		}

	case tea.MouseClickMsg:
		mouse := e.Mouse()
		if mouse.Button == tea.MouseLeft && m.isScrollbarX(mouse.X) && m.isViewportY(mouse.Y) {
			m.dragging = true
			m.setScrollFromY(mouse.Y)
			m.needAutoScroll = m.chatViewport.AtBottom()
		} else if mouse.Button == tea.MouseLeft && m.isViewportY(mouse.Y) && mouse.X < m.chatViewport.Width() {
			m.selecting = true
			contentLine := m.chatViewport.YOffset() + mouse.Y
			m.selection = &Selection{
				StartLine: contentLine,
				StartCol:  mouse.X,
				EndLine:   contentLine,
				EndCol:    mouse.X,
			}
			m.dirty = true
		}

	case tea.MouseReleaseMsg:
		if m.selecting && m.selection != nil {
			text := m.getSelectedText()
			m.copyToClipboard(text)
			m.selecting = false
			m.selection = nil
			m.dirty = true
		}
		m.dragging = false

	case tea.MouseMotionMsg:
		mouse := e.Mouse()
		if m.dragging {
			if m.isScrollbarX(mouse.X) && m.isViewportY(mouse.Y) {
				m.setScrollFromY(mouse.Y)
				m.needAutoScroll = m.chatViewport.AtBottom()
			}
		}
		if m.selecting && m.selection != nil {
			if mouse.X < 0 {
				mouse.X = 0
			}
			if mouse.X >= m.chatViewport.Width() {
				mouse.X = m.chatViewport.Width() - 1
			}
			y := mouse.Y
			if y < 0 {
				y = 0
			}
			if y >= m.chatViewport.Height() {
				y = m.chatViewport.Height() - 1
			}
			m.selection.EndLine = m.chatViewport.YOffset() + y
			m.selection.EndCol = mouse.X
			m.dirty = true
		}
	}

	if d := time.Since(t0); d > 50*time.Millisecond {
		m.changeLog.Info("handle mouse", "ms", d.Milliseconds())
	}
	return m, nil
}

func (m *Model) handleOutputEvent(ev client.OutputEvent) {
	m.selecting = false
	m.selection = nil

	if ev.Update != nil {
		m.processUpdate(ev.Update.Update)
		m.dirty = true
		m.changeLog.Info("handle update",
			"has_text", ev.Update.Update.AgentMessageChunk != nil,
			"has_thought", ev.Update.Update.AgentThoughtChunk != nil,
			"has_tool_call", ev.Update.Update.ToolCall != nil,
			"has_plan", ev.Update.Update.Plan != nil,
		)
		return
	}

	switch ev.Kind {
	case "done":
		m.promptRunning = false
		m.loading = false
		m.statusText = "Ready"
		m.dirty = true
		m.changeLog.Info("prompt done")
	case "error":
		m.promptRunning = false
		m.loading = false
		m.statusText = "Error: " + ev.Error.Error()
		m.dirty = true
		m.changeLog.Info("prompt error", "error", ev.Error.Error())
	}
}

func (m *Model) handleBlink(msg cursor.BlinkMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *Model) processUpdate(update acp.SessionUpdate) {
	switch {
	case update.AgentMessageChunk != nil && update.AgentMessageChunk.Content.Text != nil:
		m.appendOrNewMessage(roleAgent, update.AgentMessageChunk.Content.Text.Text)

	case update.AgentThoughtChunk != nil && update.AgentThoughtChunk.Content.Text != nil:
		m.appendOrNewMessage(roleThought, update.AgentThoughtChunk.Content.Text.Text)

	case update.ToolCall != nil:
		tc := update.ToolCall
		inputJSON, _ := json.Marshal(tc.RawInput)
		content := tc.Title + "\n" + string(inputJSON)
		m.messages = append(m.messages, component.Message{Role: roleTool, Content: content})
		m.chars += len(content)

	case update.ToolCallUpdate != nil:
		tu := update.ToolCallUpdate
		status := "completed"
		if tu.Status != nil {
			status = string(*tu.Status)
		}
		for i := len(m.messages) - 1; i >= 0; i-- {
			if m.messages[i].Role == roleTool {
				if tu.RawOutput != nil {
					if output := fmt.Sprintf("%v", tu.RawOutput); output != "" {
						m.messages[i].Content += "\n" + output
						m.chars += 1 + len(output) // \n + output
					}
				}
				m.messages[i].Status = status
				break
			}
		}

	case update.Plan != nil:
		var lines []string
		for _, e := range update.Plan.Entries {
			mark := " "
			switch e.Status {
			case acp.PlanEntryStatusCompleted:
				mark = "✓"
			case acp.PlanEntryStatusInProgress:
				mark = "→"
			}
			lines = append(lines, fmt.Sprintf("[%s] %s", mark, e.Content))
		}
		content := strings.Join(lines, "\n")
		m.messages = append(m.messages, component.Message{Role: rolePlan, Content: content})
		m.chars += len(content)
	}
}

func (m *Model) appendOrNewMessage(role, content string) {
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == role {
		m.messages[len(m.messages)-1].Content += content
	} else {
		m.messages = append(m.messages, component.Message{Role: role, Content: content})
	}
	m.chars += len(content)
}

func (m *Model) sendPrompt() (tea.Model, tea.Cmd) {
	if m.promptRunning {
		return m, nil
	}
	text := m.textarea.Value()
	if text == "" {
		return m, nil
	}

	m.selecting = false
	m.selection = nil
	m.textarea.Reset()
	m.messages = append(m.messages, component.Message{Role: roleUser, Content: text})
	m.chars += len(text)
	m.promptRunning = true
	m.loading = true
	m.statusText = "Processing..."
	m.chatViewport.SetContent(m.renderMessages())
	m.chatViewport.GotoBottom()
	m.needAutoScroll = true

	m.changeLog.Info("prompt sent", "text_length", len(text))

	return m, sendInput(m.inputCh, client.InputCommand{Type: client.CmdPrompt, Text: text})
}
