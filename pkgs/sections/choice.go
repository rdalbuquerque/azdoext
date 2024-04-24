package sections

import (
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/logger"
	"azdoext/pkgs/styles"
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OptionsMsg []list.Item

type Choice struct {
	logger  *logger.Logger
	hidden  bool
	focused bool
	choice  list.Model
}

func NewChoice(_ context.Context) Section {
	logger := logger.NewLogger("choice.log")

	choice := list.New([]list.Item{}, listitems.ChoiceItemDelegate{}, 0, 0)
	choice.Title = "PR or pipelines:"
	choice.SetHeight(styles.ActiveStyle.GetHeight() - 2)
	choice.SetShowTitle(false)
	return &Choice{
		logger: logger,
		choice: choice,
	}
}

func (c *Choice) SetDimensions(width, height int) {
	c.choice.SetWidth(styles.DefaultSectionWidth)
	c.choice.SetHeight(height - 1)
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
				c.logger.LogToFile("debug", fmt.Sprintf("submiting choice [%s]", c.choice.SelectedItem().(listitems.ChoiceItem).Option))
				return c, func() tea.Msg { return SubmitChoiceMsg(c.choice.SelectedItem().(listitems.ChoiceItem).Option) }
			}
		case OptionsMsg:
			return c, c.choice.SetItems(msg)
		}
		choice, cmd := c.choice.Update(msg)

		c.choice = choice
		return c, cmd
	}

	return c, nil
}

func (c *Choice) View() string {
	if !c.hidden {
		if c.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, c.choice.Title, c.choice.View()))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, c.choice.Title, c.choice.View()))
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
	c.Show()
	c.focused = true
}

func (c *Choice) Blur() {
	c.focused = false
}

type SubmitChoiceMsg listitems.OptionName
