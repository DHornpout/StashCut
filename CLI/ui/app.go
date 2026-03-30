package ui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/stashcut/cli/keybindings"
	"github.com/stashcut/cli/model"
	"github.com/stashcut/cli/store"
)

type focusPanel int

const (
	panelSidebar focusPanel = iota
	panelShortcuts
)

// SavedMsg is sent after a successful background save.
type SavedMsg struct{}

// ImportErrMsg carries an error from a failed import.
type ImportErrMsg struct{ err error }

// ImportDoneMsg signals a successful import with counts.
type ImportDoneMsg struct {
	apps      int
	shortcuts int
}

type AppModel struct {
	data      *model.ShortcutFile
	dataPath  string
	dirty     bool
	osFilter  string // "all" | "mac" | "windows"
	focus     focusPanel
	sidebar   Sidebar
	list      ShortcutList
	search    Search
	form      *Form
	showHelp  bool
	statusMsg string
	width     int
	height    int
	// Command bar (activated by ":")
	cmdActive bool
	cmdInput  textinput.Model
}

func NewAppModel(data *model.ShortcutFile, dataPath string) AppModel {
	sortApps(data)

	sidebar := NewSidebar(data.Apps)
	sidebar.Focused = true
	list := NewShortcutList()
	list.OSFilter = "all"

	cmd := textinput.New()
	cmd.Placeholder = "command  (e.g. import ~/path/shortcuts.json)"
	cmd.CharLimit = 256

	m := AppModel{
		data:     data,
		dataPath: dataPath,
		osFilter: "all",
		focus:    panelSidebar,
		sidebar:  sidebar,
		list:     list,
		search:   NewSearch(),
		cmdInput: cmd,
	}
	m.refreshShortcuts()
	return m
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to form if active
	if m.form != nil && m.form.Active {
		return m.handleFormUpdate(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.SetHeight(m.height - 3)
		m.list.SetSize(m.width, m.height-3)
		m.search.SetWidth(m.width)
		return m, nil

	case FormSubmitMsg:
		return m.handleFormSubmit(msg)

	case FormCancelMsg:
		m.form = nil
		return m, nil

	case SavedMsg:
		m.dirty = false
		return m, nil

	case ImportDoneMsg:
		m.dirty = false
		m.sidebar.Apps = m.data.Apps
		m.refreshShortcuts()
		m.statusMsg = fmt.Sprintf("Imported: +%d apps, +%d shortcuts", msg.apps, msg.shortcuts)
		return m, nil

	case ImportErrMsg:
		m.statusMsg = "Import error: " + msg.err.Error()
		return m, nil

	case tea.KeyMsg:
		if m.cmdActive {
			return m.handleCmdKey(msg)
		}
		if m.search.Active {
			return m.handleSearchKey(msg)
		}
		return m.handleKey(msg)
	}

	return m, nil
}

func (m AppModel) handleFormUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	f, cmd := m.form.Update(msg)
	m.form = &f
	return m, cmd
}

func (m AppModel) handleCmdKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.cmdActive = false
		m.cmdInput.Blur()
		m.cmdInput.SetValue("")
		return m, nil
	case tea.KeyEnter:
		raw := strings.TrimSpace(m.cmdInput.Value())
		m.cmdActive = false
		m.cmdInput.Blur()
		m.cmdInput.SetValue("")
		return m.dispatchCmd(raw)
	}
	var cmd tea.Cmd
	m.cmdInput, cmd = m.cmdInput.Update(msg)
	return m, cmd
}

func (m AppModel) dispatchCmd(raw string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return m, nil
	}
	switch parts[0] {
	case "import":
		if len(parts) < 2 {
			m.statusMsg = "Usage: import <path>"
			return m, nil
		}
		path := expandHome(strings.Join(parts[1:], " "))
		return m, m.importFile(path)
	case "set-path":
		if len(parts) < 2 {
			m.statusMsg = "Usage: set-path <path>"
			return m, nil
		}
		m.dataPath = expandHome(strings.Join(parts[1:], " "))
		m.statusMsg = "Path set to: " + m.dataPath
		return m, m.save()
	default:
		m.statusMsg = "Unknown command: " + parts[0]
	}
	return m, nil
}

