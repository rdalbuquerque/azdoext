package sections

import (
	"azdoext/pkg/azdo"
	"azdoext/pkg/gitexec"
	"azdoext/pkg/listitems"
	"azdoext/pkg/logger"
	"azdoext/pkg/styles"
	"azdoext/pkg/teamsg"
	"errors"

	bubbleshelp "github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WorktreeSection struct {
	logger            *logger.Logger
	hidden            bool
	focused           bool
	status            list.Model
	customhelp        string
	branch            string
	sectionIdentifier SectionName
	azdoconfig        azdo.Config
}

func (ws *WorktreeSection) push() tea.Msg {
	gitexec.Push("origin", ws.branch, ws.azdoconfig.PAT)
	return teamsg.GitPushedMsg(true)
}

func (ws *WorktreeSection) addAllToStage() {
	gitexec.AddGlob(".")
	ws.setStagedFileList()
}

func NewWorktreeSection(secid SectionName, currentBranch string, azdoconfig azdo.Config) Section {
	logger := logger.NewLogger("worktree.log")
	worktreeSection := &WorktreeSection{}
	worktreeSection.branch = currentBranch
	worktreeSection.logger = logger
	worktreeSection.status = newFileList()
	worktreeSection.setStagedFileList()
	statusHelp := bubbleshelp.New()
	hk := listitems.HelpKeys{}
	hk.AdditionalShortHelpKeys = func() []key.Binding {
		stageKey := []key.Binding{key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "unstage"),
		)}
		unstageKey := []key.Binding{key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "stage"),
		)}
		return append(unstageKey, stageKey...)
	}
	customhelp := statusHelp.View(hk)
	worktreeSection.customhelp = customhelp
	worktreeSection.sectionIdentifier = secid
	worktreeSection.azdoconfig = azdoconfig
	return worktreeSection
}

func (ws *WorktreeSection) GetSectionIdentifier() SectionName {
	return ws.sectionIdentifier
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
		if ws.focused {
			switch msg.String() {
			case "ctrl+a":
				ws.stageFile()
				status, cmd := ws.status.Update(msg)
				ws.status = status
				return ws, cmd
			case "ctrl+d":
				ws.unstageFile()
				status, cmd := ws.status.Update(msg)
				ws.status = status
				return ws, cmd
			default:
				status, cmd := ws.status.Update(msg)
				ws.status = status
				return ws, cmd
			}
		}
		return ws, nil
	}
	switch msg := msg.(type) {
	case teamsg.CommitMsg:
		if ws.noStagedFiles() {
			ws.addAllToStage()
		}
		ws.status.Title = "Pushing..."
		gitexec.Commit(string(msg))
		return ws, tea.Batch(ws.push, func() tea.Msg { return teamsg.GitPushingMsg(true) })
	case teamsg.GitPushedMsg:
		ws.status.Title = "Pushed"
	}
	if len(ws.status.Items()) == 0 {
		return ws, func() tea.Msg { return teamsg.NothingToCommitMsg{} }
	}
	return ws, nil
}

func (ws *WorktreeSection) View() string {
	title := styles.TitleStyle.Render(ws.status.Title)
	if !ws.hidden {
		if ws.focused {
			return styles.ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, title, ws.status.View(), ws.customhelp))
		}
		return styles.InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Top, title, ws.status.View(), ws.customhelp))
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
	for _, file := range status {
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file.Name, RawStatus: file.RawStatus, Staged: file.Staged})
	}
	ws.status.SetItems(fileItems)
}

func (ws *WorktreeSection) stageFile() {
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

func (ws *WorktreeSection) unstageFile() {
	selected := ws.status.SelectedItem()
	if selected == nil {
		panic(errors.New("no item selected"))
	}
	item, ok := selected.(listitems.StagedFileItem)
	if !ok {
		panic(errors.New("selected item is not a StagedFileItem"))
	}
	gitexec.Unstage(item.Name)
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
