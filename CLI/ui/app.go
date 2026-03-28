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
		// Layout: searchBar(1) + panels(h) + newline(1) + statusBar(1) = m.height
		// → panels height = m.height - 3
		// sidebar/list internal: subtract 1 more for the "\n" before statusBar
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
		// Command bar eats keys
		if m.cmdActive {
			return m.handleCmdKey(msg)
		}
		// Search mode eats keys
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
	return func() tea.Msg {
		incoming, err := store.Load(path)
		if err != nil {
			return ImportErrMsg{err: err}
		}
		if incoming == nil {
			return ImportErrMsg{err: fmt.Errorf("file not found: %s", path)}
		}
		prevApps := len(data.Apps)
		prevShortcuts := len(data.Shortcuts)
		store.Merge(data, incoming)
		if err := store.Save(dataPath, data); err != nil {
			return ImportErrMsg{err: err}
		}
		return ImportDoneMsg{
			apps:      len(data.Apps) - prevApps,
			shortcuts: len(data.Shortcuts) - prevShortcuts,
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
				f := NewAddShortcutForm(app.ID)
				m.form = &f
			}
		}
		return m, nil

	case key.Matches(msg, kb.Edit):
		if m.focus == panelShortcuts {
			sc := m.list.SelectedShortcut()
			if sc != nil {
				app := m.sidebar.SelectedApp()
				appID := ""
				if app != nil {
					appID = app.ID
				}
				f := NewEditShortcutForm(appID, sc)
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
	for i, s := range m.data.Shortcuts {
		if s.ID == sc.ID {
			m.data.Shortcuts[i].IsFavorite = !m.data.Shortcuts[i].IsFavorite
			break
		}
	}
	m.refreshShortcuts()
	return m, m.save()
}

func (m AppModel) handleDelete() (tea.Model, tea.Cmd) {
	if m.focus == panelSidebar {
		app := m.sidebar.SelectedApp()
		if app == nil {
			return m, nil
		}
		// Remove app and its shortcuts
		newApps := make([]model.App, 0, len(m.data.Apps))
		for _, a := range m.data.Apps {
			if a.ID != app.ID {
				newApps = append(newApps, a)
			}
		}
		newShortcuts := make([]model.Shortcut, 0)
		for _, s := range m.data.Shortcuts {
			if s.AppID != app.ID {
				newShortcuts = append(newShortcuts, s)
			}
		}
		m.data.Apps = newApps
		m.data.Shortcuts = newShortcuts
		m.sidebar.Apps = newApps
		if m.sidebar.Selected >= len(newApps) && len(newApps) > 0 {
			m.sidebar.Selected = len(newApps) - 1
		}
		m.refreshShortcuts()
		m.statusMsg = fmt.Sprintf("Deleted app: %s", app.Name)
		return m, m.save()
	}

	if m.focus == panelShortcuts {
		sc := m.list.SelectedShortcut()
		if sc == nil {
			return m, nil
		}
		newShortcuts := make([]model.Shortcut, 0, len(m.data.Shortcuts))
		for _, s := range m.data.Shortcuts {
			if s.ID != sc.ID {
				newShortcuts = append(newShortcuts, s)
			}
		}
		m.data.Shortcuts = newShortcuts
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
	app := m.sidebar.SelectedApp()
	if app == nil {
		return m, nil
	}

	// Collect app shortcuts in order
	appShortcuts := store.ShortcutsForApp(m.data, app.ID)
	idx := -1
	for i, s := range appShortcuts {
		if s.ID == sc.ID {
			idx = i
			break
		}
	}
	newIdx := idx + direction
	if newIdx < 0 || newIdx >= len(appShortcuts) {
		return m, nil
	}

	// Swap sort orders
	aID := appShortcuts[idx].ID
	bID := appShortcuts[newIdx].ID
	aOrder := appShortcuts[idx].SortOrder
	bOrder := appShortcuts[newIdx].SortOrder
	for i := range m.data.Shortcuts {
		if m.data.Shortcuts[i].ID == aID {
			m.data.Shortcuts[i].SortOrder = bOrder
		} else if m.data.Shortcuts[i].ID == bID {
			m.data.Shortcuts[i].SortOrder = aOrder
		}
	}

	m.refreshShortcuts()
	m.list.Selected = newIdx
	return m, m.save()
}

func getSortOrders(shortcuts []model.Shortcut, id string) (int, bool) {
	for _, s := range shortcuts {
		if s.ID == id {
			return s.SortOrder, true
		}
	}
	return 0, false
}

func (m AppModel) handleFormSubmit(msg FormSubmitMsg) (tea.Model, tea.Cmd) {
	// Capture edit original ID before clearing the form
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
		m.data.Apps = append(m.data.Apps, app)
		sortApps(m.data)
		m.sidebar.Apps = m.data.Apps
		// Select the new app
		for i, a := range m.data.Apps {
			if a.ID == app.ID {
				m.sidebar.Selected = i
				break
			}
		}
		m.refreshShortcuts()
		m.statusMsg = fmt.Sprintf("Added app: %s", app.Name)

	case FormModeAddShortcut:
		if msg.Shortcut == nil || msg.Shortcut.Description == "" {
			return m, nil
		}
		sc := *msg.Shortcut
		sc.ID = uuid.New().String()
		sc.SortOrder = len(store.ShortcutsForApp(m.data, sc.AppID))
		sc.CreatedAt = now
		sc.UpdatedAt = now
		if sc.Tags == nil {
			sc.Tags = []string{}
		}
		m.data.Shortcuts = append(m.data.Shortcuts, sc)
		m.refreshShortcuts()
		m.statusMsg = "Shortcut added"

	case FormModeEditShortcut:
		if msg.Shortcut != nil && editOriginalID != "" {
			sc := msg.Shortcut
			for i, s := range m.data.Shortcuts {
				if s.ID == editOriginalID {
					m.data.Shortcuts[i].Description = sc.Description
					m.data.Shortcuts[i].KeysByOS = sc.KeysByOS
					m.data.Shortcuts[i].Tags = sc.Tags
					m.data.Shortcuts[i].UpdatedAt = now
					break
				}
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
		// Search across all apps
		appNames := make(map[string]string, len(m.data.Apps))
		for _, a := range m.data.Apps {
			name := a.Name
			if a.Icon != "" {
				name = a.Icon + " " + name
			}
			appNames[a.ID] = name
		}
		results := Filter(m.data.Shortcuts, query)
		sort.Slice(results, func(i, j int) bool {
			if results[i].IsFavorite != results[j].IsFavorite {
				return results[i].IsFavorite
			}
			return results[i].AppID < results[j].AppID
		})
		m.list.AppNames = appNames
		m.list.SearchMode = true
		m.list.SetShortcuts(results)
		return
	}

	// Normal mode: show selected app's shortcuts
	m.list.SearchMode = false
	m.list.AppNames = nil

	app := m.sidebar.SelectedApp()
	if app == nil {
		m.list.SetShortcuts(nil)
		return
	}

	shortcuts := store.ShortcutsForApp(m.data, app.ID)
	sort.Slice(shortcuts, func(i, j int) bool {
		if shortcuts[i].IsFavorite != shortcuts[j].IsFavorite {
			return shortcuts[i].IsFavorite
		}
		return shortcuts[i].SortOrder < shortcuts[j].SortOrder
	})
	m.list.SetShortcuts(shortcuts)
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

	// Form overlay
	if m.form != nil && m.form.Active {
		return m.form.View(m.width)
	}

	// Help overlay
	if m.showHelp {
		return m.helpView()
	}

	// Set focus flags
	m.sidebar.Focused = m.focus == panelSidebar
	m.list.Focused = m.focus == panelShortcuts

	// Layout
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

	// Top bar: search (always 1 line) or command input
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

	// Status bar
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
	// Use rune length for proper padding
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
    e        Edit selected shortcut
    d        Delete selected item
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
