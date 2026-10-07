package http

import (
	"net/http"
	"strings"

	"github.com/sales-system/go-real-time/config"
	wsAdapter "github.com/sales-system/go-real-time/internal/adapters/primary/websocket"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

// RouterConfig contains dependencies for setting up the main HTTP multiplexer.
type RouterConfig struct {
	AuthHandler    *AuthHandler
	ProductHandler *ProductHandler
	OrderHandler   *OrderHandler
	AuditHandler   *AuditHandler
	WSHandler      *wsAdapter.Handler
	TokenService   ports.TokenService
	Config         *config.Config
}

// NewRouter constructs the enterprise HTTP router with middleware chains.
func NewRouter(rc RouterConfig) http.Handler {
	mux := http.NewServeMux()

	authRequired := AuthMiddleware(rc.TokenService)

	// Public Auth Endpoints
	mux.HandleFunc("/api/v1/auth/register", rc.AuthHandler.Register)
	mux.HandleFunc("/api/v1/auth/login", rc.AuthHandler.Login)

	// Protected Auth Me
	mux.Handle("/api/v1/auth/me", authRequired(http.HandlerFunc(rc.AuthHandler.Profile)))

	// Product Catalog Routes
	mux.Handle("/api/v1/products", authRequired(http.HandlerFunc(rc.ProductHandler.HandleProducts)))
	mux.Handle("/api/v1/products/", authRequired(http.HandlerFunc(rc.ProductHandler.HandleProductByID)))

	// Sales Order Routes
	mux.Handle("/api/v1/orders", authRequired(http.HandlerFunc(rc.OrderHandler.HandleOrders)))
	mux.Handle("/api/v1/orders/", authRequired(http.HandlerFunc(rc.OrderHandler.HandleOrderByID)))

	// Audit & Telemetry
	mux.Handle("/api/v1/audit", authRequired(http.HandlerFunc(rc.AuditHandler.ListLogs)))
	mux.Handle("/api/v1/metrics", authRequired(http.HandlerFunc(rc.AuditHandler.Metrics)))

	// WebSocket Real-time Endpoint
	mux.Handle("/ws", rc.WSHandler)

	// Static UI Subpath hosting (e.g. /app/ -> ./web)
	staticServer := NewStaticServer(rc.Config.StaticPrefix, rc.Config.StaticPath)
	mux.Handle(rc.Config.StaticPrefix, staticServer)

	// Also serve root redirect / fallback to /app/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, rc.Config.StaticPrefix, http.StatusTemporaryRedirect)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/ws") {
			staticServer.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// Wrap root handler with Global Middlewares: CORS + Logging/Metrics
	var rootHandler http.Handler = mux
	rootHandler = CORSMiddleware(rootHandler)
	rootHandler = LoggingMiddleware(rootHandler)

	return rootHandler
}
