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
	Commit        SectionName = "commit"
	Worktree      SectionName = "worktree"
	ChoiceSection SectionName = "choice"
	AzdoSection   SectionName = "azdoSection"
	OpenPR        SectionName = "openPR"
	HelpSection   SectionName = "help"
)
