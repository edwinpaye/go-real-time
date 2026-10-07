package domain

import "time"

// EventType categorizes real-time change events.
type EventType string

const (
	EventEntityCreated     EventType = "ENTITY_CREATED"
	EventEntityUpdated     EventType = "ENTITY_UPDATED"
	EventEntityDeleted     EventType = "ENTITY_DELETED"
	EventStockChanged      EventType = "STOCK_CHANGED"
	EventBatchTransaction  EventType = "BATCH_TRANSACTION"
	EventSystemBroadcast   EventType = "SYSTEM_BROADCAST"
)

// ResourceType names the entity kind.
type ResourceType string

const (
	ResourceProduct ResourceType = "products"
	ResourceOrder   ResourceType = "orders"
	ResourceUser    ResourceType = "users"
	ResourceAudit   ResourceType = "audit"
)

// ChangeEvent represents an atomic change produced by a backend mutation.
type ChangeEvent struct {
	EventID       string                 `json:"event_id"`
	Type          EventType              `json:"type"`
	Resource      ResourceType           `json:"resource"`
	EntityID      string                 `json:"entity_id"`
	Data          interface{}            `json:"data,omitempty"`
	Delta         map[string]interface{} `json:"delta,omitempty"` // Specific changed fields for efficient client patch
	Timestamp     time.Time              `json:"timestamp"`
	ActorID       string                 `json:"actor_id,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// BatchChangeEvent organizes multiple atomic changes made in a single backend transaction/stored procedure.
type BatchChangeEvent struct {
	BatchID       string        `json:"batch_id"`
	Type          EventType     `json:"type"` // Always EventBatchTransaction
	Operation     string        `json:"operation"`
	EventsCount   int           `json:"events_count"`
	Changes       []ChangeEvent `json:"changes"`
	Timestamp     time.Time     `json:"timestamp"`
	ActorID       string        `json:"actor_id,omitempty"`
	CorrelationID string        `json:"correlation_id,omitempty"`
}

// WSMessage is the envelope sent over WebSockets to clients.
type WSMessage struct {
	Topic     string      `json:"topic"`
	Event     EventType   `json:"event"`
	Payload   interface{} `json:"payload"`
	Timestamp int64       `json:"timestamp"`
}
