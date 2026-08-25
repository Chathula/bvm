package command

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chathula/bvm/config"
	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

type check struct {
	name   string
	ok     bool
	detail string
}

// Doctor diagnoses the local bvm/bun setup and exits non-zero on failures.
func Doctor() error {
	var checks []check

	root, err := util.BVMDir()
	if err == nil {
		installed, _ := util.LocalVersions()
		checks = append(checks, check{
			name:   fmt.Sprintf("bvm directory (%s)", root),
			ok:     err == nil,
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
		detail := active
		if !util.IsInstalled(active) {
			detail += " (binary missing!)"
		}
		checks = append(checks, check{name: "active version", ok: util.IsInstalled(active), detail: detail})
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
	onPath := false
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p == binDir {
			onPath = true
			break
		}
	}
	hint := ""
	if runtime.GOOS == "windows" && !onPath {
		hint = fmt.Sprintf("add '%s' to your PATH manually", binDir)
	}
	checks = append(checks, check{name: "~/.bun/bin on PATH", ok: onPath, detail: hint})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(config.BunReleasesAPIURL + "/tags?per_page=1")
	apiOK := err == nil && resp.StatusCode == http.StatusOK
	if resp != nil {
		resp.Body.Close()
	}
	detail := config.BunReleasesAPIURL
	if err != nil {
		detail = err.Error()
	}
	checks = append(checks, check{name: "releases API reachable", ok: apiOK, detail: detail})

	failed := 0
	for _, c := range checks {
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
