package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"ropa-sci-frontend/bubbletea/models"
	"ropa-sci-frontend/bubbletea/server"
	"ropa-sci-frontend/bubbletea/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// wsMsgMsg wraps incoming LAN server messages
type wsMsgMsg struct {
	msg server.WSMessage
}

// wsErrMsg wraps LAN connection errors
type wsErrMsg struct {
	err error
}

// spinnerFrames is the Braille dot animation cycle used during AI think phase
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ─── Tea Commands ─────────────────────────────────────────────────────────────

// spinnerTick returns a command that fires a spinnerTickMsg every 100ms
func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// aiThinkCmd returns a command that fires after 1.5s with the AI's chosen move
func (m model) aiThinkCmd() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		return aiDecidedMsg{move: m.aiEngine.ChooseMove()}
	})
}

// readNetMsg reads a single WebSocket message in a background Bubbletea thread
func readNetMsg(conn net.Conn) tea.Cmd {
	return func() tea.Msg {
		if conn == nil {
			return wsErrMsg{err: fmt.Errorf("connection is closed")}
		}
		payload, err := server.ReadWSFrame(conn)
		if err != nil {
			return wsErrMsg{err: err}
		}
		var wsMsg server.WSMessage
		if err := json.Unmarshal(payload, &wsMsg); err != nil {
			return wsErrMsg{err: err}
		}
		return wsMsgMsg{msg: wsMsg}
	}
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
	state           models.GameState
	aiEngine        *models.AIEngine      // runtime AI engine
	lanServer       *server.LANGameServer // P2P Host Server
	wsConn          net.Conn              // active WebSocket client conn
	netOpponentName string                // remote player name
	nextWinsHost    int                   // temporary host score buffer
	nextWinsGuest   int                   // temporary guest score buffer
}

