<div align="center">

![bvm](https://socialify.git.ci/Chathula/bvm/image?description=1&font=Inter&forks=1&issues=1&language=1&logo=https%3A%2F%2Fuser-images.githubusercontent.com%2F709451%2F182802334-d9c42afe-f35d-4a7b-86ea-9985f73f20c3.png&name=1&owner=1&pattern=Circuit%20Board&pulls=1&stargazers=1&theme=Light)

**🚀 Bun Version Manager — manage multiple [Bun](https://bun.sh) versions easily**

</div>

## Features

- Install any published Bun version (`bvm install 1.1.0`, `bvm install latest`)
- nvm-style **default version**: new shells and unpinned directories use it
- **Per-project versions** via an optional `.bvmrc` (`bvm use --save 1.4.0`)
- Works natively on **Linux, macOS (Intel & Apple Silicon) and Windows**
- Zero runtime dependencies — a single static binary

## Installation

### Linux / macOS

```sh
curl -s -S -L https://raw.githubusercontent.com/chathula/bvm/main/install.sh | bash
```

Or if you are using zsh just change `bash` with `zsh`:

```zsh
curl -s -S -L https://raw.githubusercontent.com/chathula/bvm/main/install.sh | zsh
```

> **Using a different shell?** No problem — the command above works in bash,
> zsh and fish as-is. The installer detects your login shell, updates the
> right profile (`.zshrc`, `.bashrc` or fish's `config.fish`), and prints the
> exact command to activate bvm in your current session when it finishes.

The installer puts the `bvm` binary in `~/.bvm/bin` and adds it to your PATH.
**`bvm` is not available in the current session yet** — either open a new
terminal, or run the `source ~/.zshrc` / `source ~/.bashrc` command the
installer prints when it finishes.

Prefer not piping scripts into a shell? Two-step equivalent:

```sh
curl -fsSLO https://raw.githubusercontent.com/chathula/bvm/main/install.sh
less install.sh        # inspect it first
bash install.sh && rm install.sh
```

### Windows (PowerShell)

```powershell
powershell -c "irm https://raw.githubusercontent.com/chathula/bvm/main/install.ps1 | iex"
```

Installs to `%USERPROFILE%\.bvm\bin` and adds it to your user PATH — reopen
your terminal afterwards.

### Other options

```sh
# If you have Go tooling installed:
go install github.com/chathula/bvm@latest
```

Or download an archive for your platform from the
[releases page](https://github.com/Chathula/bvm/releases), unzip it, and put
the `bvm` binary somewhere on your `PATH`.

## Troubleshooting

- **`command not found: bvm` right after installing** — your current shell
  hasn't re-read its profile. Open a new terminal, or run
  `source ~/.zshrc` (zsh) / `source ~/.bashrc` (bash) /
  `source ~/.config/fish/config.fish` (fish).
- **Installer prints nothing / 404** — make sure the URL matches your branch
  (`main`) and that the repository is public; retry with `-v` on curl.
- **`unzip` missing** — the installer needs it; on Debian/Ubuntu run
  `sudo apt-get install unzip`, on Alpine `apk add unzip`.
- **Existing Bun installed another way** — bvm activates versions through
  `~/.bun/bin/bun`; if that path belongs to another install, remove it from
  your PATH or let bvm manage it going forward. Run `bvm doctor` any time to
  diagnose your setup.

## Usage

```text
bvm install [version]   Install a bun version ('latest' allowed; defaults to .bvmrc)
bvm use [--save] [v]    Show what applies here; '--save <v>' pins this project (.bvmrc)
bvm use default         Remove this directory's pin (if any)
bvm use --reset         Clear a session override ($BVM_VERSION)
bvm exec <v> [args]     Run a one-off command with a specific version
bvm alias default <v>   Set the version used outside pinned projects
bvm list                List installed versions
bvm list-remote         List all remote versions
bvm uninstall <version> Remove an installed version
bvm doctor              Diagnose your bvm/bun setup
```

(No separate `which` command — `bvm use` with no argument shows which
version applies in the current directory and why.)

Examples:

```sh
bvm install latest      # first install ever -> becomes the default
bvm install 1.1.0       # installs a specific version
bvm use                 # what version applies in this directory, and why
bvm use --save 1.1.0    # pins 1.1.0 to the current project (creates .bvmrc)
bvm use default         # remove this project's pin
bvm ls                  # * marks the default, (this project) marks the pin
bvm doctor              # verify shim, PATH, resolution, API reachability
```

> Windows note: Bun ships native Windows builds starting at **v1.1.0**;
> older versions cannot be installed on Windows.

## How version resolution works

bvm works like nvm's default alias, implemented through a **shim**: the
`bun` command on your PATH is bvm itself, and it resolves which real Bun
binary to run on every invocation.

1. `$BVM_VERSION` if set — a **temporary, session-only** override
   (see below)
2. The nearest `.bvmrc` walking up from your current directory
   (project pin — **always opt-in**, created only by `bvm use --save <version>`
   or written by hand)
3. The **default** alias (`bvm alias default <version>`)
4. Nothing — you get a helpful error instead of a mystery binary

Consequences (matching nvm's mental model):

- The **first version you install becomes the default**
- New shells and unpinned directories always run the **default** version
- `bvm use` **never creates files** — it reports what applies here; only
  `bvm use --save <version>` writes a `.bvmrc`, and only when you ask
- `.bvmrc` is completely optional: commit it to share a project's Bun
  version, or skip it entirely and rely on the default
- `bun` works everywhere: no shell hooks or PATH juggling per project

A `.bvmrc` follows the same rules everywhere: blank lines and `#` comments
are ignored, the first version line wins, and `1.4.0` / `v1.4.0` are
equivalent. `bvm install` with no argument reads it too.

### Temporary version switching (nvm-style `use`)

Like `nvm use`, you can switch versions for your **current terminal session**
only — files, other terminals and other projects are untouched:

```sh
bvm use 1.4.0        # this shell now runs 1.4.0 (sets $BVM_VERSION)
bvm use --reset      # back to .bvmrc / default
bvm use default      # clears the override AND removes this project's pin
```

When you close the shell (or `unset BVM_VERSION`), the project's `.bvmrc`
— or the default — applies again, exactly like coming back to a project
under nvm.

For one-off commands without changing your session:

```sh
bvm exec 1.1.0 bun test
```

> This works through the small `bvm()` shell function the installer adds to
> your profile (bash/zsh/fish): the function runs inside your shell and sets
> `$BVM_VERSION`, which the shim checks first. Existing installs can re-run
> the installer to pick it up, or add the snippet to your profile manually.

`bvm ls` shows both roles:

```text
  v1.3.1 (this project)
* v1.4.0 (default)
```

### Uninstalling a Bun version

```sh
bvm uninstall 1.3.1
```

- Removing a regular version just deletes it
- Removing the **default** version promotes the highest remaining one —
  `bun` keeps working everywhere
- Removing the **last** version also removes the shim and the default
  alias; `bun` prints a clear "not installed" message until you install again
- If the current project's `.bvmrc` pinned the removed version, bvm warns
  you to update it

## Uninstalling bvm

Manual removal, nvm-style:

```sh
# 1. remove the installation directory (versions, shim, aliases)
bvm_dir="${BVM_DIR:-~/.bvm}"
rm -rf "$bvm_dir"

# 2. remove the shim bvm placed on your PATH
rm -f ~/.bun/bin/bun        # unix (skip if you installed Bun independently)
```

Then edit your shell profile (`~/.zshrc`, `~/.bashrc` or fish's
`config.fish`) and delete the lines bvm added:

```sh
# bvm & bun
export BVM_DIR="$HOME/.bvm"
export PATH="$BVM_DIR/bin:$PATH"
```

On Windows (PowerShell):

```powershell
Remove-Item -Recurse -Force "$env:USERPROFILE\.bvm"
Remove-Item -Force "$env:USERPROFILE\.bun\bin\bun.exe"
# then remove "%USERPROFILE%\.bvm\bin" from your PATH (Settings > Environment Variables)
```

Run `bvm doctor` before uninstalling if you want a report of everything
bvm currently manages.

## Development

Requires Go ≥ 1.21 — the exact toolchain (`go1.27.0`) is pinned in `go.mod`
and fetched automatically.

```sh
make build      # build to bin/
make test       # unit tests
make test-e2e   # end-to-end tests (downloads real Bun releases)
make cover      # coverage report + enforces a 90% floor (excluding main())
make fmt vet    # formatting + static checks
```

E2E tests run the compiled binary in an isolated environment against real
Bun release archives, on every OS in CI (ubuntu / macos / windows).

Releases are cut by pushing a `v*` tag; GoReleaser builds and publishes
archives for all platforms, then smoke-tests both installer scripts against
the published release.
