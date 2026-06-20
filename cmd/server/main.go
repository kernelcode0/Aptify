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
	"syscall"

	"github.com/kernelcode0/aptify/internal/api"
	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/indexqueue"
	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
	"github.com/kernelcode0/aptify/internal/web"
	"golang.org/x/crypto/bcrypt"
)

// version is injected at build time via -ldflags "-X main.version=vX.Y.Z".
// It defaults to "dev" for local builds.
var version = "dev"

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
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

	// Ensure admin user exists
	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin123"
	}

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
		log.Printf("Created default admin user: %s", adminUser)
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

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		// Auto-generate a random secret so the binary is secure out of the box.
		// The tradeoff: tokens are invalidated on every restart because the secret
		// is ephemeral. Set JWT_SECRET in the environment for persistent sessions.
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			log.Fatalf("failed to generate JWT secret: %v", err)
		}
		jwtSecret = base64.StdEncoding.EncodeToString(raw)
		log.Printf("WARNING: JWT_SECRET not set — using a random secret. All tokens will be invalidated on restart.")
	}

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

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("Starting Aptify server on :%s", port)
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
