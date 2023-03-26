package command

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chathula/bvm/config"
	"github.com/chathula/bvm/util"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"

	semver "github.com/Masterminds/semver/v3"
)

func Install(version string) error {
	// Detect the os and cpu architecture
	// if windows or any other unsupported OS print an error

	var systemOS string
	var arch string

	switch runtime.GOOS {
	case "windows":
		return errors.New(color.RedString("Please install bun using Windows Subsystem for Linux"))
	case "darwin":
		systemOS = "darwin"
	case "linux":
		systemOS = "linux"
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

	filename := fmt.Sprintf("bun-%s-%s.zip", systemOS, arch)

	// if the version == 'latest', get the latest version from list-remote
	var downloadVersion string

	if version == "latest" {
		versions, err := util.GetRemoteVersions()
		if err != nil {
			return errors.New(color.RedString(err.Error()))
		}

		downloadVersion = strings.ToLower(versions[len(versions)-1])
	} else {
		v, err := semver.NewVersion(version)
		if err != nil {
			return errors.New(color.RedString("Invalid version"))
		}

		downloadVersion = strings.ToLower(fmt.Sprintf("v%s", v.String()))

		invalidErr := util.ValidVersion(version)
		if invalidErr != nil {
			return errors.New(color.RedString(invalidErr.Error()))
		}
	}

	// TODO: Use the version and check whether it is available currently in local. if so print message

	// generate download url to download source file
	// https://github.com/oven-sh/bun/releases/download/bun-v0.5.8/bun-darwin-aarch64.zip
	downloadUrl := fmt.Sprintf("%s/releases/download/bun-%s/%s", config.BUN_GITHUB_REPO_URL, downloadVersion, filename)

	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}
	defer resp.Body.Close()

	downloadFileName := fmt.Sprintf("%s-%s", downloadVersion, filename)

	tmpFile, err := ioutil.TempFile("", downloadFileName)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	defer os.Remove(tmpFile.Name()) // Delete the temporary file when done

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"downloading",
	)

	_, err = io.Copy(io.MultiWriter(tmpFile, bar), resp.Body)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	// unzip the source and move to ~/.bvm folder with subdirectory with vx.x.x name
	// Move the downloaded file to the current directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	os.MkdirAll(filepath.Join(homeDir, ".bvm", version), os.ModePerm)

	dstFilepath := filepath.Join(fmt.Sprintf("%s/.bvm/%s", homeDir, version), downloadFileName)
	if err := os.Rename(tmpFile.Name(), dstFilepath); err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	zipFile, err := zip.OpenReader(dstFilepath)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	defer zipFile.Close()

	for _, zipFile := range zipFile.File {
		filePath := filepath.Join(homeDir, ".bvm", version, zipFile.Name)

		if zipFile.FileInfo().IsDir() {
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		file, err := os.Create(filePath)
		if err != nil {
			return errors.New(color.RedString(err.Error()))
		}
		defer file.Close()

		zipContent, err := zipFile.Open()
		if err != nil {
			return errors.New(color.RedString(err.Error()))
		}
		defer zipContent.Close()

		_, err = io.Copy(file, zipContent)
		if err != nil {
			return errors.New(color.RedString(err.Error()))
		}
	}

	filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	basePath := filepath.Join(homeDir, ".bvm", version)

	bytesRead, err := ioutil.ReadFile(filepath.Join(basePath, filenameWithoutExt, "bun"))
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	err = ioutil.WriteFile(filepath.Join(basePath, "bun"), bytesRead, 0755)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	err = os.RemoveAll(filepath.Join(basePath, filenameWithoutExt))
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	err = os.Remove(dstFilepath)
	if err != nil {
		return errors.New(color.RedString(err.Error()))
	}

	useVersionErr := util.UseVersion(downloadVersion)

	if useVersionErr != nil {
		return errors.New(color.RedString(err.Error()))
	}

	fmt.Println(color.GreenString("Successfully installed bun version: %s", downloadVersion))

	return nil
}
