package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/sales-system/go-real-time/internal/core/ports"
)

// ProductHandler exposes HTTP routes for inventory and products.
type ProductHandler struct {
	productService ports.ProductService
}

func NewProductHandler(productService ports.ProductService) *ProductHandler {
	return &ProductHandler{productService: productService}
}

func (h *ProductHandler) HandleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		h.Create(w, r)
	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProductHandler) HandleProductByID(w http.ResponseWriter, r *http.Request) {
	// Path format: /api/v1/products/{id} or /api/v1/products/{id}/stock
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid product url path"})
		return
	}

	productID := parts[3]

	if len(parts) == 5 && parts[4] == "stock" {
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			h.AdjustStock(w, r, productID)
			return
		}
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.Get(w, r, productID)
	case http.MethodPut, http.MethodPatch:
		h.Update(w, r, productID)
	case http.MethodDelete:
		h.Delete(w, r, productID)
	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	includeDeleted := r.URL.Query().Get("include_deleted") == "true"

	products, total, err := h.productService.ListProducts(r.Context(), includeDeleted, limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   products,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ports.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	product, err := h.productService.CreateProduct(r.Context(), req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request, id string) {
	product, err := h.productService.GetProduct(r.Context(), id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request, id string) {
	var req ports.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	product, err := h.productService.UpdateProduct(r.Context(), id, req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request, id string) {
	actCtx := GetAuthContext(r)
	if err := h.productService.DeleteProduct(r.Context(), id, actCtx); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "product deleted successfully"})
}

func (h *ProductHandler) AdjustStock(w http.ResponseWriter, r *http.Request, id string) {
	var req ports.AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON request"})
		return
	}

	actCtx := GetAuthContext(r)
	product, err := h.productService.AdjustStock(r.Context(), id, req, actCtx)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, product)
}
