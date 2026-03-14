package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestReadTracker_NoGuidanceBeforeThreshold(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// 1st and 2nd read should not produce guidance
	for i := 0; i < readTrackerThreshold-1; i++ {
		guidance := rt.record(tc)
		if guidance != "" {
			t.Errorf("read %d: expected no guidance, got %q", i+1, guidance)
		}
	}
}

func TestReadTracker_GuidanceAtThreshold(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Record up to threshold-1 (no guidance yet)
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(tc)
	}

	// The threshold-th call should produce guidance
	guidance := rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Errorf("expected guidance at threshold, got %q", guidance)
	}
}

func TestReadTracker_GuidanceOnlyOnceAfterConfirm(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Trigger guidance and confirm it was shown
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(tc)
	}
	guidance := rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Fatal("expected guidance at threshold")
	}
	// Simulate successful append
	appendGuidanceWithConfirm(rt, "foo.go", "success result", guidance)

	// Subsequent reads should NOT produce guidance again
	for i := 0; i < 3; i++ {
		guidance := rt.record(tc)
		if guidance != "" {
			t.Errorf("read %d after confirmed guidance: expected empty, got %q", i+1, guidance)
		}
	}
}

func TestReadTracker_ErrorResultDoesNotConsumeGuidance(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Build up to threshold
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(tc)
	}

	// Threshold-th call returns guidance
	guidance := rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Fatal("expected guidance at threshold")
	}

	// But the result is an error — guidance is NOT appended, shown is NOT confirmed
	errorResult := "Error: file not found"
	got := appendGuidanceWithConfirm(rt, "foo.go", errorResult, guidance)
	if got != errorResult {
		t.Errorf("error result should not have guidance, got %q", got)
	}
	if rt.guidanceShown["foo.go"] {
		t.Error("guidanceShown should NOT be true after error result")
	}

	// Next successful read should still produce guidance (not consumed)
	guidance = rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Error("expected guidance to be available again after error result")
	}

	// This time append succeeds
	successResult := "file contents"
	got = appendGuidanceWithConfirm(rt, "foo.go", successResult, guidance)
	if !strings.Contains(got, "[GUIDANCE]") {
		t.Errorf("expected guidance appended to success result, got %q", got)
	}
	if !rt.guidanceShown["foo.go"] {
		t.Error("guidanceShown should be true after successful append")
	}

	// Now it should be consumed — no more guidance
	guidance = rt.record(tc)
	if guidance != "" {
		t.Errorf("expected no guidance after confirmed show, got %q", guidance)
	}
}

func TestReadTracker_WriteResetsGuidanceShown(t *testing.T) {
	rt := newReadTracker()
	readTC := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}
	writeTC := &tools.ToolCall{
		Tool: "str_replace",
		Args: map[string]string{"path": "foo.go"},
	}

	// Trigger and confirm guidance
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(readTC)
	}
	guidance := rt.record(readTC)
	appendGuidanceWithConfirm(rt, "foo.go", "ok", guidance)

	// Write resets both count and guidanceShown
	rt.record(writeTC)

	// After write, reading again can trigger guidance at threshold
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(readTC)
	}
	guidance = rt.record(readTC)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Error("expected guidance to be available again after write reset")
	}
}

func TestReadTracker_ResetAllowsGuidanceAgain(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Trigger and confirm guidance
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(tc)
	}
	guidance := rt.record(tc)
	appendGuidanceWithConfirm(rt, "foo.go", "ok", guidance)

	// Reset
	rt.reset()

	// After reset, build up again
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(tc)
	}
	guidance = rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Error("expected guidance to be available again after reset")
	}
}

func TestReadTracker_DifferentFilesIndependent(t *testing.T) {
	rt := newReadTracker()
	tc1 := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}
	tc2 := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "bar.go"},
	}

	// Read foo.go twice, bar.go once
	rt.record(tc1)
	rt.record(tc1)
	guidance := rt.record(tc2)
	if guidance != "" {
		t.Errorf("bar.go first read should not trigger guidance, got %q", guidance)
	}
}

func TestReadTracker_WriteToolResetsCount(t *testing.T) {
	rt := newReadTracker()
	readTC := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}
	writeTC := &tools.ToolCall{
		Tool: "str_replace",
		Args: map[string]string{"path": "foo.go"},
	}

	// Read up to threshold - 1
	for i := 0; i < readTrackerThreshold-1; i++ {
		rt.record(readTC)
	}

	// Write resets the count
	rt.record(writeTC)

	// Next read should be count=1, no guidance
	guidance := rt.record(readTC)
	if guidance != "" {
		t.Errorf("expected no guidance after write reset, got %q", guidance)
	}
}

