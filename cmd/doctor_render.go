package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type doctorCheckLine struct {
	Status     string
	Name       string
	Message    string
	Detail     string
	Suggestion string
}

type doctorSmokeUsageLine struct {
	InputTokens         int
	CachedInputTokens   int
	OutputTokens        int
	ThinkingTokens      int
	CacheCreationTokens int
}

func renderDoctorJSON(w io.Writer, report any) error {
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(payload))
	return err
}

func renderDoctorChecks(w io.Writer, checks []doctorCheckLine) {
	for _, check := range checks {
		fmt.Fprintf(w, "%-4s %s: %s\n", strings.ToUpper(check.Status), check.Name, check.Message)
		if strings.TrimSpace(check.Detail) != "" {
			fmt.Fprintf(w, "     detail: %s\n", check.Detail)
		}
		if strings.TrimSpace(check.Suggestion) != "" {
			fmt.Fprintf(w, "     suggestion: %s\n", check.Suggestion)
		}
	}
}

func renderDoctorRequestPreview(w io.Writer, preview any) {
	renderDoctorJSONBlock(w, "Request preview", preview)
}

func renderDoctorCapabilities(w io.Writer, capabilities any) {
	renderDoctorJSONBlock(w, "Capabilities", capabilities)
}

func renderDoctorJSONBlock(w io.Writer, title string, payloadValue any) {
	var payload bytes.Buffer
	encoder := json.NewEncoder(&payload)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payloadValue); err != nil {
		fmt.Fprintf(w, "%s: (failed to render: %v)\n", title, err)
		return
	}
	fmt.Fprintf(w, "%s:\n%s", title, payload.String())
}

func formatDoctorSmokeUsage(usage doctorSmokeUsageLine) string {
	return fmt.Sprintf(
		"input=%d cached=%d output=%d reasoning=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
}

func doctorOptionalIDText(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "(not returned)"
	}
	return id
}

func doctorSmokeCostText(usageObserved, pricingUnavailable bool, usd float64) string {
	if !usageObserved {
		return "N/A (usage unavailable)"
	}
	if pricingUnavailable {
		return "N/A (pricing unavailable)"
	}
	return fmt.Sprintf("$%.8f USD", usd)
}
