package api

import (
	"net/http"
	"os"
)

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := h.db.Ping(); err != nil {
		dbStatus = "error"
	}
	jsonOK(w, map[string]string{
		"status": "ok",
		"db":     dbStatus,
	}, http.StatusOK)
}

func (h *Handler) exportGPGKey(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENABLE_PRIVATE_KEY_EXPORT") != "true" {
		jsonError(w, "private key export is disabled", http.StatusForbidden)
		return
	}

	privKey, err := h.signer.PrivateKeyArmored()
	if err != nil {
		jsonError(w, "failed to export key", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pgp-keys")
	w.Header().Set("Content-Disposition", `attachment; filename="private-key.asc"`)
	_, _ = w.Write(privKey)
}
