# claude-dotfiles

Claude Code 雙帳號切換，以及同時顯示兩個帳號用量的狀態列。

```
● A Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱   3% ↻ 4h46m  │ 週 ▱▱▱▱▱▱▱▱▱▱   0% ↻ 1d16h  │ Opus 5.5
○ B Pro  │ 5h ▰▰▰▰▰▰▰▰▰▰ 100% ↻ 2h36m  │ 週 ▰▱▱▱▱▱▱▱▱▱  14% ↻ 6d11h
```

## 安裝（新電腦）

需求：Claude Code、Python 3.7+、git。

```powershell
git clone <這個 repo 的網址> claude-dotfiles
cd claude-dotfiles
python install.py
```

開一個新的 PowerShell 視窗，分別執行 `claude-a`、`claude-b`，各自在 Claude 裡輸入 `/login` 一次。

## 使用

| 指令 | 說明 |
|---|---|
| `claude-a` | 用帳號 A（`~/.claude`） |
| `claude-b` | 用帳號 B（`~/.claude-b`） |
| `claude-b --resume` | 用 B 接續任何一個帳號的 session |

- 共用：session、記憶、skills、agents、commands、plugins、file-history。
- 各自獨立：登入憑證、`.claude.json`（含使用者層級的 MCP 設定）。
- `settings.json` 以 A 為主，每次啟動 `claude-b` 時複製過去。

## 更新

```powershell
cd claude-dotfiles
git pull
```

狀態列與 PowerShell 指令都直接引用這個資料夾的檔案，拉下來就生效。**搬移這個資料夾後要重跑 `python install.py`。**

## 檔案

| 檔案 | 用途 |
|---|---|
| `usage_statusline.py` | 狀態列：呼叫 `api/oauth/usage` 取兩個帳號用量，快取於 `~/.claude/usage-cache.json` |
| `claude-accounts.ps1` | `claude-a`／`claude-b` 指令，由 PowerShell profile 載入 |
| `install.py` | 建立 `~/.claude-b` 與共用連結、設定 statusLine、寫入 profile；可重複執行 |

## 注意

- `api/oauth/usage` 不是公開 API，Claude Code 改版後可能失效。
- 沒在使用的帳號 token 過期時，狀態列會替它刷新；正在執行的帳號由 Claude 自己刷新。
- 兩個帳號不要同時開同一個 session。
- repo 內沒有任何憑證；憑證只存在各電腦的 `~/.claude*/.credentials.json`。
