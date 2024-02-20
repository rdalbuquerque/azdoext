package sections

import (
	"explore-bubbletea/pkgs/listitems"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Choice struct {
	hidden  bool
	focused bool
	choices list.Model
}

func NewChoice() Section {
	choices := list.New([]list.Item{
		listitems.ChoiceItem{Choice: "Open PR"},
		listitems.ChoiceItem{Choice: "Go to pipeline"},
	}, listitems.ChoiceItemDelegate{}, 0, 0)
	choices.Title = "PR or pipelines:"
	choices.SetHeight(ActiveStyle.GetHeight() - 2)
	choices.SetShowTitle(false)
	return &Choice{
		hidden:  false,
		focused: true,
		choices: choices,
	}
}

func (c *Choice) SetDimensions(width, height int) {
	c.choices.SetWidth(40)
	c.choices.SetHeight(height - 3)
}

func (c *Choice) IsHidden() bool {
	return c.hidden
}

func (c *Choice) IsFocused() bool {
	return c.focused
}

func (c *Choice) Update(msg tea.Msg) (Section, tea.Cmd) {
	if c.focused {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				return c, func() tea.Msg { return SubmitChoiceMsg(c.choices.SelectedItem().(listitems.ChoiceItem).Choice) }
			}
		}
		choice, cmd := c.choices.Update(msg)
		c.choices = choice
		return c, cmd
	}
	return c, nil
}

func (c *Choice) View() string {
	if !c.hidden {
		if c.focused {
			return ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, c.choices.Title, c.choices.View()))
		}
		return InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, c.choices.Title, c.choices.View()))
	}
	return ""
}

func (c *Choice) Hide() {
	c.hidden = true
}

func (c *Choice) Show() {
	c.hidden = false
}

func (c *Choice) Focus() {
	c.focused = true
}

func (c *Choice) Blur() {
	c.focused = false
}

type SubmitChoiceMsg string
