package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
	"github.com/sales-system/go-real-time/internal/infra/tracker"
)

type contextKey string

const (
	AuthContextKey contextKey = "authContext"
	RequestIDKey   contextKey = "requestID"
)

// LoggingMiddleware logs request metadata and records telemetry metrics.
func LoggingMiddleware(next http.Handler) http.Handler {
	tr := tracker.GetTracker()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		w.Header().Set("X-Request-ID", reqID)

		tr.IncRequests()
		tr.IncRouteHit(r.URL.Path)

		wrapped := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r.WithContext(ctx))

		duration := time.Since(start)

		if wrapped.statusCode >= 500 {
			tr.IncErrors()
			logger.Error("HTTP Request Server Error", logger.Fields{
				"request_id": reqID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     wrapped.statusCode,
				"duration":   duration.String(),
				"ip":         r.RemoteAddr,
			})
		} else {
			logger.Info("HTTP Request", logger.Fields{
				"request_id": reqID,
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     wrapped.statusCode,
				"duration":   duration.String(),
			})
		}
	})
}

// CORSMiddleware enables CORS headers for enterprise API consumers.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware validates JWT Bearer tokens and injects AuthContext into request context.
func AuthMiddleware(tokenService ports.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := tokenService.ValidateToken(tokenStr)
			if err != nil {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
				return
			}

			actCtx := ports.AuthContext{
				UserID:    claims.UserID,
				UserEmail: claims.Email,
				UserRole:  claims.Role,
				IPAddress: r.RemoteAddr,
				UserAgent: r.UserAgent(),
			}

			ctx := context.WithValue(r.Context(), AuthContextKey, actCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole guards endpoints by user role.
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actCtx, ok := r.Context().Value(AuthContextKey).(ports.AuthContext)
			if !ok {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			hasRole := false
			for _, role := range roles {
				if actCtx.UserRole == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				respondJSON(w, http.StatusForbidden, map[string]string{"error": "insufficient permissions"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper to extract AuthContext safely.
func GetAuthContext(r *http.Request) ports.AuthContext {
	if val, ok := r.Context().Value(AuthContextKey).(ports.AuthContext); ok {
		return val
	}
	return ports.AuthContext{
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
