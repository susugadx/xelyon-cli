package review

import "time"

// ReviewProgressPhase は /review runner の進行段階を表す。
type ReviewProgressPhase string

const (
	// ReviewProgressPhaseEvidence は current changes evidence 収集段階を表す。
	ReviewProgressPhaseEvidence ReviewProgressPhase = "evidence"
	// ReviewProgressPhaseProbePlan は probe plan 生成段階を表す。
	ReviewProgressPhaseProbePlan ReviewProgressPhase = "probe_plan"
	// ReviewProgressPhaseProbe は planned probe 実行段階を表す。
	ReviewProgressPhaseProbe ReviewProgressPhase = "probe"
	// ReviewProgressPhaseReport は review report 生成段階を表す。
	ReviewProgressPhaseReport ReviewProgressPhase = "report"
	// ReviewProgressPhaseSaturationCheck は report の漏れ確認段階を表す。
	ReviewProgressPhaseSaturationCheck ReviewProgressPhase = "saturation_check"
	// ReviewProgressPhaseReportRevision は saturation 指摘後の report 修正段階を表す。
	ReviewProgressPhaseReportRevision ReviewProgressPhase = "report_revision"
)

// ReviewProgressStatus は /review progress item の状態を表す。
type ReviewProgressStatus string

const (
	// ReviewProgressRunning は item が進行中であることを表す。
	ReviewProgressRunning ReviewProgressStatus = "running"
	// ReviewProgressOK は item が正常完了したことを表す。
	ReviewProgressOK ReviewProgressStatus = "ok"
	// ReviewProgressError は item が失敗または blocked になったことを表す。
	ReviewProgressError ReviewProgressStatus = "error"
)

// ReviewProgressEvent は ReviewRunner から UI/adapter へ渡す進行イベントを表す。
//
// UI には依存せず、prompt/raw JSON/stdout 全文ではなく、runner phase と probe command の
// 短い要約だけを伝える。
type ReviewProgressEvent struct {
	ID       string
	Phase    ReviewProgressPhase
	Status   ReviewProgressStatus
	Label    string
	Detail   string
	Duration time.Duration
}

// ReviewProgressSink は /review progress event の受け取り口を表す。
type ReviewProgressSink func(ReviewProgressEvent)

func (s ReviewProgressSink) emit(event ReviewProgressEvent) {
	if s == nil {
		return
	}
	s(event)
}
