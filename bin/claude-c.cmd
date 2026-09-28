@echo off
rem Claude Code account C (~/.claude-c, API billing). cmd.exe counterpart of claude-c in claude-accounts.ps1.
setlocal
set "CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-c"
copy /Y "%USERPROFILE%\.claude\settings.json" "%CLAUDE_CONFIG_DIR%\settings.json" >nul 2>&1
claude %*
