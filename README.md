# claude-dotfiles

Claude Code 雙帳號切換，加上一條同時顯示兩個帳號用量的狀態列。

```
● A Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱  12% ↻ 4h39m  │ 週 ▰▱▱▱▱▱▱▱▱▱   2% ↻ 1d16h  │ Opus 5.5
○ B Pro  │ 5h ▰▰▰▰▰▰▰▰▰▰ 100% ↻ 2h29m  │ 週 ▰▱▱▱▱▱▱▱▱▱  14% ↻ 6d11h
```

- 兩個帳號**共用** session、記憶、skills、agents：用 A 做到一半，額度用完就換 B 接著做。
- 狀態列隨時顯示**兩個帳號**的 5 小時與每週用量，包括沒在使用的那個帳號。

---

## 目錄

1. [安裝](#安裝)
2. [日常使用](#日常使用)
3. [看懂狀態列](#看懂狀態列)
4. [常見情境](#常見情境)
5. [共用與不共用的東西](#共用與不共用的東西)
6. [更新與修改](#更新與修改)
7. [疑難排解](#疑難排解)
8. [解除安裝](#解除安裝)
9. [檔案說明](#檔案說明)

---

## 安裝

### 需求

- Windows（PowerShell 或 cmd 皆可）
- [Claude Code](https://docs.claude.com/en/docs/claude-code)，`claude` 指令可在終端機直接執行
- Python 3.8 以上（只用標準函式庫，不必 `pip install`）
- git

### 步驟

```powershell
git clone https://github.com/TwZhon/claude-dotfiles.git
cd claude-dotfiles
python install.py
```

安裝完成時會看到：

```
安裝 claude-dotfiles …
  ✓ ~/.claude-b 共用 projects, skills, agents, commands, plugins, file-history, sessions
  ✓ settings.json 已設定 statusLine（原檔備份為 settings.json.bak）
  ✓ PowerShell profile：C:\Users\<你>\Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1
  ✓ cmd.exe 指令：C:\Users\<你>\.local\bin\claude-a.cmd、claude-b.cmd
```

### 第一次登入

**開一個新的 PowerShell 或 cmd 視窗**，才會載入新指令。

```powershell
claude-a      # 進入後輸入 /login，登入第一個帳號
claude-b      # 進入後輸入 /login，登入第二個帳號
```

每台電腦的每個帳號只需要登入一次。

> 狀態列會使用「執行 `install.py` 的那個 Python」。電腦上有多個 Python 時，請用你想固定使用的那個來執行安裝。

---

## 日常使用

| 指令 | 說明 |
|---|---|
| `claude-a` | 用帳號 A 開啟 Claude |
| `claude-b` | 用帳號 B 開啟 Claude |
| `claude-a --resume` | 用帳號 A 挑一個舊 session 接續 |
| `claude-b --continue` | 用帳號 B 接續這個資料夾最近的 session |

`claude-a`、`claude-b` 後面可以接任何原本 `claude` 的參數，例如：

```powershell
claude-b --agent implementer
claude-a -p "幫我摘要這個 repo"
```

原本的 `claude` 指令仍然可用，等同於 `claude-a`。

---

## 看懂狀態列

```
● A Pro  │ 5h ▰▰▰▱▱▱▱▱▱▱  31% ↻ 2h05m  │ 週 ▰▱▱▱▱▱▱▱▱▱   8% ↻ 3d04h  │ Opus 5.5
└┬┘ └┬┘    └┬┘ └────┬───┘ └┬┘ └──┬──┘    └┬┘                                 └──┬───┘
 │   │      │       │      │     │       每週額度（欄位同左）                   目前模型
 │   │      │       │      │     └ 距離重置還有多久
 │   │      │       │      └ 已使用百分比
 │   │      │       └ 用量條（每格 10%）
 │   │      └ 5 小時額度
 │   └ 訂閱方案
 └ ● 目前這個視窗用的帳號；○ 另一個帳號
```

### 顏色

| 顏色 | 用量 | 意思 |
|---|---|---|
| 藍 | 0–49% | 充足 |
| 琥珀 | 50–79% | 注意 |
| 橘 | 80–100% | 快用完，準備換帳號 |

配色避開紅綠，對色弱友善（搭配 Claude 的 `dark-daltonized` 主題）。

### 時間格式

| 顯示 | 意思 |
|---|---|
| `↻ 43m` | 43 分鐘後重置 |
| `↻ 2h05m` | 2 小時 5 分後重置 |
| `↻ 3d04h` | 3 天 4 小時後重置 |

### 特殊狀態

| 顯示 | 意思 | 處理 |
|---|---|---|
| `│ 快取 · 3 天前` | 抓不到最新資料，顯示的是最後一次成功的數值 | 通常是網路問題、token 失效或已退訂；不處理也沒關係 |
| `未登入 · 執行 claude-b 後 /login` | 這個帳號在這台電腦還沒登入 | 照提示登入 |
| `token 過期` | 目前視窗的帳號 token 剛好過期 | 送出任何訊息後 Claude 會自動刷新 |
| `HTTP 401` / `HTTP 403` | 伺服器拒絕，多半是登入失效 | 用該帳號執行 `/login` |
| `5h ▱▱▱▱▱▱▱▱▱▱    —` | 伺服器沒有回傳這個額度 | 常見於未訂閱的帳號 |

**帳號退訂後**，狀態列仍會顯示最後一次抓到的數值並標上「快取」；若該額度的重置時間已過，會直接顯示 0%。

用量每 2 分鐘最多查詢一次，資料快取在 `~/.claude/usage-cache.json`。

---

## 常見情境

### A 的 5 小時額度用完了，想換 B 繼續同一份工作

1. 在 A 的視窗按 `Ctrl+C` 兩次（或輸入 `/exit`）離開。
2. 在同一個資料夾執行：

   ```powershell
   claude-b --continue
   ```

B 會接著 A 最後的對話繼續，前面的上下文都在。

### 想挑一個比較舊的 session

```powershell
claude-b --resume
```

清單裡會同時看到 A 和 B 開過的 session。

### 同時開兩個視窗，一個 A 一個 B

可以，兩個帳號的額度各自計算。**但不要讓兩個視窗同時開啟同一個 session**，對話紀錄會互相覆蓋。

A 和 B 的 session 可以用 ListAgents 互相看到，也能用 SendMessage 互傳訊息（本機 session 才行；claude.ai 上的 Remote Control session 仍只看得到同一個帳號的）。

### 新增 skill 或 agent

放在 `~/.claude/skills/` 或 `~/.claude/agents/`，兩個帳號都會看到。專案內的 `.claude/agents/` 本來就跟著 repo 走，兩個帳號也都看得到。

### 修改 Claude 設定（主題、模型、hooks…）

在 `claude-a` 裡用 `/config` 修改，或直接編輯 `~/.claude/settings.json`。下次啟動 `claude-b` 時會自動同步過去。

> 在 `claude-b` 裡改的設定**不會**回傳給 A，而且下次啟動 B 時會被 A 的設定覆蓋。

---

## 共用與不共用的東西

| 項目 | 位置 | 兩帳號之間 |
|---|---|---|
| session 對話紀錄、記憶 | `projects/` | 共用 |
| skills | `skills/` | 共用 |
| 使用者層級 agents | `agents/` | 共用 |
| 自訂斜線指令 | `commands/` | 共用 |
| plugins | `plugins/` | 共用 |
| 檔案編輯紀錄（還原用） | `file-history/` | 共用 |
| 執行中 session 的登記（ListAgents、SendMessage 用） | `sessions/` | 共用 |
| 設定 | `settings.json` | A → B 單向同步 |
| 登入憑證 | `.credentials.json` | 各自獨立 |
| 帳號資訊、使用者層級 MCP 設定 | `.claude.json` | 各自獨立 |
| 輸入歷史（↑ 鍵） | `history.jsonl` | 各自獨立 |

A 用 `~/.claude`，B 用 `~/.claude-b`。B 裡面共用的資料夾是指向 A 的 junction（目錄連結），資料實際上只存一份。

---

## 更新與修改

### 其他電腦取得最新版

```powershell
cd claude-dotfiles
git pull
```

狀態列與 `claude-a`／`claude-b` 都直接讀這個資料夾裡的檔案，**拉下來就生效**，不必重新安裝。

例外是新增了共用資料夾的版本（例如加入 `sessions/` 共用那一版），需要**先關掉所有 `claude-b` 視窗**，再重跑：

```powershell
python install.py
```

B 原本獨立的資料夾會被併入 A 的對應資料夾，再換成 junction。

### 修改設計

改 `usage_statusline.py` 後可以先在終端機預覽：

```powershell
'{}' | python usage_statusline.py
```

常改的地方都在檔案開頭：

| 想改的 | 變數 |
|---|---|
| 用量條長度 | `BAR_WIDTH` |
| 顏色 | `LOW`、`MID`、`HIGH`、`ACCENT` 等（`#RRGGBB`） |
| 帳號顯示名稱 | `ACCOUNTS` 裡的 `"A"`、`"B"` |
| 查詢間隔（秒） | `CACHE_TTL` |

改好後 commit 並 push，其他電腦 `git pull` 即可。

### 搬移這個資料夾

settings.json 與 PowerShell profile 都記錄了這個資料夾的絕對路徑，搬移後重新執行：

```powershell
python install.py
```

`install.py` 可以重複執行，不會產生重複的設定。

---

## 疑難排解

**找不到 `claude-a` 指令**
- 確認是**新開的** PowerShell 視窗。
- 執行 `notepad $PROFILE`，確認裡面有 `# >>> claude-dotfiles >>>` 區塊。
- 若出現「因為這個系統上已停用指令碼執行」：

  ```powershell
  Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
  ```

- 如果你用的是 PowerShell 7（`pwsh`），而安裝時它還沒裝，請重跑 `python install.py`。
- 在 cmd 裡：確認 `~/.local/bin` 裡有 `claude-a.cmd`、`claude-b.cmd`，且這個資料夾在 PATH 上（Claude Code 安裝時會加入）。沒有的話重跑 `python install.py`。

**狀態列沒出現**
- 重新啟動 Claude。
- 確認 `~/.claude/settings.json` 裡有 `statusLine`，且其中的 Python 路徑存在。
- 手動執行預覽指令（見[修改設計](#修改設計)），看是否有錯誤訊息。

**狀態列出現亂碼（方塊、問號）**
- 換用支援 Unicode 的終端機，例如 Windows Terminal。
- 字型需要有 `▰ ▱ ● ○ ↻ │` 這些字元，例如 Cascadia Code。

**建立 junction 失敗**
- `install.py` 會把 `~/.claude-b` 裡的一般資料夾併入 `~/.claude` 的對應位置。若出現「有同名的項目，未建立連結」，代表兩邊有同名檔案，自動合併時不會覆蓋。請手動決定保留哪一份，把 `~/.claude-b` 裡的那個資料夾清空並刪除後，再重跑 `install.py`。

**B 的設定跟 A 不一樣**
- B 的 `settings.json` 每次啟動 `claude-b` 時才會從 A 複製。若直接執行 `$env:CLAUDE_CONFIG_DIR=...; claude`，就不會同步。

---

## 解除安裝

1. 執行 `notepad $PROFILE`，刪除 `# >>> claude-dotfiles >>>` 到 `# <<< claude-dotfiles <<<` 之間的內容。
2. 編輯 `~/.claude/settings.json`，刪除 `statusLine` 區塊（或用 `settings.json.bak` 還原）。
3. 刪除 `~/.claude-b`，不再需要帳號 B 的話：

   ```powershell
   cmd /c rmdir /s /q "$HOME\.claude-b"
   ```

   `rmdir` 只會移除 junction 本身，**不會**刪到 `~/.claude` 裡共用的資料。
   請不要用檔案總管或 `Remove-Item -Recurse` 刪除，某些情況下會連同目標資料一起刪除。
4. 刪除 `~/.claude/usage-cache.json`。
5. 刪除 `~/.local/bin/claude-a.cmd`、`claude-b.cmd`。

---

## 檔案說明

| 檔案 | 用途 |
|---|---|
| `usage_statusline.py` | 狀態列。以各帳號自己的 OAuth token 呼叫 `api/oauth/usage` 取得用量並快取 |
| `claude-accounts.ps1` | 定義 `claude-a`、`claude-b`，由 PowerShell profile 載入 |
| `bin/claude-a.cmd`、`bin/claude-b.cmd` | cmd.exe 版的 `claude-a`、`claude-b`；install.py 在 `~/.local/bin` 放轉呼叫它們的小檔 |
| `install.py` | 建立 `~/.claude-b` 與共用連結、設定 statusLine、寫入 PowerShell profile、放 cmd 指令 |

### 安全與限制

- repo 內**沒有**任何憑證；憑證只存在各電腦的 `~/.claude*/.credentials.json`，也不應該被加進 repo。
- `api/oauth/usage` 是 Claude Code 內部使用的端點，不是公開 API，Claude Code 改版後可能失效。
- 沒在使用的帳號 token 過期時，狀態列會替它刷新；正在執行的帳號由 Claude 自己刷新，避免互相衝突。
