package main

import (
	"encoding/json"
	"fmt"
	"io"
)

// PrintJSON writes the full entry list as an indented JSON array, suitable
// for feeding into another tool (jq, a setup script, CI).
func PrintJSON(w io.Writer, entries []Entry) error {
	if entries == nil {
		entries = []Entry{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

// PrintHuman writes one line per entry plus a summary line, meant to be read
// in a terminal.
func PrintHuman(w io.Writer, entries []Entry) error {
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, "no entries in manifest")
		return err
	}

	counts := map[Status]int{}
	for _, e := range entries {
		counts[e.Status]++

		line := fmt.Sprintf("%4d  %-14s  %s -> %s", e.Line, e.Status, e.Source, e.Target)
		if e.Detail != "" {
			line += "  (" + e.Detail + ")"
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\n%d entries: %d ok, %d pending, %d blocked, %d conflict, %d missing source, %d invalid\n",
		len(entries),
		counts[StatusOK],
		counts[StatusPending],
		counts[StatusBlocked],
		counts[StatusConflict],
		counts[StatusMissingSource],
		counts[StatusInvalid],
	)
	return err
}
