package index

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/torisetxd/turbogrep/internal/regexq"
)

type SearchOptions struct {
	Pattern      string
	PathPattern  string // optional glob pattern to filter files
	ContextLines int   // number of lines before/after to include
}

type LineMatch struct {
	LineNum     int
	Line        string
	BeforeLines []string // context lines before (up to ContextLines)
	AfterLines  []string // context lines after (up to ContextLines)
}

type SearchResult struct {
	File   string
	Matches []LineMatch
}

func searchWithOptions(idx *Index, opts SearchOptions) ([]SearchResult, error) {
	re, err := regexp.Compile(opts.Pattern)
	if err != nil {
		return nil, err
	}

	var pathRe *regexp.Regexp
	if opts.PathPattern != "" {
		pathRe, err = regexp.Compile(opts.PathPattern)
		if err != nil {
			return nil, err
		}
	}

	plan := regexq.Plan(opts.Pattern, re)

	candidate := make(map[uint32]struct{})
	if len(plan.Terms) == 0 {
		for i := range idx.Files {
			candidate[uint32(i)] = struct{}{}
		}
	} else {
		for i, tri := range plan.Terms {
			list, ok := idx.Postings[tri]
			if !ok {
				return nil, nil
			}
			if i == 0 {
				for _, p := range list {
					candidate[p.FileID] = struct{}{}
				}
				continue
			}
			next := make(map[uint32]struct{}, len(candidate))
			for _, p := range list {
				if _, ok := candidate[p.FileID]; ok {
					next[p.FileID] = struct{}{}
				}
			}
			candidate = next
			if len(candidate) == 0 {
				return nil, nil
			}
		}

		if len(plan.Pairs) > 0 {
			candidate = applyAdjacencyFilter(idx, candidate, plan.Pairs)
			if len(candidate) == 0 {
				return nil, nil
			}
		}
	}

	// Early path filtering using index (avoid reading files)
	if pathRe != nil {
		filtered := make(map[uint32]struct{}, len(candidate))
		for id := range candidate {
			if int(id) >= len(idx.Files) {
				continue
			}
			if pathRe.MatchString(idx.Files[id]) {
				filtered[id] = struct{}{}
			}
		}
		candidate = filtered
		if len(candidate) == 0 {
			return nil, nil
		}
	}

	// Read files and find matching lines
	results := make([]SearchResult, 0, len(candidate))
	allIDs := make([]uint32, 0, len(candidate))
	for id := range candidate {
		allIDs = append(allIDs, id)
	}
	sort.Slice(allIDs, func(i, j int) bool {
		return idx.Files[allIDs[i]] < idx.Files[allIDs[j]]
	})

	ctx := opts.ContextLines
	for _, id := range allIDs {
		if int(id) >= len(idx.Files) {
			continue
		}

		filePath := idx.Files[id]
		fullPath := filepath.Join(idx.RepoRoot, filePath)

		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		if !re.Match(content) {
			continue
		}

		// If path filter wasn't applied earlier, apply now
		if pathRe != nil && !pathRe.MatchString(filePath) {
			continue
		}

		// Find matching lines with context
		matches := findMatchesWithContext(content, re, ctx)
		if len(matches) > 0 {
			results = append(results, SearchResult{
				File:    filePath,
				Matches: matches,
			})
		}
	}

	return results, nil
}

func findMatchesWithContext(content []byte, re *regexp.Regexp, ctx int) []LineMatch {
	lines := strings.Split(string(content), "\n")
	matches := make([]LineMatch, 0)

	for i, line := range lines {
		if !re.MatchString(line) {
			continue
		}

		match := LineMatch{
			LineNum: i + 1, // 1-based line numbers
			Line:    line,
		}

		// Add context before
		if ctx > 0 {
			start := i - ctx
			if start < 0 {
				start = 0
			}
			match.BeforeLines = lines[start:i]
		}

		// Add context after
		if ctx > 0 {
			end := i + ctx + 1
			if end > len(lines) {
				end = len(lines)
			}
			match.AfterLines = lines[i+1:end]
		}

		matches = append(matches, match)
	}

	return matches
}

// SearchWithOptions performs a search with full options
func (idx *Index) SearchWithOptions(opts SearchOptions) ([]SearchResult, error) {
	if idx == nil {
		return nil, fmt.Errorf("nil index")
	}
	return searchWithOptions(idx, opts)
}

// SearchWithContext searches for pattern and returns matches with surrounding lines
func (idx *Index) SearchWithContext(pattern string, contextLines int) ([]SearchResult, error) {
	return idx.SearchWithOptions(SearchOptions{
		Pattern:      pattern,
		ContextLines: contextLines,
	})
}

// SearchWithPath searches for pattern in files matching pathPattern
func (idx *Index) SearchWithPath(pattern, pathPattern string) ([]SearchResult, error) {
	return idx.SearchWithOptions(SearchOptions{
		Pattern:     pattern,
		PathPattern: pathPattern,
	})
}

