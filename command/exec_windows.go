//go:build windows

package command

import (
	"fmt"
	"os"
	"os/exec"
)

// execBun runs the resolved bun binary and forwards its exit code. Windows
// cannot replace a running process, so we wait and mirror the result.
func execBun(bin string) error {
	cmd := exec.Command(bin, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run %s: %w", bin, err)
	}
	return nil
}

func asExitError(err error, target **exec.ExitError) bool {
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	return false
}
