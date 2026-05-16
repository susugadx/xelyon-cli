package bedrock

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildBedrockDiagnosticRequestPlan(t *testing.T) {
	tests := []struct {
		name        string
		options     DiagnosticOptions
		wantNames   []string
		wantToolUse bool
	}{
		{
			name:      "empty",
			options:   DiagnosticOptions{},
			wantNames: nil,
		},
		{
			name: "print request defaults to text preview",
			options: DiagnosticOptions{
				PrintRequest: true,
			},
			wantNames: []string{bedrockDiagnosticTextRequestName},
		},
		{
			name: "all smoke requests keep stable order",
			options: DiagnosticOptions{
				TextSmoke:     true,
				ToolSmoke:     true,
				ImageSmoke:    true,
				ThinkingSmoke: true,
			},
			wantNames: []string{
				bedrockDiagnosticTextRequestName,
				bedrockDiagnosticToolRequestName,
				bedrockDiagnosticImageRequestName,
				bedrockDiagnosticThinkingRequestName,
			},
			wantToolUse: true,
		},
		{
			name: "text only does not use tool payload",
			options: DiagnosticOptions{
				TextSmoke: true,
			},
			wantNames: []string{bedrockDiagnosticTextRequestName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildBedrockDiagnosticRequestPlan(tt.options)
			var gotNames []string
			for _, request := range plan.Requests {
				gotNames = append(gotNames, request.Name)
			}
			if !reflect.DeepEqual(gotNames, tt.wantNames) {
				t.Fatalf("request names = %v, want %v", gotNames, tt.wantNames)
			}
			if plan.UsesToolPayload() != tt.wantToolUse {
				t.Fatalf("UsesToolPayload() = %v, want %v", plan.UsesToolPayload(), tt.wantToolUse)
			}
		})
	}
}

func TestBedrockDiagnosticRequestMaxOutputTokens(t *testing.T) {
	if got := bedrockDiagnosticRequestMaxOutputTokens(DiagnosticOptions{}); got != defaultBedrockDiagnosticSmokeMaxOutputTokens {
		t.Fatalf("default max output tokens = %d, want %d", got, defaultBedrockDiagnosticSmokeMaxOutputTokens)
	}
	if got := bedrockDiagnosticRequestMaxOutputTokens(DiagnosticOptions{MaxOutputTokens: 123}); got != 123 {
		t.Fatalf("explicit max output tokens = %d, want 123", got)
	}
}

func TestBedrockDiagnosticRequestSkipReason(t *testing.T) {
	tests := []struct {
		name       string
		report     DiagnosticReport
		request    bedrockDiagnosticSmokeRequest
		wantSkip   bool
		wantReason string
	}{
		{
			name: "tool payload skipped when function calling disabled",
			report: DiagnosticReport{
				Route:                  string(bedrockRouteClaudeMessages),
				FunctionCallingEnabled: false,
			},
			request:    bedrockDiagnosticSmokeRequest{Name: bedrockDiagnosticToolRequestName, ToolPayload: true},
			wantSkip:   true,
			wantReason: "function calling payloads are disabled",
		},
		{
			name: "converse image skipped",
			report: DiagnosticReport{
				Route:                  string(bedrockRouteConverseStream),
				FunctionCallingEnabled: true,
			},
			request:    bedrockDiagnosticSmokeRequest{Name: bedrockDiagnosticImageRequestName, ImagePayload: true},
			wantSkip:   true,
			wantReason: "ConverseStream route does not support image or thinking smoke",
		},
		{
			name: "converse text runs",
			report: DiagnosticReport{
				Route:                  string(bedrockRouteConverseStream),
				FunctionCallingEnabled: true,
			},
			request: bedrockDiagnosticSmokeRequest{Name: bedrockDiagnosticTextRequestName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, gotSkip := bedrockDiagnosticRequestSkipReason(tt.report, tt.request)
			if gotSkip != tt.wantSkip {
				t.Fatalf("skip = %v, want %v (reason=%q)", gotSkip, tt.wantSkip, reason)
			}
			if tt.wantReason != "" && !strings.Contains(reason, tt.wantReason) {
				t.Fatalf("reason = %q, want substring %q", reason, tt.wantReason)
			}
		})
	}
}

func TestNewBedrockDiagnosticSkippedSmokeRequest(t *testing.T) {
	request := bedrockDiagnosticSmokeRequest{
		Name:            bedrockDiagnosticThinkingRequestName,
		ToolPayload:     true,
		ImagePayload:    true,
		ThinkingEnabled: true,
	}

	result := newBedrockDiagnosticSkippedSmokeRequest(request, "skip reason")

	if !result.Skipped || result.Ran {
		t.Fatalf("skipped result = %#v, want skipped without ran", result)
	}
	if result.Name != request.Name || result.SkipReason != "skip reason" {
		t.Fatalf("skipped result = %#v, want name and skip reason copied", result)
	}
	if !result.ToolPayload || !result.ImagePayload || !result.ThinkingEnabled {
		t.Fatalf("skipped result payload flags = %#v, want copied flags", result)
	}
}
