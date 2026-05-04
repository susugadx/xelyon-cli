package tui

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	pdf "rsc.io/pdf"
)

type attachedPDFPreview struct {
	text      string
	truncated bool
}

func readAttachedPDFPreview(path string) (attachedPDFPreview, error) {
	f, err := os.Open(path)
	if err != nil {
		return attachedPDFPreview{}, fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return attachedPDFPreview{}, fmt.Errorf("failed to stat PDF file: %w", err)
	}

	reader, err := pdf.NewReader(f, info.Size())
	if err != nil {
		return attachedPDFPreview{}, fmt.Errorf("failed to parse PDF: %w", err)
	}
	return extractAttachedPDFPreview(reader)
}

func extractAttachedPDFPreview(reader *pdf.Reader) (attachedPDFPreview, error) {
	return extractAttachedPDFPreviewWithPageReader(
		reader.NumPage(),
		maxAttachedPDFPreviewPages,
		maxAttachedPDFPreviewChars,
		func(pageIndex int) (string, error) {
			return extractPDFPageText(reader.Page(pageIndex))
		},
	)
}

func extractAttachedPDFPreviewWithPageReader(totalPages, pageLimit, charLimit int, readPageText func(pageIndex int) (string, error)) (attachedPDFPreview, error) {
	if totalPages <= 0 || pageLimit <= 0 || charLimit <= 0 {
		return attachedPDFPreview{}, nil
	}

	maxPagesToRead := totalPages
	truncatedByPageLimit := false
	if maxPagesToRead > pageLimit {
		maxPagesToRead = pageLimit
		truncatedByPageLimit = true
	}

	var out strings.Builder
	charsWritten := 0
	truncated := truncatedByPageLimit

	for pageIndex := 1; pageIndex <= maxPagesToRead; pageIndex++ {
		if charsWritten >= charLimit {
			truncated = true
			break
		}

		pageText, err := readPageText(pageIndex)
		if err != nil {
			return attachedPDFPreview{}, fmt.Errorf("failed to extract page %d: %w", pageIndex, err)
		}
		if pageText == "" {
			continue
		}

		if out.Len() > 0 {
			if appendTextWithinRuneLimit(&out, "\n\n", &charsWritten, charLimit) {
				truncated = true
				break
			}
		}
		if appendTextWithinRuneLimit(&out, pageText, &charsWritten, charLimit) {
			truncated = true
			break
		}
	}

	return attachedPDFPreview{text: out.String(), truncated: truncated}, nil
}

func extractPDFPageText(page pdf.Page) (string, error) {
	content, err := readPDFPageContentSafe(page)
	if err != nil {
		return "", err
	}
	return buildPDFTextFromParts(content.Text), nil
}

func readPDFPageContentSafe(page pdf.Page) (pdf.Content, error) {
	return readPDFPageContentWithRecover(page.Content)
}

func readPDFPageContentWithRecover(readContent func() pdf.Content) (content pdf.Content, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("panic while reading PDF page content: %v", rec)
		}
	}()
	content = readContent()
	return content, nil
}

func buildPDFTextFromParts(parts []pdf.Text) string {
	if len(parts) == 0 {
		return ""
	}

	filtered := make([]pdf.Text, 0, len(parts))
	for _, part := range parts {
		part.S = strings.TrimSpace(part.S)
		if part.S == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	if len(filtered) == 0 {
		return ""
	}

	sort.Sort(pdf.TextVertical(filtered))

	var out strings.Builder
	prev := filtered[0]
	out.WriteString(prev.S)

	for i := 1; i < len(filtered); i++ {
		cur := filtered[i]
		switch {
		case shouldBreakPDFLine(prev, cur):
			out.WriteByte('\n')
		case shouldInsertPDFSpace(prev, cur):
			out.WriteByte(' ')
		}
		out.WriteString(cur.S)
		prev = cur
	}

	return strings.TrimSpace(out.String())
}

func shouldBreakPDFLine(prev, cur pdf.Text) bool {
	yDelta := math.Abs(prev.Y - cur.Y)
	lineBreakThreshold := math.Max(1.5, math.Max(prev.FontSize, cur.FontSize)*0.6)
	return yDelta > lineBreakThreshold
}

func shouldInsertPDFSpace(prev, cur pdf.Text) bool {
	gap := cur.X - (prev.X + prev.W)
	spaceThreshold := math.Max(1.0, math.Max(prev.FontSize, cur.FontSize)*0.25)
	return gap > spaceThreshold
}

func appendTextWithinRuneLimit(dst *strings.Builder, text string, currentCount *int, limit int) bool {
	if text == "" {
		return false
	}
	if *currentCount >= limit {
		return true
	}

	runes := []rune(text)
	remaining := limit - *currentCount
	if len(runes) > remaining {
		dst.WriteString(string(runes[:remaining]))
		*currentCount = limit
		return true
	}
	dst.WriteString(text)
	*currentCount += len(runes)
	return false
}
