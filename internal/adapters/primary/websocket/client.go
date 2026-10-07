package websocket

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/infra/logger"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a connected WebSocket user connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	userEmail string
	userRole  string
	ip        string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, email, role, ip string) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, 256),
		userID:    userID,
		userEmail: email,
		userRole:  role,
		ip:        ip,
	}
}

// ReadPump handles incoming heartbeat, subscriptions or client messages.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("WebSocket unexpected close", logger.Fields{
					"user_id": c.userID,
					"error":   err.Error(),
				})
			}
			break
		}

		// Optional client ping or echo message handling
		var inbound map[string]interface{}
		if err := json.Unmarshal(message, &inbound); err == nil {
			if action, ok := inbound["action"].(string); ok && action == "ping" {
				c.SendMessage(domain.WSMessage{
					Topic:     "system",
					Event:     "PONG",
					Payload:   map[string]interface{}{"status": "alive", "server_time": time.Now().UTC().UnixMilli()},
					Timestamp: time.Now().UTC().UnixMilli(),
				})
			}
		}
	}
}

// WritePump pushes outbound messages and heartbeats to the client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Drain any queued messages into current frame for throughput
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// SendMessage writes a structured domain WSMessage to client's channel.
func (c *Client) SendMessage(msg domain.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
		logger.Warn("Client send buffer full; dropping message or evicting client", logger.Fields{
			"user_id": c.userID,
		})
	}
}
