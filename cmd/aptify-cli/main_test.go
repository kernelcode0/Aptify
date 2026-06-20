package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	aptifyapi "github.com/kernelcode0/aptify/internal/api"
	"github.com/kernelcode0/aptify/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type handlerTransport struct{ handler http.Handler }

func (t handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	t.handler.ServeHTTP(recorder, r)
	return recorder.Result(), nil
}

func TestLoginClientCarriesSessionAndOrigin(t *testing.T) {
	const origin = "http://aptify.example"
	var sawSession bool
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Origin") != origin {
			t.Errorf("Origin = %q, want %q", r.Header.Get("Origin"), origin)
		}
		response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}
		switch r.URL.Path {
		case "/api/auth/login":
			response.Header.Add("Set-Cookie", "apt_session=session; Path=/")
		case "/api/auth/keys":
			cookie, err := r.Cookie("apt_session")
			sawSession = err == nil && cookie.Value == "session"
		}
		return response, nil
	})

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Transport: transport}

	login, err := apiDoWithClient(client, http.MethodPost, origin+"/api/auth/login", "", strings.NewReader(`{}`), "application/json", origin)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, login.Body)
	_ = login.Body.Close()

	key, err := apiDoWithClient(client, http.MethodPost, origin+"/api/auth/keys", "", strings.NewReader(`{}`), "application/json", origin)
	if err != nil {
		t.Fatal(err)
	}
	_ = key.Body.Close()
	if !sawSession {
		t.Fatal("API-key request did not carry the login session cookie")
	}
}

func TestExistingLoginValidReusesSavedAPIKey(t *testing.T) {
	const serverURL = "https://aptify.example"
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := saveConfig(configPath, &Config{Server: serverURL + "/", Token: "aptify_saved"}); err != nil {
		t.Fatal(err)
	}

	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != serverURL+"/api/auth/check" {
			t.Errorf("URL = %q", r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer aptify_saved" {
			t.Errorf("Authorization = %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"valid":true}`))}, nil
	})}

	valid, err := existingLoginValid(client, configPath, serverURL)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("expected saved API key to be valid")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestExistingLoginValidAllowsRejectedKeyToBeReplaced(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := saveConfig(configPath, &Config{Server: "https://aptify.example", Token: "expired"}); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusUnauthorized, Status: "401 Unauthorized", Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"error":"Unauthorized"}`))}, nil
	})}

	valid, err := existingLoginValid(client, configPath, "https://aptify.example")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Fatal("expected rejected API key to require fresh login")
	}
}

func TestLoginCreatesOneReusableAPIKey(t *testing.T) {
	const serverURL = "https://aptify.example"
	db, err := storage.Open("sqlite", filepath.Join(t.TempDir(), "aptify.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	hash, err := bcrypt.GenerateFromPassword([]byte("securepassword"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := db.CreateUser("wajahat.ali", string(hash), "admin")
	if err != nil {
		t.Fatal(err)
	}

	handler := aptifyapi.New(db, nil, nil, nil, "integration_test_jwt_secret_32_chars", "v1.0.9", nil)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Transport: handlerTransport{handler: handler.Router(http.NotFoundHandler())}}

	loginBody, contentType, err := jsonBody(map[string]string{"username": "wajahat.ali", "password": "securepassword"})
	if err != nil {
		t.Fatal(err)
	}
	login, err := apiDoWithClient(client, http.MethodPost, serverURL+"/api/auth/login", "", loginBody, contentType, serverURL)
	if err != nil {
		t.Fatal(err)
	}
	defer login.Body.Close()
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", login.StatusCode, errBody(login))
	}
	_, _ = io.Copy(io.Discard, login.Body)

	keyBody, contentType, err := jsonBody(map[string]string{"name": "integration-cli"})
	if err != nil {
		t.Fatal(err)
	}
	keyResponse, err := apiDoWithClient(client, http.MethodPost, serverURL+"/api/auth/keys", "", keyBody, contentType, serverURL)
	if err != nil {
		t.Fatal(err)
	}
	defer keyResponse.Body.Close()
	if keyResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create key status = %d, body = %s", keyResponse.StatusCode, errBody(keyResponse))
	}
	var created struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(keyResponse.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := saveConfig(configPath, &Config{Server: serverURL, Token: created.Key}); err != nil {
		t.Fatal(err)
	}
	valid, err := existingLoginValid(client, configPath, serverURL)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("newly issued API key was not reusable")
	}

	keys, err := db.ListAPIKeys(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 {
		t.Fatalf("API key count = %d, want 1", len(keys))
	}
}
