package promptreduction

// ReviewPromptReductionReport は review model prompt 専用削減の集計結果。
type ReviewPromptReductionReport struct {
	Mode                             ReviewPromptReductionMode
	CandidateCount                   int
	ReplacedCount                    int
	StateSummaryCount                int
	AbsorbedItemCount                int
	RawOutputLedgerCount             int
	RawOutputRequiredRefCount        int
	RawOutputRehydratedRefCount      int
	RawOutputMissingRefCount         int
	RawOutputBudgetExhaustedRefCount int
	EstimatedSavedBytes              int
	ApproxEstimatedSavedTokens       int
	ReplacementSavedBytes            int
	ApproxReplacementSavedTokens     int
	ClassifierCounts                 map[string]int
	FamilyCounts                     map[string]int
	StatusCounts                     map[string]int
	KeptReasonCounts                 map[string]int
	RawOutputLedgers                 []ReviewProbeRawOutputLedger
	QualityFloorPreserved            bool
}

// CloneReviewPromptReductionReport は report を defensive copy する。
func CloneReviewPromptReductionReport(report ReviewPromptReductionReport) ReviewPromptReductionReport {
	if len(report.ClassifierCounts) > 0 {
		counts := make(map[string]int, len(report.ClassifierCounts))
		for classifier, count := range report.ClassifierCounts {
			counts[classifier] = count
		}
		report.ClassifierCounts = counts
	}
	if len(report.FamilyCounts) > 0 {
		counts := make(map[string]int, len(report.FamilyCounts))
		for family, count := range report.FamilyCounts {
			counts[family] = count
		}
		report.FamilyCounts = counts
	}
	if len(report.StatusCounts) > 0 {
		counts := make(map[string]int, len(report.StatusCounts))
		for status, count := range report.StatusCounts {
			counts[status] = count
		}
		report.StatusCounts = counts
	}
	if len(report.KeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.KeptReasonCounts))
		for reason, count := range report.KeptReasonCounts {
			counts[reason] = count
		}
		report.KeptReasonCounts = counts
	}
	if len(report.RawOutputLedgers) > 0 {
		report.RawOutputLedgers = cloneReviewProbeRawOutputLedgers(report.RawOutputLedgers)
	}
	return report
}

// ReviewPromptReductionReportIsEmpty は report が未記録かどうかを返す。
func ReviewPromptReductionReportIsEmpty(report ReviewPromptReductionReport) bool {
	return NormalizeReviewPromptReductionMode(report.Mode) == ReviewPromptReductionModeOff &&
		report.CandidateCount == 0 &&
		report.ReplacedCount == 0 &&
		report.EstimatedSavedBytes == 0 &&
		report.ApproxEstimatedSavedTokens == 0 &&
		report.ReplacementSavedBytes == 0 &&
		report.ApproxReplacementSavedTokens == 0 &&
		report.StateSummaryCount == 0 &&
		report.AbsorbedItemCount == 0 &&
		report.RawOutputLedgerCount == 0 &&
		report.RawOutputRequiredRefCount == 0 &&
		report.RawOutputRehydratedRefCount == 0 &&
		report.RawOutputMissingRefCount == 0 &&
		report.RawOutputBudgetExhaustedRefCount == 0 &&
		len(report.ClassifierCounts) == 0 &&
		len(report.FamilyCounts) == 0 &&
		len(report.StatusCounts) == 0 &&
		len(report.KeptReasonCounts) == 0 &&
		len(report.RawOutputLedgers) == 0 &&
		!report.QualityFloorPreserved
}

// Stats は review prompt reduction の集計 owner。
type Stats struct {
	report ReviewPromptReductionReport
}

// NewStats は指定 mode 用の reduction stats を構築する。
func NewStats(mode ReviewPromptReductionMode) *Stats {
	return &Stats{
		report: ReviewPromptReductionReport{
			Mode:                  NormalizeReviewPromptReductionMode(mode),
			ClassifierCounts:      map[string]int{},
			FamilyCounts:          map[string]int{},
			StatusCounts:          map[string]int{},
			KeptReasonCounts:      map[string]int{},
			QualityFloorPreserved: NormalizeReviewPromptReductionMode(mode) != ReviewPromptReductionModeOff,
		},
	}
}

