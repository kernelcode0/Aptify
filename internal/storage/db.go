package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// DB wraps a database connection.
type DB struct {
	db     *sql.DB
	dbType string
}

// APIKey represents an API key record.
type APIKey struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used"`
	KeyHash   string     `json:"-"` // only populated for auth lookups
}

// User represents an administrator user.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditEntry represents one immutable audit log record.
type AuditEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

// Repo represents a repository record.
type Repo struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Codename  string    `json:"codename"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

// Package represents a package record.
type Package struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Filename    string    `json:"filename"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
	Release     string    `json:"release,omitempty"`
	Arch        string    `json:"arch"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	SHA1        string    `json:"sha1"`
	MD5         string    `json:"md5"`
	ControlJSON string    `json:"-"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

func Open(dbType, dsn string) (*DB, error) {
	if dbType == "" {
		dbType = "sqlite"
	}
	var db *sql.DB
	var err error
	if dbType == "sqlite" {
		db, err = sql.Open("sqlite", dsn+"?_journal=WAL&_busy_timeout=5000")
	} else if dbType == "mysql" {
		db, err = sql.Open("mysql", dsn)
	} else {
		return nil, fmt.Errorf("unsupported dbType: %s", dbType)
	}
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if dbType == "sqlite" {
		db.SetMaxOpenConns(1)
		if _, err := db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
		}
	}
	d := &DB{db: db, dbType: dbType}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate() error {
	var repoTable, pkgTable, userTable, apiKeyTable, auditLogTable string
	if d.dbType == "mysql" {
		userTable = `
		CREATE TABLE IF NOT EXISTS users (
			id            VARCHAR(36) PRIMARY KEY,
			username      VARCHAR(255) UNIQUE NOT NULL,
			role          VARCHAR(32) NOT NULL DEFAULT 'viewer',
			password_hash VARCHAR(255) NOT NULL,
			created_at    DATETIME NOT NULL
		);`
		repoTable = `
		CREATE TABLE IF NOT EXISTS repos (
			id          VARCHAR(36) PRIMARY KEY,
			slug        VARCHAR(255) UNIQUE NOT NULL,
			name        VARCHAR(255) NOT NULL,
			codename    VARCHAR(255) NOT NULL DEFAULT 'stable',
			type        VARCHAR(16) NOT NULL DEFAULT 'deb',
			created_at  DATETIME NOT NULL
		);`
		pkgTable = `
		CREATE TABLE IF NOT EXISTS packages (
			id           VARCHAR(36) PRIMARY KEY,
			repo_id      VARCHAR(36) NOT NULL,
			filename     VARCHAR(255) NOT NULL,
			package      VARCHAR(255) NOT NULL,
			version      VARCHAR(255) NOT NULL,
			` + "`release`" + `      VARCHAR(255) NOT NULL DEFAULT '',
			` + "`arch`" + `         VARCHAR(255) NOT NULL,
			size         BIGINT NOT NULL,
			sha256       VARCHAR(64) NOT NULL,
			sha1         VARCHAR(40) NOT NULL,
			md5          VARCHAR(32) NOT NULL,
			control_json TEXT NOT NULL,
			uploaded_at  DATETIME NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE,
			INDEX idx_packages_repo (repo_id)
		);`
		apiKeyTable = `
		CREATE TABLE IF NOT EXISTS api_keys (
			id          VARCHAR(36) PRIMARY KEY,
			user_id     VARCHAR(36) NOT NULL,
			name        VARCHAR(255) NOT NULL,
			key_hash    VARCHAR(255) NOT NULL UNIQUE,
			prefix      VARCHAR(8) NOT NULL,
			created_at  DATETIME NOT NULL,
			last_used   DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);`
		auditLogTable = `
		CREATE TABLE IF NOT EXISTS audit_log (
			id          VARCHAR(36) PRIMARY KEY,
			user_id     VARCHAR(36) NOT NULL,
			username    VARCHAR(255) NOT NULL,
			action      VARCHAR(64) NOT NULL,
			resource    VARCHAR(255) NOT NULL,
			detail      TEXT,
			created_at  DATETIME NOT NULL,
			INDEX idx_audit_created (created_at),
			INDEX idx_audit_user (user_id)
		);`
	} else {
		userTable = `
		CREATE TABLE IF NOT EXISTS users (
			id            TEXT PRIMARY KEY,
			username      TEXT UNIQUE NOT NULL,
			role          TEXT NOT NULL DEFAULT 'viewer',
			password_hash TEXT NOT NULL,
			created_at    DATETIME NOT NULL
		);`
		repoTable = `
		CREATE TABLE IF NOT EXISTS repos (
			id          TEXT PRIMARY KEY,
			slug        TEXT UNIQUE NOT NULL,
			name        TEXT NOT NULL,
			codename    TEXT NOT NULL DEFAULT 'stable',
			type        TEXT NOT NULL DEFAULT 'deb',
			created_at  DATETIME NOT NULL
		);`
		pkgTable = `
		CREATE TABLE IF NOT EXISTS packages (
			id           TEXT PRIMARY KEY,
			repo_id      TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
			filename     TEXT NOT NULL,
			package      TEXT NOT NULL,
			version      TEXT NOT NULL,
			release      TEXT NOT NULL DEFAULT '',
			arch         TEXT NOT NULL,
			size         INTEGER NOT NULL,
			sha256       TEXT NOT NULL,
			sha1         TEXT NOT NULL,
			md5          TEXT NOT NULL,
			control_json TEXT NOT NULL DEFAULT '{}',
			uploaded_at  DATETIME NOT NULL
		);`
		apiKeyTable = `
		CREATE TABLE IF NOT EXISTS api_keys (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name        TEXT NOT NULL,
			key_hash    TEXT NOT NULL UNIQUE,
			prefix      TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			last_used   DATETIME
		);`
		auditLogTable = `
		CREATE TABLE IF NOT EXISTS audit_log (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			username    TEXT NOT NULL,
			action      TEXT NOT NULL,
			resource    TEXT NOT NULL,
			detail      TEXT,
			created_at  DATETIME NOT NULL
		);`
	}

	if _, err := d.db.Exec(userTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(repoTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(pkgTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(apiKeyTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(auditLogTable); err != nil {
		return err
	}
	if err := d.ensureUserRoleColumn(); err != nil {
		return err
	}
	if err := d.ensureRepoTypeColumn(); err != nil {
		return err
	}
	if err := d.ensurePackageReleaseColumn(); err != nil {
		return err
	}
	if d.dbType == "sqlite" {
		if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_packages_repo ON packages(repo_id);`); err != nil {
			return err
		}
		if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at);`); err != nil {
			return err
		}
		if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_log(user_id);`); err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ensureRepoTypeColumn() error {
	exists, err := d.columnExists("repos", "type")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	colType := "TEXT"
	if d.dbType == "mysql" {
		colType = "VARCHAR(16)"
	}
	_, err = d.db.Exec(`ALTER TABLE repos ADD COLUMN type ` + colType + ` NOT NULL DEFAULT 'deb'`)
	return err
}

func (d *DB) ensurePackageReleaseColumn() error {
	exists, err := d.columnExists("packages", "release")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if d.dbType == "mysql" {
		_, err = d.db.Exec("ALTER TABLE packages ADD COLUMN `release` VARCHAR(255) NOT NULL DEFAULT ''")
	} else {
		_, err = d.db.Exec("ALTER TABLE packages ADD COLUMN release TEXT NOT NULL DEFAULT ''")
	}
	return err
}

func (d *DB) ensureUserRoleColumn() error {
	exists, err := d.columnExists("users", "role")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	roleType := "TEXT"
	if d.dbType == "mysql" {
		roleType = "VARCHAR(32)"
	}
	if _, err := d.db.Exec(`ALTER TABLE users ADD COLUMN role ` + roleType + ` NOT NULL DEFAULT 'viewer'`); err != nil {
		return err
	}
	_, err = d.db.Exec(`UPDATE users SET role=?`, "admin")
	return err
}

func (d *DB) columnExists(table, column string) (bool, error) {
	// Whitelist table names to prevent SQL injection via string concatenation
	validTables := map[string]bool{"users": true, "repos": true, "packages": true, "api_keys": true, "audit_log": true}
	if !validTables[table] {
		return false, fmt.Errorf("invalid table name")
	}

	var rows *sql.Rows
	var err error
	if d.dbType == "mysql" {
		rows, err = d.db.Query("SHOW COLUMNS FROM "+table+" WHERE Field = ?", column) // #nosec G202
	} else {
		rows, err = d.db.Query(`PRAGMA table_info(` + table + `)`) // #nosec G202
	}
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if d.dbType == "mysql" {
		return rows.Next(), rows.Err()
	}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// GetUserByUsername retrieves a user by their username.
func (d *DB) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := d.db.QueryRow(
		`SELECT id, username, role, password_hash, created_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// CreateUser inserts a new user.
func (d *DB) CreateUser(username, passwordHash, role string) (*User, error) {
	u := &User{
		ID:           uuid.NewString(),
		Username:     username,
		Role:         role,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	_, err := d.db.Exec(
		`INSERT INTO users (id, username, role, password_hash, created_at) VALUES (?,?,?,?,?)`,
		u.ID, u.Username, u.Role, u.PasswordHash, u.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// ListUsers returns all users ordered by creation time.
func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.db.Query(`SELECT id, username, role, created_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserByID retrieves a user by ID.
func (d *DB) GetUserByID(id string) (*User, error) {
	u := &User{}
	err := d.db.QueryRow(
		`SELECT id, username, role, password_hash, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.Role, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// UpdateUserRole updates a user's role.
func (d *DB) UpdateUserRole(id, role string) error {
	_, err := d.db.Exec(`UPDATE users SET role=? WHERE id=?`, role, id)
	return err
}

// UpdateUserPassword updates a user's password hash.
func (d *DB) UpdateUserPassword(id, passwordHash string) error {
	_, err := d.db.Exec(`UPDATE users SET password_hash=? WHERE id=?`, passwordHash, id)
	return err
}

// DeleteUser removes a user and cascades their API keys.
func (d *DB) DeleteUser(id string) error {
	_, err := d.db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

// CreateRepo inserts a new repo and returns it.
func (d *DB) CreateRepo(slug, name, codename, repoType string) (*Repo, error) {
	r := &Repo{
		ID:        uuid.NewString(),
		Slug:      slug,
		Name:      name,
		Codename:  codename,
		Type:      repoType,
		CreatedAt: time.Now().UTC(),
	}
	_, err := d.db.Exec(
		`INSERT INTO repos (id, slug, name, codename, type, created_at) VALUES (?,?,?,?,?,?)`,
		r.ID, r.Slug, r.Name, r.Codename, r.Type, r.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create repo: %w", err)
	}
	return r, nil
}

// GetRepo fetches a repo by ID.
func (d *DB) GetRepo(id string) (*Repo, error) {
	r := &Repo{}
	err := d.db.QueryRow(
		`SELECT id, slug, name, codename, type, created_at FROM repos WHERE id=?`, id,
	).Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.Type, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetRepoBySlug fetches a repo by slug.
func (d *DB) GetRepoBySlug(slug string) (*Repo, error) {
	r := &Repo{}
	err := d.db.QueryRow(
		`SELECT id, slug, name, codename, type, created_at FROM repos WHERE slug=?`, slug,
	).Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.Type, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// ListRepos returns all repos ordered by creation time.
func (d *DB) ListRepos() ([]Repo, error) {
	rows, err := d.db.Query(`SELECT id, slug, name, codename, type, created_at FROM repos ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.Type, &r.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

// DeleteRepo removes a repo (packages cascade via FK).
func (d *DB) DeleteRepo(id string) error {
	_, err := d.db.Exec(`DELETE FROM repos WHERE id=?`, id)
	return err
}

// UpdateRepo updates the repo's name and codename.
func (d *DB) UpdateRepo(id, name, codename string) error {
	_, err := d.db.Exec(`UPDATE repos SET name=?, codename=? WHERE id=?`, name, codename, id)
	return err
}

// GetRepoArchitectures fetches unique architectures for a repo.
func (d *DB) GetRepoArchitectures(repoID string) ([]string, error) {
	rows, err := d.db.Query(`SELECT DISTINCT arch FROM packages WHERE repo_id=?`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var arches []string
	for rows.Next() {
		var arch string
		if err := rows.Scan(&arch); err != nil {
			return nil, err
		}
		arches = append(arches, arch)
	}
	return arches, rows.Err()
}

// AddPackage inserts a package record.
func (d *DB) AddPackage(p *Package) error {
	p.ID = uuid.NewString()
	p.UploadedAt = time.Now().UTC()
	_, err := d.db.Exec(
		`INSERT INTO packages (id, repo_id, filename, package, version, `+"`release`"+`, arch, size, sha256, sha1, md5, control_json, uploaded_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.RepoID, p.Filename, p.Package, p.Version, p.Release, p.Arch,
		p.Size, p.SHA256, p.SHA1, p.MD5, p.ControlJSON, p.UploadedAt,
	)
	return err
}

// ListPackages returns all packages for a repo. If limit > 0, it applies pagination.
func (d *DB) ListPackages(repoID string, limit, offset int) ([]Package, error) {
	query := `SELECT id, repo_id, filename, package, version, ` + "`release`" + `, arch, size, sha256, sha1, md5, control_json, uploaded_at
		 FROM packages WHERE repo_id=? ORDER BY uploaded_at DESC`

	var rows *sql.Rows
	var err error
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		rows, err = d.db.Query(query, repoID, limit, offset)
	} else {
		rows, err = d.db.Query(query, repoID)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pkgs []Package
	for rows.Next() {
		var p Package
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Filename, &p.Package, &p.Version, &p.Release, &p.Arch,
			&p.Size, &p.SHA256, &p.SHA1, &p.MD5, &p.ControlJSON, &p.UploadedAt); err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, rows.Err()
}

// CountPackages returns the total number of packages for a repo.
func (d *DB) CountPackages(repoID string) (int, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM packages WHERE repo_id=?`, repoID).Scan(&count)
	return count, err
}

// GetPackage fetches a single package by ID.
func (d *DB) GetPackage(id string) (*Package, error) {
	p := &Package{}
	err := d.db.QueryRow(
		`SELECT id, repo_id, filename, package, version, `+"`release`"+`, arch, size, sha256, sha1, md5, control_json, uploaded_at
		 FROM packages WHERE id=?`, id,
	).Scan(&p.ID, &p.RepoID, &p.Filename, &p.Package, &p.Version, &p.Release, &p.Arch,
		&p.Size, &p.SHA256, &p.SHA1, &p.MD5, &p.ControlJSON, &p.UploadedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// DeletePackage removes a package record by ID.
func (d *DB) DeletePackage(id string) error {
	_, err := d.db.Exec(`DELETE FROM packages WHERE id=?`, id)
	return err
}

// Ping checks that the database connection is alive.
func (d *DB) Ping() error {
	_, err := d.db.Exec(`SELECT 1`)
	return err
}

// PackageFilenameExists checks whether a package with the same filename already exists in the given repo.
func (d *DB) PackageFilenameExists(repoID, filename string) (string, bool, error) {
	var id string
	err := d.db.QueryRow(
		`SELECT id FROM packages WHERE repo_id=? AND filename=? LIMIT 1`,
		repoID, filename,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (d *DB) Close() error { return d.db.Close() }

// ClearAuditLogExcept deletes all audit logs except the provided eventID.
// This is an Option B forensic marker mechanism.
func (d *DB) ClearAuditLogExcept(ctx context.Context, keepEventID string) error {
	_, err := d.db.ExecContext(ctx, `DELETE FROM audit_log WHERE id != ?`, keepEventID)
	return err
}

func (d *DB) LogAuditWithID(userID, username, action, resource, detail string) (string, error) {
	id := uuid.NewString()
	_, err := d.db.Exec(`
		INSERT INTO audit_log (id, user_id, username, action, resource, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, userID, username, action, resource, detail, time.Now().UTC())
	return id, err
}

// CreateAPIKey inserts a new API key record.
func (d *DB) CreateAPIKey(userID, name, keyHash, prefix string) (*APIKey, error) {
	k := &APIKey{
		ID:        uuid.NewString(),
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		CreatedAt: time.Now().UTC(),
	}
	_, err := d.db.Exec(
		`INSERT INTO api_keys (id, user_id, name, key_hash, prefix, created_at) VALUES (?,?,?,?,?,?)`,
		k.ID, k.UserID, k.Name, keyHash, k.Prefix, k.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return k, nil
}

// ListAPIKeys returns all API keys for a user.
func (d *DB) ListAPIKeys(userID string) ([]APIKey, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, prefix, created_at, last_used FROM api_keys WHERE user_id=? ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.CreatedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// DeleteAPIKey removes an API key by ID and owner.
func (d *DB) DeleteAPIKey(id, userID string) error {
	_, err := d.db.Exec(`DELETE FROM api_keys WHERE id=? AND user_id=?`, id, userID)
	return err
}

// GetAPIKeyByPrefix returns all keys matching the given prefix for bcrypt comparison.
func (d *DB) GetAPIKeyByPrefix(prefix string) ([]APIKey, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, key_hash, prefix, created_at, last_used FROM api_keys WHERE prefix=?`,
		prefix,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.Prefix, &k.CreatedAt, &k.LastUsed); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// UpdateAPIKeyLastUsed sets last_used to now for the given key.
func (d *DB) UpdateAPIKeyLastUsed(id string) error {
	_, err := d.db.Exec(`UPDATE api_keys SET last_used=? WHERE id=?`, time.Now().UTC(), id)
	return err
}

// AddAuditEntry inserts an audit log entry.
func (d *DB) AddAuditEntry(userID, username, action, resource, detail string) error {
	_, err := d.db.Exec(
		`INSERT INTO audit_log (id, user_id, username, action, resource, detail, created_at) VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), userID, username, action, resource, detail, time.Now().UTC(),
	)
	return err
}

// ListAuditLog returns audit entries ordered newest first.
func (d *DB) ListAuditLog(limit, offset int) ([]AuditEntry, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, username, action, resource, COALESCE(detail, ''), created_at
		 FROM audit_log ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

// ListAuditLogByUser returns audit entries for one user ordered newest first.
func (d *DB) ListAuditLogByUser(userID string, limit, offset int) ([]AuditEntry, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, username, action, resource, COALESCE(detail, ''), created_at
		 FROM audit_log WHERE user_id=? ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAuditEntries(rows)
}

// CountAuditLog returns the number of audit entries, optionally filtered by user.
func (d *DB) CountAuditLog(userID string) (int, error) {
	var count int
	var err error
	if userID == "" {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM audit_log`).Scan(&count)
	} else {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE user_id=?`, userID).Scan(&count)
	}
	return count, err
}

// ClearAuditLog permanently removes every audit log entry.
func (d *DB) ClearAuditLog() error {
	_, err := d.db.Exec(`DELETE FROM audit_log`)
	return err
}

func scanAuditEntries(rows *sql.Rows) ([]AuditEntry, error) {
	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Username, &e.Action, &e.Resource, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
