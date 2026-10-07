package ports

import (
	"context"
	"database/sql"

	"github.com/sales-system/go-real-time/internal/core/domain"
)

// UserRepository contract for persistence.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, limit, offset int) ([]domain.User, int, error)
	Update(ctx context.Context, user *domain.User) error
}

// ProductRepository contract for inventory persistence and atomic stock updates.
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	GetBySKU(ctx context.Context, sku string) (*domain.Product, error)
	List(ctx context.Context, includeDeleted bool, limit, offset int) ([]domain.Product, int, error)
	Update(ctx context.Context, product *domain.Product) error
	SoftDelete(ctx context.Context, id string) error
	AdjustStockTx(ctx context.Context, tx *sql.Tx, productID string, quantityDelta int) (*domain.Product, error)
	AdjustStock(ctx context.Context, productID string, quantityDelta int) (*domain.Product, error)
}

// OrderRepository contract for sales transactions.
type OrderRepository interface {
	CreateOrderWithItemsTx(ctx context.Context, tx *sql.Tx, order *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	List(ctx context.Context, limit, offset int) ([]domain.Order, int, error)
	UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error
}

// AuditRepository contract for storing and querying audit logs.
type AuditRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, entityName string, limit, offset int) ([]domain.AuditLog, int, error)
}

// TransactionManager contract for atomic database operations.
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error
}
