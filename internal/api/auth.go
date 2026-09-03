package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code,omitempty"`
}

type verify2FAReq struct {
	PreAuthToken string `json:"pre_auth_token"`
	Code         string `json:"code"`
}

type enable2FAReq struct {
	Code string `json:"code"`
}

type disable2FAReq struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

type regenRecoveryReq struct {
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		jsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if user == nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	if user.TOTPEnabled {
		if req.Code != "" {
			valid := ValidateTOTP(user.TOTPSecret, req.Code, time.Now())
			usedRecovery := false
			if !valid {
				valid, err = h.verifyAndConsumeRecoveryCode(user, req.Code)
				if err != nil {
					jsonError(w, "server error", http.StatusInternalServerError)
					return
				}
				usedRecovery = valid
			}
			if !valid {
				jsonError(w, "invalid two-factor authentication code", http.StatusUnauthorized)
				return
			}
			h.issueSessionCookie(w, user)
			if usedRecovery {
				h.audit(user, "login_2fa_recovery", "user:"+user.Username, "")
			} else {
				h.audit(user, "login_2fa_totp", "user:"+user.Username, "")
			}
			jsonOK(w, map[string]string{"status": "ok"}, http.StatusOK)
			return
		}

		// Issue short-lived pre-auth token (5 minutes)
		claims := jwt.MapClaims{
			"sub":     user.ID,
			"purpose": "2fa_preauth",
			"exp":     time.Now().Add(5 * time.Minute).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(h.jwtSecret))
		if err != nil {
			jsonError(w, "failed to sign token", http.StatusInternalServerError)
			return
		}

		jsonOK(w, map[string]string{
			"status":         "2fa_required",
			"pre_auth_token": tokenString,
		}, http.StatusOK)
		return
	}

	h.issueSessionCookie(w, user)
	h.audit(user, "login", "user:"+user.Username, "")
	jsonOK(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *Handler) verify2FA(w http.ResponseWriter, r *http.Request) {
	var req verify2FAReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, err := jwt.Parse(req.PreAuthToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		jsonError(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["purpose"] != "2fa_preauth" {
		jsonError(w, "invalid token purpose", http.StatusUnauthorized)
		return
	}

	userID, _ := claims["sub"].(string)
	user, err := h.db.GetUserByID(userID)
	if err != nil || user == nil || !user.TOTPEnabled {
		jsonError(w, "invalid user or 2FA not enabled", http.StatusUnauthorized)
		return
	}

	code := strings.TrimSpace(req.Code)
	valid := ValidateTOTP(user.TOTPSecret, code, time.Now())
	usedRecovery := false
	if !valid {
		valid, err = h.verifyAndConsumeRecoveryCode(user, code)
		if err != nil {
			jsonError(w, "server error", http.StatusInternalServerError)
			return
		}
		usedRecovery = valid
	}

	if !valid {
		jsonError(w, "invalid two-factor authentication code", http.StatusUnauthorized)
		return
	}

	h.issueSessionCookie(w, user)
	if usedRecovery {
		h.audit(user, "login_2fa_recovery", "user:"+user.Username, "")
	} else {
		h.audit(user, "login_2fa_totp", "user:"+user.Username, "")
	}
	jsonOK(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *Handler) setup2FA(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	secret, err := GenerateTOTPSecret()
	if err != nil {
		jsonError(w, "failed to generate secret", http.StatusInternalServerError)
		return
	}

	otpURL := BuildOTPAuthURI(user.Username, secret)
	qrDataURI, err := GenerateQRCodeDataURI(otpURL)
	if err != nil {
		jsonError(w, "failed to generate QR code", http.StatusInternalServerError)
		return
	}

	plainCodes, hashedCodes, err := GenerateRecoveryCodes(10)
	if err != nil {
		jsonError(w, "failed to generate recovery codes", http.StatusInternalServerError)
		return
	}

	hashedJSON, _ := json.Marshal(hashedCodes)
	if err := h.db.UpdateUser2FA(user.ID, false, secret, string(hashedJSON)); err != nil {
		jsonError(w, "failed to save pending 2FA secret", http.StatusInternalServerError)
		return
	}

	jsonOK(w, map[string]any{
		"secret":         secret,
		"otpauth_url":    otpURL,
		"qr_code":        qrDataURI,
		"recovery_codes": plainCodes,
	}, http.StatusOK)
}

func (h *Handler) enable2FA(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req enable2FAReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	freshUser, err := h.db.GetUserByID(user.ID)
	if err != nil || freshUser == nil || freshUser.TOTPSecret == "" {
		jsonError(w, "2FA setup has not been initiated", http.StatusBadRequest)
		return
	}

	if !ValidateTOTP(freshUser.TOTPSecret, req.Code, time.Now()) {
		jsonError(w, "invalid verification code", http.StatusBadRequest)
		return
	}

	if err := h.db.UpdateUser2FA(freshUser.ID, true, freshUser.TOTPSecret, freshUser.TOTPRecoveryCodes); err != nil {
		jsonError(w, "failed to activate 2FA", http.StatusInternalServerError)
		return
	}

	h.audit(user, "enable_2fa", "user:"+user.Username, "")
	jsonOK(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *Handler) disable2FA(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req disable2FAReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	freshUser, err := h.db.GetUserByID(user.ID)
	if err != nil || freshUser == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(freshUser.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "incorrect password", http.StatusUnauthorized)
		return
	}

	if freshUser.TOTPEnabled {
		code := strings.TrimSpace(req.Code)
		valid := ValidateTOTP(freshUser.TOTPSecret, code, time.Now())
		if !valid {
			valid, _ = h.verifyAndConsumeRecoveryCode(freshUser, code)
		}
		if !valid {
			jsonError(w, "invalid two-factor authentication code", http.StatusUnauthorized)
			return
		}
	}

	if err := h.db.ResetUser2FA(freshUser.ID); err != nil {
		jsonError(w, "failed to disable 2FA", http.StatusInternalServerError)
		return
	}

	h.audit(user, "disable_2fa", "user:"+user.Username, "")
	jsonOK(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *Handler) regenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req regenRecoveryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	freshUser, err := h.db.GetUserByID(user.ID)
	if err != nil || freshUser == nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	if !freshUser.TOTPEnabled {
		jsonError(w, "2FA is not enabled", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(freshUser.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "incorrect password", http.StatusUnauthorized)
		return
	}

	plainCodes, hashedCodes, err := GenerateRecoveryCodes(10)
	if err != nil {
		jsonError(w, "failed to generate recovery codes", http.StatusInternalServerError)
		return
	}

	hashedJSON, _ := json.Marshal(hashedCodes)
	if err := h.db.UpdateUserRecoveryCodes(freshUser.ID, string(hashedJSON)); err != nil {
		jsonError(w, "failed to save recovery codes", http.StatusInternalServerError)
		return
	}

	h.audit(user, "regenerate_recovery_codes", "user:"+user.Username, "")
	jsonOK(w, map[string]any{"recovery_codes": plainCodes}, http.StatusOK)
}

func (h *Handler) verifyAndConsumeRecoveryCode(user *storage.User, code string) (bool, error) {
	code = strings.TrimSpace(code)
	if code == "" || user.TOTPRecoveryCodes == "" || user.TOTPRecoveryCodes == "[]" {
		return false, nil
	}

	var hashes []string
	if err := json.Unmarshal([]byte(user.TOTPRecoveryCodes), &hashes); err != nil {
		return false, nil
	}

	matchedIdx := -1
	for i, hash := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(code)) == nil {
			matchedIdx = i
			break
		}
	}

	if matchedIdx == -1 {
		return false, nil
	}

	remaining := append(hashes[:matchedIdx], hashes[matchedIdx+1:]...)
	remJSON, err := json.Marshal(remaining)
	if err != nil {
		return false, err
	}

	if err := h.db.UpdateUserRecoveryCodes(user.ID, string(remJSON)); err != nil {
		return false, err
	}
	user.TOTPRecoveryCodes = string(remJSON)
	return true, nil
}

func (h *Handler) issueSessionCookie(w http.ResponseWriter, user *storage.User) {
	claims := jwt.MapClaims{
		"sub":     user.ID,
		"usr":     user.Username,
		"rol":     user.Role,
		"purpose": "session",
		"exp":     time.Now().Add(2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "apt_session",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7200, // 2 hours
	})
}

func (h *Handler) authCheck(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	jsonOK(w, map[string]any{
		"valid":              true,
		"username":           user.Username,
		"role":               user.Role,
		"user_id":            user.ID,
		"two_factor_enabled": user.TOTPEnabled,
	}, http.StatusOK)
}
