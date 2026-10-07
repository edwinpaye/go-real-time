package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

type productService struct {
	productRepo  ports.ProductRepository
	auditService ports.AuditService
	eventBus     ports.EventBus
}

// NewProductService instantiates product service.
func NewProductService(
	productRepo ports.ProductRepository,
	auditService ports.AuditService,
	eventBus ports.EventBus,
) ports.ProductService {
	return &productService{
		productRepo:  productRepo,
		auditService: auditService,
		eventBus:     eventBus,
	}
}

func (s *productService) CreateProduct(ctx context.Context, req ports.CreateProductRequest, actCtx ports.AuthContext) (*domain.Product, error) {
	sku := strings.ToUpper(strings.TrimSpace(req.SKU))
	if _, err := s.productRepo.GetBySKU(ctx, sku); err == nil {
		return nil, errors.New("product with this SKU already exists")
	}

	product := &domain.Product{
		ID:          uuid.New().String(),
		SKU:         sku,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Price:       req.Price,
		Stock:       req.Stock,
		MinStock:    req.MinStock,
		IsDeleted:   false,
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	// Audit log
	newVals, _ := json.Marshal(product)
	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "products",
		EntityID:   product.ID,
		Action:     domain.AuditActionCreate,
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		NewValues:  string(newVals),
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Real-time Event Notification
	s.eventBus.Publish(domain.ChangeEvent{
		Type:      domain.EventEntityCreated,
		Resource:  domain.ResourceProduct,
		EntityID:  product.ID,
		Data:      product,
		ActorID:   actCtx.UserID,
		Timestamp: time.Now().UTC(),
	})

	return product, nil
}

func (s *productService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return s.productRepo.GetByID(ctx, id)
}

func (s *productService) ListProducts(ctx context.Context, includeDeleted bool, limit, offset int) ([]domain.Product, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.productRepo.List(ctx, includeDeleted, limit, offset)
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req ports.UpdateProductRequest, actCtx ports.AuthContext) (*domain.Product, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product.IsDeleted {
		return nil, errors.New("cannot update a deleted product")
	}

	oldVals, _ := json.Marshal(product)

	product.Name = req.Name
	product.Description = req.Description
	product.Category = req.Category
	product.Price = req.Price
	product.MinStock = req.MinStock

	if err := product.Validate(); err != nil {
		return nil, err
	}

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	newVals, _ := json.Marshal(product)

	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "products",
		EntityID:   product.ID,
		Action:     domain.AuditActionUpdate,
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		OldValues:  string(oldVals),
		NewValues:  string(newVals),
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Precise delta payload for client patcher
	delta := map[string]interface{}{
		"name":        product.Name,
		"description": product.Description,
		"category":    product.Category,
		"price":       product.Price,
		"min_stock":   product.MinStock,
		"updated_at":  product.UpdatedAt,
	}

	s.eventBus.Publish(domain.ChangeEvent{
		Type:      domain.EventEntityUpdated,
		Resource:  domain.ResourceProduct,
		EntityID:  product.ID,
		Data:      product,
		Delta:     delta,
		ActorID:   actCtx.UserID,
		Timestamp: time.Now().UTC(),
	})

	return product, nil
}

func (s *productService) DeleteProduct(ctx context.Context, id string, actCtx ports.AuthContext) error {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	oldVals, _ := json.Marshal(product)

	if err := s.productRepo.SoftDelete(ctx, id); err != nil {
		return err
	}

	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "products",
		EntityID:   id,
		Action:     domain.AuditActionDelete,
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		OldValues:  string(oldVals),
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Real-time Event Notification
	s.eventBus.Publish(domain.ChangeEvent{
		Type:      domain.EventEntityDeleted,
		Resource:  domain.ResourceProduct,
		EntityID:  id,
		Delta: map[string]interface{}{
			"is_deleted": true,
		},
		ActorID:   actCtx.UserID,
		Timestamp: time.Now().UTC(),
	})

	return nil
}

func (s *productService) AdjustStock(ctx context.Context, id string, req ports.AdjustStockRequest, actCtx ports.AuthContext) (*domain.Product, error) {
	product, err := s.productRepo.AdjustStock(ctx, id, req.QuantityDelta)
	if err != nil {
		return nil, err
	}

	// Audit log
	s.auditService.RecordAsync(&domain.AuditLog{
		EntityName: "products",
		EntityID:   id,
		Action:     "STOCK_ADJUST",
		ActorID:    actCtx.UserID,
		ActorEmail: actCtx.UserEmail,
		NewValues:  req.Reason,
		IPAddress:  actCtx.IPAddress,
		UserAgent:  actCtx.UserAgent,
	})

	// Real-time Stock Changed Event
	s.eventBus.Publish(domain.ChangeEvent{
		Type:     domain.EventStockChanged,
		Resource: domain.ResourceProduct,
		EntityID: id,
		Data:     product,
		Delta: map[string]interface{}{
			"stock":      product.Stock,
			"updated_at": product.UpdatedAt,
		},
		ActorID:   actCtx.UserID,
		Timestamp: time.Now().UTC(),
	})

	return product, nil
}
