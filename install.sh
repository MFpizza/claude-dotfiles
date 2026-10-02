#!/bin/sh
# Installs or updates claude-accounts:
#   curl -fsSL https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.sh | sh
set -eu

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) echo "Unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac

url="${CLAUDE_ACCOUNTS_URL:-https://github.com/MFpizza/claude-dotfiles/releases/latest/download/claude-accounts-$os-$arch}"
dir="${CLAUDE_ACCOUNTS_DIR:-$HOME/.local/bin}"
mkdir -p "$dir"
tmp="$dir/.claude-accounts.download"
echo "Downloading $url"
curl -fsSL -o "$tmp" "$url"
chmod +x "$tmp"
mv -f "$tmp" "$dir/claude-accounts"

case ":$PATH:" in
    *":$dir:"*) ;;
    *) echo "Add $dir to your PATH, e.g. in ~/.bashrc:  export PATH=\"$dir:\$PATH\"" ;;
esac

# Piped into sh, stdin is this script; the menu needs the terminal instead.
if (exec </dev/tty) 2>/dev/null; then
    "$dir/claude-accounts" </dev/tty
else
    "$dir/claude-accounts" list
fi