func (m AppModel) importFile(path string) tea.Cmd {
	data := m.data
	dataPath := m.dataPath
	prevApps := len(data.Apps)
	prevShortcuts := store.TotalShortcuts(data)
	return func() tea.Msg {
		incoming, err := store.Load(path)
		if err != nil {
			return ImportErrMsg{err: err}
		}
		if incoming == nil {
			return ImportErrMsg{err: fmt.Errorf("file not found: %s", path)}
		}
		store.Merge(data, incoming)
		if err := store.Save(dataPath, data); err != nil {
			return ImportErrMsg{err: err}
		}
		return ImportDoneMsg{
			apps:      len(data.Apps) - prevApps,
			shortcuts: store.TotalShortcuts(data) - prevShortcuts,
		}
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}

func (m AppModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
		m.search.Deactivate()
		m.refreshShortcuts()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.refreshShortcuts()
	return m, cmd
}

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := keybindings.Keys

	switch {
	case key.Matches(msg, kb.Quit):
		return m, tea.Quit

	case key.Matches(msg, kb.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, kb.Tab), key.Matches(msg, kb.ShiftTab):
		m = m.toggleFocus()
		return m, nil

	case key.Matches(msg, kb.Up):
		if m.focus == panelSidebar {
			m.sidebar.MoveUp()
			m.refreshShortcuts()
		} else {
			m.list.MoveUp()
		}
		return m, nil

	case key.Matches(msg, kb.Down):
		if m.focus == panelSidebar {
			m.sidebar.MoveDown()
			m.refreshShortcuts()
		} else {
			m.list.MoveDown()
		}
		return m, nil

	case key.Matches(msg, kb.Search):
		m.search.Activate()
		return m, nil

	case msg.String() == ":":
		m.cmdActive = true
		m.cmdInput.SetValue("")
		m.cmdInput.Focus()
		return m, nil

	case key.Matches(msg, kb.FilterMac):
		m.osFilter = "mac"
		m.list.OSFilter = "mac"
		m.statusMsg = "Filter: macOS"
		return m, nil

	case key.Matches(msg, kb.FilterWin):
		m.osFilter = "windows"
		m.list.OSFilter = "windows"
		m.statusMsg = "Filter: Windows"
		return m, nil

	case key.Matches(msg, kb.FilterAll):
		m.osFilter = "all"
		m.list.OSFilter = "all"
		m.statusMsg = "Filter: All"
		return m, nil

	case key.Matches(msg, kb.Favorite):
		if m.focus == panelShortcuts {
			return m.toggleFavorite()
		}

	case key.Matches(msg, kb.New):
		if m.focus == panelSidebar {
			f := NewAddAppForm()
			m.form = &f
		} else {
			app := m.sidebar.SelectedApp()
			if app != nil {
				groupNames := make([]string, len(app.Groups))
				for i, g := range app.Groups {
					groupNames[i] = g.Name
				}
				// Pre-select the group of the currently highlighted row.
				defaultGroup := ""
				if row := m.list.SelectedRow(); row != nil {
					defaultGroup = row.GroupName
				}
				f := NewAddShortcutForm(app.ID, groupNames, defaultGroup)
				m.form = &f
			}
		}
		return m, nil

	case key.Matches(msg, kb.NewGroup):
		if m.focus == panelShortcuts {
			app := m.sidebar.SelectedApp()
			if app != nil {
				f := NewAddGroupForm(app.ID)
				m.form = &f
			}
		}
		return m, nil

	case key.Matches(msg, kb.Edit):
		if m.focus == panelShortcuts {
			sc := m.list.SelectedShortcut()
			row := m.list.SelectedRow()
			if sc != nil {
				app := m.sidebar.SelectedApp()
				appID := ""
				groupNames := []string{}
				currentGroup := ""
				if app != nil {
					appID = app.ID
					groupNames = make([]string, len(app.Groups))
					for i, g := range app.Groups {
						groupNames[i] = g.Name
					}
				}
				if row != nil {
					currentGroup = row.GroupName
				}
				f := NewEditShortcutForm(appID, sc, groupNames, currentGroup)
				m.form = &f
			}
		}
		return m, nil

	case key.Matches(msg, kb.Delete):
		return m.handleDelete()

	case key.Matches(msg, kb.MoveUp):
		if m.focus == panelShortcuts {
			return m.moveShortcut(-1)
		}

	case key.Matches(msg, kb.MoveDown):
		if m.focus == panelShortcuts {
			return m.moveShortcut(1)
		}
	}

	return m, nil
}

