package ports

import (
	"time"

	"github.com/sales-system/go-real-time/internal/core/domain"
)

// JWTClaims holds parsed user identity.
type JWTClaims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
}

// TokenService manages cryptographically secure JWT issuance and parsing.
type TokenService interface {
	GenerateToken(user *domain.User, ttl time.Duration) (string, error)
	ValidateToken(tokenString string) (*JWTClaims, error)
}

// PasswordHasher manages salt and one-way password hashing.
type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePassword(hashedPassword, password string) bool
}
