package index

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	lookupFile   = "lookup.bin"
	postingsFile = "postings.bin"
	metaFile     = "meta.json"
	magicLookup  = "TGRPLK1"
)

type Posting struct {
	FileID   uint32
	LocMask  uint8
	NextMask uint8
}

type entry struct {
	Tri    [3]byte
	Offset uint64
	Count  uint32
}

type meta struct {
	Version  int      `json:"version"`
	RepoRoot string   `json:"repo_root"`
	Files    []string `json:"files"`
}

type BuildStats struct {
	FilesIndexed int
	Trigrams     int
}

type Index struct {
	RepoRoot string
	Files    []string
	Postings map[[3]byte][]Posting
}

type aggMask struct {
	loc  uint8
	next uint8
}

func Build(repoRoot, indexDir string) (BuildStats, error) {
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return BuildStats{}, err
	}

	files := make([]string, 0, 1024)
	trigramMap := make(map[[3]byte]map[uint32]aggMask)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == ".hg" || base == ".svn" {
				return fs.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		if strings.HasPrefix(rel, "..") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() == 0 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fileID := uint32(len(files))
		files = append(files, filepath.ToSlash(rel))

		if len(content) < 3 {
			return nil
		}

		for i := 0; i+2 < len(content); i++ {
			tri := [3]byte{content[i], content[i+1], content[i+2]}
			perTri, ok := trigramMap[tri]
			if !ok {
				perTri = make(map[uint32]aggMask)
				trigramMap[tri] = perTri
			}

			m := perTri[fileID]
			m.loc |= 1 << uint(i%8)
			if i+3 < len(content) {
				m.next |= 1 << uint(followHash(content[i+3]))
			}
			perTri[fileID] = m
		}

		return nil
	})
	if err != nil {
		return BuildStats{}, err
	}

	entries := make([]entry, 0, len(trigramMap))
	postingsWriter, err := os.Create(filepath.Join(indexDir, postingsFile))
	if err != nil {
		return BuildStats{}, err
	}
	defer postingsWriter.Close()
	bw := bufio.NewWriterSize(postingsWriter, 1<<20)

	offset := uint64(0)
	keys := make([][3]byte, 0, len(trigramMap))
	for k := range trigramMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return bytes.Compare(keys[i][:], keys[j][:]) < 0
	})

	for _, tri := range keys {
		perFile := trigramMap[tri]
		ids := make([]int, 0, len(perFile))
		for id := range perFile {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)

		count := uint32(len(ids))
		entries = append(entries, entry{Tri: tri, Offset: offset, Count: count})
		for _, id := range ids {
			m := perFile[uint32(id)]
			if err := binary.Write(bw, binary.LittleEndian, uint32(id)); err != nil {
				return BuildStats{}, err
			}
			if err := bw.WriteByte(m.loc); err != nil {
				return BuildStats{}, err
			}
			if err := bw.WriteByte(m.next); err != nil {
				return BuildStats{}, err
			}
			offset += 6
		}
	}

	if err := bw.Flush(); err != nil {
		return BuildStats{}, err
	}
	if err := postingsWriter.Close(); err != nil {
		return BuildStats{}, err
	}

	lookupWriter, err := os.Create(filepath.Join(indexDir, lookupFile))
	if err != nil {
		return BuildStats{}, err
	}
	defer lookupWriter.Close()

	if _, err := lookupWriter.Write([]byte(magicLookup)); err != nil {
		return BuildStats{}, err
	}
	if err := binary.Write(lookupWriter, binary.LittleEndian, uint32(len(entries))); err != nil {
		return BuildStats{}, err
	}
	for _, e := range entries {
		if _, err := lookupWriter.Write(e.Tri[:]); err != nil {
			return BuildStats{}, err
		}
		if err := binary.Write(lookupWriter, binary.LittleEndian, e.Offset); err != nil {
			return BuildStats{}, err
		}
		if err := binary.Write(lookupWriter, binary.LittleEndian, e.Count); err != nil {
			return BuildStats{}, err
		}
	}

	m := meta{Version: 1, RepoRoot: repoRoot, Files: files}
	metaBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return BuildStats{}, err
	}
	if err := os.WriteFile(filepath.Join(indexDir, metaFile), metaBytes, 0o644); err != nil {
		return BuildStats{}, err
	}

	return BuildStats{FilesIndexed: len(files), Trigrams: len(entries)}, nil
}

func Load(indexDir string) (*Index, error) {
	metaBytes, err := os.ReadFile(filepath.Join(indexDir, metaFile))
	if err != nil {
		return nil, err
	}
	var m meta
	if err := json.Unmarshal(metaBytes, &m); err != nil {
		return nil, err
	}

	lookupReader, err := os.Open(filepath.Join(indexDir, lookupFile))
	if err != nil {
		return nil, err
	}
	defer lookupReader.Close()

	magic := make([]byte, len(magicLookup))
	if _, err := io.ReadFull(lookupReader, magic); err != nil {
		return nil, err
	}
	if string(magic) != magicLookup {
		return nil, errors.New("invalid lookup file")
	}

	var n uint32
	if err := binary.Read(lookupReader, binary.LittleEndian, &n); err != nil {
		return nil, err
	}

	entries := make([]entry, 0, n)
	for i := uint32(0); i < n; i++ {
		var tri [3]byte
		if _, err := io.ReadFull(lookupReader, tri[:]); err != nil {
			return nil, err
		}
		var off uint64
		if err := binary.Read(lookupReader, binary.LittleEndian, &off); err != nil {
			return nil, err
		}
		var cnt uint32
		if err := binary.Read(lookupReader, binary.LittleEndian, &cnt); err != nil {
			return nil, err
		}
		entries = append(entries, entry{Tri: tri, Offset: off, Count: cnt})
	}

	postingsReader, err := os.Open(filepath.Join(indexDir, postingsFile))
	if err != nil {
		return nil, err
	}
	defer postingsReader.Close()

	postings := make(map[[3]byte][]Posting, len(entries))
	for _, e := range entries {
		if _, err := postingsReader.Seek(int64(e.Offset), io.SeekStart); err != nil {
			return nil, err
		}
		list := make([]Posting, 0, e.Count)
		for i := uint32(0); i < e.Count; i++ {
			var id uint32
			if err := binary.Read(postingsReader, binary.LittleEndian, &id); err != nil {
				return nil, err
			}
			var b [2]byte
			if _, err := io.ReadFull(postingsReader, b[:]); err != nil {
				return nil, err
			}
			list = append(list, Posting{FileID: id, LocMask: b[0], NextMask: b[1]})
		}
		postings[e.Tri] = list
	}

	return &Index{RepoRoot: m.RepoRoot, Files: m.Files, Postings: postings}, nil
}

func (idx *Index) Search(pattern string) ([]string, error) {
	if idx == nil {
		return nil, fmt.Errorf("nil index")
	}
	return search(idx, pattern)
}

func followHash(b byte) uint8 {
	// Small deterministic hash suitable for an 8-bit bloom-like mask.
	x := uint8((b*131 + 17) & 0x7)
	return x
}
