package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stashcut/cli/model"
)

type FormMode int

const (
	FormModeAddApp FormMode = iota
	FormModeAddShortcut
	FormModeEditShortcut
)

type FormSubmitMsg struct {
	Mode     FormMode
	App      *model.App
	Shortcut *model.Shortcut
}

type FormCancelMsg struct{}

type Form struct {
	Mode    FormMode
	Active  bool
	fields  []textinput.Model
	focused int
	// For edit mode: the original shortcut
	Original *model.Shortcut
	AppID    string
}

const (
	fieldDesc    = 0
	fieldMacKeys = 1
	fieldWinKeys = 2
	fieldTags    = 3

	fieldAppName = 0
	fieldAppIcon = 1
)

func NewAddAppForm() Form {
	name := textinput.New()
	name.Placeholder = "App name (e.g. VS Code)"
	name.CharLimit = 64
	name.Focus()

	icon := textinput.New()
	icon.Placeholder = "Icon emoji (e.g. 🧑‍💻) — optional"
	icon.CharLimit = 4

	return Form{
		Mode:    FormModeAddApp,
		Active:  true,
		fields:  []textinput.Model{name, icon},
		focused: 0,
	}
}

func NewAddShortcutForm(appID string) Form {
	return newShortcutForm(FormModeAddShortcut, appID, nil)
}

func NewEditShortcutForm(appID string, s *model.Shortcut) Form {
	f := newShortcutForm(FormModeEditShortcut, appID, s)
	f.Original = s
	return f
}

func newShortcutForm(mode FormMode, appID string, s *model.Shortcut) Form {
	desc := textinput.New()
	desc.Placeholder = "Description"
	desc.CharLimit = 128
	desc.Focus()

	macKeys := textinput.New()
	macKeys.Placeholder = "Mac keys (e.g. Cmd+Shift+P)"
	macKeys.CharLimit = 64

	winKeys := textinput.New()
	winKeys.Placeholder = "Windows keys (e.g. Ctrl+Shift+P)"
	winKeys.CharLimit = 64

	tags := textinput.New()
	tags.Placeholder = "Tags (comma separated)"
	tags.CharLimit = 128

	if s != nil {
		desc.SetValue(s.Description)
		if kfo, ok := s.KeysByOS["macos"]; ok {
			macKeys.SetValue(kfo.Keys)
		}
		if kfo, ok := s.KeysByOS["windows"]; ok {
			winKeys.SetValue(kfo.Keys)
		}
		tags.SetValue(strings.Join(s.Tags, ", "))
	}

	return Form{
		Mode:    mode,
		Active:  true,
		AppID:   appID,
		fields:  []textinput.Model{desc, macKeys, winKeys, tags},
		focused: 0,
	}
}

func (f *Form) Update(msg tea.Msg) (Form, tea.Cmd) {
	if !f.Active {
		return *f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab, tea.KeyDown:
			f.fields[f.focused].Blur()
			f.focused = (f.focused + 1) % len(f.fields)
			f.fields[f.focused].Focus()
			return *f, nil
		case tea.KeyShiftTab, tea.KeyUp:
			f.fields[f.focused].Blur()
			f.focused = (f.focused - 1 + len(f.fields)) % len(f.fields)
			f.fields[f.focused].Focus()
			return *f, nil
		case tea.KeyEnter:
			if f.focused == len(f.fields)-1 {
				f.Active = false
				return *f, f.submit()
			}
			f.fields[f.focused].Blur()
			f.focused = (f.focused + 1) % len(f.fields)
			f.fields[f.focused].Focus()
			return *f, nil
		case tea.KeyEsc:
			f.Active = false
			return *f, func() tea.Msg { return FormCancelMsg{} }
		}
	}

	var cmd tea.Cmd
	f.fields[f.focused], cmd = f.fields[f.focused].Update(msg)
	return *f, cmd
}

func (f Form) submit() tea.Cmd {
	return func() tea.Msg {
		switch f.Mode {
		case FormModeAddApp:
			app := &model.App{
				Name: strings.TrimSpace(f.fields[fieldAppName].Value()),
				Icon: strings.TrimSpace(f.fields[fieldAppIcon].Value()),
			}
			return FormSubmitMsg{Mode: f.Mode, App: app}
		default:
			keysByOS := map[string]model.KeysForOS{}
			if mac := strings.TrimSpace(f.fields[fieldMacKeys].Value()); mac != "" {
				keysByOS["macos"] = model.KeysForOS{Keys: mac, KeysDisplay: mac}
			}
			if win := strings.TrimSpace(f.fields[fieldWinKeys].Value()); win != "" {
				keysByOS["windows"] = model.KeysForOS{Keys: win, KeysDisplay: win}
			}
			rawTags := strings.Split(f.fields[fieldTags].Value(), ",")
			var tags []string
			for _, t := range rawTags {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
			sc := &model.Shortcut{
				AppID:       f.AppID,
				Description: strings.TrimSpace(f.fields[fieldDesc].Value()),
				KeysByOS:    keysByOS,
				Tags:        tags,
			}
			return FormSubmitMsg{Mode: f.Mode, Shortcut: sc}
		}
	}
}

func (f Form) View(width int) string {
	if !f.Active {
		return ""
	}

	title := ""
	switch f.Mode {
	case FormModeAddApp:
		title = "New App"
	case FormModeAddShortcut:
		title = "New Shortcut"
	case FormModeEditShortcut:
		title = "Edit Shortcut"
	}

	var lines []string
	lines = append(lines, StyleTitle.Render(title))

	labels := f.fieldLabels()
	for i, fi := range f.fields {
		label := StyleMuted.Render(labels[i] + ":")
		field := fi.View()
		if i == f.focused {
			label = StyleKey.Render(labels[i] + ":")
		}
		lines = append(lines, label)
		lines = append(lines, field)
		lines = append(lines, "")
	}

	lines = append(lines, StyleMuted.Render("tab/↑↓ navigate  •  enter submit  •  esc cancel"))

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 2).
		Width(width - 4).
		Render(content)
}

func (f Form) fieldLabels() []string {
	if f.Mode == FormModeAddApp {
		return []string{"Name", "Icon"}
	}
	return []string{"Description", "Mac Keys", "Windows Keys", "Tags"}
}
