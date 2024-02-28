package sections

import (
	"azdoext/pkgs/gitexec"
	"azdoext/pkgs/listitems"
	"azdoext/pkgs/styles"
	"errors"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WorktreeSection struct {
	hidden     bool
	focused    bool
	status     list.Model
	customhelp string
	branch     string
}

func (ws *WorktreeSection) push() tea.Msg {
	gitexec.Push("origin", ws.branch)
	return GitPushedMsg(true)
}

func (ws *WorktreeSection) addAllToStage() {
	gitexec.AddGlob(".")
	ws.setStagedFileList()
}

func NewWorktreeSection() Section {
	worktreeSection := &WorktreeSection{}
	worktreeSection.status = newFileList()
	worktreeSection.setStagedFileList()
	statusHelp := bubbleshelp.New()
	hk := listitems.HelpKeys{}
	hk.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "add to stage"),
		)}
	}
	customhelp := statusHelp.View(hk)
	worktreeSection.customhelp = customhelp
	return worktreeSection
}

func (ws *WorktreeSection) SetDimensions(width, height int) {
	ws.status.SetWidth(styles.DefaultSectionWidth)
	ws.status.SetHeight(height - 2)
}

func (ws *WorktreeSection) IsHidden() bool {
	return ws.hidden
}

func (ws *WorktreeSection) IsFocused() bool {
	return ws.focused
}

func (ws *WorktreeSection) Update(msg tea.Msg) (Section, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return nil, tea.Quit
		case "ctrl+a":
			ws.addToStage()
			status, cmd := ws.status.Update(msg)
			ws.status = status
			return ws, cmd
		default:
			if ws.focused {
				status, cmd := ws.status.Update(msg)
				ws.status = status
				return ws, cmd
			}
			return ws, nil
		}
	case BroadcastGitInfoMsg:
		log2file("BroadcastGitInfoMsg")
		gitconfig := gitexec.Config()
		remoteUrl := gitconfig.Origin
		curBranch := gitconfig.CurrentBranch
		ref := "refs/heads/" + curBranch
		ws.branch = curBranch
		return ws, func() tea.Msg { return GitInfoMsg{CurrentBranch: ref, RemoteUrl: remoteUrl} }
	case commitMsg:
		log2file("commitMsg on WorktreeSection")
		if ws.noStagedFiles() {
			ws.addAllToStage()
		}
		ws.status.Title = "Pushing..."
		gitexec.Commit(string(msg))
		return ws, tea.Batch(ws.push, func() tea.Msg { return GitPushingMsg(true) })
	case GitPushedMsg:
		ws.status.Title = "Pushed"
	}
	status, cmd := ws.status.Update(msg)
	ws.status = status
	return ws, cmd
}

func (ws *WorktreeSection) View() string {
	if !ws.hidden {
		if ws.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, ws.status.Title, ws.status.View(), ws.customhelp))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, ws.status.Title, ws.status.View(), ws.customhelp))
	}
	return ""
}

func (ws *WorktreeSection) Hide() {
	ws.hidden = true
}

func (ws *WorktreeSection) Show() {
	ws.hidden = false
}

func (ws *WorktreeSection) Focus() {
	ws.focused = true
}

func (ws *WorktreeSection) Blur() {
	ws.focused = false
}

func newFileList() list.Model {
	stagedFileList := list.New([]list.Item{}, listitems.GitItemDelegate{}, 0, 0)
	stagedFileList.Title = "Git status:"
	stagedFileList.SetShowTitle(false)
	stagedFileList.SetShowStatusBar(false)
	stagedFileList.SetShowHelp(false)
	return stagedFileList
}

func (ws *WorktreeSection) setStagedFileList() {
	status := gitexec.Status()
	fileItems := []list.Item{}
	for file, _ := range status {
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file, RawStatus: status[file].RawStatus, Staged: status[file].Staged})
	}
	ws.status.SetItems(fileItems)
}

func (ws *WorktreeSection) addToStage() {
	selected := ws.status.SelectedItem()
	if selected == nil {
		panic(errors.New("no item selected"))
	}
	item, ok := selected.(listitems.StagedFileItem)
	if !ok {
		panic(errors.New("selected item is not a StagedFileItem"))
	}
	gitexec.Add(item.Name)
	ws.setStagedFileList()
}

func (ws *WorktreeSection) noStagedFiles() bool {
	for _, file := range ws.status.Items() {
		if file.(listitems.StagedFileItem).Staged {
			return false
		}
	}
	return true
}

type GitPushedMsg bool
type GitPushingMsg bool
type BroadcastGitInfoMsg bool
type GitInfoMsg struct {
	CurrentBranch string
	RemoteUrl     string
}
