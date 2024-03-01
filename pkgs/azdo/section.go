package azdo

func (m *Model) Hide() {
	m.hidden = true
}

func (m *Model) Show() {
	m.hidden = false
}

func (m *Model) Focus() {
	m.Show()
	m.focused = true
}

func (m *Model) Blur() {
	m.focused = false
}

func (m *Model) IsFocused() bool {
	return m.focused
}

func (m *Model) IsHidden() bool {
	return m.hidden
}

func (m *Model) SetDimensions(width, height int) {
	m.TaskList.SetHeight(height + 1)
	m.logViewPort.SetDimensions(100, height)
	m.PipelineList.SetHeight(height)
	m.RunOrFollowList.SetHeight(height)
}
