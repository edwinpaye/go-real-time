package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

type auditRepo struct {
	db *DB
}

// NewAuditRepository instantiates audit repo.
func NewAuditRepository(db *DB) ports.AuditRepository {
	return &auditRepo{db: db}
}

func (r *auditRepo) Create(ctx context.Context, a *domain.AuditLog) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO audit_logs (id, entity_name, entity_id, action, actor_id, actor_email, old_values, new_values, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::jsonb, NULLIF($8, '')::jsonb, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.EntityName, a.EntityID, a.Action, a.ActorID, a.ActorEmail, a.OldValues, a.NewValues, a.IPAddress, a.UserAgent, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("auditRepo.Create failed: %w", err)
	}
	return nil
}

func (r *auditRepo) List(ctx context.Context, entityName string, limit, offset int) ([]domain.AuditLog, int, error) {
	var total int
	var countQuery string
	var countArgs []interface{}

	if entityName != "" {
		countQuery = `SELECT COUNT(*) FROM audit_logs WHERE entity_name = $1`
		countArgs = []interface{}{entityName}
	} else {
		countQuery = `SELECT COUNT(*) FROM audit_logs`
		countArgs = []interface{}{}
	}

	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("auditRepo.List count failed: %w", err)
	}

	var selectQuery string
	var args []interface{}

	if entityName != "" {
		selectQuery = `
			SELECT id, entity_name, entity_id, action, COALESCE(actor_id, ''), COALESCE(actor_email, ''),
			       COALESCE(old_values::text, ''), COALESCE(new_values::text, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at
			FROM audit_logs WHERE entity_name = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`
		args = []interface{}{entityName, limit, offset}
	} else {
		selectQuery = `
			SELECT id, entity_name, entity_id, action, COALESCE(actor_id, ''), COALESCE(actor_email, ''),
			       COALESCE(old_values::text, ''), COALESCE(new_values::text, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), created_at
			FROM audit_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("auditRepo.List query failed: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(&a.ID, &a.EntityName, &a.EntityID, &a.Action, &a.ActorID, &a.ActorEmail, &a.OldValues, &a.NewValues, &a.IPAddress, &a.UserAgent, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, a)
	}

	return logs, total, nil
}
