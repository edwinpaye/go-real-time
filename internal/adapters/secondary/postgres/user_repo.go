package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

type userRepo struct {
	db *DB
}

// NewUserRepository instantiates postgres user repository.
func NewUserRepository(db *DB) ports.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, u *domain.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	query := `
		INSERT INTO users (id, email, full_name, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query, u.ID, u.Email, u.FullName, u.PasswordHash, u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("userRepo.Create failed: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, full_name, password_hash, role, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	var u domain.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID failed: %w", err)
	}
	return &u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, full_name, password_hash, role, is_active, created_at, updated_at
		FROM users WHERE LOWER(email) = LOWER($1)
	`
	var u domain.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail failed: %w", err)
	}
	return &u, nil
}

func (r *userRepo) List(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("userRepo.List count failed: %w", err)
	}

	query := `
		SELECT id, email, full_name, password_hash, role, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("userRepo.List query failed: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, nil
}

func (r *userRepo) Update(ctx context.Context, u *domain.User) error {
	u.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE users
		SET email = $1, full_name = $2, role = $3, is_active = $4, updated_at = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, u.Email, u.FullName, u.Role, u.IsActive, u.UpdatedAt, u.ID)
	if err != nil {
		return fmt.Errorf("userRepo.Update failed: %w", err)
	}
	return nil
}
