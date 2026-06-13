package server

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

// DialLANServer establishes a TCP connection to the host address and performs WebSocket handshake
func DialLANServer(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to host %s: %w", addr, err)
	}

	// Generate a random 16-byte base64 key
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to generate random secret: %w", err)
	}
	secKey := base64.StdEncoding.EncodeToString(keyBytes)

	// Send WebSocket upgrade HTTP request
	req := fmt.Sprintf("GET /ws HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Upgrade: websocket\r\n"+
		"Connection: Upgrade\r\n"+
		"Sec-WebSocket-Key: %s\r\n"+
		"Sec-WebSocket-Version: 13\r\n\r\n", addr, secKey)

	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send upgrade request: %w", err)
	}

	// Read upgrade response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read upgrade response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("websocket upgrade rejected: status code %d", resp.StatusCode)
	}

	return conn, nil
}

// WriteClientMessage writes a masked text WebSocket frame to the connection
func WriteClientMessage(conn net.Conn, msg WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal WS message: %w", err)
	}

	var header []byte
	header = append(header, 0x81) // FIN + Text frame type

	length := len(data)
	if length <= 125 {
		header = append(header, byte(length)|0x80) // Set Mask bit (0x80)
	} else if length <= 65535 {
		header = append(header, 126|0x80)
		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		header = append(header, lenBytes...)
	} else {
		header = append(header, 127|0x80)
		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, uint64(length))
		header = append(header, lenBytes...)
	}

	// Generate 4-byte random masking key (required for client-to-server frames)
	maskKey := make([]byte, 4)
	if _, err := rand.Read(maskKey); err != nil {
		return fmt.Errorf("failed to generate masking key: %w", err)
	}
	header = append(header, maskKey...)

	// Mask payload
	maskedPayload := make([]byte, length)
	for i := 0; i < length; i++ {
		maskedPayload[i] = data[i] ^ maskKey[i%4]
	}

	// Write header then masked payload
	if _, err := conn.Write(header); err != nil {
		return fmt.Errorf("failed to write frame header: %w", err)
	}
	if _, err := conn.Write(maskedPayload); err != nil {
		return fmt.Errorf("failed to write frame payload: %w", err)
	}

	return nil
}
