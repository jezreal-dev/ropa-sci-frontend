package main

import (
	"os"
	"testing"
)

func TestRunApp(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping interactive TUI run in CI")
	}
	main()
}