// initialModel returns the app's starting state — always begins on the welcome screen
func initialModel() model {
	return model{
		state: models.GameState{
			Screen: "welcome",
			Score:  models.MatchScore{Round: 1},
			Cursor: 0,
			Phase:  models.PhasePick,
		},
		aiEngine: models.NewAIEngine(models.AIDifficultyHard),
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
		if m.state.Phase == models.PhaseThink ||
			m.state.Screen == "create-room" ||
			m.state.Screen == "quick-match" {
			return m, spinnerTick()
		}
		return m, nil

	// ── AI decided — transition from think to reveal, then schedule result ──
	case aiDecidedMsg:
		m.state.AIMove = msg.move
		m.state.Phase = models.PhaseReveal
		m.state.RoundOutcome = calculateOutcome(m.state.PlayerMove, msg.move)
		// Brief pause so the player can see both cards before the verdict
		return m, tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
			return showResultMsg{}
		})

	// ── Show result — update score and transition to result phase ──
	case showResultMsg:
		m.state.Phase = models.PhaseResult
		if m.state.GameMode == "multi" {
			m.state.Score.PlayerWins = m.nextWinsHost
			m.state.Score.OpponentWins = m.nextWinsGuest
		} else {
			switch m.state.RoundOutcome {
			case "win":
				m.state.Score.PlayerWins++
			case "lose":
				m.state.Score.OpponentWins++
			}
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

	// ── Incoming LAN server network packets ──
	case wsMsgMsg:
		switch msg.msg.Type {
		case "start":
			m.state.Screen = "game"
			m.state.GameMode = "multi"
			m.state.Phase = models.PhasePick
			m.state.Score = models.MatchScore{Round: 1}
			m.netOpponentName = msg.msg.Payload
			m.state.PlayerMove = models.None
			m.state.AIMove = models.None
			m.state.RoundOutcome = ""
			m.state.Cursor = 0
			slog.Info("Multiplayer match started", "opponent", m.netOpponentName)
			return m, readNetMsg(m.wsConn)

		case "round":
			var payload server.RoundOutcomePayload
			if err := json.Unmarshal([]byte(msg.msg.Payload), &payload); err != nil {
				slog.Error("Failed to parse round outcome payload", "error", err)
				return m, readNetMsg(m.wsConn)
			}
			m.state.AIMove = models.Move(payload.OpponentMove)
			m.state.RoundOutcome = payload.Outcome
			m.nextWinsHost = payload.YourWins
			m.nextWinsGuest = payload.OpponentWins
			m.state.Phase = models.PhaseReveal
			slog.Info("Round outcome received", "outcome", payload.Outcome, "wins", payload.YourWins, "losses", payload.OpponentWins)

			return m, tea.Batch(
				readNetMsg(m.wsConn),
				tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
					return showResultMsg{}
				}),
			)

		case "match":
			m.state.Player.TotalMatches++
			if msg.msg.Payload == "win" {
				m.state.Player.Wins++
			} else {
				m.state.Player.Losses++
			}
			_ = models.UpdatePlayer(m.state.Player)
			return m, readNetMsg(m.wsConn)

		case "error":
			m.state.FormError = msg.msg.Payload
			m.state.Screen = "multi-menu"
			if m.wsConn != nil {
				_ = m.wsConn.Close()
				m.wsConn = nil
			}
			if m.lanServer != nil {
				m.lanServer.Stop()
				m.lanServer = nil
			}
			return m, nil
		}
		return m, readNetMsg(m.wsConn)

	// ── LAN Connection errors ──
	case wsErrMsg:
		slog.Error("Network connection error", "error", msg.err)
		m.state.FormError = "Connection error: " + msg.err.Error()
		m.state.Screen = "multi-menu"
		if m.wsConn != nil {
			_ = m.wsConn.Close()
			m.wsConn = nil
		}
		if m.lanServer != nil {
			m.lanServer.Stop()
			m.lanServer = nil
		}
		return m, nil

	// ── Mouse clicks — coordinate mapping for menu and game screens ──
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			switch m.state.Screen {

			// Welcome screen — click highlights a menu option
			case "welcome":
				offset := 8 // wide banner (ASCII art) ends at Y=7, options start at Y=8
				if m.state.TermWidth < 70 {
					offset = 3 // narrow banner is just text
				}
				idx := msg.Y - offset
				if idx >= 0 && idx < 3 {
					m.state.Cursor = idx
				}

			// Main menu — title + greeting + blank = 4 lines of header
			case "menu":
				idx := msg.Y - 4
				if idx >= 0 && idx < 3 {
					m.state.Cursor = idx
				}

			// Multiplayer menu — title + subtitle + blanks = 5 lines of header
			case "multi-menu":
				idx := msg.Y - 5
				if idx >= 0 && idx < 3 {
					m.state.Cursor = idx
				}

			// Difficulty menu — Y starts at 4
			case "difficulty":
				idx := msg.Y - 4
				if idx >= 0 && idx < 4 {
					m.state.Cursor = idx
				}

			// Game screen — click on a card to play it immediately
			case "game":
				if m.state.Phase == models.PhasePick {
					if m.state.TermWidth >= 60 {
						// Wide layout: cards are ~15 chars wide, joined side-by-side
						cardIdx := -1
						if msg.X >= 2 && msg.X <= 16 {
							cardIdx = 0
						} else if msg.X >= 19 && msg.X <= 33 {
							cardIdx = 1
						} else if msg.X >= 36 && msg.X <= 50 {
							cardIdx = 2
						}
						if cardIdx >= 0 && msg.Y >= 5 && msg.Y <= 12 {
							move := indexToMove(cardIdx)
							m.state.PlayerMove = move
							m.state.Phase = models.PhaseThink
							if m.state.GameMode == "multi" {
								_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "move", Payload: string(move)})
								return m, spinnerTick()
							} else {
								m.aiEngine.RecordPlayerMove(move)
								return m, tea.Batch(spinnerTick(), m.aiThinkCmd())
							}
						}
					} else {
						// Narrow layout: cards stacked at Y=5,6,7
						idx := msg.Y - 5
						if idx >= 0 && idx < 3 {
							m.state.Cursor = idx
						}
					}
				}
				// Click anywhere during result phase advances to next round
				if m.state.Phase == models.PhaseResult {
					if m.state.Score.PlayerWins == 2 || m.state.Score.OpponentWins == 2 {
						m.state.Score = models.MatchScore{Round: 1}
						if m.state.GameMode == "multi" {
							m.state.Screen = "multi-menu"
							m.state.Cursor = 0
							if m.wsConn != nil {
								_ = m.wsConn.Close()
								m.wsConn = nil
							}
							if m.lanServer != nil {
								m.lanServer.Stop()
								m.lanServer = nil
							}
							return m, nil
						}
					}
					m.state.Phase = models.PhasePick
					m.state.PlayerMove = models.None
					m.state.AIMove = models.None
					m.state.RoundOutcome = ""
					m.state.Cursor = 0
				}
			}
		}

	// ── Keyboard input ──────────────────────────────────────────────────────
	case tea.KeyMsg:
		// ── Form screen input guard ─────────────────────────────────────
		// On text-input screens (login, register, join-room), intercept
		// ALL single printable characters BEFORE game shortcuts (r/p/s)
		// and vim navigation keys (h/j/k/l) can silently swallow them.
		// Only structural keys (ctrl+c, esc, enter, backspace) pass through.
		if isFormScreen(m.state.Screen) {
			key := msg.String()
			if len(key) == 1 {
				if len(m.state.InputBuffer) >= models.MaxInputLength {
					return m, nil
				}
				r := rune(key[0])
				valid := false

				switch m.state.Screen {
				case "login":
					// Auto-lowercase so capslock doesn't block input
					if r >= 'A' && r <= 'Z' {
						r = r + 32 // ASCII uppercase → lowercase
						key = string(r)
					}
					valid = models.IsValidUsernameChar(r)
				case "join-room":
					valid = (r >= '0' && r <= '9') || r == '.' || r == ':' ||
						(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-'
				case "register":
					switch m.state.ActiveField {
					case 0, 1: // First Name, Last Name
						valid = models.IsValidNameChar(r)
					case 2: // Username — auto-lowercase
						if r >= 'A' && r <= 'Z' {
							r = r + 32
							key = string(r)
						}
						valid = models.IsValidUsernameChar(r)
					case 3: // State
						valid = models.IsValidStateChar(r)
					case 4: // Email
						valid = models.IsValidEmailChar(r)
					}
				}

				if valid {
					m.state.InputBuffer += key
					m.state.FormError = ""
					if m.state.Screen == "register" && m.state.ActiveField == 3 {
						m.state.StateSuggestions = models.SuggestStates(m.state.InputBuffer)
					}
				}
				return m, nil
			}
		}

		switch msg.String() {

		// Hard quit — works everywhere, no exceptions
		case "ctrl+c":
			return m, tea.Quit

		// Soft quit — blocked on form screens to allow typing the letter q
		case "q":
			// Form screens are already handled by the guard above, so this
			// case only fires on non-form screens where q should quit.
			return m, tea.Quit

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

			case "game", "waiting", "create-room", "join-room", "quick-match":
				m.state.Screen = "menu"
				m.state.Phase = models.PhasePick
				m.state.Cursor = 0
				m.state.FormError = ""
				m.state.RoomCode = ""
				if m.wsConn != nil {
					_ = m.wsConn.Close()
					m.wsConn = nil
				}
				if m.lanServer != nil {
					m.lanServer.Stop()
					m.lanServer = nil
				}
			case "difficulty":
				m.state.Screen = "menu"
				m.state.Cursor = 0

			case "multi-menu":
				m.state.Screen = "menu"
				m.state.Cursor = 1
			case "result":
				m.state.Screen = "menu"
				m.state.Cursor = 0
				m.state.Score = models.MatchScore{Round: 1}
				m.state.Phase = models.PhasePick
				if m.wsConn != nil {
					_ = m.wsConn.Close()
					m.wsConn = nil
				}
				if m.lanServer != nil {
					m.lanServer.Stop()
					m.lanServer = nil
				}
			case "help":
				m.state.Screen = m.state.PreviousScreen
			case "admin":
				return m, tea.Quit
			case "admin-players":
				m.state.Screen = "admin"
				m.state.Cursor = 0
				m.state.FormError = ""
			case "admin-player-detail":
				if m.state.AdminConfirm != "" {
					m.state.AdminConfirm = ""
				} else {
					m.state.Screen = "admin-players"
					m.state.FormError = ""
				}
			case "menu":
				if m.state.Player.IsAdmin() {
					m.state.Screen = "admin"
					m.state.Cursor = 0
					m.state.FormError = ""
				}
			}

		// Menu cursor — up arrow and vim-style k
		case "up", "k":
			if m.state.Screen == "admin-players" {
				if m.state.AdminSelectedIndex > 0 {
					m.state.AdminSelectedIndex--
				}
			} else if isMenuScreen(m.state.Screen) && m.state.Cursor > 0 {
				m.state.Cursor--
			}

		// Menu cursor — down arrow and vim-style j
		case "down", "j":
			if m.state.Screen == "admin-players" {
				if m.state.AdminSelectedIndex < len(m.state.AdminPlayers)-1 {
					m.state.AdminSelectedIndex++
				}
			} else if isMenuScreen(m.state.Screen) && m.state.Cursor < menuLength(m.state.Screen)-1 {
				m.state.Cursor++
			}

		// Game card cursor — left arrow and vim-style h
		case "left", "h":
			if m.state.Screen == "game" && m.state.Phase == models.PhasePick && m.state.Cursor > 0 {
				m.state.Cursor--
			} else if m.state.Screen == "help" && m.state.Cursor > 0 {
				m.state.Cursor--
			}

		// Game card cursor — right arrow and vim-style l
		case "right", "l":
			if m.state.Screen == "game" && m.state.Phase == models.PhasePick && m.state.Cursor < 2 {
				m.state.Cursor++
			} else if m.state.Screen == "help" && m.state.Cursor < 1 {
				m.state.Cursor++
			}

		// Direct move shortcuts — Rock
		case "1", "r", "R":
			if m.state.Screen == "game" && m.state.Phase == models.PhasePick {
				m.state.PlayerMove = models.Rock
				m.state.Phase = models.PhaseThink
				if m.state.GameMode == "multi" {
					_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "move", Payload: string(models.Rock)})
					return m, spinnerTick()
				} else {
					m.aiEngine.RecordPlayerMove(models.Rock)
					return m, tea.Batch(spinnerTick(), m.aiThinkCmd())
				}
			} else if m.state.Screen == "admin-player-detail" && m.state.AdminConfirm == "" {
				target := m.state.AdminPlayers[m.state.AdminSelectedIndex]
				if target.IsRootAdmin() {
					m.state.FormError = "Permission denied: Root administrator stats cannot be reset"
				} else if target.IsAdmin() && !m.state.Player.IsRootAdmin() {
					m.state.FormError = "Permission denied: Standard admins cannot modify other admins"
				} else {
					m.state.AdminConfirm = "reset"
					m.state.FormError = ""
				}
			}

		// Direct move shortcuts — Paper
		case "2", "p", "P":
			if m.state.Screen == "game" && m.state.Phase == models.PhasePick {
				m.state.PlayerMove = models.Paper
				m.state.Phase = models.PhaseThink
				if m.state.GameMode == "multi" {
					_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "move", Payload: string(models.Paper)})
					return m, spinnerTick()
				} else {
					m.aiEngine.RecordPlayerMove(models.Paper)
					return m, tea.Batch(spinnerTick(), m.aiThinkCmd())
				}
			} else if m.state.Screen == "admin-player-detail" && m.state.AdminConfirm == "" {
				target := m.state.AdminPlayers[m.state.AdminSelectedIndex]
				if target.IsRootAdmin() {
					m.state.FormError = "Permission denied: Root administrator role cannot be modified"
				} else if target.Username == m.state.Player.Username {
					m.state.FormError = "Permission denied: You cannot demote yourself"
				} else if !m.state.Player.IsRootAdmin() {
					m.state.FormError = "Permission denied: Only the Root Admin can promote/demote admins"
				} else {
					m.state.AdminConfirm = "role"
					m.state.FormError = ""
				}
			}

		// Direct move shortcuts — Scissors
		case "3", "s", "S":
			if m.state.Screen == "game" && m.state.Phase == models.PhasePick {
				m.state.PlayerMove = models.Scissors
				m.state.Phase = models.PhaseThink
				if m.state.GameMode == "multi" {
					_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "move", Payload: string(models.Scissors)})
					return m, spinnerTick()
				} else {
					m.aiEngine.RecordPlayerMove(models.Scissors)
					return m, tea.Batch(spinnerTick(), m.aiThinkCmd())
				}
			}

		// Delete Player shortcut (admin only)
		case "d", "D":
			if m.state.Screen == "admin-player-detail" && m.state.AdminConfirm == "" {
				target := m.state.AdminPlayers[m.state.AdminSelectedIndex]
				if target.IsRootAdmin() {
					m.state.FormError = "Permission denied: Root administrator cannot be deleted"
				} else if target.Username == m.state.Player.Username {
					m.state.FormError = "Permission denied: You cannot delete yourself"
				} else if target.IsAdmin() && !m.state.Player.IsRootAdmin() {
					m.state.FormError = "Permission denied: Standard admins cannot delete other admins"
				} else {
					m.state.AdminConfirm = "delete"
					m.state.FormError = ""
				}
			}

		// Enter — confirms selections and form fields
		case "enter":
			switch m.state.Screen {
			case "admin":
				switch m.state.Cursor {
				case 0: // Manage Players
					players, err := models.LoadPlayers()
					if err != nil {
						m.state.FormError = "Failed to load players: " + err.Error()
						return m, nil
					}
					m.state.AdminPlayers = players
					m.state.AdminSelectedIndex = 0
					m.state.Screen = "admin-players"
					m.state.FormError = ""
				case 1: // Play Game
					m.state.PreviousScreen = "admin"
					m.state.Screen = "menu"
					m.state.Cursor = 0
					m.state.FormError = ""
				case 2: // Quit
					return m, tea.Quit
				}

			case "admin-players":
				if len(m.state.AdminPlayers) > 0 {
					m.state.Screen = "admin-player-detail"
					m.state.FormError = ""
					m.state.AdminConfirm = ""
				}

			case "admin-player-detail":
				if m.state.AdminConfirm != "" {
					target := m.state.AdminPlayers[m.state.AdminSelectedIndex]
					var err error

					switch m.state.AdminConfirm {
					case "reset":
						err = models.ResetPlayerStats(target.Username)
						if err == nil {
							m.state.FormError = "Stats reset successfully!"
						}
					case "delete":
						err = models.DeletePlayer(target.Username)
						if err == nil {
							m.state.Screen = "admin-players"
							m.state.FormError = "Player deleted successfully!"
							m.state.AdminSelectedIndex = 0
						}
					case "role":
						newRole := "admin"
						if target.Role == "admin" {
							newRole = "player"
						}
						err = models.SetPlayerRole(target.Username, newRole)
						if err == nil {
							m.state.FormError = fmt.Sprintf("Role updated to %s!", newRole)
						}
					}

					if err != nil {
						m.state.FormError = "Error: " + err.Error()
					}

					// Reload database state
					players, loadErr := models.LoadPlayers()
					if loadErr == nil {
						m.state.AdminPlayers = players
						// Ensure index is valid
						if m.state.AdminSelectedIndex >= len(players) {
							m.state.AdminSelectedIndex = len(players) - 1
						}
						if m.state.AdminSelectedIndex < 0 {
							m.state.AdminSelectedIndex = 0
						}
					}
					m.state.AdminConfirm = ""
				}

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
					if errMsg := models.ValidateName(input); errMsg != "" {
						m.state.FormError = errMsg
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
					if errMsg := models.ValidateName(input); errMsg != "" {
						m.state.FormError = errMsg
						return m, nil
					}
					m.state.Player.LastName = caser.String(strings.ToLower(input)) // normalise then capitalise
					m.state.ActiveField++
					m.state.InputBuffer = ""
					m.state.FormError = ""

				case 2: // Username
					if errMsg := models.ValidateUsername(strings.ToLower(input)); errMsg != "" {
						m.state.FormError = errMsg
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
					if player.IsAdmin() {
						m.state.Screen = "admin"
					} else {
						m.state.Screen = "menu"
					}
					m.state.InputBuffer = ""
					m.state.FormError = ""
					m.state.Cursor = 0
				}

			// ── Main menu — route to game mode or quit ──
			case "menu":
				switch m.state.Cursor {
				case 0:
					m.state.PreviousScreen = "menu"
					m.state.Screen = "difficulty"
					m.state.Cursor = 0
				case 1:
					m.state.PreviousScreen = "menu"
					m.state.Screen = "multi-menu"
					m.state.GameMode = "multi"
					m.state.Cursor = 0
				case 2:
					return m, tea.Quit
				}

			case "difficulty":
				switch m.state.Cursor {
				case 0:
					m.aiEngine = models.NewAIEngine(models.AIDifficultyEasy)
				case 1:
					m.aiEngine = models.NewAIEngine(models.AIDifficultyMedium)
				case 2:
					m.aiEngine = models.NewAIEngine(models.AIDifficultyHard)
				case 3: // Back
					m.state.Screen = "menu"
					m.state.Cursor = 0
					return m, nil
				}
				m.state.Screen = "game"
				m.state.GameMode = "single"
				m.state.Phase = models.PhasePick
				m.state.Score = models.MatchScore{Round: 1}
				m.state.PlayerMove = models.None
				m.state.AIMove = models.None
				m.state.RoundOutcome = ""

			case "multi-menu":
				switch m.state.Cursor {
				case 0:
					// Create Room (Host)
					m.lanServer = server.NewLANServer()
					port, err := m.lanServer.Start(8080)
					if err != nil {
						m.state.FormError = "Host failed: " + err.Error()
						return m, nil
					}
					localIP := getLocalIP()
					addr := fmt.Sprintf("%s:%d", localIP, port)
					m.state.RoomCode = addr

					conn, err := server.DialLANServer(fmt.Sprintf("127.0.0.1:%d", port))
					if err != nil {
						m.lanServer.Stop()
						m.lanServer = nil
						m.state.FormError = "Host connect failed: " + err.Error()
						return m, nil
					}
					m.wsConn = conn
					_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "join", Payload: m.state.Player.Username})

					m.state.Screen = "create-room"
					m.state.PreviousScreen = "multi-menu"
					m.state.Cursor = 0
					return m, tea.Batch(spinnerTick(), readNetMsg(m.wsConn))

				case 1:
					// Join Room
					m.state.Screen = "join-room"
					m.state.InputBuffer = "127.0.0.1:8080"
					m.state.FormError = ""
					m.state.PreviousScreen = "multi-menu"
					m.state.Cursor = 0

				case 2:
					m.state.Screen = "menu"
					m.state.Cursor = 1
				}

			case "join-room":
				input := strings.TrimSpace(m.state.InputBuffer)
				if input == "" {
					m.state.FormError = "Host address required"
					return m, nil
				}
				conn, err := server.DialLANServer(input)
				if err != nil {
					m.state.FormError = "Connect failed: " + err.Error()
					return m, nil
				}
				m.wsConn = conn
				_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "join", Payload: m.state.Player.Username})

				m.state.Screen = "waiting"
				m.state.FormError = ""
				return m, tea.Batch(spinnerTick(), readNetMsg(m.wsConn))

			// ── Game screen — confirm move or continue after result ──
			case "game":
				switch m.state.Phase {
				case models.PhasePick:
					move := indexToMove(m.state.Cursor)
					m.state.PlayerMove = move
					m.state.Phase = models.PhaseThink
					if m.state.GameMode == "multi" {
						_ = server.WriteClientMessage(m.wsConn, server.WSMessage{Type: "move", Payload: string(move)})
						return m, spinnerTick()
					} else {
						m.aiEngine.RecordPlayerMove(move)
						return m, tea.Batch(spinnerTick(), m.aiThinkCmd())
					}

				case models.PhaseResult:
					if m.state.Score.PlayerWins == 2 || m.state.Score.OpponentWins == 2 {
						m.state.Score = models.MatchScore{Round: 1}
						if m.state.GameMode == "multi" {
							m.state.Screen = "multi-menu"
							m.state.Cursor = 0
							if m.wsConn != nil {
								_ = m.wsConn.Close()
								m.wsConn = nil
							}
							if m.lanServer != nil {
								m.lanServer.Stop()
								m.lanServer = nil
							}
							return m, nil
						}
					}
					m.state.Phase = models.PhasePick
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

				// ? opens help screen — but only on non-input screens
				// On login/register/join-room, ? falls through to normal typing below
				if char == "?" && m.state.Screen != "login" && m.state.Screen != "register" && m.state.Screen != "join-room" {
					// Block during think/reveal phases — animation is running
					if m.state.Screen != "game" || m.state.Phase == models.PhasePick || m.state.Phase == models.PhaseResult {
						m.state.PreviousScreen = m.state.Screen
						m.state.Screen = "help"
						m.state.Cursor = 0
					}
					return m, nil
				}

				// Enforce maximum input length across all fields
				if len(m.state.InputBuffer) >= models.MaxInputLength {
					return m, nil
				}

				// Login — only accept valid username characters
				if m.state.Screen == "login" {
					r := rune(char[0])
					if models.IsValidUsernameChar(r) {
						m.state.InputBuffer += char
					}
				}

				// Join Room — accept IP, port, dots, colons, localhost text
				if m.state.Screen == "join-room" {
					r := rune(char[0])
					if (r >= '0' && r <= '9') || r == '.' || r == ':' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '-' {
						m.state.InputBuffer += char
					}
				}

				// Register — filter characters based on which field is active
				if m.state.Screen == "register" {
					r := rune(char[0])
					valid := false
					switch m.state.ActiveField {
					case 0, 1: // First Name, Last Name
						valid = models.IsValidNameChar(r)
					case 2: // Username
						valid = models.IsValidUsernameChar(r)
					case 3: // State
						valid = models.IsValidStateChar(r)
					case 4: // Email
						valid = models.IsValidEmailChar(r)
					}
					if valid {
						m.state.InputBuffer += char
						if m.state.ActiveField == 3 {
							m.state.StateSuggestions = models.SuggestStates(m.state.InputBuffer)
						}
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
	return screen == "welcome" || screen == "menu" || screen == "multi-menu" || screen == "difficulty" || screen == "admin"
}

// isFormScreen returns true for any screen that accepts text input.
// On these screens, single-character keys must be routed to the input buffer
// instead of being consumed by game shortcuts or vim navigation bindings.
func isFormScreen(screen string) bool {
	return screen == "login" || screen == "register" || screen == "join-room"
}

// menuLength returns the number of selectable options for a given menu screen
func menuLength(screen string) int {
	switch screen {
	case "welcome":
		return 3 // Register, Sign In, Quit
	case "menu":
		return 3 // Single Player, Multiplayer, Quit
	case "multi-menu":
		return 3 // Host, Join, Back
	case "difficulty":
		return 4 // Easy, Medium, Hard, Back
	case "admin":
		return 3 // Manage Players, Play Game, Quit
	default:
		return 0
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

// View renders the current screen as a string — called automatically on every state change.
// All content is wrapped in an AppContainerStyle panel and centered in the terminal.
func (m model) View() string {
	// Global guard — terminal too narrow to render safely
	if m.state.TermWidth > 0 && m.state.TermWidth < 50 {
		return lipgloss.Place(
			m.state.TermWidth, m.state.TermHeight,
			lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("⚠  Terminal too narrow!\n\nPlease resize to at least 50 columns.\nCurrent width: %d columns.", m.state.TermWidth),
		)
	}
	// Too short vertically
	if m.state.TermHeight > 0 && m.state.TermHeight < 16 {
		return lipgloss.Place(
			m.state.TermWidth, m.state.TermHeight,
			lipgloss.Center, lipgloss.Center,
			"⚠  Terminal too short!\n\nPlease resize to at least 16 rows.",
		)
	}

	var content string
	switch m.state.Screen {
	case "welcome":
		content = renderWelcome(m)
	case "register":
		content = renderRegister(m)
	case "login":
		content = renderLogin(m)
	case "menu":
		content = renderMenu(m)
	case "difficulty":
		content = renderDifficulty(m)
	case "game":
		content = renderGame(m)
	case "multi-menu":
		content = renderMultiMenu(m)
	case "create-room":
		content = renderCreateRoom(m)
	case "join-room":
		content = renderJoinRoom(m)
	case "waiting":
		content = renderQuickMatch(m)
	case "help":
		content = renderHelp(m)
	case "admin":
		content = renderAdmin(m)
	case "admin-players":
		content = renderAdminPlayers(m)
	case "admin-player-detail":
		content = renderAdminPlayerDetail(m)
	default:
		content = "Unknown screen\n\nctrl+c to quit"
	}

	// Wrap in the master container panel, then center in the full terminal
	styledContent := ui.AppContainerStyle.Render(content)
	return lipgloss.Place(
		m.state.TermWidth, m.state.TermHeight,
		lipgloss.Center, lipgloss.Center,
		styledContent,
	)
}

// ─── Screen Renderers ─────────────────────────────────────────────────────────

// banner returns a compact styled ASCII logo that fits within the container panel
func banner() string {
	return `
 ╔═══════════════════════════════════════════╗
 ║   ██████   ██████  ██████   █████         ║
 ║   ██  ██  ██    ██ ██  ██  ██   ██        ║
 ║   ██████  ██    ██ ██████  ███████        ║
 ║   ██  ██  ██    ██ ██      ██   ██        ║
 ║   ██  ██   ██████  ██      ██   ██        ║
 ║───────────────── S C I ──────────────────║
 ╚═══════════════════════════════════════════╝`
}

// renderWelcome draws the landing screen with register/login/quit options
func renderWelcome(m model) string {
	options := []string{
		"✦  Register — I am new",
		"✦  Sign In  — I have an account",
		"✦  Quit",
	}

	s := ui.BannerStyle.Render(banner()) + "\n\n"
	s += ui.HeaderStyle.Render("Rock · Paper · Scissors") + "\n\n"
	s += ui.Divider() + "\n\n"

	for i, opt := range options {
		if m.state.Cursor == i {
			s += ui.SelectedStyle.Render(" ▸ "+opt) + "\n"
		} else {
			s += ui.MutedStyle.Render("   "+opt) + "\n"
		}
	}

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError)
	}
	s += "\n" + ui.Footer("↑/↓ navigate · Enter select · ctrl+c quit")
	return s
}

// renderRegister draws the registration form with five fields inside a styled panel
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

	s := ui.BannerStyle.Render(banner()) + "\n"
	s += ui.TitleStyle.Render("REGISTER") + "\n"
	s += ui.DimStyle.Render("Create your player account") + "\n\n"

	var formContent string
	for i, field := range fields {
		if i == m.state.ActiveField {
			formContent += ui.SelectedStyle.Render(" ▸ "+field+": ") +
				m.state.InputBuffer + "▊\n"
		} else if values[i] != "" {
			formContent += ui.DimStyle.Render("   "+field+": ") +
				values[i] + " " + ui.Checkmark() + "\n"
		} else {
			formContent += ui.MutedStyle.Render("   " + field + ":\n")
		}
	}
	s += ui.FormPanelStyle.Render(formContent) + "\n"

	// Live state suggestions
	if m.state.ActiveField == 3 && len(m.state.StateSuggestions) > 0 {
		s += "\n" + ui.InfoStyle.Render("Suggestions: ")
		for _, state := range m.state.StateSuggestions {
			s += ui.DimStyle.Render(state.Name + " (" + state.Abbreviation + ")  ")
		}
		s += "\n"
	}

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
	}
	if m.state.ActiveField == 4 {
		s += "\n" + ui.WarningStyle.Render("Email is optional — press Enter to skip") + "\n"
	}
	s += "\n" + ui.Footer("Enter confirm · Esc back · ctrl+c quit")
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
	s := ui.BannerStyle.Render(banner()) + "\n"
	s += ui.TitleStyle.Render("SIGN IN") + "\n"
	s += ui.DimStyle.Render("Enter your username to continue") + "\n\n"

	formContent := ui.SelectedStyle.Render(" ▸ Username: ") + m.state.InputBuffer + "▊\n"
	s += ui.FormPanelStyle.Render(formContent) + "\n"

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
	}
	s += "\n" + ui.Footer("Enter sign in · Esc back · ctrl+c quit")
	return s
}

