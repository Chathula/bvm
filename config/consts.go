// Package config holds constants and filesystem layout used across bvm.
package config

const (
	// BunReleasesAPIURL lists Bun release tags; mirrors oven-sh/bun releases
	// but is stable and dedicated to update tooling.
	BunReleasesAPIURL = "https://api.github.com/repos/oven-sh/bun-releases-for-updater"

	// BunGitHubRepoURL serves the downloadable release archives.
	BunGitHubRepoURL = "https://github.com/oven-sh/bun"
)
