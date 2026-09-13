package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// fixBlocked walks every blocked entry and, after interactive confirmation,
// removes whatever is at the target and replaces it with the intended
// symlink. It only ever touches entries already reported as blocked - ok,
// pending, conflict, missing_source, and invalid entries are left exactly as
// reported, same as apply.
//
// It returns the first error encountered, but keeps going after a failure so
// one bad entry doesn't stop the rest of the manifest from being fixed.
func fixBlocked(entries []Entry, root, home string, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	var firstErr error

	for _, e := range entries {
		if e.Status != StatusBlocked {
			continue
		}

		fmt.Fprintf(out, "%s already exists and is not a symlink; replace it with a link to %s? [y/N] ", e.Target, e.Source)
		if !scanner.Scan() {
			// No more input to read; nothing left to confirm.
			fmt.Fprintln(out)
			break
		}

		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer != "y" && answer != "yes" {
			continue
		}

		resolvedTarget, err := expandHome(e.Target, home)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := os.RemoveAll(resolvedTarget); err != nil {
			fmt.Fprintf(os.Stderr, "dotlink-lint: removing %s: %v\n", resolvedTarget, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := linkEntry(e, root, home); err != nil {
			fmt.Fprintf(os.Stderr, "dotlink-lint: linking %s: %v\n", e.Target, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}
