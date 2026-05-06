package kimi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

func buildKimiPromptCacheKey(ctx context.Context, model, systemPrompt string) string {
	scope, ok := api.PromptCacheScopeFromContext(ctx)
	if !ok || scope.SessionID == "" {
		return openaicompat.BuildPromptCacheKey(model, systemPrompt)
	}

	scopeInput := "session:" + scope.SessionID
	if scope.TaskID != "" {
		scopeInput += "\x00task:" + scope.TaskID
	}
	modelInput := "model:" + strings.TrimSpace(model)
	return fmt.Sprintf("xelyon:kimi:v1:%s:%s", shortKimiPromptCacheHash(scopeInput), shortKimiPromptCacheHash(modelInput))
}

func shortKimiPromptCacheHash(input string) string {
	sum := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", sum[:4])
}
