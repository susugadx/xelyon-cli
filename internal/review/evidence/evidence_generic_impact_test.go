package evidence

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestBuildReviewGenericImpactCandidatesFindsSameStemTestOrSpec(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "src", "widget.ts"), "export function widget() {}\n")
	writeTestFile(t, filepath.Join(repo, "src", "widget.test.ts"), "import { widget } from './widget'\n")
	writeTestFile(t, filepath.Join(repo, "src", "widget.spec.ts"), "describe('widget', () => {})\n")

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "src/widget.ts", ""))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleSameStemTestOrSpec, "src/widget.test.ts", "widget")
	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleSameStemTestOrSpec, "src/widget.spec.ts", "widget")
}

func TestBuildReviewGenericImpactCandidatesFindsNearbyProjectConfig(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "package.json"), `{"scripts":{"test":"vitest"}}`)
	writeTestFile(t, filepath.Join(repo, "packages", "app", "tsconfig.json"), `{"compilerOptions":{}}`)
	writeTestFile(t, filepath.Join(repo, "packages", "app", "src", "review.ts"), "export const review = true\n")

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "packages/app/src/review.ts", ""))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleNearbyProjectConfig, "package.json", "package")
	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleNearbyProjectConfig, "packages/app/tsconfig.json", "tsconfig")
}

func TestBuildReviewGenericImpactCandidatesFindsBoundedTextualReference(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "cmd", "review.ts"), `export const command = "/review"`+"\n")
	writeTestFile(t, filepath.Join(repo, "docs", "commands.md"), "Run /review before merging.\n")
	writeTestFile(t, filepath.Join(repo, "src", "usage.ts"), "const example = '/review --strict'\n")
	diff := "diff --git a/cmd/review.ts b/cmd/review.ts\n+export const command = \"/review\"\n+const config_key = true\n"

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "cmd/review.ts", diff))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleDocsReference, "docs/commands.md", "/review")
	hit := requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleTextualReference, "src/usage.ts", "/review")
	if hit.Line != 1 || !strings.Contains(hit.Snippet, "/review --strict") {
		t.Fatalf("textual reference = %#v, want line/snippet for /review reference", hit)
	}
}

func TestBuildReviewGenericImpactCandidatesExtractsStandaloneFlagDiffLines(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "docs", "options.md"), "--old\n")
	writeTestFile(t, filepath.Join(repo, "src", "usage.ts"), "const flags = '--strict --legacy'\n")
	diff := "diff --git a/docs/options.md b/docs/options.md\n--- a/docs/options.md\n+++ b/docs/options.md\n+--strict\n---legacy\n"

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "docs/options.md", diff))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleTextualReference, "src/usage.ts", "--strict")
	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleTextualReference, "src/usage.ts", "--legacy")
}

func TestReviewEvidenceBuilderCurrentChangesCollectsGenericImpactForNonGoCandidates(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "package.json"), `{"scripts":{"test":"vitest"}}`)
	writeTestFile(t, filepath.Join(repo, "src", "feature.ts"), "export const command = '/old'\n")
	writeTestFile(t, filepath.Join(repo, "src", "feature.test.ts"), "test('feature', () => {})\n")
	writeTestFile(t, filepath.Join(repo, "docs", "commands.md"), "Run /review before merging.\n")
	runGit(t, repo, "add", "package.json", "src", "docs")
	runGit(t, repo, "commit", "-m", "add generic impact fixtures")

	writeTestFile(t, filepath.Join(repo, "src", "feature.ts"), "export const command = '/review'\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	requireReviewGenericImpactCandidate(t, bundle.GenericImpactCandidates, ReviewGenericImpactRoleSameStemTestOrSpec, "src/feature.test.ts", "feature")
	requireReviewGenericImpactCandidate(t, bundle.GenericImpactCandidates, ReviewGenericImpactRoleNearbyProjectConfig, "package.json", "package")
	requireReviewGenericImpactCandidate(t, bundle.GenericImpactCandidates, ReviewGenericImpactRoleDocsReference, "docs/commands.md", "/review")
}

