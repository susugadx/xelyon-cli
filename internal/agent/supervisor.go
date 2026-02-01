package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/planning"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// Color variables are defined in agent.go

// Supervisor は Plan Mode の実行を監督
type Supervisor struct {
	// プロバイダー
	supervisorProvider api.Provider // Supervisor 用（計画生成、エスカレーション判断）
	lightProvider      api.Provider // Worker 用軽量モデル
	heavyProvider      api.Provider // エスカレーション用高性能モデル（nil 可）

	// モデル名
	supervisorModel string
	lightModel      string
	heavyModel      string

	// 共有状態
	sharedContext *SharedContext
	workerPool    *WorkerPool

	// 設定
	maxWorkers   int
	maxRetry     int
	stepTimeout  int
	confirmLevel string

	// Agent 参照（履歴・ツール実行用）
	agent *Agent

	// リトライ回数追跡
	retryCount map[int]int
}

// NewSupervisor は新しい Supervisor を作成
func NewSupervisor(agent *Agent, cfg *config.PlanModeConfig) (*Supervisor, error) {
	s := &Supervisor{
		agent:           agent,
		maxWorkers:      cfg.MaxWorkers,
		maxRetry:        cfg.MaxRetry,
		stepTimeout:     cfg.StepTimeout,
		confirmLevel:    cfg.ConfirmLevel,
		sharedContext:   NewSharedContext(),
		supervisorModel: cfg.SupervisorModel,
		lightModel:      cfg.LightModel,
		heavyModel:      cfg.HeavyModel,
		retryCount:      make(map[int]int),
	}

	// プロバイダー初期化
	if err := s.initProviders(); err != nil {
		return nil, err
	}

	// WorkerPool 作成
	s.workerPool = NewWorkerPool(
		s.maxWorkers,
		s.lightProvider,
		s.lightModel,
		s.sharedContext,
		agent,
		s.stepTimeout,
	)

	return s, nil
}

// initProviders はモデル設定に基づいてプロバイダーを初期化
func (s *Supervisor) initProviders() error {
	// supervisor_model（空ならメインプロバイダーを使用）
	if s.supervisorModel == "" {
		s.supervisorProvider = s.agent.CurrentProvider
		s.supervisorModel = s.agent.CurrentModel
	} else {
		// モデル名からプロバイダーを推論して作成
		provider, err := s.createProviderForModel(s.supervisorModel)
		if err != nil {
			return fmt.Errorf("failed to create supervisor provider: %w", err)
		}
		s.supervisorProvider = provider
	}

	// light_model（Worker 用）
	if s.lightModel == "" {
		s.lightProvider = s.agent.CurrentProvider
		s.lightModel = s.agent.CurrentModel
	} else {
		provider, err := s.createProviderForModel(s.lightModel)
		if err != nil {
			return fmt.Errorf("failed to create light provider: %w", err)
		}
		s.lightProvider = provider
	}

	// heavy_model（エスカレーション用、空なら無効）
	if s.heavyModel != "" {
		provider, err := s.createProviderForModel(s.heavyModel)
		if err != nil {
			return fmt.Errorf("failed to create heavy provider: %w", err)
		}
		s.heavyProvider = provider
	}

	return nil
}

