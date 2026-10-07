package domain

import (
	"errors"
	"strings"
	"time"
)

// Product represents an item in the sales inventory.
type Product struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	MinStock    int       `json:"min_stock"`
	IsDeleted   bool      `json:"is_deleted"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Validate ensures domain product properties meet business standards.
func (p *Product) Validate() error {
	if strings.TrimSpace(p.SKU) == "" {
		return errors.New("sku cannot be empty")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("product name cannot be empty")
	}
	if p.Price < 0 {
		return errors.New("product price cannot be negative")
	}
	if p.Stock < 0 {
		return errors.New("product stock cannot be negative")
	}
	return nil
}

// HasSufficientStock checks if requested quantity can be fulfilled.
func (p *Product) HasSufficientStock(qty int) bool {
	return !p.IsDeleted && p.Stock >= qty
}

// DeductStock deducts inventory quantity safely.
func (p *Product) DeductStock(qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	if p.Stock < qty {
		return errors.New("insufficient product inventory stock")
	}
	p.Stock -= qty
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// AddStock replenishes inventory quantity.
func (p *Product) AddStock(qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be greater than zero")
	}
	p.Stock += qty
	p.UpdatedAt = time.Now().UTC()
	return nil
}
