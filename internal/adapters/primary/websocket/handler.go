package websocket

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow enterprise origin validation; default allows all for demo
		return true
	},
}

// Handler handles incoming WebSocket upgrade requests.
type Handler struct {
	hub          *Hub
	tokenService ports.TokenService
	userRepo     ports.UserRepository
}

// NewHandler instantiates the websocket handler.
func NewHandler(hub *Hub, tokenService ports.TokenService, userRepo ports.UserRepository) *Handler {
	return &Handler{
		hub:          hub,
		tokenService: tokenService,
		userRepo:     userRepo,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract token from query parameter or Authorization header
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if tokenStr == "" {
		http.Error(w, `{"error":"missing authentication token"}`, http.StatusUnauthorized)
		return
	}

	claims, err := h.tokenService.ValidateToken(tokenStr)
	if err != nil {
		logger.Warn("WebSocket upgrade rejected: invalid token", logger.Fields{
			"ip":    r.RemoteAddr,
			"error": err.Error(),
		})
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade WebSocket connection", logger.Fields{
			"error": err.Error(),
		})
		return
	}

	client := NewClient(h.hub, conn, claims.UserID, claims.Email, string(claims.Role), r.RemoteAddr)
	h.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
