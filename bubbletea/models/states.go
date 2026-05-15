package models

import "strings"

// State holds the full name and abbreviation of a Nigerian state
type State struct {
	Name         string
	Abbreviation string
}

// NigerianStates is the complete list of all 36 states + FCT
var NigerianStates = []State{
	{Name: "Abia", Abbreviation: "AB"},
	{Name: "Adamawa", Abbreviation: "AD"},
	{Name: "Akwa Ibom", Abbreviation: "AK"},
	{Name: "Anambra", Abbreviation: "AN"},
	{Name: "Bauchi", Abbreviation: "BA"},
	{Name: "Bayelsa", Abbreviation: "BY"},
	{Name: "Benue", Abbreviation: "BE"},
	{Name: "Borno", Abbreviation: "BO"},
	{Name: "Cross River", Abbreviation: "CR"},
	{Name: "Delta", Abbreviation: "DE"},
	{Name: "Ebonyi", Abbreviation: "EB"},
	{Name: "Edo", Abbreviation: "ED"},
	{Name: "Ekiti", Abbreviation: "EK"},
	{Name: "Enugu", Abbreviation: "EN"},
	{Name: "FCT (Abuja)", Abbreviation: "FC"},
	{Name: "Gombe", Abbreviation: "GO"},
	{Name: "Imo", Abbreviation: "IM"},
	{Name: "Jigawa", Abbreviation: "JI"},
	{Name: "Kaduna", Abbreviation: "KD"},
	{Name: "Kano", Abbreviation: "KN"},
	{Name: "Katsina", Abbreviation: "KT"},
	{Name: "Kebbi", Abbreviation: "KE"},
	{Name: "Kogi", Abbreviation: "KO"},
	{Name: "Kwara", Abbreviation: "KW"},
	{Name: "Lagos", Abbreviation: "LA"},
	{Name: "Nasarawa", Abbreviation: "NA"},
	{Name: "Niger", Abbreviation: "NI"},
	{Name: "Ogun", Abbreviation: "OG"},
	{Name: "Ondo", Abbreviation: "ON"},
	{Name: "Osun", Abbreviation: "OS"},
	{Name: "Oyo", Abbreviation: "OY"},
	{Name: "Plateau", Abbreviation: "PL"},
	{Name: "Rivers", Abbreviation: "RI"},
	{Name: "Sokoto", Abbreviation: "SO"},
	{Name: "Taraba", Abbreviation: "TA"},
	{Name: "Yobe", Abbreviation: "YO"},
	{Name: "Zamfara", Abbreviation: "ZA"},
}

// FindState looks up a state by full name, abbreviation, or partial match
// Returns the full State and true if found, empty State and false if not
func FindState(input string) (State, bool) {
	input = strings.TrimSpace(input)

	for _, s := range NigerianStates {
		// Match full name — case insensitive e.g. "lagos" matches "Lagos"
		if strings.EqualFold(s.Name, input) {
			return s, true
		}

		// Match abbreviation — case insensitive e.g. "kw" matches "KW"
		if strings.EqualFold(s.Abbreviation, input) {
			return s, true
		}
	}

	return State{}, false
}

// SuggestStates returns all states matching the input
// Matches from start of name, anywhere in name, or abbreviation
// e.g. "River" finds "Cross River", "K" finds all K-states
func SuggestStates(input string) []State {
	input = strings.TrimSpace(input)
	if input == "" {
		return []State{}
	}

	lower := strings.ToLower(input)
	var matches []State

	for _, s := range NigerianStates {
		nameLower := strings.ToLower(s.Name)
		abbrLower := strings.ToLower(s.Abbreviation)

		// Match name (from start or anywhere inside)
		if strings.HasPrefix(nameLower, lower) || strings.Contains(nameLower, lower) {
			matches = append(matches, s)
			continue // name matched — skip abbreviation check
		}

		// Match abbreviation
		if strings.HasPrefix(abbrLower, lower) {
			matches = append(matches, s)
		}
	}
	return matches
}

// IsValidState returns true if input exactly matches a state name or abbreviation
func IsValidState(input string) bool {
	_, found := FindState(input)
	return found
}
