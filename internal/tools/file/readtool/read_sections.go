package readtool

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ReadExecutionSection は単一 read request の描画結果と成否を表す。
type ReadExecutionSection struct {
	Output      string
	Failed      bool
	Observation *tools.RuntimeObservation
}

// ExecuteReadPathsWithDetailSections は path-based read を構造化 section で返す。
func ExecuteReadPathsWithDetailSections(execCtx tools.ExecutionContext, paths []string, detail string) []ReadExecutionSection {
	mode, errResult := resolveReadDetail(detail, "")
	if errResult != "" {
		return []ReadExecutionSection{{Output: errResult, Failed: true}}
	}

	requests := buildReadRequestsFromPaths(paths, mode)
	return executeReadFilesRequestsSections(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		requests,
		0,
		execCtx.EffectiveLocatorRegistry(),
	)
}

// ExecuteReadTargetsWithDetailSections は locator targets の read を構造化 section で返す。
func ExecuteReadTargetsWithDetailSections(execCtx tools.ExecutionContext, targets string, detail string) []ReadExecutionSection {
	mode, errResult := resolveReadDetail(detail, "")
	if errResult != "" {
		return []ReadExecutionSection{{Output: errResult, Failed: true}}
	}

	requests, reg, result, err := resolveReadTargets(execCtx, targets, "", mode)
	if result != "" {
		return []ReadExecutionSection{{Output: result, Failed: true}}
	}
	if err != nil {
		return []ReadExecutionSection{{Output: fmt.Sprintf("Error: %v", err), Failed: true}}
	}

	return executeReadFilesRequestsSections(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		requests,
		0,
		reg,
	)
}

// ExecuteResolvedRequestsWithDetailSections は解決済み read request を read_file と同じ runtime で実行する。
func ExecuteResolvedRequestsWithDetailSections(execCtx tools.ExecutionContext, requests []ResolvedRequest, detail string) []ReadExecutionSection {
	mode, errResult := resolveReadDetail(detail, "")
	if errResult != "" {
		return []ReadExecutionSection{{Output: errResult, Failed: true}}
	}

	readRequests := buildReadRequestsFromResolvedRequests(requests, mode)
	return executeReadFilesRequestsSections(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		readRequests,
		0,
		execCtx.EffectiveLocatorRegistry(),
	)
}

func executeReadFilesRequestsSections(out common.Output, cfg *config.Config, cache tools.ToolCacheInterface, requests []readRequest, budgetOverride int, reg *locator.Registry) []ReadExecutionSection {
	prepared := prepareReadRequests(out, cfg, cache, requests)
	if errResult := validateReadRequests(prepared); errResult != "" {
		return []ReadExecutionSection{{Output: errResult, Failed: true}}
	}
	results := readRequestsInParallel(out, cfg, cache, prepared, resolveReadFilesBudget(budgetOverride))
	printReadStatus(out, "📄 Read: %d files\n", len(results))
	return renderReadFilesSections(results, reg)
}

func renderReadExecutionSections(sections []ReadExecutionSection) string {
	rendered := make([]string, 0, len(sections))
	for _, section := range sections {
		if strings.TrimSpace(section.Output) == "" {
			continue
		}
		rendered = append(rendered, section.Output)
	}
	return strings.Join(rendered, "\n\n")
}

// RenderReadExecutionSections は read section 群を read_file と同じ形式で描画する。
func RenderReadExecutionSections(sections []ReadExecutionSection) string {
	return renderReadExecutionSections(sections)
}

// MergeReadExecutionSectionObservations は read section 群の observation を順序保持で統合する。
func MergeReadExecutionSectionObservations(sections []ReadExecutionSection) *tools.RuntimeObservation {
	if len(sections) == 0 {
		return nil
	}
	observations := make([]*tools.RuntimeObservation, 0, len(sections))
	for _, section := range sections {
		observations = append(observations, section.Observation)
	}
	return tools.MergeRuntimeObservations(observations...)
}
