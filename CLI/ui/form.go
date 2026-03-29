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
	FormModeAddGroup
)

type FormSubmitMsg struct {
	Mode      FormMode
	App       *model.App
	Shortcut  *model.Shortcut
	GroupName string // used for AddGroup and AddShortcut group selection
	AppID     string // which app the group/shortcut belongs to
}

type FormCancelMsg struct{}

type Form struct {
	Mode       FormMode
	Active     bool
	fields     []textinput.Model
	focused    int
	Original   *model.Shortcut
	AppID      string
	multiGroup bool // true when shortcut form shows a group selector field
}

// Field index constants for shortcut form (single-group mode, no group field).
const (
	fieldDesc    = 0
	fieldMacKeys = 1
	fieldWinKeys = 2
	fieldTags    = 3
)

// Field index constants for shortcut form (multi-group mode, group field at index 1).
const (
	fieldDescMG    = 0
	fieldGroupMG   = 1
	fieldMacKeysMG = 2
	fieldWinKeysMG = 3
	fieldTagsMG    = 4
)

// Field index constants for app form.
const (
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

// NewAddGroupForm creates a form for adding a new group to an app.
func NewAddGroupForm(appID string) Form {
	name := textinput.New()
	name.Placeholder = "Group name (e.g. Navigation)"
	name.CharLimit = 64
	name.Focus()

	return Form{
		Mode:    FormModeAddGroup,
		Active:  true,
		AppID:   appID,
		fields:  []textinput.Model{name},
		focused: 0,
	}
}

// NewAddShortcutForm creates a form for adding a shortcut.
// groupNames lists the existing groups; if more than one exists a group selector field is shown.
func NewAddShortcutForm(appID string, groupNames []string) Form {
	return newShortcutForm(FormModeAddShortcut, appID, nil, groupNames)
}

func NewEditShortcutForm(appID string, s *model.Shortcut, groupNames []string) Form {
	f := newShortcutForm(FormModeEditShortcut, appID, s, groupNames)
	f.Original = s
	return f
}

func newShortcutForm(mode FormMode, appID string, s *model.Shortcut, groupNames []string) Form {
	multiGroup := len(groupNames) > 1

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

	var fields []textinput.Model
	if multiGroup {
		groupInput := textinput.New()
		hint := strings.Join(groupNames, ", ")
		if len(hint) > 40 {
			hint = hint[:37] + "…"
		}
		groupInput.Placeholder = "Group (" + hint + ")"
		groupInput.CharLimit = 64
		fields = []textinput.Model{desc, groupInput, macKeys, winKeys, tags}
	} else {
		fields = []textinput.Model{desc, macKeys, winKeys, tags}
	}

	return Form{
		Mode:       mode,
		Active:     true,
		AppID:      appID,
		fields:     fields,
		focused:    0,
		multiGroup: multiGroup,
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

		case FormModeAddGroup:
			return FormSubmitMsg{
				Mode:      f.Mode,
				GroupName: strings.TrimSpace(f.fields[0].Value()),
				AppID:     f.AppID,
			}

		default: // AddShortcut / EditShortcut
			var descVal, macVal, winVal, tagsVal, groupVal string
			if f.multiGroup {
				descVal = strings.TrimSpace(f.fields[fieldDescMG].Value())
				groupVal = strings.TrimSpace(f.fields[fieldGroupMG].Value())
				macVal = strings.TrimSpace(f.fields[fieldMacKeysMG].Value())
				winVal = strings.TrimSpace(f.fields[fieldWinKeysMG].Value())
				tagsVal = f.fields[fieldTagsMG].Value()
			} else {
				descVal = strings.TrimSpace(f.fields[fieldDesc].Value())
				macVal = strings.TrimSpace(f.fields[fieldMacKeys].Value())
				winVal = strings.TrimSpace(f.fields[fieldWinKeys].Value())
				tagsVal = f.fields[fieldTags].Value()
			}

			keysByOS := map[string]model.KeysForOS{}
			if macVal != "" {
				keysByOS["macos"] = model.KeysForOS{Keys: macVal, KeysDisplay: macVal}
			}
			if winVal != "" {
				keysByOS["windows"] = model.KeysForOS{Keys: winVal, KeysDisplay: winVal}
			}
			rawTags := strings.Split(tagsVal, ",")
			var tags []string
			for _, t := range rawTags {
				if trimmed := strings.TrimSpace(t); trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
			sc := &model.Shortcut{
				Description: descVal,
				KeysByOS:    keysByOS,
				Tags:        tags,
			}
			return FormSubmitMsg{
				Mode:      f.Mode,
				Shortcut:  sc,
				GroupName: groupVal,
				AppID:     f.AppID,
			}
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
	case FormModeAddGroup:
		title = "New Group"
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
	switch f.Mode {
	case FormModeAddApp:
		return []string{"Name", "Icon"}
	case FormModeAddGroup:
		return []string{"Group Name"}
	default:
		if f.multiGroup {
			return []string{"Description", "Group", "Mac Keys", "Windows Keys", "Tags"}
		}
		return []string{"Description", "Mac Keys", "Windows Keys", "Tags"}
	}
}
