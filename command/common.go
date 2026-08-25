package command

import (
	"errors"

	"github.com/fatih/color"
)

// fail wraps an error message with red coloring for CLI output.
func fail(format string, a ...any) error {
	return errors.New(color.RedString(format, a...))
}
