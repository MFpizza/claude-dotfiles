# claude-accounts：以 Go 重寫成單一執行檔

日期：2026-10-02
狀態：設計已確認，待審閱規格

## 背景與目標

目前的 claude-dotfiles 是 Python 安裝腳本加上 shell／PowerShell／cmd 腳本，帳號固定為 A、B、C，這三個字母分散寫死在 5 個地方（`install.py`、`claude-accounts.sh`、`claude-accounts.ps1`、`bin/*.cmd`、`usage_statusline.py` 的 `ACCOUNTS`）。

repo 要公開給其他人使用，對方不一定有 Python。目標：

1. 打包成**單一執行檔**，下載即可使用，不需要任何執行環境。
2. **帳號數量不寫死**，可以自由新增、移除訂閱或 API 帳號。
3. **只有一個帳號也能用**：只裝狀態列（用量 + 小寵物），之後想加帳號再加。
4. 可以選擇**哪些帳號要出現在狀態列**。
5. 介面**預設英文，支援繁體中文**。
6. 既有使用者（A/B/C）**自動遷移**，寵物等資料保留。

### 非目標

- 不做程式碼簽章。
- 不做自我更新指令；重跑一行安裝指令即為更新。
- 不引入第三方 Go 套件或 TUI 函式庫。
- 不支援自訂帳號名稱，名稱一律是字母。

## 決策摘要

| 項目 | 決定 | 理由 |
|---|---|---|
| 語言 | Go，只用標準函式庫 | 單一靜態執行檔、跨平台編譯、啟動約 5ms（狀態列每次刷新都會執行） |
| 不選 PyInstaller | — | 單檔 exe 每次啟動要解壓（0.3–1 秒），Windows 防毒常誤判 |
| 帳號名稱 | 字母 a–z，主帳號固定 `a`，新增時自動分配最小的空字母 | 使用者偏好，沿用 A/B/C 慣例 |
| 帳號指令 | 執行檔的 hard link（Windows）／symlink（Linux、macOS），依 argv[0] 分派 | 不必改 `$PROFILE`／`.bashrc`；Windows 不再出現 `.cmd` 的「終止批次工作 (Y/N)?」 |
| 目前帳號的用量 | 優先使用狀態列輸入中的 `rate_limits` | 不必呼叫非公開 API；macOS 也能用 |
| 介面語言 | 預設英文；偵測到繁體中文語系時用繁中；可手動指定 | 公開給國際使用者，同時保留中文 |

## 1. 整體架構

### 執行檔與子指令

執行檔名稱 `claude-accounts`（Windows 為 `claude-accounts.exe`），安裝在 `~/.local/bin`。

| 執行方式 | 功能 |
|---|---|
| `claude-accounts` | 互動選單；尚未設定時進入首次設定（含遷移） |
| `claude-accounts add [--api] [--dir DIR]` | 新增帳號，自動分配字母 |
| `claude-accounts remove <字母>` | 移除帳號（需確認） |
| `claude-accounts list` | 列出帳號 |
| `claude-accounts show <字母>` / `hide <字母>` | 設定帳號是否顯示在狀態列 |
| `claude-accounts layout full\|compact` | 切換狀態列版型 |
| `claude-accounts lang en\|zh-TW\|auto` | 介面語言 |
| `claude-accounts statusline` | 由 Claude Code 呼叫，從 stdin 讀 JSON、輸出狀態列 |
| `claude-accounts uninstall` | 解除安裝 |

以 `claude-<字母>` 這個名字被呼叫時（例如 `claude-b`），所有參數原封不動轉給 claude，以該帳號啟動（見第 2 節）。

互動選單是編號選單，例如：

```
Accounts:
  a  subscription  ~/.claude        (status line: on)
  b  subscription  ~/.claude-b      (status line: on)

  1) Add account
  2) Remove account
  3) Show / hide in status line
  4) Status line layout (current: full)
  5) Language (current: auto)
  6) Uninstall
  0) Exit
```

### 設定檔 `claude-accounts.json`

位於主帳號目錄。主帳號目錄的決定順序：`$CLAUDE_CONFIG_DIR` → `~/.claude`。

