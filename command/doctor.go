package command

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

type check struct {
	name   string
	ok     bool
	detail string
}

// probeAPI is a var so tests can stub the reachability probe offline.
var probeAPI = util.ProbeReleases

// pathHint returns guidance when the bun bin dir is off PATH; only Windows
// users are told to edit PATH by hand (unix profiles are managed for them).
func pathHint(goos string, onPath bool, binDir string) string {
	if goos != "windows" || onPath {
		return ""
	}
	return fmt.Sprintf("add '%s' to your PATH manually", binDir)
}

// linkDetail appends the symlink target for diagnostics. Unix-only: the
// activated binary is a plain copy on Windows.
func linkDetail(goos, binPath string, exists bool) string {
	if goos == "windows" || !exists {
		return ""
	}
	if target, err := os.Readlink(binPath); err == nil {
		return " -> " + target
	}
	return ""
}

// doctorChecks returns each diagnostic as a thunk so Doctor can print
// results progressively (the network probe runs last, after the user has
// already seen the local results).
func doctorChecks() []func() check {
	return []func() check{
		func() check {
			root, err := util.BVMDir()
			if err != nil {
				return check{name: "bvm directory", ok: false, detail: err.Error()}
			}
			installed, _ := util.LocalVersions()
			return check{
				name:   fmt.Sprintf("bvm directory (%s)", root),
				ok:     true,
				detail: fmt.Sprintf("%d version(s) installed", len(installed)),
			}
		},
		func() check {
			def, err := util.DefaultVersion()
			switch {
			case err != nil:
				return check{name: "default alias", ok: false, detail: err.Error()}
			case def == "":
				return check{name: "default alias", ok: true, detail: "not set — 'bvm use' falls back to highest installed"}
			default:
				ok := util.IsInstalled(def)
				detail := def
				if !ok {
					detail += " (not installed)"
				}
				return check{name: "default alias", ok: ok, detail: detail}
			}
		},
		func() check {
			binPath, err := util.BunBinPath()
			if err != nil {
				return check{name: "bun shim", ok: false, detail: err.Error()}
			}
			_, statErr := os.Stat(binPath)
			detail := binPath + linkDetail(runtime.GOOS, binPath, statErr == nil)
			return check{name: "bun shim (~/.bun/bin)", ok: statErr == nil, detail: detail}
		},
		func() check {
			binPath, _ := util.BunBinPath()
			binDir := filepath.Dir(binPath)
			onPath := util.PathContains(binDir)
			return check{name: "bun shim on PATH", ok: onPath, detail: pathHint(runtime.GOOS, onPath, binDir)}
		},
		func() check {
			version, source, err := util.ResolveVersion()
			if err != nil {
				return check{name: "version for this directory", ok: false, detail: err.Error()}
			}
			return check{name: "version for this directory", ok: true, detail: version + " (" + source + ")"}
		},
		func() check {
			ok, detail := probeAPI()
			return check{name: "releases API reachable", ok: ok, detail: detail}
		},
	}
}

// Doctor diagnoses the local bvm/bun setup and exits non-zero on failures.
// Results print as they are evaluated so nothing appears to hang.
func Doctor() error {
	failed := 0
	for _, evaluate := range doctorChecks() {
		c := evaluate()
		mark := color.GreenString("✔")
		if !c.ok {
			mark = color.RedString("✘")
			failed++
		}
		line := fmt.Sprintf("%s %s", mark, c.name)
		if c.detail != "" {
			line += color.New(color.FgHiBlack).Sprintf("  (%s)", c.detail)
		}
		fmt.Println(line)
	}

	if failed > 0 {
		return fail("%d check(s) failed", failed)
	}
	return nil
}
