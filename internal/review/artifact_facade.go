package review

import reviewartifact "github.com/susugadx/xelyon-cli/internal/review/artifact"

// ReviewRunArtifactWriter は /review runner の debug artifact 保存境界を表す。
// nil の場合、runner は artifact を完全に無効化する。
type ReviewRunArtifactWriter = reviewartifact.ReviewRunArtifactWriter

// DirectoryReviewRunArtifactWriter は 1 回の /review 実行 artifact を
// 指定 directory へ保存する writer。
type DirectoryReviewRunArtifactWriter = reviewartifact.DirectoryReviewRunArtifactWriter

// BufferedReviewRunArtifactWriter は artifact を保存先 I/O なしで一時保持する writer。
type BufferedReviewRunArtifactWriter = reviewartifact.BufferedReviewRunArtifactWriter

// NewBufferedReviewRunArtifactWriter は flush 可能な in-memory artifact writer を返す。
func NewBufferedReviewRunArtifactWriter() *BufferedReviewRunArtifactWriter {
	return reviewartifact.NewBufferedReviewRunArtifactWriter()
}

// NewReviewRunDirectoryArtifactWriter は保存先 directory を作成して writer を返す。
func NewReviewRunDirectoryArtifactWriter(dir string) (*DirectoryReviewRunArtifactWriter, error) {
	return reviewartifact.NewReviewRunDirectoryArtifactWriter(dir)
}

// NewReviewRunRepoArtifactWriter は repo-local な /review artifact 保存先を作成する。
func NewReviewRunRepoArtifactWriter(repoRoot, runID string) (*DirectoryReviewRunArtifactWriter, error) {
	return reviewartifact.NewReviewRunRepoArtifactWriter(repoRoot, runID)
}