```json
{
  "version": 1,
  "layout": "full",
  "lang": "auto",
  "accounts": [
    {"name": "a", "dir": "~/.claude",   "type": "subscription", "statusline": true},
    {"name": "b", "dir": "~/.claude-b", "type": "subscription", "statusline": true},
    {"name": "c", "dir": "~/.claude-c", "type": "api",          "statusline": false}
  ]
}
```

- `dir` 以 `~` 開頭時相對於家目錄，寫入時一律盡量使用 `~` 形式。
- `accounts[0]` 是主帳號（名稱固定 `a`）：共用資料實際存放處，不能移除。
- 只有一個帳號＝只用狀態列的模式。
- 舊版的 `statusline.json`（`{"layout": ...}`）在遷移時併入此檔後刪除。

### 執行檔所在位置

狀態列設定與帳號指令連結都需要執行檔的絕對路徑，以 `os.Executable()`（並解析 symlink）取得。帳號指令連結一律建立在**執行檔所在的目錄**：執行檔能被執行，代表這個目錄在 PATH 上；同一目錄也保證 hard link 在同一個磁碟區。執行任何管理子指令（含互動選單）時，都會檢查並修復：

- 每個非主帳號的 `claude-<字母>` 連結是否存在且指向目前的執行檔；Windows hard link 在執行檔被取代後會指向舊檔，因此一律重建。
- 主帳號的 `claude-a` 連結同樣建立。
- 主帳號 `settings.json` 的 `statusLine.command` 是否指向目前的執行檔。

## 2. 帳號的運作

### 啟動 `claude-<字母>`

1. 讀設定檔，找出該字母的帳號；找不到時印出錯誤，並提示可用的帳號。
2. 非主帳號：把主帳號的 `settings.json` 複製到該帳號目錄（單向同步）。
3. 該帳號目錄有 `account-settings.json` 時，在參數最前面加上 `--settings <路徑>`。
4. 決定 `CLAUDE_CONFIG_DIR`：帳號目錄等於 `~/.claude` 時**移除**這個變數（讓 `claude-a` 與直接執行 `claude` 共用 `~/.claude.json`），否則設為帳號目錄。
5. 找 claude：`$CLAUDE_BIN` → PATH 上的 `claude`（排除自己）→ `npx --no -- claude`。
6. 啟動：Linux／macOS 用 `syscall.Exec` 取代目前行程；Windows 以子行程執行，父行程忽略 Ctrl+C，等待結束後回傳相同的退出碼。

### `add`

1. 分配 `b`–`z` 中最小、未使用的字母；全滿時報錯。
2. 目錄預設為 `<主帳號目錄>-<字母>`，可用 `--dir` 指定。
3. 建立目錄，並把共用資料夾連結到主帳號：`projects`、`skills`、`agents`、`commands`、`plugins`、`file-history`、`sessions`（Windows 用 junction，其他平台用 symlink）。
   - 目錄裡已有同名的實體資料夾時，先把內容併入主帳號（不覆蓋任何既有檔案）再建立連結；有衝突時保留原狀並警告。
4. `--api` 時 `type` 為 `api`，否則為 `subscription`。`statusline` 預設為 `true`。
5. 建立 `claude-<字母>` 指令連結，寫入設定檔。
6. 提示：執行 `claude-<字母>`，再輸入 `/login`（API 帳號選 Anthropic Console；走 gateway 的話改用 `account-settings.json`）。

### `remove <字母>`

1. 不能移除 `a`。
2. 確認後，移除 `claude-<字母>` 指令連結，並從設定檔刪除這個帳號。
3. 詢問是否刪除帳號目錄（預設：否）。要刪除時，**先逐一移除目錄最上層的 junction／symlink，再刪除目錄**，絕不跟隨連結刪到主帳號的資料。
4. 狀態列快取中該帳號的資料一併清除。

### 從舊版遷移

首次執行（主帳號目錄沒有 `claude-accounts.json`）時：

