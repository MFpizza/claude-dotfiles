@echo off
rem Claude Code account B (~/.claude-b). cmd.exe counterpart of claude-b in claude-accounts.ps1.
setlocal
set "CLAUDE_CONFIG_DIR=%USERPROFILE%\.claude-b"
copy /Y "%USERPROFILE%\.claude\settings.json" "%CLAUDE_CONFIG_DIR%\settings.json" >nul 2>&1
set "OWN="
if exist "%CLAUDE_CONFIG_DIR%\account-settings.json" set OWN=--settings "%CLAUDE_CONFIG_DIR%\account-settings.json"
claude %OWN% %*