func (m AppModel) toggleFocus() AppModel {
	if m.focus == panelSidebar {
		m.focus = panelShortcuts
		m.sidebar.Focused = false
		m.list.Focused = true
	} else {
		m.focus = panelSidebar
		m.sidebar.Focused = true
		m.list.Focused = false
	}
	return m
}

func (m AppModel) toggleFavorite() (tea.Model, tea.Cmd) {
	sc := m.list.SelectedShortcut()
	if sc == nil {
		return m, nil
	}
	ai, gi, si, ok := store.FindShortcut(m.data, sc.ID)
	if !ok {
		return m, nil
	}
	m.data.Apps[ai].Groups[gi].Shortcuts[si].IsFavorite = !m.data.Apps[ai].Groups[gi].Shortcuts[si].IsFavorite
	m.refreshShortcuts()
	return m, m.save()
}

func (m AppModel) handleDelete() (tea.Model, tea.Cmd) {
	if m.focus == panelSidebar {
		app := m.sidebar.SelectedApp()
		if app == nil {
			return m, nil
		}
		newApps := make([]model.App, 0, len(m.data.Apps))
		for _, a := range m.data.Apps {
			if a.ID != app.ID {
				newApps = append(newApps, a)
			}
		}
		m.data.Apps = newApps
		m.sidebar.Apps = newApps
		if m.sidebar.Selected >= len(newApps) && len(newApps) > 0 {
			m.sidebar.Selected = len(newApps) - 1
		}
		m.refreshShortcuts()
		m.statusMsg = fmt.Sprintf("Deleted app: %s", app.Name)
		return m, m.save()
	}

	if m.focus == panelShortcuts {
		row := m.list.SelectedRow()
		if row == nil {
			return m, nil
		}
		app := m.sidebar.SelectedApp()
		if app == nil {
			return m, nil
		}
		ai := findAppIndex(m.data, app.ID)
		if ai < 0 {
			return m, nil
		}

		if row.Kind == RowKindHeader {
			if row.GroupName == "Uncategorized" {
				m.statusMsg = "Cannot delete Uncategorized group"
				return m, nil
			}
			gi := store.FindGroup(&m.data.Apps[ai], row.GroupName)
			if gi < 0 {
				return m, nil
			}
			if len(m.data.Apps[ai].Groups[gi].Shortcuts) > 0 {
				m.statusMsg = "Group is not empty — delete its shortcuts first"
				return m, nil
			}
			groups := m.data.Apps[ai].Groups
			newGroups := make([]model.Group, 0, len(groups)-1)
			for _, g := range groups {
				if g.Name != row.GroupName {
					newGroups = append(newGroups, g)
				}
			}
			m.data.Apps[ai].Groups = newGroups
			m.refreshShortcuts()
			m.statusMsg = fmt.Sprintf("Deleted group: %s", row.GroupName)
			return m, m.save()
		}

		// Delete shortcut
		sc := m.list.SelectedShortcut()
		if sc == nil {
			return m, nil
		}
		_, gi, _, ok := store.FindShortcut(m.data, sc.ID)
		if !ok {
			return m, nil
		}
		shortcuts := m.data.Apps[ai].Groups[gi].Shortcuts
		newShortcuts := make([]model.Shortcut, 0, len(shortcuts))
		for _, s := range shortcuts {
			if s.ID != sc.ID {
				newShortcuts = append(newShortcuts, s)
			}
		}
		m.data.Apps[ai].Groups[gi].Shortcuts = newShortcuts
		m.refreshShortcuts()
		m.statusMsg = "Shortcut deleted"
		return m, m.save()
	}

	return m, nil
}

