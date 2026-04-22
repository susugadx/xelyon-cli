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

type responsesStreamState struct {
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
	return &responsesStreamState{
		spinner:       spinner,
		errOut:        errOut,
		debug:         os.Getenv("XELYON_DEBUG_OPENAI") == "1",
		functionCalls: make(map[string]*responsesFunctionCallAccumulator),
	}
}

func (s *responsesStreamState) parseLine(line string) (string, bool, error) {
	if s.debug && line != "" {
		s.debugf("[DEBUG OpenAI Responses] SSE line: %s\n", line)
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
	s.debugf("[DEBUG OpenAI Responses] event: %s\n", chunk.Type)
	if chunk.Type == "response.completed" {
		s.debugf("[DEBUG OpenAI Responses] raw data: %s\n", rawData)
	}
}

func (s *responsesStreamState) captureResponseID(chunk ResponsesStreamChunk) {
	if chunk.Type == "response.created" && chunk.Response != nil {
		s.responseID = chunk.Response.ID
	}
}

func (s *responsesStreamState) handleErrorEvent(chunk ResponsesStreamChunk) (bool, error) {
	if chunk.Type == "error" {
		errMsg := "OpenAI API error"
		if chunk.Error != nil {
			if chunk.Error.Message != "" {
				errMsg = chunk.Error.Message
			} else if chunk.Error.Code != "" {
				errMsg = fmt.Sprintf("OpenAI API error: %s", chunk.Error.Code)
			}
		}
		return true, fmt.Errorf("%s", errMsg)
	}

	if chunk.Type == "response.failed" {
		return true, fmt.Errorf("OpenAI Responses API request failed")
	}

	return false, nil
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却（content, responseID, error）
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	state := newResponsesStreamState(spinner, api.ErrorWriterFromContext(ctx))
	content, err := api.ParseStreamingResponse(ctx, resp, spinner, state.parseLine)
	if err != nil {
		return "", state.responseID, err
	}

	// usage コールバックを呼び出し
	if state.lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*state.lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if state.toolCallsOut.Len() > 0 {
		if content != "" {
			return content + state.toolCallsOut.String(), state.responseID, nil
		}
		return state.toolCallsOut.String(), state.responseID, nil
	}
	return content, state.responseID, nil
}
