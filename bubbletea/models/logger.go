package models

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var logFile *os.File

// InitLogger initializes a structured slog JSON logger writing to logs/app.log.
// It creates the logs/ directory if it doesn't already exist.
// Returns a function that closes the log file upon termination.
func InitLogger() (func(), error) {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	logPath := filepath.Join(logDir, "app.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logFile = file

	// Setup slog handler
	// In production, writing JSON is standard for structured log analysis.
	handler := slog.NewJSONHandler(io.MultiWriter(logFile), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("Structured logger initialized successfully")

	cleanup := func() {
		slog.Info("Structured logger shutting down")
		if logFile != nil {
			logFile.Close()
		}
	}
	return cleanup, nil
}
