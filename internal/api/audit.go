package api

import (
	"fmt"
	"net/http"

	"github.com/kernelcode0/aptify/internal/storage"
)

func (h *Handler) listAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &offset)
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	userID := r.URL.Query().Get("user")
	var entries []storage.AuditEntry
	var err error
	if userID != "" {
		entries, err = h.db.ListAuditLogByUser(userID, limit, offset)
	} else {
		entries, err = h.db.ListAuditLog(limit, offset)
	}
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	total, err := h.db.CountAuditLog(userID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []storage.AuditEntry{}
	}
	jsonOK(w, map[string]any{
		"total":   total,
		"entries": entries,
	}, http.StatusOK)
}

func (h *Handler) clearAuditLog(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(ctxUser).(*storage.User)

	// Forensic Marker Strategy: Insert a clear event FIRST, then delete all other events EXCEPT the new clear event.
	clearEventID, err := h.db.LogAuditWithID(user.ID, user.Username, "clear_audit_log", "audit_log", "Audit log truncated by admin")
	if err != nil {
		jsonError(w, "Failed to create forensic marker", http.StatusInternalServerError)
		return
	}

	err = h.db.ClearAuditLogExcept(r.Context(), clearEventID)
	if err != nil {
		jsonError(w, "Failed to clear audit log", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
