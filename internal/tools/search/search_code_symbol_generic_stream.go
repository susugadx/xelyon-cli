package search

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type genericSymbolSearchResult struct {
	matches         []genericSymbolMatch
	cancelRequested bool
}

type genericSymbolMatchVisitor func(genericSymbolMatch) bool

func findGenericSymbolMatches(symbol string, opts SearchOptions, limit int) []genericSymbolMatch {
	return findGenericSymbolMatchResult(symbol, opts, limit).matches
}

func findGenericSymbolMatchResult(symbol string, opts SearchOptions, limit int) genericSymbolSearchResult {
	if !common.IsRipgrepAvailable() {
		return genericSymbolSearchResult{}
	}

	return runGenericSymbolRipgrep(symbol, opts, limit)
}

func runGenericSymbolRipgrep(symbol string, opts SearchOptions, limit int) genericSymbolSearchResult {
	return runGenericSymbolRipgrepWithCollector(symbol, opts, func(stdout io.Reader) genericSymbolSearchResult {
		return collectGenericSymbolMatches(stdout, opts, limit)
	})
}

func streamGenericSymbolMatches(symbol string, opts SearchOptions, visit genericSymbolMatchVisitor) genericSymbolSearchResult {
	if !common.IsRipgrepAvailable() {
		return genericSymbolSearchResult{}
	}

	return runGenericSymbolRipgrepWithCollector(symbol, opts, func(stdout io.Reader) genericSymbolSearchResult {
		return visitGenericSymbolMatches(stdout, opts, visit)
	})
}

func runGenericSymbolRipgrepWithCollector(symbol string, opts SearchOptions, collect func(io.Reader) genericSymbolSearchResult) genericSymbolSearchResult {
	args, workdir := buildGenericRgArgs(symbol, opts)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
	if workdir != "" {
		cmd.Dir = workdir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return genericSymbolSearchResult{}
	}
	if err := cmd.Start(); err != nil {
		return genericSymbolSearchResult{}
	}

	if collect == nil {
		collect = func(io.Reader) genericSymbolSearchResult {
			return genericSymbolSearchResult{}
		}
	}
	result := collect(stdout)
	if result.cancelRequested {
		cancel()
	}
	_ = cmd.Wait()
	return result
}

func collectGenericSymbolMatches(reader io.Reader, opts SearchOptions, limit int) genericSymbolSearchResult {
	if reader == nil {
		return genericSymbolSearchResult{}
	}

	capacity := 0
	if limit > 0 {
		capacity = limit
		if capacity > maxGenericRefs {
			capacity = maxGenericRefs
		}
	}
	matches := make([]genericSymbolMatch, 0, capacity)
	result := visitGenericSymbolMatches(reader, opts, func(match genericSymbolMatch) bool {
		matches = append(matches, match)
		return limit <= 0 || len(matches) < limit
	})
	result.matches = matches
	return result
}

func visitGenericSymbolMatches(reader io.Reader, opts SearchOptions, visit genericSymbolMatchVisitor) genericSymbolSearchResult {
	if reader == nil {
		return genericSymbolSearchResult{}
	}
	if visit == nil {
		visit = func(genericSymbolMatch) bool {
			return true
		}
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		match, ok := parseGenericSymbolMatchLine(scanner.Text(), opts)
		if !ok {
			continue
		}
		if !visit(match) {
			return genericSymbolSearchResult{cancelRequested: true}
		}
	}
	return genericSymbolSearchResult{
		cancelRequested: scanner.Err() != nil,
	}
}

func parseGenericSymbolMatchLine(line string, opts SearchOptions) (genericSymbolMatch, bool) {
	parts := strings.SplitN(line, ":", 3)
	if len(parts) < 3 {
		return genericSymbolMatch{}, false
	}

	file := parts[0]
	if matchesSearchIgnoreFilter(file, opts) {
		return genericSymbolMatch{}, false
	}
	if !matchesSearchFileFilter(file, opts) {
		return genericSymbolMatch{}, false
	}
	lineNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return genericSymbolMatch{}, false
	}
	return genericSymbolMatch{
		File:    file,
		Line:    lineNum,
		Content: strings.TrimSpace(parts[2]),
	}, true
}

// buildGenericRgArgs は多言語シンボル検索用の ripgrep 引数を構築する。
func buildGenericRgArgs(symbol string, opts SearchOptions) ([]string, string) {
	basis := resolveSearchPathBasisForOptions(opts)

	args := []string{
		"-n", "--no-heading", "--with-filename", "--color", "never",
		"-w",
	}
	args = appendRipgrepFileFilterArgs(args, opts)
	args = appendRipgrepVisibilityFilterArgs(args, opts)
	args = append(args, symbol, basis.Target)
	return args, basis.Workdir
}
