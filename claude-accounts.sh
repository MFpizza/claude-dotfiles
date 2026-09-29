# Claude Code multiple accounts (bash/zsh). Sourced from the rc file by install.py.
# Account A: $CLAUDE_DOTFILES_BASE (default ~/.claude)   B: <A>-b   C: <A>-c (API billing)
# Shared via symlinks: projects, skills, agents, commands, plugins, file-history, sessions.
# Per account: credentials and .claude.json. settings.json is copied A -> B/C on launch;
# <dir>/account-settings.json (if any) is layered on top via --settings and never
# overwritten, e.g. env with a gateway URL and token for the API-billed account.
# Runs `claude` from PATH; if it isn't there (installed per project and started with
# `npx claude`), falls back to `npx --no -- claude`. Set CLAUDE_BIN to override.

: "${CLAUDE_DOTFILES_BASE:=$HOME/.claude}"

_claude_run() {
    if [ -n "$CLAUDE_BIN" ]; then
        "$CLAUDE_BIN" "$@"
    elif command -v claude >/dev/null 2>&1; then
        command claude "$@"
    else
        # --no: never download a package from the registry, only run an installed one
        npx --no -- claude "$@"
    fi
}

_claude_account() {
    local dir="$1"
    shift
    if [ "$dir" != "$CLAUDE_DOTFILES_BASE" ]; then
        cp -f "$CLAUDE_DOTFILES_BASE/settings.json" "$dir/settings.json" 2>/dev/null
    fi
    if [ -f "$dir/account-settings.json" ]; then
        set -- --settings "$dir/account-settings.json" "$@"
    fi
    if [ "$dir" = "$HOME/.claude" ]; then
        # Default dir: leave the variable unset so claude-a and plain claude share
        # ~/.claude.json (with it set, Claude reads <dir>/.claude.json instead).
        (unset CLAUDE_CONFIG_DIR; _claude_run "$@")
    else
        CLAUDE_CONFIG_DIR="$dir" _claude_run "$@"
    fi
}

claude-a() { _claude_account "$CLAUDE_DOTFILES_BASE" "$@"; }

claude-b() { _claude_account "$CLAUDE_DOTFILES_BASE-b" "$@"; }

claude-c() { _claude_account "$CLAUDE_DOTFILES_BASE-c" "$@"; }
