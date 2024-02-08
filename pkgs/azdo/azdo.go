package azdo

import (
	"fmt"
	"os"
	"time"

	"explore-bubbletea/pkgs/searchableviewport"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ActiveSection int

const (
	ListSection ActiveSection = iota
	ViewportSection
)

var (
	activeStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#00ff00"))

	inactiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#6c6c6c"))

	pending   = lipgloss.NewStyle().SetString("•").Foreground(lipgloss.Color("#ffbf00"))
	succeeded = lipgloss.NewStyle().SetString("✓").Foreground(lipgloss.Color("#00ff00"))
	failed    = lipgloss.NewStyle().SetString("x").Foreground(lipgloss.Color("#ff0000"))
	skipped   = lipgloss.NewStyle().SetString(">").Foreground(lipgloss.Color("#ffffff"))
	symbolMap = map[string]interface{}{
		"pending":   pending,
		"succeeded": succeeded,
		"failed":    failed,
		"skipped":   skipped,
	}
)

type model struct {
	taskList        list.Model
	pipelineId      int
	pipelineState   pipelineState
	pipelineSpinner spinner.Model
	done            bool
	logViewPort     *searchableviewport.Model
	activeSection   ActiveSection
	azdoClient      *AzdoClient
}

func newModel() *model {
	height := 15
	vp := searchableviewport.New(80, height)
	pspinner := spinner.New()
	pspinner.Spinner = spinner.Dot
	pspinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a9ff"))
	tl := list.New([]list.Item{}, itemDelegate{}, 30, height)
	tl.Title = "Pipeline"
	tl.SetShowStatusBar(false)
	azdoclient := NewAzdoClient("rdalbuquerque", "explore-bubbletea", os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
	return &model{
		taskList:        tl,
		pipelineSpinner: pspinner,
		logViewPort:     vp,
		azdoClient:      azdoclient,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.pipelineSpinner.Tick, func() tea.Msg { return m.azdoClient.runOrFollowPipeline(false) })
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		case "tab":
			log2file("tab\n")
			if m.activeSection == ListSection {
				m.activeSection = ViewportSection
			} else {
				m.activeSection = ListSection
			}
			log2file(fmt.Sprintf("activeSection: %d\n", m.activeSection))
			return m, nil
		}
	case pipelineStateMsg:
		ps := pipelineState(msg)
		m.pipelineState = ps
		if ps.isRunning {
			m.pipelineState = pipelineState(msg)
			m.SetTaskList(ps)
			m.logViewPort.SetContent(m.taskList.SelectedItem().(item).desc)
			m.logViewPort.GotoBottom()
			return m, m.azdoClient.getPipelineState(m.pipelineId, 1*time.Second)
		}
		return m, nil
	case pipelineIdMsg:
		m.pipelineState.isRunning = true
		m.pipelineId = int(msg)
		return m, m.azdoClient.getPipelineState(int(msg), 0)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.pipelineSpinner, cmd = m.pipelineSpinner.Update(msg)
		m.SetTaskList(m.pipelineState)
		return m, cmd
	}
	var cmd tea.Cmd
	switch m.activeSection {
	case ListSection:
		log2file("ListSection\n")
		var selectedRecord item
		m.taskList, cmd = m.taskList.Update(msg)
		selectedRecord, ok := m.taskList.SelectedItem().(item)
		if !ok {
			return m, cmd
		}
		m.logViewPort.SetContent(selectedRecord.desc)
	case ViewportSection:
		m.logViewPort, cmd = m.logViewPort.Update(msg)
	}

	return m, cmd
}

func (m *model) View() string {
	var taskListView, logViewportView string
	if m.activeSection == ListSection {
		taskListView = activeStyle.Blink(true).Render(m.taskList.View())
		logViewportView = inactiveStyle.Render(m.logViewPort.View())
	} else {
		taskListView = inactiveStyle.Render(m.taskList.View())
		logViewportView = activeStyle.Render(m.logViewPort.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, taskListView, "  ", logViewportView)
}

// log2file logs to file logs.txt
func log2file(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s", timestamp, msg)

	f, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, err := f.WriteString(logMsg); err != nil {
		panic(err)
	}
}

func (m *model) formatStatusView(state, result, name, indent string) string {
	if state == "inProgress" {
		return fmt.Sprintf("%s%s %s", indent, m.pipelineSpinner.View(), name)
	} else if state == "completed" {
		return fmt.Sprintf("%s%s %s", indent, symbolMap[result], name)
	} else {
		return fmt.Sprintf("%s%s %s", indent, symbolMap[state], name)
	}
}

func main() {
	p := tea.NewProgram(newModel())
	if _, err := p.Run(); err != nil {
		panic(err)
	}
}
