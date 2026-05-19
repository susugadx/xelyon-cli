package cmd

import "testing"

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

func requireDoctorJSONRequestPreviewURLs(t *testing.T, preview doctorJSONRequestPreview, wantCount int, wantURL string) {
	t.Helper()
	requireDoctorJSONRequestPreviewCount(t, preview, wantCount)
	requireDoctorJSONRequestPreviewAllURLs(t, preview, wantURL)
}

func requireDoctorJSONRequestPreviewAllURLs(t *testing.T, preview doctorJSONRequestPreview, wantURL string) {
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

func requireDoctorJSONRequestPreviewRouteAndURL(t *testing.T, preview doctorJSONRequestPreview, wantCount int, wantRoute, wantURL string) {
	t.Helper()
	requireDoctorJSONRequestPreviewCount(t, preview, wantCount)
	for _, request := range preview.Requests {
		if request.Route != wantRoute || request.URL != wantURL {
			t.Fatalf("request_preview = %#v, want route=%q url=%q", preview, wantRoute, wantURL)
		}
	}
}
