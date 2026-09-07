package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTOTPGenerationAndValidation(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("failed to generate secret: %v", err)
	}
	if len(secret) != 32 {
		t.Fatalf("expected 32-char base32 secret, got %d chars (%s)", len(secret), secret)
	}

	now := time.Now()
	code, err := GenerateTOTP(secret, now)
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	// Current time code must be valid
	if !ValidateTOTP(secret, code, now) {
		t.Fatalf("valid code %s was rejected", code)
	}

	// +/- 30s window must be valid
	if !ValidateTOTP(secret, code, now.Add(25*time.Second)) {
		t.Fatalf("code should be valid within +25s")
	}
	if !ValidateTOTP(secret, code, now.Add(-25*time.Second)) {
		t.Fatalf("code should be valid within -25s")
	}

	// Outside window (+90s) must be rejected
	if ValidateTOTP(secret, code, now.Add(90*time.Second)) {
		t.Fatalf("code should NOT be valid at +90s")
	}
}

func TestRecoveryCodesGenerationAndHashing(t *testing.T) {
	plain, hashed, err := GenerateRecoveryCodes(10)
	if err != nil {
		t.Fatalf("failed to generate recovery codes: %v", err)
	}
	if len(plain) != 10 || len(hashed) != 10 {
		t.Fatalf("expected 10 codes, got plain=%d hashed=%d", len(plain), len(hashed))
	}
	for i := range plain {
		if plain[i] == hashed[i] {
			t.Fatalf("recovery code should be hashed with bcrypt")
		}
	}
}

func newPostRequest(path string, body any) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest("POST", path, &buf)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func Test2FALifecycle(t *testing.T) {
	t.Setenv("LOGIN_RATE_LIMIT_ATTEMPTS", "50")
	h := setupTestHandler(t)
	router := h.Router(nil)

	// 1. Initial login without 2FA -> returns status: ok and sets apt_session cookie
	loginData := map[string]string{
		"username": "admin",
		"password": "securepassword",
	}
	req := newPostRequest("/api/auth/login", loginData)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var loginResp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&loginResp)
	if loginResp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", loginResp)
	}

	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "apt_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected apt_session cookie")
	}

	// 2. Setup 2FA
	setupReq := newPostRequest("/api/auth/2fa/setup", nil)
	setupReq.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, setupReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from setup, got %d: %s", w.Code, w.Body.String())
	}
	var setupResp struct {
		Secret        string   `json:"secret"`
		OTPAuthURL    string   `json:"otpauth_url"`
		QRCode        string   `json:"qr_code"`
		RecoveryCodes []string `json:"recovery_codes"`
	}
	_ = json.NewDecoder(w.Body).Decode(&setupResp)
	if setupResp.Secret == "" || len(setupResp.RecoveryCodes) != 10 {
		t.Fatalf("invalid setup response: %+v", setupResp)
	}

	// 3. Enable 2FA with invalid code -> should fail
	enableReq := newPostRequest("/api/auth/2fa/enable", map[string]string{"code": "000000"})
	enableReq.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, enableReq)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 with bad code, got %d", w.Code)
	}

	// 4. Enable 2FA with valid TOTP code -> should succeed
	validCode, err := GenerateTOTP(setupResp.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP: %v", err)
	}
	enableReq = newPostRequest("/api/auth/2fa/enable", map[string]string{"code": validCode})
	enableReq.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, enableReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from enable, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Next login requires 2FA -> returns status: 2fa_required & pre_auth_token
	req = newPostRequest("/api/auth/login", loginData)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var req2FAResp struct {
		Status       string `json:"status"`
		PreAuthToken string `json:"pre_auth_token"`
	}
	_ = json.NewDecoder(w.Body).Decode(&req2FAResp)
	if req2FAResp.Status != "2fa_required" || req2FAResp.PreAuthToken == "" {
		t.Fatalf("expected 2fa_required and pre_auth_token, got %+v", req2FAResp)
	}

	// 6. Verify pre_auth_token CANNOT access protected routes
	checkReq := httptest.NewRequest("GET", "/api/auth/check", nil)
	checkReq.Header.Set("Authorization", "Bearer "+req2FAResp.PreAuthToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, checkReq)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("pre_auth_token should be rejected on protected routes, got %d", w.Code)
	}

	// 7. Verify 2FA with invalid code -> should fail 401
	verifyReq := newPostRequest("/api/auth/2fa/verify", map[string]string{
		"pre_auth_token": req2FAResp.PreAuthToken,
		"code":           "999999",
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, verifyReq)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad 2FA code, got %d", w.Code)
	}

	// 8. Verify 2FA with valid TOTP code -> succeeds and sets apt_session cookie
	nowCode, _ := GenerateTOTP(setupResp.Secret, time.Now())
	verifyReq = newPostRequest("/api/auth/2fa/verify", map[string]string{
		"pre_auth_token": req2FAResp.PreAuthToken,
		"code":           nowCode,
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, verifyReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid 2FA code, got %d: %s", w.Code, w.Body.String())
	}

	hasSessionCookie := false
	for _, c := range w.Result().Cookies() {
		if c.Name == "apt_session" && c.Value != "" {
			hasSessionCookie = true
			sessionCookie = c
			break
		}
	}
	if !hasSessionCookie {
		t.Fatalf("expected session cookie after 2fa verify")
	}

	// 9. Test Recovery Code login
	req = newPostRequest("/api/auth/login", loginData)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	_ = json.NewDecoder(w.Body).Decode(&req2FAResp)

	recoveryCodeToUse := setupResp.RecoveryCodes[0]
	verifyReq = newPostRequest("/api/auth/2fa/verify", map[string]string{
		"pre_auth_token": req2FAResp.PreAuthToken,
		"code":           recoveryCodeToUse,
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, verifyReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid recovery code, got %d: %s", w.Code, w.Body.String())
	}

	// 10. Re-using the same recovery code must fail!
	req = newPostRequest("/api/auth/login", loginData)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	_ = json.NewDecoder(w.Body).Decode(&req2FAResp)

	verifyReq = newPostRequest("/api/auth/2fa/verify", map[string]string{
		"pre_auth_token": req2FAResp.PreAuthToken,
		"code":           recoveryCodeToUse,
	})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, verifyReq)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("re-used recovery code should fail with 401, got %d", w.Code)
	}

	// 11. Test Admin Reset 2FA
	user, err := h.db.GetUserByUsername("admin")
	if err != nil || user == nil {
		t.Fatalf("failed to fetch user: %v", err)
	}
	resetReq := newPostRequest("/api/users/"+user.ID+"/reset-2fa", nil)
	resetReq.AddCookie(sessionCookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, resetReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from reset-2fa, got %d", w.Code)
	}

	// 12. Login after reset should not require 2FA anymore
	req = newPostRequest("/api/auth/login", loginData)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	_ = json.NewDecoder(w.Body).Decode(&loginResp)
	if loginResp["status"] != "ok" {
		t.Fatalf("expected status ok after 2fa reset, got %v", loginResp)
	}
}

