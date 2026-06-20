package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateUserRequest struct {
	Role     *string `json:"role"`
	Password *string `json:"password"`
}

func validRole(role string) bool {
	return role == "admin" || role == "member" || role == "viewer"
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.db.ListUsers()
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []storage.User{}
	}
	jsonOK(w, users, http.StatusOK)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if !usernameRe.MatchString(req.Username) {
		jsonError(w, "invalid username", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}
	if !validRole(req.Role) {
		jsonError(w, "invalid role", http.StatusBadRequest)
		return
	}

	existing, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		jsonError(w, "username already exists", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, "password error", http.StatusInternalServerError)
		return
	}
	user, err := h.db.CreateUser(req.Username, string(hash), req.Role)
	if err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	detail, _ := json.Marshal(map[string]string{"role": req.Role})
	h.audit(currentUser(r), "create_user", "user:"+user.Username, string(detail))
	jsonOK(w, user, http.StatusCreated)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.db.GetUserByID(id)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	current := currentUser(r)
	if req.Role != nil {
		role := strings.TrimSpace(*req.Role)
		if !validRole(role) {
			jsonError(w, "invalid role", http.StatusBadRequest)
			return
		}
		if current != nil && current.ID == id {
			jsonError(w, "cannot change your own role", http.StatusBadRequest)
			return
		}
		if err := h.db.UpdateUserRole(id, role); err != nil {
			jsonError(w, "db error", http.StatusInternalServerError)
			return
		}
		user.Role = role
	}

	if req.Password != nil {
		if len(*req.Password) < 8 {
			jsonError(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			jsonError(w, "password error", http.StatusInternalServerError)
			return
		}
		if err := h.db.UpdateUserPassword(id, string(hash)); err != nil {
			jsonError(w, "db error", http.StatusInternalServerError)
			return
		}
	}

	jsonOK(w, user, http.StatusOK)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	current := currentUser(r)
	if current != nil && current.ID == id {
		jsonError(w, "cannot delete yourself", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil || user == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}
	if err := h.db.DeleteUser(id); err != nil {
		jsonError(w, "db error", http.StatusInternalServerError)
		return
	}

	h.audit(current, "delete_user", "user:"+user.Username, "")
	w.WriteHeader(http.StatusNoContent)
}
