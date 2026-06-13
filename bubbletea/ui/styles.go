package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── Brand Colors ─────────────────────────────────────────────────────────────

const (
	ColorPrimary  = "#A78BFA" // Cyber Lavender — brand, titles, active cursor
	ColorAccent   = "#818CF8" // Neon Electric Indigo — secondary accents, glow
	ColorSuccess  = "#34D399" // Emerald Jade — win, confirmed fields
	ColorDanger   = "#FB7185" // Sunset Rose — lose, errors
	ColorWarning  = "#FBBF24" // Cyber Amber — tie, optional hints
	ColorInfo     = "#60A5FA" // Sky Blue — keybind hints, info text
	ColorMuted    = "#475569" // Dark Slate — inactive fields, borders
	ColorText     = "#F1F5F9" // Pearl White — primary readable text
	ColorDim      = "#94A3B8" // Slate Grey — secondary text
	ColorBg       = "#0F172A" // Midnight Navy — deep background
	ColorBgPanel  = "#1E293B" // Deep Charcoal — card/panel background
	ColorBgAccent = "#1E1B4B" // Deep Indigo — highlighted panel bg
	ColorGlow     = "#C4B5FD" // Soft Lilac — glow effects, subtle highlights
)

// ─── Text Styles ──────────────────────────────────────────────────────────────

var (
	// TitleStyle — used for screen headings like "REGISTER", "SIGN IN"
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true).
			MarginBottom(1)

	// BannerStyle — used for the ROPA-SCI ASCII logo
	BannerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPrimary)).
			Bold(true)

	// SelectedStyle — menu cursor and active form field
	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorAccent)).
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

// ─── Layout Styles ────────────────────────────────────────────────────────────

var (
	// AppContainerStyle — the master frame wrapping all screen content
	AppContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorMuted)).
				Padding(1, 3).
				Margin(0).
				Width(62).
				Align(lipgloss.Center)

	// FormPanelStyle — housing for login/register forms
	FormPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorAccent)).
			Padding(1, 2).
			Width(54).
			Align(lipgloss.Left)

	// ScoreHeaderStyle — the persistent score banner during gameplay
	ScoreHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorText)).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color(ColorMuted)).
				PaddingBottom(1).
				Width(54).
				Align(lipgloss.Center)

	// HeaderStyle — centered section headers within panels
	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGlow)).
			Bold(true).
			Align(lipgloss.Center).
			Width(54)

	// DividerStyle — horizontal separator
	DividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorMuted)).
			Align(lipgloss.Center).
			Width(54)
)

// ─── Box Styles ───────────────────────────────────────────────────────────────

var (
	// WinBoxStyle — result box when player wins a round
	WinBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorSuccess)).
			Padding(1, 3).
			Width(50).
			Background(lipgloss.Color(ColorBgPanel)).
			Align(lipgloss.Center)

	// LoseBoxStyle — result box when AI wins a round
	LoseBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorDanger)).
			Padding(1, 3).
			Width(50).
			Background(lipgloss.Color(ColorBgPanel)).
			Align(lipgloss.Center)

	// TieBoxStyle — result box on a draw
	TieBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color(ColorWarning)).
			Padding(1, 3).
			Width(50).
			Background(lipgloss.Color(ColorBgPanel)).
			Align(lipgloss.Center)

	// RoomCodeBoxStyle — the room code display box
	RoomCodeBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorAccent)).
				Padding(1, 3).
				Width(50).
				Background(lipgloss.Color(ColorBgAccent)).
				Align(lipgloss.Center)

	// WaitingBoxStyle — the quick match waiting box
	WaitingBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorInfo)).
			Padding(1, 3).
			Width(50).
			Align(lipgloss.Center)
)

// ─── Card Styles ──────────────────────────────────────────────────────────────

var (
	// SelectedCardStyle — game move card when selected
	SelectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color(ColorAccent)).
				Foreground(lipgloss.Color(ColorText)).
				Background(lipgloss.Color(ColorBgAccent)).
				Padding(1, 2).
				Width(16).
				Align(lipgloss.Center).
				Bold(true)

	// NormalCardStyle — game move card when not selected
	NormalCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorMuted)).
			Foreground(lipgloss.Color(ColorDim)).
			Padding(1, 2).
			Width(16).
			Align(lipgloss.Center)
)

// ─── Footer ───────────────────────────────────────────────────────────────────

// Footer renders a consistent keybind hint line at the bottom of any screen
func Footer(hints string) string {
	line := DimStyle.Render(strings.Repeat("─", 54))
	return line + "\n" + InfoStyle.Render("  "+hints)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// Checkmark returns a styled green checkmark for completed form fields
func Checkmark() string {
	return SuccessStyle.Render("✓")
}

// ErrorLine returns a styled red error message
func ErrorLine(msg string) string {
	return DangerStyle.Render(" ✖  " + msg)
}

// ScorePip returns a filled or empty pip for the score bar
func ScorePip(filled bool) string {
	if filled {
		return SuccessStyle.Render("●")
	}
	return MutedStyle.Render("○")
}

// Divider returns a styled horizontal rule
func Divider() string {
	return DividerStyle.Render(strings.Repeat("─", 48))
}
