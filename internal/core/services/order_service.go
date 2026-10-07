package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

type orderService struct {
	txManager    ports.TransactionManager
	orderRepo    ports.OrderRepository
	productRepo  ports.ProductRepository
	auditService ports.AuditService
	eventBus     ports.EventBus
}

// NewOrderService instantiates sales order service.
func NewOrderService(
	txManager ports.TransactionManager,
	orderRepo ports.OrderRepository,
	productRepo ports.ProductRepository,
	auditService ports.AuditService,
	eventBus ports.EventBus,
) ports.OrderService {
	return &orderService{
		txManager:    txManager,
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		auditService: auditService,
		eventBus:     eventBus,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req ports.CreateOrderRequest, actCtx ports.AuthContext) (*domain.Order, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("cannot create empty sales order")
	}

	orderID := uuid.New().String()
	orderNumber := fmt.Sprintf("ORD-%d", time.Now().UnixNano()/1e6)

	order := &domain.Order{
		ID:            orderID,
		OrderNumber:   orderNumber,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		Status:        domain.OrderStatusCompleted,
		CreatedBy:     actCtx.UserID,
		Items:         make([]domain.OrderItem, 0, len(req.Items)),
	}

	// We'll collect atomic changes across the transaction to publish in an organized batch
	var batchedChanges []domain.ChangeEvent

	err := s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		for _, itemReq := range req.Items {
			if itemReq.Quantity <= 0 {
				return errors.New("item quantity must be greater than zero")
			}

			// Atomic stock deduction inside TX with FOR UPDATE
			updatedProd, err := s.productRepo.AdjustStockTx(ctx, tx, itemReq.ProductID, -itemReq.Quantity)
			if err != nil {
				return fmt.Errorf("stock deduction failed for item %s: %w", itemReq.ProductID, err)
			}

			itemSubtotal := float64(itemReq.Quantity) * updatedProd.Price
			orderItem := domain.OrderItem{
				ID:          uuid.New().String(),
				OrderID:     orderID,
				ProductID:   updatedProd.ID,
				ProductName: updatedProd.Name,
				ProductSKU:  updatedProd.SKU,
				Quantity:    itemReq.Quantity,
				UnitPrice:   updatedProd.Price,
				Subtotal:    itemSubtotal,
			}
			order.Items = append(order.Items, orderItem)

			// Record product delta change event for batch
			batchedChanges = append(batchedChanges, domain.ChangeEvent{
				EventID:  uuid.New().String(),
				Type:     domain.EventStockChanged,
				Resource: domain.ResourceProduct,
				EntityID: updatedProd.ID,
				Data:     updatedProd,
				Delta: map[string]interface{}{
					"stock":      updatedProd.Stock,
					"updated_at": updatedProd.UpdatedAt,
				},
				ActorID:   actCtx.UserID,
				Timestamp: time.Now().UTC(),
			})
		}

		if err := order.CalculateTotals(); err != nil {
			return err
		}

		if err := s.orderRepo.CreateOrderWithItemsTx(ctx, tx, order); err != nil {
			return fmt.Errorf("failed to persist order: %w", err)
		}

		// Add order created event to the batch
		batchedChanges = append(batchedChanges, domain.ChangeEvent{
			EventID:   uuid.New().String(),
			Type:      domain.EventEntityCreated,
			Resource:  domain.ResourceOrder,
			EntityID:  order.ID,
			Data:      order,
			ActorID:   actCtx.UserID,
			Timestamp: time.Now().UTC(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Async audit log
	orderJSON, _ := json.Marshal(order)
	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "orders",
		EntityID:   order.ID,
		Action:     domain.AuditActionBatch,
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		NewValues:  string(orderJSON),
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Publish organized BatchChangeEvent over WebSocket
	batchEvent := domain.BatchChangeEvent{
		BatchID:       uuid.New().String(),
		Operation:     "CREATE_SALES_ORDER",
		EventsCount:   len(batchedChanges),
		Changes:       batchedChanges,
		Timestamp:     time.Now().UTC(),
		ActorID:       actCtx.UserID,
		CorrelationID: order.ID,
	}
	s.eventBus.PublishBatch(batchEvent)

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

func (s *orderService) ListOrders(ctx context.Context, limit, offset int) ([]domain.Order, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.orderRepo.List(ctx, limit, offset)
}

func (s *orderService) CancelOrder(ctx context.Context, id string, actCtx ports.AuthContext) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusCancelled {
		return errors.New("order is already cancelled")
	}

	var batchedChanges []domain.ChangeEvent

	err = s.txManager.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Replenish stock for all items
		for _, item := range order.Items {
			updatedProd, err := s.productRepo.AdjustStockTx(ctx, tx, item.ProductID, item.Quantity)
			if err != nil {
				return fmt.Errorf("failed to replenish stock for product %s: %w", item.ProductID, err)
			}

			batchedChanges = append(batchedChanges, domain.ChangeEvent{
				EventID:  uuid.New().String(),
				Type:     domain.EventStockChanged,
				Resource: domain.ResourceProduct,
				EntityID: updatedProd.ID,
				Data:     updatedProd,
				Delta: map[string]interface{}{
					"stock":      updatedProd.Stock,
					"updated_at": updatedProd.UpdatedAt,
				},
				ActorID:   actCtx.UserID,
				Timestamp: time.Now().UTC(),
			})
		}

		if err := s.orderRepo.UpdateStatus(ctx, id, domain.OrderStatusCancelled); err != nil {
			return err
		}

		order.Status = domain.OrderStatusCancelled
		batchedChanges = append(batchedChanges, domain.ChangeEvent{
			EventID:  uuid.New().String(),
			Type:     domain.EventEntityUpdated,
			Resource: domain.ResourceOrder,
			EntityID: order.ID,
			Data:     order,
			Delta: map[string]interface{}{
				"status": domain.OrderStatusCancelled,
			},
			ActorID:   actCtx.UserID,
			Timestamp: time.Now().UTC(),
		})

		return nil
	})

	if err != nil {
		return err
	}

	// Audit
	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "orders",
		EntityID:   order.ID,
		Action:     "CANCEL_ORDER",
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Publish organized BatchChangeEvent
	s.eventBus.PublishBatch(domain.BatchChangeEvent{
		BatchID:       uuid.New().String(),
		Operation:     "CANCEL_SALES_ORDER",
		EventsCount:   len(batchedChanges),
		Changes:       batchedChanges,
		Timestamp:     time.Now().UTC(),
		ActorID:       actCtx.UserID,
		CorrelationID: order.ID,
	})

	return nil
}
