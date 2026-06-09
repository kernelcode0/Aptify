package api

import (
	"net/http"
)

func (h *Handler) authCheck(w http.ResponseWriter, r *http.Request) {
	// If the request made it here, it passed the authMiddleware
	jsonOK(w, map[string]bool{"authenticated": true}, http.StatusOK)
}
