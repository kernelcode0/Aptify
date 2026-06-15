package storage

import (
	"os"
	"testing"
)

func TestDeleteRepoDirRemovesRepositoryFiles(t *testing.T) {
	t.Parallel()

	fs, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	if err := fs.InitRepo("tools", "stable"); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	if _, err := os.Stat(fs.RepoDir("tools")); err != nil {
		t.Fatalf("repo dir should exist before delete: %v", err)
	}

	if err := fs.DeleteRepoDir("tools"); err != nil {
		t.Fatalf("DeleteRepoDir: %v", err)
	}
	if _, err := os.Stat(fs.RepoDir("tools")); !os.IsNotExist(err) {
		t.Fatalf("expected repo dir to be gone, got err=%v", err)
	}
}
