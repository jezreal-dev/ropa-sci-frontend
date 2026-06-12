package models

import "unicode"

// MaxInputLength is the maximum characters allowed in any single input field.
// Prevents runaway input from consuming memory or breaking the UI layout.
const MaxInputLength = 50

// ─── Per-Character Validators ─────────────────────────────────────────────────
// These run on every keystroke to reject invalid characters BEFORE they enter
// the input buffer. This gives instant feedback — the character simply doesn't
// appear — which is better UX than showing an error after the user hits Enter.

// IsValidNameChar checks if a rune is allowed in a first/last name.
// Allows letters (any script), hyphens, and apostrophes for names like O'Brien or Jean-Pierre.
func IsValidNameChar(r rune) bool {
	return unicode.IsLetter(r) || r == '-' || r == '\''
}

// IsValidUsernameChar checks if a rune is allowed in a username.
// Restricted to lowercase ASCII letters, digits, underscores, and hyphens.
func IsValidUsernameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
}

// IsValidStateChar checks if a rune is allowed in the state input field.
// Allows letters and spaces (for multi-word states like "Federal Capital Territory").
func IsValidStateChar(r rune) bool {
	return unicode.IsLetter(r) || r == ' '
}

// IsValidEmailChar checks if a rune is allowed in an email address.
func IsValidEmailChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) ||
		r == '@' || r == '.' || r == '-' || r == '_' || r == '+'
}

// ─── Field-Level Validators ──────────────────────────────────────────────────
// These run on Enter to validate the complete field value.
// They return an error message string, or "" if the value is valid.

// ValidateName validates a first or last name field.
func ValidateName(name string) string {
	if len(name) < 2 {
		return "Must be at least 2 characters"
	}
	if len(name) > 30 {
		return "Must be 30 characters or less"
	}
	for _, r := range name {
		if !IsValidNameChar(r) {
			return "Only letters, hyphens, and apostrophes allowed"
		}
	}
	return ""
}

// ValidateUsername validates a username field.
func ValidateUsername(username string) string {
	if len(username) < 3 {
		return "Username must be at least 3 characters"
	}
	if len(username) > 20 {
		return "Username must be 20 characters or less"
	}
	for _, r := range username {
		if !IsValidUsernameChar(r) {
			return "Only lowercase letters, numbers, underscores, and hyphens"
		}
	}
	return ""
}
