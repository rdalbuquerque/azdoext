package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"azdoext/pkgs/azdo"
	"azdoext/pkgs/gitexec"
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/pages"
	"azdoext/pkgs/sections"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/teamsg"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	logger    *logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	pages     map[pages.PageName]pages.PageInterface
	pageStack pages.Stack
	height    int
	width     int
	spinner   spinner.Model
}

var version string

var azdoextLogo = `
  __  ____ ____  __ ____ _  _ ____ 
 / _\(__  (    \/  (  __( \/ (_  _)
/    \/ _/ ) D (  O ) _) )  (  )(  
\_/\_(____(____/\__(____(_/\_)(__) 
`

func initialModel() model {
	ctx, cancel := context.WithCancel(context.Background())
	spnr := spinner.New()
	spnr.Spinner = spinner.Line
	spnr.Style = styles.SpinnerStyle

	logger := logger.NewLogger("main.log")
	helpPage := pages.NewHelpPage()
	pagesMap := map[pages.PageName]pages.PageInterface{
		pages.Help: helpPage,
	}
	pageStack := pages.Stack{}
	m := model{
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
		pages:     pagesMap,
		pageStack: pageStack,
		spinner:   spnr,
	}
	return m
}

func getAzdoConfig() tea.Cmd {
	return func() tea.Msg {
		gitconf := gitexec.Config()
		azdoconfig := azdo.GetAzdoConfig(gitconf.Origin, gitconf.CurrentBranch)
		return teamsg.AzdoConfigMsg(azdoconfig)
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(getAzdoConfig(), m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		spnr, cmd := m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		m.spinner = spnr
	case teamsg.AzdoConfigMsg:
		buildclient := azdo.NewBuildClient(m.ctx, msg.OrgUrl, msg.ProjectId, msg.PAT)
		gitclient := azdo.NewGitClient(m.ctx, msg.OrgUrl, msg.ProjectId, msg.PAT)
		gitpage := pages.NewGitPage(m.ctx, gitclient, azdo.Config(msg))
		pipelistpage := pages.NewPipelineListPage(m.ctx, buildclient, azdo.Config(msg))
		pipelinetaskpage := pages.NewPipelineRunPage(m.ctx, buildclient, azdo.Config(msg))
		m.pages[pages.Git] = gitpage
		m.pages[pages.PipelineList] = pipelistpage
		m.pages[pages.PipelineRun] = pipelinetaskpage
		m.addPage(pages.Git)
		return m, nil
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
		return m, nil
	case teamsg.SubmitChoiceMsg:
		m.logger.LogToFile("debug", fmt.Sprintf("choice received: %s", msg))
		switch listitems.OptionName(msg) {
		case sections.Options.GoToPipelines:
			m.addPage(pages.PipelineList)
		}
	case teamsg.NothingToCommitMsg:
		m.logger.LogToFile("info", "nothing to commit")
		m.addPage(pages.PipelineList)

	case teamsg.GitPRCreatedMsg:
		m.logger.LogToFile("info", "PR created")
		m.addPage(pages.PipelineList)

	case teamsg.PipelineRunIdMsg:
		m.logger.LogToFile("info", fmt.Sprintf("received run id: %d", msg.RunId))
		m.addPage(pages.PipelineRun)
	}
	// update all pages
	updatedPages := make(map[pages.PageName]pages.PageInterface)
	for _, p := range m.pages {
		updatedPage, cmd := p.Update(msg)
		updatedPages[updatedPage.GetPageName()] = updatedPage
		cmds = append(cmds, cmd)
	}
	m.pages = updatedPages
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	loadingStr := lipgloss.NewStyle().Bold(true).Render("Loading...")
	spnerWithLoading := lipgloss.NewStyle().Bold(true).Padding(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), " ", loadingStr))
	if len(m.pageStack) == 0 {
		return lipgloss.JoinVertical(lipgloss.Top, styles.LogoStyle.Render(azdoextLogo), spnerWithLoading)
	}
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
	return &model, model.Init()
}

func main() {
	versionFlag := flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	initialModel := initialModel()
	if _, err := tea.NewProgram(&initialModel).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
