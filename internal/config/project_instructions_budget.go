package config

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type instructionByteBudget struct {
	maxFileBytes   int
	maxTotalBytes  int
	usedBytes      int
	hasTotalBudget bool
}

func newInstructionByteBudget(aiCfg AgentInstructionsConfig) instructionByteBudget {
	maxFile := aiCfg.MaxFileBytes
	if maxFile <= 0 {
		maxFile = defaultAgentInstructionsConfig().MaxFileBytes
	}
	maxTotal := aiCfg.MaxTotalBytes
	if maxTotal <= 0 {
		maxTotal = defaultAgentInstructionsConfig().MaxTotalBytes
	}
	return instructionByteBudget{
		maxFileBytes:   maxFile,
		maxTotalBytes:  maxTotal,
		hasTotalBudget: maxTotal > 0,
	}
}

func (b *instructionByteBudget) exhausted() bool {
	if b == nil {
		return false
	}
	if !b.hasTotalBudget {
		return false
	}
	return b.usedBytes >= b.maxTotalBytes
}

func (b *instructionByteBudget) remaining() (remaining int, limited bool) {
	if b == nil || !b.hasTotalBudget {
		return 0, false
	}
	remaining = b.maxTotalBytes - b.usedBytes
	if remaining < 0 {
		return 0, true
	}
	return remaining, true
}

func applyInstructionContentLimits(data []byte, budget *instructionByteBudget) (content string, truncated bool, consumed int) {
	if budget == nil {
		return string(data), false, len(data)
	}

	limit := len(data)
	var notes []string
	if budget.maxFileBytes > 0 && limit > budget.maxFileBytes {
		limit = budget.maxFileBytes
		truncated = true
		notes = append(notes, fmt.Sprintf("[Content truncated after %d bytes by XELYON agent_instructions.max_file_bytes]", budget.maxFileBytes))
	}

	if remaining, limited := budget.remaining(); limited && limit > remaining {
		limit = remaining
		truncated = true
		notes = append(notes, fmt.Sprintf("[Content truncated after %d bytes by XELYON agent_instructions.max_total_bytes]", budget.maxTotalBytes))
	}

	if limit < 0 {
		limit = 0
	}

	prefix := truncateValidUTF8ByBytes(data, limit)
	consumed = len(prefix)
	budget.usedBytes += consumed

	content = string(prefix)
	if truncated {
		trimmed := strings.TrimRight(content, "\n")
		if trimmed == "" {
			content = strings.Join(notes, "\n")
		} else {
			content = trimmed + "\n\n" + strings.Join(notes, "\n")
		}
	}
	return content, truncated, consumed
}

func truncateValidUTF8ByBytes(data []byte, maxBytes int) []byte {
	if maxBytes <= 0 {
		return nil
	}
	if len(data) <= maxBytes {
		return data
	}

	chunk := data[:maxBytes]
	if utf8.Valid(chunk) {
		return chunk
	}

	for len(chunk) > 0 && !utf8.Valid(chunk) {
		_, size := utf8.DecodeLastRune(chunk)
		if size <= 0 {
			chunk = chunk[:len(chunk)-1]
			continue
		}
		chunk = chunk[:len(chunk)-size]
	}
	return chunk
}
