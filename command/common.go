package command

import (
	"errors"
	"os"
	"strings"

	"github.com/fatih/color"
)

const rcFileName = ".bvmrc"

// fail wraps an error message with red coloring for CLI output.
func fail(format string, a ...any) error {
	return errors.New(color.RedString(format, a...))
}

// resolveVersionArg returns an explicit version argument, or falls back to
// the .bvmrc file in the current directory (first non-empty, non-comment
// line) — mirroring how nvm treats .nvmrc. Returns "" when neither exists,
// leaving further fallbacks (default alias) to the caller.
func resolveVersionArg(arg string) string {
	if strings.TrimSpace(arg) != "" {
		return arg
	}

	data, err := os.ReadFile(rcFileName)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
