// Package all は組み込み LLM provider をすべて登録する。
package all

import (
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	_ "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
)
