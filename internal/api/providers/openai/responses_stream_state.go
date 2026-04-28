package openai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ResponsesStreamingOptions は Responses API streaming 解析時の provider 差分を表す。
type ResponsesStreamingOptions struct {
	ProviderName string
	DebugName    string
	Debug        bool
	// DebugOverride が non-nil の場合はその値を優先する。
	// nil の場合は後方互換のため Debug=true を優先し、未指定時は環境変数で判定する。
	DebugOverride *bool
	DebugWriter   io.Writer
	UsageCallback api.UsageCallback
}

type responsesStreamStateOptions struct {
	providerName string
	debugName    string
	debug        bool
}

type responsesStreamState struct {
	providerName  string
	debugName     string
	spinner       *ui.Spinner
	errOut        io.Writer
	debug         bool
	responseID    string
	functionCalls map[string]*responsesFunctionCallAccumulator
	callOrder     []string
	toolCallsOut  strings.Builder
	lastUsage     *api.Usage
}

func newResponsesStreamState(spinner *ui.Spinner, errOut io.Writer) *responsesStreamState {
	return newResponsesStreamStateWithOptions(spinner, errOut, responsesStreamStateOptions{
		providerName: "OpenAI",
		debugName:    "OpenAI",
		debug:        os.Getenv("XELYON_DEBUG_OPENAI") == "1",
	})
}

func newResponsesStreamStateWithOptions(spinner *ui.Spinner, errOut io.Writer, options responsesStreamStateOptions) *responsesStreamState {
	providerName := strings.TrimSpace(options.providerName)
	if providerName == "" {
		providerName = "OpenAI"
	}
	debugName := strings.TrimSpace(options.debugName)
	if debugName == "" {
		debugName = providerName
	}
	return &responsesStreamState{
		providerName:  providerName,
		debugName:     debugName,
		spinner:       spinner,
		errOut:        errOut,
		debug:         options.debug,
		functionCalls: make(map[string]*responsesFunctionCallAccumulator),
	}
}

func (s *responsesStreamState) parseLine(line string) (string, bool, error) {
	if s.debug && line != "" {
		s.debugf("[DEBUG %s Responses] SSE line: %s\n", s.debugName, line)
	}

	data, done, handled := parseResponsesSSEDataLine(line)
	if !handled {
		return "", false, nil
	}
	if done {
		return "", true, nil
	}

	chunk, err := decodeResponsesStreamChunk(data)
	if err != nil {
		return "", false, nil // パースエラーはスキップ
	}

	return s.handleChunk(chunk, data)
}

func (s *responsesStreamState) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}
	fmt.Fprintf(s.errOut, format, args...)
}

func (s *responsesStreamState) handleChunk(chunk ResponsesStreamChunk, rawData string) (string, bool, error) {
	s.logChunkEvent(chunk, rawData)
	action, ok := responsesChunkActionTable[chunk.Type]
	if !ok {
		return "", false, nil
	}

	result := action(s, chunk)
	return result.textDelta, result.done, result.err
}

func (s *responsesStreamState) logChunkEvent(chunk ResponsesStreamChunk, rawData string) {
	s.debugf("[DEBUG %s Responses] event: %s\n", s.debugName, chunk.Type)
	if chunk.Type == "response.completed" {
		s.debugf("[DEBUG %s Responses] raw data: %s\n", s.debugName, rawData)
	}
}

func (s *responsesStreamState) captureResponseID(chunk ResponsesStreamChunk) {
	if chunk.Type == "response.created" && chunk.Response != nil {
		s.responseID = chunk.Response.ID
	}
}

func (s *responsesStreamState) handleErrorEvent(chunk ResponsesStreamChunk) (bool, error) {
	if chunk.Type == "error" {
		errMsg := fmt.Sprintf("%s API error", s.providerName)
		if chunk.Error != nil {
			if chunk.Error.Message != "" {
				errMsg = chunk.Error.Message
			} else if chunk.Error.Code != "" {
				errMsg = fmt.Sprintf("%s API error: %s", s.providerName, chunk.Error.Code)
			}
		}
		return true, fmt.Errorf("%s", errMsg)
	}

	if chunk.Type == "response.failed" {
		return true, fmt.Errorf("%s Responses API request failed", s.providerName)
	}

	return false, nil
}

// HandleResponsesStreaming は Responses API のストリーミングを処理する。
// Response ID も抽出して返却する（content, responseID, error）。
func HandleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner, options ResponsesStreamingOptions) (string, string, error) {
	errOut := options.DebugWriter
	if errOut == nil {
		errOut = api.ErrorWriterFromContext(ctx)
	}
	providerName := strings.TrimSpace(options.ProviderName)
	if providerName == "" {
		providerName = "OpenAI"
	}
	debugName := strings.TrimSpace(options.DebugName)
	if debugName == "" {
		debugName = providerName
	}
	state := newResponsesStreamStateWithOptions(spinner, errOut, responsesStreamStateOptions{
		providerName: providerName,
		debugName:    debugName,
		debug:        resolveResponsesStreamingDebug(options),
	})
	content, err := api.ParseStreamingResponse(ctx, resp, spinner, state.parseLine)
	return newResponsesStreamFinalizePolicy(state, options.UsageCallback).finalize(content, err)
}

func resolveResponsesStreamingDebug(options ResponsesStreamingOptions) bool {
	if options.DebugOverride != nil {
		return *options.DebugOverride
	}
	if options.Debug {
		return true
	}
	return os.Getenv("XELYON_DEBUG_OPENAI") == "1"
}

// handleResponsesStreaming は OpenAI provider 用の Responses API streaming handler。
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	debugEnabled := os.Getenv("XELYON_DEBUG_OPENAI") == "1"
	return HandleResponsesStreaming(ctx, resp, spinner, ResponsesStreamingOptions{
		ProviderName:  "OpenAI",
		DebugName:     "OpenAI",
		Debug:         debugEnabled,
		DebugOverride: &debugEnabled,
		DebugWriter:   api.ErrorWriterFromContext(ctx),
		UsageCallback: p.usageCallback,
	})
}
