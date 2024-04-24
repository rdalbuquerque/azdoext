package sections

import (
	tea "github.com/charmbracelet/bubbletea"
)

type Section interface {
	IsHidden() bool
	IsFocused() bool
	Hide()
	Show()
	Focus()
	Blur()
	View() string
	Update(msg tea.Msg) (Section, tea.Cmd)
	SetDimensions(width, height int)
}

type SectionName string

const (
	PrOrPipelineChoice   SectionName = "prOrPipelineChoice"
	PipelineActionChoice SectionName = "pipelineActionChoice"
	Commit               SectionName = "commit"
	Worktree             SectionName = "worktree"
	AzdoSection          SectionName = "azdoSection"
	OpenPR               SectionName = "openPR"
	HelpSection          SectionName = "help"
	PipelineTasks        SectionName = "pipelineTasks"
	LogViewport          SectionName = "logviewport"
	PipelineList         SectionName = "pipelineList"
)
