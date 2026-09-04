package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	jsonOut := flag.Bool("json", false, "emit machine-readable JSON instead of the human-readable report")
	root := flag.String("root", ".", "directory that manifest source paths are resolved against")
	apply := flag.Bool("apply", false, "create symlinks for pending entries; leaves ok, blocked, conflict, and missing_source entries untouched")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <manifest-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Checks a dotfile symlink manifest against the filesystem and reports\n")
		fmt.Fprintf(os.Stderr, "what's already linked, what's pending, and what's blocked.\n\nflags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	manifestPath := flag.Arg(0)
	f, err := os.Open(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dotlink-lint: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dotlink-lint: resolving home directory: %v\n", err)
		os.Exit(1)
	}

	lines, err := readLines(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dotlink-lint: reading %s: %v\n", manifestPath, err)
		os.Exit(1)
	}

	entries := Validate(lines, *root, home)

	if *apply {
		applyErr := applyPending(entries, *root, home)
		// Re-validate so the printed report reflects what actually landed on
		// disk rather than the pre-apply plan.
		entries = Validate(lines, *root, home)
		if applyErr != nil {
			fmt.Fprintf(os.Stderr, "dotlink-lint: apply finished with errors\n")
		}
	}

	var printErr error
	if *jsonOut {
		printErr = PrintJSON(os.Stdout, entries)
	} else {
		printErr = PrintHuman(os.Stdout, entries)
	}
	if printErr != nil {
		fmt.Fprintf(os.Stderr, "dotlink-lint: %v\n", printErr)
		os.Exit(1)
	}

	if hasProblems(entries) {
		os.Exit(1)
	}
}

func hasProblems(entries []Entry) bool {
	for _, e := range entries {
		switch e.Status {
		case StatusBlocked, StatusConflict, StatusMissingSource, StatusInvalid:
			return true
		}
	}
	return false
}
