package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection.
type DB struct {
	db     *sql.DB
	dbType string
}

// Repo represents a repository record.
type Repo struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Codename  string    `json:"codename"`
	CreatedAt time.Time `json:"created_at"`
}

// Package represents a package record.
type Package struct {
	ID          string    `json:"id"`
	RepoID      string    `json:"repo_id"`
	Filename    string    `json:"filename"`
	Package     string    `json:"package"`
	Version     string    `json:"version"`
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
	d := &DB{db: db, dbType: dbType}
	if err := d.migrate(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DB) migrate() error {
	var repoTable, pkgTable string
	if d.dbType == "mysql" {
		repoTable = `
		CREATE TABLE IF NOT EXISTS repos (
			id          VARCHAR(36) PRIMARY KEY,
			slug        VARCHAR(255) UNIQUE NOT NULL,
			name        VARCHAR(255) NOT NULL,
			codename    VARCHAR(255) NOT NULL DEFAULT 'stable',
			created_at  DATETIME NOT NULL
		);`
		pkgTable = `
		CREATE TABLE IF NOT EXISTS packages (
			id           VARCHAR(36) PRIMARY KEY,
			repo_id      VARCHAR(36) NOT NULL,
			filename     VARCHAR(255) NOT NULL,
			package      VARCHAR(255) NOT NULL,
			version      VARCHAR(255) NOT NULL,
			arch         VARCHAR(255) NOT NULL,
			size         BIGINT NOT NULL,
			sha256       VARCHAR(64) NOT NULL,
			sha1         VARCHAR(40) NOT NULL,
			md5          VARCHAR(32) NOT NULL,
			control_json TEXT NOT NULL,
			uploaded_at  DATETIME NOT NULL,
			FOREIGN KEY (repo_id) REFERENCES repos(id) ON DELETE CASCADE,
			INDEX idx_packages_repo (repo_id)
		);`
	} else {
		repoTable = `
		CREATE TABLE IF NOT EXISTS repos (
			id          TEXT PRIMARY KEY,
			slug        TEXT UNIQUE NOT NULL,
			name        TEXT NOT NULL,
			codename    TEXT NOT NULL DEFAULT 'stable',
			created_at  DATETIME NOT NULL
		);`
		pkgTable = `
		CREATE TABLE IF NOT EXISTS packages (
			id           TEXT PRIMARY KEY,
			repo_id      TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
			filename     TEXT NOT NULL,
			package      TEXT NOT NULL,
			version      TEXT NOT NULL,
			arch         TEXT NOT NULL,
			size         INTEGER NOT NULL,
			sha256       TEXT NOT NULL,
			sha1         TEXT NOT NULL,
			md5          TEXT NOT NULL,
			control_json TEXT NOT NULL DEFAULT '{}',
			uploaded_at  DATETIME NOT NULL
		);`
	}

	if _, err := d.db.Exec(repoTable); err != nil {
		return err
	}
	if _, err := d.db.Exec(pkgTable); err != nil {
		return err
	}
	if d.dbType == "sqlite" {
		if _, err := d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_packages_repo ON packages(repo_id);`); err != nil {
			return err
		}
	}
	return nil
}

// CreateRepo inserts a new repo and returns it.
func (d *DB) CreateRepo(slug, name, codename string) (*Repo, error) {
	r := &Repo{
		ID:        uuid.NewString(),
		Slug:      slug,
		Name:      name,
		Codename:  codename,
		CreatedAt: time.Now().UTC(),
	}
	_, err := d.db.Exec(
		`INSERT INTO repos (id, slug, name, codename, created_at) VALUES (?,?,?,?,?)`,
		r.ID, r.Slug, r.Name, r.Codename, r.CreatedAt,
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
		`SELECT id, slug, name, codename, created_at FROM repos WHERE id=?`, id,
	).Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// GetRepoBySlug fetches a repo by slug.
func (d *DB) GetRepoBySlug(slug string) (*Repo, error) {
	r := &Repo{}
	err := d.db.QueryRow(
		`SELECT id, slug, name, codename, created_at FROM repos WHERE slug=?`, slug,
	).Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

// ListRepos returns all repos ordered by creation time.
func (d *DB) ListRepos() ([]Repo, error) {
	rows, err := d.db.Query(`SELECT id, slug, name, codename, created_at FROM repos ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []Repo
	for rows.Next() {
		var r Repo
		if err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.Codename, &r.CreatedAt); err != nil {
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
		`INSERT INTO packages (id, repo_id, filename, package, version, arch, size, sha256, sha1, md5, control_json, uploaded_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.RepoID, p.Filename, p.Package, p.Version, p.Arch,
		p.Size, p.SHA256, p.SHA1, p.MD5, p.ControlJSON, p.UploadedAt,
	)
	return err
}

// ListPackages returns all packages for a repo. If limit > 0, it applies pagination.
func (d *DB) ListPackages(repoID string, limit, offset int) ([]Package, error) {
	query := `SELECT id, repo_id, filename, package, version, arch, size, sha256, sha1, md5, control_json, uploaded_at
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
		if err := rows.Scan(&p.ID, &p.RepoID, &p.Filename, &p.Package, &p.Version, &p.Arch,
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
		`SELECT id, repo_id, filename, package, version, arch, size, sha256, sha1, md5, control_json, uploaded_at
		 FROM packages WHERE id=?`, id,
	).Scan(&p.ID, &p.RepoID, &p.Filename, &p.Package, &p.Version, &p.Arch,
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

func (d *DB) Close() error { return d.db.Close() }
