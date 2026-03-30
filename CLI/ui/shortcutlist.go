package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/stashcut/cli/model"
)

// RowKind distinguishes group header rows from shortcut rows in the list.
type RowKind int

const (
	RowKindHeader   RowKind = iota
	RowKindShortcut
)

// ListRow is a single renderable row — either a group header or a shortcut.
type ListRow struct {
	Kind        RowKind
	GroupName   string
	Shortcut    model.Shortcut // valid when Kind == RowKindShortcut
	GroupIndex  int
	ShortcutIdx int
}

type ShortcutList struct {
	Rows       []ListRow
	Selected   int // always points to a RowKindShortcut when Rows is non-empty
	Focused    bool
	OSFilter   string            // "mac" | "windows" | "all"
	SearchMode bool
	AppNames   map[string]string // shortcutID -> app display name, used in search mode
	GroupNames map[string]string // shortcutID -> groupName, used in search mode
	width      int
	height     int
	offset     int
}

func NewShortcutList() ShortcutList {
	return ShortcutList{OSFilter: "all"}
}

func (sl *ShortcutList) SetSize(w, h int) {
	sl.width = w
	sl.height = h
}

func (sl *ShortcutList) SetRows(rows []ListRow) {
	sl.Rows = rows
	sl.Selected = sl.firstSelectableRow()
	sl.offset = 0
}

// firstSelectableRow returns the index of the first RowKindShortcut, or 0.
func (sl *ShortcutList) firstSelectableRow() int {
	for i, r := range sl.Rows {
		if r.Kind == RowKindShortcut {
			return i
		}
	}
	return 0
}

func (sl *ShortcutList) MoveUp() {
	for i := sl.Selected - 1; i >= 0; i-- {
		if sl.Rows[i].Kind == RowKindShortcut {
			sl.Selected = i
			sl.clampOffset()
			return
		}
	}
}

func (sl *ShortcutList) MoveDown() {
	for i := sl.Selected + 1; i < len(sl.Rows); i++ {
		if sl.Rows[i].Kind == RowKindShortcut {
			sl.Selected = i
			sl.clampOffset()
			return
		}
	}
}

func (sl *ShortcutList) SelectedShortcut() *model.Shortcut {
	if sl.Selected < 0 || sl.Selected >= len(sl.Rows) {
		return nil
	}
	row := sl.Rows[sl.Selected]
	if row.Kind != RowKindShortcut {
		return nil
	}
	sc := sl.Rows[sl.Selected].Shortcut
	return &sc
}

func (sl *ShortcutList) SelectedRow() *ListRow {
	if sl.Selected >= 0 && sl.Selected < len(sl.Rows) {
		return &sl.Rows[sl.Selected]
	}
	return nil
}

func (sl *ShortcutList) clampOffset() {
	visible := sl.visibleRows()
	if sl.Selected < sl.offset {
		sl.offset = sl.Selected
	} else if sl.Selected >= sl.offset+visible {
		sl.offset = sl.Selected - visible + 1
	}
}

func (sl *ShortcutList) visibleRows() int {
	// content height = height - 2 (border)
	// overhead: title(1) + title margin(1) + col header(1) + divider(1) = 4
	rows := sl.height - 6
	if rows < 1 {
		rows = 1
	}
	return rows
}

// countShortcutRows returns the number of RowKindShortcut rows.
func (sl *ShortcutList) countShortcutRows() int {
	n := 0
	for _, r := range sl.Rows {
		if r.Kind == RowKindShortcut {
			n++
		}
	}
	return n
}

// keyFor returns the display string for a shortcut on a given OS key.
func (sl ShortcutList) keyFor(s model.Shortcut, osKey string) string {
	if kfo, ok := s.KeysByOS[osKey]; ok {
		if kfo.KeysDisplay != "" {
			return kfo.KeysDisplay
		}
		return kfo.Keys
	}
	return ""
}

// colLayout computes column widths given the available panel width.
type colLayout struct {
	panelW int
	favW   int
	appW   int // 0 when not in search mode
	groupW int // 0 when not in search mode
	descW  int
	macW   int // 0 when OS filter hides it
	winW   int // 0 when OS filter hides it
}

