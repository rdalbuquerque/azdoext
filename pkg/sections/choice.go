package sections

import (
	"azdoext/pkg/listitems"
	"azdoext/pkg/logger"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Choice struct {
	logger            *logger.Logger
	hidden            bool
	focused           bool
	choice            list.Model
	sectionIdentifier SectionName
}

func NewChoice(secid SectionName) Section {
	logger := logger.NewLogger("choice.log")

	choice := list.New([]list.Item{}, listitems.ChoiceItemDelegate{}, 0, 0)
	choice.Title = "PR or pipelines:"
	choice.SetHeight(styles.ActiveStyle.GetHeight() - 2)
	choice.SetShowTitle(false)
	return &Choice{
		logger:            logger,
		choice:            choice,
		sectionIdentifier: secid,
	}
}

func (c *Choice) GetSectionIdentifier() SectionName {
	return c.sectionIdentifier
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
			case "q":
				return c, nil
			case "enter":
				c.logger.LogToFile("debug", fmt.Sprintf("submiting choice [%s]", c.choice.SelectedItem().(listitems.ChoiceItem).Option))
				return c, func() tea.Msg { return teamsg.SubmitChoiceMsg(c.choice.SelectedItem().(listitems.ChoiceItem).Option) }
			}
		case teamsg.OptionsMsg:
			return c, c.choice.SetItems(msg)
		}
		choice, cmd := c.choice.Update(msg)

		c.choice = choice
		return c, cmd
	}

	return c, nil
}

func (c *Choice) View() string {
	title := styles.TitleStyle.Render(c.choice.Title)
	secView := lipgloss.JoinVertical(lipgloss.Top, title, c.choice.View())
	if !c.hidden {
		if c.focused {
			return styles.ActiveStyle.Render(secView)
		}
		return styles.InactiveStyle.Render(secView)
	}
	return ""
}

func (c *Choice) Hide() {
	c.focused = false
	c.hidden = true
}

func (c *Choice) Show() {
	c.focused = true
	c.hidden = false
}

func (c *Choice) Focus() {
	c.Show()
	c.focused = true
}

func (c *Choice) Blur() {
	c.focused = false
}
