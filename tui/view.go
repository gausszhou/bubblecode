package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	flex "github.com/gausszhou/bubbleflex"

	"github.com/gausszhou/bubblecode/tui/component"
	"github.com/gausszhou/bubblecode/tui/layout"
	"github.com/gausszhou/bubblecode/tui/overlay"
	"github.com/gausszhou/bubblecode/tui/theme"
)

func (m *Model) View() tea.View {
	chat := m.chatViewport.View()
	if m.focus == FocusChat {
		h := m.chatViewport.Height()
		sb := renderScrollbar(h, m.chatViewport.ScrollPercent(), m.chatViewport.TotalLineCount())
		chat = lipgloss.JoinHorizontal(lipgloss.Top, chat, sb)
	}
	input := m.textarea.View()
	status := m.renderStatus()
	slashSugg := m.renderSlashSuggestions()

	content := lipgloss.JoinVertical(lipgloss.Left, chat, "\n"+input, status)

	switch m.focus {
	case FocusCommands:
		overlayContent := m.renderCommandOverlay()
		content = overlay.CompositeMasked(overlayContent, content, overlay.Center, overlay.Center, 0, 0, true)
	case FocusSessions:
		overlayContent := m.renderSessionOverlay()
		content = overlay.CompositeMasked(overlayContent, content, overlay.Center, overlay.Center, 0, 0, true)
	case FocusModels:
		overlayContent := m.renderModelsOverlay()
		content = overlay.CompositeMasked(overlayContent, content, overlay.Center, overlay.Center, 0, 0, true)
	default:
		if slashSugg != "" {
			content = lipgloss.JoinVertical(lipgloss.Left, chat, slashSugg, "\n"+input, status)
		}
	}

	view := tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, layout.PaddingHorizontal).
		Render(content))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeAllMotion

	if c := m.textarea.Cursor(); c != nil && m.focus == FocusChat {
		c.Y += lipgloss.Height(chat) + 1
		if slashSugg != "" {
			c.Y += lipgloss.Height(slashSugg)
		}
		view.Cursor = c
	}

	return view
}

func (m *Model) renderCommandOverlay() string {
	return component.DefaultCommands().OverlayView(m.commands.selected)
}

func (m *Model) renderSessionOverlay() string {
	return m.sessions.list.OverlayView()
}

func (m *Model) renderModelsOverlay() string {
	return m.models.list.OverlayView()
}

func (m *Model) renderSlashSuggestions() string {
	val := m.textarea.Value()
	if !strings.HasPrefix(val, "/") {
		return ""
	}
	w := m.chatViewport.Width()
	if w < 10 {
		w = 40
	}
	return component.RenderSlashSuggestions(val, w)
}

func (m *Model) renderMessages() string {
	w := m.chatViewport.Width()
	var sb strings.Builder
	for i := range m.messages {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(m.messages[i].Render(w))
	}
	return sb.String()
}

func comma(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var buf []byte
	for i, c := range s {
		buf = append(buf, byte(c))
		if c == '-' {
			continue
		}
		fromRight := len(s) - i
		if fromRight > 3 && fromRight%3 == 1 {
			buf = append(buf, ',')
		}
	}
	return string(buf)
}

func renderScrollbar(height int, percent float64, totalLines int) string {
	if totalLines < 1 {
		totalLines = 1
	}
	thumbHeight := max(1, height*height/totalLines)
	if thumbHeight > height {
		thumbHeight = height
	}
	maxOffset := height - thumbHeight
	thumbStart := int(percent * float64(maxOffset))
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart > maxOffset {
		thumbStart = maxOffset
	}
	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i >= thumbStart && i < thumbStart+thumbHeight {
			sb.WriteString(theme.ScrollbarThumb)
		} else {
			sb.WriteString(theme.ScrollbarTrack)
		}
		if i < height-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *Model) renderStatus() string {
	left := m.statusText
	if m.loading {
		left = m.spinner.View() + " " + left
	} else {
		left = "✓ " + left
	}
	right := fmt.Sprintf("%s  •  %s chars  •  %d ms", m.modelName, comma(m.chars), m.times)
	line := flex.New(flex.Row).
		JustifyContent(flex.SpaceBetween).
		Width(m.width-2*layout.PaddingHorizontal).
		Join(left, right)
	return theme.StatusBar().Width(m.width - 2*layout.PaddingHorizontal).Render(line)
}
