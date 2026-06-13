<div align="center">
  <h1>🎮 Ropa-Sci v2.0</h1>
  <p><b>A modern, multiplayer Rock-Paper-Scissors TUI gaming platform built in Go.</b></p>

  <a href="https://github.com/jezreal-dev/ropa-sci-frontend/releases"><kbd>⤓ Download Latest Release</kbd></a>
  &nbsp;&nbsp;•&nbsp;&nbsp;
  <a href="./bubbletea/walkthrough.md"><kbd>📖 Read Developer Docs</kbd></a>
</div>

<br/>

Ropa-Sci is a collaborative, group-built terminal game featuring stunning console-grade visuals, smart game-theory opponent predictive models, peer-to-peer LAN multiplayer rooms, and role-based admin controls.

---

## ⚡ Tech Stack & Architecture

Built with a focus on speed, concurrency, and aesthetics:
- **Core Engine:** Go (Golang)
- **TUI Rendering:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) & [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **Networking:** P2P Local WebSockets via `gorilla/websocket`
- **Data Layer:** Thread-safe, Mutex-locked JSON state store

The project is split into modular client engines:
- **[`bubbletea/`](./bubbletea/README.md):** The flagship client. Features a floating-frame cyber-neon interface, Braille loading animations, a predictive Markov Chain AI, LAN matchmaking, and a multi-level role-based admin dashboard.
- **`tview/`:** A lightweight, alternative UI featuring classic form inputs and navigation panels.

---

## 🚀 How to Play

### No Installation Required
You do not need to install Go, compile code, or use a terminal wizard to play Ropa-Sci.
1. Head over to our <a href="https://github.com/jezreal-dev/ropa-sci-frontend/releases"><kbd>Releases Page</kbd></a>.
2. Download the executable file for your operating system (e.g., `ropa-sci-windows-amd64.exe` for Windows, or the `macos/linux` variants).
3. Double-click the file to launch the game instantly! *(Mac/Linux users may need to right-click and select "Open", or run `chmod +x` first).*

### Run from Source (Developers)
If you want to hack on the codebase or run it locally via Go:
```bash
git clone https://github.com/jezreal-dev/ropa-sci-frontend.git
cd ropa-sci-frontend/bubbletea
go run cmd/main.go
```

To run structural tests across models, state validation, and storage:
```bash
cd bubbletea
go test ./...
```

---

## 🤝 Credits & Contribution

- **Game Logic Architecture:** Massive shoutout to **Michael Adejoro**, the core visionary who originally brought out and designed the foundational game logic and mechanics before the frontend architecture was even built.
- **Development Team:** Built collaboratively by Jezreal Momoh and Ahmed Toyyib.

### License & Security
Distributed under our custom collaborative [LICENSE](./LICENSE). You may contribute pull requests, but modifications may not be distributed separately or claimed under different ownership. For vulnerability reporting and data handling rules, review our [SECURITY.md](./SECURITY.md).