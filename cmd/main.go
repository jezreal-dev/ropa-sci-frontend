package main

import (
	"fmt"
	"os"

	"ropa-sci-frontend/models"

	tea "github.com/charmbracelet/bubbletea"
)

// This holds the app's current state
type model struct {
	state models.GameState
}

// This creates the starting state of our app now "WELCOME"
func initialModel() model {
	return model{
		state: models.GameState{
			Screen: "menu", // ← changed
			Score:  models.MatchScore{Round: 1},
			Cursor: 0,
		},
	}
}

// INIT — runs once at startup
func (m model) Init() tea.Cmd {
	return nil
}

// UPDATE — handles events like key presses
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "q":
			// Only quit from screens where user isn't typing
			if m.state.Screen != "login" && m.state.Screen != "register" {
				return m, tea.Quit
			}
			// Otherwise fall through to default and capture it as typed character
			if len(m.state.InputBuffer) < 20 {
				m.state.InputBuffer += "q"
			}

		case "esc":
			// Go back to welcome from any sub-screen
			if m.state.Screen == "login" || m.state.Screen == "register" {
				m.state.Screen = "welcome"
				m.state.InputBuffer = ""
				m.state.FormError = ""
				m.state.Cursor = 0
			}

		case "up", "k":
			if isMenuScreen(m.state.Screen) && m.state.Cursor > 0 {
				m.state.Cursor--
			}

		case "down", "j":
			if isMenuScreen(m.state.Screen) && m.state.Cursor < menuLength(m.state.Screen)-1 {
				m.state.Cursor++
			}
		case "enter":
			switch m.state.Screen {

			case "welcome":
				switch m.state.Cursor {
				case 0:
					// Go to registration
					m.state.Screen = "register"
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.ActiveField = 0
				case 1:
					// Go to login
					m.state.Screen = "login"
					m.state.InputBuffer = ""
					m.state.FormError = ""
				case 2:
					return m, tea.Quit
				}

			case "login":
				// Look up username in players.json
				player, found, err := models.FindPlayerByUsername(m.state.InputBuffer)
				if err != nil {
					m.state.FormError = "Error reading player data — please try again"
				} else if !found {
					m.state.FormError = "Username not found — check spelling or register"
				} else {
					// Found — load their data and go to menu
					m.state.Player = player
					m.state.Screen = "menu"
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.Cursor = 0
				}

			case "menu":
				switch m.state.Cursor {
				case 0:
					m.state.Screen = "game"
					m.state.GameMode = "single"
				case 1:
					m.state.Screen = "waiting"
					m.state.GameMode = "multi"
				case 2:
					return m, tea.Quit
				}
			}

		case "backspace":
			// Allow user to delete characters while typing
			if len(m.state.InputBuffer) > 0 {
				m.state.InputBuffer = m.state.InputBuffer[:len(m.state.InputBuffer)-1]
				m.state.FormError = "" // clear error when they start correcting
			}

		default:
			// Capture typed characters for login input
			if m.state.Screen == "login" {
				if len(msg.String()) == 1 {
					m.state.InputBuffer += msg.String()
				}
			}
		}
	}
	return m, nil
}

// isMenuScreen returns true for screens that use cursor navigation
func isMenuScreen(screen string) bool {
	return screen == "welcome" || screen == "menu"
}

// menuLength returns how many options a given menu screen has
func menuLength(screen string) int {
	switch screen {
	case "welcome":
		return 3 // Register, Sign In, Quit
	case "menu":
		return 3 // Single Player, Multiplayer, Quit
	default:
		return 0
	}
}

// VIEW — draws the screen as a string
func (m model) View() string {
	switch m.state.Screen {
	case "welcome":
		return renderWelcome(m)

	case "register":
		return renderRegister(m)

	case "login":
		return renderLogin(m)

	case "menu":
		return renderMenu(m)

	case "game":
		return "Game screen — coming in Week 3!\n\nPress (q) to quit"

	case "waiting":
		return "Waiting for opponent...\n\nPress (q) to quit"

	default:
		return "Unknown screen\n\nPress (q) to quit"
	}
}

func renderWelcome(m model) string {
	options := []string{
		"Register — I am new",
		"Sign In  — I have an account",
		"Quit",
	}
	s := `
  ██████╗  ██████╗ ██████╗  █████╗       ███████╗ ██████╗██╗
  ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗      ██╔════╝██╔════╝██║
  ██████╔╝██║   ██║██████╔╝███████║█████╗███████╗██║     ██║
  ██╔══██╗██║   ██║██╔═══╝ ██╔══██║╚════╝╚════██║██║     ██║
  ██║  ██║╚██████╔╝██║     ██║  ██║      ███████║╚██████╗██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝      ╚══════╝ ╚═════╝╚═╝

`
	for i, opt := range options {
		cursor := "  "
		if m.state.Cursor == i {
			cursor = "> "
		}
		s += "  " + cursor + opt + "\n"
	}
	s += "\n  ↑/↓ to move · Enter to select"
	if m.state.FormError != "" {
		s += "\n\n  ⚠  " + m.state.FormError
	}
	return s
}

func renderLogin(m model) string {
	s := "\n  SIGN IN\n\n"
	s += "  Enter your Gitea username: " + m.state.InputBuffer + "_\n"
	if m.state.FormError != "" {
		s += "\n  ⚠  " + m.state.FormError + "\n"
	}
	s += "\n  Enter to confirm · Esc to go back · ctrl+c to quit"
	return s
}

// renderRegister draws the registration screen
func renderRegister(m model) string {
	return `
  ██████╗  ██████╗ ██████╗  █████╗       ███████╗ ██████╗██╗
  ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗      ██╔════╝██╔════╝██║
  ██████╔╝██║   ██║██████╔╝███████║█████╗███████╗██║     ██║
  ██╔══██╗██║   ██║██╔═══╝ ██╔══██║╚════╝╚════██║██║     ██║
  ██║  ██║╚██████╔╝██║     ██║  ██║      ███████║╚██████╗██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝      ╚══════╝ ╚═════╝╚═╝

  Welcome! Please register to continue.

  [Registration form coming next step]

  Press (ctrl+c) to quit
`
}

// renderMenu draws the main menu with a moving cursor
func renderMenu(m model) string {
	// The > cursor shows which option is selected
	options := []string{
		"Single Player",
		"Multiplayer",
		"Quit",
	}

	menu := "\n  ROPA-SCI\n\n"
	for i, option := range options {
		cursor := "  " // no cursor
		if m.state.Cursor == i {
			cursor = "> " // cursor here
		}
		menu += "  " + cursor + option + "\n"
	}
	menu += "\n  ↑/↓ or k/j to move · Enter to select"
	return menu
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
