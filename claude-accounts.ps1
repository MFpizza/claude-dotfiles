# Claude Code multiple accounts. Dot-sourced from the PowerShell profile by install.py.
# Account A: ~/.claude (default)   Account B: ~/.claude-b   Account C: ~/.claude-c (API billing)
# Shared via junctions: projects, skills, agents, commands, plugins, file-history, sessions.
# Per account: credentials and .claude.json. settings.json is copied A -> B/C on launch.

function Get-ClaudeExe {
    (Get-Command claude -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
}

function Invoke-ClaudeAccount([string]$dir, [object[]]$rest) {
    Copy-Item (Join-Path $HOME '.claude\settings.json') (Join-Path $dir 'settings.json') -Force -ErrorAction SilentlyContinue
    $env:CLAUDE_CONFIG_DIR = $dir
    try { & (Get-ClaudeExe) @rest }
    finally { Remove-Item Env:CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue }
}

function claude-a {
    Remove-Item Env:CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue
    & (Get-ClaudeExe) @args
}

function claude-b { Invoke-ClaudeAccount (Join-Path $HOME '.claude-b') $args }

function claude-c { Invoke-ClaudeAccount (Join-Path $HOME '.claude-c') $args }
