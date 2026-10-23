package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chathula/bvm/util"
)

// execProcessFn is a var so tests can stub process replacement.
var execProcessFn = execProcess

// versionDir is a var so tests can inject lookup failures portably.
var versionDir = util.VersionDir

// exitWithError is a var so tests can capture the fatal exit path.
var exitWithError = func(msg string) {
	fmt.Fprintln(os.Stderr, "bvm:", msg)
	os.Exit(1)
}

// RunShim implements the `bun` and `bunx` shims: bvm resolves which
// installed version applies to the current directory (nearest .bvmrc walking
// up, else the default alias) and hands the arguments over to that binary.
// When invoked as bunx, the equivalent `bun x` form is used so the selected
// version runs the package runner too.
func RunShim(alias string) {
	if err := runShim(alias); err != nil {
		exitWithError(err.Error())
	}
}

func runShim(alias string) error {
	version, _, err := util.ResolveVersion()
	if err != nil {
		return err
	}

	dir, err := versionDir(version)
	if err != nil {
		return err
	}
	bin := filepath.Join(dir, util.BinaryName())
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("bun %s is not installed — run 'bvm install %s'", version, version)
	}

	args := os.Args[1:]
	if alias == "bunx" {
		args = append([]string{"x"}, args...)
	}
	return execProcessFn(bin, args)
}
