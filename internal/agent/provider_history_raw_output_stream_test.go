package agent

import (
	"strings"
	"testing"
)

func TestProviderHistoryRawOutputContextScannerBoundsLongLineState(t *testing.T) {
	scanner := newProviderHistoryRawOutputContextScanner([]string{"target-9999"}, 128)
	longLine := `{"items":[` +
		strings.Repeat(`{"id":"prefix","value":"`+strings.Repeat("a", 64)+`"},`, 8000) +
		`{"id":"target-9999","value":"matched"},` +
		strings.Repeat(`{"id":"suffix","value":"`+strings.Repeat("b", 64)+`"},`, 8000) +
		`{}]}`

	for start := 0; start < len(longLine); start += 4096 {
		end := start + 4096
		if end > len(longLine) {
			end = len(longLine)
		}
		if err := scanner.Scan([]byte(longLine[start:end])); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		if len(scanner.remainder) > providerHistoryRawOutputContextMaxLineBytes {
			t.Fatalf("remainder len = %d, want <= %d", len(scanner.remainder), providerHistoryRawOutputContextMaxLineBytes)
		}
	}

	body, reason := scanner.Body()
	if reason != "" || !strings.Contains(body, "target-9999") {
		t.Fatalf("Body() = (%q, %q), want matched bounded excerpt", body, reason)
	}
	if strings.Contains(body, strings.Repeat("a", providerHistoryRawOutputContextMaxLineBytes)) ||
		strings.Contains(body, strings.Repeat("b", providerHistoryRawOutputContextMaxLineBytes)) {
		t.Fatalf("Body() retained an overlong raw line:\n%s", body)
	}
}
