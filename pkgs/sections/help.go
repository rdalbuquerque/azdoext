package sections

import (
	"azdoext/pkgs/styles"
	"io"
	"net/http"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type HelpSection struct {
	hidden            bool
	focused           bool
	viewport          viewport.Model
	style             lipgloss.Style
	sectionIdentifier SectionName
}

func fetchContent(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func NewHelpSection(secid SectionName) Section {
	vp := viewport.New(0, 0)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err != nil {
		panic(err)
	}

	content, err := fetchContent("https://raw.githubusercontent.com/rdalbuquerque/azdoext/main/README.md")
	if err != nil {
		panic(err)
	}

	str, err := renderer.Render(content)
	if err != nil {
		panic(err)
	}

	vp.SetContent(str)

	return &HelpSection{
		sectionIdentifier: "help",
		viewport:          vp,
		style:             styles.ActiveStyle.Copy(),
	}
}

func (h *HelpSection) GetSectionIdentifier() SectionName {
	return h.sectionIdentifier
}

func (h *HelpSection) SetDimensions(width, height int) {
	h.viewport.Width = 100
	h.viewport.Height = height
}

func (h *HelpSection) IsHidden() bool {
	return h.hidden
}

func (h *HelpSection) IsFocused() bool {
	return h.focused
}

func (h *HelpSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	if h.focused {
		vp, cmd := h.viewport.Update(msg)
		h.viewport = vp
		return h, cmd
	}
	return h, nil
}

func (h *HelpSection) View() string {
	if h.focused {
		return h.style.Width(styles.Width).Render(h.viewport.View())
	}
	return ""
}

func (h *HelpSection) Hide() {
	h.hidden = true
}

func (h *HelpSection) Show() {
	h.hidden = false
}

func (h *HelpSection) Focus() {
	h.Show()
	h.focused = true
}

func (h *HelpSection) Blur() {
	h.focused = false
}
