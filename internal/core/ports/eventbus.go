package ports

import "github.com/sales-system/go-real-time/internal/core/domain"

// EventBus represents the publish-subscribe real-time event dispatcher.
type EventBus interface {
	Publish(event domain.ChangeEvent)
	PublishBatch(batch domain.BatchChangeEvent)
	Subscribe(handler func(msg domain.WSMessage)) (unsubscribe func())
	Broadcast(topic string, eventType domain.EventType, payload interface{})
}
