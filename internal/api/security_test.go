package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// setupTestHandler creates a Handler with an in-memory SQLite database
// and a dummy GPG signer for testing.
func setupTestHandler(t *testing.T) *Handler {
	db, err := storage.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory db: %v", err)
	}

	// Initialize the admin user
	hash, err := bcrypt.GenerateFromPassword([]byte("securepassword"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if _, err := db.CreateUser("admin", string(hash), "admin"); err != nil {
		t.Fatalf("failed to init admin: %v", err)
	}

	tmpDir := t.TempDir()
	signer, err := signing.LoadOrGenerate(filepath.Join(tmpDir, "repo.key"), filepath.Join(tmpDir, "repo.pub"), "Test Key", "test@example.com")
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}

	return New(db, nil, nil, signer, "super_secret_test_jwt_key", "1.0.0", nil)
}


func TestSecurityHeaders(t *testing.T) {
	h := setupTestHandler(t)
	router := h.Router(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Content-Security-Policy",
	}

	for _, header := range headers {
		if val := rr.Header().Get(header); val == "" {
			t.Errorf("Missing security header: %s", header)
		}
	}
}

func TestCSRFProtection(t *testing.T) {
	h := setupTestHandler(t)
	router := h.Router(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test 1: POST without Origin/Referer should fail
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"securepassword"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for missing CSRF headers, got %v", rr.Code)
	}

	// Test 2: POST with valid Origin should succeed
	req = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"securepassword"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for valid Origin, got %v", rr.Code)
	}

	// Test 3: POST with Bearer token should skip CSRF check
	req = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"securepassword"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer somedummytoken")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for Bearer token, got %v", rr.Code)
	}
}

func TestCookieAuth(t *testing.T) {
	h := setupTestHandler(t)
	router := h.Router(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"securepassword"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Login failed: %v", rr.Code)
	}

	cookies := rr.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "apt_session" {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatal("apt_session cookie not found")
	}

	if !sessionCookie.HttpOnly {
		t.Error("Cookie is not HttpOnly")
	}

	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Error("Cookie SameSite is not Strict")
	}

	// Check that the token is NOT in the JSON body
	var body map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&body)
	if _, hasToken := body["token"]; hasToken {
		t.Error("Token leaked in JSON response body")
	}
}

func TestRateLimiter(t *testing.T) {
	os.Setenv("LOGIN_RATE_LIMIT_ATTEMPTS", "2")
	os.Setenv("LOGIN_RATE_LIMIT_WINDOW_SECONDS", "2")
	defer os.Unsetenv("LOGIN_RATE_LIMIT_ATTEMPTS")
	defer os.Unsetenv("LOGIN_RATE_LIMIT_WINDOW_SECONDS")

	h := setupTestHandler(t)
	router := h.Router(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Attempt 1: Success
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req.RemoteAddr = "1.2.3.4:1234"
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized { // Because password is wrong
		t.Errorf("Attempt 1: Expected 401, got %v", rr.Code)
	}

	// Attempt 2: Success
	req = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req.RemoteAddr = "1.2.3.4:5678" // Different port, same IP
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Attempt 2: Expected 401, got %v", rr.Code)
	}

	// Attempt 3: Rate Limited
	req = httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req.RemoteAddr = "1.2.3.4:9012"
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Attempt 3: Expected 429 Too Many Requests, got %v", rr.Code)
	}
}

func TestGPGKeyExportDisabled(t *testing.T) {
	os.Setenv("ENABLE_PRIVATE_KEY_EXPORT", "false")
	h := setupTestHandler(t)

	// Bypass the router to test the handler directly to easily inject an admin user
	req := httptest.NewRequest("GET", "/api/system/gpg-key", nil)
	rr := httptest.NewRecorder()

	// Direct call to exportGPGKey
	h.exportGPGKey(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for disabled GPG key export, got %v", rr.Code)
	}

	// Enable and test
	os.Setenv("ENABLE_PRIVATE_KEY_EXPORT", "true")
	defer os.Unsetenv("ENABLE_PRIVATE_KEY_EXPORT")
	rr2 := httptest.NewRecorder()
	h.exportGPGKey(rr2, req)
	
	if rr2.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for enabled GPG key export, got %v", rr2.Code)
	}
}

func TestClearAuditLogExcept(t *testing.T) {
	h := setupTestHandler(t)
	db := h.db
	router := h.Router(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Perform login to get session cookie
	loginReq := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{"username":"admin","password":"securepassword"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("Origin", "http://example.com")
	loginRR := httptest.NewRecorder()
	router.ServeHTTP(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("Login failed during clear audit test: %v", loginRR.Code)
	}

	var sessionCookie *http.Cookie
	for _, c := range loginRR.Result().Cookies() {
		if c.Name == "apt_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("No apt_session cookie returned")
	}

	// Wait a moment so timestamps don't exactly collide
	time.Sleep(10 * time.Millisecond)

	// 2. Insert some dummy audit logs
	db.AddAuditEntry("user1", "admin", "create_repo", "repo", "repo created")

	// Verify we have entries (login + create_repo)
	total, err := db.CountAuditLog("")
	if err != nil || total < 2 {
		t.Fatalf("Expected at least 2 audit logs, got %v", total)
	}

	// 3. Call clear audit log endpoint with cookie and CSRF headers
	req := httptest.NewRequest("DELETE", "/api/audit", nil)
	req.AddCookie(sessionCookie)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected 204 No Content for clearAuditLog, got %v", rr.Code)
	}

	// 4. Verify we have exactly 1 entry left (the marker)
	total, err = db.CountAuditLog("")
	if err != nil || total != 1 {
		t.Fatalf("Expected exactly 1 audit log after clear, got %v", total)
	}

	// Verify it's the marker
	entries, _ := db.ListAuditLog(10, 0)
	if len(entries) != 1 || entries[0].Action != "clear_audit_log" {
		t.Errorf("Expected remaining log to be clear_audit_log marker, got: %v", entries)
	}
}

