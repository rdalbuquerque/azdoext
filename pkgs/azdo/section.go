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
	ActiveStyle = ActiveStyle.Height(height - 2)
	InactiveStyle = InactiveStyle.Height(height - 2)
	m.TaskList.SetHeight(height - 2)
	m.logViewPort.SetDimensions(width-50, height-2)
	m.PipelineList.SetHeight(height - 2)
	m.RunOrFollowList.SetHeight(height - 2)
}
