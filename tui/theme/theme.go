package theme

import (
	"charm.land/lipgloss/v2"
)

var (
	ThemeBg      = lipgloss.Color("#201d1d")
	ThemeSurface = lipgloss.Color("#302c2c")
	ThemeBorder  = lipgloss.Color("#646262")
	ThemeText    = lipgloss.Color("#fdfcfc")
	ThemeMuted   = lipgloss.Color("#9a9898")
	ThemeDim     = lipgloss.Color("#6e6e73")
	ThemeAccent  = lipgloss.Color("#007aff")
	ThemeSuccess = lipgloss.Color("#30d158")
	ThemeWarning = lipgloss.Color("#ff9f0a")
	ThemeDanger  = lipgloss.Color("#ff3b30")

	ThemeUser    = lipgloss.Color("#fdfcfc")
	ThemeAgent   = lipgloss.Color("#9a9898")
	ThemeThought = lipgloss.Color("#6e6e73")
	ThemeTool    = lipgloss.Color("#646262")
	ThemeSystem  = lipgloss.Color("#9a9898")

	ThemeBgDark    = lipgloss.Color("#0c0c0c")
	ThemeBgOverlay = lipgloss.Color("#1e1e1e")
)

func BaseStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(ThemeBg).
		Foreground(ThemeText)
}

func TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeText)
}

func MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeMuted)
}

func DimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeDim)
}

func AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeAccent)
}

func SurfaceStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeSurface)
}

func BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeBorder)
}

func StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeMuted)
}

var (
	StyleUser    = lipgloss.NewStyle().Foreground(ThemeUser).Bold(true)
	StyleAgent   = lipgloss.NewStyle().Foreground(ThemeAgent).Bold(true)
	StyleThought = lipgloss.NewStyle().Foreground(ThemeThought).Italic(true)
	StyleTool    = lipgloss.NewStyle().Foreground(ThemeTool).Bold(true)
	StyleSystem  = lipgloss.NewStyle().Foreground(ThemeSystem)
	StyleContent = lipgloss.NewStyle().Foreground(ThemeMuted)
)

func OverlayBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(60).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ThemeWarning).
		Background(ThemeBgOverlay).
		Padding(1, 2)
}

func LoadingSpinner() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeAccent)
}

func HelpLabel() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ThemeDim).Padding(0, 1)
}

func StatusBarBg() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(ThemeBgDark).
		Foreground(ThemeMuted)
}

var (
	TodoTitleStyle    = lipgloss.NewStyle().Foreground(ThemeAccent).Bold(true)
	TodoPendingStyle  = lipgloss.NewStyle().Foreground(ThemeDim)
	TodoProgressStyle = lipgloss.NewStyle().Foreground(ThemeWarning)
	TodoCompleteStyle = lipgloss.NewStyle().Foreground(ThemeSuccess)
	TodoEmptyStyle    = lipgloss.NewStyle().Foreground(ThemeDim).PaddingLeft(2)

	QuestionBoxStyle         = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(ThemeWarning).Padding(1, 2)
	QuestionBoxTitleStyle    = lipgloss.NewStyle().Foreground(ThemeWarning).Bold(true)
	QuestionBoxActiveStyle   = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeAccent).Padding(0, 1)
	QuestionBoxInactiveStyle = lipgloss.NewStyle().Foreground(ThemeMuted).Padding(0, 1)
	QuestionBoxMessageStyle  = lipgloss.NewStyle().Foreground(ThemeText)

	ButtonNormalStyle = lipgloss.NewStyle().Foreground(ThemeMuted).Padding(0, 1)
	ButtonFocusStyle  = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeAccent).Padding(0, 1)

	UsageTitleStyle = lipgloss.NewStyle().Foreground(ThemeAccent).Bold(true)
	UsageLabelStyle = lipgloss.NewStyle().Foreground(ThemeDim)
	UsageValueStyle = lipgloss.NewStyle().Foreground(ThemeText)

	CommandPanelStyle = lipgloss.NewStyle().Padding(0, 1)
	CommandKeyStyle   = lipgloss.NewStyle().Foreground(ThemeAccent).Bold(true)
	CommandDescStyle  = lipgloss.NewStyle().Foreground(ThemeDim)

	SessionListTitleStyle = lipgloss.NewStyle().Foreground(ThemeAccent).Bold(true)
	SessionActiveStyle    = lipgloss.NewStyle().Foreground(ThemeSuccess)
	SessionNormalStyle    = lipgloss.NewStyle().Foreground(ThemeMuted)
	SessionSelectStyle    = lipgloss.NewStyle().Foreground(ThemeText).Background(ThemeAccent).Padding(0, 1)
	SessionEmptyStyle     = lipgloss.NewStyle().Foreground(ThemeDim).PaddingLeft(2)

	ScrollbarThumb = lipgloss.NewStyle().Foreground(ThemeDim).Render("▓")
	ScrollbarTrack = lipgloss.NewStyle().Foreground(ThemeSurface).Render("░")
)
