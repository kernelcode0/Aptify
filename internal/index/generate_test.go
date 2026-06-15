package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kernelcode0/aptify/internal/storage"
)

func TestRegenerateWritesArchitectureSpecificPackages(t *testing.T) {
	t.Parallel()

	fs, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	repo := &storage.Repo{Slug: "tools", Name: "Tools", Codename: "stable"}
	packages := []storage.Package{
		{
			Package:     "amd-tool",
			Version:     "1.0",
			Arch:        "amd64",
			Filename:    "amd-tool_1.0_amd64.deb",
			Size:        10,
			SHA256:      strings.Repeat("a", 64),
			SHA1:        strings.Repeat("b", 40),
			MD5:         strings.Repeat("c", 32),
			ControlJSON: "Package: amd-tool\nVersion: 1.0\nArchitecture: amd64\n",
		},
		{
			Package:     "arm-tool",
			Version:     "1.0",
			Arch:        "arm64",
			Filename:    "arm-tool_1.0_arm64.deb",
			Size:        10,
			SHA256:      strings.Repeat("d", 64),
			SHA1:        strings.Repeat("e", 40),
			MD5:         strings.Repeat("f", 32),
			ControlJSON: "Package: arm-tool\nVersion: 1.0\nArchitecture: arm64\n",
		},
		{
			Package:     "shared-tool",
			Version:     "1.0",
			Arch:        "all",
			Filename:    "shared-tool_1.0_all.deb",
			Size:        10,
			SHA256:      strings.Repeat("1", 64),
			SHA1:        strings.Repeat("2", 40),
			MD5:         strings.Repeat("3", 32),
			ControlJSON: "Package: shared-tool\nVersion: 1.0\nArchitecture: all\n",
		},
	}

	if err := New(fs, nil).Regenerate(repo, packages); err != nil {
		t.Fatalf("Regenerate: %v", err)
	}

	amd64Packages := readIndex(t, fs, repo, "amd64")
	assertContains(t, amd64Packages, "Package: amd-tool")
	assertContains(t, amd64Packages, "Package: shared-tool")
	assertNotContains(t, amd64Packages, "Package: arm-tool")

	arm64Packages := readIndex(t, fs, repo, "arm64")
	assertContains(t, arm64Packages, "Package: arm-tool")
	assertContains(t, arm64Packages, "Package: shared-tool")
	assertNotContains(t, arm64Packages, "Package: amd-tool")

	allPackages := readIndex(t, fs, repo, "all")
	assertContains(t, allPackages, "Package: shared-tool")
	assertNotContains(t, allPackages, "Package: amd-tool")
	assertNotContains(t, allPackages, "Package: arm-tool")

	release, err := os.ReadFile(filepath.Join(fs.DistsDir(repo.Slug, repo.Codename), "Release"))
	if err != nil {
		t.Fatalf("read Release: %v", err)
	}
	assertContains(t, string(release), "main/binary-amd64/Packages")
	assertContains(t, string(release), "main/binary-arm64/Packages")
	assertContains(t, string(release), "main/binary-all/Packages")
}

func readIndex(t *testing.T, fs *storage.FileStore, repo *storage.Repo, arch string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(fs.DistsDir(repo.Slug, repo.Codename), "main", "binary-"+arch, "Packages"))
	if err != nil {
		t.Fatalf("read Packages for %s: %v", arch, err)
	}
	return string(data)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}

func assertNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected %q not to contain %q", got, want)
	}
}
