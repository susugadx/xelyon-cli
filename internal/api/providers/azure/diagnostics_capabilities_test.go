package azure

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestDiagnosticCapabilitiesFromSnapshot(t *testing.T) {
	snapshot := providerdiag.CapabilitySnapshot{
		RequestModel:       "corp-codex-deployment",
		CatalogModel:       "gpt-5.3-codex",
		Route:              DiagnosticRouteResponsesStreaming,
		RouteReason:        "deployment=corp-codex-deployment uses Responses API; catalog_model=gpt-5.3-codex supports Responses streaming",
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

	got := diagnosticCapabilitiesFromSnapshot(snapshot)
	if got.Deployment != snapshot.RequestModel || got.CatalogModel != snapshot.CatalogModel || got.Route != snapshot.Route || got.RouteReason != snapshot.RouteReason {
		t.Fatalf("route/model projection = %+v, want snapshot values", got)
	}
	if !got.ResponsesAPI || !got.ResponsesStreaming || !got.FunctionCalling || !got.ImageInput {
		t.Fatalf("feature projection = %+v, want enabled features", got)
	}
	if !got.Retention.PreviousResponseID || !got.Retention.SessionPersistence {
		t.Fatalf("retention projection = %+v, want previous_response_id and session persistence", got.Retention)
	}
	if !got.ServerCompaction.RequestPayload || got.ServerCompaction.CompactThreshold != 272000 || !got.ServerCompaction.SkipLocalAutoCompression {
		t.Fatalf("server compaction projection = %+v, want snapshot values", got.ServerCompaction)
	}
	if got.ContextWindowTokens != 400000 || !got.ContextWindowKnown || got.MaxOutputTokens != 128000 || !got.MaxOutputTokensKnown || got.MaxOutputTokensSource != "catalog" || got.MaxOutputRuntimeFallback != 0 {
		t.Fatalf("catalog projection = %+v, want context and max output snapshot values", got)
	}
	if !got.Pricing.Available || got.Pricing.Detail != "pricing=input $1.75/M cached $0.175/M output $14.00/M" {
		t.Fatalf("pricing projection = %+v, want formatted pricing detail", got.Pricing)
	}
}

func TestDiagnosticCapabilitiesFromSnapshotKeepsAzureRuntimeFallback(t *testing.T) {
	got := diagnosticCapabilitiesFromSnapshot(providerdiag.CapabilitySnapshot{
		RequestModel: "corp-gpt52-pro-deployment",
		CatalogModel: "gpt-5.2-pro",
		MaxOutput: providerdiag.MaxOutputPolicy{
			Source:          "missing",
			RuntimeFallback: 16384,
		},
	})

	if got.MaxOutputTokens != 16384 || got.MaxOutputTokensKnown || got.MaxOutputTokensSource != "runtime_fallback" || got.MaxOutputRuntimeFallback != 16384 {
		t.Fatalf("max output projection = %+v, want Azure runtime fallback fields", got)
	}
}
