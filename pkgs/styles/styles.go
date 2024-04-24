package styles

import "github.com/charmbracelet/lipgloss"

var (
	ActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#00ff00"))

	InactiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#6c6c6c"))
	Height                   int
	Width                    int
	DefaultSectionWidth      = 40
	DefaultSectionHeightDiff = 1

	noRuns             = lipgloss.NewStyle().SetString("■").Foreground(lipgloss.Color("#808080"))
	pending            = lipgloss.NewStyle().SetString("⊛").Foreground(lipgloss.Color("#ffbf00"))
	succeeded          = lipgloss.NewStyle().SetString("✔").Foreground(lipgloss.Color("#00ff00"))
	failed             = lipgloss.NewStyle().SetString("✖").Foreground(lipgloss.Color("#ff0000"))
	skipped            = lipgloss.NewStyle().SetString("➤").Foreground(lipgloss.Color("#ffffff"))
	partiallySucceeded = lipgloss.NewStyle().SetString("⚠").Foreground(lipgloss.Color("#ffbf00"))
	canceled           = lipgloss.NewStyle().SetString("⊝").Foreground(lipgloss.Color("#ffbf00"))
	SymbolMap          = map[string]lipgloss.Style{
		"pending":            pending,
		"succeeded":          succeeded,
		"failed":             failed,
		"skipped":            skipped,
		"noRuns":             noRuns,
		"partiallySucceeded": partiallySucceeded,
		"canceled":           canceled,
		"notStarted":         pending,
	}
)

func SetDimensions(width, height int) {
	Height = height
	Width = width
	ActiveStyle.Height(Height)
	InactiveStyle.Height(Height)
}
