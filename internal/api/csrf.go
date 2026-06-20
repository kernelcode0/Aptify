package api

import (
	"net/http"
	"strings"
)

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