// renderAdmin renders the admin dashboard landing page
func renderAdmin(m model) string {
	options := []string{
		"🛡️  Manage Players",
		"⚔️  Play Game",
		"✦  Quit",
	}
	s := ui.BannerStyle.Render(banner()) + "\n"
	s += ui.TitleStyle.Render("ADMIN DASHBOARD") + "\n"
	s += ui.DimStyle.Render("Logged in as: "+m.state.Player.FirstName+" ("+m.state.Player.Role+")") + "\n\n"
	s += ui.Divider() + "\n\n"

	for i, option := range options {
		if m.state.Cursor == i {
			s += ui.SelectedStyle.Render(" ▸ "+option) + "\n"
		} else {
			s += ui.MutedStyle.Render("   "+option) + "\n"
		}
	}

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError)
	}
	s += "\n" + ui.Footer("↑/↓ navigate · Enter select · ctrl+c quit")
	return s
}

// renderAdminPlayers renders the scrollable list of players in a table format
func renderAdminPlayers(m model) string {
	s := ui.TitleStyle.Render("PLAYERS LIST") + "\n"
	s += ui.DimStyle.Render(fmt.Sprintf("Total registered: %d", len(m.state.AdminPlayers))) + "\n\n"

	// Table headers - fits in 42 columns
	s += ui.HeaderStyle.Render(fmt.Sprintf("   %-10s %-14s %-6s %-10s", "Username", "Name", "Record", "Role")) + "\n"
	s += ui.Divider() + "\n"

	if len(m.state.AdminPlayers) == 0 {
		s += ui.MutedStyle.Render("   No players registered yet.") + "\n"
	} else {
		maxVisible := 5
		start := 0
		total := len(m.state.AdminPlayers)
		if total > maxVisible {
			start = m.state.AdminSelectedIndex - maxVisible/2
			if start < 0 {
				start = 0
			}
			if start+maxVisible > total {
				start = total - maxVisible
			}
		}
		end := start + maxVisible
		if end > total {
			end = total
		}

		for i := start; i < end; i++ {
			p := m.state.AdminPlayers[i]
			fullName := p.FirstName + " " + p.LastName
			if len(fullName) > 14 {
				fullName = fullName[:11] + "..."
			}
			record := fmt.Sprintf("%d-%d", p.Wins, p.Losses)
			line := fmt.Sprintf("   %-10s %-14s %-6s %-10s", p.Username, fullName, record, p.Role)

			if m.state.AdminSelectedIndex == i {
				s += ui.SelectedStyle.Render(" ▸ "+line[3:]) + "\n"
			} else {
				s += ui.MutedStyle.Render("   "+line[3:]) + "\n"
			}
		}
		if total > maxVisible {
			s += ui.DimStyle.Render(fmt.Sprintf("   ... (%d more players) ...", total-maxVisible)) + "\n"
		}
	}

	s += ui.Divider() + "\n"
	if m.state.FormError != "" {
		s += ui.ErrorLine(m.state.FormError) + "\n"
	}
	s += ui.Footer("↑/↓ scroll · Enter view · Esc dashboard")
	return s
}

