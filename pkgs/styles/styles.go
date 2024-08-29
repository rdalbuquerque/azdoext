package styles

import (
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	azdoBlue       = lipgloss.Color("#0178d4")
	LogoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(azdoBlue)).Bold(true)
	ShortHelpStyle = help.New().Styles.ShortKey
	TitleStyle     = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0).Styles.Title.Align(lipgloss.Center).Background(lipgloss.Color("#000000")).Foreground(lipgloss.Color("#ffffff")).Padding(0, 1)
	ActiveStyle    = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color(azdoBlue))

	InactiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#6c6c6c"))
	SpinnerStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color(azdoBlue))
	Height                   int
	Width                    int
	DefaultSectionWidth      = 40
	DefaultSectionHeightDiff = 1

	noRuns             = lipgloss.NewStyle().SetString("■").Foreground(lipgloss.Color("8"))                  // Gray
	pending            = lipgloss.NewStyle().SetString("•").Foreground(lipgloss.Color("39"))                 // Blue
	succeededVSCode    = lipgloss.NewStyle().SetString("✔").Foreground(lipgloss.Color("#54a362"))            // Green
	succeededBasic     = lipgloss.NewStyle().Bold(true).SetString("✓").Foreground(lipgloss.Color("#54a362")) // Green
	failedVSCode       = lipgloss.NewStyle().SetString("✖").Foreground(lipgloss.Color("#cd4944"))            // Red
	failedBasic        = lipgloss.NewStyle().Bold(true).SetString("✗").Foreground(lipgloss.Color("#cd4944")) // Red
	skipped            = lipgloss.NewStyle().Bold(true).SetString(">").Foreground(lipgloss.Color("#ffffff")) // White
	partiallySucceeded = lipgloss.NewStyle().Bold(true).SetString("!").Foreground(lipgloss.Color("#d67e3c")) // Yellow
	canceled           = lipgloss.NewStyle().Bold(true).SetString("-").Foreground(lipgloss.Color("#ffffff")) // White
	SymbolMap          = map[string]lipgloss.Style{
		"pending":            pending,
		"succeeded":          GetSucceededSymbol(),
		"failed":             GetFailedSymbol(),
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

func GetSucceededSymbol() lipgloss.Style {
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return succeededVSCode
	} else {
		return succeededBasic
	}
}

func GetFailedSymbol() lipgloss.Style {
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return failedVSCode
	} else {
		return failedBasic
	}
}