func TestReadTracker_NonReadToolIgnored(t *testing.T) {
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "search_code",
		Args: map[string]string{"pattern": "foo"},
	}

	guidance := rt.record(tc)
	if guidance != "" {
		t.Errorf("non-read_file tool should not produce guidance, got %q", guidance)
	}
}

func TestReadTracker_NilSafe(t *testing.T) {
	var rt *readTracker
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Should not panic
	guidance := rt.record(tc)
	if guidance != "" {
		t.Errorf("nil tracker should return empty, got %q", guidance)
	}
	rt.reset()                // should not panic
	rt.confirmShown("foo.go") // should not panic
}

func TestReadTracker_CountsEvenWhenResultIsDeduplicated(t *testing.T) {
	// Simulates the compactToolResult flow where dedupe fires but
	// tracker.record() is called first (before dedupe).
	// The tracker should count every read_file call regardless of dedupe.
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Simulate: record is called for every read, even if result would be deduped.
	// After threshold calls, guidance should be available.
	for i := 0; i < readTrackerThreshold-1; i++ {
		guidance := rt.record(tc)
		if guidance != "" {
			t.Errorf("read %d: expected no guidance, got %q", i+1, guidance)
		}
	}

	// Threshold-th call — guidance fires regardless of whether result is deduped
	guidance := rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Errorf("expected guidance at threshold even in dedupe scenario, got %q", guidance)
	}

	// Verify count matches expectations
	if rt.fileCounts["foo.go"] != readTrackerThreshold {
		t.Errorf("fileCounts = %d, want %d", rt.fileCounts["foo.go"], readTrackerThreshold)
	}
}

func TestReadTracker_GuidanceOnDeduplicatedResult(t *testing.T) {
	// When result is a dedupe reference string (not an Error), guidance should
	// still be appended and confirmShown should fire.
	rt := newReadTracker()
	tc := &tools.ToolCall{
		Tool: "read_file",
		Args: map[string]string{"path": "foo.go"},
	}

	// Build up to threshold
	for i := 0; i < readTrackerThreshold; i++ {
		rt.record(tc)
	}

	// threshold-th call returned guidance; reset guidanceShown to test dedupe path
	rt.record(tc)
	rt.guidanceShown["foo.go"] = false

	guidance := rt.record(tc)
	if !strings.Contains(guidance, "[GUIDANCE]") {
		t.Fatal("expected guidance candidate")
	}

	// appendGuidanceWithConfirm on a dedupe reference (non-error) should succeed
	dedupedResult := "[See previous read_file result at turn 2]"
	got := appendGuidanceWithConfirm(rt, "foo.go", dedupedResult, guidance)
	if !strings.Contains(got, "[GUIDANCE]") {
		t.Errorf("expected guidance appended to deduped result, got %q", got)
	}
	if !rt.guidanceShown["foo.go"] {
		t.Error("guidanceShown should be confirmed on deduped result")
	}
}

func TestAppendGuidanceWithConfirm_ErrorResultUnchanged(t *testing.T) {
	rt := newReadTracker()
	result := "Error: file not found"
	guidance := "\n\n[GUIDANCE] some guidance"

	got := appendGuidanceWithConfirm(rt, "foo.go", result, guidance)
	if got != result {
		t.Errorf("error result should not have guidance appended, got %q", got)
	}
	if rt.guidanceShown["foo.go"] {
		t.Error("guidanceShown should not be set for error result")
	}
}

func TestAppendGuidanceWithConfirm_EmptyGuidance(t *testing.T) {
	rt := newReadTracker()
	result := "file contents here"
	got := appendGuidanceWithConfirm(rt, "foo.go", result, "")
	if got != result {
		t.Errorf("empty guidance should return result unchanged, got %q", got)
	}
}

func TestAppendGuidanceWithConfirm_Normal(t *testing.T) {
	rt := newReadTracker()
	result := "file contents here"
	guidance := "\n\n[GUIDANCE] read less"
	got := appendGuidanceWithConfirm(rt, "foo.go", result, guidance)
	if got != result+guidance {
		t.Errorf("expected result+guidance, got %q", got)
	}
	if !rt.guidanceShown["foo.go"] {
		t.Error("guidanceShown should be set after successful append")
	}
}
