package tuiagent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	agentpkg "github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/providerpicker"
	"github.com/susugadx/xelyon-cli/internal/review"
	agentskills "github.com/susugadx/xelyon-cli/internal/skills"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

var red = color.New(color.FgRed)

// TUIAdapter は Agent を tui.AgentInterface に適合させるアダプタ
type TUIAdapter struct {
	agent              *agentpkg.Agent
	sendMsg            func(tui.AppendMessageMsg)
	sendToolResult     func(tui.AppendToolResultMsg)
	sendReviewProgress func(tui.ReviewProgressMsg)
	captureWriter      *tuiCaptureWriter
	processing         atomic.Bool
	flushMu            sync.RWMutex
	tuiEventFlush      func()
}

// NewTUIAdapter は TUIAdapter を作成する。
func NewTUIAdapter(agent *agentpkg.Agent, sendMsg func(tui.AppendMessageMsg)) *TUIAdapter {
	return &TUIAdapter{
		agent:   agent,
		sendMsg: sendMsg,
	}
}

// SetOutputCapture は agent の出力先を TUI キャプチャ用 Writer に差し替える。
func (a *TUIAdapter) SetOutputCapture() {
	capture := newTUICaptureWriter(func(text string) {
		if a.sendMsg != nil {
			a.sendMsg(tui.AppendMessageMsg{
				Message: tui.ChatMessage{
					Role:    tui.ChatRoleAssistantChunk,
					Content: text,
				},
			})
		}
	})
	a.captureWriter = capture
	runtime := a.agent.RuntimeUI()
	runtime.SetOutput(capture)
	runtime.SetErrorOutput(capture)
}

// Chat はユーザー入力をAIに送信する。goroutine で呼ぶこと。
func (a *TUIAdapter) Chat(input string) error {
	a.processing.Store(true)
	defer a.processing.Store(false)
	defer a.finishTUITurnOutput()

	// 画像入力チェック
	if strings.Contains(input, "image:") {
		textPart, image := a.agent.ParseImageInput(input)
		if image != nil {
			return a.chatWithImage(textPart, image)
		}
	}

	return a.agent.Chat(input)
}

// ChatWithImagePath は画像パス付き入力を AI に送信する。goroutine で呼ぶこと。
func (a *TUIAdapter) ChatWithImagePath(input string, imagePath string) error {
	a.processing.Store(true)
	defer a.processing.Store(false)
	defer a.finishTUITurnOutput()

	image, err := api.LoadImage(imagePath)
	if err != nil {
		red.Fprintf(a.agent.Output(), "Failed to load image: %v\n", err)
		return tui.WrapAgentTurnError(tui.AgentErrorValidation, fmt.Errorf("failed to load image: %w", err))
	}

	return a.chatWithImage(input, image)
}

// ChatWithImage は読み込み済み画像をAIに送信する。goroutine または tea.Cmd で呼ぶこと。
func (a *TUIAdapter) ChatWithImage(input string, image *api.ImageData) error {
	a.processing.Store(true)
	defer a.processing.Store(false)
	defer a.finishTUITurnOutput()

	return a.chatWithImage(input, image)
}

func (a *TUIAdapter) chatWithImage(input string, image *api.ImageData) error {
	return a.agent.ChatWithImage(input, image)
}

func (a *TUIAdapter) finishTUITurnOutput() {
	a.flushCapture()
	a.flushTUIEvents()
}

func (a *TUIAdapter) flushCapture() {
	if a.captureWriter != nil {
		a.captureWriter.Flush()
	}
}

func (a *TUIAdapter) setTUIEventFlush(fn func()) {
	a.flushMu.Lock()
	defer a.flushMu.Unlock()
	a.tuiEventFlush = fn
}

func (a *TUIAdapter) flushTUIEvents() {
	a.flushMu.RLock()
	fn := a.tuiEventFlush
	a.flushMu.RUnlock()
	if fn != nil {
		fn()
	}
}

// HandleCommand は特殊コマンドを処理する。処理した場合 true を返す。
func (a *TUIAdapter) HandleCommand(cmd string) bool {
	return a.agent.HandleCommandForSurface(cmd, commandcatalog.CommandSurfaceTUI)
}

// ResumeSessionCandidates は TUI picker 用の再開候補を返す。
func (a *TUIAdapter) ResumeSessionCandidates(opts tui.SessionResumeOptions) ([]tui.SessionCandidate, error) {
	sessions, err := a.agent.ResumeSessionCandidates(history.ResumeListOptions{All: opts.All})
	if err != nil {
		return nil, err
	}
	candidates := make([]tui.SessionCandidate, 0, len(sessions))
	for _, session := range sessions {
		candidates = append(candidates, sessionCandidateFromMetadata(session))
	}
	return candidates, nil
}

