# turbogrep

`turbogrep` is a local indexed regex search CLI inspired by trigram-based code search from Cursor (https://cursor.com/blog/fast-regex-search).

## Install

```bash
go get github.com/torisetxd/turbogrep@latest
```

## Library Usage

```go
package main

import (
	"fmt"

	"github.com/torisetxd/turbogrep"
)

func main() {
	_, err := turbogrep.BuildWithOptions("/path/to/repo", "/path/to/index", turbogrep.BuildOptions{RespectGitIgnore: true})
	if err != nil {
		panic(err)
	}

	s, err := turbogrep.Open("/path/to/index")
	if err != nil {
		panic(err)
	}

	matches, err := s.Search("React\\.createElement")
	if err != nil {
		panic(err)
	}

	for _, m := range matches {
		fmt.Println(m)
	}
}
```

## Commands

- `turbogrep index --repo <path> --index <path> [--gitignore=true]`: Build/update an index.
- `turbogrep search --index <path> --pattern <regex>`: Search using index + regex verification.
- `turbogrep bench --repo <path> --index <path> --pattern <regex> --runs <n>`: Benchmark against `rg`.

## Notes

- Uses overlapping trigrams for inverted indexing.
- Stores lightweight per-file masks for adjacency filtering.
- Final matches are always regex-verified, so false positives are safe.
- By default, indexing respects `.gitignore` rules.
