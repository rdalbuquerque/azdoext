package pages

import (
	"azdoext/pkg/sections"

	tea "charm.land/bubbletea/v2"
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
