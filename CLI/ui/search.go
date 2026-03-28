package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stashcut/cli/model"
)

type Search struct {
	input  textinput.Model
	Active bool
	width  int
}

func NewSearch() Search {
	ti := textinput.New()
	ti.Placeholder = "Search shortcuts…"
	ti.CharLimit = 100
	return Search{input: ti}
}

func (s *Search) Activate() {
	s.Active = true
	s.input.Focus()
	s.input.SetValue("")
}

func (s *Search) Deactivate() {
	s.Active = false
	s.input.Blur()
}

func (s Search) Query() string {
	return s.input.Value()
}

func (s Search) Update(msg tea.Msg) (Search, tea.Cmd) {
	if !s.Active {
		return s, nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return s, cmd
}

func (s *Search) SetWidth(w int) {
	s.width = w
	// Single-line bar: "/ " prompt (2 chars) + 1 padding each side = 4 overhead
	inputWidth := w - 4
	if inputWidth < 10 {
		inputWidth = 10
	}
	s.input.Width = inputWidth
}

func (s Search) View() string {
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1E1B4B")).
		Foreground(lipgloss.Color("#E5E7EB")).
		Padding(0, 1).
		Width(s.width - 2) // -2 so total (Width + padding 2) = s.width

	if !s.Active {
		return barStyle.Render(StyleMuted.Render("  / to search"))
	}

	// Set input width in the copy for rendering
	s.input.Width = s.width - 4
	prompt := StyleKey.Render("/ ")
	return barStyle.Render(prompt + s.input.View())
}

// Filter returns shortcuts matching the query across description, keys, and tags.
func Filter(shortcuts []model.Shortcut, query string) []model.Shortcut {
	if query == "" {
		return shortcuts
	}
	q := strings.ToLower(query)
	var result []model.Shortcut
	for _, s := range shortcuts {
		if strings.Contains(strings.ToLower(s.Description), q) {
			result = append(result, s)
			continue
		}
		for _, t := range s.Tags {
			if strings.Contains(strings.ToLower(t), q) {
				result = append(result, s)
				break
			}
		}
	}
	return result
}
