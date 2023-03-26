#!/usr/bin/env bash

set -e

if ! command -v unzip >/dev/null; then
	echo "Error: unzip is required to install bvm." 1>&2
	exit 1
fi

if [ "$OS" = "Windows_NT" ]; then
  echo "Error: bun requires Windows Subsystem for Linux." 1>&2
	exit 1
else
	case $(uname -sm) in
	"Darwin x86_64") target="darwin_x86_64" ;;
	"Darwin arm64") target="darwin_arm64" ;;
	"Linux x86_64") target="linux_x86_64" ;;
	*) echo "Unsupported OS + CPU combination: $(uname -sm)"; exit 1 ;;
	esac
fi

bvm_url="https://github.com/chathula/bvm/releases/latest/download/bvm_${target}.zip"

bvm_dir="${BVM_DIR:-$HOME/.bvm}"
bvm_bin_dir="$bvm_dir/bin"
exe="$bvm_bin_dir/bvm"

if [ ! -d "$bvm_bin_dir" ]; then
	mkdir -p "$bvm_bin_dir"
fi

if [ "$1" = "" ]; then
	cd "$bvm_bin_dir"
	curl --fail --location --progress-bar -k --output "$exe.zip" "$bvm_url"
	unzip -o "$exe.zip"
	rm "$exe.zip"
else
	echo "Install path override detected: $1"
	if [ ! -f "$1" ]; then
		echo "File does not exist: $1"
		exit 1
	fi
	cp "$1" "$exe"
fi
cd "$bvm_bin_dir"
chmod +x "$exe"

case $SHELL in
/bin/zsh) shell_profile=".zshrc" ;;
*) shell_profile=".basrc" ;;
esac

if [ ! $BVM_DIR ];then
    command echo "export BVM_DIR=\"$bvm_dir\"" >> "$HOME/$shell_profile"
    command echo "export PATH=\"\$BVM_DIR/bin:\$PATH\"" >> "$HOME/$shell_profile"
fi

echo "bvm was installed successfully to $exe"
if command -v bvm >/dev/null; then
	echo "Run 'bvm --help' to get started."
else
	echo "Reopen your shell, or run 'source $HOME/$shell_profile' to get started"
fi
