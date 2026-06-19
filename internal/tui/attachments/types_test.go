package attachments

import "testing"

func TestPrepareAppendRejectsEmptyDuplicateAndLimit(t *testing.T) {
	existing := []Attachment{{Kind: KindFile, Path: "/tmp/a.txt"}}

	if _, got := PrepareAppend(existing, Attachment{Kind: KindFile, Path: " \t "}, MaxComposerAttachments); got != AppendRejectedEmptyPath {
		t.Fatalf("PrepareAppend(empty) = %v, want %v", got, AppendRejectedEmptyPath)
	}
	if _, got := PrepareAppend(existing, Attachment{Kind: KindFile, Path: "/tmp/a.txt"}, MaxComposerAttachments); got != AppendRejectedDuplicate {
		t.Fatalf("PrepareAppend(duplicate) = %v, want %v", got, AppendRejectedDuplicate)
	}
	if _, got := PrepareAppend(existing, Attachment{Kind: KindFile, Path: "/tmp/b.txt"}, 1); got != AppendRejectedLimit {
		t.Fatalf("PrepareAppend(limit) = %v, want %v", got, AppendRejectedLimit)
	}

	att, got := PrepareAppend(existing, Attachment{Kind: KindFile, Path: " /tmp/b.txt "}, MaxComposerAttachments)
	if got != AppendAdded {
		t.Fatalf("PrepareAppend(add) = %v, want %v", got, AppendAdded)
	}
	if att.Path != "/tmp/b.txt" {
		t.Fatalf("prepared Path = %q, want trimmed path", att.Path)
	}
}
