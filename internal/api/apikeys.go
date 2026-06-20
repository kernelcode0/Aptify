package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// generateAPIKey creates a new raw key, its 8-char prefix, and a bcrypt hash.
// Raw format: aptify_<32 base64url chars>
func generateAPIKey() (raw, prefix, hash string, err error) {
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = "aptify_" + base64.RawURLEncoding.EncodeToString(b)
	prefix = raw[7:15] // first 8 chars after "aptify_"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	hash = string(hashBytes)
	return
}

func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	userID, _ := r.Context().Value(ctxUserID).(string)
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	raw, prefix, hash, err := generateAPIKey()
	if err != nil {
		jsonError(w, "failed to generate key", http.StatusInternalServerError)
		return
	}

	key, err := h.db.CreateAPIKey(userID, req.Name, hash, prefix)
	if err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	detail, _ := json.Marshal(map[string]string{"prefix": key.Prefix})
	h.audit(currentUser(r), "create_api_key", "apikey:"+key.Name, string(detail))
	// Return the raw key exactly once — it is never stored or returned again.
	jsonOK(w, map[string]any{
		"id":         key.ID,
		"name":       key.Name,
		"prefix":     key.Prefix,
		"key":        raw,
		"created_at": key.CreatedAt,
	}, http.StatusCreated)
}

func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(ctxUserID).(string)
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := h.db.ListAPIKeys(userID)
	if err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []storage.APIKey{}
	}
	jsonOK(w, keys, http.StatusOK)
}

func (h *Handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := chi.URLParam(r, "keyID")
	userID, _ := r.Context().Value(ctxUserID).(string)
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.db.DeleteAPIKey(keyID, userID); err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.audit(currentUser(r), "delete_api_key", "apikey:"+keyID, "")
	w.WriteHeader(http.StatusNoContent)
}
