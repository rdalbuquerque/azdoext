package sections

import (
	"azdoext/pkgs/styles"
	"io"
	"net/http"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/rdalbuquerque/viewsearch"
)

type HelpSection struct {
	hidden            bool
	focused           bool
	viewsearch        viewsearch.Model
	style             lipgloss.Style
	sectionIdentifier SectionName
	ViewsearchHelp    string
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
	vs := viewsearch.New()
	helpstr := help.New().ShortHelpView(vs.HelpBindings)
	vs.SetShowHelp(false)

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

	vs.SetContent(str)

	return &HelpSection{
		sectionIdentifier: "help",
		viewsearch:        vs,
		ViewsearchHelp:    helpstr,
		style:             styles.ActiveStyle.Copy(),
	}
}

func (h *HelpSection) GetSectionIdentifier() SectionName {
	return h.sectionIdentifier
}

func (h *HelpSection) SetDimensions(width, height int) {
	h.viewsearch.SetDimensions(100, height)
}

func (h *HelpSection) IsHidden() bool {
	return h.hidden
}

func (h *HelpSection) IsFocused() bool {
	return h.focused
}

func (h *HelpSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	if h.focused {
		vp, cmd := h.viewsearch.Update(msg)
		h.viewsearch = vp
		return h, cmd
	}
	return h, nil
}

func (h *HelpSection) View() string {
	if h.focused {
		return h.style.Width(styles.Width).Render(h.viewsearch.View())
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
