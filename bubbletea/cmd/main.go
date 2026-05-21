package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"ropa-sci-frontend/bubbletea/models"
	"ropa-sci-frontend/bubbletea/ui"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── Spinner & AI message types ───────────────────────────────────────────────

// spinnerTickMsg fires every 100ms to advance the spinner animation
type spinnerTickMsg struct{}

// aiDecidedMsg fires after 1.5s carrying the AI's chosen move
type aiDecidedMsg struct {
	move models.Move
}

// showResultMsg fires after the reveal pause to transition to result phase
type showResultMsg struct{}

// spinnerFrames is the Braille dot animation cycle used during AI think phase
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ─── Tea Commands ─────────────────────────────────────────────────────────────

// spinnerTick returns a command that fires a spinnerTickMsg every 100ms
func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// aiThink returns a command that fires after 1.5s with a randomly chosen AI move
func aiThink() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		moves := []models.Move{models.Rock, models.Paper, models.Scissors}
		return aiDecidedMsg{move: moves[time.Now().UnixNano()%3]}
	})
}

// ─── Game Logic Helpers ───────────────────────────────────────────────────────

// calculateOutcome returns "win", "lose", or "tie" from the player's perspective
func calculateOutcome(player, ai models.Move) string {
	if player == ai {
		return "tie"
	}
	if (player == models.Rock && ai == models.Scissors) ||
		(player == models.Paper && ai == models.Rock) ||
		(player == models.Scissors && ai == models.Paper) {
		return "win"
	}
	return "lose"
}

// outcomeMessage returns flavour text describing the result of a move combination
func outcomeMessage(player, ai models.Move) string {
	if player == ai {
		return "Great minds think alike"
	}
	messages := map[string]string{
		"rock-scissors":  "Rock crushes Scissors!",
		"scissors-paper": "Scissors cuts Paper!",
		"paper-rock":     "Paper covers Rock!",
		"scissors-rock":  "Rock crushes your Scissors...",
		"rock-paper":     "Paper covers your Rock...",
		"paper-scissors": "Scissors cuts your Paper...",
	}
	return messages[string(player)+"-"+string(ai)]
}

// indexToMove converts a cursor position (0-2) to the corresponding Move
func indexToMove(cursor int) models.Move {
	switch cursor {
	case 0:
		return models.Rock
	case 1:
		return models.Paper
	case 2:
		return models.Scissors
	default:
		return models.None
	}
}

// ─── Model ────────────────────────────────────────────────────────────────────

// model wraps GameState as the single source of truth for the entire TUI
type model struct {
	state models.GameState
}

// initialModel returns the app's starting state — always begins on the welcome screen
func initialModel() model {
	return model{
		state: models.GameState{
			Screen: "welcome",
			Score:  models.MatchScore{Round: 1},
			Cursor: 0,
			Phase:  "pick",
		},
	}
}

// ─── Init ─────────────────────────────────────────────────────────────────────