// ResumeLastSession は resume scope 内の最新 session を復元する。
func (a *TUIAdapter) ResumeLastSession(opts tui.SessionResumeOptions) (tui.SessionCandidate, error) {
	session, err := a.agent.ResumeLastSession(history.ResumeListOptions{All: opts.All})
	if err != nil {
		a.flushCapture()
		return tui.SessionCandidate{}, err
	}
	a.flushCapture()
	return tui.SessionCandidate{
		ID:           session.ID,
		ProviderName: session.ProviderName,
		Model:        session.Model,
		WorkingDir:   session.WorkingDir,
		LastModified: session.LastModified,
		MessageCount: len(session.ToAPIMessages()),
	}, nil
}

// ResumeSession は指定 session を現在の interactive runtime に復元する。
func (a *TUIAdapter) ResumeSession(id string) error {
	if _, err := a.agent.ResumeSession(id); err != nil {
		a.flushCapture()
		return err
	}
	a.flushCapture()
	return nil
}

func sessionCandidateFromMetadata(session history.SessionMetadata) tui.SessionCandidate {
	return tui.SessionCandidate{
		ID:           session.ID,
		Preview:      session.Preview,
		ProviderName: session.ProviderName,
		Model:        session.Model,
		WorkingDir:   session.WorkingDir,
		LastModified: session.LastModified,
		MessageCount: session.MessageCount,
	}
}

// StartNewSession は新しい interactive session を開始する。
func (a *TUIAdapter) StartNewSession() (string, error) {
	session, err := a.agent.StartNewSession()
	if err != nil {
		a.flushCapture()
		return "", err
	}
	a.flushCapture()
	return session.ID, nil
}

// SkillCatalog は TUI の /skills 補完に現在の skill catalog を提供する。
func (a *TUIAdapter) SkillCatalog() agentskills.SkillCatalog {
	if a == nil || a.agent == nil {
		return agentskills.SkillCatalog{}
	}
	return a.agent.SkillCatalog()
}

// GetStatusLine はステータスバーに表示する文字列を返す。
func (a *TUIAdapter) GetStatusLine() string {
	return a.agent.FormatStatusLine()
}

// StatusSnapshot は TUI ステータスバー用の構造化状態を返す。
func (a *TUIAdapter) StatusSnapshot() tui.StatusSnapshot {
	if a == nil || a.agent == nil {
		return tui.StatusSnapshot{}
	}
	snapshot := a.agent.InteractiveStatusSnapshot()
	return tui.StatusSnapshot{
		Provider:   snapshot.Provider,
		Model:      snapshot.Model,
		Mode:       snapshot.Mode,
		Tokens:     snapshot.Tokens,
		Cost:       snapshot.Cost,
		LegacyLine: snapshot.LegacyLine,
	}
}

// Cancel は現在のAPI呼び出しをキャンセルする。
func (a *TUIAdapter) Cancel() {
	a.agent.CancelActiveRequest("user cancelled")
}

// Cleanup は終了処理を行う。
func (a *TUIAdapter) Cleanup() {
	a.agent.Cleanup()
}

// IsProcessing はAI処理中かどうかを返す。
func (a *TUIAdapter) IsProcessing() bool {
	return a.processing.Load()
}

// RunReview は /review 実行中だけ TUI の処理中状態を立て、Agent の runner へ委譲する。
func (a *TUIAdapter) RunReview(ctx context.Context, req review.ReviewRequest) (tui.ReviewRunResult, error) {
	a.processing.Store(true)
	defer a.processing.Store(false)

	report, summary, err := a.agent.RunReviewWithProgress(ctx, req, a.reviewProgressSink(ctx))
	a.finishTUITurnOutput()
	return tui.ReviewRunResult{
		Report: report,
		Usage: tui.ReviewRunUsageSummary{
			Tokens: summary.Tokens,
			Cost:   summary.Cost,
		},
	}, err
}

// CopyText は指定テキストをクリップボードにコピーする。
func (a *TUIAdapter) CopyText(text string) error {
	return agentpkg.CopyText(text)
}

// LoadConfigForEdit は設定ファイルを読み込み、編集用のクローンを返す。
func (a *TUIAdapter) LoadConfigForEdit() (*config.Config, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	return config.CloneConfig(cfg), nil
}

// SaveAndSyncConfig は設定をファイルに保存し、runtime に反映する。
func (a *TUIAdapter) SaveAndSyncConfig(cfg *config.Config) error {
	return a.agent.SaveAndSyncConfig(cfg)
}

// LoadProjectForEdit は xelyon.yaml を読み込み、編集用のクローンを返す。
func (a *TUIAdapter) LoadProjectForEdit() (*config.ProjectConfig, error) {
	pc, err := config.LoadProjectConfigWithError()
	if err != nil {
		return nil, err
	}
	return config.CloneProjectConfig(pc), nil
}

