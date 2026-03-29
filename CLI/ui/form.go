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
	GroupName string // used for AddGroup and AddShortcut/EditShortcut group selection
	AppID     string // which app the group/shortcut belongs to
}

type FormCancelMsg struct{}

// groupFieldIdx is the position of the cycle-select group slot within a multi-group form.
const groupFieldIdx = 1

type Form struct {
	Mode         FormMode
	Active       bool
	fields       []textinput.Model // text inputs only (does not include the group slot)
	focused      int               // virtual index over all fields including the group slot
	Original     *model.Shortcut
	AppID        string
	multiGroup   bool
	groupOptions []string // valid group names for the cycle selector
	groupIdx     int      // currently selected index into groupOptions
}

// Field index constants for shortcut form text inputs (desc, mac, win, tags).
const (
	fieldDesc    = 0
	fieldMacKeys = 1
	fieldWinKeys = 2
	fieldTags    = 3
)

// Field index constants for app form.
const (
	fieldAppName = 0
	fieldAppIcon = 1
)

// fieldCount returns the total number of navigable fields (text inputs + group slot if present).
func (f *Form) fieldCount() int {
	if f.multiGroup {
		return len(f.fields) + 1 // +1 for the group cycle slot
	}
	return len(f.fields)
}

// textInputIndex maps the current virtual focused index to a slice index into f.fields.
// Returns -1 when the focused slot is the group cycle selector.
func (f *Form) textInputIndex() int {
	if !f.multiGroup {
		return f.focused
	}
	if f.focused < groupFieldIdx {
		return f.focused
	}
	if f.focused == groupFieldIdx {
		return -1
	}
	return f.focused - 1
}

// formIdxToTextInput maps any virtual form index to a f.fields index (-1 for group slot).
func (f *Form) formIdxToTextInput(formIdx int) int {
	if !f.multiGroup {
		return formIdx
	}
	if formIdx < groupFieldIdx {
		return formIdx
	}
	if formIdx == groupFieldIdx {
		return -1
	}
	return formIdx - 1
}

func (f *Form) blurCurrentTextInput() {
	if idx := f.textInputIndex(); idx >= 0 {
		f.fields[idx].Blur()
	}
}

func (f *Form) focusCurrentTextInput() {
	if idx := f.textInputIndex(); idx >= 0 {
		f.fields[idx].Focus()
	}
}

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
// When groupNames has more than one entry, a cycle-select group field is shown.
func NewAddShortcutForm(appID string, groupNames []string) Form {
	return newShortcutForm(FormModeAddShortcut, appID, nil, groupNames, "")
}

// NewEditShortcutForm creates a pre-filled form for editing a shortcut.
// currentGroupName pre-selects the correct option in the cycle selector.
func NewEditShortcutForm(appID string, s *model.Shortcut, groupNames []string, currentGroupName string) Form {
	f := newShortcutForm(FormModeEditShortcut, appID, s, groupNames, currentGroupName)
	f.Original = s
	return f
}

func newShortcutForm(mode FormMode, appID string, s *model.Shortcut, groupNames []string, currentGroupName string) Form {
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

	// Determine initial group selection.
	groupIdx := 0
	if multiGroup && currentGroupName != "" {
		for i, g := range groupNames {
			if g == currentGroupName {
				groupIdx = i
				break
			}
		}
	}

	return Form{
		Mode:         mode,
		Active:       true,
		AppID:        appID,
		fields:       []textinput.Model{desc, macKeys, winKeys, tags},
		focused:      0,
		multiGroup:   multiGroup,
		groupOptions: groupNames,
		groupIdx:     groupIdx,
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
			f.blurCurrentTextInput()
			f.focused = (f.focused + 1) % f.fieldCount()
			f.focusCurrentTextInput()
			return *f, nil

		case tea.KeyShiftTab, tea.KeyUp:
			f.blurCurrentTextInput()
			f.focused = (f.focused - 1 + f.fieldCount()) % f.fieldCount()
			f.focusCurrentTextInput()
			return *f, nil

		case tea.KeyLeft:
			if f.multiGroup && f.focused == groupFieldIdx && len(f.groupOptions) > 0 {
				f.groupIdx = (f.groupIdx - 1 + len(f.groupOptions)) % len(f.groupOptions)
				return *f, nil
			}

		case tea.KeyRight:
			if f.multiGroup && f.focused == groupFieldIdx && len(f.groupOptions) > 0 {
				f.groupIdx = (f.groupIdx + 1) % len(f.groupOptions)
				return *f, nil
			}

		case tea.KeyEnter:
			if f.focused == f.fieldCount()-1 {
				f.Active = false
				return *f, f.submit()
			}
			f.blurCurrentTextInput()
			f.focused = (f.focused + 1) % f.fieldCount()
			f.focusCurrentTextInput()
			return *f, nil

		case tea.KeyEsc:
			f.Active = false
			return *f, func() tea.Msg { return FormCancelMsg{} }
		}
	}

	// Delegate key event to the focused text input (if it's not the group cycle slot).
	if idx := f.textInputIndex(); idx >= 0 {
		var cmd tea.Cmd
		f.fields[idx], cmd = f.fields[idx].Update(msg)
		return *f, cmd
	}
	return *f, nil
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
			groupVal := ""
			if f.multiGroup && len(f.groupOptions) > 0 {
				groupVal = f.groupOptions[f.groupIdx]
			}

			macVal := strings.TrimSpace(f.fields[fieldMacKeys].Value())
			winVal := strings.TrimSpace(f.fields[fieldWinKeys].Value())
			tagsVal := f.fields[fieldTags].Value()

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
				Description: strings.TrimSpace(f.fields[fieldDesc].Value()),
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
	totalFields := f.fieldCount()
	for i := 0; i < totalFields; i++ {
		label := labels[i]
		isActive := i == f.focused

		if f.multiGroup && i == groupFieldIdx {
			// Render cycle selector.
			currentGroup := ""
			if len(f.groupOptions) > 0 {
				currentGroup = f.groupOptions[f.groupIdx]
			}
			if isActive {
				lines = append(lines, StyleKey.Render(label+":"))
				lines = append(lines, StyleKey.Render("◀  "+currentGroup+"  ▶"))
			} else {
				lines = append(lines, StyleMuted.Render(label+":"))
				lines = append(lines, StyleNormal.Render("   "+currentGroup))
			}
			lines = append(lines, "")
			continue
		}

		tiIdx := f.formIdxToTextInput(i)
		fi := f.fields[tiIdx]
		if isActive {
			lines = append(lines, StyleKey.Render(label+":"))
		} else {
			lines = append(lines, StyleMuted.Render(label+":"))
		}
		lines = append(lines, fi.View())
		lines = append(lines, "")
	}

	hint := "tab/↑↓ navigate  •  enter submit  •  esc cancel"
	if f.multiGroup {
		hint += "  •  ◀▶ cycle group"
	}
	lines = append(lines, StyleMuted.Render(hint))

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