func (m AppModel) moveShortcut(direction int) (tea.Model, tea.Cmd) {
	sc := m.list.SelectedShortcut()
	if sc == nil {
		return m, nil
	}
	ai, gi, si, ok := store.FindShortcut(m.data, sc.ID)
	if !ok {
		return m, nil
	}

	group := m.data.Apps[ai].Groups[gi].Shortcuts
	newSi := si + direction
	if newSi < 0 || newSi >= len(group) {
		return m, nil
	}

	group[si].SortOrder, group[newSi].SortOrder = group[newSi].SortOrder, group[si].SortOrder
	m.data.Apps[ai].Groups[gi].Shortcuts = group

	m.refreshShortcuts()

	// Restore selection to the moved shortcut.
	for i, r := range m.list.Rows {
		if r.Kind == RowKindShortcut && r.Shortcut.ID == sc.ID {
			m.list.Selected = i
			break
		}
	}
	return m, m.save()
}

func (m AppModel) handleFormSubmit(msg FormSubmitMsg) (tea.Model, tea.Cmd) {
	var editOriginalID string
	if m.form != nil && m.form.Original != nil {
		editOriginalID = m.form.Original.ID
	}
	m.form = nil
	now := time.Now()

	switch msg.Mode {
	case FormModeAddApp:
		if msg.App == nil || msg.App.Name == "" {
			return m, nil
		}
		app := *msg.App
		app.ID = uuid.New().String()
		app.SortOrder = len(m.data.Apps)
		app.CreatedAt = now
		app.UpdatedAt = now
		app.Groups = []model.Group{{Name: "Uncategorized", Shortcuts: []model.Shortcut{}}}
		m.data.Apps = append(m.data.Apps, app)
		sortApps(m.data)
		m.sidebar.Apps = m.data.Apps
		for i, a := range m.data.Apps {
			if a.ID == app.ID {
				m.sidebar.Selected = i
				break
			}
		}
		m.refreshShortcuts()
		m.statusMsg = fmt.Sprintf("Added app: %s", app.Name)

	case FormModeAddGroup:
		if msg.GroupName == "" {
			return m, nil
		}
		ai := findAppIndex(m.data, msg.AppID)
		if ai < 0 {
			return m, nil
		}
		if store.FindGroup(&m.data.Apps[ai], msg.GroupName) >= 0 {
			m.statusMsg = "Group already exists: " + msg.GroupName
			return m, nil
		}
		m.data.Apps[ai].Groups = append(m.data.Apps[ai].Groups, model.Group{
			Name:      msg.GroupName,
			Shortcuts: []model.Shortcut{},
		})
		m.data.Apps[ai].UpdatedAt = now
		m.refreshShortcuts()
		m.statusMsg = "Group added: " + msg.GroupName

	case FormModeAddShortcut:
		if msg.Shortcut == nil || msg.Shortcut.Description == "" {
			return m, nil
		}
		app := m.sidebar.SelectedApp()
		if app == nil {
			return m, nil
		}
		ai := findAppIndex(m.data, app.ID)
		if ai < 0 {
			return m, nil
		}

		targetGroup := msg.GroupName
		if targetGroup == "" {
			targetGroup = "Uncategorized"
		}

		gi := store.FindGroup(&m.data.Apps[ai], targetGroup)
		if gi < 0 {
			// Auto-create the group.
			m.data.Apps[ai].Groups = append(m.data.Apps[ai].Groups, model.Group{
				Name:      targetGroup,
				Shortcuts: []model.Shortcut{},
			})
			gi = len(m.data.Apps[ai].Groups) - 1
		}

		sc := *msg.Shortcut
		sc.ID = uuid.New().String()
		sc.SortOrder = len(m.data.Apps[ai].Groups[gi].Shortcuts)
		sc.CreatedAt = now
		sc.UpdatedAt = now
		if sc.Tags == nil {
			sc.Tags = []string{}
		}
		m.data.Apps[ai].Groups[gi].Shortcuts = append(m.data.Apps[ai].Groups[gi].Shortcuts, sc)
		m.data.Apps[ai].UpdatedAt = now
		m.refreshShortcuts()
		m.statusMsg = "Shortcut added"

	case FormModeEditShortcut:
		if msg.Shortcut != nil && editOriginalID != "" {
			ai, gi, si, ok := store.FindShortcut(m.data, editOriginalID)
			if ok {
				sc := m.data.Apps[ai].Groups[gi].Shortcuts[si]
				sc.Description = msg.Shortcut.Description
				sc.KeysByOS = msg.Shortcut.KeysByOS
				sc.Tags = msg.Shortcut.Tags
				sc.UpdatedAt = now

				targetGroup := msg.GroupName
				if targetGroup == "" {
					targetGroup = m.data.Apps[ai].Groups[gi].Name // no group field shown — keep current
				}

				if targetGroup != m.data.Apps[ai].Groups[gi].Name {
					// Remove from current group.
					cur := m.data.Apps[ai].Groups[gi].Shortcuts
					newCur := make([]model.Shortcut, 0, len(cur)-1)
					for _, s := range cur {
						if s.ID != editOriginalID {
							newCur = append(newCur, s)
						}
					}
					m.data.Apps[ai].Groups[gi].Shortcuts = newCur

					// Find or create the target group.
					tgi := store.FindGroup(&m.data.Apps[ai], targetGroup)
					if tgi < 0 {
						m.data.Apps[ai].Groups = append(m.data.Apps[ai].Groups, model.Group{
							Name: targetGroup, Shortcuts: []model.Shortcut{},
						})
						tgi = len(m.data.Apps[ai].Groups) - 1
					}
					sc.SortOrder = len(m.data.Apps[ai].Groups[tgi].Shortcuts)
					m.data.Apps[ai].Groups[tgi].Shortcuts = append(m.data.Apps[ai].Groups[tgi].Shortcuts, sc)
				} else {
					m.data.Apps[ai].Groups[gi].Shortcuts[si] = sc
				}
				m.data.Apps[ai].UpdatedAt = now
			}
			m.refreshShortcuts()
			m.statusMsg = "Shortcut updated"
		}
	}

	return m, m.save()
}

