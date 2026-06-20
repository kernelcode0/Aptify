package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/kernelcode0/aptify/internal/storage"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	required := ""
	if len(roles) > 0 {
		required = roles[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := currentUser(r)
			if user == nil || !allowed[user.Role] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":         "forbidden",
					"required_role": required,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func currentUser(r *http.Request) *storage.User {
	user, _ := r.Context().Value(ctxUser).(*storage.User)
	return user
}

func (h *Handler) audit(user *storage.User, action, resource, detail string) {
	if user == nil {
		return
	}
	go func() {
		if err := h.db.AddAuditEntry(user.ID, user.Username, action, resource, detail); err != nil {
			log.Printf("audit log write failed: %v", err)
		}
	}()
}
