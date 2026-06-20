package main

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

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
