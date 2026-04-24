package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (m *Manager) callToolWithRetry(
	ctx context.Context,
	session *mcp.ClientSession,
	toolName string,
	params *mcp.CallToolParams,
) (*mcp.CallToolResult, error) {
	var result *mcp.CallToolResult
	var callErr error
	attempted := 0

	for attempt := 1; attempt <= toolCallMaxAttempts; attempt++ {
		attempted = attempt
		result, callErr = session.CallTool(ctx, params)
		if callErr == nil && result != nil && !result.IsError {
			return result, nil
		}
		if !shouldRetryToolCall(ctx, attempt, callErr, result) {
			break
		}

		fmt.Fprintf(
			m.out(),
			"⚠️  MCP tool '%s' call attempt %d failed: %s (retrying...)\n",
			toolName,
			attempt,
			toolCallFailureReason(callErr, result),
		)
		time.Sleep(toolRetryDelay(attempt))
	}

	if callErr != nil {
		return nil, fmt.Errorf("failed to call tool after %d attempts: %w", attempted, callErr)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to call tool after %d attempts: empty response", attempted)
	}
	return result, nil
}

func shouldRetryToolCall(ctx context.Context, attempt int, callErr error, result *mcp.CallToolResult) bool {
	if attempt >= toolCallMaxAttempts {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	if callErr != nil {
		if errors.Is(callErr, context.Canceled) || errors.Is(callErr, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	return result != nil && result.IsError
}

func toolRetryDelay(attempt int) time.Duration {
	shift := min(attempt-1, 30)
	return time.Duration(1<<uint(shift)) * time.Second
}

func toolCallFailureReason(callErr error, result *mcp.CallToolResult) string {
	if callErr != nil {
		return callErr.Error()
	}
	if result != nil && result.IsError {
		return toolResultErrorMessage(result)
	}
	return "tool returned error"
}

func toolResultErrorMessage(result *mcp.CallToolResult) string {
	errMsg := "tool returned error"
	if result == nil {
		return errMsg
	}
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			errMsg = textContent.Text
			break
		}
	}
	return errMsg
}

func toolResultText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	var output strings.Builder
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			output.WriteString(textContent.Text)
			output.WriteByte('\n')
		}
	}
	return output.String()
}
