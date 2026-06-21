package evidence

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

var reviewWebSearchExternalDocIDRE = regexp.MustCompile(`^external-doc-(\d+)$`)

func nextReviewWebSearchExternalDocIndex(docs []externaldoc.Evidence) int {
	next := len(docs) + 1
	for _, doc := range docs {
		matches := reviewWebSearchExternalDocIDRE.FindStringSubmatch(doc.DocID)
		if len(matches) != 2 {
			continue
		}
		index, err := strconv.Atoi(matches[1])
		if err == nil && index >= next {
			next = index + 1
		}
	}
	return next
}

func reviewWebSearchExternalDocIDSet(docs []externaldoc.Evidence) map[string]struct{} {
	seen := make(map[string]struct{}, len(docs))
	for _, doc := range docs {
		if doc.DocID != "" {
			seen[doc.DocID] = struct{}{}
		}
	}
	return seen
}

func nextReviewWebSearchExternalDocID(nextIndex *int, seen map[string]struct{}) string {
	if *nextIndex <= 0 {
		*nextIndex = 1
	}
	for {
		docID := fmt.Sprintf("external-doc-%d", *nextIndex)
		if _, exists := seen[docID]; !exists {
			seen[docID] = struct{}{}
			return docID
		}
		*nextIndex++
	}
}
