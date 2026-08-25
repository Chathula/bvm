package command

import (
	"fmt"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
)

// ListRemote prints all remote Bun versions, marking the newest as latest.
func ListRemote() error {
	versions, err := util.RemoteVersions()
	if err != nil {
		return fail("%v", err)
	}
	for i, version := range versions {
		if i == len(versions)-1 {
			fmt.Printf("%s %s\n", version, color.GreenString("(latest)"))
		} else {
			fmt.Println(version)
		}
	}
	return nil
}
