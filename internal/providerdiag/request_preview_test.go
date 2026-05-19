package providerdiag

import "testing"

func TestRequestPreviewHeaders(t *testing.T) {
	if got := JSONHeaders(); got["Content-Type"] != "application/json" || len(got) != 1 {
		t.Fatalf("JSONHeaders() = %+v, want JSON content type only", got)
	}

	apiKeyHeaders := RedactedAPIKeyHeaders("x-api-key")
	if apiKeyHeaders["Content-Type"] != "application/json" || apiKeyHeaders["x-api-key"] != "<redacted>" {
		t.Fatalf("RedactedAPIKeyHeaders() = %+v, want JSON and redacted API key", apiKeyHeaders)
	}

	sigV4Headers := RedactedSigV4Headers()
	if sigV4Headers["Content-Type"] != "application/json" || sigV4Headers["Accept"] != "application/json" || sigV4Headers["Authorization"] != "<redacted: AWS SigV4>" {
		t.Fatalf("RedactedSigV4Headers() = %+v, want redacted SigV4 headers", sigV4Headers)
	}
}
