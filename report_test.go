package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintJSONRoundTrip(t *testing.T) {
	entries := []Entry{
		{Line: 3, Status: StatusOK, Source: "zshrc", Target: "~/.zshrc"},
		{Line: 4, Status: StatusConflict, Source: "vimrc", Target: "~/.vimrc", Detail: "points to /elsewhere instead"},
	}

	var buf bytes.Buffer
	if err := PrintJSON(&buf, entries); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}

	var got []Entry
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(got), len(entries))
	}
	for i := range entries {
		if got[i] != entries[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], entries[i])
		}
	}
}

func TestPrintJSONNilIsEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSON(&buf, nil); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "[]" {
		t.Errorf("output = %q, want []", got)
	}
}

func TestPrintJSONOmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintJSON(&buf, []Entry{{Line: 1, Status: StatusInvalid}}); err != nil {
		t.Fatalf("PrintJSON: %v", err)
	}
	out := buf.String()
	for _, key := range []string{`"source"`, `"target"`, `"detail"`} {
		if strings.Contains(out, key) {
			t.Errorf("output contains %s for an entry that has none:\n%s", key, out)
		}
	}
}

func TestPrintHumanEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintHuman(&buf, nil); err != nil {
		t.Fatalf("PrintHuman: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "no entries in manifest" {
		t.Errorf("output = %q", got)
	}
}

func TestPrintHumanLinesAndSummary(t *testing.T) {
	entries := []Entry{
		{Line: 1, Status: StatusOK, Source: "a", Target: "~/a"},
		{Line: 2, Status: StatusPending, Source: "b", Target: "~/b"},
		{Line: 3, Status: StatusPending, Source: "c", Target: "~/c"},
		{Line: 4, Status: StatusBlocked, Source: "d", Target: "~/d", Detail: "already exists"},
		{Line: 5, Status: StatusMissingSource, Source: "e", Target: "~/e"},
		{Line: 6, Status: StatusInvalid, Detail: "bad line"},
	}

	var buf bytes.Buffer
	if err := PrintHuman(&buf, entries); err != nil {
		t.Fatalf("PrintHuman: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "d -> ~/d  (already exists)") {
		t.Errorf("detail not rendered after the entry:\n%s", out)
	}
	if strings.Contains(out, "a -> ~/a  (") {
		t.Errorf("entry without detail got a parenthesized suffix:\n%s", out)
	}

	want := "6 entries: 1 ok, 2 pending, 1 blocked, 0 conflict, 1 missing source, 1 invalid"
	if !strings.Contains(out, want) {
		t.Errorf("summary line missing, want %q in:\n%s", want, out)
	}
}
