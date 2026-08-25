package util

import (
	"fmt"
	"testing"

	"github.com/chathula/bvm/config"
)

func TestPlatformAssetMatrix(t *testing.T) {
	cases := []struct {
		goos, goarch, asset, binary string
	}{
		{"darwin", "amd64", "bun-darwin-x64.zip", "bun"},
		{"darwin", "arm64", "bun-darwin-aarch64.zip", "bun"},
		{"linux", "amd64", "bun-linux-x64.zip", "bun"},
		{"linux", "arm64", "bun-linux-aarch64.zip", "bun"},
		{"windows", "amd64", "bun-windows-x64.zip", "bun.exe"},
		{"windows", "arm64", "bun-windows-aarch64.zip", "bun.exe"},
	}

	for _, c := range cases {
		p, err := NewPlatform(c.goos, c.goarch)
		if err != nil {
			t.Fatalf("NewPlatform(%s/%s) error = %v", c.goos, c.goarch, err)
		}
		if got := p.AssetName(); got != c.asset {
			t.Errorf("%s/%s AssetName() = %q, want %q", c.goos, c.goarch, got, c.asset)
		}
		if got := BinaryNameForOS(c.goos); got != c.binary {
			t.Errorf("BinaryNameForOS(%q) = %q, want %q", c.goos, got, c.binary)
		}
		wantURL := fmt.Sprintf("%s/releases/download/bun-v1.1.0/%s", config.BunGitHubRepoURL, c.asset)
		if got := p.DownloadURL("v1.1.0"); got != wantURL {
			t.Errorf("%s/%s DownloadURL() = %q, want %q", c.goos, c.goarch, got, wantURL)
		}
	}
}
