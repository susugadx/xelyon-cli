package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/planning"
)

// runInvestigationPhase は調査フェーズを実行
// create_plan ツールが呼び出されるまでループし、作成された Plan を返す
func (a *Agent) runInvestigationPhase(ctx context.Context) (*plan.Plan, error) {
	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// create_plan ツールを取得（LastPlan() で Plan を取得するため）
	createPlanTool := a.getCreatePlanTool()
	if createPlanTool != nil {
		createPlanTool.ClearLastPlan() // 前回の Plan をクリア
	}

	for i := 0; i < maxIterations; i++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return nil, fmt.Errorf("investigation failed: %w", err)
		}

		a.History = append(a.History, api.Message{
			Role:             "assistant",
			Content:          response,
			ReasoningContent: a.getLastReasoningContent(),
		})
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		// デバッグモード: レスポンスの診断情報を出力
		if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] response length: %d\n", len(response))
			if idx := strings.Index(response, `{"tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found {\"tool\" at index %d\n", idx)
			}
			if idx := strings.Index(response, `{ "tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found { \"tool\" at index %d\n", idx)
			}
			codeBlockOpens := strings.Count(response, "```")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] code block markers (```): %d (odd = unclosed)\n", codeBlockOpens)
			openBraces := strings.Count(response, "{")
			closeBraces := strings.Count(response, "}")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] braces: { = %d, } = %d (mismatch = incomplete JSON)\n", openBraces, closeBraces)
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] ParseToolCalls returned 0 tools\n")
				if strings.Contains(response, `{"tool"`) || strings.Contains(response, `{ "tool"`) {
					fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] WARNING: tool pattern exists but not parsed!\n")
					if len(response) > 200 {
						fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] tail: ...%s\n", response[len(response)-200:])
					}
				}
			}

			// テキストフォールバック: FC失敗時にレスポンスから直接 Plan JSON を抽出
			if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
				if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
					fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] text fallback: found plan JSON (%d bytes)\n", len(planJSON))
				}
				if p, err := plan.ParsePlan(planJSON); err == nil && len(p.Steps) > 0 {
					yellow.Println("⚠️  FC fallback: extracted plan from text response")
					// 保存
					if createPlanTool != nil {
						// storage 経由で保存するため create_plan を手動実行
						tc := &tools.ToolCall{
							Tool: "create_plan",
							Args: map[string]string{
								"title":   p.Title,
								"summary": p.Summary,
							},
						}
						// steps を JSON 文字列にシリアライズ
						if stepsJSON, err := json.Marshal(p.Steps); err == nil {
							tc.Args["steps"] = string(stepsJSON)
						}
						tools.Execute(tc)
						if savedPlan := createPlanTool.LastPlan(); savedPlan != nil {
							return savedPlan, nil
						}
					}
					// storage なしでも Plan を返す
					return p, nil
				}
			}

			// ツール呼び出しがない場合は終了（AIが調査を終えて説明している）
			fmt.Println(response)
			return nil, nil
		}

		// ツールを分類して実行
		var allResults []string
		for _, tc := range toolCalls {
			safety := tools.GetToolSafety(tc.Tool)

			// create_plan ツールの場合
			if tc.Tool == "create_plan" {
				if a.Stats != nil {
					a.Stats.AddToolExecution(tc.Tool)
				}
				result, _ := tools.Execute(tc)
				allResults = append(allResults, fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, result))

				// create_plan ツールから Plan を取得
				if createPlanTool != nil {
					if p := createPlanTool.LastPlan(); p != nil {
						// 結果を履歴に追加してから Plan を返す
						a.History = append(a.History, api.Message{
							Role:    "user",
							Content: strings.Join(allResults, "\n\n"),
						})
						return p, nil
					}
				}
				continue
			}

			// SafetyHigh ツール（読み取り専用）
			if safety == tools.SafetyHigh {
				// ループ検知
				if a.shouldAbortToolLoop(tc, lastToolCall, &sameCallCount) {
					return nil, fmt.Errorf("tool loop detected during investigation")
				}
				lastToolCall = tc

				if a.Stats != nil {
					a.Stats.AddToolExecution(tc.Tool)
				}
				result, _ := tools.Execute(tc)
				allResults = append(allResults, fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, result))
				continue
			}

			// SafetyMedium/Low ツール（変更系）→ 計画生成を要求
			cyan.Printf("\n⚡ Implementation tool detected: %s\n", tc.Tool)
			cyan.Println("   Requesting implementation plan...")

			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: promptplan.BuildPlanRequestMessage(tc.Tool),
			})
			break // 変更系ツールが見つかったらループを抜けて次のイテレーションへ
		}

		// 結果を履歴に追加
		if len(allResults) > 0 {
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: strings.Join(allResults, "\n\n"),
			})
		}
	}

	yellow.Printf("⚠️  調査フェーズが%d回のツール実行に達しました。続けて指示してください。\n", maxIterations)
	return nil, nil
}

// getCreatePlanTool は ToolRegistry から create_plan ツールを取得
func (a *Agent) getCreatePlanTool() *planning.CreatePlanTool {
	tool := tools.DefaultRegistry.GetTool("create_plan")
	if tool == nil {
		return nil
	}
	if createPlanTool, ok := tool.(*planning.CreatePlanTool); ok {
		return createPlanTool
	}
	return nil
}
