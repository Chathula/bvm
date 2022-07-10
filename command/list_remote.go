package command

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

//  TODO: improve code to get all the version instead of initial 30 records
func ListRemote() error {

	response, err := http.Get(config.BUN_RELEASE_UPDATED_GITHUB_API_URL + "/tags")

	if err != nil {
		return fmt.Errorf("request failed on url: `%s` ", config.BUN_RELEASE_UPDATED_GITHUB_API_URL)
	}

	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("request failed withh response code:  %d", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return errors.New("error in response body")
	}

	var data []node

	if err := json.Unmarshal(body, &data); err != nil {
		return errors.New("JSON parsing failed")
	}

	for _, d := range data {
		version := strings.Replace(strings.TrimSpace(d.Name), "bun-", "", -1)
		fmt.Println(version)
	}

	return nil
}
