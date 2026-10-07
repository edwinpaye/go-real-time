package websocket

import (
	"sync"

	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
	"github.com/sales-system/go-real-time/internal/infra/tracker"
)

// Hub manages active WebSocket client connections and broadcasts messages.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan domain.WSMessage
	eventBus   ports.EventBus
	tracker    *tracker.Tracker
	mu         sync.RWMutex
}

// NewHub creates and starts the WebSocket connection Hub.
func NewHub(eventBus ports.EventBus) *Hub {
	hub := &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan domain.WSMessage, 1024),
		eventBus:   eventBus,
		tracker:    tracker.GetTracker(),
	}

	// Wire EventBus subscription to Hub broadcast
	eventBus.Subscribe(func(msg domain.WSMessage) {
		hub.Broadcast(msg)
	})

	go hub.run()
	return hub
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.tracker.IncActiveConnections()

			logger.Info("WebSocket client connected", logger.Fields{
				"user_id": client.userID,
				"ip":      client.ip,
			})

			// Welcome message
			client.SendMessage(domain.WSMessage{
				Topic: "system",
				Event: domain.EventSystemBroadcast,
				Payload: map[string]interface{}{
					"message": "Connected to real-time sales feed",
					"user_id": client.userID,
				},
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.tracker.DecActiveConnections()
				logger.Info("WebSocket client disconnected", logger.Fields{
					"user_id": client.userID,
				})
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				client.SendMessage(message)
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast queues a message for all active clients.
func (h *Hub) Broadcast(msg domain.WSMessage) {
	select {
	case h.broadcast <- msg:
	default:
		logger.Warn("Hub broadcast queue full; dropped message", logger.Fields{
			"topic": msg.Topic,
			"event": msg.Event,
		})
	}
}

// ClientCount returns the number of active connections.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
