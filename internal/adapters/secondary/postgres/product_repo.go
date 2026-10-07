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

type productRepo struct {
	db *DB
}

// NewProductRepository instantiates product repo.
func NewProductRepository(db *DB) ports.ProductRepository {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *domain.Product) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	query := `
		INSERT INTO products (id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query, p.ID, p.SKU, p.Name, p.Description, p.Category, p.Price, p.Stock, p.MinStock, p.IsDeleted, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("productRepo.Create failed: %w", err)
	}
	return nil
}

func (r *productRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at
		FROM products WHERE id = $1
	`
	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock, &p.MinStock, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("productRepo.GetByID failed: %w", err)
	}
	return &p, nil
}

func (r *productRepo) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	query := `
		SELECT id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at
		FROM products WHERE UPPER(sku) = UPPER($1)
	`
	var p domain.Product
	err := r.db.QueryRowContext(ctx, query, sku).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock, &p.MinStock, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("productRepo.GetBySKU failed: %w", err)
	}
	return &p, nil
}

func (r *productRepo) List(ctx context.Context, includeDeleted bool, limit, offset int) ([]domain.Product, int, error) {
	var countQuery string
	if includeDeleted {
		countQuery = `SELECT COUNT(*) FROM products`
	} else {
		countQuery = `SELECT COUNT(*) FROM products WHERE is_deleted = FALSE`
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("productRepo.List count failed: %w", err)
	}

	var selectQuery string
	if includeDeleted {
		selectQuery = `
			SELECT id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at
			FROM products ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
	} else {
		selectQuery = `
			SELECT id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at
			FROM products WHERE is_deleted = FALSE ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`
	}

	rows, err := r.db.QueryContext(ctx, selectQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.List query failed: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock, &p.MinStock, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, total, nil
}

func (r *productRepo) Update(ctx context.Context, p *domain.Product) error {
	p.UpdatedAt = time.Now().UTC()
	query := `
		UPDATE products
		SET name = $1, description = $2, category = $3, price = $4, min_stock = $5, updated_at = $6
		WHERE id = $7 AND is_deleted = FALSE
	`
	res, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.Category, p.Price, p.MinStock, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("productRepo.Update failed: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("product not found or already deleted")
	}
	return nil
}

func (r *productRepo) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE products SET is_deleted = TRUE, updated_at = $1 WHERE id = $2 AND is_deleted = FALSE`
	res, err := r.db.ExecContext(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("productRepo.SoftDelete failed: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("product not found or already deleted")
	}
	return nil
}

func (r *productRepo) AdjustStockTx(ctx context.Context, tx *sql.Tx, productID string, quantityDelta int) (*domain.Product, error) {
	// Lock row for update
	queryLock := `
		SELECT id, sku, name, description, category, price, stock, min_stock, is_deleted, created_at, updated_at
		FROM products WHERE id = $1 FOR UPDATE
	`
	var p domain.Product
	err := tx.QueryRowContext(ctx, queryLock, productID).Scan(
		&p.ID, &p.SKU, &p.Name, &p.Description, &p.Category, &p.Price, &p.Stock, &p.MinStock, &p.IsDeleted, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("product not found")
	}
	if err != nil {
		return nil, fmt.Errorf("productRepo.AdjustStockTx lock failed: %w", err)
	}

	if p.IsDeleted {
		return nil, errors.New("cannot adjust stock of deleted product")
	}

	newStock := p.Stock + quantityDelta
	if newStock < 0 {
		return nil, fmt.Errorf("insufficient stock for product %s (current: %d, delta: %d)", p.Name, p.Stock, quantityDelta)
	}

	p.Stock = newStock
	p.UpdatedAt = time.Now().UTC()

	updateQuery := `UPDATE products SET stock = $1, updated_at = $2 WHERE id = $3`
	if _, err := tx.ExecContext(ctx, updateQuery, p.Stock, p.UpdatedAt, p.ID); err != nil {
		return nil, fmt.Errorf("productRepo.AdjustStockTx update failed: %w", err)
	}

	return &p, nil
}

func (r *productRepo) AdjustStock(ctx context.Context, productID string, quantityDelta int) (*domain.Product, error) {
	var product *domain.Product
	err := r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		var err error
		product, err = r.AdjustStockTx(ctx, tx, productID, quantityDelta)
		return err
	})
	return product, err
}