func (m *AppModel) refreshShortcuts() {
	query := m.search.Query()

	if m.search.Active && query != "" {
		// Build app name lookup keyed by shortcut ID for display.
		appNames := make(map[string]string)
		groupNames := make(map[string]string)
		for _, a := range m.data.Apps {
			displayName := a.Name
			if a.Icon != "" {
				displayName = a.Icon + " " + a.Name
			}
			for _, g := range a.Groups {
				for _, sc := range g.Shortcuts {
					appNames[sc.ID] = displayName
					groupNames[sc.ID] = g.Name
				}
			}
		}

		rows := FilterApps(m.data.Apps, query)

		sort.Slice(rows, func(i, j int) bool {
			if rows[i].Shortcut.IsFavorite != rows[j].Shortcut.IsFavorite {
				return rows[i].Shortcut.IsFavorite
			}
			return rows[i].GroupName < rows[j].GroupName
		})

		m.list.AppNames = appNames
		m.list.GroupNames = groupNames
		m.list.SearchMode = true
		m.list.SetRows(rows)
		return
	}

	// Normal mode: show selected app's shortcuts, grouped.
	m.list.SearchMode = false
	m.list.AppNames = nil
	m.list.GroupNames = nil

	app := m.sidebar.SelectedApp()
	if app == nil {
		m.list.SetRows(nil)
		return
	}

	var rows []ListRow
	for gi, grp := range app.Groups {
		rows = append(rows, ListRow{
			Kind:       RowKindHeader,
			GroupName:  grp.Name,
			GroupIndex: gi,
		})

		sorted := make([]model.Shortcut, len(grp.Shortcuts))
		copy(sorted, grp.Shortcuts)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].IsFavorite != sorted[j].IsFavorite {
				return sorted[i].IsFavorite
			}
			return sorted[i].SortOrder < sorted[j].SortOrder
		})
		for si, sc := range sorted {
			rows = append(rows, ListRow{
				Kind:        RowKindShortcut,
				GroupName:   grp.Name,
				Shortcut:    sc,
				GroupIndex:  gi,
				ShortcutIdx: si,
			})
		}
	}
	m.list.SetRows(rows)
}

