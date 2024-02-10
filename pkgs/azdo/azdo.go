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
	PipelineListSection ActiveSection = iota
	TaskListSection
	ViewportSection
)

var (
	ActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, true, false).
			BorderForeground(lipgloss.Color("#00ff00"))

	InactiveStyle = lipgloss.NewStyle().
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

type Model struct {
	TaskList        list.Model
	pipelineId      int
	PipelineState   pipelineState
	pipelineSpinner spinner.Model
	done            bool
	logViewPort     *searchableviewport.Model
	activeSection   ActiveSection
	Client          *AzdoClient
	PipelineList    list.Model
	azdoClient      *AzdoClient
}

func New(org, project, pat string) *Model {
	height := 15
	vp := searchableviewport.New(80, height)
	pspinner := spinner.New()
	pspinner.Spinner = spinner.Dot
	pspinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a9ff"))
	tl := list.New([]list.Item{}, itemDelegate{}, 30, height)
	tl.SetShowStatusBar(false)
	azdoclient := NewAzdoClient(org, project, pat)
	pipelineList := list.New([]list.Item{}, itemDelegate{}, 30, height)
	pipelineList.Title = "Pipelines"
	pipelineList.SetShowStatusBar(false)
	return &Model{
		TaskList:        tl,
		pipelineSpinner: pspinner,
		logViewPort:     vp,
		azdoClient:      azdoclient,
		PipelineList:    pipelineList,
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.done = true
			return m, tea.Quit
		case tea.KeyTab:
			log2file("tab\n")
			if m.activeSection == TaskListSection {
				m.activeSection = ViewportSection
			} else {
				m.activeSection = TaskListSection
			}
			log2file(fmt.Sprintf("activeSection: %d\n", m.activeSection))
			return m, nil
		case tea.KeyEnter:
			log2file("enter\n")
			log2file(fmt.Sprintf("activeSection: %d\n", m.activeSection))
			if m.activeSection == PipelineListSection {
				selectedPipeline, ok := m.PipelineList.SelectedItem().(PipelineItem)
				if !ok {
					return m, nil
				}
				m.TaskList.Title = selectedPipeline.Title
				return m, func() tea.Msg { return m.RunOrFollowPipeline(selectedPipeline.Desc.(int), false) }
			}
		case tea.KeyBackspace:
			if m.activeSection == TaskListSection && !m.TaskList.SettingFilter() {
				m.activeSection = PipelineListSection
			}
			return m, nil
		}
	case PipelineStateMsg:
		ps := pipelineState(msg)
		m.PipelineState = ps
		if ps.IsRunning {
			m.PipelineState = pipelineState(msg)
			m.SetTaskList(ps)
			m.logViewPort.SetContent(m.TaskList.SelectedItem().(PipelineItem).Desc.(string))
			m.logViewPort.GotoBottom()
			return m, m.azdoClient.getPipelineState(m.pipelineId, 1*time.Second)
		}
		return m, nil
	case PipelinesFetchedMsg:
		log2file("pipelinesFetchedMsg\n")
		log2file(fmt.Sprintf("msg: %v\n", msg))
		m.PipelineList.SetItems(msg)
		return m, m.pipelineSpinner.Tick
	case PipelineIdMsg:
		m.PipelineState.IsRunning = true
		m.activeSection = TaskListSection
		m.pipelineId = int(msg)
		return m, m.azdoClient.getPipelineState(int(msg), 0)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.pipelineSpinner, cmd = m.pipelineSpinner.Update(msg)
		m.SetTaskList(m.PipelineState)
		return m, cmd
	}
	var cmd tea.Cmd
	switch m.activeSection {
	case TaskListSection:
		log2file("ListSection\n")
		var selectedRecord PipelineItem
		m.TaskList, cmd = m.TaskList.Update(msg)
		selectedRecord, ok := m.TaskList.SelectedItem().(PipelineItem)
		if !ok {
			return m, cmd
		}
		m.logViewPort.SetContent(selectedRecord.Desc.(string))
	case ViewportSection:
		m.logViewPort, cmd = m.logViewPort.Update(msg)
		return m, cmd
	default:
		pipelineList, cmd := m.PipelineList.Update(msg)
		m.PipelineList = pipelineList
		return m, cmd
	}
	return m, nil
}

func (m *Model) View() string {
	var taskListView, logViewportView, pipelineListView string
	switch m.activeSection {
	case PipelineListSection:
		pipelineListView = ActiveStyle.Render(m.PipelineList.View())
		return pipelineListView
	case TaskListSection:
		taskListView = ActiveStyle.Render(m.TaskList.View())
		logViewportView = InactiveStyle.Render(m.logViewPort.View())
		return lipgloss.JoinHorizontal(lipgloss.Left, taskListView, "  ", logViewportView)
	case ViewportSection:
		taskListView = InactiveStyle.Render(m.TaskList.View())
		logViewportView = ActiveStyle.Render(m.logViewPort.View())
		return lipgloss.JoinHorizontal(lipgloss.Left, taskListView, "  ", logViewportView)
	}
	return ""
}

// log2file logs to file logs.txt
func log2file(msg string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMsg := fmt.Sprintf("[%s] %s", timestamp, msg)

	f, err := os.OpenFile("azdo-logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if _, err := f.WriteString(logMsg); err != nil {
		panic(err)
	}
}

func (m *Model) formatStatusView(state, result, name, indent string) string {
	if state == "inProgress" {
		return fmt.Sprintf("%s%s %s", indent, m.pipelineSpinner.View(), name)
	} else if state == "completed" {
		return fmt.Sprintf("%s%s %s", indent, symbolMap[result], name)
	} else {
		return fmt.Sprintf("%s%s %s", indent, symbolMap[state], name)
	}
}
