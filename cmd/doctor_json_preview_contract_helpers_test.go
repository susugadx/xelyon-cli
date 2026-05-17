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

func requireDoctorJSONPreviewIdentity(t *testing.T, report doctorJSONContractReport, want doctorJSONPreviewContractIdentity) {
	t.Helper()
	if report.Provider == "" {
		t.Fatal("provider is empty")
	}
	if want.model != "" && (report.Model != want.model || report.ModelSource != want.modelSource) {
		t.Fatalf("model = %q (%s), want %q (%s)", report.Model, report.ModelSource, want.model, want.modelSource)
	}
	if want.deployment != "" && report.Deployment != want.deployment {
		t.Fatalf("deployment = %q, want %q", report.Deployment, want.deployment)
	}
	if want.catalogModel != "" && (report.CatalogModel != want.catalogModel || report.CatalogModelSource != want.catalogModelSource) {
		t.Fatalf("catalog_model = %q (%s), want %q (%s)", report.CatalogModel, report.CatalogModelSource, want.catalogModel, want.catalogModelSource)
	}
	if want.route != "" && report.Route != want.route {
		t.Fatalf("route = %q, want %q", report.Route, want.route)
	}
	if want.region != "" && report.Region != want.region {
		t.Fatalf("region = %q, want %q", report.Region, want.region)
	}
	requireContainsAll(t, "api_url", report.APIURL, want.apiURLContains)
	requireContainsAll(t, "responses_url", report.ResponsesURL, want.responsesURLContains)
	requireContainsAll(t, "normalized_base_url", report.NormalizedBaseURL, want.normalizedURLContains)
}

func requireDoctorJSONPreviewRequests(t *testing.T, preview doctorJSONRequestPreview, wantCount int, wants []doctorJSONPreviewRequestContract) {
	t.Helper()
	requireDoctorJSONRequestPreviewCount(t, preview, wantCount)
	for _, want := range wants {
		request := requireDoctorJSONRequestPreviewByName(t, preview, want.name)
		requireDoctorJSONPreviewRequestContract(t, request, want)
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

func requireContainsAll(t *testing.T, label, got string, wants []string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("%s = %q, want substring %q", label, got, want)
		}
	}
}
