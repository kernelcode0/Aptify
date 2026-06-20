package api

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
)

// Handler holds all API dependencies.
type Handler struct {
	db        *storage.DB
	fs        *storage.FileStore
	gen       *index.Generator
	signer    *signing.Signer
	jwtSecret string
	version   string
	indexMu   sync.Mutex
}

func New(db *storage.DB, fs *storage.FileStore, gen *index.Generator, signer *signing.Signer, jwtSecret, version string) *Handler {
	return &Handler{db: db, fs: fs, gen: gen, signer: signer, jwtSecret: jwtSecret, version: version}
}

func (h *Handler) Router(spa http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public endpoints — no auth.
	r.Get("/health", h.health)
	r.Get("/signing-key.asc", h.servePublicKey)
	r.Get("/repo/{slug}/dists/*", h.serveRepoFile)
	r.Get("/repo/{slug}/pool/*", h.serveRepoFile)

	// Auth endpoint
	r.Post("/api/auth/login", h.login)

	// Admin API — protected.
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware)
		r.Get("/api/auth/check", h.authCheck)
		r.Post("/api/repos", h.createRepo)
		r.Get("/api/repos", h.listRepos)
		r.Put("/api/repos/{id}", h.updateRepo)
		r.Delete("/api/repos/{id}", h.deleteRepo)
		r.Post("/api/repos/{id}/packages", h.uploadPackage)
		r.Get("/api/repos/{id}/packages", h.listPackages)
		r.Delete("/api/repos/{id}/packages/{pkgID}", h.deletePackage)
		r.Get("/api/repos/{id}/setup", h.getSetup)
		r.Get("/api/system/gpg-key", h.exportGPGKey)
	})

	// Serve embedded SPA for everything else.
	r.Handle("/*", spa)
	return r
}

func (h *Handler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(h.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
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
	rest := chi.URLParam(r, "*")

	repoDir := h.fs.RepoDir(slug)
	var filePath string
	if strings.Contains(r.URL.Path, "/dists/") {
		filePath = filepath.Join(repoDir, "dists", rest)
	} else {
		filePath = filepath.Join(repoDir, "pool", rest)
	}

	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(repoDir)+string(filepath.Separator)) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, cleanPath)
}
