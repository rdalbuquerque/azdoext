package pages

import (
	"azdoext/pkgs/sections"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type SectionConstructor func(context.Context) sections.Section

var (
	sectionNewFuncs = map[sections.SectionName]SectionConstructor{
		sections.PrOrPipelineChoice:   sections.NewChoice,
		sections.PipelineActionChoice: sections.NewChoice,
		sections.Commit:               sections.NewCommitSection,
		sections.Worktree:             sections.NewWorktreeSection,
		sections.OpenPR:               sections.NewPRSection,
		sections.HelpSection:          sections.NewHelp,
		sections.PipelineTasks:        sections.NewPipelineTasks,
		sections.PipelineList:         sections.NewPipelineList,
		sections.LogViewport:          sections.NewLogViewport,
	}
)

type PageInterface interface {
	GetPageName() PageName
	AddSection(context.Context, sections.SectionName)
	SetDimensions(width, height int)
	Update(tea.Msg) (PageInterface, tea.Cmd)
	View() string
	IsCurrentPage() bool
	SetAsCurrentPage()
	UnsetCurrentPage()
}
