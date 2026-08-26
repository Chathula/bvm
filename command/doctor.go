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

// buildDoctorChecks assembles every diagnostic, split from printing so it
// stays unit-testable.
func buildDoctorChecks() []check {
	var checks []check

	root, err := util.BVMDir()
	if err == nil {
		installed, _ := util.LocalVersions()
		checks = append(checks, check{
			name:   fmt.Sprintf("bvm directory (%s)", root),
			ok:     true,
			detail: fmt.Sprintf("%d version(s) installed", len(installed)),
		})
	} else {
		checks = append(checks, check{name: "bvm directory", ok: false, detail: err.Error()})
	}

	active, activeErr := util.ActiveVersion()
	switch {
	case activeErr != nil:
		checks = append(checks, check{name: "active version marker", ok: false, detail: activeErr.Error()})
	case active == "":
		checks = append(checks, check{name: "active version", ok: true, detail: "none — run 'bvm install latest'"})
	default:
		ok := util.IsInstalled(active)
		detail := active
		if !ok {
			detail += " (binary missing!)"
		}
		checks = append(checks, check{name: "active version", ok: ok, detail: detail})
	}

	def, defErr := util.DefaultVersion()
	switch {
	case defErr != nil:
		checks = append(checks, check{name: "default alias", ok: false, detail: defErr.Error()})
	case def == "":
		checks = append(checks, check{name: "default alias", ok: true, detail: "not set — 'bvm use' falls back to highest installed"})
	default:
		ok := util.IsInstalled(def)
		detail := def
		if !ok {
			detail += " (not installed)"
		}
		checks = append(checks, check{name: "default alias", ok: ok, detail: detail})
	}

	binPath, err := util.BunBinPath()
	if err != nil {
		checks = append(checks, check{name: "bun binary link", ok: false, detail: err.Error()})
	} else {
		_, statErr := os.Stat(binPath)
		detail := binPath
		if runtime.GOOS != "windows" && statErr == nil {
			if target, linkErr := os.Readlink(binPath); linkErr == nil {
				detail += " -> " + target
			}
		}
		checks = append(checks, check{name: "bun binary link", ok: statErr == nil, detail: detail})
	}

	binDir := filepath.Dir(binPath)
	onPath := util.PathContains(binDir)
	checks = append(checks, check{name: "~/.bun/bin on PATH", ok: onPath, detail: pathHint(runtime.GOOS, onPath, binDir)})

	apiOK, apiDetail := probeAPI()
	checks = append(checks, check{name: "releases API reachable", ok: apiOK, detail: apiDetail})

	return checks
}

// Doctor diagnoses the local bvm/bun setup and exits non-zero on failures.
func Doctor() error {
	failed := 0
	for _, c := range buildDoctorChecks() {
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
