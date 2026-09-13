package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixBlockedConfirmed(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	sourcePath := filepath.Join(root, "blocked-file")
	if err := os.WriteFile(sourcePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	targetPath := filepath.Join(home, "blocked-target")
	if err := os.WriteFile(targetPath, []byte("old contents"), 0o644); err != nil {
		t.Fatalf("writing target: %v", err)
	}

	entries := []Entry{{Line: 1, Status: StatusBlocked, Source: "blocked-file", Target: targetPath}}

	var out bytes.Buffer
	if err := fixBlocked(entries, root, home, strings.NewReader("y\n"), &out); err != nil {
		t.Fatalf("fixBlocked: %v", err)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target %s is not a symlink after fix", targetPath)
	}

	dest, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if dest != absSource {
		t.Errorf("target links to %q, want %q", dest, absSource)
	}
}

func TestFixBlockedDeclined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "blocked-file"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	targetPath := filepath.Join(home, "blocked-target")
	if err := os.WriteFile(targetPath, []byte("old contents"), 0o644); err != nil {
		t.Fatalf("writing target: %v", err)
	}

	entries := []Entry{{Line: 1, Status: StatusBlocked, Source: "blocked-file", Target: targetPath}}

	var out bytes.Buffer
	if err := fixBlocked(entries, root, home, strings.NewReader("n\n"), &out); err != nil {
		t.Fatalf("fixBlocked: %v", err)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("target %s was replaced despite declined confirmation", targetPath)
	}
	contents, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(contents) != "old contents" {
		t.Errorf("target contents = %q, want unchanged", contents)
	}
}

func TestFixBlockedSkipsOtherStatuses(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	entries := []Entry{
		{Line: 1, Status: StatusOK, Source: "a", Target: filepath.Join(home, "a")},
		{Line: 2, Status: StatusPending, Source: "b", Target: filepath.Join(home, "b")},
	}

	var out bytes.Buffer
	// No input available; if fixBlocked tried to prompt for either of these
	// entries it would have nothing to read and the test would still pass,
	// so also check the prompt text never appears.
	if err := fixBlocked(entries, root, home, strings.NewReader(""), &out); err != nil {
		t.Fatalf("fixBlocked: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no prompts for non-blocked entries, got %q", out.String())
	}
}

func TestFixBlockedReplacesDirectory(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	sourcePath := filepath.Join(root, "nvim")
	if err := os.WriteFile(sourcePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	targetPath := filepath.Join(home, "config-nvim")
	if err := os.MkdirAll(filepath.Join(targetPath, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetPath, "nested", "init.vim"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing nested file: %v", err)
	}

	entries := []Entry{{Line: 1, Status: StatusBlocked, Source: "nvim", Target: targetPath}}

	var out bytes.Buffer
	if err := fixBlocked(entries, root, home, strings.NewReader("yes\n"), &out); err != nil {
		t.Fatalf("fixBlocked: %v", err)
	}

	info, err := os.Lstat(targetPath)
	if err != nil {
		t.Fatalf("lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("target %s is not a symlink after fix", targetPath)
	}
}
