#!/usr/bin/env bash

set -e

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
	curl --fail --location --progress-bar -o "$exe.zip" "$bvm_url"
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

case $SHELL in
*/zsh) shell_profile=".zshrc" ;;
*/fish) shell_profile=".config/fish/config.fish" ;;
*) shell_profile=".bashrc" ;;
esac

if [ ! $BVM_DIR ]; then
	case $shell_profile in
	.config/fish/config.fish)
		{
			echo -e '\n# bvm & bun'
			command echo "set --export BVM_DIR \"$bvm_dir\""
			command echo "fish_add_path \"\$BVM_DIR/bin\""
		} >>"$HOME/$shell_profile"
		;;
	*)
		{
			echo -e '\n# bvm & bun'
			command echo "export BVM_DIR=\"$bvm_dir\""
			command echo "export PATH=\"\$BVM_DIR/bin:\$PATH\""
		} >>"$HOME/$shell_profile"
		;;
	esac
fi

echo "bvm was installed successfully to $exe"
if command -v bvm >/dev/null; then
	echo "Run 'bvm --help' to get started."
else
	echo "Reopen your shell, or run 'source $HOME/$shell_profile' to get started"
fi
