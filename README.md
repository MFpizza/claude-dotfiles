# claude-dotfiles

**Use several Claude Code accounts side by side. When one runs out, switch and keep the same conversation.**

```
● A Pro  │ 5h ▰▰▰▰▰▰▰▰▰▱  92% ↻ 0h41m  │ wk ▰▰▰▰▰▰▰▱▱▱  67% ↻ 1d16h  │ Opus 5.5 │ 🐔✨ Lv7
○ B Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱  13% ↻ 3h46m  │ wk ▰▰▰▰▱▱▱▱▱▱  44% ↻ 3d21h
```

A's 5-hour limit almost gone? Quit and run `claude-b --continue`: B picks up right where A stopped.

[繁體中文說明](README.zh-TW.md)

## Features

- 🔄 **Switch without losing anything**: all accounts share conversations, memory, skills and agents.
- 📊 **Every account's usage at a glance**: 5-hour and weekly limits with reset countdowns, even for accounts you aren't using. Turns orange when it's nearly used up.
- ➕ **As many accounts as you like**: subscription (Pro/Max) or API-billed. Only one account? You still get the status line.
- 🐣 **A pet that grows** with the time you spend in Claude, then is reborn as something else: 16 final forms to collect.
- 📏 **Two layouts**: one line per account, or everything on one line:

  ```
  ● A Pro 5h ▰▰▰▰▰▰▰▰▰▱ 92% ↻0h41m wk 67% │ ○ B Pro 5h ▰▱▱▱▱▱▱▱▱▱ 13% wk 44% │ Opus 5.5 │ 🐔✨ Lv7
  ```
- 🌐 English and Traditional Chinese, picked from your system language.
- 📦 One file, nothing else to install. Windows, Linux and macOS.

## Install

```powershell
irm https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.ps1 | iex     # Windows
```
```bash
curl -fsSL https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.sh | sh  # Linux, macOS
```

This puts `claude-accounts` in `~/.local/bin` and opens its menu. Run the same line again to update.

## Usage

| To | Run |
|---|---|
| Open the menu | `claude-accounts` |
| Add an account (gets the next letter: b, c, …) | `claude-accounts add` (API-billed: `add --api`) |
| Start Claude with account b | `claude-b` (takes every `claude` option) |
| **Continue A's conversation with B** | quit A, then `claude-b --continue` in the same folder |
| Pick an older conversation | `claude-b --resume` (lists every account's conversations) |
| Hide an account from the status line | `claude-accounts hide c` (`show c` to bring it back) |
| One-line status line | `claude-accounts layout compact` (`full` to go back) |
| Language | `claude-accounts lang zh-TW` (`en`, `auto`) |
| Remove an account | `claude-accounts remove c` |

The first time you use an account, type `/login` in Claude.

Good to know:
- Two windows on different accounts are fine; **two windows on the same conversation are not**, they overwrite each other.
- Settings flow one way: change them with `claude-a`, and the other accounts get them on their next start.
- An account's own settings, such as a gateway URL and token for an API account, go in `account-settings.json` in its folder:

  ```json
  { "env": { "ANTHROPIC_BASE_URL": "https://your-gateway/", "ANTHROPIC_AUTH_TOKEN": "…" } }
  ```

## Reading the status line

| Shows | Means |
|---|---|
| `●` / `○` | the account in this window / other accounts |
| `5h ▰▰▰▱▱▱▱▱▱▱ 31% ↻ 2h05m` | 5-hour limit: 31% used, resets in 2 h 5 min |
| blue / amber / orange | under 50% / under 80% / 80% or more |
| `cached · 3h ago` (compact: `*`) | couldn't refresh; showing the last values |
| `today $3.10 │ month $25.70` | an API account's spend on this computer (check the Console for your bill) |
| `not logged in · run claude-b, then /login` | that account isn't logged in on this computer |

### The pet

It levels up with active time (gaps over 5 minutes don't count): Lv7 takes about 5 hours, Lv20 150. At Lv20 it is reborn with a ⭐ as a random new kind, preferring ones you haven't raised. Its mood follows the conversation's context: ✨ fresh, 💦 tired, 💤 time to `/compact`.

| Line | Grows into |
|---|---|
| 🥚 Birds | 🐣 🐥 🐔 → 🐓 🦃 🦚 or 🐦 🦉 🦅 |
| 🥚 Water birds | 🐣 🐥 🦆 🐧 🦩 🦢 |
| 🥚 Reptiles | 🦎 🐢 🐍 🐊 → 🦕 🦖 or 🐲 🐉 |
| 🥚 Sea | 🦐 🐟 🐠 🐡 → 🦑 🐙 or 🦀 🦞 |
| 🥚 Insects | 🐛 🐜 🐞 🦗 🐝 🦋 |
| 🌰 Plants | 🌱 🌿 🍀 → 🌷 🌹 🌻, 🌳 🌸 🍒, 🌳 🍏 🍎 or 🍃 🍇 🍷 |
| 🍼 Mammals | 🐾 → 🐱 🐈 🐆 🐅 🦁, 🐶 🐕 🐺, 🦦 🐬 🐳 🐋 or 🐵 🙈 🐒 🦧 🦍 |

Each line ends crowned (👑) at Lv20.

## How it works

- Account a is `~/.claude` (or `$CLAUDE_CONFIG_DIR`); account x is `~/.claude-x`. The list lives in `claude-accounts.json` next to account a's settings.
- The other accounts link `projects`, `skills`, `agents`, `commands`, `plugins`, `file-history` and `sessions` to account a, so the data exists once. Logins and `.claude.json` stay per account.
- `claude-b` is the same program under another name (a hard link on Windows, a symlink elsewhere).
- The active account's usage comes from Claude Code itself. Other accounts are asked through `api/oauth/usage`, an internal Claude Code endpoint that may change without notice.
- Nothing in this repo holds credentials; they stay in each account's `.credentials.json`.

## Uninstall

`claude-accounts uninstall` removes the commands and the status line and keeps every account folder. Then delete `~/.local/bin/claude-accounts`. To delete an account folder too, use `claude-accounts remove <letter>` first, which removes its links before the folder so account a's data is never touched.

## Upgrading from the Python version

Install as above. The first run finds your accounts B and C, keeps your pet, and removes the old block from `$PROFILE` / `~/.bashrc` (backed up as `.bak`). Open a new terminal afterwards.
