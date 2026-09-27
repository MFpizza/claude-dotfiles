# Claude Code dual accounts. Dot-sourced from the PowerShell profile by install.py.
# Account A: ~/.claude (default)   Account B: ~/.claude-b
# Shared via junctions: projects, skills, agents, commands, plugins, file-history.
# Per account: credentials and .claude.json. settings.json is copied A -> B on launch.

function Get-ClaudeExe {
    (Get-Command claude -CommandType Application -ErrorAction Stop | Select-Object -First 1).Source
}

function claude-a {
    Remove-Item Env:CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue
    & (Get-ClaudeExe) @args
}

function claude-b {
    $b = Join-Path $HOME '.claude-b'
    Copy-Item (Join-Path $HOME '.claude\settings.json') (Join-Path $b 'settings.json') -Force -ErrorAction SilentlyContinue
    $env:CLAUDE_CONFIG_DIR = $b
    try { & (Get-ClaudeExe) @args }
    finally { Remove-Item Env:CLAUDE_CONFIG_DIR -ErrorAction SilentlyContinue }
}
