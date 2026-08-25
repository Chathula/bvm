<div align="center">

![bvm](https://socialify.git.ci/Chathula/bvm/image?description=1&font=Inter&forks=1&issues=1&language=1&logo=https%3A%2F%2Fuser-images.githubusercontent.com%2F709451%2F182802334-d9c42afe-f35d-4a7b-86ea-9985f73f20c3.png&name=1&owner=1&pattern=Circuit%20Board&pulls=1&stargazers=1&theme=Light)

**🚀 Bun Version Manager — manage multiple [Bun](https://bun.sh) versions easily**

</div>

## Features

- Install any published Bun version (`bvm install 1.1.0`, `bvm install latest`)
- Switch between installed versions instantly (`bvm use 1.4.0`)
- Works natively on **Linux, macOS (Intel & Apple Silicon) and Windows**
- Zero runtime dependencies — a single static binary

## Installation

### Linux / macOS

```sh
curl -fsSL https://raw.githubusercontent.com/chathula/bvm/main/install.sh | bash
```

### Windows (PowerShell)

```powershell
powershell -c "irm https://raw.githubusercontent.com/chathula/bvm/main/install.ps1 | iex"
```

Restart your shell afterwards so the updated `PATH` is picked up.

## Usage

```text
bvm install <version>   Install a bun version ('latest' allowed)
bvm use <version>       Activate an installed version
bvm list                List installed versions
bvm list-remote         List all remote versions
bvm uninstall <version> Remove an installed version
bvm doctor              Diagnose your bvm/bun setup
```

Examples:

```sh
bvm install latest     # installs and activates the newest Bun release
bvm install 1.1.0      # installs a specific version
bvm use 1.1.0          # switches to it
bvm ls                 # * marks the active version
bvm doctor             # verify PATH, active version, API reachability
```

> Windows note: Bun ships native Windows builds starting at **v1.1.0**;
> older versions cannot be installed on Windows.

## How it works

| Location | Purpose |
| --- | --- |
| `$BVM_DIR/versions/<vX.Y.Z>/` | Installed Bun binaries (`$BVM_DIR` defaults to `~/.bvm`) |
| `~/.bun/bin/bun` | Symlink (unix) or copy (Windows) to the active version |
| `$BVM_DIR/active` | Marker recording the active version |

Because activation goes through `~/.bun/bin`, any existing scripts or tooling
that expect Bun there keep working unchanged.

## Development

Requires Go ≥ 1.21 — the exact toolchain (`go1.27.0`) is pinned in `go.mod`
and fetched automatically.

```sh
make build      # build to bin/
make test       # unit tests
make test-e2e   # end-to-end tests (downloads real Bun releases)
make fmt vet    # formatting + static checks
```

E2E tests run the compiled binary in an isolated environment against real
Bun release archives, on every OS in CI (ubuntu / macos / windows).

Releases are cut by pushing a `v*` tag; GoReleaser builds and publishes
archives for all platforms, then smoke-tests both installer scripts against
the published release.
