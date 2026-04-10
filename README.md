# turbogrep

`turbogrep` is a local indexed regex search CLI inspired by trigram-based code search from Cursor (https://cursor.com/blog/fast-regex-search).

## Commands

- `turbogrep index --repo <path> --index <path>`: Build/update an index.
- `turbogrep search --index <path> --pattern <regex>`: Search using index + regex verification.
- `turbogrep bench --repo <path> --index <path> --pattern <regex> --runs <n>`: Benchmark against `rg`.

## Notes

- Uses overlapping trigrams for inverted indexing.
- Stores lightweight per-file masks for adjacency filtering.
- Final matches are always regex-verified, so false positives are safe.
