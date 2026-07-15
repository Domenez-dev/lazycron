#!/bin/bash
# run this command to excute this script and installs lazycron on your system
# for only current user access
# curl -fsSL "https://raw.githubusercontent.com/Domenez-dev/lazycron/main/install.sh" | bash
#
# for system-wide access
# curl -fsSL "https://raw.githubusercontent.com/Domenez-dev/lazycron/main/install.sh" | sudo bash

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)


# get latest release tag name
tag=$(curl -fsSL "https://api.github.com/repos/Domenez-dev/lazycron/releases/latest" | grep "tag_name" | cut -d '"' -f 4)

# check if bin is writable
if [ -w "/usr/local/bin" ]; then install_dir="/usr/local/bin"; else mkdir -p "$HOME/.local/bin" && install_dir="$HOME/.local/bin"; fi
echo "Installing lazycron $tag for $os/$arch to $install_dir"


temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT

# Download lazycron binary
curl -fsSL "https://github.com/Domenez-dev/lazycron/releases/download/$tag/lazycron" -o "$temp_dir/lazycron"
chmod +x "$temp_dir/lazycron"
install -m 755 "$temp_dir/lazycron" "$install_dir/lazycron"

echo "lazycron $tag for $os/$arch installed to $install_dir"
echo "you can run it using the command: lazycron"
