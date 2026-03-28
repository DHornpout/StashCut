package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/stashcut/cli/model"
)

type Sidebar struct {
	Apps     []model.App
	Selected int
	Focused  bool
	height   int
	offset   int // scroll offset
}

func NewSidebar(apps []model.App) Sidebar {
	return Sidebar{
		Apps:     apps,
		Selected: 0,
	}
}

func (s *Sidebar) SetHeight(h int) {
	s.height = h
}

func (s *Sidebar) MoveUp() {
	if s.Selected > 0 {
		s.Selected--
		s.clampOffset()
	}
}

func (s *Sidebar) MoveDown() {
	if s.Selected < len(s.Apps)-1 {
		s.Selected++
		s.clampOffset()
	}
}

func (s *Sidebar) SelectedApp() *model.App {
	if len(s.Apps) == 0 || s.Selected >= len(s.Apps) {
		return nil
	}
	return &s.Apps[s.Selected]
}

func (s *Sidebar) clampOffset() {
	visible := s.visibleRows()
	if s.Selected < s.offset {
		s.offset = s.Selected
	} else if s.Selected >= s.offset+visible {
		s.offset = s.Selected - visible + 1
	}
}

func (s *Sidebar) visibleRows() int {
	// height minus title (1) + margin (1) + padding (2 top/bottom) + border (2)
	rows := s.height - 5
	if rows < 1 {
		rows = 1
	}
	if rows > 20 {
		rows = 20
	}
	return rows
}

func (s Sidebar) View() string {
	style := StylePanel
	if s.Focused {
		style = StylePanelFocused
	}

	title := StyleTitle.Render("Apps")
	var rows []string
	rows = append(rows, title)

	visible := s.visibleRows()
	end := s.offset + visible
	if end > len(s.Apps) {
		end = len(s.Apps)
	}

	for i := s.offset; i < end; i++ {
		app := s.Apps[i]
		icon := app.Icon
		if icon == "" {
			icon = "  "
		}
		label := fmt.Sprintf("%s %s", icon, app.Name)
		// truncate to fit sidebar width
		maxLen := SidebarWidth - 6
		if len(label) > maxLen {
			label = label[:maxLen-1] + "…"
		}
		// pad to fixed width
		label = label + strings.Repeat(" ", maxLen-len([]rune(label)))

		if i == s.Selected && s.Focused {
			rows = append(rows, StyleSelected.Render(label))
		} else if i == s.Selected {
			rows = append(rows, lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Render(label))
		} else {
			rows = append(rows, StyleNormal.Render(label))
		}
	}

	if len(s.Apps) == 0 {
		rows = append(rows, StyleMuted.Render("No apps yet. Press n to add."))
	}

	// Content height: 1 title row + actual app rows shown (capped at 20)
	shown := len(s.Apps)
	if shown == 0 {
		shown = 1 // "no apps" message
	}
	if shown > 20 {
		shown = 20
	}
	contentHeight := 1 + shown // title + rows
	if maxH := s.height - 2; contentHeight > maxH {
		contentHeight = maxH
	}

	content := strings.Join(rows, "\n")
	return style.Width(SidebarWidth).Height(contentHeight).Render(content)
}
