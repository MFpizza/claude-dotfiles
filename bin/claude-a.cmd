@echo off
rem Claude Code account A (~/.claude). cmd.exe counterpart of claude-a in claude-accounts.ps1.
setlocal
set "CLAUDE_CONFIG_DIR="
claude %*
