package command

import (
	"fmt"

	"github.com/chathula/bvm/util"
)

//  TODO: improve code to get all the version instead of initial 30 records
func ListRemote() error {

	versions, err := util.GetRemoteVersions()

	if err != nil {
		return err
	}

	for _, version := range versions {
		fmt.Println(version)
	}

	return nil
}