1. 主帳號為 `$CLAUDE_CONFIG_DIR` 或 `~/.claude`，名稱 `a`。
2. 掃描 `<主帳號目錄>-<單一字母>` 形式的目錄，其中至少一個共用資料夾是指向主帳號的連結，就匯入成同字母的帳號。
3. 依舊版慣例，`c` 預設為 `api`，其餘為 `subscription`；互動模式下讓使用者確認類型。
4. 移除舊安裝（修改前先備份為 `.bak`）：
   - Windows PowerShell 與 PowerShell 7 的 `$PROFILE` 中 `# >>> claude-dotfiles >>>` 到 `# <<< claude-dotfiles <<<` 的區塊。
   - `~/.bashrc`、`~/.zshrc` 中的同一區塊（舊版的 shell 函式會蓋過新的指令）。
   - `~/.local/bin/claude-a.cmd`、`claude-b.cmd`、`claude-c.cmd`。
5. `statusline.json` 的 layout 併入新設定檔。
6. 寵物存檔（`statusline-pet.json`）、用量快取（`usage-cache.json`）、API 花費紀錄（`api-cost.json`）的位置與格式不變，直接沿用。
7. 設定主帳號 `settings.json` 的 `statusLine`（先備份為 `settings.json.bak`）。

## 3. 狀態列

### 輸入與輸出

Claude Code 透過 stdin 傳入 JSON。用到的欄位：`session_id`、`model.display_name`、`context_window`（`used_percentage`，或由 `current_usage` 與 `context_window_size` 計算）、`cost.total_cost_usd`、`rate_limits.five_hour` 與 `rate_limits.seven_day`（`used_percentage`、`resets_at` 為 Unix 秒）。

目前帳號：取 `$CLAUDE_CONFIG_DIR`（未設則為 `~/.claude`），比對設定檔中各帳號的目錄。

### 用量來源

| 帳號 | 來源 |
|---|---|
| 目前帳號（訂閱） | 輸入中的 `rate_limits`；沒有時改呼叫 API |
| 其他訂閱帳號 | 讀 `<dir>/.credentials.json` 的 OAuth token，呼叫 `https://api.anthropic.com/api/oauth/usage`（header `anthropic-beta: oauth-2025-04-20`），每帳號最多每 120 秒查一次；token 將在 60 秒內過期時，以 refresh token 向 `https://platform.claude.com/v1/oauth/token` 刷新（client id 沿用現行程式），並寫回憑證檔。目前帳號不做刷新，交給 Claude 自己處理 |
| API 帳號 | 依 `session_id` 累計 `cost.total_cost_usd` 到 `api-cost.json`（續接 session 時花費歸零的處理沿用現行邏輯），顯示本次、今日、本月 |

- 多個帳號的 API 查詢**並行**執行，每個請求逾時 4 秒。
- 每次取得目前帳號的 `rate_limits`，都寫入該帳號的快取；因此讀不到憑證的平台（macOS 鑰匙圈）也能顯示其他帳號最後一次的數值。
- 快取存在主帳號目錄的 `usage-cache.json`，鍵改用帳號字母（小寫）。舊版的鍵是大寫 `"A"`、`"B"`，讀取時要相容。

### 顯示規則

- 顯示 `statusline: true` 的帳號；**目前帳號一律顯示**。
- 完整版：每個帳號一行，模型與寵物接在第一行後面。緊湊版：全部擠成一行。各區塊的格式、配色（藍 < 50% ≤ 琥珀 < 80% ≤ 橘）、重置倒數、快取標記、錯誤提示（未登入、token 過期、HTTP 錯誤、無資料）與現行 Python 版一致。
- 狀態列在任何錯誤下都必須輸出內容：個別帳號失敗時顯示錯誤；整體出錯時，至少輸出寵物與錯誤摘要；退出碼一律為 0。

### 小寵物

邏輯與存檔格式完全沿用現行 Python 版（commit 7271762）：

- 經驗值是活躍時數：與上次刷新間隔小於 300 秒才累加。
- 一世 150 小時，等級 = 1 + floor(19 × ∛(時數 / 150))，上限 20。
- 滿 150 小時轉生：`life + 1`，超出的時數帶到下一世，`past` 記錄 `{line, branch, ended}`，`⭐` 數量為 `life − 1`（超過 3 顆顯示成 `⭐N`）。
- 下一世隨機挑選一系，優先挑還有未收集最終型態的系；分支同樣優先挑未收集的；全部收集完後，挑目前這一系以外的任一系。
- 7 系、16 種最終型態，定義與目前的 `PET_LINES` 相同。emoji 限 Unicode 12 以內，不使用 ZWJ 組合。
- 升級或轉生後 600 秒內顯示 `🎉` 與亮色等級。
- 心情依 context 用量：< 30% ✨、30–59% 無、60–84% 💦、≥ 85% 💤。
- 舊版以 session 數計算的存檔，以及已不存在的 `dragon` 系存檔，處理方式與現行相同。
- 存檔寫入使用「暫存檔 + rename」，暫存檔名含 PID，避免多個 session 同時寫入時衝突。

