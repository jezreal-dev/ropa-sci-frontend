package models

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var dataFile = "data/players.json"

// dbMutex protects all concurrent file operations on players.json.
// RWMutex allows parallel reads, but exclusive writes.
var dbMutex sync.RWMutex

// loadPlayersUnlocked reads all players from disk without acquiring a lock.
// This is an internal helper used inside public locked operations.
func loadPlayersUnlocked() ([]Player, error) {
	cleanedPath := filepath.Clean(dataFile)
	data, err := os.ReadFile(cleanedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Player{}, nil // File doesn't exist yet - fine for bootstrap
		}
		return nil, err
	}

	var players []Player
	err = json.Unmarshal(data, &players)
	return players, err
}

// SavePlayer writes a new player to disk thread-safely
// Returns an error if the username already exists
func SavePlayer(p Player) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	// Create data/ folder first if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("could not create data folder: %w", err)
	}

	// Load existing players
	players, err := loadPlayersUnlocked()
	if err != nil {
		return fmt.Errorf("failed to load players: %w", err)
	}

	// Check for duplicate username before appending
	for _, existing := range players {
		if strings.EqualFold(existing.Username, p.Username) {
			return fmt.Errorf("username '%s' already exists", p.Username)
		}
	}

	// Add the new player
	players = append(players, p)

	// Write back to file with readable formatting
	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(dataFile), data, 0600)
}

// LoadPlayers reads all players from disk thread-safely
func LoadPlayers() ([]Player, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return loadPlayersUnlocked()
}

// UsernameExists checks if a username is already registered thread-safely
func UsernameExists(username string) bool {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return false
	}
	for _, p := range players {
		if strings.EqualFold(p.Username, username) {
			return true
		}
	}
	return false
}

// FindPlayerByUsername returns a player if they exist thread-safely
// Returns the player, a found bool, and any error
func FindPlayerByUsername(username string) (Player, bool, error) {
	dbMutex.RLock()
	defer dbMutex.RUnlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return Player{}, false, err
	}
	for _, p := range players {
		if strings.EqualFold(p.Username, username) {
			return p, true, nil
		}
	}
	return Player{}, false, nil
}

// UpdatePlayer finds an existing player by username and overwrites their record thread-safely.
// Used to save lifetime stats after every match ends.
func UpdatePlayer(p Player) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return fmt.Errorf("could not load players: %w", err)
	}

	found := false
	for i, existing := range players {
		if strings.EqualFold(existing.Username, p.Username) {
			players[i] = p // overwrite with updated data
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("player '%s' not found", p.Username)
	}

	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(dataFile), data, 0600)
}

// GenerateRoomCode creates a random 4-digit room code
func GenerateRoomCode() string {
	digits := rand.Intn(9000) + 1000 // always 4 digits: 1000-9999
	return fmt.Sprintf("RPS-%d", digits)
}

// DeletePlayer removes a player from the JSON file thread-safely
func DeletePlayer(username string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return fmt.Errorf("could not load players: %w", err)
	}

	found := false
	var updated []Player
	for _, existing := range players {
		if strings.EqualFold(existing.Username, username) {
			found = true
			continue
		}
		updated = append(updated, existing)
	}

	if !found {
		return fmt.Errorf("player '%s' not found", username)
	}

	data, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(dataFile), data, 0600)
}

// ResetPlayerStats zeroes W/L/T/TotalMatches for a player thread-safely
func ResetPlayerStats(username string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return fmt.Errorf("could not load players: %w", err)
	}

	found := false
	for i, existing := range players {
		if strings.EqualFold(existing.Username, username) {
			players[i].Wins = 0
			players[i].Losses = 0
			players[i].Ties = 0
			players[i].TotalMatches = 0
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("player '%s' not found", username)
	}

	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(dataFile), data, 0644)
}

// SetPlayerRole changes the role of a player thread-safely
func SetPlayerRole(username, role string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	players, err := loadPlayersUnlocked()
	if err != nil {
		return fmt.Errorf("could not load players: %w", err)
	}

	found := false
	for i, existing := range players {
		if strings.EqualFold(existing.Username, username) {
			players[i].Role = role
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("player '%s' not found", username)
	}

	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Clean(dataFile), data, 0644)
}