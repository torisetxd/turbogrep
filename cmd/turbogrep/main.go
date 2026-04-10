package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/torisetxd/turbogrep/internal/bench"
	"github.com/torisetxd/turbogrep/internal/index"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "index":
		runIndex(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "bench":
		runBench(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("turbogrep <command> [flags]")
	fmt.Println("commands: index, search, bench")
}

func runIndex(args []string) {
	fs := flag.NewFlagSet("index", flag.ExitOnError)
	repo := fs.String("repo", "", "repository root")
	idxDir := fs.String("index", ".turbogrep-index", "index directory")
	respectGitIgnore := fs.Bool("gitignore", true, "respect .gitignore while indexing")
	fs.Parse(args)

	if *repo == "" {
		fmt.Fprintln(os.Stderr, "--repo is required")
		os.Exit(2)
	}

	stats, err := index.BuildWithOptions(*repo, *idxDir, index.BuildOptions{RespectGitIgnore: *respectGitIgnore})
	if err != nil {
		fmt.Fprintf(os.Stderr, "index build failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("indexed files=%d trigrams=%d\n", stats.FilesIndexed, stats.Trigrams)
}

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	idxDir := fs.String("index", ".turbogrep-index", "index directory")
	pattern := fs.String("pattern", "", "regex pattern")
	fs.Parse(args)

	if *pattern == "" {
		fmt.Fprintln(os.Stderr, "--pattern is required")
		os.Exit(2)
	}

	idx, err := index.Load(*idxDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "index load failed: %v\n", err)
		os.Exit(1)
	}

	matches, err := idx.Search(*pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "search failed: %v\n", err)
		os.Exit(1)
	}
	for _, m := range matches {
		fmt.Println(m)
	}
	fmt.Printf("matches=%d\n", len(matches))
}

func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	repo := fs.String("repo", "", "repository root")
	idxDir := fs.String("index", ".turbogrep-index", "index directory")
	pattern := fs.String("pattern", "", "regex pattern")
	runs := fs.Int("runs", 5, "number of runs")
	fs.Parse(args)

	if *repo == "" || *pattern == "" {
		fmt.Fprintln(os.Stderr, "--repo and --pattern are required")
		os.Exit(2)
	}

	res, err := bench.Run(*repo, *idxDir, *pattern, *runs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchmark failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("runs=%d\n", res.Runs)
	fmt.Println(bench.Summary("turbogrep", res.TurboTimes))
	fmt.Println(bench.Summary("ripgrep", res.RGTimes))
	fmt.Printf("matches turbogrep=%d rg=%d\n", res.TurboMatches, res.RGMatches)
}
