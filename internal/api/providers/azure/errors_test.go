package azure

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHandleAzureResponsesHTTPErrorExplainsCommonStatuses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		headers    http.Header
		body       string
		context    azureHTTPErrorContext
		want       []string
	}{
		{
			name:       "unauthorized credentials",
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"message":"invalid key sk-abcdefghijklmnopqrstuvwxyz"}}`,
			want:       []string{"authentication failed", apiKeyEnv, authTokenEnv, authTokenCommandEnv, "[REDACTED]"},
		},
		{
			name:       "forbidden authorization",
			statusCode: http.StatusForbidden,
			body:       `{"error":{"message":"Principal does not have access"}}`,
			want:       []string{"authorization failed", "role assignment", "deployment access"},
		},
		{
			name:       "not found deployment",
			statusCode: http.StatusNotFound,
			body:       `{"error":{"message":"Deployment not found"}}`,
			context:    azureHTTPErrorContext{Deployment: "corp-gpt55"},
			want:       []string{"resource was not found", baseURLEnv, "corp-gpt55"},
		},
		{
			name:       "rate limit",
			statusCode: http.StatusTooManyRequests,
			headers:    http.Header{"Retry-After": []string{"30"}},
			body:       `{"error":{"message":"Requests to the deployment are rate limited"}}`,
			want:       []string{"rate limit", "quota", "capacity", "Retry-After: 30"},
		},
		{
			name:       "tool payload rejected",
			statusCode: http.StatusBadRequest,
			body:       `{"error":{"message":"The deployment does not support tools"}}`,
			context:    azureHTTPErrorContext{Deployment: "corp-gpt55", ToolPayload: true},
			want:       []string{"tool payload was rejected", "AZURE_OPENAI_FUNCTION_CALLING=0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     tt.headers,
				Body:       io.NopCloser(strings.NewReader(tt.body)),
			}
			err := handleAzureResponsesHTTPError(resp, nil, tt.context)
			if err == nil {
				t.Fatal("handleAzureResponsesHTTPError() error = nil, want error")
			}
			errMsg := err.Error()
			for _, want := range tt.want {
				if !strings.Contains(errMsg, want) {
					t.Fatalf("error = %q, want substring %q", errMsg, want)
				}
			}
			if strings.Contains(errMsg, "sk-abcdefghijklmnopqrstuvwxyz") {
				t.Fatalf("error = %q, should redact API key", errMsg)
			}
		})
	}
}
