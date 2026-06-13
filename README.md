# 🎮 Ropa-Sci

> A modern, multiplayer Rock-Paper-Scissors CLI gaming platform built in Go.

Ropa-Sci is designed as a collaborative, group-built terminal game featuring beautiful console-grade visuals, smart game-theory opponent predictive models, peer-to-peer LAN multiplayer rooms, and role-based admin controls.

---

## 🛠️ Tech Stack & Key Technologies
- **Language:** Go (Golang)
- **TUI Frameworks:** [Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), and `tview`
- **Networking:** P2P Local WebSockets
- **Architecture:** Thread-safe JSON state store

---

## 📂 Project Architecture

The project is split into two primary CLI client engines:

- **[`bubbletea/`](./bubbletea/README.md):** Features a customized floating-frame cyber-neon interface, Braille animations, predictive game-theory Markov Chain AI, local network multiplayer matchmaking, and a multi-level role-based admin dashboard.
- **`tview/`:** Contains alternative form inputs and navigation panels.

---

## 🚀 How to Run

### Play Immediately (No Installation)
You do not need to install Go or build the code to play!
1. Go to the **[Releases](https://github.com/jezreal-dev/ropa-sci-frontend/releases)** page on GitHub.
2. Download the correct executable file for your machine:
   - **Windows:** Download `ropa-sci-windows-amd64.exe`
   - **Mac (M1/M2/M3 chips):** Download `ropa-sci-macos-arm64`
   - **Mac (Older Intel chips):** Download `ropa-sci-macos-amd64`
   - **Linux:** Download `ropa-sci-linux-amd64`
3. Double-click the file to open the terminal game instantly! (On Mac/Linux, you may need to right-click and select "Open", or run `chmod +x` in the terminal first).

### Run from Source (For Developers)
Navigate to the `bubbletea` folder and run:
```bash
cd bubbletea
go run cmd/main.go
```

### Run Unit Tests
To run structural tests across models, state validation, and storage:
```bash
cd bubbletea
go test ./...
```

---

## 🛡️ License, Contribution & Security
- **License:** Distributed under our custom collaborative [LICENSE](./LICENSE). The code is open to view and you may contribute pull requests, but modifications may not be distributed separately or claimed under different ownership to preserve the rights of all group contributors.
- **Security Policy:** Review our vulnerability reporting guidelines and data handling rules in the [SECURITY.md](./SECURITY.md) document.
- **Developer Walkthrough:** For details on execution flow, MVU state machine, and subsystems, check out [bubbletea/walkthrough.md](./bubbletea/walkthrough.md).