package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"azdoext/pkg/azdo"
	"azdoext/pkg/gitexec"
	"azdoext/pkg/listitems"
	"azdoext/pkg/logger"
	"azdoext/pkg/pages"
	"azdoext/pkg/sections"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	logger       *logger.Logger
	initError    string
	ctx          context.Context
	cancel       context.CancelFunc
	pages        map[pages.PageName]pages.PageInterface
	pageStack    pages.Stack
	height       int
	width        int
	spinner      spinner.Model
	authProvider azdo.AuthProvider
}

var version string

var azdoextLogo = `
  __  ____ ____  __ ____ _  _ ____ 
 / _\(__  (    \/  (  __( \/ (_  _)
/    \/ _/ ) D (  O ) _) )  (  )(  
\_/\_(____(____/\__(____(_/\_)(__) 
`

func initialModel(authProvider azdo.AuthProvider) model {
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
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		pages:        pagesMap,
		pageStack:    pageStack,
		spinner:      spnr,
		authProvider: authProvider,
	}
	return m
}

func getAzdoConfig(authProvider azdo.AuthProvider) tea.Cmd {
	return func() tea.Msg {
		gitconf, err := gitexec.Config()
		if err != nil {
			return teamsg.AzdoConfigErrorMsg(err)
		}
		azdoconfig, err := azdo.GetAzdoConfig(context.Background(), gitconf.Origin, gitconf.CurrentBranch, authProvider)
		if err != nil {
			return teamsg.AzdoConfigErrorMsg(err)
		}
		return teamsg.AzdoConfigMsg(azdoconfig)
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(getAzdoConfig(m.authProvider), m.spinner.Tick)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		spnr, cmd := m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		m.spinner = spnr
	case teamsg.AzdoConfigErrorMsg:
		m.initError = msg.Error()
		return m, nil
	case teamsg.AzdoConfigMsg:
		buildclient := azdo.NewBuildClient(m.ctx, msg.OrgUrl, msg.ProjectId, msg.AuthHeader)

		gitclient := azdo.NewGitClient(m.ctx, msg.OrgUrl, msg.ProjectId, msg.AuthHeader)
		gitpage := pages.NewGitPage(m.ctx, gitclient, azdo.Config(msg))
		pipelistpage := pages.NewPipelineListPage(m.ctx, buildclient, azdo.Config(msg))
		pipelinetaskpage := pages.NewPipelineRunPage(m.ctx, buildclient, azdo.Config(msg))
		m.pages[pages.Git] = gitpage
		m.pages[pages.PipelineList] = pipelistpage
		m.pages[pages.PipelineRun] = pipelinetaskpage
		m.addPage(pages.Git)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if m.initError != "" {
				m.cancel()
				return m, tea.Quit
			}
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

func (m *model) View() tea.View {
	if m.initError != "" {
		return tea.NewView(lipgloss.NewStyle().Foreground(styles.Red).Bold(true).Render(m.initError) + "\n\nPress 'enter' or 'ctrl+c' to exit")
	}
	loadingStr := lipgloss.NewStyle().Bold(true).Render("Loading...")

	spnerWithLoading := lipgloss.NewStyle().Bold(true).Padding(1).Render(lipgloss.JoinHorizontal(lipgloss.Left, m.spinner.View(), " ", loadingStr))

	if len(m.pageStack) == 0 {
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Top, styles.LogoStyle.Render(azdoextLogo), spnerWithLoading))
	}
	return tea.NewView(m.pageStack.Peek().View())
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
	if len(m.pageStack) == 1 {
		return
	}
	m.pageStack.Peek().UnsetCurrentPage()
	m.pageStack.Pop()
	m.pageStack.Peek().SetAsCurrentPage()
}

func restart() (*model, tea.Cmd) {
	authProvider := resolveAuth()
	model := initialModel(authProvider)
	return &model, model.Init()
}

// resolveAuth discovers the git remote and resolves authentication before the TUI starts.
// This ensures interactive prompts (device code) are visible to the user.
func resolveAuth() azdo.AuthProvider {
	gitconf, err := gitexec.Config()
	if err != nil {
		// Return a provider that always errors; the TUI will show the error
		return func(ctx context.Context) (string, error) {
			return "", err
		}
	}
	orgUrl := azdo.GetOrgUrl(gitconf.Origin)
	authProvider, err := azdo.NewAuthProvider(orgUrl)
	if err != nil {
		return func(ctx context.Context) (string, error) {
			return "", err
		}
	}
	return authProvider
}

func main() {
	versionFlag := flag.Bool("version", false, "Print the version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	authProvider := resolveAuth()
	m := initialModel(authProvider)
	if _, err := tea.NewProgram(&m).Run(); err != nil {
		fmt.Println("Error running program:", err)
	}
}
