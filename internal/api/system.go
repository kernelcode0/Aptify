package api

import (
	"net/http"
	"os"
)

// health returns a minimal liveness/readiness response.
// The application version is intentionally omitted to avoid version
// fingerprinting by unauthenticated clients.
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

// exportGPGKey exports the repository's private GPG signing key.
//
// This endpoint is DISABLED by default. Exposing a private key over HTTP
// is a supply-chain risk: any user with a valid token can exfiltrate the
// key and sign malicious packages that downstream clients will accept.
//
// To enable: set ENABLE_PRIVATE_KEY_EXPORT=true AND ensure the route is
// protected by RequireRole("admin") (see router.go).
//
// Recommendation: retrieve the private key directly from the server filesystem
// (DATA_DIR/repo.key) rather than enabling this endpoint in production.
func (h *Handler) exportGPGKey(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("ENABLE_PRIVATE_KEY_EXPORT") != "true" {
		jsonError(w,
			"private key export is disabled. Set ENABLE_PRIVATE_KEY_EXPORT=true to enable (admin only).",
			http.StatusForbidden)
		return
	}

	privKey, err := h.signer.PrivateKeyArmored()
	if err != nil {
		jsonError(w, "failed to export key", http.StatusInternalServerError)
		return
	}

	// Audit synchronously — this is a high-value destructive-information event.
	if user := currentUser(r); user != nil {
		_ = h.db.AddAuditEntry(user.ID, user.Username, "export_private_gpg_key", "gpg:private",
			"private key exported via HTTP API")
	}

	w.Header().Set("Content-Type", "application/pgp-keys")
	w.Header().Set("Content-Disposition", `attachment; filename="private-key.asc"`)
	_, _ = w.Write(privKey)
}

