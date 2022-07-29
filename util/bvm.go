package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
