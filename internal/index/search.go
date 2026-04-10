package index

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"turbogrep/internal/regexq"
)

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
