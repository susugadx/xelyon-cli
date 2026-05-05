package review

import (
	"strconv"
	"strings"
)

// RenderReviewEvidenceMarkdown は ReviewEvidenceBundle を LLM 入力 Markdown に変換する。
func RenderReviewEvidenceMarkdown(bundle ReviewEvidenceBundle) string {
	input := BuildReviewEvidenceModelInput(bundle)
	var b strings.Builder

	b.WriteString("# Review Evidence\n\n")
	appendReviewEvidenceMarkdownSection(&b, "target kind")
	appendReviewEvidenceMarkdownFence(&b, "text", string(input.TargetKind))
	appendReviewEvidenceMarkdownSection(&b, "repo root display")
	appendReviewEvidenceMarkdownFence(&b, "text", input.RepoRoot)
	appendReviewEvidenceMarkdownSection(&b, "cwd display")
	appendReviewEvidenceMarkdownFence(&b, "text", input.CWDDisplay)
	appendReviewEvidenceMarkdownSection(&b, "git status --short")
	appendReviewEvidenceMarkdownFence(&b, "text", input.GitStatusShort.Content)
	appendReviewEvidenceMarkdownSection(&b, "change inventory")
	appendReviewEvidenceMarkdownJSONFence(&b, input.ChangeInventory)
	appendReviewEvidenceMarkdownSection(&b, "changed files")
	appendReviewEvidenceMarkdownJSONFence(&b, input.ChangedFiles)

	appendReviewEvidenceMarkdownSection(&b, "rule files")
	appendReviewEvidenceMarkdownJSONFence(&b, reviewEvidenceRuleFileMetadataInputs(input.RuleFiles))
	for _, file := range input.RuleFiles {
		appendReviewEvidenceMarkdownSubsection(&b, "rule file: "+file.Path)
		appendReviewEvidenceMarkdownFence(&b, "text", file.Content)
	}

	appendReviewEvidenceMarkdownSection(&b, "diffs")
	if len(input.Diffs) == 0 {
		appendReviewEvidenceMarkdownJSONFence(&b, input.Diffs)
	} else {
		for _, diff := range input.Diffs {
			appendReviewEvidenceMarkdownSubsection(&b, "diff: "+diff.Source)
			appendReviewEvidenceMarkdownSmallHeading(&b, "stat")
			appendReviewEvidenceMarkdownFence(&b, "text", diff.Stat.Content)
			appendReviewEvidenceMarkdownSmallHeading(&b, "name-status")
			appendReviewEvidenceMarkdownFence(&b, "text", diff.NameStatus.Content)
			appendReviewEvidenceMarkdownSmallHeading(&b, "diff body")
			appendReviewEvidenceMarkdownFence(&b, "diff", diff.Diff.Content)
		}
	}

	appendReviewEvidenceMarkdownSection(&b, "untracked files")
	appendReviewEvidenceMarkdownJSONFence(&b, reviewEvidenceUntrackedFileMetadataInputs(input.UntrackedFiles))
	for _, file := range input.UntrackedFiles {
		appendReviewEvidenceMarkdownSubsection(&b, "untracked file: "+file.Path)
		appendReviewEvidenceMarkdownFence(&b, "text", reviewEvidenceUntrackedFileBody(file))
	}

	appendReviewEvidenceMarkdownSection(&b, "limits")
	appendReviewEvidenceMarkdownJSONFence(&b, input.Limits)
	appendReviewEvidenceMarkdownSection(&b, "truncation flags")
	appendReviewEvidenceMarkdownJSONFence(&b, input.TruncationFlags)

	return b.String()
}

func reviewEvidenceRuleFileMetadataInputs(files []ReviewEvidenceRuleFileInput) []reviewEvidenceRuleFileMetadataInput {
	result := make([]reviewEvidenceRuleFileMetadataInput, 0, len(files))
	for _, file := range files {
		result = append(result, reviewEvidenceRuleFileMetadataInput{
			Path:      file.Path,
			Truncated: file.Truncated,
			SizeBytes: file.SizeBytes,
		})
	}
	return result
}

func reviewEvidenceUntrackedFileMetadataInputs(files []ReviewEvidenceUntrackedFileInput) []reviewEvidenceUntrackedFileMetadataInput {
	result := make([]reviewEvidenceUntrackedFileMetadataInput, 0, len(files))
	for _, file := range files {
		result = append(result, reviewEvidenceUntrackedFileMetadataInput{
			Path:      file.Path,
			Symlink:   file.Symlink,
			Binary:    file.Binary,
			Truncated: file.Truncated,
			SizeBytes: file.SizeBytes,
			ReadBytes: file.ReadBytes,
		})
	}
	return result
}

type reviewEvidenceRuleFileMetadataInput struct {
	Path      string `json:"path"`
	Truncated bool   `json:"truncated"`
	SizeBytes int64  `json:"size_bytes"`
}

type reviewEvidenceUntrackedFileMetadataInput struct {
	Path      string `json:"path"`
	Symlink   bool   `json:"symlink"`
	Binary    bool   `json:"binary"`
	Truncated bool   `json:"truncated"`
	SizeBytes int64  `json:"size_bytes"`
	ReadBytes int64  `json:"read_bytes"`
}

func reviewEvidenceUntrackedFileBody(file ReviewEvidenceUntrackedFileInput) string {
	if file.Symlink {
		return file.LinkTarget
	}
	return file.Snapshot
}

func appendReviewEvidenceMarkdownSection(b *strings.Builder, title string) {
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n")
}

func appendReviewEvidenceMarkdownSubsection(b *strings.Builder, title string) {
	b.WriteString("\n### ")
	b.WriteString(sanitizeReviewEvidenceMarkdownHeadingTitle(title))
	b.WriteString("\n")
}

func sanitizeReviewEvidenceMarkdownHeadingTitle(title string) string {
	for i := 0; i < len(title); i++ {
		if title[i] < 0x20 || title[i] == 0x7f {
			return strconv.Quote(title)
		}
	}
	return title
}

func appendReviewEvidenceMarkdownSmallHeading(b *strings.Builder, title string) {
	b.WriteString("\n#### ")
	b.WriteString(title)
	b.WriteString("\n")
}

func appendReviewEvidenceMarkdownJSONFence(b *strings.Builder, value any) {
	data, err := marshalReviewEvidenceJSONIndent(value)
	if err != nil {
		appendReviewEvidenceMarkdownFence(b, "json", "null")
		return
	}
	appendReviewEvidenceMarkdownFence(b, "json", string(data))
}

func appendReviewEvidenceMarkdownFence(b *strings.Builder, language, content string) {
	fence := reviewEvidenceMarkdownFence(content)
	b.WriteString(fence)
	b.WriteString(language)
	b.WriteByte('\n')
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString(fence)
	b.WriteString("\n\n")
}

func reviewEvidenceMarkdownFence(content string) string {
	return strings.Repeat("`", max(3, longestReviewEvidenceBacktickRun(content)+1))
}

func longestReviewEvidenceBacktickRun(content string) int {
	longest := 0
	current := 0
	for _, r := range content {
		if r == '`' {
			current++
			longest = max(longest, current)
			continue
		}
		current = 0
	}
	return longest
}
