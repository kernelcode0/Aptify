package api

import (
	"bytes"
	"crypto/md5"  // #nosec G501
	"crypto/sha1" // #nosec G505
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kernelcode0/aptify/internal/deb"
	"github.com/kernelcode0/aptify/internal/storage"
)

var validPkgName = regexp.MustCompile(`^[a-z0-9\+\-\.]+$`)
var rpmFilenameRe = regexp.MustCompile(`^(.+)-([^-]+)-([^-]+)\.([A-Za-z0-9_]+)\.rpm$`)

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

	if repo.Type == "rpm" {
		h.uploadRPMPackage(w, r, repo, file, header.Filename)
		return
	}

	if !strings.HasSuffix(header.Filename, ".deb") {
		jsonError(w, "this repo expects .deb files", http.StatusBadRequest)
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

	// Reject duplicate file overwrites in this repo.
	if existingID, exists, err := h.db.PackageFilenameExists(repo.ID, filename); err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "package already exists", "id": existingID})
		return
	}

	// Save to pool.
	savedPath, err := h.fs.SavePackage(repo.Slug, info.Package, filename, bytes.NewReader(data))
	if err != nil {
		jsonError(w, "storage error", http.StatusInternalServerError)
		return
	}
	cleanupSavedFile := func() {
		if savedPath != "" {
			/* #nosec G703 */
			_ = os.Remove(savedPath)
		}
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
		cleanupSavedFile()
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Enqueue index regeneration asynchronously; return 201 immediately.
	h.queue.Enqueue(repo.ID)
	detail, _ := json.Marshal(map[string]string{
		"repo":    repo.Slug,
		"version": info.Version,
		"arch":    info.Architecture,
	})
	h.audit(currentUser(r), "upload", "package:"+filename, string(detail))
	jsonOK(w, pkg, http.StatusCreated)
}

func (h *Handler) uploadRPMPackage(w http.ResponseWriter, r *http.Request, repo *storage.Repo, file io.Reader, originalFilename string) {
	if !strings.HasSuffix(originalFilename, ".rpm") {
		jsonError(w, "this repo expects .rpm files", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "read error", http.StatusInternalServerError)
		return
	}
	filename := sanitizeFilename(originalFilename)
	info, err := parseRPMFilename(filename, data)
	if err != nil {
		jsonError(w, "invalid .rpm: "+err.Error(), http.StatusBadRequest)
		return
	}
	if !validPkgName.MatchString(info.Package) {
		jsonError(w, "invalid package name in rpm filename", http.StatusBadRequest)
		return
	}
	// Reject duplicate file overwrites in this repo.
	if existingID, exists, err := h.db.PackageFilenameExists(repo.ID, filename); err != nil {
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "package already exists", "id": existingID})
		return
	}

	savedPath, err := h.fs.SaveRPMPackage(repo.Slug, filename, bytes.NewReader(data))
	if err != nil {
		jsonError(w, "storage error", http.StatusInternalServerError)
		return
	}
	cleanupSavedFile := func() {
		if savedPath != "" {
			/* #nosec G703 */
			_ = os.Remove(savedPath)
		}
	}

	pkg := &storage.Package{
		RepoID:   repo.ID,
		Filename: filename,
		Package:  info.Package,
		Version:  info.Version,
		Release:  info.Release,
		Arch:     info.Arch,
		Size:     int64(len(data)),
		SHA256:   info.SHA256,
		SHA1:     info.SHA1,
		MD5:      info.MD5,
	}
	if err := h.db.AddPackage(pkg); err != nil {
		cleanupSavedFile()
		jsonError(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.queue.Enqueue(repo.ID)
	detail, _ := json.Marshal(map[string]string{
		"repo":    repo.Slug,
		"version": info.Version,
		"release": info.Release,
		"arch":    info.Arch,
	})
	h.audit(currentUser(r), "upload", "package:"+filename, string(detail))
	jsonOK(w, struct {
		*storage.Package
		Name string `json:"name"`
	}{
		Package: pkg,
		Name:    pkg.Package,
	}, http.StatusCreated)
}

type rpmFilenameInfo struct {
	Package string
	Version string
	Release string
	Arch    string
	SHA256  string
	SHA1    string
	MD5     string
}

func parseRPMFilename(filename string, data []byte) (*rpmFilenameInfo, error) {
	m := rpmFilenameRe.FindStringSubmatch(filename)
	if m == nil {
		return nil, fmt.Errorf("expected name-version-release.arch.rpm")
	}
	s256 := sha256.Sum256(data)
	s1 := sha1.Sum(data) // #nosec G401 -- compatibility checksum
	m5 := md5.Sum(data)  // #nosec G401 -- compatibility checksum
	return &rpmFilenameInfo{
		Package: m[1],
		Version: m[2],
		Release: m[3],
		Arch:    m[4],
		SHA256:  hex.EncodeToString(s256[:]),
		SHA1:    hex.EncodeToString(s1[:]),
		MD5:     hex.EncodeToString(m5[:]),
	}, nil
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

	if err := h.db.DeletePackage(pkgID); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	if err := h.regenerateRepoIndex(repo); err != nil {
		jsonError(w, "index regenerate error", http.StatusInternalServerError)
		return
	}
	var deleteErr error
	if repo.Type == "rpm" {
		deleteErr = h.fs.DeleteRPMPackageFile(repo.Slug, pkg.Filename)
	} else {
		deleteErr = h.fs.DeletePackageFile(repo.Slug, pkg.Package, pkg.Filename)
	}
	if deleteErr != nil && !errors.Is(deleteErr, os.ErrNotExist) {
		jsonError(w, "package deleted but file cleanup failed", http.StatusInternalServerError)
		return
	}

	detail, _ := json.Marshal(map[string]string{"repo": repo.Slug})
	h.audit(currentUser(r), "delete_package", "package:"+pkg.Filename, string(detail))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getRepoStatus(w http.ResponseWriter, r *http.Request) {
	repoID := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(repoID)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}
	s := h.queue.RepoStatus(repoID)
	jsonOK(w, map[string]any{
		"id":           repoID,
		"indexing":     s.Indexing,
		"last_indexed": s.LastIndexed,
	}, http.StatusOK)
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
		if strings.TrimSpace(line) == "" {
			continue
		}
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
