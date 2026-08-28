package command

import (
	"os"
	"path/filepath"

	"github.com/chathula/bvm/util"
)

// Exec runs a one-off command against a specific bun version without
// changing any state (session, .bvmrc or default):
//
//	bvm exec 1.1.0 bun test
func Exec(version string, args []string) error {
	norm, err := util.NormalizeVersion(version)
	if err != nil {
		return fail("%v", err)
	}
	dir, err := util.VersionDir(norm)
	if err != nil {
		return fail("%v", err)
	}
	bin := filepath.Join(dir, util.BinaryName())
	if _, err := os.Stat(bin); err != nil {
		return fail("bun %s is not installed — run 'bvm install %s'", norm, norm)
	}

	// Tolerate an explicit "bun" so both `bvm exec 1.1.0 bun test` and
	// `bvm exec 1.1.0 test` work.
	if len(args) > 0 && (args[0] == "bun" || args[0] == "bun.exe") {
		args = args[1:]
	}
	return execProcessFn(bin, args)
}
