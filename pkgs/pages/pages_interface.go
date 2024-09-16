package pages

import (
	"azdoext/pkgs/sections"

	tea "github.com/charmbracelet/bubbletea"
)

type PageInterface interface {
	GetPageName() PageName
	AddSection(sections.Section)
	SetDimensions(width, height int)
	Update(tea.Msg) (PageInterface, tea.Cmd)
	View() string
	IsCurrentPage() bool
	SetAsCurrentPage()
	UnsetCurrentPage()
}
