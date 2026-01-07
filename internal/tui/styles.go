package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	dangerColor    = lipgloss.Color("#EF4444") // Red
	mutedColor     = lipgloss.Color("#6B7280") // Gray
	bgColor        = lipgloss.Color("#1F2937") // Dark gray
	fgColor        = lipgloss.Color("#F9FAFB") // Light gray

	// App styles
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgColor).
			Background(primaryColor).
			Padding(0, 2).
			MarginBottom(1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// List styles
	listItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true).
				PaddingLeft(2)

	cursorStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Repo details
	repoNameStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fgColor)

	repoDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	// Tags
	privateTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(warningColor).
			Padding(0, 1).
			MarginLeft(1)

	publicTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(secondaryColor).
			Padding(0, 1).
			MarginLeft(1)

	archivedTagStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFF")).
				Background(mutedColor).
				Padding(0, 1).
				MarginLeft(1)

	forkTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#3B82F6")).
			Padding(0, 1).
			MarginLeft(1)

	// Stats
	statsStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	starStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FCD34D"))

	forkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA"))

	// Help
	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Panels
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 2)

	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	// Filter input
	filterInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(0, 1)

	// Confirmation dialog
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(dangerColor).
			Padding(1, 2).
			Width(50)

	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(dangerColor).
				MarginBottom(1)

	// Selection indicator
	checkboxStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	uncheckedStyle = lipgloss.NewStyle().
			Foreground(mutedColor)

	// Style helpers for inline rendering
	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)
	dangerStyle = lipgloss.NewStyle().Foreground(dangerColor)
	successStyle = lipgloss.NewStyle().Foreground(secondaryColor)

	// Language colors (common languages)
	langColors = map[string]lipgloss.Color{
		"Go":         lipgloss.Color("#00ADD8"),
		"Python":     lipgloss.Color("#3776AB"),
		"JavaScript": lipgloss.Color("#F7DF1E"),
		"TypeScript": lipgloss.Color("#3178C6"),
		"Rust":       lipgloss.Color("#DEA584"),
		"Ruby":       lipgloss.Color("#CC342D"),
		"Java":       lipgloss.Color("#B07219"),
		"C":          lipgloss.Color("#555555"),
		"C++":        lipgloss.Color("#F34B7D"),
		"C#":         lipgloss.Color("#239120"),
		"PHP":        lipgloss.Color("#777BB4"),
		"Swift":      lipgloss.Color("#FA7343"),
		"Kotlin":     lipgloss.Color("#A97BFF"),
		"Shell":      lipgloss.Color("#89E051"),
		"HTML":       lipgloss.Color("#E34C26"),
		"CSS":        lipgloss.Color("#563D7C"),
	}
)

// GetLangStyle returns a style for a language
func GetLangStyle(lang string) lipgloss.Style {
	color, ok := langColors[lang]
	if !ok {
		color = mutedColor
	}
	return lipgloss.NewStyle().Foreground(color)
}
