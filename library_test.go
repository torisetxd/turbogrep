package turbogrep

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLibraryBuildOpenSearch(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	idxDir := filepath.Join(tmp, "index")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "x.txt"), []byte("HELLO_WORLD\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := BuildWithOptions(repo, idxDir, BuildOptions{RespectGitIgnore: true})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	s, err := Open(idxDir)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	matches, err := s.Search("HELLO_WORLD")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(matches) != 1 || matches[0] != "x.txt" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}