### 文字（英文／繁體中文）

| 英文 | 繁體中文 |
|---|---|
| `wk` | `週` |
| `today` / `month` / `session` | `今日` / `本月` / `本次` |
| `cached · 3d ago` | `快取 · 3 天前` |
| `not logged in · run claude-b, then /login` | `未登入 · 執行 claude-b 後 /login` |
| `token expired` | `token 過期` |
| `no data` | `無資料` |

語言判斷：設定檔 `lang` 為 `en` 或 `zh-TW` 時直接使用；為 `auto` 時，依 `LC_ALL` → `LC_MESSAGES` → `LANG`（Windows 則用使用者介面語言）判斷，`zh_TW`、`zh_HK`、`zh_Hant` 開頭的語系使用繁中，其餘一律英文。

## 4. 專案結構、發佈與測試

### 目錄

```
go.mod                          module github.com/MFpizza/claude-dotfiles
cmd/claude-accounts/main.go     依 argv[0] 分派：claude-<字母> → launch；否則為子指令／選單
internal/config                 設定檔讀寫、字母分配、路徑展開
internal/accounts               add / remove、共用資料夾連結、指令連結、修復
internal/launch                 啟動 claude
internal/migrate                舊版偵測與遷移
internal/statusline             輸入解析、用量查詢、快取、API 花費、排版
internal/pet                    小寵物
internal/i18n                   字串表與語系判斷
internal/menu                   互動選單
install.ps1、install.sh         一行安裝腳本
.github/workflows/              CI 測試與 Release
README.md、README.zh-TW.md      英文與繁中說明
```

移植完成後，刪除 `install.py`、`usage_statusline.py`、`claude-accounts.ps1`、`claude-accounts.sh`、`bin/`。

### 發佈

- 推送 `v*` tag 時，GitHub Actions 編譯 `windows`、`linux`、`darwin` × `amd64`、`arm64`，檔名 `claude-accounts-<os>-<arch>[.exe]`，上傳到 GitHub Releases。
- `install.ps1`／`install.sh`：偵測平台，下載最新 Release 到 `~/.local/bin/claude-accounts[.exe]`。`~/.local/bin` 不在 PATH 上時：Windows 加入使用者 PATH，Linux／macOS 則印出提示。最後執行 `claude-accounts` 進入首次設定。重跑同一行即為更新；更新時會重建帳號指令連結。
- 不做程式碼簽章；README 說明以瀏覽器下載時可能出現 SmartScreen 警告。

### 測試

- **單元測試**：字母分配（含補空字母與 26 個上限）、設定檔讀寫與 `~` 展開、遷移偵測（在暫存目錄模擬舊版 A/B/C 與 rc 檔區塊）、寵物（等級邊界、遷移、16 世收集完、不連續重複、🎉 時效）、API 花費累計、語系判斷。
- **狀態列輸出**：固定時間、假快取與假輸入，比對完整版與緊湊版的輸出（去除 ANSI 色碼），中英文各一組。
- **安全**：`remove` 並刪除目錄之後，主帳號的共用資料必須完整。
- **CI**：在 `ubuntu-latest` 與 `windows-latest` 上執行 `go test ./...`；junction 與 hard link 的測試只會在 Windows 上執行。
- **本機**：開發機為 Linux，Windows 行為靠 CI 驗證。

## 風險

- `api/oauth/usage` 與 OAuth token 端點不是公開 API，可能改版失效；目前帳號改用 `rate_limits` 後，影響只剩其他帳號的即時數值。
- 狀態列輸入的 `rate_limits` 欄位格式可能變動；解析失敗時退回 API。
- Windows 上若 hard link 建立失敗（例如檔案系統不支援），退而複製執行檔，並在每次執行管理指令時更新這些複本。
