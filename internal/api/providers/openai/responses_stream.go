package openai

import (
	"context"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, string, error) {
	debugEnabled := os.Getenv("XELYON_DEBUG_OPENAI") == "1"
	return openairesponses.HandleStreaming(ctx, resp, spinner, openairesponses.StreamingOptions{
		ProviderName:        "OpenAI",
		DebugName:           "OpenAI",
		Debug:               debugEnabled,
		DebugOverride:       &debugEnabled,
		DebugWriter:         api.ErrorWriterFromContext(ctx),
		UsageCallback:       p.usageCallback,
		ReplayItemsCallback: p.setLastOpenAIResponsesInputItems,
	})
}
