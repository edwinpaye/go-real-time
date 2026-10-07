package eventbus

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
	"github.com/sales-system/go-real-time/internal/infra/tracker"
)

type subscription struct {
	id      string
	handler func(msg domain.WSMessage)
}

// InMemoryEventBus is an in-process, non-blocking pub/sub event bus with buffer queues.
type InMemoryEventBus struct {
	mu            sync.RWMutex
	subscriptions map[string]*subscription
	eventChan     chan domain.WSMessage
	tracker       *tracker.Tracker
	closed        bool
}

// NewInMemoryEventBus creates and runs the real-time event bus.
func NewInMemoryEventBus(bufferSize int) ports.EventBus {
	if bufferSize <= 0 {
		bufferSize = 2048
	}

	bus := &InMemoryEventBus{
		subscriptions: make(map[string]*subscription),
		eventChan:     make(chan domain.WSMessage, bufferSize),
		tracker:       tracker.GetTracker(),
	}

	go bus.dispatchLoop()
	return bus
}

func (b *InMemoryEventBus) dispatchLoop() {
	for msg := range b.eventChan {
		b.mu.RLock()
		subs := make([]*subscription, 0, len(b.subscriptions))
		for _, sub := range b.subscriptions {
			subs = append(subs, sub)
		}
		b.mu.RUnlock()

		for _, sub := range subs {
			// Execute handler safely
			func(s *subscription, m domain.WSMessage) {
				defer func() {
					if r := recover(); r != nil {
						logger.Error("Panic in eventbus subscriber", logger.Fields{
							"sub_id": s.id,
							"panic":  r,
						})
					}
				}()
				s.handler(m)
			}(sub, msg)
		}
	}
}

// Publish queues a single resource change event for instant broadcast.
func (b *InMemoryEventBus) Publish(event domain.ChangeEvent) {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	msg := domain.WSMessage{
		Topic:     string(event.Resource),
		Event:     event.Type,
		Payload:   event,
		Timestamp: event.Timestamp.UnixMilli(),
	}

	b.tracker.IncEventsEmitted(1)

	select {
	case b.eventChan <- msg:
	default:
		logger.Warn("EventBus buffer full; event dropped or delayed", logger.Fields{
			"event_id": event.EventID,
			"type":     event.Type,
		})
	}
}

// PublishBatch publishes an organized transaction of multiple changes made at the same time.
func (b *InMemoryEventBus) PublishBatch(batch domain.BatchChangeEvent) {
	if batch.BatchID == "" {
		batch.BatchID = uuid.New().String()
	}
	if batch.Timestamp.IsZero() {
		batch.Timestamp = time.Now().UTC()
	}
	batch.Type = domain.EventBatchTransaction
	batch.EventsCount = len(batch.Changes)

	msg := domain.WSMessage{
		Topic:     "batch",
		Event:     domain.EventBatchTransaction,
		Payload:   batch,
		Timestamp: batch.Timestamp.UnixMilli(),
	}

	b.tracker.IncEventsEmitted(uint64(len(batch.Changes)))

	select {
	case b.eventChan <- msg:
	default:
		logger.Warn("EventBus buffer full; batch event dropped", logger.Fields{
			"batch_id": batch.BatchID,
			"count":    batch.EventsCount,
		})
	}
}

// Broadcast sends generic broadcast messages to a topic.
func (b *InMemoryEventBus) Broadcast(topic string, eventType domain.EventType, payload interface{}) {
	msg := domain.WSMessage{
		Topic:     topic,
		Event:     eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC().UnixMilli(),
	}

	b.tracker.IncEventsEmitted(1)

	select {
	case b.eventChan <- msg:
	default:
		logger.Warn("EventBus buffer full; broadcast dropped", logger.Fields{
			"topic": topic,
			"event": eventType,
		})
	}
}

// Subscribe registers a listener to receive all events.
func (b *InMemoryEventBus) Subscribe(handler func(msg domain.WSMessage)) func() {
	b.mu.Lock()
	subID := uuid.New().String()
	sub := &subscription{
		id:      subID,
		handler: handler,
	}
	b.subscriptions[subID] = sub
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		delete(b.subscriptions, subID)
		b.mu.Unlock()
	}
}
