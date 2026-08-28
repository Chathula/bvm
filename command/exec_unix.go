//go:build !windows

package command

import (
	"os"
	"path/filepath"
	"syscall"
)

// execProcess replaces the current process with bin and args (unix).
func execProcess(bin string, args []string) error {
	return syscall.Exec(bin, append([]string{filepath.Base(bin)}, args...), os.Environ())
}
