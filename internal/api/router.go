package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kernelcode/apt-repository/internal/index"
	"github.com/kernelcode/apt-repository/internal/signing"
	"github.com/kernelcode/apt-repository/internal/storage"
)

// Handler holds all API dependencies.
type Handler struct {
	db    *storage.DB
	fs    *storage.FileStore
	gen   *index.Generator
	signer *signing.Signer
	token string
}

func New(db *storage.DB, fs *storage.FileStore, gen *index.Generator, signer *signing.Signer, adminToken string) *Handler {
	return &Handler{db: db, fs: fs, gen: gen, signer: signer, token: adminToken}
}

func (h *Handler) Router(spa http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Public APT endpoints — no auth.
	r.Get("/signing-key.asc", h.servePublicKey)
	r.Get("/repo/{slug}/dists/*", h.serveRepoFile)
	r.Get("/repo/{slug}/pool/*", h.serveRepoFile)

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
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != h.token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) servePublicKey(w http.ResponseWriter, r *http.Request) {
	pubKey, err := h.signer.PublicKeyArmored()
	if err != nil {
		http.Error(w, "key error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pgp-keys")
	w.Write(pubKey)
}

// serveRepoFile serves static files from the repo's data directory.
func (h *Handler) serveRepoFile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	rest := chi.URLParam(r, "*")

	var filePath string
	if strings.Contains(r.URL.Path, "/dists/") {
		filePath = h.fs.RepoDir(slug) + "/dists/" + rest
	} else {
		filePath = h.fs.RepoDir(slug) + "/pool/" + rest
	}
	http.ServeFile(w, r, filePath)
}
