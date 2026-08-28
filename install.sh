#!/usr/bin/env bash

set -e

fetch() {
	if command -v curl >/dev/null; then
		curl --fail --location --progress-bar -o "$2" "$1"
	elif command -v wget >/dev/null; then
		wget -q --show-progress -O "$2" "$1"
	else
		echo "Error: curl or wget is required to install bvm." 1>&2
		exit 1
	fi
}

if ! command -v unzip >/dev/null; then
	echo "Error: unzip is required to install bvm." 1>&2
	exit 1
fi

if [ "$OS" = "Windows_NT" ]; then
	echo "Error: use install.ps1 on Windows (or WSL with the bash installer)." 1>&2
	exit 1
fi

case $(uname -sm) in
"Darwin x86_64") target="darwin_x86_64" ;;
"Darwin arm64") target="darwin_arm64" ;;
"Linux x86_64") target="linux_x86_64" ;;
"Linux aarch64") target="linux_aarch64" ;;
*) echo "Unsupported OS + CPU combination: $(uname -sm)" 1>&2; exit 1 ;;
esac

bvm_url="https://github.com/chathula/bvm/releases/latest/download/bvm_${target}.zip"

bvm_dir="${BVM_DIR:-$HOME/.bvm}"
bvm_bin_dir="$bvm_dir/bin"
exe="$bvm_bin_dir/bvm"

mkdir -p "$bvm_bin_dir"

if [ "$1" = "" ]; then
	fetch "$bvm_url" "$exe.zip"
	unzip -o "$exe.zip" -d "$bvm_bin_dir"
	rm "$exe.zip"
else
	echo "Install path override detected: $1"
	if [ ! -f "$1" ]; then
		echo "File does not exist: $1"
		exit 1
	fi
	cp "$1" "$exe"
fi

chmod +x "$exe"

# macOS: Apple Silicon kills unsigned or quarantine-flagged binaries with
# "zsh: killed". Re-sign ad-hoc and drop the flag so the binary runs.
if [ "$(uname)" = "Darwin" ] && command -v codesign >/dev/null; then
	xattr -d com.apple.quarantine "$exe" 2>/dev/null || true
	codesign --force --sign - "$exe" 2>/dev/null || true
fi

case $SHELL in
*/zsh) shell_profile=".zshrc"; source_cmd="source ~/.zshrc" ;;
*/fish) shell_profile=".config/fish/config.fish"; source_cmd="source ~/.config/fish/config.fish" ;;
*) shell_profile=".bashrc"; source_cmd="source ~/.bashrc" ;;
esac

if [ ! $BVM_DIR ]; then
	case $shell_profile in
	.config/fish/config.fish)
		{
			printf '\n# bvm & bun\n'
			command echo "set --export BVM_DIR \"$bvm_dir\""
			command echo "fish_add_path \"\$BVM_DIR/bin\""
		} >>"$HOME/$shell_profile"
		;;
	*)
		{
			printf '\n# bvm & bun\n'
			command echo "export BVM_DIR=\"$bvm_dir\""
			command echo "export PATH=\"\$BVM_DIR/bin:\$PATH\""
		} >>"$HOME/$shell_profile"
		;;
	esac
fi

echo "bvm was installed successfully to $exe"
if ! command -v bvm >/dev/null; then
	echo
	echo "bvm is not on this shell's PATH yet. To finish setup, either:"
	echo "  1. open a new terminal, or"
	echo "  2. run:  $source_cmd"
fi

# Shell integration: enables temporary, nvm-style `bvm use <version>` that
# switches versions for the current session only (via $BVM_VERSION).
case $shell_profile in
.config/fish/config.fish)
	if ! grep -q "bvm session override support" "$HOME/$shell_profile" 2>/dev/null; then
		cat >>"$HOME/$shell_profile" <<'EOF'

# bvm session override support (temporary `bvm use <version>`)
function bvm
	if test "$argv[1]" = use -a "$argv[2]" = --reset
		set -e BVM_VERSION
		command bvm use
		return
	end
	if test "$argv[1]" = use; and test -n "$argv[2]"; and not string match -q -e "--save" "$argv[2]"; and test "$argv[2]" != -s; and test "$argv[2]" != default
		set -gx BVM_VERSION "$argv[2]"
		command bvm use
		return
	end
	if test "$argv[1]" = use -a "$argv[2]" = default
		set -e BVM_VERSION
	end
	command bvm $argv
end
EOF
	fi
;;
*)
	if ! grep -q "bvm session override support" "$HOME/$shell_profile" 2>/dev/null; then
		cat >>"$HOME/$shell_profile" <<'EOF'

# bvm session override support (temporary `bvm use <version>`)
bvm() {
	if [ "$1" = "use" ] && [ "$2" = "--reset" ]; then
		unset BVM_VERSION
		command bvm use
		return
	fi
	if [ "$1" = "use" ] && [ -n "$2" ] && [ "$2" != "--save" ] && [ "$2" != "-s" ] && [ "$2" != "default" ]; then
		export BVM_VERSION="$2"
		command bvm use
		return
	fi
	if [ "$1" = "use" ] && [ "$2" = "default" ]; then
		unset BVM_VERSION
	fi
	command bvm "$@"
}
EOF
	fi
;;
esac
