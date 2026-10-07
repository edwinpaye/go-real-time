package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sales-system/go-real-time/internal/core/domain"
	"github.com/sales-system/go-real-time/internal/core/ports"
)

type orderRepo struct {
	db *DB
}

// NewOrderRepository instantiates order repo.
func NewOrderRepository(db *DB) ports.OrderRepository {
	return &orderRepo{db: db}
}

func (r *orderRepo) CreateOrderWithItemsTx(ctx context.Context, tx *sql.Tx, o *domain.Order) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	o.CreatedAt = now
	o.UpdatedAt = now

	orderQuery := `
		INSERT INTO orders (id, order_number, customer_name, customer_email, status, total_amount, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, orderQuery, o.ID, o.OrderNumber, o.CustomerName, o.CustomerEmail, o.Status, o.TotalAmount, o.CreatedBy, o.CreatedAt, o.UpdatedAt)
	if err != nil {
		return fmt.Errorf("orderRepo.CreateOrderWithItemsTx order insert failed: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, product_name, product_sku, quantity, unit_price, subtotal)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	for i := range o.Items {
		item := &o.Items[i]
		if item.ID == "" {
			item.ID = uuid.New().String()
		}
		item.OrderID = o.ID
		_, err := tx.ExecContext(ctx, itemQuery, item.ID, item.OrderID, item.ProductID, item.ProductName, item.ProductSKU, item.Quantity, item.UnitPrice, item.Subtotal)
		if err != nil {
			return fmt.Errorf("orderRepo.CreateOrderWithItemsTx item insert failed: %w", err)
		}
	}

	return nil
}

func (r *orderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	orderQuery := `
		SELECT id, order_number, customer_name, customer_email, status, total_amount, COALESCE(created_by, ''), created_at, updated_at
		FROM orders WHERE id = $1
	`
	var o domain.Order
	err := r.db.QueryRowContext(ctx, orderQuery, id).Scan(
		&o.ID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail, &o.Status, &o.TotalAmount, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("orderRepo.GetByID failed: %w", err)
	}

	itemsQuery := `
		SELECT id, order_id, product_id, product_name, product_sku, quantity, unit_price, subtotal
		FROM order_items WHERE order_id = $1
	`
	rows, err := r.db.QueryContext(ctx, itemsQuery, id)
	if err != nil {
		return nil, fmt.Errorf("orderRepo.GetByID items query failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.ProductName, &item.ProductSKU, &item.Quantity, &item.UnitPrice, &item.Subtotal); err != nil {
			return nil, err
		}
		o.Items = append(o.Items, item)
	}

	return &o, nil
}

func (r *orderRepo) List(ctx context.Context, limit, offset int) ([]domain.Order, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("orderRepo.List count failed: %w", err)
	}

	orderQuery := `
		SELECT id, order_number, customer_name, customer_email, status, total_amount, COALESCE(created_by, ''), created_at, updated_at
		FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, orderQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("orderRepo.List query failed: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.CustomerName, &o.CustomerEmail, &o.Status, &o.TotalAmount, &o.CreatedBy, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}

	return orders, total, nil
}

func (r *orderRepo) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`
	res, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("orderRepo.UpdateStatus failed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return errors.New("order not found")
	}
	return nil
}
