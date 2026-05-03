package skills

import (
	"os"
	"strings"
)

type discoverContext struct {
	cwd  string
	home string
}

func resolveDiscoverContext(opts DiscoverOptions) discoverContext {
	cwd := strings.TrimSpace(opts.InvocationCWD)
	if cwd == "" {
		if current, err := os.Getwd(); err == nil {
			cwd = current
		}
	}
	if cwd == "" {
		cwd = "."
	}

	home := strings.TrimSpace(opts.HomeDir)
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = h
		}
	}

	return discoverContext{cwd: cwd, home: home}
}
