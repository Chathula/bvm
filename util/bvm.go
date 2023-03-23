package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chathula/bvm/config"
)

type node struct {
	Name       string `json:"name"`
	ZipBallUrl string `json:"zipball_url"`
	TarBallUrl string `json:"tarball_url"`
	commit     struct {
		SHA string `json:"sha"`
		URL string `json:"url"`
	}
	NodeId string `json:"node_id"`
}

func GetRemoteVersions() ([]string, error) {
	response, err := http.Get(config.BUN_RELEASE_UPDATED_GITHUB_API_URL + "/tags")

	if err != nil {
		return nil, fmt.Errorf("request failed on url: `%s` ", config.BUN_RELEASE_UPDATED_GITHUB_API_URL)
	}

	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("request failed withh response code:  %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, errors.New("error in response body")
	}

	var data []node

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, errors.New("JSON parsing failed")
	}

	var versions []string

	for i := len(data) - 1; i >= 0; i-- {
		version := strings.Replace(strings.TrimSpace(data[i].Name), "bun-", "", -1)
		versions = append(versions, version)
	}

	return versions, nil
}

func ValidVersion(version string) error {
	versions, err := GetRemoteVersions()

	if err != nil {
		return err
	}

	for _, v := range versions {
		if strings.EqualFold(v, version) {
			return nil
		}

	}

	return fmt.Errorf("version %s not found", version)
}

func UseVersion(version string) error {
	err := ValidVersion(version)
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	bunPath := filepath.Join(homeDir, ".bun", "bin", "bun")
	if _, err := os.Stat(bunPath); err == nil {
		err = os.Remove(bunPath)
		if err != nil {
			return err
		}
	}

	bunSourcePath := filepath.Join(homeDir, ".bvm", version, "bun")

	err = os.MkdirAll(filepath.Join(homeDir, ".bun", "bin"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(bunSourcePath, bunPath)
	if err != nil {
		return err
	}

	SetPathVariable()

	return nil
}

// TODO: work on proper $PATH env variable update
func SetPathVariable() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	bunPath := filepath.Join(homeDir, ".bun", "bin")
	paths := os.Getenv("PATH")

	if strings.Contains(paths, bunPath) {
		return nil
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		zshFilePath, err := exec.LookPath("zsh")
		bashFilePath, err := exec.LookPath("bash")
		// fishFilePath, err := exec.LookPath("fish")
		if err != nil {
			return err
		}

		if zshFilePath != "" {
			zshrcPath := filepath.Join(homeDir, ".zshrc")
			file, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = file.WriteString("\nexport PATH=\"$PATH:" + bunPath + "\"\n")
			if err != nil {
				return err
			}

			exec.Command("exec", "zsh").Run()
		}

		if bashFilePath != "" {
			zshrcPath := filepath.Join(homeDir, ".bashrc")
			file, err := os.OpenFile(zshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = file.WriteString("\nexport PATH=\"$PATH:" + bunPath + "\"\n")
			if err != nil {
				return err
			}

			exec.Command("source", "~/.bashrc").Run()
		}

	}

	return nil
}
