package bedrock

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func (p *Provider) chatWithConverseStream(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, req bedrockRequestContext) (string, error) {
	return "", unsupportedBedrockConverseRouteError(req.model, req.catalogModel)
}
