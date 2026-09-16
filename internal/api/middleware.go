package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/kernelcode0/aptify/internal/storage"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next.ServeHTTP(w, r)
	})
}

func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "PATCH" {
			// Skip CSRF check if using API Key / Bearer token
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			origin := r.Header.Get("Origin")
			referer := r.Header.Get("Referer")
			if origin == "" && referer == "" {
				http.Error(w, "Missing CSRF headers", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

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
	if err := h.db.AddAuditEntry(user.ID, user.Username, action, resource, detail); err != nil {
		log.Printf("audit log write failed: %v", err)
	}
}
