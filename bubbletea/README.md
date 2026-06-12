# Ropa-Sci Bubbletea TUI Engine

This subdirectory contains the Bubbletea terminal user interface (TUI) implementation of **Ropa-Sci** — a modern, cyber-neon styled Rock-Paper-Scissors game with game-theory AI, local multiplayer, and administration tools.

## Tech Stack
- **Go** (Golang)
- **Bubble Tea** (TUI framework)
- **Lip Gloss** (Terminal styling and layout)
- **Gorilla WebSocket** (Local P2P Multiplayer connection)

---

## Directory Structure

- `cmd/main.go`: Application entry point, central update event loop, and screen renderers.
- `models/`: Code relating to game logic, players database, validation, AI models, and logging.
  - `ai_engine.go`: Game-theory based predictor using Markov Chain algorithms.
  - `logger.go`: Configured structured log system outputting to `logs/app.log`.
  - `player.go`: Definitions for player structures, game phases, and state layers.
  - `storage.go`: Local JSON-based persistent database with thread-safe mutex locks.
- `server/`: WebSocket service code facilitating local network peer-to-peer multiplayer.
- `ui/`: CSS-like Lipgloss styles, neon colors definitions, and layout configurations.
- `data/players.json`: Plain-text file storing local player accounts and lifetime statistics.

---

## Features

### 1. Cyber-Neon Layout & UI
Overhauled in 2026 using responsive double-bordered card styling, interactive lists, and `lipgloss.Place` vertical/horizontal centering for a premium, console-grade desktop feel.

### 2. Local P2P Multiplayer
Allows players on the same local area network (LAN) to host or join match rooms. Connections are brokered over WebSockets with zero dependency on third-party cloud infrastructure.

### 3. Smart Predictor AI
Includes a predictive Markov Chain AI engine that analyzes player move histories during matches to predict future selections, making the single-player experience highly engaging.

### 4. Admin Dashboard
Privileged accounts (e.g. `jmomoh`) route to the Admin Panel upon signing in. Admins can view scrollable lists of registered accounts, toggle roles, reset stats, and delete player data.

---

## Getting Started

### Prerequisites
Make sure you have Go installed on your machine (v1.20+ recommended).

### Running the App
From the `bubbletea` directory, execute:
```bash
go run cmd/main.go
```

### Running Tests
Execute unit tests for database and validation logic:
```bash
go test ./...
```