// renderAdminPlayerDetail renders details of a selected player and action shortcuts
func renderAdminPlayerDetail(m model) string {
	if m.state.AdminSelectedIndex >= len(m.state.AdminPlayers) {
		return "No player selected"
	}
	p := m.state.AdminPlayers[m.state.AdminSelectedIndex]

	s := ui.TitleStyle.Render("PLAYER PROFILE") + "\n"
	s += ui.DimStyle.Render("Review details and perform actions") + "\n\n"

	// Read-only notification
	if p.IsAdmin() && !m.state.Player.IsRootAdmin() && p.Username != m.state.Player.Username {
		s += ui.WarningStyle.Render("🔒 Read-Only: Standard admin cannot modify other admins") + "\n\n"
	}

	// Profile card layout
	var profile string
	profile += fmt.Sprintf("Username:  %s\n", ui.InfoStyle.Render(p.Username))
	profile += fmt.Sprintf("Full Name: %s %s\n", p.FirstName, p.LastName)
	profile += fmt.Sprintf("State:     %s\n", p.State)
	emailStr := "N/A"
	if p.Email != nil {
		emailStr = *p.Email
	}
	profile += fmt.Sprintf("Email:     %s\n", emailStr)
	profile += fmt.Sprintf("Role:      %s\n", ui.WarningStyle.Render(p.Role))
	profile += fmt.Sprintf("Lifetime:  W:%d  L:%d  T:%d  Total:%d\n", p.Wins, p.Losses, p.Ties, p.TotalMatches)

	s += ui.FormPanelStyle.Render(profile) + "\n\n"

	if m.state.AdminConfirm != "" {
		var confirmMsg string
		switch m.state.AdminConfirm {
		case "reset":
			confirmMsg = "⚠️  Are you sure you want to RESET stats to 0?"
		case "delete":
			confirmMsg = "⚠️  Are you sure you want to DELETE this account?"
		case "role":
			targetRole := "admin"
			if p.Role == "admin" {
				targetRole = "player"
			}
			confirmMsg = fmt.Sprintf("⚠️  Are you sure you want to change role to %s?", targetRole)
		}
		s += ui.WarningStyle.Render(confirmMsg) + "\n"
		s += ui.Footer("Enter confirm · Esc cancel") + "\n"
	} else {
		var actions []string
		actions = append(actions, "[R] Reset Stats")
		actions = append(actions, "[D] Delete Player")
		if m.state.Player.IsRootAdmin() {
			actions = append(actions, "[P] Toggle Role")
		}

		s += ui.DimStyle.Render("Actions:") + "\n"
		s += " " + strings.Join(actions, "  ·  ") + "\n"
		s += "\n" + ui.Footer("Esc back to list")
	}

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError)
	}
	return s
}

