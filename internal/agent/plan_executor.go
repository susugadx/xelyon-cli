package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// runImplementationPhase は実装フェーズを実行（順次実行）
func (a *Agent) runImplementationPhase(ctx context.Context, p *plan.Plan) error {
	return newImplementationPhaseRunner(a, ctx, p).Run()
}

// executeStepV2 は単一ステップを実行（失敗検知・空回り検出による自動リトライ対応）
func (a *Agent) executeStepV2(ctx context.Context, p *plan.Plan, step *plan.PlanStep, idx int, rs *retryState) error {
	return newTurnRunner(a, ctx).ExecuteStep(p, step, idx, rs)
}

// isAIQuestionWithToolParser は AI が質問しているかを判定する。
// tool call parser は明示注入し、Agent には依存しない。
func isAIQuestionWithToolParser(response string, parseToolCalls func(string) []*tools.ToolCall) bool {
	// ツール呼び出しがある場合は質問とみなさない
	if parseToolCalls != nil && len(parseToolCalls(response)) > 0 {
		return false
	}

	questionPatterns := []string{
		"続行しますか", "よろしいですか", "確認してください", "どうしますか",
		"選択してください", "指定してください", "教えてください",
		"Should I", "Do you want", "Would you like", "Shall I",
		"Can you confirm", "Please confirm", "proceed?", "continue?",
	}

	lowered := strings.ToLower(response)
	for _, pattern := range questionPatterns {
		if strings.Contains(lowered, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func isAIQuestion(response string) bool {
	return isAIQuestionWithToolParser(response, tools.ParseToolCalls)
}

// getGitDiffHash は git diff HEAD + untracked files の出力を SHA256 ハッシュ化して返す。
// ファイル名だけでなく内容の変化も検知する（同じファイルへの追加変更を検出）。
// untracked ファイルはファイル名リストだけでなく内容もハッシュに含めることで、
// git add 前のファイルへの str_replace 等の変更を正しく検知する。
// git が使えない場合は空文字を返す（Level 2 スキップ用）。
func getGitDiffHash() string {
	// 1. tracked の差分（unstaged + staged）
	out, err := exec.Command("git", "diff", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write(out)

	// 2. untracked ファイルの名前と内容を両方ハッシュに含める
	untrackedOut, _ := exec.Command("git", "ls-files", "--others", "--exclude-standard").CombinedOutput()
	h.Write(untrackedOut) // ファイル名リスト（新規追加検知）
	for _, f := range strings.Split(strings.TrimSpace(string(untrackedOut)), "\n") {
		if f == "" {
			continue
		}
		content, err := os.ReadFile(f)
		if err == nil {
			h.Write(content) // 内容（編集検知）
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
