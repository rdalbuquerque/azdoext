package sections

import (
	"azdoext/pkgs/styles"
	"azdoext/pkgs/teamsg"

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
	pushed            bool
	pushInProgress    bool
	help              string
}

func (cs *CommitSection) IsHidden() bool {
	return cs.hidden
}

func (cs *CommitSection) IsFocused() bool {
	return cs.focused
}

func NewCommitSection(secid SectionName) Section {
	title := styles.TitleStyle.Render("Git commit:")
	styledHelpText := styles.ShortHelpStyle.Render("ctrl+s save and push")
	textarea := textarea.New()
	return &CommitSection{
		title:             title,
		textarea:          textarea,
		sectionIdentifier: secid,
		help:              styledHelpText,
	}
}

func (cs *CommitSection) GetSectionIdentifier() SectionName {
	return cs.sectionIdentifier
}

func (cs *CommitSection) SetDimensions(width, height int) {
	cs.textarea.SetWidth(styles.DefaultSectionWidth)
	cs.textarea.SetHeight(height - 4)
}

func (cs *CommitSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg.(type) {
	case teamsg.GitPushedMsg:
		cs.pushed = true
		cs.pushInProgress = false
		return cs, nil
	}
	if cs.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+s":
				if cs.pushed || cs.pushInProgress {
					cs.textarea.InsertString("\n")
					cs.textarea.FocusedStyle.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Yellow))
					cs.textarea.BlurredStyle.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color(styles.Yellow))
					cs.textarea.InsertString("Already pushing or pushed...")
					return cs, nil
				}
				cs.pushInProgress = true
				cs.textarea.Blur()
				return cs, func() tea.Msg { return teamsg.CommitMsg(cs.textarea.Value()) }
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
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, cs.title, "", cs.textarea.View(), "", cs.help))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, cs.title, "", cs.textarea.View(), "", cs.help))
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
