# Installs or updates claude-accounts:
#   irm https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.ps1 | iex
$ErrorActionPreference = 'Stop'

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$url = if ($env:CLAUDE_ACCOUNTS_URL) { $env:CLAUDE_ACCOUNTS_URL } else {
    "https://github.com/MFpizza/claude-dotfiles/releases/latest/download/claude-accounts-windows-$arch.exe"
}
$dir = if ($env:CLAUDE_ACCOUNTS_DIR) { $env:CLAUDE_ACCOUNTS_DIR } else { Join-Path $HOME '.local\bin' }
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$exe = Join-Path $dir 'claude-accounts.exe'
$tmp = "$exe.download"

Write-Host "Downloading $url"
Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $tmp
# A running copy can't be overwritten, but it can be renamed out of the way.
if (Test-Path $exe) {
    Move-Item -Force $exe "$exe.old"
    Remove-Item -Force "$exe.old" -ErrorAction SilentlyContinue
}
Move-Item -Force $tmp $exe

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not (($userPath -split ';') -contains $dir)) {
    [Environment]::SetEnvironmentVariable('Path', "$dir;$userPath", 'User')
    $env:Path = "$dir;$env:Path"
    Write-Host "Added $dir to your PATH"
}

& $exe