// SaveProjectConfig は xelyon.yaml を保存する。
func (a *TUIAdapter) SaveProjectConfig(pc *config.ProjectConfig) error {
	return a.agent.SaveAndSyncProjectConfig(pc)
}

// CreateProjectConfigTemplate は xelyon.yaml template を作成し、編集用に読み込む。
func (a *TUIAdapter) CreateProjectConfigTemplate() (*config.ProjectConfig, error) {
	if err := config.CreateProjectConfigTemplate("", false); err != nil {
		return nil, err
	}
	pc, err := config.LoadProjectConfigWithError()
	if err != nil {
		return nil, err
	}
	if pc == nil {
		return nil, fmt.Errorf("created xelyon.yaml but failed to load it")
	}
	return config.CloneProjectConfig(pc), nil
}

// GetProviderName は現在のプロバイダー名を返す。
func (a *TUIAdapter) GetProviderName() string {
	return a.agent.ProviderName
}

// GetProviderConfigKey は現在セッションが代表する provider_models key を返す。
func (a *TUIAdapter) GetProviderConfigKey() string {
	return a.agent.GetProviderConfigKey()
}

// ProviderCandidates は provider picker の候補を返す。
func (a *TUIAdapter) ProviderCandidates() []providerpicker.ProviderCandidate {
	return a.agent.ProviderCandidates()
}

// ModelCandidates は provider に対応する model/deployment picker 候補を返す。
func (a *TUIAdapter) ModelCandidates(provider string) []providerpicker.ModelCandidate {
	return a.agent.ModelCandidates(provider)
}

// AzureCatalogModelCandidates は Azure deployment に紐づける catalog_model 候補を返す。
func (a *TUIAdapter) AzureCatalogModelCandidates(deployment string) []providerpicker.ModelCandidate {
	return a.agent.AzureCatalogModelCandidates(deployment)
}

// SwitchProviderModel は provider と model/deployment を切り替える。
func (a *TUIAdapter) SwitchProviderModel(provider string, model string) error {
	if err := a.agent.SwitchProviderModelWithOutput(provider, model); err != nil {
		a.flushCapture()
		return err
	}
	a.flushCapture()
	return nil
}

// SwitchModelForCurrentProvider は current provider の model/deployment を切り替える。
func (a *TUIAdapter) SwitchModelForCurrentProvider(model string) error {
	if err := a.agent.SwitchModelForCurrentProviderWithOutput(model); err != nil {
		a.flushCapture()
		return err
	}
	a.flushCapture()
	return nil
}

// ConfigureAndSwitchAzureDeployment は Azure deployment setup を保存して provider を切り替える。
func (a *TUIAdapter) ConfigureAndSwitchAzureDeployment(deployment string, catalogModel string) error {
	if _, err := a.agent.ConfigureAndSwitchAzureDeployment(deployment, catalogModel); err != nil {
		a.flushCapture()
		return err
	}
	a.flushCapture()
	return nil
}

// CopyLastOutput は直近のAI出力をクリップボードにコピーする。
// historyMu でロックし、chat goroutine との data race を防ぐ。
func (a *TUIAdapter) CopyLastOutput() (string, error) {
	return a.agent.CopyLastOutput()
}

// tuiCaptureWriter は agent の出力をキャプチャし TUI に送信する io.Writer
type tuiCaptureWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	sendFn func(string)
}

func newTUICaptureWriter(sendFn func(string)) *tuiCaptureWriter {
	return &tuiCaptureWriter{sendFn: sendFn}
}

// Write は書き込まれた内容をバッファに追加し、改行区切りでバッチフラッシュする。
// \r を含む書き込みはスピナー等の行上書きアニメーションなのでドロップする。
// 複数行が一度に届いた場合は1回の sendFn 呼び出しにまとめて Update+View サイクルを削減する。
func (w *tuiCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// \r を含む書き込みはスピナー → ドロップ
	if bytes.Contains(p, []byte("\r")) {
		return len(p), nil
	}

	n, err := w.buf.Write(p)

	// 改行を含む場合、全行をまとめて1回の sendFn で送る
	if bytes.Contains(p, []byte("\n")) {
		var batch strings.Builder
		for {
			line, readErr := w.buf.ReadString('\n')
			if readErr != nil {
				if line != "" {
					w.buf.WriteString(line)
				}
				break
			}
			if batch.Len() > 0 {
				batch.WriteByte('\n')
			}
			batch.WriteString(strings.TrimRight(line, "\n"))
		}
		if batch.Len() > 0 && w.sendFn != nil {
			w.sendFn(batch.String())
		}
	}

	return n, err
}

// Flush はバッファに残っている内容をフラッシュする。
func (w *tuiCaptureWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf.Len() > 0 {
		if w.sendFn != nil {
			w.sendFn(w.buf.String())
		}
		w.buf.Reset()
	}
}
