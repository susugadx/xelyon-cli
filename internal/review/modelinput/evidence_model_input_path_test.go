package modelinput

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
)

func TestBuildReviewEvidenceModelInputCWDOutsideRepo(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	outside := filepath.Join(t.TempDir(), "elsewhere")
	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		CWD:        outside,
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if input.CWDDisplay != reviewevidence.OutsideRepoPathDisplay {
		t.Fatalf("CWDDisplay = %q, want %q", input.CWDDisplay, reviewevidence.OutsideRepoPathDisplay)
	}
}

func TestBuildReviewEvidenceModelInputRedactsOutsideAbsoluteSymlinkTarget(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	outsideTarget := filepath.Join(t.TempDir(), "secret.txt")
	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []reviewevidence.ReviewUntrackedFile{{
			Path:       "outside-link",
			Symlink:    true,
			LinkTarget: outsideTarget,
		}},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if got := input.UntrackedFiles[0].LinkTarget; got != reviewevidence.OutsideRepoPathDisplay {
		t.Fatalf("LinkTarget = %q, want %q", got, reviewevidence.OutsideRepoPathDisplay)
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
	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []reviewevidence.ReviewUntrackedFile{{
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

	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []reviewevidence.ReviewUntrackedFile{
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
		reviewevidence.OutsideRepoPathDisplay,
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
			want:        reviewevidence.OutsideRepoPathDisplay,
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
			bundle := reviewevidence.ReviewEvidenceBundle{
				TargetKind: domain.TargetCurrentChanges,
				RepoRoot:   repo,
				UntrackedFiles: []reviewevidence.ReviewUntrackedFile{{
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
	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []reviewevidence.ReviewUntrackedFile{{
			Path:       "nested/link",
			Symlink:    true,
			LinkTarget: ".",
			Truncated:  true,
			SizeBytes:  int64(len("../../secret-outside")),
			ReadBytes:  int64(len(".")),
		}},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if got := input.UntrackedFiles[0].LinkTarget; got != "<truncated-link-target>" {
		t.Fatalf("LinkTarget = %q, want %q", got, "<truncated-link-target>")
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
