package openairesponses

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// StreamingOptions は Responses API streaming 解析時の provider 差分を表す。
type StreamingOptions struct {
	ProviderName string
	DebugName    string
	Debug        bool
	// DebugOverride が non-nil の場合はその値を優先する。
	// nil の場合は後方互換のため Debug=true を優先し、未指定時は環境変数で判定する。
	DebugOverride       *bool
	DebugRawPayload     *bool
	DebugWriter         io.Writer
	UsageCallback       api.UsageCallback
	ReplayItemsCallback func([]api.InputItem)
}

type responsesStreamStateOptions struct {
	providerName    string
	debugName       string
	debug           bool
	debugRawPayload bool
}

type responsesStreamState struct {
	providerName              string
	debugName                 string
	spinner                   *uiruntime.Spinner
	errOut                    io.Writer
	debug                     bool
	debugRawPayload           bool
	responseID                string
	textOut                   strings.Builder
	messages                  map[string]*responsesMessageAccumulator
	currentMessageKey         string
	messageKeysByOutputIndex  map[int]string
	reasoningItems            map[string]api.InputItem
	functionCalls             map[string]*responsesFunctionCallAccumulator
	functionKeysByItemID      map[string]string
	functionKeysByOutputIndex map[int]string
	callOrder                 []string
	replayOrder               []responsesReplayRef
	replayOrderSeen           map[string]struct{}
	toolCallsOut              strings.Builder
	lastUsage                 *api.Usage
}

func newResponsesStreamState(spinner *uiruntime.Spinner, errOut io.Writer) *responsesStreamState {
	return newResponsesStreamStateWithOptions(spinner, errOut, responsesStreamStateOptions{
		providerName:    "OpenAI",
		debugName:       "OpenAI",
		debug:           os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		debugRawPayload: true,
	})
}

func newResponsesStreamStateWithOptions(spinner *uiruntime.Spinner, errOut io.Writer, options responsesStreamStateOptions) *responsesStreamState {
	providerName := strings.TrimSpace(options.providerName)
	if providerName == "" {
		providerName = "OpenAI"
	}
	debugName := strings.TrimSpace(options.debugName)
	if debugName == "" {
		debugName = providerName
	}
	return &responsesStreamState{
		providerName:              providerName,
		debugName:                 debugName,
		spinner:                   spinner,
		errOut:                    errOut,
		debug:                     options.debug,
		debugRawPayload:           options.debugRawPayload,
		messages:                  make(map[string]*responsesMessageAccumulator),
		messageKeysByOutputIndex:  make(map[int]string),
		reasoningItems:            make(map[string]api.InputItem),
		functionCalls:             make(map[string]*responsesFunctionCallAccumulator),
		functionKeysByItemID:      make(map[string]string),
		functionKeysByOutputIndex: make(map[int]string),
		replayOrderSeen:           make(map[string]struct{}),
	}
}

func (s *responsesStreamState) parseLine(line string) (string, bool, error) {
	if s.debug && s.debugRawPayload && line != "" {
		s.debugf("[DEBUG %s Responses] SSE line: %s\n", s.debugName, line)
	}

	data, done, handled := parseResponsesSSEDataLine(line)
	if !handled {
		return "", false, nil
	}
	if done {
		return "", true, nil
	}

	chunk, err := decodeStreamChunk(data)
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

func (s *responsesStreamState) handleChunk(chunk StreamChunk, rawData string) (string, bool, error) {
	s.logChunkEvent(chunk, rawData)
	action, ok := responsesChunkActionTable[chunk.Type]
	if !ok {
		return "", false, nil
	}

	result := action(s, chunk)
	return result.textDelta, result.done, result.err
}

func (s *responsesStreamState) logChunkEvent(chunk StreamChunk, rawData string) {
	s.debugf("[DEBUG %s Responses] event: %s\n", s.debugName, chunk.Type)
	if s.debugRawPayload && chunk.Type == "response.completed" {
		s.debugf("[DEBUG %s Responses] raw data: %s\n", s.debugName, rawData)
	}
}

func (s *responsesStreamState) captureResponseID(chunk StreamChunk) {
	if chunk.Type == "response.created" && chunk.Response != nil {
		s.responseID = chunk.Response.ID
	}
}

func (s *responsesStreamState) handleErrorEvent(chunk StreamChunk) (bool, error) {
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

// HandleStreaming は Responses API のストリーミングを処理する。
// Response ID も抽出して返却する（content, responseID, error）。
func HandleStreaming(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner, options StreamingOptions) (string, string, error) {
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
		providerName:    providerName,
		debugName:       debugName,
		debug:           resolveResponsesStreamingDebug(options),
		debugRawPayload: resolveResponsesStreamingDebugRawPayload(options),
	})
	content, err := api.ParseStreamingResponse(ctx, resp, spinner, state.parseLine)
	if err == nil && options.ReplayItemsCallback != nil {
		options.ReplayItemsCallback(state.openAIResponsesReplayItems())
	}
	return newResponsesStreamFinalizePolicy(state, options.UsageCallback).finalize(content, err)
}

func resolveResponsesStreamingDebug(options StreamingOptions) bool {
	if options.DebugOverride != nil {
		return *options.DebugOverride
	}
	if options.Debug {
		return true
	}
	return os.Getenv("XELYON_DEBUG_OPENAI") == "1"
}

func resolveResponsesStreamingDebugRawPayload(options StreamingOptions) bool {
	if options.DebugRawPayload != nil {
		return *options.DebugRawPayload
	}
	return true
}
