package listitems

import "github.com/charmbracelet/bubbles/key"

type CustomHelp struct {
	AdditionalShortHelpKeys func() []key.Binding
}

func (c CustomHelp) FullHelp() [][]key.Binding {
	return nil
}

func (c CustomHelp) ShortHelp() []key.Binding {
	keys := defaultKeys()
	if c.AdditionalShortHelpKeys == nil {
		return keys
	}
	extra := c.AdditionalShortHelpKeys()
	keys = append(keys, extra...)
	return keys
}

func New() CustomHelp {
	return CustomHelp{}
}