// renderMenu draws the main game menu with a personalised greeting
func renderMenu(m model) string {
	options := []string{
		"⚔  Single Player",
		"🌐  Multiplayer",
		"✦  Quit",
	}
	s := ui.TitleStyle.Render("ROPA-SCI") + "\n"
	s += ui.DimStyle.Render("Welcome back, "+m.state.Player.FirstName+"!") + "\n"

	// Stats bar
	stats := fmt.Sprintf("W: %d  L: %d  Matches: %d",
		m.state.Player.Wins, m.state.Player.Losses, m.state.Player.TotalMatches)
	s += ui.InfoStyle.Render(stats) + "\n\n"
	s += ui.Divider() + "\n\n"

	for i, option := range options {
		if m.state.Cursor == i {
			s += ui.SelectedStyle.Render(" ▸ "+option) + "\n"
		} else {
			s += ui.MutedStyle.Render("   "+option) + "\n"
		}
	}
	s += "\n" + ui.Footer("↑/↓ navigate · Enter select · ? help")
	return s
}

// ─── Game Screen ──────────────────────────────────────────────────────────────

// MoveCard returns a Lipgloss-styled card string for a given move.
// Selected cards get an accented double border, normal cards get a rounded muted border.
func MoveCard(move models.Move, selected bool) string {
	style := ui.NormalCardStyle
	if selected {
		style = ui.SelectedCardStyle
	}

	type cardInfo struct {
		emoji   string
		content string
	}

	cards := map[models.Move]cardInfo{
		models.Rock:     {"🪨", "R O C K\n\n[ 1 / R ]"},
		models.Paper:    {"📄", "P A P E R\n\n[ 2 / P ]"},
		models.Scissors: {"✂️", "S C I S S\n\n[ 3 / S ]"},
		models.None:     {"🔒", "? ? ? ? ?\n\n▓ ▓ ▓ ▓ ▓"},
	}

	info := cards[move]
	card := style.Render(info.content)

	return "     " + info.emoji + "\n" + card
}

