package kimi

import (
	"reflect"
	"testing"
)

func TestKimiDiagnosticRequestPlan(t *testing.T) {
	tests := []struct {
		name        string
		options     DiagnosticOptions
		wantNames   []string
		wantRunText bool
	}{
		{
			name:        "default text smoke",
			options:     DiagnosticOptions{},
			wantNames:   []string{kimiDiagnosticSmokeCacheFirstName, kimiDiagnosticSmokeCacheSecondName, kimiDiagnosticSmokeThinkingName},
			wantRunText: true,
		},
		{
			name:        "tool smoke keeps text fallback and requested tool",
			options:     DiagnosticOptions{ToolSmoke: true},
			wantNames:   []string{kimiDiagnosticSmokeCacheFirstName, kimiDiagnosticSmokeCacheSecondName, kimiDiagnosticSmokeThinkingName, kimiDiagnosticSmokeToolName},
			wantRunText: true,
		},
		{
			name:        "image smoke only",
			options:     DiagnosticOptions{ImageSmoke: true},
			wantNames:   []string{kimiDiagnosticSmokeImageName},
			wantRunText: false,
		},
		{
			name:        "web search smoke only",
			options:     DiagnosticOptions{WebSearchSmoke: true},
			wantNames:   []string{kimiDiagnosticSmokeWebSearchName},
			wantRunText: false,
		},
		{
			name:        "all payloads",
			options:     DiagnosticOptions{ToolSmoke: true, ImageSmoke: true, WebSearchSmoke: true},
			wantNames:   []string{kimiDiagnosticSmokeCacheFirstName, kimiDiagnosticSmokeCacheSecondName, kimiDiagnosticSmokeThinkingName, kimiDiagnosticSmokeImageName, kimiDiagnosticSmokeWebSearchName, kimiDiagnosticSmokeToolName},
			wantRunText: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildKimiDiagnosticRequestPlan(tt.options)
			if !reflect.DeepEqual(kimiDiagnosticRequestNames(plan.Requests), tt.wantNames) {
				t.Fatalf("request names = %#v, want %#v", kimiDiagnosticRequestNames(plan.Requests), tt.wantNames)
			}
			if plan.RunTextSmoke != tt.wantRunText {
				t.Fatalf("RunTextSmoke = %t, want %t", plan.RunTextSmoke, tt.wantRunText)
			}
		})
	}
}

func kimiDiagnosticRequestNames(requests []kimiDiagnosticSmokeRequest) []string {
	names := make([]string, 0, len(requests))
	for _, request := range requests {
		names = append(names, request.Name)
	}
	return names
}
