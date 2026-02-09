package benchmark

import "testing"

func TestSnapshotDoneMessage(t *testing.T) {
	if got := snapshotDoneMessage("snapshot:before"); got != "snapshot before done" {
		t.Fatalf("unexpected message: %s", got)
	}
	if got := snapshotDoneMessage("snapshot:after"); got != "snapshot after benchmark done" {
		t.Fatalf("unexpected message: %s", got)
	}
	if got := snapshotDoneMessage("snapshot:other"); got == "snapshot:other" {
		t.Fatalf("expected replacement, got %s", got)
	}
}
