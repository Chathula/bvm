package util

import (
	"fmt"
	"runtime"

	"github.com/chathula/bvm/config"
)

// Platform identifies a Bun release target (asset naming scheme).
type Platform struct {
	OS   string // darwin | linux | windows
	Arch string // x64 | aarch64
}

// NewPlatform maps a GOOS/GOARCH pair to a Bun release target.
func NewPlatform(goos, goarch string) (Platform, error) {
	p := Platform{OS: goos}
	switch goarch {
	case "amd64":
		p.Arch = "x64"
	case "arm64":
		p.Arch = "aarch64"
	default:
		return p, fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, goos, goarch)
	}

	switch goos {
	case "darwin", "linux", "windows":
	default:
		return p, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, goos)
	}
	return p, nil
}

// DetectPlatform resolves the host GOOS/GOARCH to a Bun release target.
func DetectPlatform() (Platform, error) {
	return NewPlatform(runtime.GOOS, runtime.GOARCH)
}

// AssetName returns the release archive filename, e.g. "bun-darwin-aarch64.zip".
func (p Platform) AssetName() string {
	return fmt.Sprintf("bun-%s-%s.zip", p.OS, p.Arch)
}

// DownloadURL builds the release archive URL for a canonical version ("v1.1.0").
func (p Platform) DownloadURL(version string) string {
	return fmt.Sprintf("%s/releases/download/bun-%s/%s", config.BunGitHubRepoURL, version, p.AssetName())
}