// renderGame routes to the correct phase renderer based on current game phase
func renderGame(m model) string {
	opponent := "AI"
	if m.state.GameMode == "multi" && m.netOpponentName != "" {
		opponent = m.netOpponentName
	}

	// Score header — styled persistent banner
	scoreText := fmt.Sprintf("Round %d of 3  ·  You: %d  ·  %s: %d",
		m.state.Score.Round,
		m.state.Score.PlayerWins,
		opponent,
		m.state.Score.OpponentWins,
	)
	s := ui.ScoreHeaderStyle.Render(scoreText) + "\n\n"

	switch m.state.Phase {
	case models.PhasePick:
		s += renderPick(m)
	case models.PhaseThink:
		s += renderThink(m)
	case models.PhaseReveal:
		s += renderReveal(m)
	case models.PhaseResult:
		s += renderResult(m)
	}
	return s
}

// renderPick shows three selectable move cards side by side using lipgloss.JoinHorizontal
func renderPick(m model) string {
	s := ui.HeaderStyle.Render("Choose your move") + "\n\n"
	moves := []models.Move{models.Rock, models.Paper, models.Scissors}
	cards := make([]string, 3)
	for i, move := range moves {
		cards[i] = MoveCard(move, m.state.Cursor == i)
	}
	s += lipgloss.JoinHorizontal(lipgloss.Top, cards[0], " ", cards[1], " ", cards[2]) + "\n"
	s += "\n" + ui.Footer("← → choose · Enter / 1-3 / R-P-S confirm")
	return s
}

