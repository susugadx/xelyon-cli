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
	BillingServiceTier  string
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

func renderDoctorRequestPreviewSection(w io.Writer, preview any) {
	fmt.Fprintln(w)
	renderDoctorRequestPreview(w, preview)
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
	text := fmt.Sprintf(
		"input=%d cached=%d output=%d reasoning=%d cache_creation=%d",
		usage.InputTokens,
		usage.CachedInputTokens,
		usage.OutputTokens,
		usage.ThinkingTokens,
		usage.CacheCreationTokens,
	)
	if tier := strings.TrimSpace(usage.BillingServiceTier); tier != "" {
		text += fmt.Sprintf(" billing_tier=%s", tier)
	}
	return text
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

type doctorSmokeRequestLine struct {
	Name               string
	Route              string
	Duration           string
	Content            string
	Error              string
	PreviousResponseID string
	Skipped            bool
	SkipReason         string
	RetentionPayload   bool
	UsageObserved      bool
	PricingUnavailable bool
	CostUSD            float64
	Usage              doctorSmokeUsageLine
}

type doctorSmokeRequestRenderOptions struct {
	IncludeRoute              bool
	IDLabel                   string
	IDValue                   string
	IncludePreviousResponseID bool
	OmitContent               bool
	PrintError                bool
	PrintUsageAndCost         bool
}

type doctorSmokeSummaryLine struct {
	Route              string
	Duration           string
	Content            string
	Error              string
	ResponseID         string
	UsageObserved      bool
	PricingUnavailable bool
	CostUSD            float64
	Usage              doctorSmokeUsageLine
}

type doctorSmokeSummaryRenderOptions struct {
	IncludeRoute      bool
	IncludeResponseID bool
	PrintError        bool
}

func renderDoctorSmokeRequestLine(w io.Writer, request doctorSmokeRequestLine, options doctorSmokeRequestRenderOptions) {
	if request.Skipped {
		fmt.Fprintf(w, "Smoke request %s: skipped (%s)\n", request.Name, request.SkipReason)
		return
	}

	line := fmt.Sprintf("Smoke request %s: %s", request.Name, doctorSmokeStatus(request.Error))
	if options.IncludeRoute {
		line += fmt.Sprintf(" route=%s", request.Route)
	}
	line += fmt.Sprintf(" duration=%s", request.Duration)
	if options.IDLabel != "" {
		line += fmt.Sprintf(" %s=%s", options.IDLabel, doctorOptionalIDText(options.IDValue))
	}
	if options.IncludePreviousResponseID && request.RetentionPayload {
		line += fmt.Sprintf(" previous_response_id=%s", doctorOptionalIDText(request.PreviousResponseID))
	}
	fmt.Fprintln(w, line)

	if !options.OmitContent && strings.TrimSpace(request.Content) != "" {
		fmt.Fprintf(w, "Smoke content %s: %s\n", request.Name, request.Content)
	}
	if options.PrintError && strings.TrimSpace(request.Error) != "" {
		fmt.Fprintf(w, "Smoke error %s: %s\n", request.Name, request.Error)
	}
	if options.PrintUsageAndCost {
		fmt.Fprintf(w, "Smoke usage %s: %s\n", request.Name, formatDoctorSmokeUsage(request.Usage))
		fmt.Fprintf(w, "Smoke cost estimate %s: %s\n", request.Name, doctorSmokeCostText(request.UsageObserved, request.PricingUnavailable, request.CostUSD))
	}
}

func renderDoctorSmokeSummary(w io.Writer, smoke doctorSmokeSummaryLine, options doctorSmokeSummaryRenderOptions) {
	if options.IncludeRoute {
		fmt.Fprintf(w, "Smoke route: %s\n", smoke.Route)
	}
	fmt.Fprintf(w, "Smoke duration: %s\n", smoke.Duration)
	if options.IncludeResponseID {
		fmt.Fprintf(w, "Smoke response ID: %s\n", doctorOptionalIDText(smoke.ResponseID))
	}
	if strings.TrimSpace(smoke.Content) != "" {
		fmt.Fprintf(w, "Smoke content: %s\n", smoke.Content)
	}
	if options.PrintError && strings.TrimSpace(smoke.Error) != "" {
		fmt.Fprintf(w, "Smoke error: %s\n", smoke.Error)
	}
	fmt.Fprintf(w, "Smoke usage: %s\n", formatDoctorSmokeUsage(smoke.Usage))
	fmt.Fprintf(w, "Smoke cost estimate: %s\n", doctorSmokeCostText(smoke.UsageObserved, smoke.PricingUnavailable, smoke.CostUSD))
}

func renderDoctorSmokeTotal(w io.Writer, usage doctorSmokeUsageLine, usageObserved, pricingUnavailable bool, costUSD float64) {
	fmt.Fprintf(w, "Smoke total usage: %s\n", formatDoctorSmokeUsage(usage))
	fmt.Fprintf(w, "Smoke total cost estimate: %s\n", doctorSmokeCostText(usageObserved, pricingUnavailable, costUSD))
}

func doctorSmokeStatus(errorText string) string {
	if strings.TrimSpace(errorText) != "" {
		return "fail"
	}
	return "ok"
}
