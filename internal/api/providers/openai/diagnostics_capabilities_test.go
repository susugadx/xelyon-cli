package openai

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestOpenAIDiagnosticCapabilitiesFromSnapshot(t *testing.T) {
	snapshot := providerdiag.CapabilitySnapshot{
		RequestModel:       "corp-openai-deployment",
		CatalogModel:       "gpt-5.3-codex",
		Route:              DiagnosticRouteResponsesStreaming,
		RouteReason:        "model=corp-openai-deployment uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
		ResponsesAPI:       true,
		ResponsesStreaming: true,
		FunctionCalling:    true,
		ImageInput:         true,
		Retention:          providerdiag.NewRetentionSnapshot(true, true, true),
		ServerCompaction: providerdiag.ServerCompactionSnapshot{
			Enabled:                  true,
			RequestPayload:           true,
			CompactThreshold:         272000,
			LocalFallback:            true,
			SkipLocalAutoCompression: true,
			Detail:                   "context_management.compaction would be sent with previous_response_id",
		},
		ContextWindowTokens: 400000,
		ContextWindowKnown:  true,
		MaxOutput: providerdiag.MaxOutputPolicy{
			Tokens:    128000,
			Source:    "catalog",
			Available: true,
		},
		Pricing: cost.PricingInfo{
			InputCostPerM:       1.75,
			CachedInputCostPerM: 0.175,
			OutputCostPerM:      14,
		},
	}

	got := openAIDiagnosticCapabilitiesFromSnapshot(snapshot)
	if got.Model != snapshot.RequestModel || got.CatalogModel != snapshot.CatalogModel || got.Route != snapshot.Route || got.RouteReason != snapshot.RouteReason {
		t.Fatalf("route/model projection = %+v, want snapshot values", got)
	}
	if !got.ResponsesAPI || !got.ResponsesStreaming || got.ChatCompletions {
		t.Fatalf("route capability projection = %+v, want Responses streaming only", got)
	}
	if !got.FunctionCalling || !got.ImageInput || !got.Retention.PreviousResponseID || !got.Retention.SessionPersistence {
		t.Fatalf("feature projection = %+v retention=%+v, want enabled features", got, got.Retention)
	}
	if !got.ServerCompaction.RequestPayload || got.ServerCompaction.CompactThreshold != 272000 || !got.ServerCompaction.SkipLocalAutoCompression {
		t.Fatalf("server compaction projection = %+v, want snapshot values", got.ServerCompaction)
	}
	if got.ContextWindowTokens != 400000 || !got.ContextWindowKnown || got.MaxOutputTokens != 128000 || !got.MaxOutputTokensKnown || got.MaxOutputTokensSource != "catalog" {
		t.Fatalf("catalog projection = %+v, want context and max output snapshot values", got)
	}
	if !got.Pricing.Available || got.Pricing.Detail != "pricing=input $1.75/M cached $0.175/M output $14.00/M" {
		t.Fatalf("pricing projection = %+v, want formatted pricing detail", got.Pricing)
	}
}