func (m AppModel) save() tea.Cmd {
	m.dirty = true
	data := m.data
	path := m.dataPath
	return func() tea.Msg {
		store.Save(path, data)
		return SavedMsg{}
	}
}

func (m AppModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	if m.form != nil && m.form.Active {
		return m.form.View(m.width)
	}

	if m.showHelp {
		return m.helpView()
	}

	m.sidebar.Focused = m.focus == panelSidebar
	m.list.Focused = m.focus == panelShortcuts

	appName := ""
	if app := m.sidebar.SelectedApp(); app != nil {
		icon := app.Icon
		if icon != "" {
			appName = icon + " " + app.Name
		} else {
			appName = app.Name
		}
	}

	sidebarView := m.sidebar.View()
	listView := m.list.View(appName)

	var topBar string
	if m.cmdActive {
		cmdStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Foreground(lipgloss.Color("#E5E7EB")).
			Padding(0, 1).
			Width(m.width - 2)
		topBar = cmdStyle.Render(StyleKey.Render(": ") + m.cmdInput.View())
	} else {
		topBar = m.search.View()
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, listView)

	osLabel := map[string]string{
		"all":     "All",
		"mac":     "macOS",
		"windows": "Windows",
	}[m.osFilter]

	dirtyMark := ""
	if m.dirty {
		dirtyMark = StyleError.Render(" [unsaved]")
	}

	statusLeft := fmt.Sprintf(" %s%s  OS: %s", m.dataPath, dirtyMark, osLabel)
	statusRight := m.statusMsg + "  ? help  q quit "
	leftLen := len([]rune(m.dataPath)) + len("  OS: ") + len(osLabel) + 1
	if m.dirty {
		leftLen += len(" [unsaved]")
	}
	statusPad := m.width - leftLen - len([]rune(statusRight))
	if statusPad < 1 {
		statusPad = 1
	}
	statusBar := StyleStatusBar.Width(m.width).Render(
		statusLeft + strings.Repeat(" ", statusPad) + statusRight,
	)

	return topBar + "\n" + panels + "\n" + statusBar
}

func (m AppModel) helpView() string {
	help := `
  StashCut CLI — Key Bindings

  Navigation
    Tab / Shift+Tab    Switch panel
    ↑ / ↓  or  k / j  Navigate list

  Actions
    n        New app (sidebar) / New shortcut (list)
    g        New group (shortcut panel)
    e        Edit selected shortcut
    d        Delete selected item (shortcut or empty group)
    f        Toggle favorite on shortcut
    J / K    Move shortcut down / up (reorder)

  Filter & Search
    /        Search (description + tags)
    m        Filter: macOS
    w        Filter: Windows
    a        Filter: All platforms

  Commands  (press : to open)
    :import <path>     Merge another shortcuts.json
    :set-path <path>   Change the active file path

  General
    ?        Toggle this help
    q        Quit
`
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorPrimary).
		Padding(1, 3).
		Render(help)
}

func sortApps(data *model.ShortcutFile) {
	sort.Slice(data.Apps, func(i, j int) bool {
		return data.Apps[i].SortOrder < data.Apps[j].SortOrder
	})
}

// findAppIndex returns the index of the app with the given ID, or -1.
func findAppIndex(sf *model.ShortcutFile, appID string) int {
	for i, a := range sf.Apps {
		if a.ID == appID {
			return i
		}
	}
	return -1
}
