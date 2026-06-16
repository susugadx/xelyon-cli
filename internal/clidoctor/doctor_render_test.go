package clidoctor

import (
	"bytes"
	"testing"
)

func TestRenderDoctorSmokeRequestLine(t *testing.T) {
	var out bytes.Buffer
	renderDoctorSmokeRequestLine(&out, doctorSmokeRequestLine{
		Name:               "web_search",
		Route:              "generate_content",
		Duration:           "3ms",
		Content:            "Summary:\nweb search ok",
		Error:              "request failed",
		UsageObserved:      false,
		PricingUnavailable: false,
		CostUSD:            0.00001,
		Usage: doctorSmokeUsageLine{
			InputTokens:         10,
			CachedInputTokens:   3,
			OutputTokens:        4,
			ThinkingTokens:      2,
			CacheCreationTokens: 1,
			BillingServiceTier:  "standard",
		},
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true, PrintError: true, PrintUsageAndCost: true})

	want := "Smoke request web_search: fail route=generate_content duration=3ms\n" +
		"Smoke content web_search: Summary:\n" +
		"web search ok\n" +
		"Smoke error web_search: request failed\n" +
		"Smoke usage web_search: input=10 cached=3 output=4 reasoning=2 cache_creation=1 billing_tier=standard\n" +
		"Smoke cost estimate web_search: N/A (usage unavailable)\n"
	if out.String() != want {
		t.Fatalf("rendered request =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestRenderDoctorSmokeRequestLineWithIDsAndSkipped(t *testing.T) {
	var out bytes.Buffer
	renderDoctorSmokeRequestLine(&out, doctorSmokeRequestLine{
		Name:               "retention_followup",
		Route:              "responses_streaming",
		Duration:           "2ms",
		PreviousResponseID: "resp_initial",
		RetentionPayload:   true,
	}, doctorSmokeRequestRenderOptions{
		IncludeRoute:              true,
		IDLabel:                   "response_id",
		IDValue:                   "",
		IncludePreviousResponseID: true,
	})
	renderDoctorSmokeRequestLine(&out, doctorSmokeRequestLine{
		Name:       "tool",
		Skipped:    true,
		SkipReason: "disabled",
	}, doctorSmokeRequestRenderOptions{IncludeRoute: true})

	want := "Smoke request retention_followup: ok route=responses_streaming duration=2ms response_id=(not returned) previous_response_id=resp_initial\n" +
		"Smoke request tool: skipped (disabled)\n"
	if out.String() != want {
		t.Fatalf("rendered request =\n%s\nwant\n%s", out.String(), want)
	}
}

func TestRenderDoctorSmokeSummary(t *testing.T) {
	var out bytes.Buffer
	renderDoctorSmokeSummary(&out, doctorSmokeSummaryLine{
		Route:              "responses_streaming",
		Duration:           "5ms",
		ResponseID:         "resp_text",
		Content:            "ok",
		Error:              "late error",
		UsageObserved:      true,
		PricingUnavailable: false,
		CostUSD:            0.00012345,
		Usage: doctorSmokeUsageLine{
			InputTokens:       10,
			CachedInputTokens: 3,
			OutputTokens:      4,
			ThinkingTokens:    2,
		},
	}, doctorSmokeSummaryRenderOptions{IncludeRoute: true, IncludeResponseID: true, PrintError: true})

	want := "Smoke route: responses_streaming\n" +
		"Smoke duration: 5ms\n" +
		"Smoke response ID: resp_text\n" +
		"Smoke content: ok\n" +
		"Smoke error: late error\n" +
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=0\n" +
		"Smoke cost estimate: $0.00012345 USD\n"
	if out.String() != want {
		t.Fatalf("rendered summary =\n%s\nwant\n%s", out.String(), want)
	}
}
