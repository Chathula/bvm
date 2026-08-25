package command

import (
	"fmt"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// List prints locally installed Bun versions, marking the active one.
func List() error {
	versions, err := util.LocalVersions()
	if err != nil {
		return fail("%v", err)
	}
	if len(versions) == 0 {
		fmt.Println("no versions installed yet — run 'bvm install latest'")
		return nil
	}

	active, err := util.ActiveVersion()
	if err != nil {
		return fail("%v", err)
	}
	for _, version := range versions {
		if version == active {
			fmt.Printf("* %s %s\n", color.GreenString(version), color.New(color.FgHiBlack).Sprint("(active)"))
		} else {
			fmt.Printf("  %s\n", version)
		}
	}
	return nil
}
