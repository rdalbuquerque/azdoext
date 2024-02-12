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
	noRuns             = lipgloss.NewStyle().SetString("■").Foreground(lipgloss.Color("#808080"))
	pending            = lipgloss.NewStyle().SetString("⊛").Foreground(lipgloss.Color("#ffbf00"))
	succeeded          = lipgloss.NewStyle().SetString("✔").Foreground(lipgloss.Color("#00ff00"))
	failed             = lipgloss.NewStyle().SetString("✖").Foreground(lipgloss.Color("#ff0000"))
	skipped            = lipgloss.NewStyle().SetString("➤").Foreground(lipgloss.Color("#ffffff"))
	partiallySucceeded = lipgloss.NewStyle().SetString("⚠").Foreground(lipgloss.Color("#ffbf00"))
	canceled           = lipgloss.NewStyle().SetString("⊝").Foreground(lipgloss.Color("#ffbf00"))
	symbolMap          = map[string]lipgloss.Style{
		"pending":            pending,
		"succeeded":          succeeded,
		"failed":             failed,
		"skipped":            skipped,
		"noRuns":             noRuns,
		"partiallySucceeded": partiallySucceeded,
		"canceled":           canceled,
	}
)

type Model struct {
	TaskList                 list.Model
	pipelineId               int
	PipelineState            pipelineState
	pipelineSpinner          spinner.Model
	done                     bool
	logViewPort              *searchableviewport.Model
	activeSection            ActiveSection
	Client                   *AzdoClient
	PipelineList             list.Model
	azdoClient               *AzdoClient
	RunOrFollowList          list.Model
	RunOrFollowChoiceEnabled bool
}

func New(org, project, pat string) *Model {
	vp := searchableviewport.New(0, 0)
	pspinner := spinner.New()
	pspinner.Spinner = spinner.Dot
	pspinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a9ff"))
	tl := list.New([]list.Item{}, itemDelegate{}, 30, 0)
	tl.SetShowStatusBar(false)
	azdoclient := NewAzdoClient(org, project, pat)
	pipelineList := list.New([]list.Item{}, itemDelegate{}, 30, 0)
	pipelineList.Title = "Pipelines"
	pipelineList.SetShowStatusBar(false)
	runOrFollowList := list.New([]list.Item{PipelineItem{Title: "Run"}, PipelineItem{Title: "Follow"}}, itemDelegate{}, 30, 0)
	runOrFollowList.Title = "Run new or follow?"
	return &Model{
		TaskList:        tl,
		pipelineSpinner: pspinner,
		logViewPort:     vp,
		azdoClient:      azdoclient,
		PipelineList:    pipelineList,
		RunOrFollowList: runOrFollowList,
	}
}

func (m *Model) SetHeights(height int) *Model {
	m.TaskList.SetHeight(height)
	m.logViewPort.SetDimensions(80, height)
	m.PipelineList.SetHeight(height)
	m.RunOrFollowList.SetHeight(height)
	return m
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
			return m, nil
		case tea.KeyEnter:
			if m.RunOrFollowChoiceEnabled {
				m.RunOrFollowChoiceEnabled = false
				runOrFollow := m.RunOrFollowList.SelectedItem().(PipelineItem).Title
				selectedPipelineId := m.PipelineList.SelectedItem().(PipelineItem).Desc.(int)
				if runOrFollow == "Run" {
					return m, func() tea.Msg { return m.RunOrFollowPipeline(selectedPipelineId, true) }
				} else {
					return m, func() tea.Msg { return m.RunOrFollowPipeline(selectedPipelineId, false) }
				}
			}
			if m.activeSection == PipelineListSection {
				selectedPipeline := m.PipelineList.SelectedItem().(PipelineItem)
				if selectedPipeline.Status != "completed" {
					m.RunOrFollowChoiceEnabled = true
					return m, nil
				}
				m.TaskList.Title = selectedPipeline.Title
				return m, func() tea.Msg { return m.RunOrFollowPipeline(selectedPipeline.Desc.(int), false) }
			}
		case tea.KeyBackspace:
			if m.activeSection != ViewportSection {
				log2file("backspace\n")
				if m.RunOrFollowChoiceEnabled {
					m.RunOrFollowChoiceEnabled = false
					return m, nil
				}
				if m.activeSection == TaskListSection && !m.TaskList.SettingFilter() {
					m.activeSection = PipelineListSection
				}
				return m, nil
			}
		default:
			log2file(fmt.Sprintf("default: %v\n", msg))
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
		m.PipelineList.SetItems(msg)
		return m, tea.Batch(m.FetchPipelines(1*time.Second), m.pipelineSpinner.Tick)
	case PipelineIdMsg:
		m.PipelineState.IsRunning = true
		m.activeSection = TaskListSection
		m.pipelineId = int(msg)
		return m, m.azdoClient.getPipelineState(int(msg), 0)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.pipelineSpinner, cmd = m.pipelineSpinner.Update(msg)
		m.SetTaskList(m.PipelineState)
		m.SetPipelineList()
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
		if m.RunOrFollowChoiceEnabled {
			m.RunOrFollowList, cmd = m.RunOrFollowList.Update(msg)
			return m, cmd
		} else {
			pipelineList, cmd := m.PipelineList.Update(msg)
			m.PipelineList = pipelineList
			return m, cmd
		}
	}
	return m, nil
}

func (m *Model) View() string {
	var taskListView, logViewportView, pipelineListView string
	switch m.activeSection {
	case PipelineListSection:
		if m.RunOrFollowChoiceEnabled {
			runOrFollowView := ActiveStyle.Render(m.RunOrFollowList.View())
			pipelineListView = InactiveStyle.Render(m.PipelineList.View())
			return lipgloss.JoinHorizontal(lipgloss.Left, pipelineListView, "  ", runOrFollowView)
		}
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

func (m *Model) formatStatusView(obj map[string]interface{}, name, indent string) string {
	symbol := m.getSymbol(obj)
	return fmt.Sprintf("%s%s %s", indent, symbol, name)
}

func (m *Model) getSymbol(obj map[string]interface{}) string {
	log2file(fmt.Sprintf("obj: %v\n", obj))
	status, ok := obj["status"].(string)
	if !ok {
		status = ""
	}
	result, ok := obj["result"].(string)
	if !ok {
		result = ""
	}
	if status == "inProgress" {
		return m.pipelineSpinner.View()
	} else if status == "completed" {
		return symbolMap[result].String()
	} else {
		return symbolMap[status].String()
	}
}
