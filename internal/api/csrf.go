package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

func CSRFProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			// Generate CSRF token if not present
			_, err := r.Cookie("csrf_token")
			if err != nil {
				b := make([]byte, 32)
				rand.Read(b)
				token := base64.RawURLEncoding.EncodeToString(b)
				http.SetCookie(w, &http.Cookie{
					Name:     "csrf_token",
					Value:    token,
					Path:     "/",
					HttpOnly: false, // Must be readable by frontend to send in header
					SameSite: http.SameSiteStrictMode,
				})
			}
			next.ServeHTTP(w, r)
			return
		}

		// Verify CSRF token for state-changing methods
		cookie, err := r.Cookie("csrf_token")
		if err != nil {
			http.Error(w, "Missing CSRF token cookie", http.StatusForbidden)
			return
		}
		header := r.Header.Get("X-CSRF-Token")
		if header == "" || header != cookie.Value {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
