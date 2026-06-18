package readtool

import (
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
)

type readFileContext struct {
	out              common.Output
	cache            tools.ToolCacheInterface
	path             string
	absPath          string
	showFileInfo     bool
	outlineThreshold int
	fileInfo         os.FileInfo
	fileSize         int64
}

func newReadFileContext(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, path string, outlineThreshold int) (readFileContext, string) {
	return newReadFileContextResolved(out, cfg, cache, path, path, nil, outlineThreshold)
}

func newReadFileContextForRequest(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, req readRequest, outlineThreshold int) (readFileContext, string) {
	return newReadFileContextResolved(out, cfg, cache, req.FilePath, req.readPath(), req.AllowedRoots, outlineThreshold)
}

func newReadFileContextResolved(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, displayPath, readPath string, allowedRoots []string, outlineThreshold int) (readFileContext, string) {
	absPath, errResult := pathpolicy.ResolveValidatedPathWithRoots(out, readPath, allowedRoots, "path is empty")
	if errResult != "" {
		return readFileContext{}, errResult
	}

	showFileInfo := cfg != nil && cfg.Streaming.ShowFileInfo
	fileInfo, fileSize, errResult := statReadFile(absPath, showFileInfo)
	if errResult != "" {
		return readFileContext{}, errResult
	}

	return readFileContext{
		out:              out,
		cache:            cache,
		path:             displayPath,
		absPath:          absPath,
		showFileInfo:     showFileInfo,
		outlineThreshold: outlineThreshold,
		fileInfo:         fileInfo,
		fileSize:         fileSize,
	}, ""
}
