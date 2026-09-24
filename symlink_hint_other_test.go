//go:build !windows

package main

import (
	"errors"
	"testing"
)

func TestSymlinkPrivilegeHintNoop(t *testing.T) {
	if got := symlinkPrivilegeHint(errors.New("permission denied")); got != "" {
		t.Errorf("symlinkPrivilegeHint = %q, want empty on non-Windows platforms", got)
	}
}
