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

func TestBuildRespectsGitIgnore(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	idxDir := filepath.Join(tmp, "idx")
	if err := os.MkdirAll(filepath.Join(repo, "ignoredir"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte("ignored.txt\nignoredir/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "ignored.txt"), []byte("DO_NOT_INDEX_ME\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "ignoredir", "nested.txt"), []byte("DO_NOT_INDEX_ME\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "kept.txt"), []byte("INDEX_ME\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := BuildWithOptions(repo, idxDir, BuildOptions{RespectGitIgnore: true})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	idx, err := Load(idxDir)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	matches, err := idx.Search("DO_NOT_INDEX_ME")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected ignored files to be excluded, got: %#v", matches)
	}

	matches, err = idx.Search("INDEX_ME")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(matches) != 1 || matches[0] != "kept.txt" {
		t.Fatalf("expected kept.txt match, got: %#v", matches)
	}
}
