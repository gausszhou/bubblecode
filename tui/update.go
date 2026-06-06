package tui

import (
	"encoding/json"
	"fmt"
	"strings"

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

	switch m.focus {
	case FocusCommands:
		switch msg.String() {
		case "esc", "ctrl+p":
			m.focus = FocusChat
			m.commands.selected = 0
		case "up", "k":
			if m.commands.selected > 0 {
				m.commands.selected--
			}
		case "down", "j":
			cmds := component.DefaultCommands().Commands
			if m.commands.selected < len(cmds)-1 {
				m.commands.selected++
			}
		case "enter":
			m.executeCommand(m.commands.selected)
		}
		return m, nil

	case FocusSessions:
		switch msg.String() {
		case "esc":
			m.focus = FocusChat
			m.sessions.list.SelectedIdx = 0
		case "up", "k":
			m.sessions.list.Up()
		case "down", "j":
			m.sessions.list.Down()
		case "enter":
			m.switchSession()
		}
		return m, nil

	case FocusModels:
		switch msg.String() {
		case "esc":
			m.focus = FocusChat
			m.models.list.SelectedIdx = 0
		case "up", "k":
			m.models.list.Up()
		case "down", "j":
			m.models.list.Down()
		case "enter":
			m.selectModel()
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+n":
		return m, sendInput(m.inputCh, client.InputCommand{Type: client.CmdNewSession})

	case "ctrl+p":
		m.focus = FocusCommands
		return m, nil

	case "ctrl+s":
		m.focus = FocusSessions
		return m, nil

	case "ctrl+m":
		m.focus = FocusModels
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
	switch e := msg.(type) {
	case tea.MouseWheelMsg:
		mouse := e.Mouse()
		if mouse.Y >= m.chatViewport.Height() {
			return m, nil
		}
		var cmd tea.Cmd
		m.chatViewport, cmd = m.chatViewport.Update(e)
		if m.chatViewport.AtBottom() {
			m.needAutoScroll = true
		} else {
			m.needAutoScroll = false
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) handleOutputEvent(ev client.OutputEvent) {
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
	case "session_created":
		id := fmt.Sprintf("%d", m.sessions.nextID)
		m.sessions.nextID++
		name := fmt.Sprintf("Session %s", id)
		m.sessions.list.Sessions = append(m.sessions.list.Sessions, component.SessionItem{
			ID:     id,
			Name:   name,
			Active: true,
		})
		for i := range m.sessions.list.Sessions {
			m.sessions.list.Sessions[i].Active = m.sessions.list.Sessions[i].ID == id
		}
		m.sessions.current = id
		m.messages = nil
		m.chars = 0
		m.statusText = "New session: " + name
		m.focus = FocusChat
		m.dirty = true
		m.changeLog.Info("session created", "id", id)
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

func (m *Model) executeCommand(idx int) {
	cmds := component.DefaultCommands().Commands
	if idx < 0 || idx >= len(cmds) {
		return
	}
	switch cmds[idx].Key {
	case "Ctrl+C":
		m.cleanup()
		// tea.Quit returned via caller if needed
	case "Ctrl+M":
		m.focus = FocusModels
	case "Ctrl+S":
		m.focus = FocusSessions
	case "Ctrl+N":
		m.textarea.Reset()
		m.statusText = "Creating new session..."
		m.focus = FocusChat
		// sendInput handled via caller
	case "Esc Esc":
		m.cancel()
		m.promptRunning = false
		m.loading = false
		m.statusText = "Interrupted"
	}
}

func (m *Model) switchSession() {
	idx := m.sessions.list.SelectedIdx
	if idx < 0 || idx >= len(m.sessions.list.Sessions) {
		return
	}
	sess := m.sessions.list.Sessions[idx]
	if sess.ID == m.sessions.current {
		m.focus = FocusChat
		return
	}
	m.sessions.current = sess.ID
	for i := range m.sessions.list.Sessions {
		m.sessions.list.Sessions[i].Active = m.sessions.list.Sessions[i].ID == sess.ID
	}
	m.messages = nil
	m.chars = 0
	m.textarea.Reset()
	m.statusText = "Switched to " + sess.Name
	m.focus = FocusChat
	m.dirty = true
}

func (m *Model) selectModel() {
	idx := m.models.list.SelectedIdx
	if idx >= 0 && idx < len(m.models.list.Models) {
		m.modelName = m.models.list.Models[idx]
		m.textarea.Reset()
		m.textarea.SetValue("/model " + m.modelName)
		m.statusText = "Model: " + m.modelName
		m.focus = FocusChat
	}
}

func (m *Model) sendPrompt() (tea.Model, tea.Cmd) {
	if m.promptRunning {
		return m, nil
	}
	text := m.textarea.Value()
	if text == "" {
		return m, nil
	}

	if text == "/new" {
		m.textarea.Reset()
		return m, sendInput(m.inputCh, client.InputCommand{Type: client.CmdNewSession})
	}

	if text == "/models" {
		m.textarea.Reset()
		m.focus = FocusModels
		return m, nil
	}

	if text == "/sessions" {
		m.textarea.Reset()
		m.focus = FocusSessions
		return m, nil
	}

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
