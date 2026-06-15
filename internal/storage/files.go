package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,62}$`)

// ValidateSlug returns an error if slug is not URL-safe.
func ValidateSlug(slug string) error {
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("slug must be lowercase alphanumeric with optional hyphens (max 63 chars)")
	}
	return nil
}

// FileStore manages the on-disk layout for repos.
type FileStore struct {
	DataDir string
}

func NewFileStore(dataDir string) (*FileStore, error) {
	if err := os.MkdirAll(filepath.Join(dataDir, "keys"), 0700); err != nil {
		return nil, err
	}
	return &FileStore{DataDir: dataDir}, nil
}

// RepoDir returns the root directory for a repo slug.
func (fs *FileStore) RepoDir(slug string) string {
	return filepath.Join(fs.DataDir, "repos", slug)
}

// PoolDir returns the pool directory for a package within a repo.
func (fs *FileStore) PoolDir(slug, pkgName string) string {
	letter := pkgName[:1]
	if len(pkgName) > 3 && pkgName[:3] == "lib" {
		letter = "lib" + pkgName[3:4]
	}
	return filepath.Join(fs.RepoDir(slug), "pool", "main", letter, pkgName)
}

// DistsDir returns the dists directory path for a given codename.
func (fs *FileStore) DistsDir(slug, codename string) string {
	return filepath.Join(fs.RepoDir(slug), "dists", codename)
}

// IndexDir returns the binary-amd64 index directory.
func (fs *FileStore) IndexDir(slug, codename string) string {
	return filepath.Join(fs.DistsDir(slug, codename), "main", "binary-amd64")
}

// InitRepo creates the required directory tree for a new repo.
func (fs *FileStore) InitRepo(slug, codename string) error {
	dirs := []string{
		fs.PoolDir(slug, "placeholder"),
		filepath.Join(fs.DistsDir(slug, codename), "main", "binary-amd64"),
		filepath.Join(fs.DistsDir(slug, codename), "main", "binary-arm64"),
		filepath.Join(fs.DistsDir(slug, codename), "main", "binary-all"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			return err
		}
	}
	return nil
}

// SavePackage writes the .deb bytes to the pool directory, creating dirs as needed.
func (fs *FileStore) SavePackage(slug, pkgName, filename string, r io.Reader) (string, error) {
	dir := fs.PoolDir(slug, pkgName)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	filename = filepath.Base(filepath.Clean(filename))
	dest := filepath.Join(dir, filename)
	// #nosec G304 -- dest is bounded by dir and filename is sanitized
	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return dest, nil
}

// PoolRelPath returns the URL path fragment for a .deb relative to the repo root.
// e.g. pool/main/u/ubuntu-toolkit/ubuntu-toolkit_1.0_amd64.deb
func (fs *FileStore) PoolRelPath(slug, pkgName, filename string) string {
	letter := pkgName[:1]
	if len(pkgName) > 3 && pkgName[:3] == "lib" {
		letter = "lib" + pkgName[3:4]
	}
	return filepath.Join("pool", "main", letter, pkgName, filename)
}

// DeletePackageFile removes the .deb file from disk.
func (fs *FileStore) DeletePackageFile(slug, pkgName, filename string) error {
	filename = filepath.Base(filepath.Clean(filename))
	path := filepath.Join(fs.PoolDir(slug, pkgName), filename)
	return os.Remove(path)
}

// DeleteRepoDir removes all on-disk files for a repository.
func (fs *FileStore) DeleteRepoDir(slug string) error {
	return os.RemoveAll(fs.RepoDir(slug))
}

// KeyPath returns the path to the private GPG key.
func (fs *FileStore) KeyPath() string {
	return filepath.Join(fs.DataDir, "keys", "signing.pgp")
}

// PubKeyPath returns the path to the exported public key.
func (fs *FileStore) PubKeyPath() string {
	return filepath.Join(fs.DataDir, "keys", "signing.pub.asc")
}

// WriteFile atomically writes data to path.
func WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
