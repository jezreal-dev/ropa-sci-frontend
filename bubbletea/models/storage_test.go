package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStorageOperations(t *testing.T) {
	// Setup temp players file
	tmpDir, err := os.MkdirTemp("", "ropa-sci-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldDataFile := dataFile
	dataFile = filepath.Join(tmpDir, "players.json")
	defer func() { dataFile = oldDataFile }()

	// Test 1: Save player
	player := Player{
		FirstName: "Test",
		LastName:  "User",
		Username:  "testuser",
		State:     "Lagos",
		Role:      "player",
	}

	err = SavePlayer(player)
	if err != nil {
		t.Fatalf("SavePlayer failed: %v", err)
	}

	// Save duplicate should fail
	err = SavePlayer(player)
	if err == nil {
		t.Fatal("expected duplicate SavePlayer to fail, but it succeeded")
	}

	// Test 2: Find player
	foundPlayer, found, err := FindPlayerByUsername("testuser")
	if err != nil {
		t.Fatalf("FindPlayerByUsername failed: %v", err)
	}
	if !found {
		t.Fatal("expected player to be found")
	}
	if foundPlayer.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", foundPlayer.Username)
	}

	// Test 3: Set role
	err = SetPlayerRole("testuser", "admin")
	if err != nil {
		t.Fatalf("SetPlayerRole failed: %v", err)
	}

	foundPlayer, found, err = FindPlayerByUsername("testuser")
	if err != nil {
		t.Fatalf("FindPlayerByUsername after SetPlayerRole failed: %v", err)
	}
	if !found {
		t.Fatal("expected player to be found")
	}
	if foundPlayer.Role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", foundPlayer.Role)
	}
	if !foundPlayer.IsAdmin() {
		t.Error("expected IsAdmin() to be true")
	}

	// Test 4: Reset stats
	foundPlayer.Wins = 5
	foundPlayer.Losses = 2
	err = UpdatePlayer(foundPlayer)
	if err != nil {
		t.Fatalf("UpdatePlayer failed: %v", err)
	}

	err = ResetPlayerStats("testuser")
	if err != nil {
		t.Fatalf("ResetPlayerStats failed: %v", err)
	}

	foundPlayer, found, err = FindPlayerByUsername("testuser")
	if err != nil {
		t.Fatalf("FindPlayerByUsername after ResetPlayerStats failed: %v", err)
	}
	if foundPlayer.Wins != 0 || foundPlayer.Losses != 0 {
		t.Errorf("expected stats to be reset to 0, got wins=%d, losses=%d", foundPlayer.Wins, foundPlayer.Losses)
	}

	// Test 5: Delete player
	err = DeletePlayer("testuser")
	if err != nil {
		t.Fatalf("DeletePlayer failed: %v", err)
	}

	_, found, err = FindPlayerByUsername("testuser")
	if err != nil {
		t.Fatalf("FindPlayerByUsername after DeletePlayer failed: %v", err)
	}
	if found {
		t.Fatal("expected player to not be found after deletion")
	}
}