// createProviderForModel はモデル名からプロバイダーを作成
func (s *Supervisor) createProviderForModel(model string) (api.Provider, error) {
	// モデル名からプロバイダー名を推論
	providerName := inferProviderFromModel(model)
	provider, err := api.NewProvider(providerName)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// inferProviderFromModel はモデル名からプロバイダー名を推論
func inferProviderFromModel(model string) string {
	switch {
	case strings.HasPrefix(model, "gpt-"), strings.HasPrefix(model, "o1"), strings.HasPrefix(model, "o3"):
		return "openai"
	case strings.HasPrefix(model, "claude"):
		return "claude"
	case strings.HasPrefix(model, "gemini"):
		return "gemini"
	case strings.HasPrefix(model, "deepseek"):
		return "deepseek"
	case strings.Contains(model, "llama"), strings.Contains(model, "qwen"):
		return "ollama"
	default:
		// メインプロバイダーと同じにする（安全）
		return config.GetGlobalConfig().DefaultProvider
	}
}

// Run は Plan Mode を実行（並列モード）
func (s *Supervisor) Run(ctx context.Context, userRequest string) error {
	// 1. 調査フェーズ
	cyan.Println("\n🔍 Investigation Phase (parallel)")
	p, err := s.runInvestigation(ctx, userRequest)
	if err != nil {
		return err
	}

	if p == nil {
		green.Println("\n✓ Investigation complete. No implementation needed.")
		return nil
	}

	if len(p.Steps) == 0 {
		green.Println("\n✓ Investigation complete. No implementation steps needed.")
		return nil
	}

	// 計画を表示
	planDisplay := ui.NewPlanDisplay("Implementation Plan (Parallel)").
		SetSummary(p.Summary)

	for _, step := range p.Steps {
		planDisplay.AddStep(step.ID, step.Description, step.Tools, step.TargetFiles)
	}

	fmt.Println()
	fmt.Print(planDisplay.Render())

	// 承認を取得（既存の UI 使用）
	approved, feedback := s.agent.confirmPlan()
	if !approved {
		if feedback != "" {
			yellow.Printf("Plan rejected with feedback: %s\n", feedback)
			return s.Run(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
		}
		red.Println("Plan execution cancelled.")
		return nil
	}

	green.Println("✓ Plan approved. Starting parallel execution...")

	// 3. 実装フェーズ
	return s.runExecution(ctx, p)
}

// runInvestigation は調査フェーズを実行
func (s *Supervisor) runInvestigation(ctx context.Context, userRequest string) (*plan.Plan, error) {
	// Supervisor が調査クエリを生成
	queries, err := s.generateInvestigationQueries(ctx, userRequest)
	if err != nil {
		// クエリ生成に失敗した場合は単一クエリで実行
		queries = []string{userRequest}
	}

	if len(queries) == 0 {
		queries = []string{userRequest}
	}

	// Worker 並列実行（調査）
	s.workerPool.Start(ctx)
	defer s.workerPool.Stop()

	// 調査を実行
	results := s.workerPool.ExecuteInvestigationsWithUI(ctx, queries)

	// 結果を集約
	aggregated := s.aggregateInvestigationResults(results)

	// Supervisor が計画を生成
	return s.generatePlan(ctx, userRequest, aggregated)
}

// generateInvestigationQueries は Supervisor が調査クエリを生成
func (s *Supervisor) generateInvestigationQueries(ctx context.Context, userRequest string) ([]string, error) {
	prompt := promptplan.BuildInvestigationQueryPrompt(userRequest)

	response, err := s.supervisorProvider.ChatWithTools(
		ctx,
		s.agent.SystemPrompt,
		[]api.Message{{Role: "user", Content: prompt}},
		s.supervisorModel,
	)
	if err != nil {
		return nil, err
	}

	// JSON からクエリを抽出
	return parseInvestigationQueries(response)
}

// parseInvestigationQueries は応答からクエリを抽出
func parseInvestigationQueries(response string) ([]string, error) {
	// JSON ブロックを探す
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start == -1 || end == -1 || start > end {
		return nil, errors.New("no JSON found in response")
	}

	jsonStr := response[start : end+1]

	var result struct {
		Queries []string `json:"queries"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, err
	}

	return result.Queries, nil
}

// aggregateInvestigationResults は調査結果を集約
func (s *Supervisor) aggregateInvestigationResults(results []WorkerResult) string {
	var builder strings.Builder

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("=== Investigation %d ===\n", i+1))
		if result.Success {
			builder.WriteString(result.Output)
		} else {
			builder.WriteString(fmt.Sprintf("Failed: %v", result.Error))
		}
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// generatePlan は Supervisor が計画を生成（create_plan ツールを使用）
func (s *Supervisor) generatePlan(ctx context.Context, userRequest, investigationResults string) (*plan.Plan, error) {
	prompt := promptplan.BuildPlanGenerationPrompt(userRequest, investigationResults)

	// create_plan ツールを取得
	createPlanTool := s.getCreatePlanTool()
	if createPlanTool != nil {
		createPlanTool.ClearLastPlan()
	}

	// 計画生成をループ（create_plan ツールが呼ばれるまで）
	history := []api.Message{{Role: "user", Content: prompt}}
	maxIterations := 5

	for i := 0; i < maxIterations; i++ {
		response, err := s.supervisorProvider.ChatWithTools(
			ctx,
			s.agent.SystemPrompt,
			history,
			s.supervisorModel,
		)
		if err != nil {
			return nil, err
		}

		history = append(history, api.Message{Role: "assistant", Content: response})

		// ツール呼び出しをパース
		toolCalls := tools.ParseToolCalls(response)

		// ツールがない場合は終了（計画なし）
		if len(toolCalls) == 0 {
			return nil, nil
		}

		// ツールを実行
		var allResults []string
		for _, tc := range toolCalls {
			if tc.Tool == "create_plan" {
				result, _ := tools.Execute(tc)
				allResults = append(allResults, fmt.Sprintf("[%s]\n%s", tc.Tool, result))

				// create_plan ツールから Plan を取得
				if createPlanTool != nil {
					if p := createPlanTool.LastPlan(); p != nil {
						return p, nil
					}
				}
			}
		}

		// 結果を履歴に追加
		if len(allResults) > 0 {
			history = append(history, api.Message{
				Role:    "user",
				Content: strings.Join(allResults, "\n\n"),
			})
		}
	}

	return nil, nil
}

// getCreatePlanTool は ToolRegistry から create_plan ツールを取得
func (s *Supervisor) getCreatePlanTool() *planning.CreatePlanTool {
	tool := tools.DefaultRegistry.GetTool("create_plan")
	if tool == nil {
		return nil
	}
	if createPlanTool, ok := tool.(*planning.CreatePlanTool); ok {
		return createPlanTool
	}
	return nil
}

// runExecution は実装フェーズを実行
func (s *Supervisor) runExecution(ctx context.Context, p *plan.Plan) error {
	cyan.Println("\n🚀 Implementation Phase (parallel)")

	s.workerPool.Start(ctx)
	defer s.workerPool.Stop()

	for {
		// 実行可能なステップを取得（依存関係解決）
		executableSteps := s.getExecutableSteps(p)

		if len(executableSteps) == 0 {
			if p.IsCompleted() {
				break // 全完了
			}
			if p.HasFailed() {
				return errors.New("plan execution failed")
			}
			return errors.New("no executable steps but plan not completed (deadlock)")
		}

		// 並列実行して UI 更新
		results := s.workerPool.ExecuteStepsWithUI(ctx, executableSteps, s.confirmLevel)

		// 結果処理
		for _, result := range results {
			step := p.GetStep(result.StepID)
			if step == nil {
				continue
			}

			if err := s.handleResult(ctx, result, step, p); err != nil {
				return err
			}
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(p.Steps))
	return nil
}

// getExecutableSteps は実行可能なステップを取得
// 既存の Plan.CanExecute() を利用
func (s *Supervisor) getExecutableSteps(p *plan.Plan) []*plan.PlanStep {
	var steps []*plan.PlanStep
	for i := range p.Steps {
		if p.CanExecute(p.Steps[i].ID) {
			steps = append(steps, &p.Steps[i])
		}
	}
	return steps
}

// handleResult は結果を処理してリトライ/エスカレーション/ユーザー確認を判断
func (s *Supervisor) handleResult(ctx context.Context, result WorkerResult, step *plan.PlanStep, p *plan.Plan) error {
	if result.Success {
		p.UpdateStatus(step.ID, "completed", result.Output)
		s.sharedContext.MarkStepCompleted(step.ID)
		return nil
	}

	count := s.retryCount[step.ID]

	// エスカレーションが必要な場合
	if result.NeedsEscalation {
		cyan.Printf("%s Step %d escalating: %s\n", IconEscalated, step.ID, result.EscalationReason)
		return s.escalate(ctx, step, p)
	}

	// 1. リトライ回数が残っている場合
	if count < s.maxRetry {
		s.retryCount[step.ID] = count + 1
		yellow.Printf("%s Step %d failed (retry %d/%d)\n", IconRetrying, step.ID, count+1, s.maxRetry)
		s.workerPool.SubmitStep(step, s.confirmLevel)
		// 結果を待つ（この関数は結果収集ループ内で呼ばれるので、次のイテレーションで結果が来る）
		return nil
	}

	// 2. heavy_model が設定されている場合はエスカレーション
	if s.heavyProvider != nil {
		cyan.Printf("%s Step %d escalating to heavy model...\n", IconEscalated, step.ID)
		return s.escalate(ctx, step, p)
	}

	// 3. ユーザーに確認
	return s.promptUser(ctx, step, result, p)
}

// escalate は heavy_model でステップを再実行
func (s *Supervisor) escalate(ctx context.Context, step *plan.PlanStep, p *plan.Plan) error {
	if s.heavyProvider == nil {
		return fmt.Errorf("escalation needed but heavy_model not configured")
	}

	// heavy_model 用の Worker を一時的に作成して実行
	heavyWorker := &Worker{
		id:            -1, // 特別な ID
		provider:      s.heavyProvider,
		model:         s.heavyModel,
		sharedContext: s.sharedContext,
		agent:         s.agent,
		stepTimeout:   s.stepTimeout,
		commands:      make(chan WorkerCommand, 1),
		results:       make(chan WorkerResult, 1),
	}

	result := heavyWorker.executeStep(ctx, step, s.confirmLevel)

	if result.Success {
		p.UpdateStatus(step.ID, "completed", result.Output)
		s.sharedContext.MarkStepCompleted(step.ID)
		green.Printf("%s Step %d completed (heavy model)\n", IconCompleted, step.ID)
		return nil
	}

	// heavy_model でも失敗 → ユーザーに確認
	return s.promptUser(ctx, step, result, p)
}

// promptUser はユーザーにアクションを確認
func (s *Supervisor) promptUser(ctx context.Context, step *plan.PlanStep, result WorkerResult, p *plan.Plan) error {
	red.Printf("%s Step %d failed after all retries\n", IconFailed, step.ID)
	if result.Error != nil {
		red.Printf("   Error: %s\n", result.Error.Error())
	}

	// 既存の失敗処理 UI を使用（plan_failure.go）
	errMsg := ""
	if result.Error != nil {
		errMsg = result.Error.Error()
	}
	action, comment := promptFailureActionWithSelector(step, result.Output, errMsg, s.maxRetry)

	switch action {
	case plan.FailureActionRetry:
		// 手動リトライ
		s.retryCount[step.ID] = 0 // リトライカウントをリセット
		s.workerPool.SubmitStep(step, s.confirmLevel)
		return nil
	case plan.FailureActionComment:
		// コメント付きリトライ（履歴に追加）
		s.sharedContext.AddStepResult(step.ID, "[User feedback]: "+comment)
		s.retryCount[step.ID] = 0
		s.workerPool.SubmitStep(step, s.confirmLevel)
		return nil
	case plan.FailureActionSkip:
		yellow.Printf("%s Step %d skipped\n", IconWaiting, step.ID)
		p.UpdateStatus(step.ID, "completed", "skipped by user")
		s.sharedContext.MarkStepCompleted(step.ID) // スキップも完了扱い
		return nil
	case plan.FailureActionAbort:
		p.UpdateStatus(step.ID, "failed", errMsg)
		return fmt.Errorf("step %d aborted by user", step.ID)
	}

	return nil
}
