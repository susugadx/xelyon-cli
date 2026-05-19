package cmd

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setDoctorCommandFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("set --%s=%s: %v", name, value, err)
	}
}

func requireDoctorJSONFields(t *testing.T, raw map[string]json.RawMessage, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, ok := raw[field]; !ok {
			t.Fatalf("JSON field %q missing from report keys %v", field, sortedDoctorJSONFieldNames(raw))
		}
	}
}

func requireDoctorJSONFieldsOmitted(t *testing.T, raw map[string]json.RawMessage, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, ok := raw[field]; ok {
			t.Fatalf("JSON field %q should be omitted from report keys %v", field, sortedDoctorJSONFieldNames(raw))
		}
	}
}

func sortedDoctorJSONFieldNames(raw map[string]json.RawMessage) []string {
	fields := make([]string, 0, len(raw))
	for field := range raw {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func requireDoctorJSONPreviewRequests(t *testing.T, preview doctorJSONRequestPreview, wantCount int, wants []doctorJSONPreviewRequestContract) {
	t.Helper()
	requireDoctorJSONRequestPreviewCount(t, preview, wantCount)
	for _, want := range wants {
		request := requireDoctorJSONRequestPreviewByName(t, preview, want.name)
		requireDoctorJSONPreviewRequestContract(t, request, want)
	}
}

func requireDoctorJSONPreviewSetupContract(t *testing.T, report doctorJSONContractReport, want doctorJSONPreviewSetupContract) {
	t.Helper()
	if want.apiURL != "" && report.APIURL != want.apiURL {
		t.Fatalf("api_url = %q, want setup URL %q", report.APIURL, want.apiURL)
	}
	if want.networkRequests != nil && want.networkRequests.Load() != 0 {
		t.Fatalf("print-request sent %d network requests, want 0", want.networkRequests.Load())
	}
}

func requireDoctorJSONRequestPreviewByName(t *testing.T, preview doctorJSONRequestPreview, name string) doctorJSONRequestPreviewRequest {
	t.Helper()
	var found *doctorJSONRequestPreviewRequest
	for i := range preview.Requests {
		if preview.Requests[i].Name != name {
			continue
		}
		if found != nil {
			t.Fatalf("request_preview has duplicate request %q: %#v", name, preview.Requests)
		}
		found = &preview.Requests[i]
	}
	if found == nil {
		t.Fatalf("request_preview missing request %q: %#v", name, preview.Requests)
	}
	return *found
}

func requireDoctorJSONPreviewRequestContract(t *testing.T, request doctorJSONRequestPreviewRequest, want doctorJSONPreviewRequestContract) {
	t.Helper()
	if request.Skipped {
		t.Fatalf("request %q is skipped, want sendable preview request: %+v", want.name, request)
	}
	if request.Route != want.route {
		t.Fatalf("request %q route = %q, want %q", want.name, request.Route, want.route)
	}
	if request.Operation != want.operation {
		t.Fatalf("request %q operation = %q, want %q", want.name, request.Operation, want.operation)
	}
	if request.ModelID != want.modelID {
		t.Fatalf("request %q model_id = %q, want %q", want.name, request.ModelID, want.modelID)
	}
	if request.Method != want.method {
		t.Fatalf("request %q method = %q, want %q", want.name, request.Method, want.method)
	}
	if request.ToolPayload != want.toolPayload ||
		request.ImagePayload != want.imagePayload ||
		request.WebSearchPayload != want.webSearchPayload ||
		request.RetentionPayload != want.retentionPayload ||
		request.ThinkingEnabled != want.thinkingEnabled {
		t.Fatalf("request %q payload flags = tool:%t image:%t web:%t retention:%t thinking:%t, want tool:%t image:%t web:%t retention:%t thinking:%t",
			want.name,
			request.ToolPayload,
			request.ImagePayload,
			request.WebSearchPayload,
			request.RetentionPayload,
			request.ThinkingEnabled,
			want.toolPayload,
			want.imagePayload,
			want.webSearchPayload,
			want.retentionPayload,
			want.thinkingEnabled,
		)
	}
	if want.previousResponseID && request.PreviousResponseID == "" {
		t.Fatalf("request %q previous_response_id is empty", want.name)
	}
	requireContainsAll(t, "request "+want.name+" url", request.URL, want.urlContains)
	for header, value := range want.headers {
		requireDoctorJSONRequestPreviewHeader(t, request, header, value)
	}
	if len(want.bodyContains) > 0 || len(want.bodyOmittedTopFields) > 0 {
		body := requireDoctorJSONRequestPreviewBodyMap(t, request)
		requireDoctorJSONPreviewBodyContains(t, body, want.bodyContains...)
		requireDoctorJSONPreviewBodyAbsent(t, body, want.bodyOmittedTopFields...)
	}
	requireDoctorJSONPreviewRedaction(t, request)
}

func requireDoctorJSONPreviewRedaction(t *testing.T, request doctorJSONRequestPreviewRequest) {
	t.Helper()
	rendered := renderedDoctorContractValue(t, request)
	for _, secret := range []string{"sk-test", "gsk-test", "sk-or-test", "moonshot-key", "gemini-key", "claude-key", "azure-key"} {
		if strings.Contains(rendered, secret) {
			t.Fatalf("request %q preview leaked secret %q: %s", request.Name, secret, rendered)
		}
	}
}
