package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kernelcode0/aptify/internal/storage"
)

var codenameRe = regexp.MustCompile(`^[a-z][a-z0-9\-\.]{0,49}$`)

type createRepoRequest struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Codename string `json:"codename"`
	Type     string `json:"type"`
}

func (h *Handler) createRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = "deb"
	}
	if req.Type != "deb" && req.Type != "rpm" {
		jsonError(w, "type must be deb or rpm", http.StatusBadRequest)
		return
	}
	if req.Type == "deb" && req.Codename == "" {
		req.Codename = "stable"
	}
	if err := storage.ValidateSlug(req.Slug); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Type == "deb" && !codenameRe.MatchString(req.Codename) {
		jsonError(w, "invalid codename", http.StatusBadRequest)
		return
	}

	existing, err := h.db.GetRepoBySlug(req.Slug)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		jsonError(w, "slug already exists", http.StatusConflict)
		return
	}

	var initErr error
	if req.Type == "rpm" {
		initErr = h.fs.InitRPMRepo(req.Slug)
	} else {
		initErr = h.fs.InitRepo(req.Slug, req.Codename)
	}
	if initErr != nil {
		jsonError(w, "failed to create repo dirs", http.StatusInternalServerError)
		return
	}

	repo, err := h.db.CreateRepo(req.Slug, req.Name, req.Codename, req.Type)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	// Generate empty indexes.
	if err := h.gen.Regenerate(repo, nil); err != nil {
		jsonError(w, "index error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.audit(currentUser(r), "create_repo", "repo:"+repo.Slug, "")
	jsonOK(w, repo, http.StatusCreated)
}

func (h *Handler) listRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := h.db.ListRepos()
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if repos == nil {
		repos = []storage.Repo{}
	}
	jsonOK(w, repos, http.StatusOK)
}

func (h *Handler) deleteRepo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(id)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}
	if err := h.db.DeleteRepo(id); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if err := h.fs.DeleteRepoDir(repo.Slug); err != nil {
		jsonError(w, "repo deleted but file cleanup failed", http.StatusInternalServerError)
		return
	}
	h.audit(currentUser(r), "delete_repo", "repo:"+repo.Slug, "")
	w.WriteHeader(http.StatusNoContent)
}

type updateRepoRequest struct {
	Name     string `json:"name"`
	Codename string `json:"codename"`
}

func (h *Handler) updateRepo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(id)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}

	var req updateRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = repo.Name
	}
	if req.Codename == "" {
		req.Codename = repo.Codename
	}
	if repo.Type == "deb" && !codenameRe.MatchString(req.Codename) {
		jsonError(w, "invalid codename", http.StatusBadRequest)
		return
	}

	codenameChanged := repo.Type == "deb" && repo.Codename != req.Codename
	if codenameChanged {
		if err := h.fs.InitRepo(repo.Slug, req.Codename); err != nil {
			jsonError(w, "failed to create repo dirs", http.StatusInternalServerError)
			return
		}
	}

	if err := h.db.UpdateRepo(id, req.Name, req.Codename); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	repo.Name = req.Name
	repo.Codename = req.Codename

	if codenameChanged {
		// Regenerate index in new path
		if err := h.regenerateRepoIndex(repo); err != nil {
			jsonError(w, "index regenerate error", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, repo, http.StatusOK)
}

func (h *Handler) getSetup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	repo, err := h.db.GetRepo(id)
	if err != nil || repo == nil {
		jsonError(w, "repo not found", http.StatusNotFound)
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	baseURL := scheme + "://" + r.Host

	arches, _ := h.db.GetRepoArchitectures(repo.ID)
	archStr := "amd64"
	if len(arches) > 0 {
		archStr = strings.Join(arches, ",")
	}

	if repo.Type == "rpm" {
		jsonOK(w, map[string]string{
			"type":    "rpm",
			"keyURL":  baseURL + "/signing-key.asc",
			"repoURL": baseURL + "/repo/" + repo.Slug,
			"repoFile": `[aptify-` + repo.Slug + `]
name=` + repo.Name + `
baseurl=` + baseURL + `/repo/` + repo.Slug + `
enabled=1
gpgcheck=1
gpgkey=` + baseURL + `/signing-key.asc`,
			"install": "sudo tee /etc/yum.repos.d/" + repo.Slug + ".repo",
			"update":  "sudo dnf makecache",
		}, http.StatusOK)
		return
	}

	jsonOK(w, map[string]string{
		"type":      "deb",
		"keyURL":    baseURL + "/signing-key.asc",
		"repoURL":   baseURL + "/repo/" + repo.Slug,
		"codename":  repo.Codename,
		"component": "main",
		"addKey":    "curl -fsSL " + baseURL + "/signing-key.asc | sudo gpg --dearmor -o /etc/apt/keyrings/" + repo.Slug + ".gpg",
		"addSource": `echo "deb [arch=` + archStr + ` signed-by=/etc/apt/keyrings/` + repo.Slug + `.gpg] ` +
			baseURL + `/repo/` + repo.Slug + ` ` + repo.Codename + ` main" | sudo tee /etc/apt/sources.list.d/` + repo.Slug + `.list`,
		"update": "sudo apt update",
	}, http.StatusOK)
}
