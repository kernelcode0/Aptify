package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/indexqueue"
	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey int

const ctxUserID ctxKey = 0
const ctxUser ctxKey = 1

// Handler holds all API dependencies.
type Handler struct {
	db        *storage.DB
	fs        *storage.FileStore
	gen       *index.Generator
	signer    *signing.Signer
	jwtSecret string
	version   string
	indexMu   sync.Mutex
	queue     *indexqueue.Queue
}

func New(db *storage.DB, fs *storage.FileStore, gen *index.Generator, signer *signing.Signer, jwtSecret, version string, queue *indexqueue.Queue) *Handler {
	return &Handler{db: db, fs: fs, gen: gen, signer: signer, jwtSecret: jwtSecret, version: version, queue: queue}
}

func (h *Handler) Router(spa http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(SecurityHeaders)
	r.Use(CSRFProtection)

	rl := newRateLimiter()

	// Public endpoints — no auth.
	r.Get("/health", h.health)
	r.Get("/signing-key.asc", h.servePublicKey)
	r.Get("/repo/{slug}/dists/*", h.serveRepoFile)
	r.Get("/repo/{slug}/pool/*", h.serveRepoFile)
	r.Get("/repo/{slug}/repodata/*", h.serveRepoFile)
	r.Get("/repo/{slug}/packages/*", h.serveRepoFile)

	// Auth endpoints
	r.With(rl.RateLimit).Post("/api/auth/login", h.login)
	r.With(rl.RateLimit).Post("/api/auth/2fa/verify", h.verify2FA)

	// Admin API — protected by JWT or API key.
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware)
		r.Get("/api/auth/check", h.authCheck)
		r.Post("/api/auth/2fa/setup", h.setup2FA)
		r.Post("/api/auth/2fa/enable", h.enable2FA)
		r.Post("/api/auth/2fa/disable", h.disable2FA)
		r.Post("/api/auth/2fa/recovery-codes", h.regenerateRecoveryCodes)
		r.Post("/api/auth/keys", h.createAPIKey)
		r.Get("/api/auth/keys", h.listAPIKeys)
		r.Delete("/api/auth/keys/{keyID}", h.deleteAPIKey)
		r.With(RequireRole("admin")).Post("/api/repos", h.createRepo)
		r.Get("/api/repos", h.listRepos)
		r.With(RequireRole("admin")).Put("/api/repos/{id}", h.updateRepo)
		r.With(RequireRole("admin")).Delete("/api/repos/{id}", h.deleteRepo)
		r.With(RequireRole("admin", "member")).Post("/api/repos/{id}/packages", h.uploadPackage)
		r.Get("/api/repos/{id}/packages", h.listPackages)
		r.With(RequireRole("admin", "member")).Delete("/api/repos/{id}/packages/{pkgID}", h.deletePackage)
		r.Get("/api/repos/{id}/setup", h.getSetup)
		r.Get("/api/repos/{id}/status", h.getRepoStatus)
		r.With(RequireRole("admin")).Get("/api/users", h.listUsers)
		r.With(RequireRole("admin")).Post("/api/users", h.createUser)
		r.With(RequireRole("admin")).Put("/api/users/{id}", h.updateUser)
		r.With(RequireRole("admin")).Delete("/api/users/{id}", h.deleteUser)
		r.With(RequireRole("admin")).Post("/api/users/{id}/reset-2fa", h.resetUser2FA)
		r.With(RequireRole("admin")).Get("/api/audit", h.listAuditLog)
		r.With(RequireRole("admin")).Delete("/api/audit", h.clearAuditLog)
		r.Get("/api/system/gpg-key", h.exportGPGKey)
	})

	// Serve embedded SPA for everything else.
	r.Handle("/*", spa)
	return r
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		// 1. Try apt_session cookie (preferred for Web UI)
		cookie, err := r.Cookie("apt_session")
		if err == nil && cookie.Value != "" {
			tokenStr = cookie.Value
		} else {
			// 2. Try Authorization header (for API keys/CLI)
			auth := r.Header.Get("Authorization")
			tokenStr = strings.TrimPrefix(auth, "Bearer ")
		}

		if tokenStr == "" {
			jsonError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 1. Try JWT.
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(h.jwtSecret), nil
		})
		if err == nil && token.Valid {
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				jsonError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if purpose, _ := claims["purpose"].(string); purpose == "2fa_preauth" {
				jsonError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			userID, _ := claims["sub"].(string)
			user, err := h.db.GetUserByID(userID)
			if err != nil || user == nil {
				jsonError(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, user.ID)
			ctx = context.WithValue(ctx, ctxUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 2. Try API key (format: aptify_<32 chars>).
		if strings.HasPrefix(tokenStr, "aptify_") && len(tokenStr) >= 15 {
			prefix := tokenStr[7:15]
			candidates, err := h.db.GetAPIKeyByPrefix(prefix)
			if err == nil {
				for _, k := range candidates {
					if bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(tokenStr)) == nil {
						_ = h.db.UpdateAPIKeyLastUsed(k.ID)
						user, err := h.db.GetUserByID(k.UserID)
						if err != nil || user == nil {
							jsonError(w, "Unauthorized", http.StatusUnauthorized)
							return
						}
						ctx := context.WithValue(r.Context(), ctxUserID, user.ID)
						ctx = context.WithValue(ctx, ctxUser, user)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
		}

		jsonError(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (h *Handler) regenerateRepoIndex(repo *storage.Repo) error {
	h.indexMu.Lock()
	defer h.indexMu.Unlock()

	packages, err := h.db.ListPackages(repo.ID, 0, 0)
	if err != nil {
		return err
	}
	return h.gen.Regenerate(repo, packages)
}

func (h *Handler) servePublicKey(w http.ResponseWriter, r *http.Request) {
	pubKey, err := h.signer.PublicKeyArmored()
	if err != nil {
		http.Error(w, "key error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pgp-keys")
	_, _ = w.Write(pubKey)
}

// serveRepoFile serves static files from the repo's data directory.
func (h *Handler) serveRepoFile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if err := storage.ValidateSlug(slug); err != nil {
		http.Error(w, "invalid slug", http.StatusBadRequest)
		return
	}
	rest := chi.URLParam(r, "*")

	repoDir := h.fs.RepoDir(slug)
	var filePath string
	if strings.Contains(r.URL.Path, "/dists/") {
		filePath = filepath.Join(repoDir, "dists", rest)
	} else if strings.Contains(r.URL.Path, "/pool/") {
		filePath = filepath.Join(repoDir, "pool", rest)
	} else if strings.Contains(r.URL.Path, "/repodata/") {
		filePath = filepath.Join(repoDir, "repodata", rest)
	} else {
		filePath = filepath.Join(repoDir, "packages", rest)
	}

	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(repoDir)+string(filepath.Separator)) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, cleanPath)
}
