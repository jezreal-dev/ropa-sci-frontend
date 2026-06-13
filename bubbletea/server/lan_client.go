package server

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// DialLANServer establishes a WebSocket connection to the host address
func DialLANServer(addr string) (*websocket.Conn, error) {
	wsURL := fmt.Sprintf("ws://%s/ws", addr)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host %s: %w", addr, err)
	}

	return conn, nil
}

// WriteClientMessage writes a JSON WebSocket message to the connection
func WriteClientMessage(conn *websocket.Conn, msg WSMessage) error {
	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to write WS message: %w", err)
	}

	return nil
}
