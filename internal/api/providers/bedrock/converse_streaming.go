package bedrock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type converseStreamState struct {
	toolUses        map[int32]*converseToolUseState
	toolCallsOutput strings.Builder
	lastUsage       *api.Usage
	spinner         *ui.Spinner
}

type converseToolUseState struct {
	id    string
	name  string
	input strings.Builder
}

func newConverseStreamState(spinner *ui.Spinner) *converseStreamState {
	return &converseStreamState{
		toolUses: make(map[int32]*converseToolUseState),
		spinner:  spinner,
	}
}

func (s *converseStreamState) finalContent(content string) string {
	if s == nil || s.toolCallsOutput.Len() == 0 {
		return content
	}
	if content != "" {
		return content + s.toolCallsOutput.String()
	}
	return s.toolCallsOutput.String()
}

func (s *converseStreamState) startToolUse(index int32, block types.ToolUseBlockStart) {
	s.toolUses[index] = &converseToolUseState{
		id:   aws.ToString(block.ToolUseId),
		name: aws.ToString(block.Name),
	}
}

func (s *converseStreamState) appendToolInput(index int32, input string) {
	toolUse := s.toolUses[index]
	if toolUse == nil {
		toolUse = &converseToolUseState{}
		s.toolUses[index] = toolUse
	}
	toolUse.input.WriteString(input)
	if s.spinner != nil && toolUse.name != "" && !s.spinner.IsActive() {
		s.spinner.Start(ui.SpinnerMessageForTool(toolUse.name))
	}
}

func (s *converseStreamState) stopToolUse(index int32) error {
	toolUse := s.toolUses[index]
	if toolUse == nil {
		return nil
	}
	delete(s.toolUses, index)
	if toolUse.name == "" {
		return fmt.Errorf("bedrock converse tool_use at index %d has no tool name", index)
	}

	args, err := parseConverseToolArguments(toolUse.input.String())
	if err != nil {
		return fmt.Errorf("bedrock converse tool_use %q input: %w", toolUse.id, err)
	}
	toolJSON, err := claude.ConvertToolUseToToolJSON(toolUse.id, toolUse.name, args)
	if err != nil {
		return fmt.Errorf("bedrock converse tool_use %q conversion failed: %w", toolUse.id, err)
	}
	s.toolCallsOutput.WriteString(toolJSON)
	return nil
}

func (s *converseStreamState) recordUsage(usage *types.TokenUsage) {
	if usage == nil {
		return
	}
	s.lastUsage = &api.Usage{
		InputTokens:         int32Value(usage.InputTokens),
		OutputTokens:        int32Value(usage.OutputTokens),
		CachedInputTokens:   int32Value(usage.CacheReadInputTokens),
		CacheCreationTokens: int32Value(usage.CacheWriteInputTokens),
	}
}

func int32Value(value *int32) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

func stopConverseSpinner(spinner *ui.Spinner) {
	if spinner != nil {
		spinner.Stop()
	}
}

func (p *Provider) handleConverseStream(ctx context.Context, output *bedrockruntime.ConverseStreamOutput, spinner *ui.Spinner) (string, error) {
	if output == nil || output.GetStream() == nil {
		stopConverseSpinner(spinner)
		return "", fmt.Errorf("bedrock converse stream is unavailable")
	}

	out := api.OutputWriterFromContext(ctx)
	state := newConverseStreamState(spinner)
	var fullResponse strings.Builder
	firstChunk := true

	stream := output.GetStream()
	defer stream.Close()

	idleTimeout := bedrockStreamIdleTimeout(ctx)
	var idleTimer *time.Timer
	var idleTimerCh <-chan time.Time
	if idleTimeout > 0 {
		idleTimer = time.NewTimer(idleTimeout)
		defer stopBedrockIdleTimer(idleTimer)
		idleTimerCh = idleTimer.C
	}

	events := stream.Events()
	for {
		select {
		case <-ctx.Done():
			stopConverseSpinner(spinner)
			return state.finalContent(fullResponse.String()), ctx.Err()

		case <-idleTimerCh:
			stopConverseSpinner(spinner)
			return state.finalContent(fullResponse.String()), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case event, ok := <-events:
			if !ok {
				stopConverseSpinner(spinner)
				if err := stream.Err(); err != nil {
					return "", fmt.Errorf("bedrock converse stream error: %w", err)
				}
				if state.lastUsage != nil && p.usageCallback != nil {
					p.usageCallback(*state.lastUsage)
				}
				return state.finalContent(fullResponse.String()), nil
			}
			resetBedrockIdleTimer(idleTimer, idleTimeout)

			text, err := processConverseStreamEvent(event, state)
			if err != nil {
				stopConverseSpinner(spinner)
				return state.finalContent(fullResponse.String()), err
			}
			if text == "" {
				continue
			}
			if firstChunk {
				stopConverseSpinner(spinner)
				firstChunk = false
				api.PrintAIHeaderWithContext(ctx)
			}
			if api.ShouldStreamAssistantText(ctx) {
				_, _ = fmt.Fprint(out, text)
			}
			fullResponse.WriteString(text)
		}
	}
}

func processConverseStreamEvent(event types.ConverseStreamOutput, state *converseStreamState) (string, error) {
	switch v := event.(type) {
	case *types.ConverseStreamOutputMemberContentBlockStart:
		if start, ok := v.Value.Start.(*types.ContentBlockStartMemberToolUse); ok {
			state.startToolUse(aws.ToInt32(v.Value.ContentBlockIndex), start.Value)
		}
	case *types.ConverseStreamOutputMemberContentBlockDelta:
		switch delta := v.Value.Delta.(type) {
		case *types.ContentBlockDeltaMemberText:
			return delta.Value, nil
		case *types.ContentBlockDeltaMemberToolUse:
			state.appendToolInput(aws.ToInt32(v.Value.ContentBlockIndex), aws.ToString(delta.Value.Input))
		}
	case *types.ConverseStreamOutputMemberContentBlockStop:
		if err := state.stopToolUse(aws.ToInt32(v.Value.ContentBlockIndex)); err != nil {
			return "", err
		}
	case *types.ConverseStreamOutputMemberMetadata:
		state.recordUsage(v.Value.Usage)
	}
	return "", nil
}