// Init runs once at startup — no initial commands needed
func (m model) Init() tea.Cmd {
	return nil
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update is the central event handler — all state changes flow through here
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Bubbletea sends a tea.WindowSizeMsg automatically whenever the terminal is resized
	case tea.WindowSizeMsg:
		m.state.TermWidth = msg.Width
		m.state.TermHeight = msg.Height
		return m, nil

	// ── Spinner tick — advances the animation frame during AI think phase ──
	case spinnerTickMsg:
		m.state.SpinnerFrame = (m.state.SpinnerFrame + 1) % len(spinnerFrames)
		if m.state.Phase == "think" ||
			m.state.Screen == "create-room" ||
			m.state.Screen == "quick-match" {
			return m, spinnerTick()
		}
		return m, nil

	// ── AI decided — transition from think to reveal, then schedule result ──
	case aiDecidedMsg:
		m.state.AIMove = msg.move
		m.state.Phase = "reveal"
		m.state.RoundOutcome = calculateOutcome(m.state.PlayerMove, msg.move)
		// Brief pause so the player can see both cards before the verdict
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return showResultMsg{}
		})

	// ── Show result — update score and transition to result phase ──
	case showResultMsg:
    m.state.Phase = "result"
    switch m.state.RoundOutcome {
    case "win":
        m.state.Score.PlayerWins++
    case "lose":
        m.state.Score.OpponentWins++
    }
    m.state.Score.Round++

    // Check if match is over and update lifetime stats
    if m.state.Score.PlayerWins == 2 || m.state.Score.OpponentWins == 2 {
        m.state.Player.TotalMatches++
        if m.state.Score.PlayerWins == 2 {
            m.state.Player.Wins++
        } else {
            m.state.Player.Losses++
        }
        // Save updated stats to disk — ignore error silently in UI
        // player will still see correct stats this session
        _ = models.UpdatePlayer(m.state.Player)
    }
    return m, nil

	// ── Mouse clicks — coordinate mapping wired in Week 7 polish phase ──
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			_ = msg.X
			_ = msg.Y
		}

	// ── Keyboard input ──────────────────────────────────────────────────────
	case tea.KeyMsg:
		switch msg.String() {

		// Hard quit — works everywhere, no exceptions
		case "ctrl+c":
			return m, tea.Quit

		// Soft quit — blocked on form screens to allow typing the letter q
		case "q":
			if m.state.Screen != "login" && m.state.Screen != "register" {
				return m, tea.Quit
			}
			if len(m.state.InputBuffer) < 20 {
				m.state.InputBuffer += "q"
			}

		// Escape — context-aware back navigation
		case "esc":
			switch m.state.Screen {
			case "register":
				// Clear all form data on exit so the next visit starts fresh
				m.state.Player = models.Player{}
				m.state.ActiveField = 0
				m.state.StateSuggestions = nil
				m.state.Screen = m.state.PreviousScreen
				m.state.InputBuffer = ""
				m.state.FormError = ""
				m.state.Cursor = 0
			case "login":
				m.state.Screen = m.state.PreviousScreen
				m.state.InputBuffer = ""
				m.state.FormError = ""
				m.state.Cursor = 0
			case "game", "waiting":
				m.state.Screen = "menu"
				m.state.Phase = "pick"
				m.state.Cursor = 0
				m.state.FormError = ""

			case "create-room", "quick-match":
				m.state.Screen = "multi-menu"
				m.state.Cursor = 0
				m.state.RoomCode = ""

			case "multi-menu":
				m.state.Screen = "menu"
				m.state.Cursor = 1
			case "result":
				// Return to menu and reset match score for the next game
				m.state.Screen = "menu"
				m.state.Cursor = 0
				m.state.Score = models.MatchScore{Round: 1}
				m.state.Phase = "pick"
			}

		// Menu cursor — up arrow and vim-style k
		case "up", "k":
			if isMenuScreen(m.state.Screen) && m.state.Cursor > 0 {
				m.state.Cursor--
			}

		// Menu cursor — down arrow and vim-style j
		case "down", "j":
			if isMenuScreen(m.state.Screen) && m.state.Cursor < menuLength(m.state.Screen)-1 {
				m.state.Cursor++
			}

		// Game card cursor — left arrow and vim-style h
		case "left", "h":
			if m.state.Screen == "game" && m.state.Phase == "pick" && m.state.Cursor > 0 {
				m.state.Cursor--
			}

		// Game card cursor — right arrow and vim-style l
		case "right", "l":
			if m.state.Screen == "game" && m.state.Phase == "pick" && m.state.Cursor < 2 {
				m.state.Cursor++
			}

		// Direct move shortcuts — Rock
		case "1", "r":
			if m.state.Screen == "game" && m.state.Phase == "pick" {
				m.state.PlayerMove = models.Rock
				m.state.Phase = "think"
				return m, tea.Batch(spinnerTick(), aiThink())
			}

		// Direct move shortcuts — Paper
		case "2", "p":
			if m.state.Screen == "game" && m.state.Phase == "pick" {
				m.state.PlayerMove = models.Paper
				m.state.Phase = "think"
				return m, tea.Batch(spinnerTick(), aiThink())
			}

		// Direct move shortcuts — Scissors
		case "3", "s":
			if m.state.Screen == "game" && m.state.Phase == "pick" {
				m.state.PlayerMove = models.Scissors
				m.state.Phase = "think"
				return m, tea.Batch(spinnerTick(), aiThink())
			}

		// Enter — confirms selections and form fields
		case "enter":
			switch m.state.Screen {

			// ── Welcome screen — navigate to register, login, or quit ──
			case "welcome":
				switch m.state.Cursor {
				case 0:
					m.state.PreviousScreen = "welcome"
					m.state.Screen = "register"
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.ActiveField = 0
				case 1:
					m.state.PreviousScreen = "welcome"
					m.state.Screen = "login"
					m.state.InputBuffer = ""
					m.state.FormError = ""
				case 2:
					return m, tea.Quit
				}

			// ── Registration form — validate and advance field by field ──
			case "register":
				input := strings.TrimSpace(m.state.InputBuffer)
				caser := cases.Title(language.English)

				switch m.state.ActiveField {

				case 0: // First Name
					if len(strings.Fields(input)) != 1 {
						m.state.FormError = "First name must be a single word"
						return m, nil
					}
					m.state.Player.FirstName = caser.String(strings.ToLower(input)) // normalise then capitalise
					m.state.ActiveField++
					m.state.InputBuffer = ""
					m.state.FormError = ""

				case 1: // Last Name
					if len(strings.Fields(input)) != 1 {
						m.state.FormError = "Last name must be a single word"
						return m, nil
					}
					m.state.Player.LastName = caser.String(strings.ToLower(input)) // normalise then capitalise
					m.state.ActiveField++
					m.state.InputBuffer = ""
					m.state.FormError = ""

				case 2: // Username
					if len(strings.Fields(input)) != 1 {
						m.state.FormError = "Username must be a single word — no spaces"
						return m, nil
					}
					if models.UsernameExists(strings.ToLower(input)) {
						m.state.FormError = "Username already taken — try signing in instead"
						return m, nil
					}
					m.state.Player.Username = strings.ToLower(input) // always stored lowercase
					m.state.ActiveField++
					m.state.InputBuffer = ""
					m.state.FormError = ""

				case 3: // State — FindState handles any casing
					state, found := models.FindState(input)
					if !found {
						m.state.FormError = "State not recognised — try 'Lagos' or 'LA'"
						return m, nil
					}
					m.state.Player.State = state.Name
					m.state.StateSuggestions = nil
					m.state.ActiveField++
					m.state.InputBuffer = ""
					m.state.FormError = ""

				case 4: // Email
					if input != "" {
						atIndex := strings.Index(input, "@")
						if atIndex < 1 || !strings.Contains(input[atIndex:], ".") {
							m.state.FormError = "Invalid email — or just press Enter to skip"
							return m, nil
						}
						emailCopy := strings.ToLower(input) // normalise on save
						m.state.Player.Email = &emailCopy
					}
					m.state.Player.Role = "player"
					err := models.SavePlayer(m.state.Player)
					if err != nil {
						m.state.FormError = "Could not save: " + err.Error()
						return m, nil
					}
					m.state.Screen = "menu"
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.Cursor = 0
				}

			// ── Login — look up username and load player data ──
			case "login":
				player, found, err := models.FindPlayerByUsername(strings.ToLower(m.state.InputBuffer))
				if err != nil {
					m.state.FormError = "Error reading player data — please try again"
				} else if !found {
					m.state.FormError = "Username not found — check spelling or register"
				} else {
					m.state.Player = player
					m.state.Screen = "menu"
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.Cursor = 0
				}

			// ── Main menu — route to game mode or quit ──
			case "menu":
				switch m.state.Cursor {
				case 0:
					m.state.PreviousScreen = "menu"
					m.state.Screen = "game"
					m.state.GameMode = "single"
					m.state.Phase = "pick"
					m.state.Score = models.MatchScore{Round: 1}
				case 1:
					m.state.PreviousScreen = "menu"
					m.state.Screen = "multi-menu"
					m.state.GameMode = "multi"
					m.state.Cursor = 0
				case 2:
					return m, tea.Quit
				}

			case "multi-menu":
				switch m.state.Cursor {
				case 0:
					// Create Room — generate code and start spinner
					m.state.Screen = "create-room"
					m.state.RoomCode = models.GenerateRoomCode()
					m.state.PreviousScreen = "multi-menu"
					m.state.Cursor = 0
					return m, spinnerTick()
				case 1:
					// Quick Match — auto search and start spinner
					m.state.Screen = "quick-match"
					m.state.PreviousScreen = "multi-menu"
					m.state.Cursor = 0
					return m, spinnerTick()
				case 2:
					// Back to main menu
					m.state.Screen = "menu"
					m.state.Cursor = 1
				}

			// ── Game screen — confirm move or continue after result ──
			case "game":
				switch m.state.Phase {
				case "pick":
					// Confirm the card currently under the cursor
					m.state.PlayerMove = indexToMove(m.state.Cursor)
					m.state.Phase = "think"
					return m, tea.Batch(spinnerTick(), aiThink())

				case "result":
					if m.state.Score.PlayerWins == 2 || m.state.Score.OpponentWins == 2 {
						// Match is over — full reset for a new match
						m.state.Score = models.MatchScore{Round: 1}
					}
					// Reset round state, keep player data and lifetime stats
					m.state.Phase = "pick"
					m.state.PlayerMove = models.None
					m.state.AIMove = models.None
					m.state.RoundOutcome = ""
					m.state.Cursor = 0
				}
			}

		// Backspace — deletes last character from input buffer on form screens
		case "backspace":
			if len(m.state.InputBuffer) > 0 {
				m.state.InputBuffer = m.state.InputBuffer[:len(m.state.InputBuffer)-1]
				m.state.FormError = ""
				// Refresh state suggestions as the user edits
				if m.state.Screen == "register" && m.state.ActiveField == 3 {
					m.state.StateSuggestions = models.SuggestStates(m.state.InputBuffer)
				}
			}

		// Default — capture printable characters on form screens
		default:
			if len(msg.String()) == 1 {
				char := msg.String()

				if m.state.Screen == "login" {
					m.state.InputBuffer += char
				}

				if m.state.Screen == "register" {
					m.state.InputBuffer += char
					if m.state.ActiveField == 3 {
						m.state.StateSuggestions = models.SuggestStates(m.state.InputBuffer)
					}
				}
			}
		}
	}
	return m, nil
}

