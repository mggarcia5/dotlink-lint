package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// applyPending creates a symlink for every entry that is currently pending.
// It leaves ok, blocked, conflict, missing_source, and invalid entries alone -
// apply mode only ever creates links, it never overwrites or removes
// anything that's already at the target.
//
// It returns the first error encountered, but keeps going after a failure so
// one bad entry doesn't block the rest of the manifest from being linked.
func applyPending(entries []Entry, root, home string) error {
	var firstErr error

	for _, e := range entries {
		if e.Status != StatusPending {
			continue
		}

		resolvedTarget, err := expandHome(e.Target, home)
		if err != nil {
			// Already validated once; this would only happen if home changed
			// between calls, which we treat as a bug rather than expected input.
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(resolvedTarget), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "dotlink-lint: creating %s: %v\n", filepath.Dir(resolvedTarget), err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		sourcePath := filepath.Join(root, e.Source)
		absSource, err := filepath.Abs(sourcePath)
		if err != nil {
			absSource = sourcePath
		}

		if err := os.Symlink(absSource, resolvedTarget); err != nil {
			fmt.Fprintf(os.Stderr, "dotlink-lint: linking %s: %v\n", resolvedTarget, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
	}

	return firstErr
}
