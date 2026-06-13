package main

import (
	// "fmt"
	"os"
	"strings"
)

// --- Data & ASCII Helpers ---

func asciiColor(text string, substring string, color string) strings.Builder {
	// tview color tags
	reset := "[-]"
	tag := "[white]"

	switch color {
	case "red":
		tag = "[red]"
	case "green":
		tag = "[green]"
	case "yellow":
		tag = "[yellow]"
	case "blue":
		tag = "[blue]"
	case "magenta":
		tag = "[magenta]"
	case "cyan":
		tag = "[cyan]"
	}

	data, err := os.ReadFile("standard.txt")
	if err != nil {
		var b strings.Builder
		b.WriteString("Error: standard.txt not found")
		return b
	}

	lines := strings.Split(string(data), "\n")
	words := strings.Split(text, "\\n")
	var outputBuilder strings.Builder

	for _, word := range words {
		if word == "" {
			outputBuilder.WriteString("\n")
			continue
		}
		// ASCII art characters are 8 lines tall
		for i := 0; i < 8; i++ {
			for j := 0; j < len(word); j++ {
				char := word[j]
				index := 9*int(char-32) + 1
				if index >= 0 && index+i < len(lines) {
					asciiLine := lines[index+i]
					// Use tview color tags instead of ANSI escape codes
					outputBuilder.WriteString(tag + asciiLine + reset)
				}
			}
			outputBuilder.WriteString("\n")
		}
	}
	return outputBuilder
}
