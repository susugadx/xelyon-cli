package claude

import (
	"reflect"
	"testing"
)

func TestClaudeDiagnosticRequestPlan(t *testing.T) {
	tests := []struct {
		name                   string
		options                DiagnosticOptions
		functionCallingEnabled bool
		wantNames              []string
	}{
		{
			name:                   "default text smoke",
			options:                DiagnosticOptions{},
			functionCallingEnabled: true,
			wantNames:              []string{claudeDiagnosticTextRequestName},
		},
		{
			name:                   "tool smoke only when function calling enabled",
			options:                DiagnosticOptions{ToolSmoke: true},
			functionCallingEnabled: true,
			wantNames:              []string{claudeDiagnosticToolRequestName},
		},
		{
			name:                   "disabled tool smoke keeps text fallback and skipped tool entry",
			options:                DiagnosticOptions{ToolSmoke: true},
			functionCallingEnabled: false,
			wantNames:              []string{claudeDiagnosticTextRequestName, claudeDiagnosticToolRequestName},
		},
		{
			name:                   "all requested payloads keep stable order",
			options:                DiagnosticOptions{TextSmoke: true, ToolSmoke: true, ImageSmoke: true, ThinkingSmoke: true, WebSearchSmoke: true},
			functionCallingEnabled: true,
			wantNames: []string{
				claudeDiagnosticTextRequestName,
				claudeDiagnosticToolRequestName,
				claudeDiagnosticImageRequestName,
				claudeDiagnosticThinkingRequestName,
				claudeDiagnosticWebSearchRequestName,
			},
		},
		{
			name:                   "web search only does not add text fallback",
			options:                DiagnosticOptions{WebSearchSmoke: true},
			functionCallingEnabled: true,
			wantNames:              []string{claudeDiagnosticWebSearchRequestName},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildClaudeDiagnosticRequestPlan(tt.options, tt.functionCallingEnabled)
			if !reflect.DeepEqual(claudeDiagnosticRequestNames(plan.Requests), tt.wantNames) {
				t.Fatalf("request names = %#v, want %#v", claudeDiagnosticRequestNames(plan.Requests), tt.wantNames)
			}
		})
	}
}

func TestClaudeDiagnosticRequestMaxOutputTokens(t *testing.T) {
	if got := claudeDiagnosticRequestMaxOutputTokens(DiagnosticOptions{MaxOutputTokens: 8}); got != 8 {
		t.Fatalf("claudeDiagnosticRequestMaxOutputTokens(explicit) = %d, want 8", got)
	}
	if got := claudeDiagnosticRequestMaxOutputTokens(DiagnosticOptions{}); got != defaultClaudeDiagnosticSmokeMaxOutputTokens {
		t.Fatalf("claudeDiagnosticRequestMaxOutputTokens(default) = %d, want %d", got, defaultClaudeDiagnosticSmokeMaxOutputTokens)
	}
}

func claudeDiagnosticRequestNames(requests []claudeDiagnosticRequest) []string {
	names := make([]string, 0, len(requests))
	for _, request := range requests {
		names = append(names, request.Name)
	}
	return names
}
