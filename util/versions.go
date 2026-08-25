package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"

	"github.com/chathula/bvm/config"
)

// releasesAPIURL is a var so tests can point it at a mock server.
var releasesAPIURL = config.BunReleasesAPIURL

// apiClient bounds how long API calls may hang; release archive downloads
// use a separate unbounded path since they transfer large files.
var apiClient = &http.Client{Timeout: 15 * time.Second}

// APIGet performs an authenticated GET when GITHUB_TOKEN/GH_TOKEN is set,
// which lifts the 60 req/hour anonymous limit on api.github.com — required
// on shared CI runners where that quota is routinely exhausted.
func APIGet(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return apiClient.Do(req)
}

type gitTag struct {
	Name string `json:"name"`
}

// stableTagRe matches only plain release tags ("bun-v1.4.0"). The upstream
// repo may carry other channels (canary, betas) which have no standard
// release assets and must never resolve as "latest".
var stableTagRe = regexp.MustCompile(`^bun-v\d+\.\d+\.\d+$`)

// RemoteVersions returns every published Bun version ordered oldest→newest,
// in canonical form "vX.Y.Z". Follows API pagination.
func RemoteVersions() ([]string, error) {
	var versions []string // collected newest-first, matching the API order

	for page := 1; ; page++ {
		url := fmt.Sprintf("%s/tags?per_page=100&page=%d", releasesAPIURL, page)
		resp, err := APIGet(url)
		if err != nil {
			return nil, fmt.Errorf("request failed on %s: %w", url, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed reading response body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
		}

		var tags []gitTag
		if err := json.Unmarshal(body, &tags); err != nil {
			return nil, fmt.Errorf("failed parsing response JSON: %w", err)
		}
		if len(tags) == 0 {
			break
		}

		for _, tag := range tags {
			if !stableTagRe.MatchString(tag.Name) {
				continue
			}
			name := strings.ToLower(strings.TrimPrefix(tag.Name, "bun-"))
			if name != "" {
				versions = append(versions, name)
			}
		}
		if len(tags) < 100 {
			break
		}
	}

	// Flip once at the end so callers always see oldest→newest.
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}
	return versions, nil
}

// NormalizeVersion converts user input ("1.2.3", "V1.2.3") to "v1.2.3".
func NormalizeVersion(input string) (string, error) {
	v, err := semver.NewVersion(strings.ToLower(strings.TrimSpace(input)))
	if err != nil {
		return "", fmt.Errorf("invalid version %q", input)
	}
	return strings.ToLower("v" + v.String()), nil
}

// ValidRemoteVersion reports whether a canonical version exists upstream.
func ValidRemoteVersion(version string) error {
	versions, err := RemoteVersions()
	if err != nil {
		return err
	}
	for _, v := range versions {
		if strings.EqualFold(v, version) {
			return nil
		}
	}
	return fmt.Errorf("version %s not found in remote releases", version)
}

// ResolveTarget turns "latest" or a concrete version into a canonical,
// remote-verified version string.
func ResolveTarget(input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.EqualFold(input, "latest") {
		versions, err := RemoteVersions()
		if err != nil {
			return "", err
		}
		if len(versions) == 0 {
			return "", errors.New("no remote versions found")
		}
		return versions[len(versions)-1], nil
	}

	version, err := NormalizeVersion(input)
	if err != nil {
		return "", err
	}
	if err := ValidRemoteVersion(version); err != nil {
		return "", err
	}
	return version, nil
}
