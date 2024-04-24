package sections

import (
	"azdoext/pkgs/styles"
	"context"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const content = `
# azdoext

A terminal UI to help streamline the process of commiting, pushing, creating PRs and following pipelines in Azure DevOps.

The app is divided into pages:
* git: where you can stage files, commit, push and create PRs
* pipelines: where you can see all pipelines related to the current repository and follow them
* pipeline instance: where you can see the logs and tasks of a specific pipeline run
* help: where you are right now

## Usage
### Keybindings
- ctrl+c: quit
- ctrl+b: go back to previous page
- ctrl+h: show/hide help
- ctrl+r: restart the process
- ctrl+s: save on any textarea
	- if on commit message: push
		- if no files are staged, stage all files before pushing
- ctrl+a: stage file on status list
- tab: switch between available sections
- enter: select an option on any list (has no effect on file status list)
- / : search for a string while on pipeline logs


## Commit and push
When the app starts you will see two sections, commit message and the changed files, files staged will be shown in green.\
Here you can either write a commit msg and hit ctrl+s to save, stage all files and push\
You can also stage individual files by pressing ctrl+a while the file is selected.

## Open a PR
After changes are pushed you will presented with a choice, you can either go directly to pipelines or open a PR.\
If you chose to open a PR, you will be presented with a text area where the first line is PR title and the rest is PR description.\
To save and open the PR press ctrl+s.\
With a opened PR you are taken to pipelines section.
OBS: Currently, only PRs to the default branch are supported.

## Monitor pipelines
On pipelines page, you will see all pipelines related to you current repository and their last run status.\
When you press enter you will be presented with a choice, go to the pipeline instance tasks or run new.\
While on the pipeline instance section you can go to logs, browse and hit '/' to search for a specific string.\





`

type Help struct {
	hidden   bool
	focused  bool
	viewport viewport.Model
	style    lipgloss.Style
}

func NewHelp(_ context.Context) Section {
	vp := viewport.New(0, 0)

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
	)
	if err != nil {
		panic(err)
	}

	str, err := renderer.Render(content)
	if err != nil {
		panic(err)
	}

	vp.SetContent(str)

	return &Help{
		viewport: vp,
		style:    styles.ActiveStyle.Copy(),
	}
}

func (h *Help) SetDimensions(width, height int) {

	h.viewport.Height = height
}

func (h *Help) IsHidden() bool {
	return h.hidden
}

func (h *Help) IsFocused() bool {
	return h.focused
}

func (h *Help) Update(msg tea.Msg) (Section, tea.Cmd) {
	if h.focused {
		vp, cmd := h.viewport.Update(msg)
		h.viewport = vp
		return h, cmd
	}
	return h, nil
}

func (h *Help) View() string {
	if h.focused {
		return h.style.Width(styles.Width).Render(h.viewport.View())
	}
	return ""
}

func (h *Help) Hide() {
	h.hidden = true
}

func (h *Help) Show() {
	h.hidden = false
}

func (h *Help) Focus() {
	h.Show()
	h.focused = true
}

func (h *Help) Blur() {
	h.focused = false
}
