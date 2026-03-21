package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/dev"
)

// runInvestigationPhase は調査フェーズを実行する。
// テキスト出力された計画（Plan JSON）を ExtractPlanJSON/ParsePlan でパースし、Plan を返す。
//
// NOTE: 本フェーズは executeToolCallsWithParallel を使用しない。
// SafetyHigh ツール制約のある独自ループ（executeToolOnly）を持ち、
// 並列実行の対象外である。ツールは1つずつ順次実行される。
func (a *Agent) runInvestigationPhase(ctx context.Context) (*plan.Plan, error) {
	cfg := a.cfg()
	hardLimit := normalizeToolLoopLimit(cfg.General.ToolLoopLimit)

	var lastToolCall *tools.ToolCall
	sameCallCount := 0

	for i := 0; ; i++ {
		if hardLimit > 0 && i >= hardLimit {
			return nil, fmt.Errorf("investigation phase exceeded max iterations (%d)", hardLimit)
		}
		if hardLimit == 0 {
			emitLoopWarning(a, i)
		}

		response, err := a.CurrentProvider.ChatWithTools(
			a.requestContext(ctx),
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return nil, fmt.Errorf("API call failed: %w", err)
		}

		toolCalls := a.parseToolCalls(response)

		// assistant message を履歴に追加
		assistantMsg := api.Message{Role: "assistant", Content: response, ReasoningContent: a.getLastReasoningContent()}
		a.History = append(a.History, assistantMsg)
		if a.session != nil {
			a.appendSessionMessageFromAPI(assistantMsg, a.CurrentModel)
		}
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		if len(toolCalls) == 0 {
			if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
				_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG runInvestigationPhase] ParseToolCalls returned 0 tools\n")
				if strings.Contains(response, `{"tool"`) || strings.Contains(response, `{ "tool"`) {
					_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG runInvestigationPhase] WARNING: tool pattern exists but not parsed!\n")
					if len(response) > 200 {
						_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG runInvestigationPhase] tail: ...%s\n", response[len(response)-200:])
					}
				}
			}

			if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
				if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
					_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG runInvestigationPhase] found plan JSON (%d bytes)\n", len(planJSON))
				}
				if p, err := plan.ParsePlan(planJSON); err == nil && len(p.Steps) > 0 {
					return p, nil
				}

				// パース失敗: JSON スキーマ例つきで再提示を促す（番号付きリストだと ExtractPlanJSON が拾えない）
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[SYSTEM] Plan JSON を**必ず**次のスキーマ例に沿って、```json``` で囲んだ1つのJSONとして出力してください（箇条書き/番号付きリスト/文章のみは禁止）。\n\n例:\n```json\n{\n  \"title\": \"調査と実装計画\",\n  \"goal\": \"<最終的に達成したいこと>\",\n  \"assumptions\": [\"<前提>\"],\n  \"steps\": [\n    {\n      \"id\": 1,\n      \"title\": \"<手順タイトル>\",\n      \"description\": \"<この手順でやること>\",\n      \"expected_output\": \"<完了条件/成果物>\"\n    }\n  ]\n}\n```",
				})
				continue
			}

			// ツール呼び出しがない場合は終了（AIが調査を終えて説明している）
			_, _ = fmt.Fprintln(a.output(), response)
			return nil, nil
		}

		// ツールを分類して実行
		for idx, tc := range toolCalls {
			safety := tools.GetToolSafety(tc.Tool)

			// SafetyHigh ツール（読み取り専用）
			if safety == tools.SafetyHigh {
				// ループ検知
				if a.shouldAbortToolLoop(tc, lastToolCall, &sameCallCount) {
					return nil, fmt.Errorf("tool loop detected during investigation")
				}
				lastToolCall = tc

				a.executeToolOnly(tc)
				continue
			}

			if tc.Tool == "bash" || tc.Tool == "command" {
				cmd := tc.Args["command"]
				if dev.IsSafeCommand(cmd, a.cfg().Bash) {
					if a.shouldAbortToolLoop(tc, lastToolCall, &sameCallCount) {
						return nil, fmt.Errorf("tool loop detected during investigation")
					}
					lastToolCall = tc
					a.executeToolOnly(tc)
					continue
				}
			}

			// 書き込み系/危険系ツールは実行しない（調査フェーズ）
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[SYSTEM] Tool '%s' is not allowed in investigation phase. Please only use read-only tools.", tc.Tool),
			})
			if idx == len(toolCalls)-1 {
				continue
			}
		}
	}

}
