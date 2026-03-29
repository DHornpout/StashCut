package ui

import "github.com/charmbracelet/lipgloss"

const (
	SidebarWidth = 28
)

var (
	colorPrimary  = lipgloss.Color("#7C3AED") // purple
	colorAccent   = lipgloss.Color("#A78BFA")
	colorMuted    = lipgloss.Color("#6B7280")
	colorSelected = lipgloss.Color("#1E1B4B")
	colorBorder   = lipgloss.Color("#374151")
	colorFav      = lipgloss.Color("#F59E0B") // amber for star
	colorError    = lipgloss.Color("#EF4444")
	colorSuccess  = lipgloss.Color("#10B981")

	StylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	StylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			MarginBottom(1)

	StyleSelected = lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	StyleNormal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	StyleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	StyleFavorite = lipgloss.NewStyle().
			Foreground(colorFav)

	StyleKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)

	StyleHelp = lipgloss.NewStyle().
			Foreground(colorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	// colorRowAlt is the background tint for alternating (odd) shortcut rows.
	colorRowAlt = lipgloss.Color("#1A1F2E")

	StyleGroupSectionHeader = lipgloss.NewStyle().
					Background(lipgloss.Color("#2D1B69")).
					Foreground(lipgloss.Color("#C4B5FD")).
					Bold(true).
					Padding(0, 1)

	StyleRowAlt = lipgloss.NewStyle().
			Background(colorRowAlt)
)
