package file

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

type readRequestSource string

const (
	readRequestSourcePathWhole readRequestSource = "pathWhole"
	readRequestSourcePathRange readRequestSource = "pathRange"
	readRequestSourceLocator   readRequestSource = "locator"
)

type readRequest struct {
	RawEntry     string
	FilePath     string
	ResolvedPath string
	AllowedRoots []string
	StartLine    int
	EndLine      int
	Source       readRequestSource
	Locator      *locator.Location
	Detail       readDetailMode
	RangeEntry   string
}

func buildReadRequestsFromPaths(paths []string, detail readDetailMode) []readRequest {
	requests := make([]readRequest, 0, len(paths))
	for _, entry := range paths {
		path, startLine, endLine := parsePath(entry)
		source := readRequestSourcePathWhole
		if startLine > 0 || endLine > 0 {
			source = readRequestSourcePathRange
		}
		requests = append(requests, readRequest{
			RawEntry:  entry,
			FilePath:  path,
			StartLine: startLine,
			EndLine:   endLine,
			Source:    source,
			Detail:    detail,
		})
	}
	return requests
}

func validateReadRequests(requests []readRequest) string {
	return validateReadRequestCount(len(requests))
}

func formatReadRangeEntry(path string, startLine, endLine int) string {
	switch {
	case startLine > 0 && endLine > 0:
		return fmt.Sprintf("%s:%d-%d", path, startLine, endLine)
	case startLine > 0:
		return fmt.Sprintf("%s:%d", path, startLine)
	default:
		return path
	}
}

func dedupeReadRequestKey(req readRequest) string {
	return fmt.Sprintf("%s\x00%d\x00%d", req.readPath(), req.StartLine, req.EndLine)
}

func (req readRequest) readPath() string {
	if strings.TrimSpace(req.ResolvedPath) != "" {
		return req.ResolvedPath
	}
	return req.FilePath
}
