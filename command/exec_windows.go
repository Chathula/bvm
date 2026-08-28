//go:build windows

package command

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// execProcess runs bin with args and forwards its exit code. Windows cannot
// replace a running process, so we wait and mirror the result.
func execProcess(bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run %s: %w", bin, err)
	}
	return nil
}
