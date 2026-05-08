package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderReviewEvidenceMarkdownSectionOrderAndPathRedaction(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)

	markdown := RenderReviewEvidenceMarkdown(bundle)

	assertReviewEvidenceSubstringsInOrder(t, markdown, []string{
		"## target kind\n",
		"## repo root display\n",
		"## cwd display\n",
		"## git status --short\n",
		"## change inventory\n",
		"## changed files\n",
		"## rule files\n",
		"## diffs\n",
		"## untracked files\n",
		"## limits\n",
		"## truncation flags\n",
	})
	assertReviewEvidenceSubstringsInOrder(t, markdown, []string{
		"### diff: unstaged\n",
		"### diff: staged\n",
	})

	if !strings.Contains(markdown, "<repo_root>") {
		t.Fatalf("markdown = %q, want repo root placeholder", markdown)
	}
	if !strings.Contains(markdown, "src/work") {
		t.Fatalf("markdown = %q, want repo-relative cwd display", markdown)
	}
	if strings.Contains(markdown, bundle.RepoRoot) || strings.Contains(markdown, bundle.CWD) {
		t.Fatalf("markdown leaked absolute path: %q", markdown)
	}
	for _, want := range []string{
		`"status_short": true`,
		`"untracked_list": true`,
		`"untracked_snapshots": true`,
		`"diff": true`,
		`"path": "AGENTS.md"`,
		`"truncated": true`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want truncation flag fragment %q", markdown, want)
		}
	}
}

func TestBuildReviewEvidenceModelInputCWDOutsideRepo(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	outside := filepath.Join(t.TempDir(), "elsewhere")
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		CWD:        outside,
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if input.CWDDisplay != reviewEvidenceOutsideRepoPathDisplay {
		t.Fatalf("CWDDisplay = %q, want %q", input.CWDDisplay, reviewEvidenceOutsideRepoPathDisplay)
	}
}

func TestBuildReviewEvidenceModelInputRedactsOutsideAbsoluteSymlinkTarget(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	outsideTarget := filepath.Join(t.TempDir(), "secret.txt")
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []ReviewUntrackedFile{{
			Path:       "outside-link",
			Symlink:    true,
			LinkTarget: outsideTarget,
		}},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if got := input.UntrackedFiles[0].LinkTarget; got != reviewEvidenceOutsideRepoPathDisplay {
		t.Fatalf("LinkTarget = %q, want %q", got, reviewEvidenceOutsideRepoPathDisplay)
	}

	data, err := RenderReviewEvidenceJSON(bundle)
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	jsonPayload := string(data)
	if strings.Contains(jsonPayload, outsideTarget) {
		t.Fatalf("json leaked outside symlink target %q: %s", outsideTarget, jsonPayload)
	}
	if !strings.Contains(jsonPayload, `"link_target": "<outside-repo>"`) {
		t.Fatalf("json = %s, want outside-repo symlink target", jsonPayload)
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)
	if strings.Contains(markdown, outsideTarget) {
		t.Fatalf("markdown leaked outside symlink target %q: %s", outsideTarget, markdown)
	}
	if !strings.Contains(markdown, "```text\n<outside-repo>\n```") {
		t.Fatalf("markdown = %q, want outside-repo symlink body", markdown)
	}
}

func TestBuildReviewEvidenceModelInputNormalizesRepoAbsoluteSymlinkTarget(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	target := filepath.Join(repo, "dir", "target.txt")
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []ReviewUntrackedFile{{
			Path:       filepath.Join(repo, "dir", "link"),
			Symlink:    true,
			LinkTarget: target,
		}},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if got := input.UntrackedFiles[0].LinkTarget; got != "dir/target.txt" {
		t.Fatalf("LinkTarget = %q, want %q", got, "dir/target.txt")
	}
}

func TestBuildReviewEvidenceModelInputSymlinkTargetsDoNotRequireRepoRootExists(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "missing-repo")
	if _, err := os.Stat(repo); !os.IsNotExist(err) {
		t.Fatalf("test repo root %q unexpectedly exists or stat failed with non-ENOENT error: %v", repo, err)
	}

	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []ReviewUntrackedFile{
			{
				Path:       "dir/absolute-link",
				Symlink:    true,
				LinkTarget: filepath.Join(repo, "dir", "target.txt"),
			},
			{
				Path:       "nested/link",
				Symlink:    true,
				LinkTarget: "../target.txt",
			},
			{
				Path:       "nested/escape-link",
				Symlink:    true,
				LinkTarget: "../../outside.txt",
			},
		},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	got := make([]string, 0, len(input.UntrackedFiles))
	for _, file := range input.UntrackedFiles {
		got = append(got, file.LinkTarget)
	}
	assertStringSlice(t, got, []string{
		"dir/target.txt",
		"target.txt",
		reviewEvidenceOutsideRepoPathDisplay,
	})
}

