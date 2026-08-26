//go:build !windows

package command

import (
	"os"
	"path/filepath"
	"syscall"
)

// execBun replaces the current process with the resolved bun binary (unix).
func execBun(bin string) error {
	argv0 := filepath.Base(bin)
	return syscall.Exec(bin, append([]string{argv0}, os.Args[1:]...), os.Environ())
}
