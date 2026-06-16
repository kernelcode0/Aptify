package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kernelcode0/aptify/internal/api"
	"github.com/kernelcode0/aptify/internal/index"
	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
	"github.com/kernelcode0/aptify/internal/web"
	"golang.org/x/crypto/bcrypt"
)

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
		if _, err := db.CreateUser(adminUser, string(hash)); err != nil {
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
		jwtSecret = "super_secret_jwt_string_change_me" // Fallback if missing
	}

	handler := api.New(db, fs, gen, signer, jwtSecret)

	spa := web.Handler()
	r := handler.Router(spa)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Aptify server on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