// renderThink shows the player's locked-in move alongside a face-down opponent card
// The spinner animates to build tension while the opponent "decides"
func renderThink(m model) string {
	opponent := "AI"
	if m.state.GameMode == "multi" && m.netOpponentName != "" {
		opponent = m.netOpponentName
	}
	playerCol := ui.DimStyle.Render("YOUR MOVE") + "\n\n" + MoveCard(m.state.PlayerMove, false)
	vsCol := ui.HeaderStyle.Render("\n\n\n  VS  ")
	aiCol := ui.DimStyle.Render(strings.ToUpper(opponent)+"'s MOVE") + "\n\n" + MoveCard(models.None, false)

	s := lipgloss.JoinHorizontal(lipgloss.Center, playerCol, vsCol, aiCol) + "\n"
	spinner := spinnerFrames[m.state.SpinnerFrame]
	s += "\n" + ui.DimStyle.Render(spinner+"  "+opponent+" is calculating...") + "\n"
	return s
}

// renderReveal shows both moves side by side before the result appears
func renderReveal(m model) string {
	opponent := "AI"
	if m.state.GameMode == "multi" && m.netOpponentName != "" {
		opponent = m.netOpponentName
	}
	playerCol := ui.DimStyle.Render("YOUR MOVE") + "\n\n" + MoveCard(m.state.PlayerMove, false)
	vsCol := ui.HeaderStyle.Render("\n\n\n  VS  ")
	aiCol := ui.DimStyle.Render(strings.ToUpper(opponent)+"'s MOVE") + "\n\n" + MoveCard(m.state.AIMove, false)

	s := lipgloss.JoinHorizontal(lipgloss.Center, playerCol, vsCol, aiCol) + "\n"
	return s
}

// renderResult shows the round outcome, score bar, and contextual next-step prompt
func renderResult(m model) string {
	opponent := "AI"
	if m.state.GameMode == "multi" && m.netOpponentName != "" {
		opponent = m.netOpponentName
	}
	s := ""
	var boxContent string

	switch m.state.RoundOutcome {
	case "win":
		s += ui.SuccessStyle.Render("🏆  YOU WIN THIS ROUND!  🏆") + "\n\n"
		boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
		s += ui.WinBoxStyle.Render(boxContent) + "\n"
	case "lose":
		s += ui.DangerStyle.Render(fmt.Sprintf("💀  %s WINS THIS ROUND  💀", strings.ToUpper(opponent))) + "\n\n"
		boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
		s += ui.LoseBoxStyle.Render(boxContent) + "\n"
	case "tie":
		s += ui.WarningStyle.Render("🤝  DRAW!  🤝") + "\n\n"
		boxContent = outcomeMessage(m.state.PlayerMove, m.state.AIMove)
		s += ui.TieBoxStyle.Render(boxContent) + "\n"
	}

	s += "\n" + scoreBar(m.state.Score.PlayerWins, m.state.Score.OpponentWins, opponent) + "\n"

	if m.state.Score.PlayerWins == 2 {
		s += "\n" + ui.SuccessStyle.Render("🎉  YOU WIN THE MATCH!  🎉") + "\n"
		s += "\n" + ui.Footer("Enter replay · Esc menu")
	} else if m.state.Score.OpponentWins == 2 {
		s += "\n" + ui.DangerStyle.Render(fmt.Sprintf("💀  %s wins the match.", opponent)) + "\n"
		s += "\n" + ui.Footer("Enter replay · Esc menu")
	} else {
		switch m.state.RoundOutcome {
		case "win":
			s += "\n" + ui.Footer("Enter next round · Esc menu")
		case "lose":
			s += "\n" + ui.Footer("Enter fight back · Esc menu")
		case "tie":
			s += "\n" + ui.Footer("Enter break tie · Esc menu")
		}
	}
	return s
}

// ─── Render Utilities ─────────────────────────────────────────────────────────

// scoreBar renders a visual two-pip progress bar for the current match score
// e.g.  You: ●○  AI: ○○
func scoreBar(playerWins, aiWins int, opponent string) string {
	bar := ui.DimStyle.Render("You: ")
	for i := 0; i < 2; i++ {
		bar += ui.ScorePip(i < playerWins)
	}
	bar += ui.DimStyle.Render("  " + opponent + ": ")
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
		"🏠  Create Room — share a code",
		"🔗  Join Room  — connect to host",
		"←  Back",
	}
	s := ui.TitleStyle.Render("MULTIPLAYER") + "\n"
	s += ui.DimStyle.Render("Choose how you want to play") + "\n\n"
	s += ui.Divider() + "\n\n"

	for i, opt := range options {
		if m.state.Cursor == i {
			s += ui.SelectedStyle.Render(" ▸ "+opt) + "\n"
		} else {
			s += ui.MutedStyle.Render("   "+opt) + "\n"
		}
	}

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
	}
	s += "\n" + ui.Footer("↑/↓ navigate · Enter select · Esc menu")
	return s
}

// renderCreateRoom draws the room code waiting screen
func renderCreateRoom(m model) string {
	spinner := spinnerFrames[m.state.SpinnerFrame]
	s := ui.TitleStyle.Render("CREATE ROOM") + "\n\n"
	roomContent := ui.DimStyle.Render("Your room code:\n\n") +
		ui.SuccessStyle.Render(m.state.RoomCode)
	s += ui.RoomCodeBoxStyle.Render(roomContent) + "\n\n"
	s += ui.DimStyle.Render("Share this code with your opponent.") + "\n\n"
	s += ui.DimStyle.Render(spinner+"  Waiting for opponent to join...") + "\n"
	s += "\n" + ui.Footer("Esc cancel · ctrl+c quit")
	return s
}

