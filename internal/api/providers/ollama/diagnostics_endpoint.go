package ollama

import (
	"fmt"
	"net/url"
	"strings"
)

func (r *DiagnosticReport) addEndpointCheck(printRequest bool) (bool, []string) {
	if check := validateOllamaDiagnosticBaseURL(r.APIURL); check != nil {
		r.Checks = append(r.Checks, *check)
		return false, nil
	}

	endpointPathWarned := endpointLooksLikeOllamaAPIPath(r.APIURL)
	if endpointPathWarned {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			fmt.Sprintf("%s is an Ollama API endpoint, but the provider expects a base URL", ollamaBaseURLEnv),
			r.APIURL,
			fmt.Sprintf("Set %s to the Ollama base URL, for example http://localhost:11434", ollamaBaseURLEnv),
		)
		return false, nil
	} else if printRequest {
		r.addCheck(DiagnosticStatusOK, "endpoint", "Ollama endpoint URL is valid for request preview", r.APIURL, "")
		return true, nil
	}

	models, err := New(r.APIURL).ListModels()
	if err != nil {
		r.addCheck(
			DiagnosticStatusFail,
			"endpoint",
			"Ollama endpoint is not reachable or did not return /api/tags",
			err.Error(),
			fmt.Sprintf("Start `ollama serve` or set %s to the correct base URL", ollamaBaseURLEnv),
		)
		return false, nil
	}

	r.addCheck(DiagnosticStatusOK, "endpoint", "Ollama endpoint returned installed models", fmt.Sprintf("%s/api/tags models=%d", strings.TrimRight(r.APIURL, "/"), len(models)), "")
	return true, models
}

func validateOllamaDiagnosticBaseURL(raw string) *DiagnosticCheck {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return &DiagnosticCheck{
			Name:       "endpoint",
			Status:     DiagnosticStatusFail,
			Message:    fmt.Sprintf("%s is not a valid absolute URL", ollamaBaseURLEnv),
			Detail:     raw,
			Suggestion: fmt.Sprintf("Set %s to a valid absolute URL such as http://localhost:11434", ollamaBaseURLEnv),
		}
	}
	return nil
}

func endpointLooksLikeOllamaAPIPath(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	path := strings.TrimRight(parsed.Path, "/")
	return path == ollamaChatEndpointPath || path == "/api/tags"
}

func (r *DiagnosticReport) addInstalledModelCheck(endpointOK bool, installedModels []string, printRequest bool) {
	if strings.TrimSpace(r.Model) == "" {
		return
	}
	if printRequest {
		r.addCheck(DiagnosticStatusOK, "installed_model", "installed model lookup was skipped for request preview", "--print-request", "")
		return
	}
	if !endpointOK {
		r.addCheck(
			DiagnosticStatusWarn,
			"installed_model",
			"installed model lookup was skipped because endpoint check failed",
			"",
			"Fix the endpoint check, then rerun doctor ollama",
		)
		return
	}
	if ollamaInstalledModelMatches(r.Model, installedModels) {
		r.addCheck(DiagnosticStatusOK, "installed_model", "Ollama request model is installed", fmt.Sprintf("model=%s", r.Model), "")
		return
	}
	r.addCheck(
		DiagnosticStatusFail,
		"installed_model",
		"Ollama request model was not found in /api/tags",
		fmt.Sprintf("model=%s installed=%s", r.Model, strings.Join(installedModels, ", ")),
		fmt.Sprintf("Run `ollama pull %s` or pass --model for an installed model", r.Model),
	)
}

func ollamaInstalledModelMatches(model string, installedModels []string) bool {
	model = normalizeOllamaModelTag(model)
	for _, installed := range installedModels {
		if normalizeOllamaModelTag(installed) == model {
			return true
		}
	}
	return false
}

func normalizeOllamaModelTag(model string) string {
	model = strings.TrimSpace(model)
	return strings.TrimSuffix(model, ":latest")
}
