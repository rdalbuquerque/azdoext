package main

import (
	"context"
	"fmt"

	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/pages"
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	logger    *logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	pages     map[pages.PageName]pages.PageInterface
	pageStack pages.Stack
	height    int
	width     int
}

func initialModel() model {
	ctx, cancel := context.WithCancel(context.Background())
	logger := logger.NewLogger("main.log")
	helpPage := pages.NewHelpPage()
	gitPage := pages.NewGitPage()
	pipelineListPage := pages.NewPipelineListPage(ctx)
	pipelineRun := pages.NewPipelineRunPage(ctx)
	pagesMap := map[pages.PageName]pages.PageInterface{
		pages.Git:          gitPage,
		pages.Help:         helpPage,
		pages.PipelineList: pipelineListPage,
		pages.PipelineRun:  pipelineRun,
	}
	pageStack := pages.Stack{}
	m := model{
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		pages:     pagesMap,
		pageStack: pageStack,
	}
	m.addPage(pages.Git)
	return m
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		case "ctrl+h":
			if m.pageStack.Peek().GetPageName() != pages.Help {
				m.addPage(pages.Help)
			}
			return m, nil
		case "ctrl+b":
			m.removeCurrentPage()
			return m, nil
		case "ctrl+r":
			m.cancel()
			return restart()
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		for _, p := range m.pages {
			styles.SetDimensions(m.width, msg.Height-3)
			p.SetDimensions(0, msg.Height-3)
		}
		return m, func() tea.Msg { return sections.BroadcastGitInfoMsg(true) }
	case sections.SubmitChoiceMsg:
		m.logger.LogToFile("debug", fmt.Sprintf("choice received: %s", msg))
		switch listitems.OptionName(msg) {
		case sections.Options.GoToPipelines:
			m.addPage(pages.PipelineList)
		}
	case sections.GitPRCreatedMsg:
		m.logger.LogToFile("info", "PR created")
		m.addPage(pages.PipelineList)

	case sections.PipelineRunIdMsg:
		m.logger.LogToFile("info", fmt.Sprintf("received run id: %d", msg))
		m.addPage(pages.PipelineRun)
	}
	// update all pages
	updatedPages := make(map[pages.PageName]pages.PageInterface)
	var cmds []tea.Cmd
	for _, p := range m.pages {
		updatedPage, cmd := p.Update(msg)
		updatedPages[updatedPage.GetPageName()] = updatedPage
		cmds = append(cmds, cmd)
	}
	m.pages = updatedPages
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	return m.pageStack.Peek().View()
}

func (m *model) addPage(pageName pages.PageName) {
	if len(m.pageStack) > 0 {
		m.pageStack.Peek().UnsetCurrentPage()
	}
	p := m.pages[pageName]
	if p == nil {
		availablePages := make([]string, 0, len(m.pages))
		for k := range m.pages {
			availablePages = append(availablePages, string(k))
		}
		m.logger.LogToFile("error", fmt.Sprintf("page %s not found, available pages are: %s", pageName, availablePages))
		return
	}
	p.SetAsCurrentPage()
	m.pageStack.Push(p)
}

func (m *model) removeCurrentPage() {
	m.pageStack.Peek().UnsetCurrentPage()
	m.pageStack.Pop()
	m.pageStack.Peek().SetAsCurrentPage()
}

func restart() (*model, tea.Cmd) {
	model := initialModel()
	return &model, func() tea.Msg { return sections.BroadcastGitInfoMsg(true) }
}

func main() {
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