func (sl ShortcutList) layout() colLayout {
	panelW := sl.width - SidebarWidth - 4
	if panelW < 50 {
		panelW = 50
	}

	const (
		favFixed   = 2
		appFixed   = 14
		groupFixed = 12
		macFixed   = 18
		winFixed   = 18
		sepW       = 3 // " │ "
	)

	showMac := sl.OSFilter == "all" || sl.OSFilter == "mac"
	showWin := sl.OSFilter == "all" || sl.OSFilter == "windows"

	used := favFixed + sepW
	if sl.SearchMode {
		used += appFixed + sepW
		used += groupFixed + sepW
	}
	if showMac {
		used += macFixed + sepW
	}
	if showWin {
		used += winFixed + sepW
	}

	descW := panelW - used - 2
	if descW < 15 {
		descW = 15
	}

	macW, winW := 0, 0
	if showMac {
		macW = macFixed
	}
	if showWin {
		winW = winFixed
	}
	appW, groupW := 0, 0
	if sl.SearchMode {
		appW = appFixed
		groupW = groupFixed
	}

	return colLayout{
		panelW: panelW,
		favW:   favFixed,
		appW:   appW,
		groupW: groupW,
		descW:  descW,
		macW:   macW,
		winW:   winW,
	}
}

// truncPad truncates s to w runes (with ellipsis) and pads to exactly w.
func truncPad(s string, w int) string {
	runes := []rune(s)
	if len(runes) > w {
		return string(runes[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(runes))
}

func (sl ShortcutList) View(appName string) string {
	style := StylePanel
	if sl.Focused {
		style = StylePanelFocused
	}

	titleText := appName
	if sl.SearchMode {
		titleText = fmt.Sprintf("Search Results (%d)", sl.countShortcutRows())
	}
	title := StyleTitle.Render(titleText)

	lo := sl.layout()
	sep := StyleMuted.Render(" │ ")

	// ── Column header row ────────────────────────────────────────────────────
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted)

	headerCells := []string{truncPad("", lo.favW)}
	if lo.appW > 0 {
		headerCells = append(headerCells, hdrStyle.Render(truncPad("App", lo.appW)))
	}
	if lo.groupW > 0 {
		headerCells = append(headerCells, hdrStyle.Render(truncPad("Group", lo.groupW)))
	}
	headerCells = append(headerCells, hdrStyle.Render(truncPad("Description", lo.descW)))
	if lo.macW > 0 {
		headerCells = append(headerCells, hdrStyle.Render(truncPad("Mac", lo.macW)))
	}
	if lo.winW > 0 {
		headerCells = append(headerCells, hdrStyle.Render(truncPad("Windows", lo.winW)))
	}
	header := strings.Join(headerCells, sep)

	// ── Divider ─────────────────────────────────────────────────────────────
	divCells := []string{strings.Repeat("─", lo.favW)}
	if lo.appW > 0 {
		divCells = append(divCells, strings.Repeat("─", lo.appW))
	}
	if lo.groupW > 0 {
		divCells = append(divCells, strings.Repeat("─", lo.groupW))
	}
	divCells = append(divCells, strings.Repeat("─", lo.descW))
	if lo.macW > 0 {
		divCells = append(divCells, strings.Repeat("─", lo.macW))
	}
	if lo.winW > 0 {
		divCells = append(divCells, strings.Repeat("─", lo.winW))
	}
	divider := StyleMuted.Render(strings.Join(divCells, "─┼─"))

	// ── Data rows ────────────────────────────────────────────────────────────
	visible := sl.visibleRows()
	end := sl.offset + visible
	if end > len(sl.Rows) {
		end = len(sl.Rows)
	}

	// Pre-compute alternating-row parity for every row across the full list so
	// the zebra pattern is stable even when headers are scrolled off-screen.
	altRow := make([]bool, len(sl.Rows))
	grpPos := 0
	for idx, r := range sl.Rows {
		if r.Kind == RowKindHeader {
			grpPos = 0
			continue
		}
		altRow[idx] = grpPos%2 == 1
		grpPos++
	}

	// Content width for the group section header background fill.
	contentW := lo.favW + lo.descW + 3
	if lo.appW > 0 {
		contentW += lo.appW + 3
	}
	if lo.groupW > 0 {
		contentW += lo.groupW + 3
	}
	if lo.macW > 0 {
		contentW += lo.macW + 3
	}
	if lo.winW > 0 {
		contentW += lo.winW + 3
	}

	dataRows := make([]string, 0, visible)
	for i := sl.offset; i < end; i++ {
		row := sl.Rows[i]

		if row.Kind == RowKindHeader {
			// Full-width section header bar.
			dataRows = append(dataRows, StyleGroupSectionHeader.Width(contentW).Render("▸ "+row.GroupName))
			continue
		}

		s := row.Shortcut
		isSelected := i == sl.Selected
		isFocused := isSelected && sl.Focused
		isAlt := altRow[i]

		favCell := truncPad("", lo.favW)
		if s.IsFavorite {
			favCell = "★ "
		}

		appCell := ""
		if lo.appW > 0 {
			appCell = truncPad(sl.AppNames[s.ID], lo.appW)
		}
		groupCell := ""
		if lo.groupW > 0 {
			gn := row.GroupName
			if sl.GroupNames != nil {
				if gn2, ok := sl.GroupNames[s.ID]; ok {
					gn = gn2
				}
			}
			groupCell = truncPad(gn, lo.groupW)
		}
		descCell := truncPad(s.Description, lo.descW)
		macCell := ""
		if lo.macW > 0 {
			macCell = truncPad(sl.keyFor(s, "macos"), lo.macW)
		}
		winCell := ""
		if lo.winW > 0 {
			winCell = truncPad(sl.keyFor(s, "windows"), lo.winW)
		}

		if isFocused {
			cells := []string{favCell}
			if lo.appW > 0 {
				cells = append(cells, appCell)
			}
			if lo.groupW > 0 {
				cells = append(cells, groupCell)
			}
			cells = append(cells, descCell)
			if lo.macW > 0 {
				cells = append(cells, macCell)
			}
			if lo.winW > 0 {
				cells = append(cells, winCell)
			}
			dataRows = append(dataRows, StyleSelected.Render(strings.Join(cells, " │ ")))
		} else {
			// Build per-cell styles that carry the alt background when needed.
			// Each cell must include Background so the entire row is filled.
			mutedS := StyleMuted
			normalS := StyleNormal
			keyS := StyleKey
			favS := StyleFavorite
			plainS := lipgloss.NewStyle()
			sepStr := sep
			if isAlt {
				mutedS = lipgloss.NewStyle().Foreground(colorMuted).Background(colorRowAlt)
				normalS = lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Background(colorRowAlt)
				keyS = lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Background(colorRowAlt)
				favS = lipgloss.NewStyle().Foreground(colorFav).Background(colorRowAlt)
				plainS = lipgloss.NewStyle().Background(colorRowAlt)
				sepStr = mutedS.Render(" │ ")
			}
			if isSelected {
				accentS := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
				if isAlt {
					accentS = accentS.Background(colorRowAlt)
				}
				normalS = accentS
				keyS = accentS
			}

			styledFav := favS.Render(favCell)
			if !s.IsFavorite {
				styledFav = plainS.Render(favCell)
			}

			cells := []string{styledFav}
			if lo.appW > 0 {
				cells = append(cells, mutedS.Render(appCell))
			}
			if lo.groupW > 0 {
				cells = append(cells, mutedS.Render(groupCell))
			}
			cells = append(cells, normalS.Render(descCell))
			if lo.macW > 0 {
				cells = append(cells, keyS.Render(macCell))
			}
			if lo.winW > 0 {
				cells = append(cells, keyS.Render(winCell))
			}

			dataRows = append(dataRows, strings.Join(cells, sepStr))
		}
	}

	if len(sl.Rows) == 0 || sl.countShortcutRows() == 0 {
		if sl.SearchMode {
			dataRows = append(dataRows, StyleMuted.Render("No matches found."))
		} else {
			dataRows = append(dataRows, StyleMuted.Render("No shortcuts yet — press n to add."))
		}
	}

	content := strings.Join(
		append([]string{title, header, divider}, dataRows...),
		"\n",
	)
	return style.Width(lo.panelW).Height(sl.height - 2).Render(content)
}
