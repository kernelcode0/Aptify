package api

import (
	"net/http"
)

func (h *Handler) exportGPGKey(w http.ResponseWriter, r *http.Request) {
	privKey, err := h.signer.PrivateKeyArmored()
	if err != nil {
		jsonError(w, "failed to export key", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pgp-keys")
	w.Header().Set("Content-Disposition", `attachment; filename="private-key.asc"`)
	w.Write(privKey) //nolint:errcheck
}
