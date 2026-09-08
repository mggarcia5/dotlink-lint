package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Status is the result of checking one manifest entry against the filesystem.
type Status string

const (
	StatusOK           Status = "ok"             // target is already a symlink to source
	StatusPending      Status = "pending"         // target does not exist yet, link can be created
	StatusBlocked      Status = "blocked"         // target exists and is not a symlink
	StatusConflict     Status = "conflict"        // target is a symlink to something else, or is claimed twice
	StatusMissingSource Status = "missing_source" // source path does not exist under root
	StatusInvalid      Status = "invalid"         // line could not be parsed
)

// Entry is one line of the manifest after validation.
type Entry struct {
	Line   int    `json:"line"`
	Status Status `json:"status"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// rawLine is a non-blank, non-comment line read from the manifest, before validation.
type rawLine struct {
	Number int
	Text   string
}

func readLines(r io.Reader) ([]rawLine, error) {
	var out []rawLine
	scanner := bufio.NewScanner(r)
	n := 0
	for scanner.Scan() {
		n++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		out = append(out, rawLine{Number: n, Text: text})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Validate resolves and checks every manifest line against the filesystem.
// root is the directory source paths are resolved against; home is used to
// expand a leading ~ in target paths.
func Validate(lines []rawLine, root, home string) []Entry {
	entries := make([]Entry, 0, len(lines))
	seenTargets := make(map[string]int) // resolved target -> line number of first claim

	for _, line := range lines {
		source, target, ok := splitArrow(line.Text)
		if !ok {
			entries = append(entries, Entry{
				Line:   line.Number,
				Status: StatusInvalid,
				Detail: `expected "source -> target"`,
			})
			continue
		}

		if isGlobPattern(source) {
			entries = append(entries, expandGlob(line.Number, source, target, root, home, seenTargets)...)
			continue
		}

		entries = append(entries, resolveEntry(line.Number, source, target, root, home, seenTargets))
	}

	return entries
}

// resolveEntry validates one concrete source/target pair (already expanded
// out of any glob) and, if it's not a duplicate target claim, checks it
// against the filesystem.
func resolveEntry(lineNo int, source, target, root, home string, seenTargets map[string]int) Entry {
	resolvedTarget, err := expandHome(target, home)
	if err != nil {
		return Entry{
			Line:   lineNo,
			Source: source,
			Target: target,
			Status: StatusInvalid,
			Detail: err.Error(),
		}
	}
	if !filepath.IsAbs(resolvedTarget) {
		return Entry{
			Line:   lineNo,
			Source: source,
			Target: target,
			Status: StatusInvalid,
			Detail: "target must be absolute or start with ~",
		}
	}

	if first, dup := seenTargets[resolvedTarget]; dup {
		return Entry{
			Line:   lineNo,
			Source: source,
			Target: target,
			Status: StatusConflict,
			Detail: fmt.Sprintf("target also claimed by line %d", first),
		}
	}
	seenTargets[resolvedTarget] = lineNo

	return checkEntry(lineNo, source, target, root, resolvedTarget)
}

// isGlobPattern reports whether source contains glob metacharacters and
// should be expanded against the filesystem rather than treated literally.
func isGlobPattern(source string) bool {
	return strings.ContainsAny(source, "*?[")
}

// expandGlob resolves a glob source pattern into one Entry per matching file
// under root. Since a single pattern can match many files, target names a
// directory they all land in rather than a single path, so it must end in a
// path separator. Each match keeps the manifest line's number, and is fed
// through the normal duplicate-target and filesystem checks via
// resolveEntry.
func expandGlob(lineNo int, pattern, target, root, home string, seenTargets map[string]int) []Entry {
	if !strings.HasSuffix(target, "/") {
		return []Entry{{
			Line:   lineNo,
			Source: pattern,
			Target: target,
			Status: StatusInvalid,
			Detail: "glob source requires a directory target ending in /",
		}}
	}

	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		return []Entry{{
			Line:   lineNo,
			Source: pattern,
			Target: target,
			Status: StatusInvalid,
			Detail: fmt.Sprintf("bad glob pattern: %v", err),
		}}
	}
	if len(matches) == 0 {
		return []Entry{{
			Line:   lineNo,
			Source: pattern,
			Target: target,
			Status: StatusMissingSource,
			Detail: "no files matched pattern",
		}}
	}

	entries := make([]Entry, 0, len(matches))
	for _, match := range matches {
		relSource, err := filepath.Rel(root, match)
		if err != nil {
			relSource = match
		}
		entryTarget := target + filepath.Base(match)
		entries = append(entries, resolveEntry(lineNo, relSource, entryTarget, root, home, seenTargets))
	}
	return entries
}

func splitArrow(text string) (source, target string, ok bool) {
	parts := strings.Split(text, "->")
	if len(parts) != 2 {
		return "", "", false
	}
	source = strings.TrimSpace(parts[0])
	target = strings.TrimSpace(parts[1])
	if source == "" || target == "" {
		return "", "", false
	}
	return source, target, true
}

func expandHome(path, home string) (string, error) {
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}
	if strings.HasPrefix(path, "~") {
		return "", fmt.Errorf("unsupported ~user expansion in %q", path)
	}
	return path, nil
}

// checkEntry compares one manifest line against the real filesystem state.
func checkEntry(lineNo int, source, target, root, resolvedTarget string) Entry {
	e := Entry{Line: lineNo, Source: source, Target: target}

	sourcePath := filepath.Join(root, source)
	if _, err := os.Lstat(sourcePath); err != nil {
		e.Status = StatusMissingSource
		e.Detail = fmt.Sprintf("%s: no such file or directory", sourcePath)
		return e
	}

	info, err := os.Lstat(resolvedTarget)
	if err != nil {
		e.Status = StatusPending
		return e
	}

	if info.Mode()&os.ModeSymlink == 0 {
		e.Status = StatusBlocked
		e.Detail = fmt.Sprintf("%s already exists and is not a symlink", resolvedTarget)
		return e
	}

	dest, err := os.Readlink(resolvedTarget)
	if err != nil {
		e.Status = StatusConflict
		e.Detail = fmt.Sprintf("could not read existing symlink: %v", err)
		return e
	}

	absSource, err := filepath.Abs(sourcePath)
	if err != nil {
		absSource = sourcePath
	}
	absDest := dest
	if !filepath.IsAbs(absDest) {
		absDest = filepath.Join(filepath.Dir(resolvedTarget), dest)
	}

	if filepath.Clean(absDest) == filepath.Clean(absSource) {
		e.Status = StatusOK
	} else {
		e.Status = StatusConflict
		e.Detail = fmt.Sprintf("points to %s instead", dest)
	}
	return e
}
