package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// retryState は retry ループの空回り検出を行う内部状態。
//
// stalled（空回り）は「停止の絶対条件」ではなく「方針変更ヒント」として扱う。
// errorFingerprint は雑な近似であり、false positive / false negative の両方があり得る。
// そのため stalled 検出時もまず retry 指示を強化し、即座に hard stop はしない。
type retryState struct {
	count       int    // 累積リトライ回数（表示用）
	lastErrorFP string // 直前のエラー fingerprint（空回り近似用）
	sameCount   int    // 同一 fingerprint の連続回数
	stalledRuns int    // stalled 検出後に続行した回数（soft→hard エスカレーション用）
}

// stalledRetryThreshold は同一 fingerprint が連続何回で stalled hint とみなすか。
const stalledRetryThreshold = 3

// stalledHardThreshold は stalled 検出後にさらに何回失敗したら hard escalation するか。
// plan mode では selector UI、normal mode では AI に委譲。
const stalledHardThreshold = 2

// recordFailure はエラーを記録し、空回りの深刻度を返す。
//   - stalledNone: 新しいエラーまたは閾値未到達 → 通常リトライ
//   - stalledSoft: 同一 fingerprint が閾値に達した → retry 指示を強化して続行
//   - stalledHard: soft 後もさらに同一エラーが続いた → 外部介入（selector / AI 委譲）
func (s *retryState) recordFailure(errorOutput string) stalledLevel {
	fp := errorFingerprint(errorOutput)
	if fp == s.lastErrorFP {
		s.sameCount++
	} else {
		s.lastErrorFP = fp
		s.sameCount = 1
		s.stalledRuns = 0
	}
	s.count++

	if s.sameCount < stalledRetryThreshold {
		return stalledNone
	}
	s.stalledRuns++
	if s.stalledRuns <= stalledHardThreshold {
		return stalledSoft
	}
	return stalledHard
}

// reset は成功時やユーザー手動リトライ時に状態をリセットする。
func (s *retryState) reset() {
	s.lastErrorFP = ""
	s.sameCount = 0
	s.stalledRuns = 0
	s.count = 0
}

// stalledLevel は空回り検出の深刻度。
type stalledLevel int

const (
	stalledNone stalledLevel = iota // 空回りなし → 通常リトライ
	stalledSoft                     // 空回りヒント → retry 指示を強化して続行
	stalledHard                     // 空回り確定 → 外部介入（selector / AI 委譲）
)

// errorFingerprint はエラー出力から空回り検出用の雑な近似 fingerprint を返す。
//
// 厳密なエラー同一性判定ではなく、「同じ根本原因のエラーが繰り返されている可能性」の
// ヒントとして使う。軽い正規化（trim, ANSI 除去, 空白圧縮）後の先頭 200 文字で比較する。
// false positive（別エラーを同一視）/ false negative（同一エラーを別扱い）の両方があり得るため、
// この fingerprint だけで hard stop の判断はしない。
func errorFingerprint(s string) string {
	s = strings.TrimSpace(s)
	s = normalizeErrorText(s)
	return truncateRunes(s, 200)
}

// truncateRunes は s を最大 n ルーンで切り詰める（rune 境界で安全に切断）。
func truncateRunes(s string, n int) string {
	count := 0
	for i := range s {
		if count >= n {
			return s[:i]
		}
		count++
	}
	return s
}

// normalizeErrorText は ANSI エスケープ除去 + 連続空白圧縮を行う。
func normalizeErrorText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for i := 0; i < len(s); i++ {
		// ANSI CSI シーケンス: \x1b[ ... 終端文字 まで読み飛ばす
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3F {
				j++
			}
			if j < len(s) {
				j++ // 終端文字をスキップ
			}
			i = j - 1
			continue
		}
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		} else {
			b.WriteByte(c)
			prevSpace = false
		}
	}
	return b.String()
}

// runImplementationPhase は実装フェーズを実行（順次実行）
func (a *Agent) runImplementationPhase(ctx context.Context, p *plan.Plan) error {
	for {
		// 次のステップを取得
		nextID := p.GetNextStep()
		if nextID == -1 {
			break // 全て完了
		}
		step := p.GetStep(nextID)
		if step == nil {
			break
		}

		_, _ = fmt.Fprintf(a.output(), "\n%s\n", ui.FormatStepProgress(nextID, len(p.Steps), step.Description, "running"))
		if err := a.executeStepV2(ctx, p, step, nextID-1, &retryState{}); err != nil {
			return err
		}
		p.UpdateStatus(nextID, "completed", "")

		// ステップ完了フックを実行
		if hooks := a.cfg().Hooks; len(hooks.OnStepComplete) > 0 {
			if !a.runStepCompleteHooksWithRetry(ctx, nextID, step.Description, "completed") {
				yellow.Fprintf(a.output(), "⚠️  Step %d hooks failed but proceeding to next step\n", nextID)
			}
		}

		// 全て完了したか確認
		if p.IsCompleted() {
			break
		}
	}

	green.Fprintf(a.output(), "\n✓ All %d steps completed!\n", len(p.Steps))

	// セッションを保存
	if a.storage != nil && a.session != nil {
		a.syncResponseIDToSession()
		if err := a.storage.Save(a.session); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to save session: %v\n", err)
		}
	}

	// git diff empty check は executeStepV2 の Level 1/Level 2 ガードでカバー済み
	// runImplementationPhase レベルではチェックしない（調査系プランなど変更なしが正常なケースがある）

	return nil
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

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan() (approved bool, feedback string) {
	result := tools.ConfirmInteractiveWithIO(a.ui().PromptIO(), "Approve this plan?")

	switch result.Action {
	case "yes":
		return true, ""
	case "no":
		return false, ""
	case "comment":
		return false, strings.TrimSpace(result.Comment)
	default:
		return false, ""
	}
}

// planModeRetryInstruction は Plan Mode の retry プロンプトを段階的に返す。
func planModeRetryInstruction(attempt int) string {
	const constraint = `
Reuse information already obtained from the failed command/output.
Do not restart broad investigation unless the current evidence is insufficient.
Prefer the smallest clarifying step before expanding the plan.`

	switch {
	case attempt <= 1:
		return `Do not react blindly.
First, identify the concrete cause of failure in 1-2 sentences using the existing output.
Point to the exact file/function/command/step involved.
Then choose the smallest next action that can resolve or verify the issue immediately.

Do not broaden investigation yet unless the current failure output is insufficient.

Do NOT skip this step. The issue must be resolved before proceeding.` + constraint
	case attempt == 2:
		return `The previous attempt did not work.

Do not repeat the same step pattern.
Briefly explain why the previous attempt failed.
If the cause is still unclear, create the smallest possible reproduction or targeted verification via bash.
If the cause is already clear, skip extra test creation and apply the next evidence-based step directly.

Then verify again.

Do NOT skip this step. The issue must be resolved before proceeding.` + constraint
	default:
		return `Multiple retries have failed.

Your current plan is not working.
Explain which assumption was wrong.
Change strategy fundamentally and choose the smallest different step that can validate the new hypothesis quickly.

Do NOT skip this step. The issue must be resolved before proceeding.` + constraint
	}
}
