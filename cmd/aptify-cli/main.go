package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// version is overridden by release builds via -ldflags "-X main.version=vX.Y.Z".
var version = "v1.0.8"

// Config is stored at ~/.config/aptify/config.json.
type Config struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

func defaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "aptify", "config.json")
}

func loadConfig(path string) (*Config, error) {
	/* #nosec G304 */
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	return &c, json.Unmarshal(data, &c)
}

func saveConfig(path string, c *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// commonFlags registers --server, --token, --config on a FlagSet.
type commonOpts struct {
	server     string
	token      string
	configPath string
}

func registerCommon(fs *flag.FlagSet) *commonOpts {
	o := &commonOpts{}
	fs.StringVar(&o.server, "server", "", "Override server URL")
	fs.StringVar(&o.token, "token", "", "Override API token")
	fs.StringVar(&o.configPath, "config", defaultConfigPath(), "Config file path")
	return o
}

// resolve loads the config and merges flag overrides, then validates.
func (o *commonOpts) resolve() (*Config, error) {
	cfg, err := loadConfig(o.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &Config{}
		} else {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	if o.server != "" {
		cfg.Server = o.server
	}
	if cfg.Server != "" && !strings.HasPrefix(cfg.Server, "http://") && !strings.HasPrefix(cfg.Server, "https://") {
		cfg.Server = "https://" + cfg.Server
	}
	if o.token != "" {
		cfg.Token = o.token
	}
	return cfg, nil
}

func (o *commonOpts) requireConfig() (*Config, error) {
	cfg, err := o.resolve()
	if err != nil {
		return nil, err
	}
	if cfg.Server == "" || cfg.Token == "" {
		return nil, fmt.Errorf("no config found. Run: aptify-cli login <server-url>")
	}
	return cfg, nil
}

// apiDo performs an authenticated JSON request and returns the response body.
func apiDo(method, url, token string, body io.Reader, contentType string) (*http.Response, error) {
	return apiDoWithClient(http.DefaultClient, method, url, token, body, contentType, "")
}

// apiDoWithClient supports the short-lived cookie session used during login.
// origin is only needed for cookie-authenticated state-changing requests.
func apiDoWithClient(client *http.Client, method, url, token string, body io.Reader, contentType, origin string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	return client.Do(req)
}

func jsonBody(v any) (io.Reader, string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	return bytes.NewReader(data), "application/json", nil
}

func errBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(b, &e) == nil && e.Error != "" {
		return e.Error
	}
	return strings.TrimSpace(string(b))
}

func normalizeServerURL(serverURL string) string {
	serverURL = strings.TrimRight(strings.TrimSpace(serverURL), "/")
	if serverURL != "" && !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}
	return serverURL
}

// existingLoginValid checks whether the config already contains a working API
// key for this server. A rejected key permits a fresh login; connectivity and
// unexpected server errors are returned so login cannot accidentally mint a
// duplicate key when validation was inconclusive.
func existingLoginValid(client *http.Client, configPath, serverURL string) (bool, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read config: %w", err)
	}
	if normalizeServerURL(cfg.Server) != normalizeServerURL(serverURL) || cfg.Token == "" {
		return false, nil
	}

	resp, err := apiDoWithClient(client, http.MethodGet, normalizeServerURL(serverURL)+"/api/auth/check", cfg.Token, nil, "", "")
	if err != nil {
		return false, fmt.Errorf("verify saved API key: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return false, nil
	}
	return false, fmt.Errorf("verify saved API key: server returned %s: %s", resp.Status, errBody(resp))
}

// ---- commands ---------------------------------------------------------------

func runLogin(args []string) {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	opts := registerCommon(fs)
	_ = fs.Parse(args)

	serverURL := fs.Arg(0)
	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "usage: aptify-cli login <server-url>")
		os.Exit(1)
	}
	serverURL = normalizeServerURL(serverURL)

	alreadyLoggedIn, err := existingLoginValid(http.DefaultClient, opts.configPath, serverURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login check failed:", err)
		os.Exit(1)
	}
	if alreadyLoggedIn {
		fmt.Printf("Already logged in to %s. Using the saved API key.\n", serverURL)
		return
	}

	fmt.Print("Username: ")
	var username string
	_, _ = fmt.Scanln(&username)

	fmt.Print("Password: ")
	pwBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading password:", err)
		os.Exit(1)
	}
	password := string(pwBytes)

	// 1. Log in with a short-lived cookie session.
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "initialize login session:", err)
		os.Exit(1)
	}
	client := &http.Client{Jar: jar}
	body, ct, _ := jsonBody(map[string]string{"username": username, "password": password})
	resp, err := apiDoWithClient(client, "POST", serverURL+"/api/auth/login", "", body, ct, serverURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "login failed: %s\n", errBody(resp))
		os.Exit(1)
	}
	_, _ = io.Copy(io.Discard, resp.Body)

	// 2. Create an API key using the cookie session.
	hostname, _ := os.Hostname()
	keyName := "cli-" + hostname
	body2, ct2, _ := jsonBody(map[string]string{"name": keyName})
	resp2, err := apiDoWithClient(client, "POST", serverURL+"/api/auth/keys", "", body2, ct2, serverURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create api key failed:", err)
		os.Exit(1)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		fmt.Fprintf(os.Stderr, "create api key failed: %s\n", errBody(resp2))
		os.Exit(1)
	}
	var keyResp struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&keyResp); err != nil {
		fmt.Fprintln(os.Stderr, "decode key response:", err)
		os.Exit(1)
	}

	// 3. Save API key (not the JWT) to config.
	cfg := &Config{Server: serverURL, Token: keyResp.Key}
	configPath := opts.configPath
	if err := saveConfig(configPath, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "save config:", err)
		os.Exit(1)
	}
	fmt.Printf("Logged in to %s. API key saved.\n", serverURL)
}

