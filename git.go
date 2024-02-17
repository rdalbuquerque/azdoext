package main

import (
	"explore-bubbletea/pkgs/listitems"

	"github.com/charmbracelet/bubbles/list"
	git "github.com/go-git/go-git/v5"
)

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
