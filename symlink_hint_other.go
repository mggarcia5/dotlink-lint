//go:build !windows

package main

// symlinkPrivilegeHint has nothing to add on platforms where creating a
// symlink doesn't require a special privilege. See symlink_hint_windows.go.
func symlinkPrivilegeHint(err error) string {
	return ""
}
