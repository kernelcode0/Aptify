package index

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kernelcode0/aptify/internal/signing"
	"github.com/kernelcode0/aptify/internal/storage"
)

// Generator regenerates APT index files for a repository.
type Generator struct {
	fs     *storage.FileStore
	signer *signing.Signer
}

func New(fs *storage.FileStore, signer *signing.Signer) *Generator {
	return &Generator{fs: fs, signer: signer}
}

// Regenerate rebuilds Packages, Packages.gz, Release, and InRelease for the given repo.
func (g *Generator) Regenerate(repo *storage.Repo, packages []storage.Package) error {
	pkgContent := buildPackages(repo.Slug, packages)
	pkgGz, err := gzipBytes(pkgContent)
	if err != nil {
		return fmt.Errorf("gzip Packages: %w", err)
	}

	indexDir := g.fs.IndexDir(repo.Slug, repo.Codename)
	if err := storage.WriteFile(indexDir+"/Packages", pkgContent); err != nil {
		return err
	}
	if err := storage.WriteFile(indexDir+"/Packages.gz", pkgGz); err != nil {
		return err
	}

	releaseContent := buildRelease(repo, pkgContent, pkgGz)
	if err := storage.WriteFile(g.fs.DistsDir(repo.Slug, repo.Codename)+"/Release", releaseContent); err != nil {
		return err
	}

	if g.signer != nil {
		inRelease, err := g.signer.ClearSign(releaseContent)
		if err != nil {
			return fmt.Errorf("clearsign: %w", err)
		}
		if err := storage.WriteFile(g.fs.DistsDir(repo.Slug, repo.Codename)+"/InRelease", inRelease); err != nil {
			return err
		}

		releaseSig, err := g.signer.DetachSign(releaseContent)
		if err != nil {
			return fmt.Errorf("detach sign: %w", err)
		}
		if err := storage.WriteFile(g.fs.DistsDir(repo.Slug, repo.Codename)+"/Release.gpg", releaseSig); err != nil {
			return err
		}
	}
	return nil
}

// buildPackages generates the Packages index content.
func buildPackages(slug string, packages []storage.Package) []byte {
	var buf bytes.Buffer
	for _, p := range packages {
		// Write the original control block fields.
		buf.WriteString(p.ControlJSON) // stored as the raw control text
		// Ensure block ends without trailing newline before we append our fields.
		if !strings.HasSuffix(p.ControlJSON, "\n") {
			buf.WriteByte('\n')
		}
		relPath := poolRelPath(slug, p.Package, p.Filename)
		fmt.Fprintf(&buf, "Filename: %s\n", relPath)
		fmt.Fprintf(&buf, "Size: %d\n", p.Size)
		fmt.Fprintf(&buf, "SHA256: %s\n", p.SHA256)
		fmt.Fprintf(&buf, "SHA1: %s\n", p.SHA1)
		fmt.Fprintf(&buf, "MD5sum: %s\n", p.MD5)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func poolRelPath(slug, pkgName, filename string) string {
	letter := pkgName[:1]
	if len(pkgName) > 3 && pkgName[:3] == "lib" {
		letter = "lib" + pkgName[3:4]
	}
	return fmt.Sprintf("pool/main/%s/%s/%s", letter, pkgName, filename)
}

// buildRelease generates the Release file.
func buildRelease(repo *storage.Repo, packages, packagesGz []byte) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Origin: %s\n", repo.Name)
	fmt.Fprintf(&buf, "Label: %s\n", repo.Name)
	fmt.Fprintf(&buf, "Suite: %s\n", repo.Codename)
	fmt.Fprintf(&buf, "Codename: %s\n", repo.Codename)
	fmt.Fprintf(&buf, "Date: %s\n", time.Now().UTC().Format(time.RFC1123))
	fmt.Fprintf(&buf, "Architectures: amd64 arm64 all\n")
	fmt.Fprintf(&buf, "Components: main\n")
	fmt.Fprintf(&buf, "Description: %s APT repository\n", repo.Name)

	type fileEntry struct {
		name string
		data []byte
	}
	files := []fileEntry{
		{"main/binary-amd64/Packages", packages},
		{"main/binary-amd64/Packages.gz", packagesGz},
	}

	buf.WriteString("MD5Sum:\n")
	for _, fe := range files {
		s := md5.Sum(fe.data)
		fmt.Fprintf(&buf, " %s %d %s\n", hex.EncodeToString(s[:]), len(fe.data), fe.name)
	}
	buf.WriteString("SHA1:\n")
	for _, fe := range files {
		s := sha1.Sum(fe.data)
		fmt.Fprintf(&buf, " %s %d %s\n", hex.EncodeToString(s[:]), len(fe.data), fe.name)
	}
	buf.WriteString("SHA256:\n")
	for _, fe := range files {
		s := sha256.Sum256(fe.data)
		fmt.Fprintf(&buf, " %s %d %s\n", hex.EncodeToString(s[:]), len(fe.data), fe.name)
	}

	return buf.Bytes()
}

func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
