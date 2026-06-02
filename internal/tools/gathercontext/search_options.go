package gathercontext

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func buildSearchOptions(execCtx tools.ExecutionContext, plan searchPlan) search.SearchOptions {
	opts := search.SearchOptions{
		Pattern:            plan.query,
		Path:               plan.path,
		Mode:               string(search.SearchModeAuto),
		LocatorRegistry:    execCtx.EffectiveLocatorRegistry(),
		ProjectMap:         execCtx.ProjectMap,
		ProjectMapRootPath: execCtx.ProjectMapRootPath,
		ProjectMapStateKey: execCtx.ProjectMapStateKey,
		InvocationCWD:      execCtx.InvocationCWD,
	}
	if plan.preferImpact {
		opts.Intent = "impact"
	}
	if plan.literalPattern {
		opts.PatternInput = search.NewLiteralPatternInput(plan.query)
	}

	opts.FileType, opts.FilePattern = filefilter.Parse(plan.fileFilter)
	attachSearchLSPAdapter(&opts, execCtx)
	return opts
}

func attachSearchLSPAdapter(opts *search.SearchOptions, execCtx tools.ExecutionContext) {
	lspClient := execCtx.EffectiveLSPClient()
	if lspClient == nil {
		return
	}

	if cwd := strings.TrimSpace(execCtx.InvocationCWD); cwd != "" {
		opts.LSPClient = navigation.NewLSPAdapter(lspClient, cwd)
		return
	}
	if cwd, err := os.Getwd(); err == nil {
		opts.LSPClient = navigation.NewLSPAdapter(lspClient, cwd)
	}
}
