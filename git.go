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

func (m *model) addAllToStage() {
	if err := m.worktree.AddGlob("."); err != nil {
		panic(err)
	}
	status, err := m.worktree.Status()
	if err != nil {
		panic(err)
	}
	fileItems := []list.Item{}
	for file, _ := range status {
		// Check if the file is staged
		fileStatus, ok := status[file]
		var staged bool
		if !ok {
			log2file(fmt.Sprintf("file: %s, status: %s\n", file, "not ok status"))
		} else {
			// File is tracked; check if it's staged for commit
			if fileStatus.Staging == git.Added || fileStatus.Staging == git.Modified || fileStatus.Staging == git.Deleted {
				log2file(fmt.Sprintf("file: %s, status: %v\n", file, fileStatus.Staging))
				staged = true
			} else {
				log2file(fmt.Sprintf("file: %s, status: %v\n", file, fileStatus.Staging))
				staged = false
			}
		}
		log2file(fmt.Sprintf("file: %s, status: %v\n", file, fileStatus))
		fileItems = append(fileItems, stagedFileItem{name: file, staged: staged})
	}
	m.stagedFileList.SetItems(fileItems)
}

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
	for i := range m.stagedFileList.Items() {
		if m.stagedFileList.Items()[i].(stagedFileItem).name == item.name {
			newItem := stagedFileItem{
				name:   item.name,
				staged: true,
			}
			m.stagedFileList.Items()[i] = newItem
		}
	}
}

func setStagedFileList(worktree *git.Worktree) list.Model {
	status, err := worktree.Status()
	if err != nil {
		panic(err)
	}
	fileItems := []list.Item{}
	for file, _ := range status {
		fileItems = append(fileItems, stagedFileItem{name: file, staged: status[file].Staging == git.Added})
	}
	stagedFileList := list.New(fileItems, gitItemDelegate{}, 20, 0)
	stagedFileList.Title = "Status"
	return stagedFileList
}

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2)
	stagedFileStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
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
	if i.staged {
		str = stagedFileStyle.Render(str)
	}
	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("| " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m *model) noStagedFiles() bool {
	for _, file := range m.stagedFileList.Items() {
		if file.(stagedFileItem).staged {
			return false
		}
	}
	return true
}
