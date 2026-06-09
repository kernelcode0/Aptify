package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kernelcode0/aptify/internal/deb"
	"github.com/kernelcode0/aptify/internal/storage"
)

var validPkgName = regexp.MustCompile(`^[a-z0-9\+\-\.]+$`)

const maxUploadSize = 512 << 20 // 512 MiB

func (h *Handler) uploadPackage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(repoID)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	// #nosec G120 -- bounded by MaxBytesReader above
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "file field required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(header.Filename, ".deb") {
		jsonError(w, "only .deb files are accepted", http.StatusBadRequest)
		return
	}

	// Read into memory to allow re-reading for parse + save.
	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "read error", http.StatusInternalServerError)
		return
	}

	info, err := deb.Parse(bytes.NewReader(data))
	if err != nil {
		jsonError(w, "invalid .deb: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := sanitizeFilename(header.Filename)

	if !validPkgName.MatchString(info.Package) {
		jsonError(w, "invalid package name in control file", http.StatusBadRequest)
		return
	}

	// Save to pool.
	if _, err := h.fs.SavePackage(repo.Slug, info.Package, filename, bytes.NewReader(data)); err != nil {
		jsonError(w, "storage error", http.StatusInternalServerError)
		return
	}

	pkg := &storage.Package{
		RepoID:      repo.ID,
		Filename:    filename,
		Package:     info.Package,
		Version:     info.Version,
		Arch:        info.Architecture,
		Size:        info.Size,
		SHA256:      info.SHA256,
		SHA1:        info.SHA1,
		MD5:         info.MD5,
		ControlJSON: cleanControl(info.ControlBlock),
	}
	if err := h.db.AddPackage(pkg); err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Regenerate index.
	packages, err := h.db.ListPackages(repo.ID, 0, 0)
	if err != nil {
		jsonError(w, "index error", http.StatusInternalServerError)
		return
	}
	if err := h.gen.Regenerate(repo, packages); err != nil {
		jsonError(w, "index regenerate error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonOK(w, pkg, http.StatusCreated)
}

func (h *Handler) listPackages(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(repoID)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}

	page := 1
	limit := 50
	if p := r.URL.Query().Get("page"); p != "" {
		_, _ = fmt.Sscanf(p, "%d", &page)
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		_, _ = fmt.Sscanf(l, "%d", &limit)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	pkgs, err := h.db.ListPackages(repo.ID, limit, offset)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	count, err := h.db.CountPackages(repo.ID)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	if pkgs == nil {
		pkgs = []storage.Package{}
	}
	jsonOK(w, map[string]any{
		"packages": pkgs,
		"total":    count,
	}, http.StatusOK)
}

func (h *Handler) deletePackage(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	pkgID := chi.URLParam(r, "pkgID")

	repo, err := h.db.GetRepo(repoID)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}
	pkg, err := h.db.GetPackage(pkgID)
	if err != nil || pkg == nil || pkg.RepoID != repoID {
		jsonError(w, "package not found", http.StatusNotFound)
		return
	}

	// Remove file from disk.
	_ = h.fs.DeletePackageFile(repo.Slug, pkg.Package, pkg.Filename)

	if err := h.db.DeletePackage(pkgID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Regenerate index without the deleted package.
	packages, err := h.db.ListPackages(repoID, 0, 0)
	if err != nil {
		jsonError(w, "index error", http.StatusInternalServerError)
		return
	}
	if err := h.gen.Regenerate(repo, packages); err != nil {
		jsonError(w, "index regenerate error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sanitizeFilename strips path components and normalises the filename.
func sanitizeFilename(name string) string {
	return filepath.Base(filepath.Clean(name))
}

// cleanControl removes fields that we add ourselves (Filename, Size, SHA*)
// to avoid duplication in the Packages index.
func cleanControl(block string) string {
	var lines []string
	skip := map[string]bool{
		"filename": true, "size": true, "sha256": true,
		"sha1": true, "md5sum": true, "md5": true,
	}
	for _, line := range strings.Split(block, "\n") {
		if idx := strings.IndexByte(line, ':'); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(line[:idx]))
			if skip[key] {
				continue
			}
		}
		lines = append(lines, line)
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if result != "" {
		result += "\n"
	}
	return fmt.Sprintf("%s", result)
}