// renderQuickMatch draws the auto matchmaking waiting screen
func renderQuickMatch(m model) string {
	spinner := spinnerFrames[m.state.SpinnerFrame]
	s := ui.TitleStyle.Render("QUICK MATCH") + "\n\n"
	s += ui.DimStyle.Render(spinner+"  Searching for an opponent...") + "\n\n"
	waitContent := ui.DimStyle.Render("This may take a moment.\nStay in the terminal!")
	s += ui.WaitingBoxStyle.Render(waitContent) + "\n"
	s += "\n" + ui.Footer("Esc cancel · ctrl+c quit")
	return s
}

// ─── Help Screen ──────────────────────────────────────────────────────────────

// keybindLine formats a single keybind row: blue key + grey description
func keybindLine(key, desc string) string {
	padded := key + strings.Repeat(" ", 14-len(key))
	if len(key) >= 14 {
		padded = key + "  "
	}
	return ui.InfoStyle.Render(padded) + ui.DimStyle.Render(desc)
}

// renderHelp draws a styled keybind reference card
func renderHelp(m model) string {
	options := []string{
		"⌨️  Keyboard Layout",
		"📜  Privacy & Data Policy",
	}

	s := ui.TitleStyle.Render("HELP & POLICIES") + "\n"
	s += ui.DimStyle.Render("Select a section to view details") + "\n\n"

	// Tab header selector
	var tabs []string
	for i, opt := range options {
		if m.state.Cursor == i {
			tabs = append(tabs, ui.SelectedStyle.Render(" ▸ "+opt))
		} else {
			tabs = append(tabs, ui.MutedStyle.Render("   "+opt))
		}
	}
	s += strings.Join(tabs, "  ") + "\n\n"
	s += ui.Divider() + "\n\n"

	var content string
	if m.state.Cursor == 0 {
		content += ui.SelectedStyle.Render("NAVIGATION") + "\n"
		content += keybindLine("↑/↓ or k/j", "Move cursor / scroll lists") + "\n"
		content += keybindLine("←/→ or h/l", "Select cards / switch tabs") + "\n"
		content += keybindLine("Enter", "Confirm selection / field") + "\n"
		content += keybindLine("Esc", "Go back / cancel") + "\n\n"

		content += ui.SelectedStyle.Render("GAME SHORTCUTS") + "\n"
		content += keybindLine("1  or  R", "Choose Rock") + "\n"
		content += keybindLine("2  or  P", "Choose Paper") + "\n"
		content += keybindLine("3  or  S", "Choose Scissors") + "\n\n"

		content += ui.SelectedStyle.Render("ADMIN PANEL KEYS") + "\n"
		content += keybindLine("R", "Reset player stats (on profile)") + "\n"
		content += keybindLine("D", "Delete player account (on profile)") + "\n"
		content += keybindLine("P", "Promote/demote player role")
	} else {
		content += ui.SelectedStyle.Render("PRIVACY & DATA PROTECTION") + "\n\n"
		content += ui.DimStyle.Render("1. Data Storage:") + "\n"
		content += "   All player accounts and match records are\n"
		content += "   stored locally in 'data/players.json'.\n"
		content += "   No data is transmitted to external servers.\n\n"
		content += ui.DimStyle.Render("2. LAN P2P Multiplayer:") + "\n"
		content += "   Rooms communicate directly over local network\n"
		content += "   connections. No chat logs or personal details\n"
		content += "   are shared beside usernames & moves.\n\n"
		content += ui.DimStyle.Render("3. Rights & Security:") + "\n"
		content += "   You can request resetting your stats or\n"
		content += "   deleting your account at any time in-app."
	}

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorAccent)).
		Padding(1, 2).
		Width(48).
		Render(content)

	s += helpBox + "\n"
	s += "\n" + ui.Footer("←/→ switch tabs · Esc go back")
	return s
}

// ─── Entry Point ──────────────────────────────────────────────────────────────

func main() {
	// Initialize structured logging
	cleanup, err := models.InitLogger()
	if err != nil {
		fmt.Println("Warning: Could not initialize logger:", err)
	} else {
		defer cleanup()
	}

	// Graceful panic recovery to restore terminal state
	defer func() {
		if r := recover(); r != nil {
			// Exit alternate screen and show cursor using standard ANSI escape codes
			fmt.Print("\x1b[?1049l\x1b[?25h")
			slog.Error("CRITICAL SYSTEM PANIC", "error", r)
			fmt.Println("\n\x1b[31;1m=== ROPA-SCI CRASH DETECTED ===\x1b[0m")
			fmt.Printf("Oops! The application encountered an unexpected error: %v\n", r)
			fmt.Println("Terminal state has been restored safely. Please check logs/app.log for details.")
			os.Exit(1)
		}
	}()

	slog.Info("Starting Ropa-Sci Bubbletea TUI application")
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),       // full-screen mode for a cleaner game feel
		tea.WithMouseCellMotion(), // mouse infrastructure ready for Week 7
	)
	if _, err := p.Run(); err != nil {
		slog.Error("Bubbletea runtime error", "error", err)
		fmt.Println("Fatal Error:", err)
		os.Exit(1)
	}
}

// renderDifficulty renders the single-player difficulty selection screen
func renderDifficulty(m model) string {
	options := []string{
		"😊  Easy   — Rando-tron AI",
		"🤔  Medium — Cycle-bot AI",
		"🔥  Hard   — Predictor AI",
		"←  Back",
	}
	s := ui.TitleStyle.Render("AI DIFFICULTY") + "\n"
	s += ui.DimStyle.Render("Choose your opponent's skill level") + "\n\n"
	s += ui.Divider() + "\n\n"

	for i, opt := range options {
		if m.state.Cursor == i {
			s += ui.SelectedStyle.Render(" ▸ " + opt) + "\n"
		} else {
			s += ui.MutedStyle.Render("   " + opt) + "\n"
		}
	}
	s += "\n" + ui.Footer("↑/↓ navigate · Enter select · Esc cancel")
	return s
}

// renderJoinRoom renders the LAN room address input form
func renderJoinRoom(m model) string {
	s := ui.TitleStyle.Render("JOIN ROOM") + "\n"
	s += ui.DimStyle.Render("Enter Host IP:Port (e.g. 192.168.1.15:8080)") + "\n\n"

	formContent := ui.SelectedStyle.Render(" ▸ Host: ") + ui.SuccessStyle.Render(m.state.InputBuffer) + "▊\n"
	s += ui.FormPanelStyle.Render(formContent) + "\n"

	if m.state.FormError != "" {
		s += "\n" + ui.ErrorLine(m.state.FormError) + "\n"
	}
	s += "\n" + ui.Footer("Enter connect · Esc cancel")
	return s
}

// getLocalIP queries network interfaces to find the host's LAN IPv4 address
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