func TestReviewEvidenceBuilderCurrentChangesCollectsGenericImpactFromUntrackedSnapshot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "docs", "commands.md"), "Run /deploy before launch.\n")
	writeTestFile(t, filepath.Join(repo, "src", "usage.ts"), "const option = '--strict'\n")
	runGit(t, repo, "add", "docs", "src")
	runGit(t, repo, "commit", "-m", "add untracked generic impact fixtures")

	writeTestFile(t, filepath.Join(repo, "docs", "new-command.md"), "Run /deploy with:\n--strict\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	requireReviewGenericImpactCandidate(t, bundle.GenericImpactCandidates, ReviewGenericImpactRoleDocsReference, "docs/commands.md", "/deploy")
	requireReviewGenericImpactCandidate(t, bundle.GenericImpactCandidates, ReviewGenericImpactRoleTextualReference, "src/usage.ts", "--strict")
}

func TestBuildReviewGenericImpactCandidatesSkipsSensitiveUntrackedTokenSource(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "src", "safe.ts"), "const safe = 'API_KEY reference'\n")
	bundle := newGenericImpactBundleForTest(repo, "secrets/new.env", "")
	bundle.UntrackedFiles = []ReviewUntrackedFile{
		{
			Path:     "secrets/new.env",
			Snapshot: "API_KEY=super-secret-env\n",
		},
	}
	bundle.Inventory.Untracked = []string{"secrets/new.env"}

	candidates := BuildReviewGenericImpactCandidates(bundle)

	for _, candidate := range candidates.Candidates {
		if candidate.Token == "API_KEY" || strings.Contains(candidate.Snippet, "super-secret-env") {
			t.Fatalf("generic impact candidate should not use sensitive untracked snapshot as token source: %#v", candidate)
		}
	}
}

func TestBuildReviewGenericImpactCandidatesExcludesLargeGeneratedAndDebugDirs(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "cmd", "review.ts"), `export const command = "/review"`+"\n")
	writeTestFile(t, filepath.Join(repo, "src", "live.ts"), "const live = '/review'\n")
	writeTestFile(t, filepath.Join(repo, "node_modules", "pkg", "ignored.ts"), "const ignored = '/review'\n")
	writeTestFile(t, filepath.Join(repo, "vendor", "pkg", "ignored.ts"), "const ignored = '/review'\n")
	writeTestFile(t, filepath.Join(repo, "dist", "ignored.ts"), "const ignored = '/review'\n")
	writeTestFile(t, filepath.Join(repo, "build", "ignored.ts"), "const ignored = '/review'\n")
	writeTestFile(t, filepath.Join(repo, "coverage", "ignored.ts"), "const ignored = '/review'\n")
	writeTestFile(t, filepath.Join(repo, ".xelyon", "review-runs", "20260101T000000.000000000Z", "evidence.md"), "/review\n")
	diff := "+export const command = \"/review\"\n"

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "cmd/review.ts", diff))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleTextualReference, "src/live.ts", "/review")
	for _, candidate := range candidates.Candidates {
		for _, excluded := range []string{"node_modules/", "vendor/", "dist/", "build/", "coverage/", ".xelyon/review-runs/"} {
			if strings.Contains(candidate.Path, excluded) {
				t.Fatalf("candidate %#v should exclude %q", candidate, excluded)
			}
		}
	}
}

