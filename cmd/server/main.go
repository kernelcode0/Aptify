package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kernelcode0/aptify/internal/api"
	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/indexqueue"
	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
	"github.com/kernelcode0/aptify/internal/web"
	"golang.org/x/crypto/bcrypt"
)

// version is injected at build time via -ldflags "-X main.version=vX.Y.Z".
// It defaults to "1.0.0" for local builds.
var version = "1.0.0"

// knownWeakPasswords is an explicit blocklist of credentials that must never
// be accepted in production. Fail fast if an operator forgets to rotate them.
var knownWeakPasswords = map[string]bool{
	"admin123":            true,
	"password":            true,
	"changeme":            true,
	"secret":              true,
	"letmein":             true,
	"admin":               true,
	"123456":              true,
	"aptifypass":          true,
	"rootpass":            true,
	"supersecretvalue123": true,
}

// knownWeakJWTSecrets lists default/example JWT secrets that must be rejected.
var knownWeakJWTSecrets = map[string]bool{
	"supersecretvalue123":               true,
	"super_secret_jwt_string_change_me": true,
	"secret":                            true,
	"changeme":                          true,
}

// validateSecrets checks that all required secrets are set, sufficiently long,
// and not known defaults. It calls log.Fatal on the first violation so the
// process exits before any network listener is opened.
func validateSecrets(jwtSecret, adminPass string) {
	// --- JWT_SECRET ---
	if jwtSecret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable must be set. " +
			"Generate one with: openssl rand -hex 32")
	}
	if len(jwtSecret) < 32 {
		log.Fatal("FATAL: JWT_SECRET must be at least 32 characters long. " +
			"Generate one with: openssl rand -hex 32")
	}
	if knownWeakJWTSecrets[jwtSecret] {
		log.Fatal("FATAL: JWT_SECRET is a known insecure default value. " +
			"Generate a secure secret with: openssl rand -hex 32")
	}

	// --- ADMIN_PASSWORD ---
	if adminPass == "" {
		log.Fatal("FATAL: ADMIN_PASSWORD environment variable must be set.")
	}
	if len(adminPass) < 8 {
		log.Fatal("FATAL: ADMIN_PASSWORD must be at least 8 characters long.")
	}
	if knownWeakPasswords[strings.ToLower(adminPass)] {
		log.Fatalf("FATAL: ADMIN_PASSWORD '%s' is a known insecure default value. "+
			"Choose a strong, unique password.", adminPass)
	}
}

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0750); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	dbType := os.Getenv("DB_TYPE")
	dbDsn := os.Getenv("DB_DSN")
	if dbType == "" || dbType == "sqlite" {
		dbType = "sqlite"
		if dbDsn == "" {
			dbDsn = filepath.Join(dataDir, "aptify.sqlite")
		}
	}

	db, err := storage.Open(dbType, dbDsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// --- Secret validation — fail fast before any further initialisation ---
	jwtSecret := os.Getenv("JWT_SECRET")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	// In development mode (APTIFY_DEV=1) we allow an ephemeral JWT secret to
	// be auto-generated for local convenience. All other validations still apply.
	devMode := os.Getenv("APTIFY_DEV") == "1"

	if jwtSecret == "" && devMode {
		// Auto-generate a random secret for ephemeral dev sessions only.
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			log.Fatalf("failed to generate JWT secret: %v", err)
		}
		jwtSecret = base64.StdEncoding.EncodeToString(raw)
		log.Printf("WARNING: JWT_SECRET not set — using ephemeral random secret (APTIFY_DEV=1). " +
			"All tokens will be invalidated on restart. Do NOT use this in production.")
	}

	// ADMIN_PASSWORD must always be explicitly set — even in dev mode.
	if adminPass == "" && devMode {
		log.Fatal("FATAL: ADMIN_PASSWORD must be set even in dev mode. " +
			"Set ADMIN_PASSWORD=your-local-dev-password")
	}

	if !devMode {
		validateSecrets(jwtSecret, adminPass)
	} else if adminPass != "" && knownWeakPasswords[strings.ToLower(adminPass)] {
		log.Printf("WARNING: ADMIN_PASSWORD is a known weak value. Do not use this in production.")
	}

	// Ensure admin user exists
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminUser = strings.ReplaceAll(strings.ReplaceAll(adminUser, "\n", ""), "\r", "")

	u, err := db.GetUserByUsername(adminUser)
	if err != nil {
		log.Fatalf("failed to get admin user: %v", err)
	}
	if u == nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}
		if _, err := db.CreateUser(adminUser, string(hash), "admin"); err != nil {
			log.Fatalf("failed to create admin user: %v", err)
		}
		log.Printf("Created default admin user: %s", adminUser) // #nosec G706
	}

	fs, err := storage.NewFileStore(dataDir)
	if err != nil {
		log.Fatalf("failed to init filestore: %v", err)
	}

	keyName := os.Getenv("KEY_NAME")
	if keyName == "" {
		keyName = "Aptify Repository"
	}
	keyEmail := os.Getenv("KEY_EMAIL")
	if keyEmail == "" {
		keyEmail = "admin@aptify.local"
	}

	keyPath := filepath.Join(dataDir, "repo.key")
	pubPath := filepath.Join(dataDir, "repo.pub")
	signer, err := signing.LoadOrGenerate(keyPath, pubPath, keyName, keyEmail)
	if err != nil {
		log.Fatalf("failed to load/generate GPG key: %v", err)
	}

	gen := index.New(fs, signer)

	// Context cancelled on SIGINT/SIGTERM so the queue worker shuts down cleanly.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	queue := indexqueue.New(gen, db)
	queue.Start(ctx)

	handler := api.New(db, fs, gen, signer, jwtSecret, version, queue)

	spa := web.Handler()
	r := handler.Router(spa)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	port = strings.ReplaceAll(strings.ReplaceAll(port, "\n", ""), "\r", "")

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
		// ReadHeaderTimeout guards against Slow-Loris header attacks.
		ReadHeaderTimeout: 5 * time.Second,
		// ReadTimeout covers the entire request body (e.g. large uploads).
		ReadTimeout: 60 * time.Second,
		// WriteTimeout is high to accommodate 512 MiB package uploads on slow links.
		WriteTimeout: 300 * time.Second,
		// IdleTimeout caps keep-alive idle connections.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MiB
	}

	go func() {
		log.Printf("Starting Aptify server on :%s", port) // #nosec G706
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
	if err := srv.Close(); err != nil {
		log.Printf("server close error: %v", err)
	}
}
