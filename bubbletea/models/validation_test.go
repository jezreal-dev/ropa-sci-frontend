package models

import (
	"testing"
)

func TestIsValidNameChar(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'-', true},
		{'\'', true},
		{'1', false},
		{'@', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsValidNameChar(tt.r); got != tt.want {
			t.Errorf("IsValidNameChar(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestIsValidUsernameChar(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'z', true},
		{'0', true},
		{'9', true},
		{'_', true},
		{'-', true},
		{'A', false}, // Username only allows lowercase ASCII letters
		{'@', false},
		{' ', false},
	}
	for _, tt := range tests {
		if got := IsValidUsernameChar(tt.r); got != tt.want {
			t.Errorf("IsValidUsernameChar(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestIsValidStateChar(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{' ', true},
		{'-', false},
		{'1', false},
	}
	for _, tt := range tests {
		if got := IsValidStateChar(tt.r); got != tt.want {
			t.Errorf("IsValidStateChar(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestIsValidEmailChar(t *testing.T) {
	tests := []struct {
		r    rune
		want bool
	}{
		{'a', true},
		{'9', true},
		{'@', true},
		{'.', true},
		{'-', true},
		{'_', true},
		{'+', true},
		{' ', false},
		{'#', false},
	}
	for _, tt := range tests {
		if got := IsValidEmailChar(tt.r); got != tt.want {
			t.Errorf("IsValidEmailChar(%q) = %v, want %v", tt.r, got, tt.want)
		}
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Al", ""}, // Min boundary (2)
		{"Jean-Pierre", ""},
		{"O'Brien", ""},
		{"A", "Must be at least 2 characters"}, // Below min boundary
		{"", "Must be at least 2 characters"},
		{"abcdefghijklmnopqrstuvwxyzabcdef", "Must be 30 characters or less"}, // Above max boundary (32 chars)
		{"John123", "Only letters, hyphens, and apostrophes allowed"},
		{"John Doe", "Only letters, hyphens, and apostrophes allowed"}, // Space not allowed in names
	}
	for _, tt := range tests {
		if got := ValidateName(tt.name); got != tt.want {
			t.Errorf("ValidateName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		username string
		want     string
	}{
		{"bob", ""}, // Min boundary (3)
		{"john_doe-12", ""},
		{"bo", "Username must be at least 3 characters"}, // Below min boundary
		{"", "Username must be at least 3 characters"},
		{"abcdefghijklmnopqrstuv", "Username must be 20 characters or less"},      // Above max boundary (22 chars)
		{"John_doe", "Only lowercase letters, numbers, underscores, and hyphens"}, // Uppercase not allowed
		{"john doe", "Only lowercase letters, numbers, underscores, and hyphens"},
	}
	for _, tt := range tests {
		if got := ValidateUsername(tt.username); got != tt.want {
			t.Errorf("ValidateUsername(%q) = %q, want %q", tt.username, got, tt.want)
		}
	}
}
