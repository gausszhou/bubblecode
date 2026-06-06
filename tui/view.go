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
	{
		h := m.chatViewport.Height()
		sb := renderScrollbar(h, m.chatViewport.ScrollPercent())
		chat = lipgloss.JoinHorizontal(lipgloss.Top, chat, sb)
	}
	input := m.textarea.View()
	status := m.renderStatus()

	content := lipgloss.JoinVertical(lipgloss.Left, chat, "\n"+input, status)

	if m.showCommands {
		overlayContent := m.renderCommandOverlay()
		content = overlay.CompositeMasked(overlayContent, content, overlay.Center, overlay.Center, 0, 0, true)
	} else if s := m.renderSlashSuggestions(); s != "" {
		content = overlay.CompositeMasked(s, content, overlay.Center, overlay.Bottom, 0, 0, true)
	}

	view := tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(0, layout.PaddingHorizontal).
		Render(content))
	view.AltScreen = true
	view.MouseMode = tea.MouseModeAllMotion

	if c := m.textarea.Cursor(); c != nil && !m.showCommands {
		c.Y += lipgloss.Height(chat) + 1
		view.Cursor = c
	}

	return view
}

func (m *Model) renderCommandOverlay() string {
	panel := component.DefaultCommands()
	bg := theme.ThemeBgOverlay
	var sb strings.Builder
	sb.WriteString(theme.AccentStyle().Background(bg).Render("Commands"))
	sb.WriteString("\n\n")
	for _, cmd := range panel.Commands {
		sb.WriteString("  ")
		sb.WriteString(theme.CommandKeyStyle.Background(bg).Render(cmd.Key))
		sb.WriteString("  ")
		sb.WriteString(theme.CommandDescStyle.Background(bg).Render(cmd.Desc))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(theme.AccentStyle().Background(bg).Render("Slash commands in chat"))
	sb.WriteString("\n\n")
	for _, sc := range component.SlashCommands {
		sb.WriteString("  ")
		sb.WriteString(theme.CommandKeyStyle.Background(bg).Render(sc.Command))
		sb.WriteString("  ")
		sb.WriteString(theme.CommandDescStyle.Background(bg).Render(sc.Desc))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(theme.HelpLabel().Background(bg).Render("Esc to close"))
	return theme.OverlayBox().Render(sb.String())
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

func renderScrollbar(height int, percent float64) string {
	thumb := int(percent * float64(height-1))
	if thumb < 0 {
		thumb = 0
	}
	if thumb >= height {
		thumb = height - 1
	}
	var sb strings.Builder
	for i := 0; i < height; i++ {
		if i == thumb {
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

func (m *Model) isScrollbarX(x int) bool {
	scrollbarX := m.width - layout.PaddingHorizontal - 1
	return x == scrollbarX
}

func (m *Model) isViewportY(y int) bool {
	return y >= 0 && y < m.chatViewport.Height()
}

func (m *Model) setScrollFromY(y int) {
	h := m.chatViewport.Height()
	total := m.chatViewport.TotalLineCount()
	maxScroll := total - h
	if maxScroll <= 0 {
		return
	}
	pct := float64(y) / float64(h)
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	m.chatViewport.SetYOffset(int(pct * float64(maxScroll)))
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
