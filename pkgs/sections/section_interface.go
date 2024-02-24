package sections

import (
	"fmt"
	"os"

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

func log2file(msg string) {
	f, err := os.OpenFile("sections-log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	if _, err := f.WriteString(msg + "\n"); err != nil {
		fmt.Println(err)
	}
}
