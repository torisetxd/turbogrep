package bench

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/torisetxd/turbogrep/internal/index"
)

type Result struct {
	Runs int

	TurboTimes []time.Duration
	RGTimes    []time.Duration

	TurboMatches int
	RGMatches    int
}

func Run(repoPath, indexPath, pattern string, runs int) (*Result, error) {
	if runs < 1 {
		runs = 1
	}

	if _, err := index.Build(repoPath, indexPath); err != nil {
		return nil, fmt.Errorf("index build failed: %w", err)
	}

	idx, err := index.Load(indexPath)
	if err != nil {
		return nil, fmt.Errorf("index load failed: %w", err)
	}

	res := &Result{Runs: runs, TurboTimes: make([]time.Duration, 0, runs), RGTimes: make([]time.Duration, 0, runs)}

	for i := 0; i < runs; i++ {
		start := time.Now()
		turboMatches, err := idx.Search(pattern)
		if err != nil {
			return nil, fmt.Errorf("turbogrep search failed: %w", err)
		}
		res.TurboTimes = append(res.TurboTimes, time.Since(start))
		res.TurboMatches = len(turboMatches)

		start = time.Now()
		cmd := exec.Command("rg", "-uuu", "-l", "-e", pattern, repoPath)
		out, err := cmd.Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
				out = nil
			} else {
				return nil, fmt.Errorf("rg failed: %w", err)
			}
		}
		res.RGTimes = append(res.RGTimes, time.Since(start))
		trimmed := strings.TrimSpace(string(out))
		if trimmed == "" {
			res.RGMatches = 0
		} else {
			res.RGMatches = len(strings.Split(trimmed, "\n"))
		}
	}

	return res, nil
}

func Summary(name string, durs []time.Duration) string {
	if len(durs) == 0 {
		return name + ": no data"
	}
	copyDur := append([]time.Duration(nil), durs...)
	sort.Slice(copyDur, func(i, j int) bool { return copyDur[i] < copyDur[j] })
	var total time.Duration
	for _, d := range copyDur {
		total += d
	}
	avg := total / time.Duration(len(copyDur))
	p50 := copyDur[len(copyDur)/2]
	p95 := copyDur[int(float64(len(copyDur)-1)*0.95)]
	return fmt.Sprintf("%s avg=%s p50=%s p95=%s min=%s max=%s", name, avg, p50, p95, copyDur[0], copyDur[len(copyDur)-1])
}