// SearchResultWithContext searches with both path filtering and context
func (idx *Index) SearchResultWithContext(pattern, pathPattern string, contextLines int) ([]SearchResult, error) {
	return idx.SearchWithOptions(SearchOptions{
		Pattern:      pattern,
		PathPattern:  pathPattern,
		ContextLines: contextLines,
	})
}

// GetFileContent reads a file from the index
func (idx *Index) GetFileContent(relPath string) ([]byte, error) {
	fullPath := filepath.Join(idx.RepoRoot, relPath)
	return os.ReadFile(fullPath)
}

// GetFileContentAtLine reads a file and returns a range of lines around a line number
func (idx *Index) GetFileContentAtLine(relPath string, lineNum, before, after int) ([]string, error) {
	content, err := idx.GetFileContent(relPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	if lineNum < 1 || lineNum > len(lines) {
		return nil, fmt.Errorf("line number out of range")
	}

	// Convert to 0-based
	idxLine := lineNum - 1

	start := idxLine - before
	if start < 0 {
		start = 0
	}

	end := idxLine + after + 1
	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end], nil
}

// Simple streaming search for very large files - reads line by line
func searchLargeFile(path string, re *regexp.Regexp, ctx int) ([]LineMatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []LineMatch
	lineNum := 0
	scanner := bufio.NewScanner(file)

	// Keep a buffer of recent lines for context
	lineBuffer := make([]string, ctx+1)
	bufferPos := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Store in circular buffer
		lineBuffer[bufferPos%len(lineBuffer)] = line
		bufferPos++

		if re.MatchString(line) {
			match := LineMatch{
				LineNum: lineNum,
				Line:    line,
			}

			// Get context before
			if ctx > 0 {
				avail := bufferPos - 1
				if avail > ctx {
					avail = ctx
				}
				match.BeforeLines = make([]string, avail)
				for j := 0; j < avail; j++ {
					idx := (bufferPos - avail + j) % len(lineBuffer)
					match.BeforeLines[j] = lineBuffer[idx]
				}
			}

			// Note: After context not available until we've read more
			// This is a limitation - we'd need to buffer matches

			matches = append(matches, match)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// Legacy search function - returns file paths only
func search(idx *Index, pattern string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	plan := regexq.Plan(pattern, re)

	candidate := make(map[uint32]struct{})
	if len(plan.Terms) == 0 {
		for i := range idx.Files {
			candidate[uint32(i)] = struct{}{}
		}
	} else {
		for i, tri := range plan.Terms {
			list, ok := idx.Postings[tri]
			if !ok {
				return nil, nil
			}
			if i == 0 {
				for _, p := range list {
					candidate[p.FileID] = struct{}{}
				}
				continue
			}
			next := make(map[uint32]struct{}, len(candidate))
			for _, p := range list {
				if _, ok := candidate[p.FileID]; ok {
					next[p.FileID] = struct{}{}
				}
			}
			candidate = next
			if len(candidate) == 0 {
				return nil, nil
			}
		}

		if len(plan.Pairs) > 0 {
			candidate = applyAdjacencyFilter(idx, candidate, plan.Pairs)
			if len(candidate) == 0 {
				return nil, nil
			}
		}
	}

	out := make([]string, 0, len(candidate))
	for id := range candidate {
		if int(id) >= len(idx.Files) {
			continue
		}
		p := filepath.Join(idx.RepoRoot, idx.Files[id])
		content, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if re.Match(content) {
			out = append(out, idx.Files[id])
		}
	}
	sort.Strings(out)
	return out, nil
}

func applyAdjacencyFilter(idx *Index, in map[uint32]struct{}, pairs [][2][3]byte) map[uint32]struct{} {
	out := make(map[uint32]struct{}, len(in))
	if len(pairs) == 0 {
		for id := range in {
			out[id] = struct{}{}
		}
		return out
	}

	for id := range in {
		okAll := true
		for _, pair := range pairs {
			left := postingForFile(idx.Postings[pair[0]], id)
			right := postingForFile(idx.Postings[pair[1]], id)
			if left == nil || right == nil {
				okAll = false
				break
			}

			rot := ((left.LocMask << 1) | (left.LocMask >> 7))
			if (rot & right.LocMask) == 0 {
				okAll = false
				break
			}
		}
		if okAll {
			out[id] = struct{}{}
		}
	}
	return out
}

func postingForFile(list []Posting, id uint32) *Posting {
	lo, hi := 0, len(list)
	for lo < hi {
		mid := (lo + hi) / 2
		if list[mid].FileID < id {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(list) && list[lo].FileID == id {
		return &list[lo]
	}
	return nil
}

func MustSearch(idx *Index, pattern string) []string {
	res, err := idx.Search(pattern)
	if err != nil {
		panic(fmt.Sprintf("search failed: %v", err))
	}
	return res
}