package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "out.json")
	payload := map[string]any{"ok": true}

	if err := WriteJSON(path, payload); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded["ok"] != true {
		t.Fatalf("expected ok=true, got %v", decoded["ok"])
	}
}
