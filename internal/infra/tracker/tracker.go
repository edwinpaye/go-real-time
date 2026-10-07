package tracker

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricSnapshot represents a snapshot of the runtime telemetry.
type MetricSnapshot struct {
	TotalRequests      uint64            `json:"total_requests"`
	ActiveConnections  int64             `json:"active_connections"`
	TotalEventsEmitted uint64            `json:"total_events_emitted"`
	TotalAuditsLogged  uint64            `json:"total_audits_logged"`
	UptimeSeconds      int64             `json:"uptime_seconds"`
	ErrorsCount        uint64            `json:"errors_count"`
	RouteHitCounts     map[string]uint64 `json:"route_hits"`
}

// Tracker provides atomic, low-overhead system metrics tracking.
type Tracker struct {
	startTime          time.Time
	totalRequests      atomic.Uint64
	activeConnections  atomic.Int64
	totalEventsEmitted atomic.Uint64
	totalAuditsLogged  atomic.Uint64
	errorsCount        atomic.Uint64

	mu         sync.RWMutex
	routeHits  map[string]uint64
}

var globalTracker *Tracker
var once sync.Once

// GetTracker returns the global singleton tracker.
func GetTracker() *Tracker {
	once.Do(func() {
		globalTracker = &Tracker{
			startTime: time.Now(),
			routeHits: make(map[string]uint64),
		}
	})
	return globalTracker
}

func (t *Tracker) IncRequests() {
	t.totalRequests.Add(1)
}

func (t *Tracker) IncRouteHit(path string) {
	t.mu.Lock()
	t.routeHits[path]++
	t.mu.Unlock()
}

func (t *Tracker) IncActiveConnections() {
	t.activeConnections.Add(1)
}

func (t *Tracker) DecActiveConnections() {
	t.activeConnections.Add(-1)
}

func (t *Tracker) IncEventsEmitted(count uint64) {
	t.totalEventsEmitted.Add(count)
}

func (t *Tracker) IncAuditsLogged() {
	t.totalAuditsLogged.Add(1)
}

func (t *Tracker) IncErrors() {
	t.errorsCount.Add(1)
}

func (t *Tracker) Snapshot() MetricSnapshot {
	t.mu.RLock()
	routesCopy := make(map[string]uint64, len(t.routeHits))
	for k, v := range t.routeHits {
		routesCopy[k] = v
	}
	t.mu.RUnlock()

	return MetricSnapshot{
		TotalRequests:      t.totalRequests.Load(),
		ActiveConnections:  t.activeConnections.Load(),
		TotalEventsEmitted: t.totalEventsEmitted.Load(),
		TotalAuditsLogged:  t.totalAuditsLogged.Load(),
		UptimeSeconds:      int64(time.Since(t.startTime).Seconds()),
		ErrorsCount:        t.errorsCount.Load(),
		RouteHitCounts:     routesCopy,
	}
}
