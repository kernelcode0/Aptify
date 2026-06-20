package index

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"  // #nosec G501
	"crypto/sha1" // #nosec G505
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"path/filepath"
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

type archIndex struct {
	arch string
	pkg  []byte
	gz   []byte
}

func New(fs *storage.FileStore, signer *signing.Signer) *Generator {
	return &Generator{fs: fs, signer: signer}
}

// Regenerate rebuilds Packages, Packages.gz, Release, and InRelease for the given repo.
func (g *Generator) Regenerate(repo *storage.Repo, packages []storage.Package) error {
	if repo.Type == "rpm" {
		return g.regenerateRPM(repo, packages)
	}
	archs := []string{"amd64", "arm64", "all"}
	indexes := make([]archIndex, 0, len(archs))
	for _, arch := range archs {
		pkgContent := buildPackages(repo.Slug, packagesForArch(packages, arch))
		pkgGz, err := gzipBytes(pkgContent)
		if err != nil {
			return fmt.Errorf("gzip Packages for %s: %w", arch, err)
		}
		indexes = append(indexes, archIndex{arch: arch, pkg: pkgContent, gz: pkgGz})
	}

	for _, idx := range indexes {
		indexDir := filepath.Join(g.fs.DistsDir(repo.Slug, repo.Codename), "main", "binary-"+idx.arch)
		if err := storage.WriteFile(indexDir+"/Packages", idx.pkg); err != nil {
			return err
		}
		if err := storage.WriteFile(indexDir+"/Packages.gz", idx.gz); err != nil {
			return err
		}
	}

	releaseContent := buildRelease(repo, indexes)
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

func (g *Generator) regenerateRPM(repo *storage.Repo, packages []storage.Package) error {
	now := time.Now().Unix()
	primary := buildRPMPrimary(packages)
	filelists := buildRPMFilelists(packages)
	other := buildRPMOther(packages)

	primaryGz, err := gzipBytes(primary)
	if err != nil {
		return err
	}
	filelistsGz, err := gzipBytes(filelists)
	if err != nil {
		return err
	}
	otherGz, err := gzipBytes(other)
	if err != nil {
		return err
	}

	files := []rpmMetaFile{
		{kind: "primary", href: "repodata/primary.xml.gz", data: primaryGz, openData: primary},
		{kind: "filelists", href: "repodata/filelists.xml.gz", data: filelistsGz, openData: filelists},
		{kind: "other", href: "repodata/other.xml.gz", data: otherGz, openData: other},
	}
	for _, fe := range files {
		if err := storage.WriteFile(filepath.Join(g.fs.RepoDir(repo.Slug), fe.href), fe.data); err != nil {
			return err
		}
	}
	repomd := buildRepomd(files, now)
	repomdPath := filepath.Join(g.fs.RepoDir(repo.Slug), "repodata", "repomd.xml")
	if err := storage.WriteFile(repomdPath, repomd); err != nil {
		return err
	}
	if g.signer != nil {
		sig, err := g.signer.DetachSign(repomd)
		if err != nil {
			return fmt.Errorf("sign repomd: %w", err)
		}
		if err := storage.WriteFile(repomdPath+".asc", sig); err != nil {
			return err
		}
	}
	return nil
}

type rpmMetaFile struct {
	kind     string
	href     string
	data     []byte
	openData []byte
}

func buildRepomd(files []rpmMetaFile, timestamp int64) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<repomd xmlns="http://linux.duke.edu/metadata/repo">` + "\n")
	for _, fe := range files {
		sum := sha256.Sum256(fe.data)
		openSum := sha256.Sum256(fe.openData)
		fmt.Fprintf(&buf, `  <data type="%s">`+"\n", fe.kind)
		fmt.Fprintf(&buf, `    <checksum type="sha256">%s</checksum>`+"\n", hex.EncodeToString(sum[:]))
		fmt.Fprintf(&buf, `    <open-checksum type="sha256">%s</open-checksum>`+"\n", hex.EncodeToString(openSum[:]))
		fmt.Fprintf(&buf, `    <location href="%s"/>`+"\n", fe.href)
		fmt.Fprintf(&buf, "    <timestamp>%d</timestamp>\n", timestamp)
		fmt.Fprintf(&buf, "    <size>%d</size>\n", len(fe.data))
		fmt.Fprintf(&buf, "    <open-size>%d</open-size>\n", len(fe.openData))
		buf.WriteString("  </data>\n")
	}
	buf.WriteString("</repomd>\n")
	return buf.Bytes()
}

func buildRPMPrimary(packages []storage.Package) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, `<metadata xmlns="http://linux.duke.edu/metadata/common" packages="%d">`+"\n", len(packages))
	for _, p := range packages {
		fmt.Fprintf(&buf, `  <package type="rpm">`+"\n")
		fmt.Fprintf(&buf, "    <name>%s</name>\n", xmlEscape(p.Package))
		fmt.Fprintf(&buf, "    <arch>%s</arch>\n", xmlEscape(p.Arch))
		fmt.Fprintf(&buf, `    <version epoch="0" ver="%s" rel="%s"/>`+"\n", xmlEscape(p.Version), xmlEscape(p.Release))
		fmt.Fprintf(&buf, `    <checksum type="sha256" pkgid="YES">%s</checksum>`+"\n", p.SHA256)
		fmt.Fprintf(&buf, "    <summary>%s</summary>\n", xmlEscape(p.Package))
		fmt.Fprintf(&buf, "    <description>%s</description>\n", xmlEscape(p.Package))
		fmt.Fprintf(&buf, "    <packager>Aptify</packager>\n")
		fmt.Fprintf(&buf, `    <time file="%d" build="%d"/>`+"\n", p.UploadedAt.Unix(), p.UploadedAt.Unix())
		fmt.Fprintf(&buf, `    <size package="%d" installed="%d" archive="%d"/>`+"\n", p.Size, p.Size, p.Size)
		fmt.Fprintf(&buf, `    <location href="packages/%s"/>`+"\n", xmlEscape(p.Filename))
		buf.WriteString("    <format/>\n")
		buf.WriteString("  </package>\n")
	}
	buf.WriteString("</metadata>\n")
	return buf.Bytes()
}

func buildRPMFilelists(packages []storage.Package) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, `<filelists xmlns="http://linux.duke.edu/metadata/filelists" packages="%d">`+"\n", len(packages))
	for _, p := range packages {
		fmt.Fprintf(&buf, `  <package pkgid="%s" name="%s" arch="%s">`+"\n", p.SHA256, xmlEscape(p.Package), xmlEscape(p.Arch))
		fmt.Fprintf(&buf, `    <version epoch="0" ver="%s" rel="%s"/>`+"\n", xmlEscape(p.Version), xmlEscape(p.Release))
		buf.WriteString("  </package>\n")
	}
	buf.WriteString("</filelists>\n")
	return buf.Bytes()
}

func buildRPMOther(packages []storage.Package) []byte {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	fmt.Fprintf(&buf, `<otherdata xmlns="http://linux.duke.edu/metadata/other" packages="%d">`+"\n", len(packages))
	for _, p := range packages {
		fmt.Fprintf(&buf, `  <package pkgid="%s" name="%s" arch="%s">`+"\n", p.SHA256, xmlEscape(p.Package), xmlEscape(p.Arch))
		fmt.Fprintf(&buf, `    <version epoch="0" ver="%s" rel="%s"/>`+"\n", xmlEscape(p.Version), xmlEscape(p.Release))
		buf.WriteString("  </package>\n")
	}
	buf.WriteString("</otherdata>\n")
	return buf.Bytes()
}

func xmlEscape(v string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(v))
	return buf.String()
}

func packagesForArch(packages []storage.Package, arch string) []storage.Package {
	filtered := make([]storage.Package, 0, len(packages))
	for _, p := range packages {
		if p.Arch == arch || p.Arch == "all" {
			filtered = append(filtered, p)
		}
	}
	return filtered
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

type releaseFile struct {
	name string
	data []byte
}

func (idx archIndex) releaseFiles() []releaseFile {
	base := "main/binary-" + idx.arch
	return []releaseFile{
		{name: base + "/Packages", data: idx.pkg},
		{name: base + "/Packages.gz", data: idx.gz},
	}
}

// buildRelease generates the Release file.
func buildRelease(repo *storage.Repo, indexes []archIndex) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Origin: %s\n", repo.Name)
	fmt.Fprintf(&buf, "Label: %s\n", repo.Name)
	fmt.Fprintf(&buf, "Suite: %s\n", repo.Codename)
	fmt.Fprintf(&buf, "Codename: %s\n", repo.Codename)
	fmt.Fprintf(&buf, "Date: %s\n", time.Now().UTC().Format(time.RFC1123))
	fmt.Fprintf(&buf, "Architectures: amd64 arm64 all\n")
	fmt.Fprintf(&buf, "Components: main\n")
	fmt.Fprintf(&buf, "Description: %s APT repository\n", repo.Name)

	var files []releaseFile
	for _, idx := range indexes {
		files = append(files, idx.releaseFiles()...)
	}

	buf.WriteString("MD5Sum:\n")
	for _, fe := range files {
		s := md5.Sum(fe.data) // #nosec G401 -- Required by APT format
		fmt.Fprintf(&buf, " %s %d %s\n", hex.EncodeToString(s[:]), len(fe.data), fe.name)
	}
	buf.WriteString("SHA1:\n")
	for _, fe := range files {
		s := sha1.Sum(fe.data) // #nosec G401 -- Required by APT format
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
