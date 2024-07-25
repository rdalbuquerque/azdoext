package sections

import (
	"azdoext/pkgs/styles"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CommitSection struct {
	hidden            bool
	focused           bool
	title             string
	textarea          textarea.Model
	sectionIdentifier SectionName
}

func (cs *CommitSection) IsHidden() bool {
	return cs.hidden
}

func (cs *CommitSection) IsFocused() bool {
	return cs.focused
}

func NewCommitSection(secid SectionName) Section {
	title := "Git commit:"
	textarea := textarea.New()
	return &CommitSection{
		title:             title,
		textarea:          textarea,
		sectionIdentifier: secid,
	}
}

func (cs *CommitSection) GetSectionIdentifier() SectionName {
	return cs.sectionIdentifier
}

func (cs *CommitSection) SetDimensions(width, height int) {
	cs.textarea.SetWidth(styles.DefaultSectionWidth)
	cs.textarea.SetHeight(height - 1)
}

func (cs *CommitSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	if cs.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if cs.focused {
				switch msg.String() {
				case "ctrl+s":
					cs.textarea.Blur()
					return cs, func() tea.Msg { return commitMsg(cs.textarea.Value()) }
				}
			}
		}
		ta, cmd := cs.textarea.Update(msg)
		cs.textarea = ta
		return cs, cmd
	}
	return cs, nil
}

func (cs *CommitSection) View() string {
	if !cs.hidden {
		if cs.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, cs.title, cs.textarea.View()))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, cs.title, cs.textarea.View()))
	}
	return ""
}

func (cs *CommitSection) Hide() {
	cs.hidden = true
}

func (cs *CommitSection) Show() {
	cs.hidden = false
}

func (cs *CommitSection) Focus() {
	cs.textarea.Focus()
	cs.focused = true
}

func (cs *CommitSection) Blur() {
	cs.textarea.Blur()
	cs.focused = false
}

type commitMsg string
