# Claude Code multiple accounts (bash/zsh). Sourced from the rc file by install.py.
# Account A: $CLAUDE_DOTFILES_BASE (default ~/.claude)   B: <A>-b   C: <A>-c (API billing)
# Shared via symlinks: projects, skills, agents, commands, plugins, file-history, sessions.
# Per account: credentials and .claude.json. settings.json is copied A -> B/C on launch.

: "${CLAUDE_DOTFILES_BASE:=$HOME/.claude}"

_claude_account() {
    local dir="$1"
    shift
    if [ "$dir" != "$CLAUDE_DOTFILES_BASE" ]; then
        cp -f "$CLAUDE_DOTFILES_BASE/settings.json" "$dir/settings.json" 2>/dev/null
    fi
    if [ "$dir" = "$HOME/.claude" ]; then
        # Default dir: leave the variable unset so claude-a and plain claude share
        # ~/.claude.json (with it set, Claude reads <dir>/.claude.json instead).
        (unset CLAUDE_CONFIG_DIR; command claude "$@")
    else
        CLAUDE_CONFIG_DIR="$dir" command claude "$@"
    fi
}

claude-a() { _claude_account "$CLAUDE_DOTFILES_BASE" "$@"; }

claude-b() { _claude_account "$CLAUDE_DOTFILES_BASE-b" "$@"; }

claude-c() { _claude_account "$CLAUDE_DOTFILES_BASE-c" "$@"; }
