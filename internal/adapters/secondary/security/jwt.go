package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"golang.org/x/crypto/bcrypt"
)

type customClaims struct {
	UserID string          `json:"user_id"`
	Email  string          `json:"email"`
	Role   domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// JWTService implements ports.TokenService
type JWTService struct {
	secretKey []byte
}

// NewJWTService instantiates JWT manager.
func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secretKey: []byte(secret),
	}
}

func (s *JWTService) GenerateToken(user *domain.User, ttl time.Duration) (string, error) {
	claims := customClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
			Issuer:    "sales-system-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

func (s *JWTService) ValidateToken(tokenString string) (*ports.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*customClaims); ok && token.Valid {
		return &ports.JWTClaims{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}, nil
	}

	return nil, errors.New("invalid token claims")
}

// BcryptHasher implements ports.PasswordHasher
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a hasher.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (b *BcryptHasher) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), b.cost)
	return string(bytes), err
}

func (b *BcryptHasher) ComparePassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
