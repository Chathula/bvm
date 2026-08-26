package command

import (
	"fmt"
	"strings"

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
	def, _ := util.DefaultVersion()

	for _, version := range versions {
		marker := "  "
		name := version
		if version == active {
			marker = "*"
			name = color.GreenString(version)
		}

		var tags []string
		if version == active {
			tags = append(tags, "active")
		}
		if version == def && def != "" {
			tags = append(tags, "default")
		}
		suffix := ""
		if len(tags) > 0 {
			suffix = color.New(color.FgHiBlack).Sprintf("(%s)", strings.Join(tags, ", "))
		}

		fmt.Printf("%s %s %s\n", marker, name, suffix)
	}
	return nil
}
