package turbogrep

import "github.com/torisetxd/turbogrep/internal/index"

type BuildOptions struct {
	RespectGitIgnore bool
}

type BuildStats struct {
	FilesIndexed int
	Trigrams     int
}

type Searcher struct {
	idx *index.Index
}

func Build(repoRoot, indexDir string) (BuildStats, error) {
	stats, err := index.Build(repoRoot, indexDir)
	if err != nil {
		return BuildStats{}, err
	}
	return BuildStats{FilesIndexed: stats.FilesIndexed, Trigrams: stats.Trigrams}, nil
}

func BuildWithOptions(repoRoot, indexDir string, opts BuildOptions) (BuildStats, error) {
	stats, err := index.BuildWithOptions(repoRoot, indexDir, index.BuildOptions{RespectGitIgnore: opts.RespectGitIgnore})
	if err != nil {
		return BuildStats{}, err
	}
	return BuildStats{FilesIndexed: stats.FilesIndexed, Trigrams: stats.Trigrams}, nil
}

func Open(indexDir string) (*Searcher, error) {
	idx, err := index.Load(indexDir)
	if err != nil {
		return nil, err
	}
	return &Searcher{idx: idx}, nil
}

func (s *Searcher) Search(pattern string) ([]string, error) {
	return s.idx.Search(pattern)
}
