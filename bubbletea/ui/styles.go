package ui

import "github.com/charmbracelet/lipgloss"

// ─── Brand Colors ─────────────────────────────────────────────────────────────

const (
	ColorPrimary = "#7C3AED" // purple  — brand, titles, active cursor
	ColorSuccess = "#10B981" // green   — win, confirmed fields
	ColorDanger  = "#EF4444" // red     — lose, errors
	ColorWarning = "#F59E0B" // amber   — tie, optional hints
	ColorInfo    = "#3B82F6" // blue    — keybind hints, info text
	ColorMuted   = "#6B7280" // grey    — inactive fields, borders
	ColorText    = "#F9FAFB" // white   — primary readable text
	ColorDim     = "#9CA3AF" // light grey — secondary text
)

// ─── Text Styles ──────────────────────────────────────────────────────────────

var (
	// TitleStyle — used for screen headings like "REGISTER", "SIGN IN"
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	// BannerStyle — used for the ROPA-SCI ASCII logo
	BannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary))

	// SelectedStyle — menu cursor and active form field
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	// MutedStyle — inactive menu options and future form fields
	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted))

	// SuccessStyle — win message, confirmed field checkmark
	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorSuccess)).
			Bold(true)

	// DangerStyle — lose message, error text
	DangerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDanger)).
			Bold(true)

	// WarningStyle — tie message, optional field hints
	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorWarning)).
			Bold(true)

	// InfoStyle — keybind footer hints
	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorInfo))

	// DimStyle — secondary text, spinner text
	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorDim))
)

// ─── Box Styles ───────────────────────────────────────────────────────────────

var (
	// WinBoxStyle — result box when player wins a round
	WinBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorSuccess)).
			Padding(0, 2).
			Width(35)

	// LoseBoxStyle — result box when AI wins a round
	LoseBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorDanger)).
			Padding(0, 2).
			Width(35)

	// TieBoxStyle — result box on a draw
	TieBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorWarning)).
			Padding(0, 2).
			Width(35)

	// RoomCodeBoxStyle — the room code display box
	RoomCodeBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(1, 3).
				Width(35)

	// WaitingBoxStyle — the quick match waiting box
	WaitingBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorInfo)).
			Padding(1, 3).
			Width(35)

	// SelectedCardStyle — game move card when selected
	SelectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color(ColorPrimary)).
				Padding(0, 1).
				Width(13).
				Align(lipgloss.Center)

	// NormalCardStyle — game move card when not selected
	NormalCardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(ColorMuted)).
			Padding(0, 1).
			Width(13).
			Align(lipgloss.Center)
)

// ─── Footer ───────────────────────────────────────────────────────────────────

// Footer renders a consistent keybind hint line at the bottom of any screen
func Footer(hints string) string {
	return InfoStyle.Render("  " + hints)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// Checkmark returns a styled green checkmark for completed form fields
func Checkmark() string {
	return SuccessStyle.Render("✓")
}

// ErrorLine returns a styled red error message
func ErrorLine(msg string) string {
	return DangerStyle.Render("  ⚠  " + msg)
}

// ScorePip returns a filled or empty pip for the score bar
func ScorePip(filled bool) string {
	if filled {
		return SuccessStyle.Render("█")
	}
	return MutedStyle.Render("░")
}
