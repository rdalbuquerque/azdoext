package main

import (
	"fmt"

	"explore-bubbletea/pkgs/azdo"
	"explore-bubbletea/pkgs/pages"
	"explore-bubbletea/pkgs/sections"
	"explore-bubbletea/pkgs/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	pages     map[pages.PageName]pages.PageInterface
	pageStack pages.Stack
	height    int
	width     int
}

func initialModel() model {
	helpPage := pages.NewHelpPage()
	gitPage := pages.NewGitPage()
	azdoPage := pages.NewAzdoPage()
	pagesMap := map[pages.PageName]pages.PageInterface{
		pages.Git:       gitPage,
		pages.Pipelines: azdoPage,
		pages.Help:      helpPage,
	}
	pageStack := pages.Stack{}
	m := model{
		pages:     pagesMap,
		pageStack: pageStack,
	}
	m.addPage(pages.Git)
	return m
}

func (m *model) Init() tea.Cmd {
	curPage := m.pageStack.Peek()
	if curPage.GetPageName() == pages.Git {
		_, cmd := m.pageStack.Peek().Update(sections.BroadcastGitInfoMsg(true))
		return cmd
	}
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	f, err := tea.LogToFile("main-update.txt", "debug")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+h":
			if m.pageStack.Peek().GetPageName() != pages.Help {
				m.addPage(pages.Help)
			}
			return m, nil
		case "ctrl+b":
			m.removeCurrentPage()
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		for _, p := range m.pages {
			styles.SetDimensions(m.width, msg.Height-3)
			p.SetDimensions(0, msg.Height-3)
		}
		return m, nil
	case sections.SubmitChoiceMsg:
		if msg == sections.SubmitChoiceMsg(sections.PipelineOption) {
			m.addPage(pages.Pipelines)
			return m, func() tea.Msg { return azdo.GoToPipelinesMsg(true) }
		}
	case azdo.PROpenedMsg:
		m.addPage(pages.Pipelines)
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
	p := m.pages[pageName]
	p.SetAsCurrentPage()
	m.pageStack.Push(p)
}

func (m *model) removeCurrentPage() {
	m.pageStack.Peek().UnsetCurrentPage()
	m.pageStack.Pop()
}

func main() {
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
