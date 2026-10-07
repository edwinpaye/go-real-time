package domain

import "time"

// AuditAction defines the operation recorded in audit logs.
type AuditAction string

const (
	AuditActionCreate AuditAction = "CREATE"
	AuditActionUpdate AuditAction = "UPDATE"
	AuditActionDelete AuditAction = "DELETE"
	AuditActionBatch  AuditAction = "BATCH_TRANSACTION"
)

// AuditLog tracks enterprise state mutations with security context.
type AuditLog struct {
	ID         string      `json:"id"`
	EntityName string      `json:"entity_name"`
	EntityID   string      `json:"entity_id"`
	Action     AuditAction `json:"action"`
	ActorID    string      `json:"actor_id"`
	ActorEmail string      `json:"actor_email"`
	OldValues  string      `json:"old_values,omitempty"`
	NewValues  string      `json:"new_values,omitempty"`
	IPAddress  string      `json:"ip_address"`
	UserAgent  string      `json:"user_agent"`
	CreatedAt  time.Time   `json:"created_at"`
}
