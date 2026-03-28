package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/stashcut/cli/model"
)

type ShortcutList struct {
	Shortcuts  []model.Shortcut
	Selected   int
	Focused    bool
	OSFilter   string // "mac" | "windows" | "all"
	SearchMode bool
	AppNames   map[string]string // appID -> display name, used in search mode
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

func (sl *ShortcutList) SetShortcuts(shortcuts []model.Shortcut) {
	sl.Shortcuts = shortcuts
	if sl.Selected >= len(sl.Shortcuts) {
		sl.Selected = len(sl.Shortcuts) - 1
	}
	if sl.Selected < 0 {
		sl.Selected = 0
	}
	sl.offset = 0
}

func (sl *ShortcutList) MoveUp() {
	if sl.Selected > 0 {
		sl.Selected--
		sl.clampOffset()
	}
}

func (sl *ShortcutList) MoveDown() {
	if sl.Selected < len(sl.Shortcuts)-1 {
		sl.Selected++
		sl.clampOffset()
	}
}

func (sl *ShortcutList) SelectedShortcut() *model.Shortcut {
	if len(sl.Shortcuts) == 0 || sl.Selected >= len(sl.Shortcuts) {
		return nil
	}
	return &sl.Shortcuts[sl.Selected]
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

// keyFor returns the display string for a shortcut on a given OS key ("macos" or "windows").
func (sl ShortcutList) keyFor(s model.Shortcut, osKey string) string {
	if kfo, ok := s.KeysByOS[osKey]; ok {
		if kfo.KeysDisplay != "" {
			return kfo.KeysDisplay
		}
		return kfo.Keys
	}
	return ""
}

// colWidths computes column widths given the available panel width.
type colLayout struct {
	panelW int
	favW   int
	appW   int // 0 when not in search mode
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
		favFixed = 2
		appFixed = 14
		macFixed = 18
		winFixed = 18
		sepW     = 3 // " │ "
	)

	showMac := sl.OSFilter == "all" || sl.OSFilter == "mac"
	showWin := sl.OSFilter == "all" || sl.OSFilter == "windows"

	used := favFixed + sepW // fav col + first separator
	if sl.SearchMode {
		used += appFixed + sepW
	}
	if showMac {
		used += macFixed + sepW
	}
	if showWin {
		used += winFixed + sepW
	}

	// StylePanel has Padding(0,1): content wraps at panelW-2, so rows must be panelW-2 wide.
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
	appW := 0
	if sl.SearchMode {
		appW = appFixed
	}

	return colLayout{
		panelW: panelW,
		favW:   favFixed,
		appW:   appW,
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
		titleText = fmt.Sprintf("Search Results (%d)", len(sl.Shortcuts))
	}
	title := StyleTitle.Render(titleText)

	lo := sl.layout()
	sep := StyleMuted.Render(" │ ")

	// ── Header row ──────────────────────────────────────────────────────────
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(colorMuted)

	headerCells := []string{truncPad("", lo.favW)}
	if lo.appW > 0 {
		headerCells = append(headerCells, hdrStyle.Render(truncPad("App", lo.appW)))
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
	if end > len(sl.Shortcuts) {
		end = len(sl.Shortcuts)
	}

	dataRows := make([]string, 0, visible)
	for i := sl.offset; i < end; i++ {
		s := sl.Shortcuts[i]
		isSelected := i == sl.Selected
		isFocused := isSelected && sl.Focused

		favCell := truncPad("", lo.favW)
		if s.IsFavorite {
			favCell = "★ "
		}

		appCell := ""
		if lo.appW > 0 {
			appCell = truncPad(sl.AppNames[s.AppID], lo.appW)
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
			// Selected + focused: plain text, let StyleSelected fill background
			cells := []string{favCell}
			if lo.appW > 0 {
				cells = append(cells, appCell)
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
			// Compose styled cells individually
			styledFav := StyleFavorite.Render(favCell)
			if !s.IsFavorite {
				styledFav = favCell
			}

			cells := []string{styledFav}
			if lo.appW > 0 {
				cells = append(cells, StyleMuted.Render(appCell))
			}

			descS := StyleNormal
			if isSelected {
				descS = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
			}
			cells = append(cells, descS.Render(descCell))

			keyS := StyleKey
			if isSelected {
				keyS = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
			}
			if lo.macW > 0 {
				cells = append(cells, keyS.Render(macCell))
			}
			if lo.winW > 0 {
				cells = append(cells, keyS.Render(winCell))
			}

			dataRows = append(dataRows, strings.Join(cells, sep))
		}
	}

	if len(sl.Shortcuts) == 0 {
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
