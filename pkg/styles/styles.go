package styles

import (
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	azdoBlue       = lipgloss.Color("#0178d4")
	Red            = lipgloss.Color("#cd4944")
	Yellow         = lipgloss.Color("#d67e3c")
	Green          = lipgloss.Color("#54a362")
	Grey           = lipgloss.Color("#6c6c6c")
	White          = lipgloss.Color("#ffffff")
	LogoStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color(azdoBlue)).Bold(true)
	ShortHelpStyle = help.New().Styles.ShortKey
	TitleStyle     = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0).Styles.Title.Align(lipgloss.Center).Background(lipgloss.Color(azdoBlue)).Foreground(White).Padding(0, 1)
	ActiveStyle    = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color(azdoBlue))

	InactiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(Grey)
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(azdoBlue))
	Height       int

	Width                    int
	DefaultSectionWidth      = 40
	DefaultSectionHeightDiff = 1

	noRuns          = lipgloss.NewStyle().SetString("-").Foreground(Grey)                  // Gray
	pending         = lipgloss.NewStyle().SetString("•").Foreground(azdoBlue)              // Blue
	succeededVSCode = lipgloss.NewStyle().SetString("✔").Foreground(lipgloss.Color(Green)) // Green

	succeededBasic = lipgloss.NewStyle().Bold(true).SetString("✓").Foreground(lipgloss.Color(Green)) // Green
	failedVSCode   = lipgloss.NewStyle().SetString("✖").Foreground(lipgloss.Color(Red))              // Red
	failedBasic    = lipgloss.NewStyle().Bold(true).SetString("✗").Foreground(lipgloss.Color(Red))   // Red
	skipped        = lipgloss.NewStyle().Bold(true).SetString(">").Foreground(White)                 // White

	partiallySucceeded = lipgloss.NewStyle().Bold(true).SetString("!").Foreground(lipgloss.Color(Yellow)) // Yellow
	canceled           = lipgloss.NewStyle().Bold(true).SetString("-").Foreground(White)                  // White
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
