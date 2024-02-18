package sections

import (
	"explore-bubbletea/pkgs/listitems"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

type WorktreeSection struct {
	hidden   bool
	focused  bool
	status   list.Model
	worktree *git.Worktree
	repo     *git.Repository
}

type WorktreeSectionData struct {
	Section  Section
	Repo     *git.Repository
	Worktree *git.Worktree
}

func (ws *WorktreeSection) push() tea.Msg {
	err := ws.repo.Push(&git.PushOptions{
		Auth:     &githttp.BasicAuth{Username: "", Password: os.Getenv("AZDO_PERSONAL_ACCESS_TOKEN")},
		Progress: nil,
	})
	if err != nil {
		panic(err)
	}
	return GitPushedMsg(true)
}

func (ws *WorktreeSection) addAllToStage() {
	if err := ws.worktree.AddGlob("."); err != nil {
		panic(err)
	}
	status, err := ws.worktree.Status()
	if err != nil {
		panic(err)
	}
	fileItems := []list.Item{}
	for file, _ := range status {
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file, Staged: true})
	}
	ws.status.SetItems(fileItems)
}

func NewWorktreeSection() WorktreeSectionData {
	r, err := git.PlainOpen(".")
	if err != nil {
		panic(err)
	}
	w, err := r.Worktree()
	if err != nil {
		panic(err)
	}
	worktreeSection := &WorktreeSection{
		hidden:   false,
		focused:  false,
		repo:     r,
		worktree: w,
	}
	worktreeSection.status = worktreeSection.setStagedFileList()

	return WorktreeSectionData{
		Section:  worktreeSection,
		Repo:     r,
		Worktree: w,
	}
}

func (ws *WorktreeSection) SetDimensions(width, height int) {
	ws.status.SetWidth(40)
	ws.status.SetHeight(height - 4)
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
	case commitMsg:
		if ws.noStagedFiles() {
			ws.addAllToStage()
		}
		ws.status.Title = "Pushing..."
		repo, err := ws.repo.Config()
		if err != nil {
			panic(err)
		}
		authorName := repo.Author.Name
		authorEmail := repo.Author.Email
		ws.worktree.Commit(string(msg), &git.CommitOptions{
			Author: &object.Signature{
				Name:  authorName,
				Email: authorEmail,
				When:  time.Now(),
			},
		})
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
			return ActiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, ws.status.Title, ws.status.View()))
		}
		return InactiveStyle.Render(lipgloss.JoinVertical(lipgloss.Center, ws.status.Title, ws.status.View()))
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

func (ws *WorktreeSection) setStagedFileList() list.Model {
	status, err := ws.worktree.Status()
	if err != nil {
		panic(err)
	}
	fileItems := []list.Item{}
	for file, _ := range status {
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file, Staged: status[file].Staging == git.Added})
	}
	stagedFileList := list.New(fileItems, listitems.GitItemDelegate{}, 0, 0)
	stagedFileList.Title = "Git status:"
	stagedFileList.SetShowTitle(false)
	stagedFileList.SetShowStatusBar(false)
	return stagedFileList
}

func (ws *WorktreeSection) addToStage() {
	selected := ws.status.SelectedItem()
	if selected == nil {
		return
	}
	item, ok := selected.(listitems.StagedFileItem)
	if !ok {
		return
	}
	if _, err := ws.worktree.Add(item.Name); err != nil {
		panic(err)
	}
	for i := range ws.status.Items() {
		if ws.status.Items()[i].(listitems.StagedFileItem).Name == item.Name {
			newItem := listitems.StagedFileItem{
				Name:   item.Name,
				Staged: true,
			}
			ws.status.Items()[i] = newItem
		}
	}
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
