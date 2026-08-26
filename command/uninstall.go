package command

import (
	"fmt"
	"sort"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// Uninstall removes a locally installed Bun version. When the removed
// version was the default, the default moves to the highest remaining
// version; when it was the last one, the shim and default are cleaned up.
func Uninstall(arg string) error {
	version, err := util.NormalizeVersion(arg)
	if err != nil {
		return fail("%v", err)
	}
	dir, err := util.VersionDir(version)
	if err != nil {
		return fail("%v", err)
	}
	if !util.IsInstalled(version) {
		return fail("version %s is not installed", version)
	}

	// Warn when the current project pins the version being removed.
	if rcPath, pinned := util.FindRCHere(); pinned == version {
		fmt.Println(color.YellowString("Note: %s pins %s — update it with 'bvm use <other-version>'.", rcPath, version))
	}

	if err := util.RemoveAll(dir); err != nil {
		return fail("%v", err)
	}

	if def, _ := util.DefaultVersion(); def == version {
		if err := reassignDefault(version); err != nil {
			return fail("%v", err)
		}
	}
	return nil
}

// setDefaultVersion is a var so tests can inject write failures portably.
var setDefaultVersion = util.SetDefaultVersion

// clearDefault is a var so tests can inject removal failures portably.
var clearDefault = util.ClearDefault

// reassignDefault moves the default alias off a removed version: to the
// highest remaining version, or clears it (and the shim) when none remain.
func reassignDefault(removed string) error {
	remaining, err := util.LocalVersions()
	if err != nil {
		return err
	}
	remaining = without(remaining, removed)

	if len(remaining) == 0 {
		if err := clearDefault(); err != nil {
			return err
		}
		if err := util.RemoveShim(); err != nil {
			return err
		}
		fmt.Println(color.YellowString("No bun versions left — removed the default alias and the bun shim. Run 'bvm install latest' to start over."))
		return nil
	}

	next := remaining[len(remaining)-1]
	if err := setDefaultVersion(next); err != nil {
		return err
	}
	fmt.Println(color.YellowString("Default bun version moved to %s.", next))
	return nil
}

func without(versions []string, v string) []string {
	out := make([]string, 0, len(versions))
	for _, x := range versions {
		if x != v {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