func TestBuildReviewEvidenceModelInputResolvesRelativeSymlinkTargetFromLinkParent(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	tests := []struct {
		name        string
		symlinkPath string
		linkTarget  string
		want        string
	}{
		{
			name:        "inside repo from parent",
			symlinkPath: "nested/link",
			linkTarget:  "../target.txt",
			want:        "target.txt",
		},
		{
			name:        "inside repo under parent sibling",
			symlinkPath: "nested/deeper/link",
			linkTarget:  "../target.txt",
			want:        "nested/target.txt",
		},
		{
			name:        "outside repo escape",
			symlinkPath: "nested/link",
			linkTarget:  "../../outside.txt",
			want:        reviewEvidenceOutsideRepoPathDisplay,
		},
		{
			name:        "empty target stays empty",
			symlinkPath: "nested/link",
			linkTarget:  "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bundle := ReviewEvidenceBundle{
				TargetKind: TargetCurrentChanges,
				RepoRoot:   repo,
				UntrackedFiles: []ReviewUntrackedFile{{
					Path:       tt.symlinkPath,
					Symlink:    true,
					LinkTarget: tt.linkTarget,
				}},
			}

			input := BuildReviewEvidenceModelInput(bundle)

			if got := input.UntrackedFiles[0].LinkTarget; got != tt.want {
				t.Fatalf("LinkTarget = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildReviewEvidenceModelInputDoesNotResolveTruncatedSymlinkTarget(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []ReviewUntrackedFile{{
			Path:       "nested/link",
			Symlink:    true,
			LinkTarget: ".",
			Truncated:  true,
			SizeBytes:  int64(len("../../secret-outside")),
			ReadBytes:  int64(len(".")),
		}},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if got := input.UntrackedFiles[0].LinkTarget; got != reviewEvidenceTruncatedLinkTargetDisplay {
		t.Fatalf("LinkTarget = %q, want %q", got, reviewEvidenceTruncatedLinkTargetDisplay)
	}

	data, err := RenderReviewEvidenceJSON(bundle)
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	jsonPayload := string(data)
	if strings.Contains(jsonPayload, `"link_target": "nested"`) {
		t.Fatalf("json resolved truncated symlink target as complete path: %s", jsonPayload)
	}
	if !strings.Contains(jsonPayload, `"link_target": "<truncated-link-target>"`) {
		t.Fatalf("json = %s, want truncated symlink target placeholder", jsonPayload)
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)
	if strings.Contains(markdown, "```text\nnested\n```") {
		t.Fatalf("markdown resolved truncated symlink target as complete path: %q", markdown)
	}
	if !strings.Contains(markdown, "```text\n<truncated-link-target>\n```") {
		t.Fatalf("markdown = %q, want truncated symlink target placeholder", markdown)
	}
}

func TestRenderReviewEvidenceMarkdownQuotesControlledSubsectionTitle(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	bundle := ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []ReviewUntrackedFile{{
			Path:     "notes\n## limits",
			Snapshot: "body\n",
		}},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	if !strings.Contains(markdown, "### \"untracked file: notes\\n## limits\"\n") {
		t.Fatalf("markdown = %q, want quoted untracked subsection heading", markdown)
	}
	if strings.Contains(markdown, "### untracked file: notes\n## limits\n") {
		t.Fatalf("markdown contains raw controlled subsection heading: %q", markdown)
	}
	if got := strings.Count(markdown, "\n## limits\n"); got != 1 {
		t.Fatalf("limits section count = %d, want 1:\n%s", got, markdown)
	}
}

func TestRenderReviewEvidenceMarkdownExpandsFenceForBackticks(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)
	bundle.RuleFiles = []ReviewRuleFileEvidence{{
		Path:    "AGENTS.md",
		Content: "rule ````` content",
	}}
	bundle.Diffs = []ReviewDiffEvidence{{
		Source: reviewDiffEvidenceSourceUnstaged,
		Diff:   "+diff ````` body",
	}}
	bundle.UntrackedFiles = []ReviewUntrackedFile{{
		Path:     "notes.md",
		Snapshot: "untracked ````` snapshot",
	}}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"``````text\nrule ````` content\n``````",
		"``````diff\n+diff ````` body\n``````",
		"``````text\nuntracked ````` snapshot\n``````",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want expanded fence block %q", markdown, want)
		}
	}
}

func TestRenderReviewEvidenceJSONUsesModelInputDTO(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)

	data, err := RenderReviewEvidenceJSON(bundle)
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	payload := string(data)

	for _, forbidden := range []string{
		bundle.RepoRoot,
		bundle.CWD,
		`"RepoRoot"`,
		`"CWD"`,
		`"CommandTimeout"`,
		`"command_timeout":`,
		`1500000000`,
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("json leaked forbidden fragment %q: %s", forbidden, payload)
		}
	}
	for _, want := range []string{
		`"repo_root": "<repo_root>"`,
		`"cwd_display": "src/work"`,
		`"command_timeout_ms": 1500`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("json = %s, want fragment %q", payload, want)
		}
	}

	var input ReviewEvidenceModelInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if input.RepoRoot != reviewEvidenceRepoRootPathDisplay {
		t.Fatalf("RepoRoot = %q, want placeholder", input.RepoRoot)
	}
	if input.Limits.CommandTimeoutMS != 1500 {
		t.Fatalf("CommandTimeoutMS = %d, want 1500", input.Limits.CommandTimeoutMS)
	}
}

