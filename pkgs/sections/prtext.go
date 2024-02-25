package sections

import (
	"explore-bubbletea/pkgs/styles"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PRSection struct {
	hidden   bool
	focused  bool
	title    string
	textarea textarea.Model
}

func (pr *PRSection) IsHidden() bool {
	return pr.hidden
}

func (pr *PRSection) IsFocused() bool {
	return pr.focused
}

func NewPRSection() Section {
	title := "Open PR:"
	textarea := textarea.New()
	textarea.SetHeight(styles.ActiveStyle.GetHeight() - 2)
	textarea.Placeholder = "Title and description"
	textarea.SetPromptFunc(6, func(i int) string {
		if i == 0 {
			return "Title:"
		} else {
			return " Desc:"
		}
	})
	return &PRSection{
		title:    title,
		textarea: textarea,
	}
}

func (pr *PRSection) SetDimensions(width, height int) {
	pr.textarea.SetWidth(styles.DefaultSectionWidth)
	pr.textarea.SetHeight(height - 1)
}

func (pr *PRSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if pr.textarea.Focused() {
				pr.textarea.Blur()
			}
			return pr, func() tea.Msg { return SubmitPRMsg(pr.textarea.Value()) }
		}
	case PRErrorMsg:
		pr.textarea.Placeholder = string(msg)
	}
	ta, cmd := pr.textarea.Update(msg)
	pr.textarea = ta
	return pr, cmd
}

func (pr *PRSection) View() string {
	if !pr.hidden {
		if pr.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pr.title, pr.textarea.View()))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, pr.title, pr.textarea.View()))
	}
	return ""
}

func (pr *PRSection) Hide() {
	pr.hidden = true
}

func (pr *PRSection) Show() {
	pr.hidden = false
}

func (pr *PRSection) Focus() {
	pr.textarea.Focus()
	pr.focused = true
}

func (pr *PRSection) Blur() {
	pr.textarea.Blur()
	pr.focused = false
}

type SubmitPRMsg string
type PRErrorMsg string