// RecordCandidate は reduction candidate と適用済み savings を記録する。
func (s *Stats) RecordCandidate(classifier string, savedBytes, savedTokens int, applied bool) {
	if s == nil {
		return
	}
	s.report.CandidateCount++
	s.report.EstimatedSavedBytes += savedBytes
	s.report.ApproxEstimatedSavedTokens += savedTokens
	if classifier != "" {
		if s.report.ClassifierCounts == nil {
			s.report.ClassifierCounts = map[string]int{}
		}
		s.report.ClassifierCounts[classifier]++
	}
	if !applied {
		return
	}
	s.report.ReplacedCount++
	s.report.ReplacementSavedBytes += savedBytes
	s.report.ApproxReplacementSavedTokens += savedTokens
}

// RecordStateSummary は state summary の挿入と吸収 item 数を記録する。
func (s *Stats) RecordStateSummary(absorbedItems int) {
	if s == nil {
		return
	}
	s.report.StateSummaryCount++
	if absorbedItems > 0 {
		s.report.AbsorbedItemCount += absorbedItems
	}
}

// RecordItem は reduction item の family/status を記録する。
func (s *Stats) RecordItem(item ReviewPromptReductionItem) {
	if s == nil {
		return
	}
	if item.Family != "" {
		if s.report.FamilyCounts == nil {
			s.report.FamilyCounts = map[string]int{}
		}
		s.report.FamilyCounts[string(item.Family)]++
	}
	if item.Status != "" {
		if s.report.StatusCounts == nil {
			s.report.StatusCounts = map[string]int{}
		}
		s.report.StatusCounts[string(item.Status)]++
	}
}

// RecordKeepReason は reduction を適用しなかった理由を記録する。
func (s *Stats) RecordKeepReason(reason string) {
	if s == nil || reason == "" {
		return
	}
	if s.report.KeptReasonCounts == nil {
		s.report.KeptReasonCounts = map[string]int{}
	}
	s.report.KeptReasonCounts[reason]++
}

// RecordRawOutputLedger は raw output rehydrate ledger の集計を記録する。
func (s *Stats) RecordRawOutputLedger(ledger ReviewProbeRawOutputLedger) {
	if s == nil || ledger.empty() {
		return
	}
	s.report.RawOutputLedgerCount++
	s.report.RawOutputRequiredRefCount += len(ledger.RequiredRefs)
	s.report.RawOutputRehydratedRefCount += len(ledger.RehydratedRefs)
	s.report.RawOutputMissingRefCount += len(ledger.MissingRefs)
	s.report.RawOutputBudgetExhaustedRefCount += len(ledger.BudgetExhaustedRefs)
	s.report.RawOutputLedgers = append(s.report.RawOutputLedgers, cloneReviewProbeRawOutputLedger(ledger))
	if ledger.FailClosedReason != "" {
		s.RecordKeepReason(ledger.FailClosedReason)
	}
}

// Report は現在の集計を defensive copy で返す。
func (s *Stats) Report() ReviewPromptReductionReport {
	if s == nil {
		return ReviewPromptReductionReport{}
	}
	return CloneReviewPromptReductionReport(s.report)
}

func cloneReviewProbeRawOutputLedgers(ledgers []ReviewProbeRawOutputLedger) []ReviewProbeRawOutputLedger {
	if len(ledgers) == 0 {
		return nil
	}
	cloned := make([]ReviewProbeRawOutputLedger, 0, len(ledgers))
	for _, ledger := range ledgers {
		cloned = append(cloned, cloneReviewProbeRawOutputLedger(ledger))
	}
	return cloned
}

func cloneReviewProbeRawOutputLedger(ledger ReviewProbeRawOutputLedger) ReviewProbeRawOutputLedger {
	ledger.RequiredRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.RequiredRefs)
	ledger.OptionalRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.OptionalRefs)
	ledger.RehydratedRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.RehydratedRefs)
	ledger.MetadataOnlyRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.MetadataOnlyRefs)
	ledger.MissingRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.MissingRefs)
	ledger.BudgetExhaustedRefs = cloneReviewProbeRawOutputLedgerRefs(ledger.BudgetExhaustedRefs)
	return ledger
}

func cloneReviewProbeRawOutputLedgerRefs(refs []ReviewProbeRawOutputLedgerRef) []ReviewProbeRawOutputLedgerRef {
	if len(refs) == 0 {
		return nil
	}
	cloned := make([]ReviewProbeRawOutputLedgerRef, 0, len(refs))
	for _, ref := range refs {
		if ref.CommandIndex != nil {
			index := *ref.CommandIndex
			ref.CommandIndex = &index
		}
		cloned = append(cloned, ref)
	}
	return cloned
}
