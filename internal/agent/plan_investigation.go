package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// runInvestigationPhase は調査フェーズを実行
// SafetyHighツールのみを実行し、計画JSONを返す
func (a *Agent) runInvestigationPhase(ctx context.Context) (string, error) {
	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int
	var lastToolCallsHash string // 並列実行時のループ検知用
	var sameSetCount int         // 並列実行時の同一セットカウント

	for i := 0; i < maxIterations; i++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return "", fmt.Errorf("investigation failed: %w", err)
		}

		a.History = append(a.History, api.Message{Role: "assistant", Content: response})
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		// 計画JSONを検出
		if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
			return planJSON, nil
		}

		// デバッグモード: レスポンスの診断情報を出力
		if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] response length: %d\n", len(response))
			// ツールパターンの存在チェック
			if idx := strings.Index(response, `{"tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found {\"tool\" at index %d\n", idx)
			}
			if idx := strings.Index(response, `{ "tool"`); idx != -1 {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] found { \"tool\" at index %d\n", idx)
			}
			// コードブロックの数をチェック
			codeBlockOpens := strings.Count(response, "```")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] code block markers (```): %d (odd = unclosed)\n", codeBlockOpens)
			// 閉じていないJSONの可能性をチェック
			openBraces := strings.Count(response, "{")
			closeBraces := strings.Count(response, "}")
			fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] braces: { = %d, } = %d (mismatch = incomplete JSON)\n", openBraces, closeBraces)
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// デバッグ: なぜ0件なのか追加診断
			if os.Getenv("XELYON_DEBUG_PARSE") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] ParseToolCalls returned 0 tools\n")
				// ツールパターンがあるのに0件の場合、原因を調査
				if strings.Contains(response, `{"tool"`) || strings.Contains(response, `{ "tool"`) {
					fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] WARNING: tool pattern exists but not parsed!\n")
					// 最後の200文字を表示
					if len(response) > 200 {
						fmt.Fprintf(os.Stderr, "[DEBUG runInvestigationPhase] tail: ...%s\n", response[len(response)-200:])
					}
				}
			}
			// ツール呼び出しも計画もない場合は終了
			// AIが調査を終えて単に説明しているか、質問している場合
			fmt.Println(response)
			return "", nil
		}

		// ツールを分類
		var safetyHighTools []*tools.ToolCall
		var otherTools []*tools.ToolCall

		for _, toolCall := range toolCalls {
			safety := tools.GetToolSafety(toolCall.Tool)
			if safety == tools.SafetyHigh {
				safetyHighTools = append(safetyHighTools, toolCall)
			} else {
				otherTools = append(otherTools, toolCall)
			}
		}

		// SafetyMedium/Low がある場合は計画生成を要求（従来通り）
		if len(otherTools) > 0 {
			tc := otherTools[0]
			cyan.Printf("\n⚡ Implementation tool detected: %s\n", tc.Tool)
			cyan.Println("   Requesting implementation plan...")

			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: promptplan.BuildPlanRequestMessage(tc.Tool),
			})
			continue
		}

		// SafetyHigh ツールがない場合（ツールなし）= 調査完了
		if len(safetyHighTools) == 0 {
			fmt.Println(response)
			return "", nil
		}

		// SafetyHigh ツールを実行
		var allResults []string
		if len(safetyHighTools) > 1 {
			// 並列実行時のループ検知
			currentHash := plan.HashToolCalls(safetyHighTools)
			if currentHash == lastToolCallsHash {
				sameSetCount++
				if sameSetCount >= config.LoopDetectionThreshold {
					return "", fmt.Errorf("tool loop detected: same tool set repeated %d times", sameSetCount)
				}
			} else {
				sameSetCount = 0
			}
			lastToolCallsHash = currentHash

			// 並列実行
			cyan.Printf("⚡ Executing %d investigation tools in parallel...\n", len(safetyHighTools))
			allResults = a.executeInvestigationToolsParallel(ctx, safetyHighTools)
		} else {
			// 単一ツールは直列実行
			tc := safetyHighTools[0]

			// ループ検知
			if a.shouldAbortToolLoop(tc, lastToolCall, &sameCallCount) {
				return "", fmt.Errorf("tool loop detected during investigation")
			}
			lastToolCall = tc

			if a.Stats != nil {
				a.Stats.AddToolExecution(tc.Tool)
			}
			result, _ := tools.Execute(tc)
			allResults = []string{fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, result)}
		}

		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: strings.Join(allResults, "\n\n"),
		})
	}

	yellow.Printf("⚠️  調査フェーズが%d回のツール実行に達しました。続けて指示してください。\n", maxIterations)
	return "", nil
}

// executeInvestigationToolsParallel は SafetyHigh ツールを並列実行
// 結果は toolCall 順で返す（順序保持）
func (a *Agent) executeInvestigationToolsParallel(ctx context.Context, toolCalls []*tools.ToolCall) []string {
	cfg := config.GetGlobalConfig()
	maxWorkers := cfg.PlanMode.MaxParallelSteps
	if maxWorkers <= 0 {
		maxWorkers = config.DefaultParallelWorkers
	}

	results := make([]string, len(toolCalls))
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i, tc := range toolCalls {
		wg.Add(1)
		go func(idx int, toolCall *tools.ToolCall) {
			defer wg.Done()

			// セマフォ取得
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[idx] = fmt.Sprintf("[%s]\nError: context cancelled", toolCall.Tool)
				return
			}

			// Stats 更新（スレッドセーフ）
			a.incrementToolExecution(toolCall.Tool)

			// ツール実行
			result, _ := tools.Execute(toolCall)
			results[idx] = fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result)
		}(i, tc)
	}

	wg.Wait()
	return results
}
