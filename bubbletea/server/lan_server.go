package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage represents the structured JSON packets sent over the P2P connection
type WSMessage struct {
	Type    string `json:"type"`    // e.g. "join", "start", "move", "round", "match", "error"
	Payload string `json:"payload"` // e.g. Username, Move, JSON outcome data
}

// RoundOutcomePayload carries details of a finished round
type RoundOutcomePayload struct {
	YourMove     string `json:"your_move"`
	OpponentMove string `json:"opponent_move"`
	Outcome      string `json:"outcome"` // "win", "lose", "tie"
	YourWins     int    `json:"your_wins"`
	OpponentWins int    `json:"opponent_wins"`
}

// LANGameServer coordinates a single P2P multiplayer match
type LANGameServer struct {
	listener net.Listener
	server   *http.Server
	port     int
	mu       sync.Mutex
	active   bool

	// Player states
	hostConn     *websocket.Conn
	guestConn    *websocket.Conn
	hostName     string
	guestName    string
	hostMove     string
	guestMove    string
	hostWins     int
	guestWins    int
	roundHistory []string
}

// NewLANServer initializes a LAN game server
func NewLANServer() *LANGameServer {
	return &LANGameServer{}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *LANGameServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket upgrade failed", "error", err)
		return
	}

	s.mu.Lock()
	if s.hostConn == nil {
		s.hostConn = conn
		s.mu.Unlock()
		go s.handleConnection(conn, true)
	} else if s.guestConn == nil {
		s.guestConn = conn
		s.mu.Unlock()
		go s.handleConnection(conn, false)
	} else {
		s.mu.Unlock()
		slog.Warn("Rejecting third-wheel connection request")
		_ = conn.Close()
	}
}

// Start opens a TCP listener on a random port (or the specified default port)
func (s *LANGameServer) Start(requestedPort int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := fmt.Sprintf(":%d", requestedPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// If requested port is busy, fallback to dynamic port assignment
		listener, err = net.Listen("tcp", ":0") // #nosec G102 - Intentional bind to all interfaces for LAN multiplayer hosting
		if err != nil {
			return 0, fmt.Errorf("failed to bind TCP listener: %w", err)
		}
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	s.active = true

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.wsHandler)
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second, // Mitigate Slowloris attacks (G112)
	}

	slog.Info("LAN P2P Server started successfully", "port", s.port)

	// Run connection handler loop in background
	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			slog.Error("Server serve error", "error", err)
		}
	}()

	return s.port, nil
}

// Stop closes the server listener and active client connections safely
func (s *LANGameServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	if s.server != nil {
		_ = s.server.Close()
	}

	if s.hostConn != nil {
		_ = s.hostConn.Close()
	}
	if s.guestConn != nil {
		_ = s.guestConn.Close()
	}

	slog.Info("LAN P2P Server stopped")
}

// handleConnection handles HTTP WS upgrade handshakes and reads frame packets
func (s *LANGameServer) handleConnection(conn *websocket.Conn, isHost bool) {
	defer conn.Close()

	slog.Info("Connection upgraded to WebSocket successfully", "isHost", isHost)

	// Read WebSocket Frame Loop
	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("Error reading WebSocket frame", "isHost", isHost, "error", err)
			} else {
				slog.Info("Client connection closed cleanly (EOF)", "isHost", isHost)
			}
			s.handleDisconnect(isHost)
			return
		}

		s.handleMessage(msg, isHost)
	}
}

func (s *LANGameServer) sendToClient(conn *websocket.Conn, msgType, payload string) {
	if conn == nil {
		return
	}
	msg := WSMessage{Type: msgType, Payload: payload}
	_ = conn.WriteJSON(msg)
}

func (s *LANGameServer) handleMessage(msg WSMessage, isHost bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch msg.Type {
	case "join":
		if isHost {
			s.hostName = msg.Payload
			slog.Info("Host player registered name", "name", s.hostName)
		} else {
			s.guestName = msg.Payload
			slog.Info("Guest player registered name", "name", s.guestName)
			// Trigger game match start since both players are now fully registered
			s.sendToClient(s.hostConn, "start", s.guestName)
			s.sendToClient(s.guestConn, "start", s.hostName)
			slog.Info("P2P Match started", "host", s.hostName, "guest", s.guestName)
		}

	case "move":
		if isHost {
			s.hostMove = msg.Payload
			slog.Info("Host move registered", "move", s.hostMove)
		} else {
			s.guestMove = msg.Payload
			slog.Info("Guest move registered", "move", s.guestMove)
		}

		// Check if both moves are ready to evaluate round outcome
		if s.hostMove != "" && s.guestMove != "" {
			s.evaluateRound()
		}
	}
}

func (s *LANGameServer) evaluateRound() {
	outcomeHost := "tie"
	outcomeGuest := "tie"

	if s.hostMove != s.guestMove {
		if (s.hostMove == "rock" && s.guestMove == "scissors") ||
			(s.hostMove == "paper" && s.guestMove == "rock") ||
			(s.hostMove == "scissors" && s.guestMove == "paper") {
			outcomeHost = "win"
			outcomeGuest = "lose"
			s.hostWins++
		} else {
			outcomeHost = "lose"
			outcomeGuest = "win"
			s.guestWins++
		}
	}

	hostPayload, _ := json.Marshal(RoundOutcomePayload{
		YourMove:     s.hostMove,
		OpponentMove: s.guestMove,
		Outcome:      outcomeHost,
		YourWins:     s.hostWins,
		OpponentWins: s.guestWins,
	})

	guestPayload, _ := json.Marshal(RoundOutcomePayload{
		YourMove:     s.guestMove,
		OpponentMove: s.hostMove,
		Outcome:      outcomeGuest,
		YourWins:     s.guestWins,
		OpponentWins: s.hostWins,
	})

	// Broadcast round details
	s.sendToClient(s.hostConn, "round", string(hostPayload))
	s.sendToClient(s.guestConn, "round", string(guestPayload))

	slog.Info("Round evaluated", "hostMove", s.hostMove, "guestMove", s.guestMove, "hostWins", s.hostWins, "guestWins", s.guestWins)

	// Clear moves for next round
	s.hostMove = ""
	s.guestMove = ""

	// Check if match is finished (best of 3: first to 2 wins)
	if s.hostWins == 2 {
		s.sendToClient(s.hostConn, "match", "win")
		s.sendToClient(s.guestConn, "match", "lose")
		slog.Info("Match finished: Host wins", "host", s.hostName, "guest", s.guestName)
	} else if s.guestWins == 2 {
		s.sendToClient(s.hostConn, "match", "lose")
		s.sendToClient(s.guestConn, "match", "win")
		slog.Info("Match finished: Guest wins", "host", s.hostName, "guest", s.guestName)
	}
}

func (s *LANGameServer) handleDisconnect(isHost bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Warn("Player disconnected", "isHost", isHost)

	// If one player leaves, inform the other player and shut down the session
	if isHost {
		if s.guestConn != nil {
			s.sendToClient(s.guestConn, "error", "Opponent disconnected from match.")
		}
	} else {
		if s.hostConn != nil {
			s.sendToClient(s.hostConn, "error", "Opponent disconnected from match.")
		}
	}
}
