package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectCreates(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	res, err := InjectInto(p)
	if err != nil {
		t.Fatal(err)
	}
	if res != Created {
		t.Errorf("got %v want Created", res)
	}
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), startMarker) || !strings.Contains(string(data), endMarker) {
		t.Error("markers missing")
	}
	if !strings.Contains(string(data), "shutup run") {
		t.Error("block content missing")
	}
}

func TestInjectAppendsPreservingContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	os.WriteFile(p, []byte("# My App\n\nExisting docs.\n"), 0o644)
	res, err := InjectInto(p)
	if err != nil {
		t.Fatal(err)
	}
	if res != Added {
		t.Errorf("got %v want Added", res)
	}
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "Existing docs.") {
		t.Error("existing content lost")
	}
	if !strings.Contains(string(data), startMarker) {
		t.Error("block not added")
	}
}

func TestInjectIdempotent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	if _, err := InjectInto(p); err != nil {
		t.Fatal(err)
	}
	res, err := InjectInto(p)
	if err != nil {
		t.Fatal(err)
	}
	if res != Unchanged {
		t.Errorf("second inject: got %v want Unchanged", res)
	}
	// only one block
	data, _ := os.ReadFile(p)
	if n := strings.Count(string(data), startMarker); n != 1 {
		t.Errorf("expected exactly 1 block, got %d", n)
	}
}

func TestInjectUpdatesStaleBlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	stale := "# Title\n\n" + startMarker + "\nOLD INSTRUCTIONS\n" + endMarker + "\n\nmore docs\n"
	if err := os.WriteFile(p, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InjectInto(p)
	if err != nil {
		t.Fatal(err)
	}
	if res != Updated {
		t.Errorf("got %v want Updated", res)
	}
	data, _ := os.ReadFile(p)
	out := string(data)
	if strings.Contains(out, "OLD INSTRUCTIONS") {
		t.Error("stale block content should be replaced")
	}
	if !strings.Contains(out, "more docs") || !strings.Contains(out, "# Title") {
		t.Error("surrounding content should be preserved")
	}
	if strings.Count(out, startMarker) != 1 {
		t.Error("should still have exactly one block")
	}
}

func TestInjectAppendsWhenNoTrailingNewline(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(p, []byte("# Title (no newline)"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := InjectInto(p)
	if err != nil {
		t.Fatal(err)
	}
	if res != Added {
		t.Errorf("got %v want Added", res)
	}
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), "# Title (no newline)") || !strings.Contains(string(data), startMarker) {
		t.Error("should preserve content and append block")
	}
}

func TestRemoveFrom(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	os.WriteFile(p, []byte("# My App\n\nExisting docs.\n"), 0o644)
	_, _ = InjectInto(p)

	removed, err := RemoveFrom(p)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Error("expected removed=true")
	}
	data, _ := os.ReadFile(p)
	out := string(data)
	if strings.Contains(out, startMarker) || strings.Contains(out, endMarker) {
		t.Error("block not fully removed")
	}
	if !strings.Contains(out, "Existing docs.") {
		t.Error("original content should survive removal")
	}
}

func TestRemoveFromNoBlock(t *testing.T) {
	p := filepath.Join(t.TempDir(), "CLAUDE.md")
	os.WriteFile(p, []byte("no block here\n"), 0o644)
	removed, err := RemoveFrom(p)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Error("expected removed=false when no block present")
	}
}

func TestRemoveFromMissingFile(t *testing.T) {
	removed, err := RemoveFrom(filepath.Join(t.TempDir(), "nope.md"))
	if err != nil {
		t.Errorf("missing file should not error: %v", err)
	}
	if removed {
		t.Error("expected removed=false for missing file")
	}
}
