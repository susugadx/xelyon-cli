package agent

import "testing"

func TestLookupConfiguredModelThreshold_ExactMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4": 260000,
	}
	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4")
	if !ok {
		t.Fatal("expected exact match to succeed")
	}
	if got != 260000 {
		t.Errorf("got %d, want 260000", got)
	}
}

func TestLookupConfiguredModelThreshold_WildcardMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4*": 260000,
	}
	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-preview")
	if !ok {
		t.Fatal("expected wildcard match to succeed")
	}
	if got != 260000 {
		t.Errorf("got %d, want 260000", got)
	}
}

func TestLookupConfiguredModelThreshold_LongestPrefixWins(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5*":       100000,
		"openai:gpt-5.4*":     260000,
		"openai:gpt-5.4-pro*": 270000,
	}

	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-pro-latest")
	if !ok {
		t.Fatal("expected longest prefix match to succeed")
	}
	if got != 270000 {
		t.Errorf("got %d, want 270000 (longest prefix)", got)
	}

	got2, ok2 := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-mini")
	if !ok2 {
		t.Fatal("expected wildcard match for gpt-5.4-mini to succeed")
	}
	if got2 != 260000 {
		t.Errorf("got %d, want 260000", got2)
	}
}

func TestLookupConfiguredModelThreshold_NoMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4*": 260000,
	}
	_, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.2")
	if ok {
		t.Error("expected no match for gpt-5.2 against gpt-5.4*")
	}
}

func TestLookupConfiguredModelThreshold_EmptyThresholds(t *testing.T) {
	_, ok := lookupConfiguredModelThreshold(nil, "openai", "gpt-5.4")
	if ok {
		t.Error("expected no match with nil thresholds")
	}

	_, ok = lookupConfiguredModelThreshold(map[string]int{}, "openai", "gpt-5.4")
	if ok {
		t.Error("expected no match with empty thresholds")
	}
}

func TestLookupConfiguredModelThreshold_EmptyProviderOrModel(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4": 260000,
	}
	_, ok := lookupConfiguredModelThreshold(thresholds, "", "gpt-5.4")
	if ok {
		t.Error("expected no match with empty provider")
	}

	_, ok = lookupConfiguredModelThreshold(thresholds, "openai", "")
	if ok {
		t.Error("expected no match with empty model")
	}
}
