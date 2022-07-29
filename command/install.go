package command

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/chathula/bvm/util"
	"github.com/fatih/color"

	semver "github.com/Masterminds/semver/v3"
)

func Install(version string) error {
	// Detect the os and cpu architecture

	// if windows print an error

	var os string
	var arch string

	switch runtime.GOOS {
	case "windows":
		return errors.New(color.RedString("Please install bun using Windows Subsystem for Linux"))
	case "darwin":
		os = "darwin"
	case "linux":
		os = "linux"
	default:
		return errors.New(color.RedString("Platform not supported"))
	}

	switch runtime.GOARCH {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "aarch64"
	default:
		return errors.New(color.RedString("Platform not supported"))
	}

	filename := fmt.Sprintf("bun-%s-%s.zip", os, arch)

	// if the version == 'latest', get the latest version from list-remote
	var downloadVersion string

	if version == "latest" {
		versions, err := util.GetRemoteVersions()
		if err != nil {
			return err
		}

		downloadVersion = versions[len(versions)-1]
	} else {
		v, err := semver.NewVersion(version)
		if err != nil {
			return errors.New(color.RedString("Invalid version"))
		}

		downloadVersion = fmt.Sprintf("v%s", v.String())
	}

	fmt.Println(filename)
	fmt.Println(downloadVersion)

	// Use the version and check whether it is available currently in local. if so print message

	// generate download url to download source file

	// download the zip or print error message by saying this version is not exists

	// unzip the source and move to ~/.bvm folder with subdirectory with vx.x.x name

	// detect shell type and update shell config to have bun command, if there is no version is currently in use

	// print success message

	return nil
}
