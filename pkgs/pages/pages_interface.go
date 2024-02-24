package pages

import (
	"explore-bubbletea/pkgs/azdo"
	"explore-bubbletea/pkgs/sections"

	tea "github.com/charmbracelet/bubbletea"
)

type SectionConstructor func() sections.Section

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
	AddSection(sections.SectionName)
	SetDimensions(width, height int)
	SwitchSection()
	Update(tea.Msg) (PageInterface, tea.Cmd)
	View() string
}
