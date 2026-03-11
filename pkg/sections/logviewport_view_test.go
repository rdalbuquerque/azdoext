package sections

import (
	"fmt"
	"strings"
	"testing"

	"azdoext/pkg/listitems"
	"azdoext/pkg/styles"

	"github.com/rdalbuquerque/viewsearch"

	bubbleshelp "charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
)

// newTestLogViewport creates a minimal LogViewportSection for view testing,
// bypassing azdo dependencies that are only needed for Update/network calls.
func newTestLogViewport() *LogViewportSection {
	vp := viewsearch.New()
	vp.SetShowHelp(false)
	styledHelpText := styles.ShortHelpStyle.Render("/ find • alt+m maximize")
	return &LogViewportSection{
		logviewport:       &vp,
		StyledHelpText:    styledHelpText,
		sectionIdentifier: LogViewport,
	}
}

// newTestPipelineTasks creates a minimal PipelineTasksSection for view testing.
func newTestPipelineTasks() *PipelineTasksSection {
	tasklist := list.New([]list.Item{}, listitems.PipelineRecordItemDelegate{}, 0, 0)
	tasklist.SetShowTitle(false)
	tasklist.SetShowStatusBar(false)
	tasklist.SetShowHelp(false)
	tasklist.SetShowPagination(false)
	return &PipelineTasksSection{
		tasklist:          tasklist,
		sectionIdentifier: PipelineTasks,
	}
}

const testSpacer = "  " // matches pages.SectionSpacer

// pageHelpKeys matches the helpKeys in pages.go
type pageHelpKeys struct{}

func (h pageHelpKeys) FullHelp() [][]key.Binding { return nil }
func (h pageHelpKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "help page")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch section")),
		key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "restart")),
		key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("ctrl+b", "previous page")),
		key.NewBinding(key.WithKeys(""), key.WithHelp("↑/k ↓/j navigate and", "↵ select on all lists")),
	}
}

// TestHelpTextFullyVisibleBehavior creates real section instances, calls
// SetDimensions and View() exactly as the app does, then checks the output.
// It simulates the real startup: styles.SetDimensions is called first (from
// WindowSizeMsg), then the page constructor runs and calls page.SetDimensions.
func TestHelpTextFullyVisibleBehavior(t *testing.T) {
	helpText := "/ find • alt+m maximize"

	for _, termWidth := range []int{80, 100, 120, 160, 200} {
		t.Run(fmt.Sprintf("termWidth_%d", termWidth), func(t *testing.T) {
			termHeight := 30

			// Simulate WindowSizeMsg setting global dimensions BEFORE page creation
			// (matches the real startup: WindowSizeMsg arrives before AzdoConfigMsg)
			styles.SetDimensions(termWidth, termHeight-3)

			tasks := newTestPipelineTasks()
			tasks.Show()
			tasks.Focus()
			tasks.SetDimensions(styles.DefaultSectionWidth, termHeight-3)

			logvp := newTestLogViewport()
			logvp.Show()
			logvp.Blur()
			logvp.SetDimensions(styles.Width-styles.DefaultSectionWidth-len(testSpacer), termHeight-3)

			tasksView := tasks.View()
			logView := logvp.View()

			// Reproduce attachView: first section has no spacer, second does
			pageView := lipgloss.JoinHorizontal(lipgloss.Left, tasksView, testSpacer, logView)

			tasksWidth := lipgloss.Width(tasksView)
			logWidth := lipgloss.Width(logView)
			pageWidth := lipgloss.Width(pageView)

			t.Logf("tasksWidth=%d logWidth=%d spacer=%d total=%d termWidth=%d",
				tasksWidth, logWidth, len(testSpacer), pageWidth, termWidth)

			if pageWidth > termWidth {
				t.Errorf("page width %d exceeds terminal width %d (overflow by %d)",
					pageWidth, termWidth, pageWidth-termWidth)
			}

			// Check the help text line specifically
			lines := strings.Split(logView, "\n")
			for i, line := range lines {
				if strings.Contains(line, "find") || strings.Contains(line, "maximize") {
					w := lipgloss.Width(line)
					t.Logf("help text line %d: width=%d content=%q", i, w, line)
				}
			}

			if !strings.Contains(pageView, helpText) {
				for i := len(helpText); i > 0; i-- {
					if strings.Contains(pageView, helpText[:i]) {
						t.Errorf("help text truncated, found %q (missing %q)",
							helpText[:i], helpText[i:])
						break
					}
				}
			}

			// Now test with log focused (ActiveStyle) instead
			tasks.Blur()
			logvp.Focus()

			tasksView2 := tasks.View()
			logView2 := logvp.View()
			pageView2 := lipgloss.JoinHorizontal(lipgloss.Left, tasksView2, testSpacer, logView2)

			tasksWidth2 := lipgloss.Width(tasksView2)
			logWidth2 := lipgloss.Width(logView2)
			pageWidth2 := lipgloss.Width(pageView2)

			t.Logf("[log focused] tasksWidth=%d logWidth=%d total=%d",
				tasksWidth2, logWidth2, pageWidth2)

			if pageWidth2 > termWidth {
				t.Errorf("[log focused] page width %d exceeds terminal width %d (overflow by %d)",
					pageWidth2, termWidth, pageWidth2-termWidth)
			}

			if !strings.Contains(pageView2, helpText) {
				for i := len(helpText); i > 0; i-- {
					if strings.Contains(pageView2, helpText[:i]) {
						t.Errorf("[log focused] help text truncated, found %q (missing %q)",
							helpText[:i], helpText[i:])
						break
					}
				}
			}

			// Now test with page-level shorthelp JoinVertical (as PipelineRunPage.View does)
			shorthelp := bubbleshelp.New().View(pageHelpKeys{})
			// Clamp shorthelp to terminal width, as PipelineRunPage.View() does
			clampedHelp := lipgloss.NewStyle().MaxWidth(termWidth).Render(shorthelp)
			t.Logf("shorthelp width=%d clampedHelp width=%d", lipgloss.Width(shorthelp), lipgloss.Width(clampedHelp))

			fullPageView := lipgloss.JoinVertical(lipgloss.Top, pageView2, clampedHelp)
			fullPageWidth := lipgloss.Width(fullPageView)
			t.Logf("[full page with shorthelp] width=%d", fullPageWidth)

			if fullPageWidth > termWidth {
				t.Errorf("[full page] width %d exceeds terminal width %d (overflow by %d)",
					fullPageWidth, termWidth, fullPageWidth-termWidth)
			}

			if !strings.Contains(fullPageView, helpText) {
				for i := len(helpText); i > 0; i-- {
					if strings.Contains(fullPageView, helpText[:i]) {
						t.Errorf("[full page] help text truncated, found %q (missing %q)",
							helpText[:i], helpText[i:])
						break
					}
				}
			}
		})
	}
}
