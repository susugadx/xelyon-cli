package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) pendingApprovedPlan() string {
	if a == nil {
		return ""
	}
	if plan := strings.TrimSpace(a.PendingApprovedPlan); plan != "" {
		return plan
	}
	if a.session == nil {
		return ""
	}
	return strings.TrimSpace(a.session.PendingApprovedPlan)
}

func (a *Agent) pendingApprovedPlanHasChanges() bool {
	if a == nil {
		return false
	}
	if strings.TrimSpace(a.PendingApprovedPlan) != "" {
		return a.PendingApprovedPlanHasChanges
	}
	if a.session == nil || strings.TrimSpace(a.session.PendingApprovedPlan) == "" {
		return false
	}
	return a.session.PendingApprovedPlanHasChanges
}

func (a *Agent) pendingApprovedPlanChangedFiles() []string {
	if a == nil {
		return nil
	}
	if strings.TrimSpace(a.PendingApprovedPlan) != "" {
		return append([]string(nil), a.PendingApprovedPlanChangedFiles...)
	}
	if a.session == nil || strings.TrimSpace(a.session.PendingApprovedPlan) == "" {
		return nil
	}
	return append([]string(nil), a.session.PendingApprovedPlanChangedFiles...)
}

func clonePendingApprovedPlanChangedFiles(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(files))
	cloned := make([]string, 0, len(files))
	for _, file := range files {
		cloned = appendRecordedChangedFile(cloned, seen, file)
	}
	return cloned
}

func mergePendingApprovedPlanChangedFiles(existing []string, change *tools.FileChange) []string {
	merged := clonePendingApprovedPlanChangedFiles(existing)
	if change == nil {
		return merged
	}
	seen := make(map[string]bool, len(merged))
	for _, file := range merged {
		seen[file] = true
	}
	return collectRecordedChangedFiles(merged, seen, *change)
}

func (a *Agent) setPendingApprovedPlanState(plan string, hasChanges bool, changedFiles []string) {
	if a == nil {
		return
	}

	plan = strings.TrimSpace(plan)
	changedFiles = clonePendingApprovedPlanChangedFiles(changedFiles)
	a.PendingApprovedPlan = plan
	a.PendingApprovedPlanHasChanges = plan != "" && hasChanges
	if plan == "" {
		changedFiles = nil
	}
	a.PendingApprovedPlanChangedFiles = changedFiles
	if a.session != nil {
		a.session.PendingApprovedPlan = plan
		a.session.PendingApprovedPlanHasChanges = a.PendingApprovedPlanHasChanges
		a.session.PendingApprovedPlanChangedFiles = append([]string(nil), changedFiles...)
	}
	a.persistSession()
}

func (a *Agent) setPendingApprovedPlan(plan string) {
	a.setPendingApprovedPlanState(plan, false, nil)
}

func (a *Agent) noteApprovedPlanRecordedChange(change *tools.FileChange) {
	if a == nil || change == nil {
		return
	}

	handoff := strings.TrimSpace(a.activeApprovedPlan)
	if handoff == "" || handoff != a.pendingApprovedPlan() {
		return
	}

	a.PendingApprovedPlanHasChanges = true
	a.PendingApprovedPlanChangedFiles = mergePendingApprovedPlanChangedFiles(a.pendingApprovedPlanChangedFiles(), change)
	if a.session != nil {
		a.session.PendingApprovedPlanHasChanges = true
		a.session.PendingApprovedPlanChangedFiles = append([]string(nil), a.PendingApprovedPlanChangedFiles...)
	}
	a.persistSession()
}

func (a *Agent) restoreApprovedPlanStateFromSession() {
	if a == nil {
		return
	}
	if a.session == nil {
		a.PendingApprovedPlan = ""
		a.PendingApprovedPlanHasChanges = false
		a.PendingApprovedPlanChangedFiles = nil
		return
	}
	a.PendingApprovedPlan = strings.TrimSpace(a.session.PendingApprovedPlan)
	a.PendingApprovedPlanHasChanges = a.session.PendingApprovedPlanHasChanges && a.PendingApprovedPlan != ""
	a.PendingApprovedPlanChangedFiles = nil
	if a.PendingApprovedPlan != "" {
		a.PendingApprovedPlanChangedFiles = clonePendingApprovedPlanChangedFiles(a.session.PendingApprovedPlanChangedFiles)
	}
}

func (a *Agent) syncApprovedPlanStateToSession() {
	if a == nil || a.session == nil {
		return
	}
	a.session.PendingApprovedPlan = strings.TrimSpace(a.PendingApprovedPlan)
	a.session.PendingApprovedPlanHasChanges = a.PendingApprovedPlanHasChanges && strings.TrimSpace(a.PendingApprovedPlan) != ""
	if strings.TrimSpace(a.PendingApprovedPlan) == "" {
		a.session.PendingApprovedPlanChangedFiles = nil
		return
	}
	a.session.PendingApprovedPlanChangedFiles = clonePendingApprovedPlanChangedFiles(a.PendingApprovedPlanChangedFiles)
}

func (a *Agent) clearApprovedPlanState() {
	if a == nil {
		return
	}
	a.setPendingApprovedPlanState("", false, nil)
	a.activeApprovedPlan = ""
}

func (a *Agent) beginApprovedPlanHandoff(req *chatRequest) {
	if a == nil || req == nil {
		return
	}
	if req.approvedPlanHandoff == "" {
		req.approvedPlanHandoff = a.pendingApprovedPlan()
	}
	a.activeApprovedPlan = req.approvedPlanHandoff
}

func (a *Agent) finishApprovedPlanHandoff(req *chatRequest) {
	if a == nil || req == nil {
		return
	}
	a.finalizeApprovedPlanHandoff(req.approvedPlanHandoff)
}

func (a *Agent) finalizeApprovedPlanHandoff(handoff string) {
	if a == nil {
		return
	}

	handoff = strings.TrimSpace(handoff)
	if handoff != "" && a.pendingApprovedPlan() == handoff {
		a.setPendingApprovedPlan("")
	}
	a.activeApprovedPlan = ""
}
