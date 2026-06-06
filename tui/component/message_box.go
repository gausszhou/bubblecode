package component

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type MessageRole int

const (
	RoleUser MessageRole = iota
	RoleAgent
	RoleThought
	RoleSystem
	RoleTool
)

type ChatMessage struct {
	Role          MessageRole
	Content       string
	ToolCallTitle string
	ToolCallID    string
	ToolStatus    string
}

func (m ChatMessage) Render(width int) string {
	prefix := ""
	var style lipgloss.Style

	switch m.Role {
	case RoleUser:
		prefix = "You"
		style = theme.StyleUser
	case RoleAgent:
		prefix = "Agent"
		style = theme.StyleAgent
	case RoleThought:
		prefix = "Thought"
		style = theme.StyleThought
	case RoleTool:
		prefix = fmt.Sprintf("Tool: %s (%s)", m.ToolCallTitle, m.ToolStatus)
		style = theme.StyleTool
	case RoleSystem:
		prefix = "System"
		style = theme.StyleSystem
	}

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	prefixStr := style.Render(prefix)
	contentStr := theme.StyleContent.Width(contentWidth).PaddingLeft(2).Render(m.Content)

	return prefixStr + "\n" + contentStr
}
