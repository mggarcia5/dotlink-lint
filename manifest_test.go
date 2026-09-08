package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLines(t *testing.T) {
	input := strings.NewReader(strings.Join([]string{
		"# a comment",
		"",
		"  ",
		"zsh/zshrc -> ~/.zshrc",
		"# another comment",
		"git/gitconfig -> ~/.gitconfig",
	}, "\n"))

	lines, err := readLines(input)
	if err != nil {
		t.Fatalf("readLines: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %+v", len(lines), lines)
	}
	if lines[0].Number != 4 || lines[0].Text != "zsh/zshrc -> ~/.zshrc" {
		t.Errorf("line 0 = %+v, want number 4, text %q", lines[0], "zsh/zshrc -> ~/.zshrc")
	}
	if lines[1].Number != 6 || lines[1].Text != "git/gitconfig -> ~/.gitconfig" {
		t.Errorf("line 1 = %+v, want number 6, text %q", lines[1], "git/gitconfig -> ~/.gitconfig")
	}
}

func TestSplitArrow(t *testing.T) {
	cases := []struct {
		text       string
		wantSource string
		wantTarget string
		wantOK     bool
	}{
		{"src -> dst", "src", "dst", true},
		{"  src  ->  dst  ", "src", "dst", true},
		{"no arrow here", "", "", false},
		{"a -> b -> c", "", "", false},
		{" -> dst", "", "", false},
		{"src -> ", "", "", false},
	}

	for _, c := range cases {
		source, target, ok := splitArrow(c.text)
		if ok != c.wantOK || source != c.wantSource || target != c.wantTarget {
			t.Errorf("splitArrow(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.text, source, target, ok, c.wantSource, c.wantTarget, c.wantOK)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := "/home/me"

	cases := []struct {
		path    string
		want    string
		wantErr bool
	}{
		{"~", "/home/me", false},
		{"~/.zshrc", "/home/me/.zshrc", false},
		{"/etc/hosts", "/etc/hosts", false},
		{"~other/.zshrc", "", true},
	}

	for _, c := range cases {
		got, err := expandHome(c.path, home)
		if c.wantErr {
			if err == nil {
				t.Errorf("expandHome(%q) = %q, want error", c.path, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("expandHome(%q) unexpected error: %v", c.path, err)
			continue
		}
		if got != c.want {
			t.Errorf("expandHome(%q) = %q, want %q", c.path, got, c.want)
		}
	}
}

func TestValidateStatuses(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeFile := func(path string) {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	absOf := func(path string) string {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("abs of %s: %v", path, err)
		}
		return abs
	}

	// line 1: ok - target already links to source
	okSource := filepath.Join(root, "ok-file")
	writeFile(okSource)
	okTarget := filepath.Join(home, "ok-target")
	if err := os.Symlink(absOf(okSource), okTarget); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// line 2: pending - source exists, target does not
	writeFile(filepath.Join(root, "pending-file"))
	pendingTarget := filepath.Join(home, "pending-target")

	// line 3: blocked - target exists and is a real file
	writeFile(filepath.Join(root, "blocked-file"))
	blockedTarget := filepath.Join(home, "blocked-target")
	writeFile(blockedTarget)

	// line 4: conflict - target links elsewhere
	writeFile(filepath.Join(root, "conflict-file"))
	otherFile := filepath.Join(root, "other-file")
	writeFile(otherFile)
	conflictTarget := filepath.Join(home, "conflict-target")
	if err := os.Symlink(absOf(otherFile), conflictTarget); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	// line 5: missing_source - source does not exist under root
	missingTarget := filepath.Join(home, "missing-target")

	// lines 7 & 8: conflict via duplicate target claim
	writeFile(filepath.Join(root, "dup-file"))
	dupTarget := filepath.Join(home, "dup-target")

	lines := []rawLine{
		{Number: 1, Text: fmt.Sprintf("ok-file -> %s", okTarget)},
		{Number: 2, Text: fmt.Sprintf("pending-file -> %s", pendingTarget)},
		{Number: 3, Text: fmt.Sprintf("blocked-file -> %s", blockedTarget)},
		{Number: 4, Text: fmt.Sprintf("conflict-file -> %s", conflictTarget)},
		{Number: 5, Text: fmt.Sprintf("missing-file -> %s", missingTarget)},
		{Number: 6, Text: "not a valid manifest line"},
		{Number: 7, Text: fmt.Sprintf("dup-file -> %s", dupTarget)},
		{Number: 8, Text: fmt.Sprintf("dup-file2 -> %s", dupTarget)},
	}

	entries := Validate(lines, root, home)

	got := make(map[int]Status, len(entries))
	for _, e := range entries {
		got[e.Line] = e.Status
	}

	want := map[int]Status{
		1: StatusOK,
		2: StatusPending,
		3: StatusBlocked,
		4: StatusConflict,
		5: StatusMissingSource,
		6: StatusInvalid,
		7: StatusPending,
		8: StatusConflict,
	}

	for line, wantStatus := range want {
		if got[line] != wantStatus {
			t.Errorf("line %d status = %q, want %q", line, got[line], wantStatus)
		}
	}
}

func TestValidateRelativeTargetIsInvalid(t *testing.T) {
	root := t.TempDir()
	lines := []rawLine{
		{Number: 1, Text: "some-file -> relative/path"},
	}

	entries := Validate(lines, root, "/home/me")
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Status != StatusInvalid {
		t.Errorf("status = %q, want %q", entries[0].Status, StatusInvalid)
	}
}
