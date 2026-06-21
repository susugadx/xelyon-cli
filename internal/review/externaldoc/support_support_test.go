package externaldoc

import (
	"strconv"
	"time"
)

func newExternalSupportDocForTest(docID string, credibility SourceCredibility, truncated bool, snippets ...string) Evidence {
	doc := Evidence{
		DocID:             docID,
		URL:               "https://docs.example.test/" + docID,
		SourceDomain:      "docs.example.test",
		SourceCredibility: credibility,
		FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		Truncated:         truncated,
		ContentHash:       reviewExternalDocContentHash("doc:" + docID),
	}
	for i, content := range snippets {
		doc.Snippets = append(doc.Snippets, SnippetEvidence{
			SnippetID:   docID + "-snippet-" + strconv.Itoa(i+1),
			Content:     content,
			ContentHash: reviewExternalDocContentHash(content),
			Truncated:   truncated,
		})
	}
	return doc
}
