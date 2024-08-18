package searchableviewport

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewportContentMsg string

type searchResult struct {
	Line  int
	Index int
}

type Model struct {
	Viewport           viewport.Model
	searchResults      []searchResult
	searchMode         bool
	ta                 textarea.Model
	originalContent    string
	currentResultIndex int
	browsingMode       bool
}

var (
	currentHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("#00FF00"))
	highlightStyle        = lipgloss.NewStyle().Background(lipgloss.Color("#FF00FF"))
	focusedStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))     // e.g., bright color
	blurredStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))     // e.g., grayed out
	noResultsStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#b22222")) // e.g., red
)

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	ta := textarea.New()
	ta.SetHeight(1)
	ta.ShowLineNumbers = false
	ta.Prompt = "/"
	return &Model{
		Viewport: vp,
		ta:       ta,
	}
}

func (m *Model) setTextAreaWidth(viewportWidth int) {
	if viewportWidth < 80 && viewportWidth > 4 {
		m.ta.SetWidth(viewportWidth - 4)
	} else {
		m.ta.SetWidth(80)
	}
}

func (m *Model) SetDimensions(width, height int) {
	m.Viewport.Height = height
	m.Viewport.Width = width
	m.ta.SetHeight(1)
	m.setTextAreaWidth(width)
}

func (m *Model) GotoBottom() {
	m.Viewport.GotoBottom()
}

func (m *Model) SetContent(content string) {
	m.originalContent = content
	m.Viewport.SetContent(content)
	if m.searchMode {
		m.highlightMatches()
	}
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/", "alt+ctrl+_":
			return m, m.handleSearchActivation(msg)
		case "esc":
			return m, m.handleDeactivations()
		case "enter":
			return m, m.handleBrowseActivation()
		case "n":
			return m, m.handleNavigation(msg)
		default:
			tacmd := m.updateTextArea(msg)
			if m.searchMode {
				m.Viewport.SetContent(m.originalContent)
				m.highlightMatches()
				return m, tacmd
			}
			vpcmd := m.updateViewPort(msg)
			return m, tea.Batch(vpcmd, tacmd)
		}
	case viewportContentMsg:
		m.SetContent(string(msg))
		if m.searchMode {
			m.highlightMatches()
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) updateTextArea(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.ta, cmd = m.ta.Update(msg)
	return cmd
}

func (m *Model) updateViewPort(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return cmd
}

func (m *Model) handleNavigation(msg tea.Msg) tea.Cmd {
	if m.browsingMode {
		m.navigateToNextResult()
		return m.updateViewPort(msg)
	}
	return m.updateTextArea(msg)
}

func (m *Model) handleBrowseActivation() tea.Cmd {
	if m.searchMode {
		m.ta.Blur()
		m.browsingMode = true
		return nil
	}
	return nil
}

func (m *Model) handleSearchActivation(msg tea.Msg) tea.Cmd {
	if m.searchMode {
		return m.updateTextArea(msg)
	}
	if !m.searchMode {
		m.searchMode = true
		m.Viewport.Height--
		m.ta.Focus()
		return nil
	}
	return nil
}

func (m *Model) handleDeactivations() tea.Cmd {
	if m.browsingMode {
		m.browsingMode = false
		m.ta.Focus()
		return nil
	}
	if m.searchMode {
		m.searchMode = false
		m.Viewport.Height++
		m.Viewport.LineDown(1)
		return nil
	}
	return nil
}

func (m *Model) View() string {
	searchCounter := fmt.Sprintf(" %d/%d", m.currentResultIndex+1, len(m.searchResults))
	if !m.hasSearchResults() {
		searchCounter = noResultsStyle.Render(" 0!")
	}
	var taView string
	if m.ta.Focused() {
		taView = focusedStyle.Render(m.ta.View())
	} else {
		taView = blurredStyle.Render(m.ta.View())
	}
	renderedViewPort := m.Viewport.View()
	if m.searchMode {
		return lipgloss.JoinVertical(lipgloss.Top, lipgloss.JoinHorizontal(lipgloss.Left, taView, searchCounter), renderedViewPort)
	}
	return renderedViewPort
}

func (m *Model) highlightMatches() {
	searchQuery := m.ta.Value()
	if searchQuery == "" {
		return
	}

	m.resetSearchResults()
	m.findAndHighlightMatches(searchQuery)
}

func (m *Model) resetSearchResults() {
	m.searchResults = []searchResult{}
}

func (m *Model) findAndHighlightMatches(searchQuery string) {
	lines := strings.Split(m.originalContent, "\n")
	var processedLines []string
	for i, line := range lines {
		processedLines = append(processedLines, m.processLineForMatches(i, line, searchQuery))
	}
	m.Viewport.SetContent(strings.Join(processedLines, "\n"))
}

func (m *Model) processLineForMatches(lineIndex int, line, searchQuery string) string {
	var highlightedLine string
	var startPos int

	for {
		index := strings.Index(line[startPos:], searchQuery)
		if index < 0 {
			highlightedLine += line[startPos:]
			break
		}

		m.storeSearchResult(lineIndex, startPos+index)
		highlightedLine += m.highlightMatch(lineIndex, startPos, index, searchQuery, line)
		startPos += index + len(searchQuery)
	}

	return highlightedLine
}

func (m *Model) highlightMatch(lineIndex, startPos, index int, searchQuery, line string) string {
	styleToUse := m.setHighlightStyle(lineIndex, startPos+index)
	matchedPart := line[startPos+index : startPos+index+len(searchQuery)]
	return line[startPos:startPos+index] + styleToUse.Render(matchedPart)
}

func (m *Model) storeSearchResult(line, index int) {
	m.searchResults = append(m.searchResults, searchResult{Line: line, Index: index})
}

func (m *Model) setHighlightStyle(lineIndex, index int) lipgloss.Style {
	if m.currentResultIndex >= 0 && m.currentResultIndex < len(m.searchResults) {
		if lineIndex == m.searchResults[m.currentResultIndex].Line && index == m.searchResults[m.currentResultIndex].Index {
			return currentHighlightStyle
		}
	}
	return highlightStyle
}

func (m *Model) navigateToNextResult() {
	if !m.hasSearchResults() {
		return
	}
	m.incrementSearchIndex()
	m.scrollToCurrentResult()
	m.highlightMatches()
}

func (m *Model) hasSearchResults() bool {
	return len(m.searchResults) > 0
}

func (m *Model) incrementSearchIndex() {
	m.currentResultIndex = (m.currentResultIndex + 1) % len(m.searchResults)
}

func (m *Model) scrollToCurrentResult() {
	nextResult := m.searchResults[m.currentResultIndex]
	m.scrollViewportToLine(nextResult.Line)
}

func (m *Model) scrollViewportToLine(line int) {
	// Check if the resultLine is currently visible
	topLine := m.Viewport.YOffset
	bottomLine := topLine + m.Viewport.Height - 1 // -1 because it's zero-based index
	for line < topLine || line > bottomLine {
		if line < topLine {
			m.Viewport.ViewUp()
		} else {
			m.Viewport.ViewDown()
		}

		// Update topLine and bottomLine after scrolling
		topLine = m.Viewport.YOffset
		bottomLine = topLine + m.Viewport.Height - 1
	}
}
