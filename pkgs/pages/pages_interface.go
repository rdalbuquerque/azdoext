package pages

import (
	"azdoext/pkgs/azdo"
	"azdoext/pkgs/sections"
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

type SectionConstructor func(context.Context) sections.Section

var (
	sectionNewFuncs = map[sections.SectionName]SectionConstructor{
		sections.Commit:        sections.NewCommitSection,
		sections.Worktree:      sections.NewWorktreeSection,
		sections.ChoiceSection: sections.NewChoice,
		sections.AzdoSection:   azdo.New,
		sections.OpenPR:        sections.NewPRSection,
		sections.HelpSection:   sections.NewHelp,
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
