package ui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FirstRunResult struct {
	Path   string
	Create bool // true = create new, false = open existing
}

type firstRunState int

const (
	frStateChoose firstRunState = iota
	frStateSpecifyPath
)

type FirstRunModel struct {
	defaultPath string
	state       firstRunState
	pathInput   textinput.Model
	Result      *FirstRunResult
	err         string
}

func NewFirstRunModel(defaultPath string) FirstRunModel {
	ti := textinput.New()
	ti.Placeholder = defaultPath
	ti.CharLimit = 256
	ti.Width = 60

	return FirstRunModel{
		defaultPath: defaultPath,
		state:       frStateChoose,
		pathInput:   ti,
	}
}

func (m FirstRunModel) Init() tea.Cmd {
	return nil
}

func (m FirstRunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case frStateChoose:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "c", "C":
				m.Result = &FirstRunResult{Path: m.defaultPath, Create: true}
				return m, tea.Quit
			case "o", "O":
				m.state = frStateSpecifyPath
				m.pathInput.Focus()
				return m, nil
			}

		case frStateSpecifyPath:
			switch msg.Type {
			case tea.KeyEsc:
				m.state = frStateChoose
				m.pathInput.Blur()
				return m, nil
			case tea.KeyEnter:
				path := strings.TrimSpace(m.pathInput.Value())
				if path == "" {
					path = m.defaultPath
				}
				// Expand ~ manually
				if strings.HasPrefix(path, "~/") {
					home, _ := os.UserHomeDir()
					path = home + path[1:]
				}
				m.Result = &FirstRunResult{Path: path, Create: false}
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m FirstRunModel) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A78BFA")).
		Render("Welcome to StashCut CLI")

	var body string
	switch m.state {
	case frStateChoose:
		body = strings.Join([]string{
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render("No shortcuts file was found."),
			"",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("[C]") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render("  Create new file at default path"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("     " + m.defaultPath),
			"",
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("[O]") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render("  Open / specify a different path"),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("Press C or O — q to quit"),
		}, "\n")

	case frStateSpecifyPath:
		body = strings.Join([]string{
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render("Enter path to shortcuts.json:"),
			"",
			m.pathInput.View(),
			"",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("enter to confirm  •  esc to go back"),
		}, "\n")
	}

	content := title + "\n\n" + body
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Padding(2, 4).
		Margin(2, 4).
		Render(content)
}
