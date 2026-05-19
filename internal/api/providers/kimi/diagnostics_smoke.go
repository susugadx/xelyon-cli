package kimi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	defaultKimiDiagnosticSmokeTimeout         = 120 * time.Second
	defaultKimiDiagnosticSmokeMaxOutputTokens = 64
)

func runKimiDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultKimiDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultKimiDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	catalogModel := report.CatalogModel
	if strings.TrimSpace(catalogModel) == "" {
		catalogModel = report.Model
	}
	smokeCfg := kimiDiagnosticPolicyConfig(cfg, report.Model, catalogModel, maxOutputTokens)

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(kimiAPIKeyEnv))
	result := DiagnosticSmokeResult{Ran: true}
	plan := buildKimiDiagnosticRequestPlan(options)

	for _, request := range plan.Requests {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			providerdiag.AddKimiSmokeRequestResult(&result, newKimiDiagnosticSkippedToolSmokeRequest(request))
			continue
		}

		requestResult, err := runKimiDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		providerdiag.AddKimiSmokeRequestResult(&result, requestResult)
		if request.ImagePayload {
			result.ImagePayload = true
			if !requestResult.PromptCacheKeyPresent {
				markLastKimiDiagnosticSmokeRequestError(&result, "image smoke request did not include prompt_cache_key")
				return result, fmt.Errorf("image smoke request did not include prompt_cache_key")
			}
		}
		if request.WebSearchPayload {
			result.WebSearchPayload = true
			if !requestResult.PromptCacheKeyPresent {
				markLastKimiDiagnosticSmokeRequestError(&result, "web search smoke request did not include prompt_cache_key")
				return result, fmt.Errorf("web search smoke request did not include prompt_cache_key")
			}
		}
		if request.ToolPayload && err == nil {
			result.ToolPayload = true
			if !diagnosticSmokeContentHasToolCall(requestResult.Content) {
				err := fmt.Errorf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
				markLastKimiDiagnosticSmokeRequestError(&result, err.Error())
				return result, err
			}
		}
		if err != nil {
			return result, err
		}
	}

	if plan.RunTextSmoke {
		first := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheFirstName)
		second := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheSecondName)
		if first == "" || second == "" || first != second {
			return result, fmt.Errorf("session-aware prompt_cache_key mismatch: first=%q second=%q", first, second)
		}
	}

	return result, nil
}

func markLastKimiDiagnosticSmokeRequestError(result *DiagnosticSmokeResult, message string) {
	if result == nil || len(result.Requests) == 0 {
		return
	}
	result.Requests[len(result.Requests)-1].Error = message
}

func diagnosticSmokePromptCacheKey(requests []DiagnosticSmokeRequestResult, name string) string {
	for _, request := range requests {
		if request.Name == name {
			return request.PromptCacheKey
		}
	}
	return ""
}

func isTransientKimiSmokeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "429") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "api error (5") ||
		strings.Contains(message, "status 5") {
		return true
	}
	return false
}
