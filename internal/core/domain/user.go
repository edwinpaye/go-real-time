package domain

import (
	"errors"
	"strings"
	"time"
)

// UserRole defines system permissions.
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleCashier UserRole = "cashier"
)

// User represents a system user account.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	PasswordHash string    `json:"-"`
	Role         UserRole  `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Validate checks user domain invariants.
func (u *User) Validate() error {
	if strings.TrimSpace(u.Email) == "" || !strings.Contains(u.Email, "@") {
		return errors.New("valid email is required")
	}
	if strings.TrimSpace(u.FullName) == "" {
		return errors.New("full name is required")
	}
	if u.Role != RoleAdmin && u.Role != RoleManager && u.Role != RoleCashier {
		return errors.New("invalid user role")
	}
	return nil
}
