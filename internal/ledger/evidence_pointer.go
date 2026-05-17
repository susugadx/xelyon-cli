package ledger

func evidencePointersFromFacts(facts []evidenceFact) []EvidencePointer {
	if len(facts) == 0 {
		return nil
	}
	pointers := make([]EvidencePointer, 0, len(facts))
	for _, fact := range facts {
		pointers = append(pointers, evidencePointerFromFact(fact))
	}
	return pointers
}

func evidencePointerFromFact(fact evidenceFact) EvidencePointer {
	return EvidencePointer{
		Path:       fact.path,
		StartLine:  fact.startLine,
		EndLine:    fact.endLine,
		Source:     fact.source,
		ToolCallID: fact.toolCallID,
		FileHash:   fact.fileHash,
		Stale:      fact.stale,
		PathBase:   EvidencePointerPathBaseRepoRoot,
	}
}
