package domain

import (
	"errors"
	"time"
)

// OrderStatus defines the lifecycle status of a sales order.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusCompleted OrderStatus = "completed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// OrderItem represents a line item within a sale.
type OrderItem struct {
	ID          string  `json:"id"`
	OrderID     string  `json:"order_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ProductSKU  string  `json:"product_sku"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

// Order represents a sales order.
type Order struct {
	ID            string      `json:"id"`
	OrderNumber   string      `json:"order_number"`
	CustomerName  string      `json:"customer_name"`
	CustomerEmail string      `json:"customer_email"`
	Status        OrderStatus `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	CreatedBy     string      `json:"created_by"`
	Items         []OrderItem `json:"items"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// CalculateTotals computes line subtotals and order total amount.
func (o *Order) CalculateTotals() error {
	if len(o.Items) == 0 {
		return errors.New("order must contain at least one item")
	}

	var total float64
	for i := range o.Items {
		item := &o.Items[i]
		if item.Quantity <= 0 {
			return errors.New("order item quantity must be positive")
		}
		if item.UnitPrice < 0 {
			return errors.New("order item unit price cannot be negative")
		}
		item.Subtotal = float64(item.Quantity) * item.UnitPrice
		total += item.Subtotal
	}

	o.TotalAmount = total
	return nil
}
