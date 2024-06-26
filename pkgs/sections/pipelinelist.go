package sections

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"azdoext/pkgs/utils"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/build"
)

type buildsFetchedMsg []list.Item
type PipelineSelectedMsg listitems.PipelineItem

type PipelineListSection struct {
	project                 string
	repositoryId            uuid.UUID
	pipelineFetchingEnabled bool
	logger                  *logger.Logger
	pipelinelist            list.Model
	hidden                  bool
	focused                 bool
	ctx                     context.Context
	spinner                 spinner.Model
	spinnerView             *string
	buildclient             build.Client
}

func NewPipelineList(ctx context.Context) Section {
	logger := logger.NewLogger("pipelinelist.log")
	pipelinelist := list.New([]list.Item{}, listitems.ItemDelegate{}, 40, 0)
	pipelinelist.Title = "Pipelines"
	pipelinelist.SetShowStatusBar(false)
	spner := spinner.New()
	spner.Spinner = spinner.Dot
	spner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#00a9ff"))

	return &PipelineListSection{
		logger:       logger,
		pipelinelist: pipelinelist,
		ctx:          ctx,
		spinner:      spner,
		spinnerView:  utils.Ptr(spner.View()),
	}
}

func (p *PipelineListSection) IsHidden() bool {
	return p.hidden
}

func (p *PipelineListSection) IsFocused() bool {
	return p.focused
}

func (p *PipelineListSection) Hide() {
	p.hidden = true
}

func (p *PipelineListSection) Show() {
	p.hidden = false
}

func (p *PipelineListSection) Focus() {
	p.Show()
	p.focused = true
}

func (p *PipelineListSection) Blur() {
	p.focused = false
}

func (p *PipelineListSection) View() string {
	if p.focused {
		return styles.ActiveStyle.Render(p.pipelinelist.View())
	}
	return styles.InactiveStyle.Render(p.pipelinelist.View())
}

func (p *PipelineListSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case SubmitChoiceMsg:
		selectedPipeline := p.pipelinelist.SelectedItem().(listitems.PipelineItem)

		var runId int
		if listitems.OptionName(msg) == Options.GoToTasks {
			p.logger.LogToFile("info", fmt.Sprintf("selected pipeline: %s, with run id: %d", selectedPipeline.Name, selectedPipeline.Id))
			runId = selectedPipeline.RunId
		} else if listitems.OptionName(msg) == Options.RunPipeline {
			p.logger.LogToFile("info", fmt.Sprintf("selected pipeline: %s", selectedPipeline.Name))
			runId = p.runPipeline(p.ctx, selectedPipeline, p.project)
		}
		return p, func() tea.Msg { return PipelineRunIdMsg{RunId: runId, PipelineName: selectedPipeline.Name} }
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			selectedPipeline, ok := p.pipelinelist.SelectedItem().(listitems.PipelineItem)
			if ok {
				return p, func() tea.Msg { return PipelineSelectedMsg(selectedPipeline) }
			}
			p.logger.LogToFile("error", "selected item is not a pipeline item")
			return p, nil
		}
	case GitInfoMsg:
		azdoInfo := utils.ExtractAzdoInfo(msg.RemoteUrl)
		azdoconn := azuredevops.NewPatConnection(azdoInfo.OrgUrl, os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN"))
		buildclient, err := build.NewClient(p.ctx, azdoconn)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return p, nil
			}
			panic(err)
		}
		repoId := utils.GetRepositoryId(p.ctx, azdoconn, azdoInfo.Project, azdoInfo.RepositoryName)
		p.buildclient, p.project, p.repositoryId = buildclient, azdoInfo.Project, repoId
		return p, nil
	case GitPushedMsg:
		if p.pipelineFetchingEnabled {
			return p, nil
		}
		p.pipelineFetchingEnabled = true
		return p, tea.Batch(p.fetchBuilds(p.ctx, 0), p.spinner.Tick)
	case buildsFetchedMsg:
		p.pipelinelist.SetItems(msg)
		return p, p.fetchBuilds(p.ctx, 10*time.Second)
	case spinner.TickMsg:
		spinner, cmd := p.spinner.Update(msg)
		p.spinner = spinner
		*p.spinnerView = spinner.View()
		cmds = append(cmds, cmd)
	}
	pipelines, cmd := p.pipelinelist.Update(msg)
	cmds = append(cmds, cmd)
	p.pipelinelist = pipelines
	return p, tea.Batch(cmds...)
}

func (p *PipelineListSection) runPipeline(ctx context.Context, pipeline listitems.PipelineItem, project string) int {
	run, err := p.buildclient.QueueBuild(ctx, build.QueueBuildArgs{
		Project: &project,
		Build: &build.Build{
			Definition: &build.DefinitionReference{
				Id: &pipeline.Id,
			},
		},
	})
	if err != nil {
		p.logger.LogToFile("error", fmt.Sprintf("error while running pipeline: %s", err))
		return 0
	}
	return *run.Id
}

func (p *PipelineListSection) SetDimensions(width, height int) {
	p.pipelinelist.SetHeight(height)
}

func (p *PipelineListSection) fetchBuilds(ctx context.Context, wait time.Duration) tea.Cmd {
	return func() tea.Msg {
		p.logger.LogToFile("info", fmt.Sprintf("fetching builds of project %s and repository %s...", p.project, p.repositoryId))
		err := utils.SleepWithContext(ctx, wait)
		if err != nil {
			p.logger.LogToFile("error", fmt.Sprintf("error while waiting: %s", err))
			return buildsFetchedMsg(p.pipelinelist.Items())
		}
		definitions, err := p.buildclient.GetDefinitions(ctx, build.GetDefinitionsArgs{
			Project:        &p.project,
			RepositoryId:   utils.Ptr(p.repositoryId.String()),
			RepositoryType: utils.Ptr("TfsGit"),
		})
		if err != nil {
			p.logger.LogToFile("error", fmt.Sprintf("error while getting definitions: %s", err))
			return buildsFetchedMsg(p.pipelinelist.Items())
		}
		if len(definitions.Value) == 0 {
			p.logger.LogToFile("info", fmt.Sprintf("no pipelines found from repository id: %s and project: %s", p.repositoryId, p.project))
		}
		pipelineList := []list.Item{}
		for _, definition := range definitions.Value {
			status, runId := p.getBuildStatus(*definition.Id)
			pipelineList = append(pipelineList, listitems.PipelineItem{Name: *definition.Name, Status: status, Symbol: p.getSymbol(status), RunId: runId, Id: *definition.Id})
		}
		return buildsFetchedMsg(pipelineList)
	}
}

func (p *PipelineListSection) getBuildStatus(pipelineId int) (string, int) {
	builds, err := p.buildclient.GetBuilds(p.ctx, build.GetBuildsArgs{
		Definitions: &[]int{pipelineId},
		Project:     &p.project,
		Top:         utils.Ptr(1),
		QueryOrder:  &build.BuildQueryOrderValues.QueueTimeDescending,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", 0
		}
		panic(err)
	}
	buildsValue := (*builds).Value
	if len(buildsValue) == 0 {
		return "noRuns", 0
	}
	buildValue := buildsValue[0]
	status := buildValue.Status
	result := buildValue.Result
	return utils.StatusOrResult(status, result), *buildsValue[0].Id
}

func (p *PipelineListSection) getSymbol(status string) *string {
	if status == "inProgress" {
		return p.spinnerView
	} else {
		symbol := styles.SymbolMap[status].String()
		return &symbol
	}
}