// ─── Navigation Helpers ───────────────────────────────────────────────────────

// isMenuScreen returns true for any screen that uses vertical cursor navigation
func isMenuScreen(screen string) bool {
	return screen == "welcome" || screen == "menu" || screen == "multi-menu"
}

// menuLength returns the number of selectable options for a given menu screen
func menuLength(screen string) int {
	switch screen {
	case "welcome":
		return 3 // Register, Sign In, Quit
	case "menu":
		return 3 // Single Player
	case "multi-menu":	// Multiplayer, Quit
		return 3
	default:
		return 0
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

// View renders the current screen as a string — called automatically on every state change
func (m model) View() string {
	// Global guard — terminal too narrow to render safely
	if m.state.TermWidth > 0 && m.state.TermWidth < 50 {
		return fmt.Sprintf(
			"\n  ⚠  Terminal too narrow!\n\n"+
				"  Please resize to at least 50 columns.\n"+
				"  Current width: %d columns.",
			m.state.TermWidth,
		)
	}
	// Too short vertically
	if m.state.TermHeight > 0 && m.state.TermHeight < 16 {
		return "\n  ⚠  Terminal too short!\n\n" +
			"  Please resize to at least 16 rows."
	}

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
		return renderGame(m)
	case "multi-menu":
		return renderMultiMenu(m)
	case "create-room":
		return renderCreateRoom(m)
	case "quick-match":
		return renderQuickMatch(m)
	case "waiting":
		return renderQuickMatch(m) // fallback — reuses quick match screen
	default:
		return "\n  Unknown screen\n\n  ctrl+c to quit"
	}
}

// ─── Screen Renderers ─────────────────────────────────────────────────────────

// banner returns the full ASCII logo on wide terminals
// and a compact text version on narrow terminals
func banner(termWidth int) string {
	if termWidth >= 70 {
		return `
  ██████╗  ██████╗ ██████╗  █████╗       ███████╗ ██████╗██╗
  ██╔══██╗██╔═══██╗██╔══██╗██╔══██╗      ██╔════╝██╔════╝██║
  ██████╔╝██║   ██║██████╔╝███████║█████╗███████╗██║     ██║
  ██╔══██╗██║   ██║██╔═══╝ ██╔══██║╚════╝╚════██║██║     ██║
  ██║  ██║╚██████╔╝██║     ██║  ██║      ███████║╚██████╗██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝      ╚══════╝ ╚═════╝╚═╝
`
	}
	return "\n  ROPA-SCI\n"
}

// renderWelcome draws the landing screen with register/login/quit options
func renderWelcome(m model) string {
    options := []string{
        "Register — I am new",
        "Sign In  — I have an account",
        "Quit",
    }

    // Banner — purple on wide terminals, plain text on narrow
    s := ui.BannerStyle.Render(banner(m.state.TermWidth)) + "\n"

    for i, opt := range options {
        if m.state.Cursor == i {
            s += ui.SelectedStyle.Render("  > " + opt) + "\n"
        } else {
            s += ui.MutedStyle.Render("    " + opt) + "\n"
        }
    }

    s += "\n" + ui.Footer("↑/↓ to move · Enter to select · ctrl+c to quit")

    if m.state.FormError != "" {
        s += "\n\n" + ui.ErrorLine(m.state.FormError)
    }
    return s
}

// renderLogin draws the sign-in screen with a live username input
func renderRegister(m model) string {
    fields := []string{
        "First Name",
        "Last Name ",
        "Username  ",
        "State     ",
        "Email     ",
    }
    values := []string{
        m.state.Player.FirstName,
        m.state.Player.LastName,
        m.state.Player.Username,
        m.state.Player.State,
        emailDisplay(m.state.Player.Email),
    }

    s := ui.BannerStyle.Render(banner(m.state.TermWidth))
    s += "\n" + ui.TitleStyle.Render("  REGISTER — Create your account") + "\n\n"

    for i, field := range fields {
        if i == m.state.ActiveField {
            // Active field — purple, bold, cursor
            s += ui.SelectedStyle.Render("  > "+field+": ") +
                m.state.InputBuffer + "_\n"
        } else if values[i] != "" {
            // Completed field — green checkmark
            s += ui.MutedStyle.Render("    "+field+": ") +
                values[i] + " " + ui.Checkmark() + "\n"
        } else {
            // Future field — dimmed
            s += ui.MutedStyle.Render("    "+field+": \n")
        }
    }

    // Live state suggestions
    if m.state.ActiveField == 3 && len(m.state.StateSuggestions) > 0 {
        s += "\n" + ui.InfoStyle.Render("  Suggestions: ")
        for _, state := range m.state.StateSuggestions {
            s += ui.DimStyle.Render(state.Name+" ("+state.Abbreviation+")  ")
        }
        s += "\n"
    }

    if m.state.FormError != "" {
        s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
    }
    if m.state.ActiveField == 4 {
        s += "\n" + ui.WarningStyle.Render("  Email is optional — press Enter to skip") + "\n"
    }
    s += "\n" + ui.Footer("Enter to confirm field · Esc to go back · ctrl+c to quit")
    return s
}

// emailDisplay safely dereferences an optional email pointer for display
func emailDisplay(email *string) string {
	if email == nil {
		return ""
	}
	return *email
}

func renderLogin(m model) string {
    s := ui.BannerStyle.Render(banner(m.state.TermWidth))
    s += "\n" + ui.TitleStyle.Render("  SIGN IN — Enter your username") + "\n\n"
    s += ui.SelectedStyle.Render("  > Username: ") + m.state.InputBuffer + "_\n"

    if m.state.FormError != "" {
        s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
    }
    s += "\n" + ui.Footer("Enter to sign in · Esc to go back · ctrl+c to quit")
    return s
}

// renderMenu draws the main game menu with a personalised greeting
func renderMenu(m model) string {
    options := []string{
        "Single Player",
        "Multiplayer",
        "Quit",
    }
    s := "\n" + ui.TitleStyle.Render("  ROPA-SCI") + "\n"
    s += ui.DimStyle.Render("  Welcome, "+m.state.Player.FirstName+"!") + "\n\n"

    for i, option := range options {
        if m.state.Cursor == i {
            s += ui.SelectedStyle.Render("  > " + option) + "\n"
        } else {
            s += ui.MutedStyle.Render("    " + option) + "\n"
        }
    }
    s += "\n" + ui.Footer("↑/↓ or k/j · Enter to select · Esc to quit")
    return s
}


// ─── Game Screen ──────────────────────────────────────────────────────────────

// MoveCard returns a 6-line ASCII art card for a given move
// The card border doubles when selected to give clear visual feedback
func MoveCard(move models.Move, selected bool) []string {
	b := "─────────────"
	if selected {
		b = "═════════════"
	}

	top := "┌" + b + "┐"
	bot := "└" + b + "┘"

	cards := map[models.Move][]string{
		models.Rock: {
			"       🪨       ", // emoji sits outside the box — no alignment issue
			top,
			"│             │",
			"│    R O C K  │",
			"│   [ 1 / R ] │",
			bot,
		},
		models.Paper: {
			"       📄       ",
			top,
			"│             │",
			"│   P A P E R │",
			"│   [ 2 / P ] │",
			bot,
		},
		models.Scissors: {
			"       ✂️       ",
			top,
			"│             │",
			"│  S C I S S  │",
			"│   [ 3 / S ] │",
			bot,
		},
		models.None: {
			"               ",
			top,
			"│   ? ? ? ? ? │",
			"│   ▓ ▓ ▓ ▓ ▓ │",
			"│   ? ? ? ? ? │",
			bot,
		},
	}
	return cards[move]
}

// renderGame routes to the correct phase renderer based on current game phase
func renderGame(m model) string {
	// Score header is always visible at the top of the game screen
	s := fmt.Sprintf("\n  Round %d of 3  ·  You: %d  ·  AI: %d\n\n",
		m.state.Score.Round,
		m.state.Score.PlayerWins,
		m.state.Score.OpponentWins,
	)
	switch m.state.Phase {
	case "pick":
		// Wide terminal — cards side by side
		if m.state.TermWidth >= 60 {
			s += renderPick(m)
		} else {
			// Narrow terminal — cards stacked vertically
			s += renderPickNarrow(m)
		}
	case "think":
		if m.state.TermWidth >= 60 {
			s += renderThink(m)
		} else {
			s += renderThinkNarrow(m)
		}
	case "reveal":
		if m.state.TermWidth >= 60 {
			s += renderReveal(m)
		} else {
			s += renderRevealNarrow(m)
		}
	case "result":
		s += renderResult(m)
	}
	return s
}

// renderPickNarrow shows cards stacked vertically for narrow terminals
func renderPickNarrow(m model) string {
	moves := []models.Move{models.Rock, models.Paper, models.Scissors}
	labels := []string{"[1/R]", "[2/P]", "[3/S]"}
	s := "  Choose your move:\n\n"
	for i, move := range moves {
		selected := m.state.Cursor == i
		prefix := "  "
		if selected {
			prefix = "> "
		}
		name := string(move)
		if selected {
			s += "  " + prefix + "[ " + strings.ToUpper(name) + " " + labels[i] + " ] ◀\n"
		} else {
			s += "  " + prefix + "[ " + strings.ToUpper(name) + " " + labels[i] + " ]\n"
		}
	}
	s += "\n  ↑/↓ to choose · Enter or 1/2/3 to confirm\n"
	s += "  Esc to return to menu\n"
	return s
}

// renderThinkNarrow shows think phase for narrow terminals
func renderThinkNarrow(m model) string {
	spinner := spinnerFrames[m.state.SpinnerFrame]
	s := "  YOUR MOVE\n\n"
	s += "  [ " + strings.ToUpper(string(m.state.PlayerMove)) + " ]\n\n"
	s += "        VS\n\n"
	s += "  AI's MOVE\n\n"
	s += "  [  ???  ]\n\n"
	s += "  " + spinner + "  AI is calculating...\n"
	return s
}

// renderRevealNarrow shows both moves stacked for narrow terminals
func renderRevealNarrow(m model) string {
	s := "  YOUR MOVE\n\n"
	s += "  [ " + strings.ToUpper(string(m.state.PlayerMove)) + " ]\n\n"
	s += "        VS\n\n"
	s += "  AI's MOVE\n\n"
	s += "  [ " + strings.ToUpper(string(m.state.AIMove)) + " ]\n\n"
	return s
}

// renderPick shows three selectable move cards side by side
func renderPick(m model) string {
	s := "  Choose your move:\n\n"
	moves := []models.Move{models.Rock, models.Paper, models.Scissors}
	cards := make([][]string, 3)
	for i, move := range moves {
		cards[i] = MoveCard(move, m.state.Cursor == i)
	}
	// Print all three cards row by row so they sit side by side
	for row := 0; row < 6; row++ {
		s += "  "
		for col := 0; col < 3; col++ {
			s += cards[col][row] + "  "
		}
		s += "\n"
	}
	s += "\n  ← → to choose · Enter or 1/2/3 or R/P/S to confirm\n"
	s += "  Esc to return to menu\n"
	return s
}

// renderThink shows the player's locked-in move alongside a face-down AI card
// The spinner animates to build tension while the AI "decides"
func renderThink(m model) string {
	s := "  YOUR MOVE                AI's MOVE\n\n"
	playerCard := MoveCard(m.state.PlayerMove, false)
	aiCard := MoveCard(models.None, false)
	for i := 0; i < 6; i++ {
		s += "  " + playerCard[i] + "    VS    " + aiCard[i] + "\n"
	}
	spinner := spinnerFrames[m.state.SpinnerFrame]
	s += "\n  " + spinner + "  AI is calculating its move...\n"
	return s
}

// renderReveal shows both moves side by side before the result appears
func renderReveal(m model) string {
	s := "  YOUR MOVE                AI's MOVE\n\n"
	playerCard := MoveCard(m.state.PlayerMove, false)
	aiCard := MoveCard(m.state.AIMove, false)
	for i := 0; i < 6; i++ {
		s += "  " + playerCard[i] + "    VS    " + aiCard[i] + "\n"
	}
	return s
}

// renderResult shows the round outcome, score bar, and contextual next-step prompt
func renderResult(m model) string {
    s := ""
    var boxContent string

    switch m.state.RoundOutcome {
    case "win":
        s += ui.SuccessStyle.Render("  🏆  YOU WIN THIS ROUND!  🏆") + "\n\n"
        boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
        s += "  " + ui.WinBoxStyle.Render(boxContent) + "\n"
    case "lose":
        s += ui.DangerStyle.Render("  💀  AI WINS THIS ROUND...  💀") + "\n\n"
        boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
        s += "  " + ui.LoseBoxStyle.Render(boxContent) + "\n"
    case "tie":
        s += ui.WarningStyle.Render("  🤝  DRAW!  🤝") + "\n\n"
        boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
        s += "  " + ui.TieBoxStyle.Render(boxContent) + "\n"
    }

    s += "\n  " + scoreBar(m.state.Score.PlayerWins, m.state.Score.OpponentWins) + "\n"

    if m.state.Score.PlayerWins == 2 {
        s += "\n" + ui.SuccessStyle.Render("  🎉🎉  YOU WIN THE MATCH!  🎉🎉") + "\n"
        s += "\n" + ui.Footer("Enter to play again · Esc for menu")
    } else if m.state.Score.OpponentWins == 2 {
        s += "\n" + ui.DangerStyle.Render("  💀  AI wins the match. Better luck next time.") + "\n"
        s += "\n" + ui.Footer("Enter to play again · Esc for menu")
    } else {
        switch m.state.RoundOutcome {
        case "win":
            s += "\n" + ui.Footer("Enter for next round · Esc for menu")
        case "lose":
            s += "\n" + ui.Footer("Enter to fight back · Esc for menu")
        case "tie":
            s += "\n" + ui.Footer("Enter to break the tie · Esc for menu")
        }
    }
    return s
}

// ─── Render Utilities ─────────────────────────────────────────────────────────

// scoreBar renders a visual two-pip progress bar for the current match score
// e.g.  You: █░  AI: ░░
func scoreBar(playerWins, aiWins int) string {
    bar := ui.DimStyle.Render("You: ")
    for i := 0; i < 2; i++ {
        bar += ui.ScorePip(i < playerWins)
    }
    bar += ui.DimStyle.Render("  AI: ")
    for i := 0; i < 2; i++ {
        bar += ui.ScorePip(i < aiWins)
    }
    return bar
}

// padCenter centres a string within a fixed width by adding spaces on both sides
func padCenter(s string, width int) string {
	if len(s) >= width {
		return s
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}

// renderMultiMenu draws the multiplayer mode selection screen
func renderMultiMenu(m model) string {
    options := []string{
        "Create Room  — get a code to share",
        "Quick Match  — find any opponent",
        "Back",
    }
    s := "\n" + ui.TitleStyle.Render("  MULTIPLAYER") + "\n\n"
    s += ui.DimStyle.Render("  Choose how you want to play:") + "\n\n"
    for i, opt := range options {
        if m.state.Cursor == i {
            s += ui.SelectedStyle.Render("  > " + opt) + "\n"
        } else {
            s += ui.MutedStyle.Render("    " + opt) + "\n"
        }
    }
    s += "\n" + ui.Footer("↑/↓ to move · Enter to select · Esc for menu")
    return s
}
// renderCreateRoom draws the room code waiting screen
func renderCreateRoom(m model) string {
    spinner := spinnerFrames[m.state.SpinnerFrame]
    s := "\n" + ui.TitleStyle.Render("  CREATE ROOM") + "\n\n"
    roomContent := ui.TitleStyle.Render("Your room code:\n\n") +
        ui.SuccessStyle.Render("  "+m.state.RoomCode)
    s += "  " + ui.RoomCodeBoxStyle.Render(roomContent) + "\n"
    s += "\n" + ui.DimStyle.Render("  Share this code with your opponent.") + "\n"
    s += "\n" + ui.DimStyle.Render("  "+spinner+"  Waiting for opponent to join...") + "\n"
    s += "\n" + ui.Footer("Esc to cancel · ctrl+c to quit")
    return s
}

// renderQuickMatch draws the auto matchmaking waiting screen
func renderQuickMatch(m model) string {
    spinner := spinnerFrames[m.state.SpinnerFrame]
    s := "\n" + ui.TitleStyle.Render("  QUICK MATCH") + "\n\n"
    s += ui.DimStyle.Render("  "+spinner+"  Searching for an opponent...") + "\n\n"
    waitContent := ui.DimStyle.Render("This may take a moment.\nStay in the terminal!")
    s += "  " + ui.WaitingBoxStyle.Render(waitContent) + "\n"
    s += "\n" + ui.Footer("Esc to cancel · ctrl+c to quit")
    return s
}

// ─── Entry Point ──────────────────────────────────────────────────────────────

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),       // full-screen mode for a cleaner game feel
		tea.WithMouseCellMotion(), // mouse infrastructure ready for Week 7
	)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
