package tools

import "testing"

func TestObservationEvidence_NormalizeLineOnlyEvidence(t *testing.T) {
	evidence := ObservationEvidence{
		Path:      "pkg/caller.go",
		StartLine: 12,
		EndLine:   0,
		Excerpt:   "target()",
	}

	normalized := evidence.Normalize()

	if normalized.StartLine != 12 || normalized.EndLine != 12 {
		t.Fatalf("Normalize() = %d-%d, want 12-12", normalized.StartLine, normalized.EndLine)
	}
	if evidence.EndLine != 0 {
		t.Fatalf("Normalize mutated receiver EndLine = %d, want 0", evidence.EndLine)
	}
}

func TestMergeRuntimeObservations_NormalizesEvidenceBeforeDedupe(t *testing.T) {
	lineOnly := &RuntimeObservation{
		Evidence: []ObservationEvidence{{
			Path:         "pkg/caller.go",
			ResolvedPath: "/repo/pkg/caller.go",
			StartLine:    12,
			EndLine:      0,
			Excerpt:      "target()",
		}},
	}
	explicitRange := &RuntimeObservation{
		Evidence: []ObservationEvidence{{
			Path:         "pkg/caller.go",
			ResolvedPath: "/repo/pkg/caller.go",
			StartLine:    12,
			EndLine:      12,
			Excerpt:      "target()",
		}},
	}

	merged := MergeRuntimeObservations(lineOnly, explicitRange)

	if merged == nil || len(merged.Evidence) != 1 {
		t.Fatalf("merged evidence = %#v, want one normalized item", merged)
	}
	evidence := merged.Evidence[0]
	if evidence.StartLine != 12 || evidence.EndLine != 12 {
		t.Fatalf("merged evidence range = %d-%d, want 12-12", evidence.StartLine, evidence.EndLine)
	}
}
