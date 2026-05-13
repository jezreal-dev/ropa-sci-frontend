package models

// Move represents a player's choice in a round
type Move string

const (
	Rock	Move = "rock"
	Paper	Move = "paper"
	Scissors	Move = "scissors"
	None	Move = ""  // simply no move has been made
)

// RoundResult this stores what happened in a single round
type RoundResult struct {
	PlayerMove	Move
	OpponentMove	Move
	Outcome	string // the final decision of the match goes here e.g; "win", "lose", "tie"
}

// Player holds everything we know about one player 
type Player struct {
	// Registration fields
	FirstName	string
	LastName	string
	Username	string
	State		string
	Email		*string

	// Lifetime Game stats
	Wins	int
	Losses	int
	Ties	int
	TotalMatches int
}

// MatchScore tracks the best-of-three score for current match
type MatchScore struct {
	PlayerWins	int
	OpponentWins	int
	Round	int // which round we'r on e.g; 1, 2, 3.
	RoundHistory []RoundResult // rounds in THIS match only
}

// GameState is the top-level state your TUI will carry everywhere
type GameState struct {
	Player	Player
	Score	MatchScore
	Screen	string // Screens: "welcome", "register", "login", "menu", "game", "result", "waiting"
	GameMode	string // handles for "single" & "multi"
	
	// Navigation
	Cursor      int     // tracks which menu option is highlighted
	ActiveField int // This tracks which field is active during registration
	
	// Form handling
	FormError	string
	InputBuffer	string
	StateSuggestions []State
}