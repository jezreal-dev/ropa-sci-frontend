package server

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
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
	port     int
	mu       sync.Mutex
	active   bool

	// Player states
	hostConn     net.Conn
	guestConn    net.Conn
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

// Start opens a TCP listener on a random port (or the specified default port)
func (s *LANGameServer) Start(requestedPort int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addr := fmt.Sprintf("0.0.0.0:%d", requestedPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// If requested port is busy, fallback to dynamic port assignment
		listener, err = net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			return 0, fmt.Errorf("failed to bind TCP listener: %w", err)
		}
	}

	s.listener = listener
	s.port = listener.Addr().(*net.TCPAddr).Port
	s.active = true

	slog.Info("LAN P2P Server started successfully", "port", s.port)

	// Run connection handler loop in background
	go s.listenLoop()

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
	if s.listener != nil {
		_ = s.listener.Close()
	}

	if s.hostConn != nil {
		_ = s.hostConn.Close()
	}
	if s.guestConn != nil {
		_ = s.guestConn.Close()
	}

	slog.Info("LAN P2P Server stopped")
}

func (s *LANGameServer) listenLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			active := s.active
			s.mu.Unlock()
			if !active {
				return // Closed normally
			}
			slog.Error("Server socket accept error", "error", err)
			continue
		}

		s.mu.Lock()
		// First connection is the host client itself
		if s.hostConn == nil {
			s.hostConn = conn
			s.mu.Unlock()
			go s.handleConnection(conn, true)
		} else if s.guestConn == nil {
			// Second connection is the guest client joining
			s.guestConn = conn
			s.mu.Unlock()
			go s.handleConnection(conn, false)
		} else {
			// Reject third wheels
			s.mu.Unlock()
			slog.Warn("Rejecting third-wheel connection request")
			_ = conn.Close()
		}
	}
}

// handleConnection handles HTTP WS upgrade handshakes and reads frame packets
func (s *LANGameServer) handleConnection(conn net.Conn, isHost bool) {
	defer conn.Close()

	// 1. Perform WebSocket Upgrade Handshake
	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		slog.Error("Failed to read WebSocket upgrade request", "isHost", isHost, "error", err)
		return
	}

	if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" {
		slog.Warn("Received non-websocket upgrade HTTP connection")
		return
	}

	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		slog.Warn("Sec-WebSocket-Key missing from request header")
		return
	}

	// Calculate Accept Key (RFC 6455)
	const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	acceptHash := sha1.Sum([]byte(key + wsGUID))
	acceptKey := base64.StdEncoding.EncodeToString(acceptHash[:])

	// Send Handshake Response
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"

	_, err = conn.Write([]byte(resp))
	if err != nil {
		slog.Error("Failed to write handshake response", "isHost", isHost, "error", err)
		return
	}

	slog.Info("Connection upgraded to WebSocket successfully", "isHost", isHost)

	// 2. Read WebSocket Frame Loop
	for {
		payload, err := ReadWSFrame(conn)
		if err != nil {
			if err == io.EOF {
				slog.Info("Client connection closed cleanly (EOF)", "isHost", isHost)
			} else {
				slog.Error("Error reading WebSocket frame", "isHost", isHost, "error", err)
			}
			s.handleDisconnect(isHost)
			return
		}

		var msg WSMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			slog.Error("Failed to parse JSON WS message", "isHost", isHost, "error", err)
			continue
		}

		s.handleMessage(msg, isHost)
	}
}

// ReadWSFrame reads and decodes a single WebSocket frame (RFC 6455)
func ReadWSFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}

	// Opcode analysis
	opcode := header[0] & 0x0F
	if opcode == 8 { // Close frame
		return nil, io.EOF
	}

	mask := (header[1] & 0x80) != 0
	payloadLen := int64(header[1] & 0x7F)

	if payloadLen == 126 {
		lenBytes := make([]byte, 2)
		if _, err := io.ReadFull(conn, lenBytes); err != nil {
			return nil, err
		}
		payloadLen = int64(binary.BigEndian.Uint16(lenBytes))
	} else if payloadLen == 127 {
		lenBytes := make([]byte, 8)
		if _, err := io.ReadFull(conn, lenBytes); err != nil {
			return nil, err
		}
		rawLen := binary.BigEndian.Uint64(lenBytes)
		if rawLen > math.MaxInt64 {
			return nil, fmt.Errorf("websocket frame payload too large: %d", rawLen)
		}
		payloadLen = int64(rawLen) // #nosec G115 - overflow guarded above
	}

	var maskKey []byte
	if mask {
		maskKey = make([]byte, 4)
		if _, err := io.ReadFull(conn, maskKey); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	if mask {
		for i := 0; i < len(payload); i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	return payload, nil
}

// WriteWSFrame writes an unmasked text WebSocket frame to a client
func WriteWSFrame(conn net.Conn, payload []byte) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	var header []byte
	header = append(header, 0x81) // FIN + Text frame type

	length := len(payload)
	if length <= 125 {
		header = append(header, byte(length))
	} else if length <= 65535 {
		header = append(header, 126)
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		header = append(header, lenBytes...)
	} else {
		header = append(header, 127)
		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, uint64(length))
		header = append(header, lenBytes...)
	}

	// Write header then body
	if _, err := conn.Write(header); err != nil {
		return err
	}
	if _, err := conn.Write(payload); err != nil {
		return err
	}
	return nil
}

func (s *LANGameServer) sendToClient(conn net.Conn, msgType, payload string) {
	msg := WSMessage{Type: msgType, Payload: payload}
	data, _ := json.Marshal(msg)
	_ = WriteWSFrame(conn, data)
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
