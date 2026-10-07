package ports

import (
	"context"

	"github.com/sales-system/go-real-time/internal/core/domain"
)

// AuthContext holds request actor information for auditing.
type AuthContext struct {
	UserID    string
	UserEmail string
	UserRole  domain.UserRole
	IPAddress string
	UserAgent string
}

// RegisterRequest data transfer object.
type RegisterRequest struct {
	Email    string          `json:"email"`
	FullName string          `json:"full_name"`
	Password string          `json:"password"`
	Role     domain.UserRole `json:"role"`
}

// LoginRequest data transfer object.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse data transfer object.
type LoginResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// AuthService contract.
type AuthService interface {
	Register(ctx context.Context, req RegisterRequest, actCtx AuthContext) (*domain.User, error)
	Login(ctx context.Context, req LoginRequest, actCtx AuthContext) (*LoginResponse, error)
	ValidateToken(ctx context.Context, tokenStr string) (*domain.User, error)
}

// CreateProductRequest data transfer object.
type CreateProductRequest struct {
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	MinStock    int     `json:"min_stock"`
}

// UpdateProductRequest data transfer object.
type UpdateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	MinStock    int     `json:"min_stock"`
}

// AdjustStockRequest data transfer object.
type AdjustStockRequest struct {
	QuantityDelta int    `json:"quantity_delta"` // e.g. +10 or -5
	Reason        string `json:"reason"`
}

// ProductService contract.
type ProductService interface {
	CreateProduct(ctx context.Context, req CreateProductRequest, actCtx AuthContext) (*domain.Product, error)
	GetProduct(ctx context.Context, id string) (*domain.Product, error)
	ListProducts(ctx context.Context, includeDeleted bool, limit, offset int) ([]domain.Product, int, error)
	UpdateProduct(ctx context.Context, id string, req UpdateProductRequest, actCtx AuthContext) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id string, actCtx AuthContext) error
	AdjustStock(ctx context.Context, id string, req AdjustStockRequest, actCtx AuthContext) (*domain.Product, error)
}

// CreateOrderItemRequest data transfer object.
type CreateOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// CreateOrderRequest data transfer object.
type CreateOrderRequest struct {
	CustomerName  string                   `json:"customer_name"`
	CustomerEmail string                   `json:"customer_email"`
	Items         []CreateOrderItemRequest `json:"items"`
}

// OrderService contract.
type OrderService interface {
	CreateOrder(ctx context.Context, req CreateOrderRequest, actCtx AuthContext) (*domain.Order, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	ListOrders(ctx context.Context, limit, offset int) ([]domain.Order, int, error)
	CancelOrder(ctx context.Context, id string, actCtx AuthContext) error
}

// AuditService contract.
type AuditService interface {
	RecordAsync(log *domain.AuditLog)
	ListLogs(ctx context.Context, entityName string, limit, offset int) ([]domain.AuditLog, int, error)
}
