//go:build windows

package main

import (
	"errors"
	"syscall"
)

// symlinkPrivilegeHint checks whether err is the error CreateSymbolicLink
// returns when the caller lacks SeCreateSymbolicLinkPrivilege - i.e. the
// process isn't elevated and Developer Mode isn't enabled - and if so
// returns a hint to append to the error message. Without this, the raw
// error ("A required privilege is not held by the client.") gives no clue
// that it's a Windows symlink permission issue rather than a filesystem
// problem with the target itself.
func symlinkPrivilegeHint(err error) string {
	var errno syscall.Errno
	if errors.As(err, &errno) && errno == 1314 { // ERROR_PRIVILEGE_NOT_HELD
		return "creating symlinks on Windows requires Developer Mode or an elevated (administrator) prompt"
	}
	return ""
}