func TestBuildReviewGenericImpactCandidatesSkipsSensitiveSearchFiles(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "cmd", "review.ts"), "export const API_KEY = 'placeholder'\n")
	writeTestFile(t, filepath.Join(repo, "src", "safe.ts"), "const safe = 'API_KEY reference'\n")
	writeTestFile(t, filepath.Join(repo, ".env"), "API_KEY=super-secret-env\n")
	writeTestFile(t, filepath.Join(repo, "config", "credentials.json"), `{"API_KEY":"super-secret-json"}`+"\n")
	writeTestFile(t, filepath.Join(repo, "credentials", "prod.json"), `{"API_KEY":"super-secret-dir-json"}`+"\n")
	writeTestFile(t, filepath.Join(repo, "secrets", "prod.json"), `{"API_KEY":"super-secret-secrets-dir"}`+"\n")
	writeTestFile(t, filepath.Join(repo, ".aws", "credentials"), "API_KEY=super-secret-aws\n")
	diff := "+export const API_KEY = 'placeholder'\n"

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "cmd/review.ts", diff))

	requireReviewGenericImpactCandidate(t, candidates, ReviewGenericImpactRoleTextualReference, "src/safe.ts", "API_KEY")
	for _, candidate := range candidates.Candidates {
		switch candidate.Path {
		case ".env", "config/credentials.json", "credentials/prod.json", "secrets/prod.json", ".aws/credentials":
			t.Fatalf("generic impact candidate should not include sensitive path: %#v", candidate)
		}
		for _, secret := range []string{"super-secret-env", "super-secret-json", "super-secret-dir-json", "super-secret-secrets-dir", "super-secret-aws"} {
			if strings.Contains(candidate.Snippet, secret) {
				t.Fatalf("generic impact candidate snippet leaked %q: %#v", secret, candidate)
			}
		}
	}
}

func TestBuildReviewGenericImpactCandidatesTruncatesByTokenHitBudgetAcrossFiles(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "cmd", "review.ts"), `export const command = "/review"`+"\n")
	for i := 0; i < reviewGenericImpactMaxHitsPerToken+2; i++ {
		writeTestFile(t, filepath.Join(repo, "src", "ref"+strconv.Itoa(i)+".ts"), "const ref = '/review'\n")
	}
	diff := "+export const command = \"/review\"\n"

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "cmd/review.ts", diff))

	if !candidates.Truncated {
		t.Fatalf("GenericImpactCandidates.Truncated = false, want true")
	}
	count := 0
	for _, candidate := range candidates.Candidates {
		if candidate.Role == ReviewGenericImpactRoleTextualReference && candidate.Token == "/review" {
			count++
		}
	}
	if count != reviewGenericImpactMaxHitsPerToken {
		t.Fatalf("textual /review candidate count = %d, want %d", count, reviewGenericImpactMaxHitsPerToken)
	}
}

func TestBuildReviewGenericImpactCandidatesTruncatesByCandidateBudget(t *testing.T) {
	repo := t.TempDir()
	writeTestFile(t, filepath.Join(repo, "src", "widget.ts"), "export function widget() {}\n")
	for i := 0; i < reviewGenericImpactMaxCandidatesPerRole+2; i++ {
		writeTestFile(t, filepath.Join(repo, "tests", "area"+strconv.Itoa(i), "widget.test.ts"), "test('widget', () => {})\n")
	}

	candidates := BuildReviewGenericImpactCandidates(newGenericImpactBundleForTest(repo, "src/widget.ts", ""))

	if !candidates.Truncated {
		t.Fatalf("GenericImpactCandidates.Truncated = false, want true")
	}
	count := 0
	for _, candidate := range candidates.Candidates {
		if candidate.Role == ReviewGenericImpactRoleSameStemTestOrSpec {
			count++
		}
	}
	if count != reviewGenericImpactMaxCandidatesPerRole {
		t.Fatalf("same stem candidate count = %d, want %d", count, reviewGenericImpactMaxCandidatesPerRole)
	}
}

func newGenericImpactBundleForTest(repoRoot, changedPath, diff string) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   repoRoot,
		CWD:        repoRoot,
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     changedPath,
				Status:   "M",
				Unstaged: true,
			},
		},
		Diffs: []ReviewDiffEvidence{
			{
				Source: "unstaged",
				Diff:   diff,
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{changedPath},
		},
		Limits: DefaultReviewEvidenceLimits(),
	}
}

func requireReviewGenericImpactCandidate(t *testing.T, candidates ReviewGenericImpactCandidates, role, path, token string) ReviewGenericImpactCandidate {
	t.Helper()
	for _, candidate := range candidates.Candidates {
		if candidate.Role == role && candidate.Path == path && candidate.Token == token {
			return candidate
		}
	}
	t.Fatalf("generic impact candidates = %#v, want role=%q path=%q token=%q", candidates.Candidates, role, path, token)
	return ReviewGenericImpactCandidate{}
}
