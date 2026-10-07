package http

import (
	"net/http"
	"strconv"

	"github.com/sales-system/go-real-time/internal/core/ports"
	"github.com/sales-system/go-real-time/internal/infra/tracker"
)

// AuditHandler exposes HTTP routes for querying audit trails and system telemetry.
type AuditHandler struct {
	auditService ports.AuditService
}

func NewAuditHandler(auditService ports.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	entityName := r.URL.Query().Get("entity")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	logs, total, err := h.auditService.ListLogs(r.Context(), entityName, limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *AuditHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	snapshot := tracker.GetTracker().Snapshot()
	respondJSON(w, http.StatusOK, snapshot)
}