func TestBuildReviewEvidenceModelInputMinimalBundleUsesStableEmptySlices(t *testing.T) {
	input := BuildReviewEvidenceModelInput(ReviewEvidenceBundle{})

	if input.ChangedFiles == nil {
		t.Fatal("ChangedFiles = nil, want empty slice")
	}
	if input.RuleFiles == nil {
		t.Fatal("RuleFiles = nil, want empty slice")
	}
	if input.Diffs == nil {
		t.Fatal("Diffs = nil, want empty slice")
	}
	if input.UntrackedFiles == nil {
		t.Fatal("UntrackedFiles = nil, want empty slice")
	}
	if input.ChangeInventory.Generated == nil {
		t.Fatal("ChangeInventory.Generated = nil, want empty slice")
	}
	if input.TruncationFlags.Diffs == nil {
		t.Fatal("TruncationFlags.Diffs = nil, want empty slice")
	}
	if input.TruncationFlags.RuleFiles == nil {
		t.Fatal("TruncationFlags.RuleFiles = nil, want empty slice")
	}

	data, err := RenderReviewEvidenceJSON(ReviewEvidenceBundle{})
	if err != nil {
		t.Fatalf("RenderReviewEvidenceJSON() error = %v", err)
	}
	payload := string(data)
	for _, want := range []string{
		`"changed_files": []`,
		`"rule_files": []`,
		`"diffs": []`,
		`"untracked_files": []`,
	} {
		if !strings.Contains(payload, want) {
			t.Fatalf("json = %s, want empty slice fragment %q", payload, want)
		}
	}

	markdown := RenderReviewEvidenceMarkdown(ReviewEvidenceBundle{})
	for _, want := range []string{
		"## git status --short\n```text\n",
		"## diffs\n```json\n[]\n```",
		"## untracked files\n```json\n[]\n```",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want empty section fragment %q", markdown, want)
		}
	}
}

func newReviewEvidenceRenderTestBundle(t *testing.T) ReviewEvidenceBundle {
	t.Helper()

	repo := filepath.Clean(t.TempDir())
	cwd := filepath.Join(repo, "src", "work")

	return ReviewEvidenceBundle{
		TargetKind:           TargetCurrentChanges,
		RepoRoot:             repo,
		CWD:                  cwd,
		StatusShort:          " M src/main.go\n?? notes.md\n",
		StatusShortTruncated: true,
		Diffs: []ReviewDiffEvidence{
			{
				Source:              reviewDiffEvidenceSourceUnstaged,
				Stat:                " src/main.go | 2 +-\n",
				StatTruncated:       true,
				NameStatus:          "M\tsrc/main.go\n",
				NameStatusTruncated: true,
				Diff:                "diff --git a/src/main.go b/src/main.go\n+changed\n",
				DiffTruncated:       true,
			},
			{
				Source:     reviewDiffEvidenceSourceStaged,
				Stat:       " README.md | 1 +\n",
				NameStatus: "M\tREADME.md\n",
				Diff:       "diff --git a/README.md b/README.md\n+docs\n",
			},
		},
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     filepath.Join(repo, "src", "main.go"),
				OldPath:  filepath.Join(repo, "src", "old.go"),
				Status:   "R100",
				Staged:   true,
				Unstaged: true,
			},
		},
		UntrackedFiles: []ReviewUntrackedFile{
			{
				Path:      filepath.Join(repo, "notes.md"),
				Snapshot:  "draft notes\n",
				Truncated: true,
				SizeBytes: 64,
				ReadBytes: 32,
			},
		},
		UntrackedListTruncated:      true,
		UntrackedSnapshotsTruncated: true,
		RuleFiles: []ReviewRuleFileEvidence{
			{
				Path:      filepath.Join(repo, "AGENTS.md"),
				Content:   "rule text\n",
				Truncated: true,
				SizeBytes: 128,
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{filepath.Join(repo, "src", "main.go")},
			Docs:       []string{"README.md"},
			RenamedFiles: []string{
				filepath.Join(repo, "src", "main.go"),
			},
			Untracked: []string{filepath.Join(repo, "notes.md")},
		},
		Limits: ReviewEvidenceLimits{
			MaxCommandOutputBytes:  1024,
			MaxUntrackedFileBytes:  64,
			MaxRuleFileBytes:       128,
			MaxTotalUntrackedBytes: 256,
			MaxUntrackedFiles:      3,
			CommandTimeout:         1500 * time.Millisecond,
		},
	}
}

func assertReviewEvidenceSubstringsInOrder(t *testing.T, text string, wants []string) {
	t.Helper()

	offset := 0
	for _, want := range wants {
		index := strings.Index(text[offset:], want)
		if index < 0 {
			t.Fatalf("text after offset %d does not contain %q:\n%s", offset, want, text)
		}
		offset += index + len(want)
	}
}