func runPush(args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	opts := registerCommon(fs)
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: aptify-cli push <repo-slug> <file> [<file> ...]")
		os.Exit(1)
	}
	slug := fs.Arg(0)
	patterns := fs.Args()[1:]

	cfg, err := opts.requireConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Resolve repo slug → ID.
	resp, err := apiDo("GET", cfg.Server+"/api/repos", cfg.Token, nil, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "list repos:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "list repos failed: %s\n", errBody(resp))
		os.Exit(1)
	}
	var repos []struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		fmt.Fprintln(os.Stderr, "decode repos:", err)
		os.Exit(1)
	}
	var repoID string
	for _, r := range repos {
		if r.Slug == slug {
			repoID = r.ID
			break
		}
	}
	if repoID == "" {
		fmt.Fprintf(os.Stderr, "repo %q not found\n", slug)
		os.Exit(1)
	}

	// Expand glob patterns.
	var files []string
	for _, p := range patterns {
		if strings.ContainsAny(p, "*?[") {
			matches, err := filepath.Glob(p)
			if err != nil {
				fmt.Fprintln(os.Stderr, "glob error:", err)
				os.Exit(1)
			}
			files = append(files, matches...)
		} else {
			files = append(files, p)
		}
	}
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "no files matched")
		os.Exit(1)
	}

	exitCode := 0
	for _, filePath := range files {
		name := filepath.Base(filePath)
		fmt.Printf("Pushing %s ... ", name)

		status, msg, err := uploadDeb(cfg.Server, repoID, cfg.Token, filePath)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			exitCode = 1
			continue
		}
		switch status {
		case http.StatusCreated:
			fmt.Println("done (201)")
		case http.StatusConflict:
			fmt.Println("already exists (409)")
		default:
			fmt.Printf("error %d: %s\n", status, msg)
			exitCode = 1
		}
	}
	os.Exit(exitCode)
}

func uploadDeb(serverURL, repoID, token, filePath string) (int, string, error) {
	/* #nosec G304 */
	f, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("file", filepath.Base(filePath))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(mw.Close())
	}()

	url := serverURL + "/api/repos/" + repoID + "/packages"
	req, err := http.NewRequest("POST", url, pr)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), nil
}

func runRepos(args []string) {
	fs := flag.NewFlagSet("repos", flag.ExitOnError)
	opts := registerCommon(fs)
	_ = fs.Parse(args)

	cfg, err := opts.requireConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	resp, err := apiDo("GET", cfg.Server+"/api/repos", cfg.Token, nil, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "list repos:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "list repos failed: %s\n", errBody(resp))
		os.Exit(1)
	}
	var repos []struct {
		Slug     string `json:"slug"`
		Name     string `json:"name"`
		Codename string `json:"codename"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		fmt.Fprintln(os.Stderr, "decode repos:", err)
		os.Exit(1)
	}

	fmt.Printf("%-20s %-30s %s\n", "SLUG", "NAME", "CODENAME")
	for _, r := range repos {
		fmt.Printf("%-20s %-30s %s\n", r.Slug, r.Name, r.Codename)
	}
}

func runWhoami(args []string) {
	fs := flag.NewFlagSet("whoami", flag.ExitOnError)
	opts := registerCommon(fs)
	_ = fs.Parse(args)

	cfg, err := opts.requireConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	resp, err := apiDo("GET", cfg.Server+"/api/auth/check", cfg.Token, nil, "")
	if err != nil || resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Not logged in or token expired. Run: aptify-cli login <server>\n")
		os.Exit(1)
	}
	_ = resp.Body.Close()
	fmt.Printf("Server: %s\nToken:  valid\n", cfg.Server)
}

func runLogout(args []string) {
	fs := flag.NewFlagSet("logout", flag.ExitOnError)
	opts := registerCommon(fs)
	_ = fs.Parse(args)

	if err := os.Remove(opts.configPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "logout:", err)
		os.Exit(1)
	}
	fmt.Println("Logged out.")
}

// ---- main -------------------------------------------------------------------

func usage() {
	fmt.Fprintf(os.Stderr, `aptify-cli %s — Aptify command-line client

Commands:
  login <server-url>             Authenticate and save an API key
  push  <slug> <file> [...]      Upload packages to a repository
  repos                          List repositories
  whoami                         Check authentication status
  logout                         Remove saved credentials

Flags (all commands):
  --server  <url>    Override config file server
  --token   <key>    Override config file token
  --config  <path>   Override config file path (default: ~/.config/aptify/config.json)
`, version)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	rest := os.Args[2:]

	switch cmd {
	case "login":
		runLogin(rest)
	case "push":
		runPush(rest)
	case "repos":
		runRepos(rest)
	case "whoami":
		runWhoami(rest)
	case "logout":
		runLogout(rest)
	case "--help", "-h", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		usage()
		os.Exit(1)
	}
}
