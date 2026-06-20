package storage

import (
	"strings"
	"testing"
)

func TestSQLiteRepoDeleteCascadesPackages(t *testing.T) {
	t.Parallel()

	db, err := Open("sqlite", t.TempDir()+"/aptify.sqlite")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	repo, err := db.CreateRepo("tools", "Tools", "stable", "deb")
	if err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	if err := db.AddPackage(&Package{
		RepoID:      repo.ID,
		Filename:    "tool_1.0_amd64.deb",
		Package:     "tool",
		Version:     "1.0",
		Arch:        "amd64",
		Size:        1,
		SHA256:      strings.Repeat("a", 64),
		SHA1:        strings.Repeat("b", 40),
		MD5:         strings.Repeat("c", 32),
		ControlJSON: "Package: tool\n",
	}); err != nil {
		t.Fatalf("AddPackage: %v", err)
	}

	if err := db.DeleteRepo(repo.ID); err != nil {
		t.Fatalf("DeleteRepo: %v", err)
	}

	count, err := db.CountPackages(repo.ID)
	if err != nil {
		t.Fatalf("CountPackages: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected packages to cascade-delete, got %d", count)
	}
}
