package agent

const runtimeSwitchContextResendWarningTokens = 50_000

// RuntimeSwitchContextNotice は provider/model 切り替え後の context 継続状態。
type RuntimeSwitchContextNotice struct {
	LocalContextKept          bool
	ResponseContinuationReset bool
	ContextResendWarning      bool
	EstimatedNextInputTokens  int
	ContextWindowTokens       int
}

func (a *Agent) runtimeSwitchContextNotice(responseContinuationReset bool) RuntimeSwitchContextNotice {
	if a == nil {
		return RuntimeSwitchContextNotice{}
	}

	notice := RuntimeSwitchContextNotice{
		LocalContextKept:          a.hasLocalConversationContext(),
		ResponseContinuationReset: responseContinuationReset,
	}
	if !notice.LocalContextKept {
		return notice
	}

	notice.EstimatedNextInputTokens = a.EstimateTokens()
	notice.ContextWindowTokens = a.currentModelTokenLimit(a.cfg())
	notice.ContextResendWarning = notice.EstimatedNextInputTokens >= runtimeSwitchContextResendWarningTokens
	return notice
}

func printRuntimeSwitchContextNotice(agent *Agent, notice RuntimeSwitchContextNotice) {
	if agent == nil {
		return
	}
	out := agent.output()
	if notice.LocalContextKept && notice.ResponseContinuationReset {
		yellow.Fprintln(out, "📌 Context kept locally; provider remote continuation reset")
	} else if notice.ResponseContinuationReset {
		yellow.Fprintln(out, "📌 Provider remote continuation reset")
	}
	if !notice.ContextResendWarning {
		return
	}
	if notice.ContextWindowTokens > 0 {
		yellow.Fprintf(out, "⚠️  Next request may resend ~%s tok (%s window)\n",
			FormatTokens(notice.EstimatedNextInputTokens),
			FormatTokens(notice.ContextWindowTokens),
		)
		return
	}
	yellow.Fprintf(out, "⚠️  Next request may resend ~%s tok\n", FormatTokens(notice.EstimatedNextInputTokens))
}
