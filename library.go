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

// SearchOptions configures a search with path targeting and context lines
type SearchOptions struct {
	Pattern      string   // regex pattern to search for
	PathPattern  string   // optional glob/regex pattern to filter files
	ContextLines int      // number of surrounding lines to include
}

// LineMatch represents a single matching line with optional context
type LineMatch struct {
	LineNum     int      `json:"line_num"`
	Line        string   `json:"line"`
	BeforeLines []string `json:"before,omitempty"` // context before (up to ContextLines)
	AfterLines  []string `json:"after,omitempty"`  // context after (up to ContextLines)
}

// SearchResult contains all matches in a single file
type SearchResult struct {
	File    string      `json:"file"`
	Matches []LineMatch `json:"matches"`
}

// Search performs a basic search returning only file paths (legacy behavior)
func (s *Searcher) Search(pattern string) ([]string, error) {
	return s.idx.Search(pattern)
}

// SearchWithOptions performs a search with full options including path filtering and context
func (s *Searcher) SearchWithOptions(opts SearchOptions) ([]SearchResult, error) {
	internalOpts := index.SearchOptions{
		Pattern:      opts.Pattern,
		PathPattern:  opts.PathPattern,
		ContextLines: opts.ContextLines,
	}
	results, err := s.idx.SearchWithOptions(internalOpts)
	if err != nil {
		return nil, err
	}
	// Convert internal results to public type
	out := make([]SearchResult, len(results))
	for i, r := range results {
		matches := make([]LineMatch, len(r.Matches))
		for j, m := range r.Matches {
			matches[j] = LineMatch{
				LineNum:     m.LineNum,
				Line:        m.Line,
				BeforeLines: m.BeforeLines,
				AfterLines:  m.AfterLines,
			}
		}
		out[i] = SearchResult{
			File:    r.File,
			Matches: matches,
		}
	}
	return out, nil
}

// SearchWithPath searches for pattern in files matching the path pattern
// pathPattern is a regex pattern (e.g., "*.go", "src/.*\.txt")
func (s *Searcher) SearchWithPath(pattern, pathPattern string) ([]SearchResult, error) {
	return s.SearchWithOptions(SearchOptions{
		Pattern:     pattern,
		PathPattern: pathPattern,
	})
}

// SearchWithContext searches for pattern and returns matches with surrounding lines
// contextLines specifies how many lines before and after to include
func (s *Searcher) SearchWithContext(pattern string, contextLines int) ([]SearchResult, error) {
	return s.SearchWithOptions(SearchOptions{
		Pattern:      pattern,
		ContextLines: contextLines,
	})
}

// SearchWithPathAndContext searches with both path filtering and context lines
func (s *Searcher) SearchWithPathAndContext(pattern, pathPattern string, contextLines int) ([]SearchResult, error) {
	return s.SearchWithOptions(SearchOptions{
		Pattern:      pattern,
		PathPattern:  pathPattern,
		ContextLines: contextLines,
	})
}

// GetFileContent reads the raw content of a file in the index
func (s *Searcher) GetFileContent(relPath string) ([]byte, error) {
	return s.idx.GetFileContent(relPath)
}

// GetFileContentAtLine reads a file and returns a range of lines around a specific line
// lineNum is 1-based, before/after specify how many lines to include before/after
func (s *Searcher) GetFileContentAtLine(relPath string, lineNum, before, after int) ([]string, error) {
	return s.idx.GetFileContentAtLine(relPath, lineNum, before, after)
}