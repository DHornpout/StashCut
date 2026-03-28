package keybindings

import "github.com/charmbracelet/bubbles/key"

type AppKeyMap struct {
	Tab        key.Binding
	ShiftTab   key.Binding
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	New        key.Binding
	Edit       key.Binding
	Delete     key.Binding
	Favorite   key.Binding
	MoveDown   key.Binding
	MoveUp     key.Binding
	Search     key.Binding
	FilterMac  key.Binding
	FilterWin  key.Binding
	FilterAll  key.Binding
	Help       key.Binding
	Quit       key.Binding
	Escape     key.Binding
}

var Keys = AppKeyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch panel"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "switch panel"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Favorite: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "favorite"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("J"),
		key.WithHelp("J", "move down"),
	),
	MoveUp: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "move up"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	FilterMac: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "mac"),
	),
	FilterWin: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "windows"),
	),
	FilterAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "all OS"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}
