package cmd

import "testing"

type doctorEndpointContractReport struct {
	APIURL            string                       `json:"api_url"`
	ResponsesURL      string                       `json:"responses_url"`
	NormalizedBaseURL string                       `json:"normalized_base_url"`
	Smoke             any                          `json:"smoke"`
	RequestPreview    doctorJSONRequestPreviewView `json:"request_preview"`
	Checks            []doctorJSONCheck            `json:"checks"`
}

type doctorJSONRequestPreviewView struct {
	Requests []doctorJSONRequestPreviewRequestView `json:"requests"`
}

type doctorJSONRequestPreviewRequestView struct {
	Route string `json:"route"`
	URL   string `json:"url"`
}

func requireDoctorJSONProxyWarning(t *testing.T, checks []doctorJSONCheck, checkName, okCheckName, wantURL string) {
	t.Helper()
	endpoint := requireDoctorJSONCheck(t, checks, checkName)
	requireDoctorJSONCheckStatus(t, endpoint, "warn")
	requireDoctorJSONCheckDetailContains(t, endpoint, wantURL)
	requireDoctorJSONCheckSuggestionContains(t, endpoint, "intentional proxy")
	if okCheckName != "" {
		requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, checks, okCheckName), "ok")
	}
}

func requireDoctorJSONPrintRequestSkippedAuth(t *testing.T, checks []doctorJSONCheck) {
	t.Helper()
	requireNoDoctorJSONChecks(t, checks, "auth")
}

func requireDoctorJSONRequestPreviewURLs(t *testing.T, preview doctorJSONRequestPreviewView, wantCount int, wantURL string) {
	t.Helper()
	if len(preview.Requests) != wantCount {
		t.Fatalf("request_preview = %#v, want %d requests", preview, wantCount)
	}
	requireDoctorJSONRequestPreviewAllURLs(t, preview, wantURL)
}

func requireDoctorJSONRequestPreviewAllURLs(t *testing.T, preview doctorJSONRequestPreviewView, wantURL string) {
	t.Helper()
	if len(preview.Requests) == 0 {
		t.Fatalf("request_preview = %#v, want request previews", preview)
	}
	for _, request := range preview.Requests {
		if request.URL != wantURL {
			t.Fatalf("request_preview = %#v, want request URL %q", preview, wantURL)
		}
	}
}

func requireDoctorJSONRequestPreviewRouteAndURL(t *testing.T, preview doctorJSONRequestPreviewView, wantCount int, wantRoute, wantURL string) {
	t.Helper()
	if len(preview.Requests) != wantCount {
		t.Fatalf("request_preview = %#v, want %d requests", preview, wantCount)
	}
	for _, request := range preview.Requests {
		if request.Route != wantRoute || request.URL != wantURL {
			t.Fatalf("request_preview = %#v, want route=%q url=%q", preview, wantRoute, wantURL)
		}
	}
}