func Test2FADisableFlow(t *testing.T) {
	t.Setenv("LOGIN_RATE_LIMIT_ATTEMPTS", "50")
	h := setupTestHandler(t)
	router := h.Router(nil)

	// Login and setup 2FA
	req := newPostRequest("/api/auth/login", map[string]string{"username": "admin", "password": "securepassword"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	cookie := w.Result().Cookies()[0]

	setupReq := newPostRequest("/api/auth/2fa/setup", nil)
	setupReq.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, setupReq)
	var setupResp struct {
		Secret string `json:"secret"`
	}
	_ = json.NewDecoder(w.Body).Decode(&setupResp)

	code, _ := GenerateTOTP(setupResp.Secret, time.Now())
	enableReq := newPostRequest("/api/auth/2fa/enable", map[string]string{"code": code})
	enableReq.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, enableReq)
	if w.Code != http.StatusOK {
		t.Fatalf("enable failed: %d", w.Code)
	}

	// Disable with wrong password -> should fail
	disableReq := newPostRequest("/api/auth/2fa/disable", map[string]string{"password": "wrongpassword", "code": code})
	disableReq.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, disableReq)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", w.Code)
	}

	// Disable with wrong code -> should fail
	disableReq = newPostRequest("/api/auth/2fa/disable", map[string]string{"password": "securepassword", "code": "000000"})
	disableReq.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, disableReq)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong code, got %d", w.Code)
	}

	// Disable with correct password and code -> succeeds
	nowCode, _ := GenerateTOTP(setupResp.Secret, time.Now())
	disableReq = newPostRequest("/api/auth/2fa/disable", map[string]string{"password": "securepassword", "code": nowCode})
	disableReq.AddCookie(cookie)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, disableReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from disable, got %d", w.Code)
	}

	// Verify user is now 2FA disabled
	u, _ := h.db.GetUserByUsername("admin")
	if u.TOTPEnabled {
		t.Fatalf("expected 2FA to be disabled")
	}
}

func TestDirectCodeLogin(t *testing.T) {
	t.Setenv("LOGIN_RATE_LIMIT_ATTEMPTS", "50")
	h := setupTestHandler(t)
	router := h.Router(nil)

	// Enable 2FA directly on db
	secret, _ := GenerateTOTPSecret()
	_, hashed, _ := GenerateRecoveryCodes(5)
	hJSON, _ := json.Marshal(hashed)
	u, _ := h.db.GetUserByUsername("admin")
	_ = h.db.UpdateUser2FA(u.ID, true, secret, string(hJSON))

	// Login passing valid TOTP code in same request
	code, _ := GenerateTOTP(secret, time.Now())
	req := newPostRequest("/api/auth/login", map[string]string{
		"username": "admin",
		"password": "securepassword",
		"code":     code,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for direct code login, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", resp)
	}
}
