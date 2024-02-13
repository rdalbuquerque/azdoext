package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	git "github.com/go-git/go-git/v5"
)

func (m *model) addToStage() {
	selected := m.stagedFileList.SelectedItem()
	if selected == nil {
		return
	}
	item, ok := selected.(stagedFileItem)
	if !ok {
		return
	}
	if _, err := m.worktree.Add(item.name); err != nil {
		panic(err)
	}
	setStagedFileList(m.worktree)
}

func setStagedFileList(worktree *git.Worktree) list.Model {
	status, err := worktree.Status()
	if err != nil {
		panic(err)
	}
	fileItems := []list.Item{}
	for file, _ := range status {
		fileItems = append(fileItems, stagedFileItem{name: file, staged: false})
	}
	stagedFileList := list.New(fileItems, gitItemDelegate{}, 20, 0)
	stagedFileList.Title = "Status"
	return stagedFileList
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

type stagedFileItem struct {
	name   string
	staged bool
}

func (i stagedFileItem) FilterValue() string { return "" }

type gitItemDelegate struct{}

func (d gitItemDelegate) Height() int                             { return 1 }
func (d gitItemDelegate) Spacing() int                            { return 0 }
func (d gitItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d gitItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(stagedFileItem)
	if !ok {
		return
	}

	str := i.name

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}
