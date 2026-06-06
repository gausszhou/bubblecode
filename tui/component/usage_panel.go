package component

import (
	"charm.land/lipgloss/v2"
	"github.com/gausszhou/bubblecode/tui/theme"
)

type UsageInfo struct {
	ModelName  string
	TitleStyle lipgloss.Style
	LabelStyle lipgloss.Style
	ValueStyle lipgloss.Style
}

func NewUsageInfo() UsageInfo {
	return UsageInfo{
		TitleStyle: theme.UsageTitleStyle,
		LabelStyle: theme.UsageLabelStyle,
		ValueStyle: theme.UsageValueStyle,
	}
}

func (ui UsageInfo) View() string {
	title := ui.TitleStyle.Render("Model")
	model := ui.LabelStyle.Render("Model: ") + ui.ValueStyle.Render(ui.ModelName)
	if ui.ModelName == "" {
		model = ui.LabelStyle.Render("—")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		model,
	)
}
