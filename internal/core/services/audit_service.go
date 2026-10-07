package services

import (
	"context"

	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/logger"
	"github.com/sales-system/go-real-time/internal/infra/tracker"
	"github.com/sales-system/go-real-time/internal/infra/workerpool"
)

type auditService struct {
	auditRepo  ports.AuditRepository
	workerPool *workerpool.WorkerPool
	eventBus   ports.EventBus
	tracker    *tracker.Tracker
}

// NewAuditService instantiates asynchronous audit service.
func NewAuditService(
	auditRepo ports.AuditRepository,
	workerPool *workerpool.WorkerPool,
	eventBus ports.EventBus,
) ports.AuditService {
	return &auditService{
		auditRepo:  auditRepo,
		workerPool: workerPool,
		eventBus:   eventBus,
		tracker:    tracker.GetTracker(),
	}
}

func (s *auditService) RecordAsync(log *domain.AuditLog) {
	if log == nil {
		return
	}

	s.workerPool.Submit(func(ctx context.Context) error {
		if err := s.auditRepo.Create(ctx, log); err != nil {
			logger.Error("Failed to persist background audit log", logger.Fields{
				"entity": log.EntityName,
				"action": log.Action,
				"error":  err.Error(),
			})
			return err
		}

		s.tracker.IncAuditsLogged()

		// Also broadcast audit update to audit listeners
		s.eventBus.Publish(domain.ChangeEvent{
			Type:      domain.EventEntityCreated,
			Resource:  domain.ResourceAudit,
			EntityID:  log.ID,
			Data:      log,
			ActorID:   log.ActorID,
			Timestamp: log.CreatedAt,
		})

		return nil
	})
}

func (s *auditService) ListLogs(ctx context.Context, entityName string, limit, offset int) ([]domain.AuditLog, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.auditRepo.List(ctx, entityName, limit, offset)
}
