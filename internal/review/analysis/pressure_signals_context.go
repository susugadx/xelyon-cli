package analysis

import "strings"

func reviewPressureSignalRelatedContextEvidence(input EvidenceInput) []string {
	evidence := make([]string, 0)
	if len(input.RelatedContextFiles) == 0 {
		evidence = append(evidence, "related_context_files: []")
	} else if reviewPressureSignalAllRelatedContextSkipped(input.RelatedContextFiles) {
		evidence = append(evidence, "related_context_files: all skipped")
	}
	if input.TruncationFlags.RelatedCandidates {
		evidence = append(evidence, "related_candidates: truncated")
	}
	for _, file := range input.RelatedContextFiles {
		if file.Skipped {
			evidence = append(evidence, reviewPressureSignalRelatedContextFileEvidence("related_context_file skipped", file))
		}
		if file.Truncated {
			evidence = append(evidence, "related_context_file truncated: "+file.Path)
		}
	}
	for _, flag := range input.TruncationFlags.RelatedContextFiles {
		if flag.Truncated {
			evidence = append(evidence, "related_context_truncation flag: "+flag.Path)
		}
	}
	return reviewPressureSignalDedupeEvidence(evidence)
}

func reviewPressureSignalAllRelatedContextSkipped(files []ContextFile) bool {
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if !file.Skipped {
			return false
		}
	}
	return true
}

func reviewPressureSignalRelatedContextFileEvidence(prefix string, file ContextFile) string {
	if strings.TrimSpace(file.SkipReason) == "" {
		return prefix + ": " + file.Path
	}
	return prefix + ": " + file.Path + " (" + file.SkipReason + ")"
}

func reviewPressureSignalRelatedSearchEvidence(input EvidenceInput) []string {
	evidence := make([]string, 0, 2)
	if len(input.RelatedSearchHits) == 0 {
		evidence = append(evidence, "related_search_hits: []")
	}
	if input.TruncationFlags.RelatedSearch {
		evidence = append(evidence, "related_search: truncated")
	}
	return evidence
}

func reviewPressureSignalDiffOrContextTruncatedEvidence(input EvidenceInput) []string {
	return reviewPressureSignalTruncationEvidence(input.TruncationFlags)
}

func reviewPressureSignalTruncationEvidence(flags TruncationFlags) []string {
	evidence := make([]string, 0)
	if flags.StatusShort {
		evidence = append(evidence, "status_short: truncated")
	}
	for _, diff := range flags.Diffs {
		if diff.Stat {
			evidence = append(evidence, "diff stat truncated: "+diff.Source)
		}
		if diff.NameStatus {
			evidence = append(evidence, "diff name_status truncated: "+diff.Source)
		}
		if diff.Diff {
			evidence = append(evidence, "diff body truncated: "+diff.Source)
		}
	}
	if flags.UntrackedList {
		evidence = append(evidence, "untracked_list: truncated")
	}
	if flags.UntrackedSnapshots {
		evidence = append(evidence, "untracked_snapshots: truncated")
	}
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("untracked_file", flags.UntrackedFiles)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("rule_file", flags.RuleFiles)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("changed_context_file", flags.ChangedFileContext)...)
	evidence = append(evidence, reviewPressureSignalPathTruncationEvidence("related_context_file", flags.RelatedContextFiles)...)
	return reviewPressureSignalDedupeEvidence(evidence)
}

func reviewPressureSignalPathTruncationEvidence(prefix string, flags []PathTruncation) []string {
	evidence := make([]string, 0, len(flags))
	for _, flag := range flags {
		if flag.Truncated {
			evidence = append(evidence, prefix+" truncated: "+flag.Path)
		}
	}
	return evidence
}
