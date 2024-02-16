package main

import (
	"fmt"

	"explore-bubbletea/pkgs/listitems"

	"github.com/charmbracelet/bubbles/list"
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
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file, Staged: staged})
	}
	m.stagedFileList.SetItems(fileItems)
}

func (m *model) addToStage() {
	selected := m.stagedFileList.SelectedItem()
	if selected == nil {
		return
	}
	item, ok := selected.(listitems.StagedFileItem)
	if !ok {
		return
	}
	if _, err := m.worktree.Add(item.Name); err != nil {
		panic(err)
	}
	for i := range m.stagedFileList.Items() {
		if m.stagedFileList.Items()[i].(listitems.StagedFileItem).Name == item.Name {
			newItem := listitems.StagedFileItem{
				Name:   item.Name,
				Staged: true,
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
		fileItems = append(fileItems, listitems.StagedFileItem{Name: file, Staged: status[file].Staging == git.Added})
	}
	stagedFileList := list.New(fileItems, listitems.GitItemDelegate{}, 20, 0)
	stagedFileList.Title = "Status"
	return stagedFileList
}

func (m *model) noStagedFiles() bool {
	for _, file := range m.stagedFileList.Items() {
		if file.(listitems.StagedFileItem).Staged {
			return false
		}
	}
	return true
}
