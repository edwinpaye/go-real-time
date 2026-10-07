package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sales-system/go-real-time/internal/core/ports"
)

// OrderHandler exposes HTTP endpoints for sales transactions.
type OrderHandler struct {
	orderService ports.OrderService
}

func NewOrderHandler(orderService ports.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

func (h *OrderHandler) HandleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *OrderHandler) HandleOrderByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid order url path"})
		return
	}

	orderID := parts[3]

	if len(parts) == 5 && parts[4] == "cancel" {
		if r.Method == http.MethodPost {
			h.Cancel(w, r, orderID)
			return
		}
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.Get(w, r, orderID)
	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	orders, total, err := h.orderService.ListOrders(r.Context(), limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   orders,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ports.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	order, err := h.orderService.CreateOrder(r.Context(), req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	order, err := h.orderService.GetOrder(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request, id string) {
	actCtx := GetAuthContext(r)
	if err := h.orderService.CancelOrder(r.Context(), id, actCtx); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "order cancelled successfully"})
}
