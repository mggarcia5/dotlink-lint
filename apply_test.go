package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPendingCreatesLinkAndParents(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	sourcePath := filepath.Join(root, "bashrc")
	if err := os.WriteFile(sourcePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}
	targetPath := filepath.Join(home, ".config", "deep", "bashrc")

	entries := []Entry{{Line: 1, Status: StatusPending, Source: "bashrc", Target: targetPath}}
	if err := applyPending(entries, root, home); err != nil {
		t.Fatalf("applyPending: %v", err)
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

func TestApplyPendingExpandsHome(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "vimrc"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source: %v", err)
	}

	entries := []Entry{{Line: 1, Status: StatusPending, Source: "vimrc", Target: "~/.vimrc"}}
	if err := applyPending(entries, root, home); err != nil {
		t.Fatalf("applyPending: %v", err)
	}

	if _, err := os.Readlink(filepath.Join(home, ".vimrc")); err != nil {
		t.Errorf("expected symlink at ~/.vimrc: %v", err)
	}
}

func TestApplyPendingSkipsOtherStatuses(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	statuses := []Status{StatusOK, StatusBlocked, StatusConflict, StatusMissingSource, StatusInvalid}
	var entries []Entry
	for i, s := range statuses {
		entries = append(entries, Entry{
			Line:   i + 1,
			Status: s,
			Source: "src",
			Target: filepath.Join(home, string(s)),
		})
	}

	if err := applyPending(entries, root, home); err != nil {
		t.Fatalf("applyPending: %v", err)
	}

	for _, e := range entries {
		if _, err := os.Lstat(e.Target); !os.IsNotExist(err) {
			t.Errorf("%s entry created something at %s (lstat err: %v)", e.Status, e.Target, err)
		}
	}
}

func TestApplyPendingContinuesAfterFailure(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "a"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing source b: %v", err)
	}

	// A regular file where a parent directory is needed makes the first
	// entry fail at MkdirAll.
	notADir := filepath.Join(home, "file")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	goodTarget := filepath.Join(home, "good")

	entries := []Entry{
		{Line: 1, Status: StatusPending, Source: "a", Target: filepath.Join(notADir, "child")},
		{Line: 2, Status: StatusPending, Source: "b", Target: goodTarget},
	}

	if err := applyPending(entries, root, home); err == nil {
		t.Fatal("expected an error from the first entry, got nil")
	}
	if _, err := os.Readlink(goodTarget); err != nil {
		t.Errorf("second entry was not linked after the first failed: %v", err)
	}
}
