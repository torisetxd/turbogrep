package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildLoadSearch(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	idxDir := filepath.Join(tmp, "idx")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("MAX_FILE_SIZE here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("other text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Build(repo, idxDir)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	idx, err := Load(idxDir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	matches, err := idx.Search("MAX_FILE_SIZE")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(matches) != 1 || matches[0] != "a.txt" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}
