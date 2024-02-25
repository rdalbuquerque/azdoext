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
	pages      map[pages.PageName]pages.PageInterface
	pagesStack pages.Stack
	height     int
	width      int
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
	pageStack.Push(pagesMap[pages.Git])
	return model{
		pages:      pagesMap,
		pagesStack: pageStack,
	}
}

func (m *model) Init() tea.Cmd {
	curPage := m.pagesStack.Peek()
	if curPage.GetPageName() == pages.Git {
		_, cmd := m.pagesStack.Peek().Update(sections.BroadcastGitInfoMsg(true))
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
			m.pagesStack.Push(m.pages[pages.Help])
			return m, nil
		case "ctrl+b":
			m.pagesStack.Pop()
			return m, nil
		case "tab":
			m.pagesStack.Peek().SwitchSection()
			return m, nil
		case "enter":
			_, cmd := m.pagesStack.Peek().Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		for _, p := range m.pages {
			styles.SetDimensions(m.width, msg.Height-3)
			p.SetDimensions(0, msg.Height-3)
		}
		return m, nil
	case sections.GitInfoMsg:
		pipelinePage, cmd := m.pages[pages.Pipelines].Update(msg)
		m.pages[pages.Pipelines] = pipelinePage
		return m, cmd
	case sections.SubmitChoiceMsg:
		if msg == sections.SubmitChoiceMsg(sections.PipelineOption) {
			m.pagesStack.Push(m.pages[pages.Pipelines])
			return m, func() tea.Msg { return azdo.GoToPipelinesMsg(true) }
		}
	case azdo.PROpenedMsg:
		m.pagesStack.Push(m.pages[pages.Pipelines])
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
	return m.pagesStack.Peek().View()
}

func main() {
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
