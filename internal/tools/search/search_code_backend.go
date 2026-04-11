package search

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var (
	gnuGrepCheckOnce sync.Once
	gnuGrepAvailable bool
)

func isGNUGrep() bool {
	gnuGrepCheckOnce.Do(func() {
		out, err := exec.Command("grep", "--version").CombinedOutput()
		if err != nil {
			gnuGrepAvailable = false
			return
		}
		gnuGrepAvailable = strings.Contains(strings.ToLower(string(out)), "gnu grep")
	})
	return gnuGrepAvailable
}

func executeSearch(pattern string, opts SearchOptions) (string, bool, []string, error) {
	basis := resolveSearchPathBasisForOptions(opts)

	if common.IsRipgrepAvailable() {
		args := []string{
			"--json",
			"-n",
		}
		if opts.CtxLines > 0 {
			args = append(args, "--context", fmt.Sprintf("%d", opts.CtxLines))
		}
		args = append(args, rawFileFilterToRipgrepArgs(opts.FileType, opts.FilePattern)...)
		if !opts.IsRegex {
			args = append(args, "--fixed-strings")
		}
		if opts.Multiline {
			args = append(args, "--multiline")
		}
		if opts.IncludeHidden {
			args = append(args, "--hidden")
		}
		if opts.IncludeIgnored {
			args = append(args, "--no-ignore")
		}
		for _, glob := range opts.ignoreGlobs {
			args = append(args, "--glob", glob)
		}
		args = append(args, pattern, basis.target)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
		if basis.workdir != "" {
			cmd.Dir = basis.workdir
		}
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
		if stdout.Len() == 0 && stderr.Len() > 0 {
			return "", true, nil, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), true, nil, nil
	}

	var warnings []string
	args := []string{
		"-rn",
		"-I",
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=.next",
	}
	if opts.IsRegex {
		args = append(args, "-E")
	} else {
		args = append(args, "-F")
	}
	if !opts.IncludeHidden {
		if isGNUGrep() {
			args = append(args,
				"--exclude=.[!.]*",
				"--exclude=..?*",
				"--exclude-dir=.[!.]*",
				"--exclude-dir=..?*",
			)
		} else {
			warnings = append(warnings, "Warning: hidden-file exclusion is not fully supported in grep fallback mode on non-GNU grep")
		}
	} else {
		warnings = append(warnings, "Warning: include_hidden is partially supported in grep fallback mode")
	}

	for _, glob := range rawFileFilterGlobs(opts.FileType, opts.FilePattern) {
		args = append(args, "--include="+glob)
	}

	if opts.Multiline {
		warnings = append(warnings, "Warning: multiline search is not supported in grep fallback mode (rg not found)")
	}
	if opts.CtxLines > 0 {
		args = append(args, "-C", fmt.Sprintf("%d", opts.CtxLines))
	}
	args = append(args, pattern, basis.target)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "grep", args...)
	if basis.workdir != "" {
		cmd.Dir = basis.workdir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	if stdout.Len() == 0 && stderr.Len() > 0 {
		return "", false, warnings, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), false, warnings, nil
}
