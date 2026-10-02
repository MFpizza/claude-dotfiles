# claude-dotfiles

**兩個 Claude 訂閱帳號輪流用，額度用完一鍵換手，對話不中斷。**

```
● A Pro  │ 5h ▰▰▰▰▰▰▰▰▰▱  92% ↻ 0h41m  │ 週 ▰▰▰▰▰▰▰▱▱▱  67% ↻ 1d16h  │ Opus 5.5 │ 🐔✨ Lv7
○ B Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱  13% ↻ 3h46m  │ 週 ▰▰▰▰▱▱▱▱▱▱  44% ↻ 3d21h
```

A 的 5 小時額度快用完了？離開後打 `claude-b --continue`，B 就從剛剛那句話接著做。

## 特色

- 🔄 **換帳號不斷線**：A、B 共用對話紀錄、記憶、skills、agents，換手後上下文都還在。
- 📊 **所有帳號的用量一眼看完**：5 小時與每週額度、重置倒數，連沒在用的帳號也看得到；快用完時會變成橘色。
- 🐣 **會長大的小寵物**：用 Claude 越久等級越高，還會轉生成別的物種，總共有 16 種最終型態可以收集（[看全部](#小寵物)）。
- 💳 **選用的 API 帳號 C**：兩個訂閱額度都用完時還能繼續，狀態列改顯示今日、本月花費。
- 📏 **兩種版型**：完整版每個帳號一行；也可以切成只佔一行的緊湊版：

  ```
  ● A Pro 5h ▰▰▰▰▰▰▰▰▰▱ 92% ↻0h41m 週 67% │ ○ B Pro 5h ▰▱▱▱▱▱▱▱▱▱ 13% 週 44% │ Opus 5.5 │ 🐔✨ Lv7
  ```
- 🪟🐧 **支援 Windows 與 Linux**，只用 Python 標準函式庫，不用裝任何套件。macOS 不支援，因為登入憑證存在鑰匙圈，狀態列讀不到。

## 30 秒安裝

```bash
git clone https://github.com/MFpizza/claude-dotfiles.git
cd claude-dotfiles
python install.py        # Linux 用 python3；也要 API 帳號 C 的話加 --with-api
```

開一個新的終端機，用 `claude-a`、`claude-b` 各登入一次（輸入 `/login`），就完成了。詳細的安裝選項見下方。

---

## 目錄

1. [安裝](#安裝)
2. [使用方式](#使用方式)
3. [看懂狀態列](#看懂狀態列)
4. [小寵物](#小寵物)
5. [帳號 C（API 計費）](#帳號-capi-計費)
6. [更新、疑難排解與解除安裝](#更新疑難排解與解除安裝)

---

## 安裝

需要：[Claude Code](https://docs.claude.com/en/docs/claude-code)、Python 3.8 以上、git。安裝指令見上方的 [30 秒安裝](#30-秒安裝)；`install.py` 可以重複執行，不會產生重複的設定。

| 參數 | 用途 |
|---|---|
| `--with-api` | 也建立 API 計費的帳號 C（`~/.claude-c`）。沒加的電腦不會有 C，狀態列也不顯示它 |
| `--base ~/.claude-zhon` | （Linux）指定帳號 A 的目錄，B、C 會放在旁邊（`~/.claude-zhon-b`）。適合多人共用 `$HOME` 的機器。預設是 `$CLAUDE_CONFIG_DIR`，沒設就用 `~/.claude` |
| `--rc ~/.zshrc` | （Linux）指令要寫進哪個啟動檔，預設 `~/.bashrc` |
| `--no-rc` | （Linux）不改任何啟動檔，只印出要加的幾行，自己貼到想放的地方 |

### 登入（每台電腦、每個帳號只要一次）

**開一個新的終端機視窗**，新指令才會生效，然後：

```
claude-a      # 輸入 /login，登入第一個帳號
claude-b      # 輸入 /login，登入第二個帳號
claude-c      # 有裝 C 才需要：/login 時選 Anthropic Console 帳號
```

> 狀態列會用「執行 `install.py` 的那個 Python」。電腦上有好幾個 Python 時，請用你想固定使用的那個來安裝。

---

## 使用方式

把平常的 `claude` 換成 `claude-a` 或 `claude-b` 就好，後面可以接任何原本的參數：

| 想做的事 | 指令 |
|---|---|
| 用帳號 A 開 Claude | `claude-a` |
| 用帳號 B 開 Claude | `claude-b` |
| 用帳號 C（API 計費）開 Claude | `claude-c` |
| **A 額度用完，換 B 接著做** | 離開 A（`/exit`），在同一個資料夾執行 `claude-b --continue` |
| 挑一個以前的對話接著做 | `claude-b --resume`（A、B 開過的對話都會列出來） |
| 切換成緊湊版狀態列 | `python usage_statusline.py --layout compact` |
| 切回完整版狀態列 | `python usage_statusline.py --layout full` |

注意事項：

- **可以同時開 A、B 兩個視窗**，額度各算各的。但**不要兩個視窗開同一個對話**，紀錄會互相覆蓋。
- **設定只從 A 同步到 B。** 主題、模型、hooks 請在 `claude-a` 裡改，下次啟動 `claude-b` 時會自動複製過去；在 B 裡改的會被蓋掉。
- **skills、agents** 放在 A 的 `~/.claude/skills/`、`~/.claude/agents/`，兩個帳號都看得到。
- **原本的 `claude` 指令還能用**，等同於 `claude-a`（Linux 上是 `$CLAUDE_CONFIG_DIR` 指的帳號）。
- Linux 上平常用 `npx claude` 的人，請直接打 `claude-b`，**不要**打 `npx claude-b`，npx 可能會去下載同名的無關套件。`claude` 不在 PATH 上時，指令會自動改用 `npx --no -- claude`（只用已安裝的版本）；要指定執行檔可以設 `CLAUDE_BIN=/路徑/claude`。

<details>
<summary>哪些東西兩個帳號共用？</summary>

| 項目 | 兩帳號之間 |
|---|---|
| 對話紀錄、記憶（`projects/`）、`skills/`、`agents/`、`commands/`、`plugins/`、`file-history/`、`sessions/` | 共用（B 的資料夾是指向 A 的連結，資料只存一份） |
| 設定 `settings.json` | A → B 單向同步 |
| 登入憑證、帳號資訊與 MCP 設定（`.claude.json`）、輸入歷史 | 各自獨立 |

因為共用 `sessions/`，A、B 的視窗也能用 ListAgents 互相看到、用 SendMessage 互傳訊息。
</details>

---

## 看懂狀態列

### 完整版

每個帳號一行：

```
● A Pro  │ 5h ▰▰▰▱▱▱▱▱▱▱  31% ↻ 2h05m  │ 週 ▰▱▱▱▱▱▱▱▱▱   8% ↻ 3d04h  │ Opus 5.5 │ 🐔✨ Lv7
│ │  │      │  └── 用量條 ─┘  │      │       └ 每週額度（同左）            │          └ 小寵物
│ │  │      │                 │      └ 距離重置還有多久                     └ 目前模型
│ │  │      │                 └ 已用百分比
│ │  │      └ 5 小時額度
│ │  └ 訂閱方案
│ └ 帳號
└ ● 這個視窗正在用的帳號；○ 其他帳號
```

### 緊湊版

所有帳號擠在同一行，用 `│` 分隔：

```
● A Pro 5h ▰▰▰▰▰▰▰▰▰▱ 88% ↻4h13m 週 39% │ ○ B Pro 5h ▰▱▱▱▱▱▱▱▱▱ 13% 週 29% │ ○ C API 今日 $0.52 │ Opus 5.5 │ 🐔✨ Lv7
└──────────────── 帳號 A ─────────────┘   └───────────── 帳號 B ─────────┘   └── 帳號 C ──┘   └ 模型 ┘  └ 小寵物
```

和完整版的差別：

- 每週額度只顯示百分比，沒有用量條。
- 重置時間只在用量達 80% 時才顯示（例如上面 A 的 `↻4h13m`）。
- 帳號那一段最後面出現 `*`（例如 `週 39%*`），代表是快取資料。

### 顏色與時間

| 顏色 | 用量 | 意思 |
|---|---|---|
| 藍 | 0–49% | 充足 |
| 琥珀 | 50–79% | 注意 |
| 橘 | 80–100% | 快用完，準備換帳號 |

配色避開紅綠，對色弱友善。時間格式：`↻ 43m` 是 43 分鐘後重置，`↻ 2h05m` 是 2 小時 5 分，`↻ 3d04h` 是 3 天 4 小時。

### 特殊狀態

| 顯示 | 意思 | 怎麼辦 |
|---|---|---|
| `快取 · 3 天前` | 抓不到最新資料，顯示最後一次成功的數值 | 通常是網路、token 失效或已退訂，不處理也沒關係 |
| `未登入 · 執行 claude-b 後 /login` | 這台電腦的這個帳號還沒登入 | 照提示登入 |
| `token 過期` | 目前視窗帳號的 token 剛好過期 | 送出任何訊息後會自動刷新 |
| `HTTP 401` / `HTTP 403` | 多半是登入失效 | 用該帳號執行 `/login` |
| `5h ▱▱▱▱▱▱▱▱▱▱ —` | 伺服器沒回傳這個額度 | 常見於未訂閱的帳號 |

用量每 2 分鐘最多查一次。版型設定存在 `~/.claude/statusline.json`，所有帳號共用。

> 本文的 `~/.claude`、`~/.claude-b` 指帳號 A、B 的目錄；Linux 上用了 `--base` 的話，請換成你指定的目錄。

---

## 小寵物

狀態列最右邊住著一隻會長大的寵物，例如 `🐔✨ Lv7`。A、B 共用同一隻。

### 成長

經驗值是**使用 Claude 的活躍時間**：狀態列兩次刷新間隔不到 5 分鐘才算，視窗掛著不動不算。越後面越難升：

| 等級 | Lv3 | Lv5 | Lv7 | Lv10 | Lv13 | Lv16 | Lv20 |
|---|---|---|---|---|---|---|---|
| 累積時數 | 0.2 | 1.4 | 4.7 | 16 | 38 | 74 | 150 |

升級後 10 分鐘內會顯示 `🎉`，等級數字也會亮起來：`🐔✨🎉 Lv8`。

### 心情

跟著目前對話的 context 用量變化：`✨` 精神飽滿（< 30%）→ 無符號 → `💦` 有點累（60% 以上）→ `💤` 撐不住了（85% 以上，該 `/compact` 或開新對話）。

### 轉生與成長線

滿 150 小時（Lv20）就會轉生：從頭開始，前面多一顆 `⭐`（4 次以上顯示成 `⭐4`），多出來的時數會帶到下一世。

下一世會**隨機**變成下面其中一系。看起始型態就知道是哪一類：🥚 卵生、🌰 植物、🍼 哺乳。有分支的系會在分歧點隨機走其中一支，而且優先抽還沒養到最後的，收集完 16 種最終型態才會重複。

| 成長線 | 前期 | 分支（→ 最終型態） |
|---|---|---|
| 鳥系（第一世） | 🥚 🐣 🐥 🐔 | 家禽 🐓 🦃 🦚 👑🦚 ／ 猛禽 🐦 🦉 🦅 👑🦅 |
| 水鳥系 | 🥚 🐣 🐥 🦆 🐧 🦩 🦢 👑🦢 | 不分歧 |
| 爬蟲系 | 🥚 🦎 🐢 🐍 🐊 | 恐龍 🦕 🦖 👑🦖 ／ 龍 🐲 🐉 👑🐉 |
| 深海系 | 🥚 🦐 🐟 🐠 🐡 | 頭足 🦑 🐙 👑🐙 ／ 甲殼 🦀 🦞 👑🦞 |
| 昆蟲系 | 🥚 🐛 🐜 🐞 🦗 🐝 🦋 👑🦋 | 不分歧 |
| 植物系 | 🌰 🌱 🌿 🍀 | 花 🌷 🌹 🌻 👑🌻 ／ 櫻 🌳 🌸 🍒 👑🌸 ／ 蘋果 🌳 🍏 🍎 👑🍎 ／ 葡萄 🍃 🍇 🍷 👑🍷 |
| 哺乳系 | 🍼 🐾 | 貓科 🐱 🐈 🐆 🐅 🦁 👑🦁 ／ 犬科 🐶 🐕 🐺 👑🐺 ／ 海獸 🦦 🐬 🐳 🐋 👑🐋 ／ 靈長 🐵 🙈 🐒 🦧 🦍 👑🦍 |

每個型態從哪一級開始，寫在 `usage_statusline.py` 的 `PET_LINES`。紀錄存在 `~/.claude/statusline-pet.json`，刪掉就從頭開始。

---

## 帳號 C（API 計費）

API 帳號沒有額度，所以狀態列改顯示**花費**：

```
● C API  │ 本次 $0.42 │ 今日 $3.10 │ 本月 $25.70
```

- `本次` 只在目前視窗用 C 時出現。
- 金額是狀態列根據 Claude Code 的費用估算自己累計的，只算這台電腦。實際扣款以 [Console](https://console.anthropic.com/) 為準。
- 安裝時沒加 `--with-api` 的電腦不會顯示 C。

**走公司 gateway 或 API token、不用 `/login` 的話**，把環境變數寫在帳號目錄裡的 `account-settings.json`（例如 `~/.claude-c/account-settings.json`）。不能寫進 `settings.json`，因為它每次啟動都會被 A 的設定蓋掉：

```json
{
  "env": {
    "ANTHROPIC_BASE_URL": "https://你的-gateway/anthropic/",
    "ANTHROPIC_AUTH_TOKEN": "你的 token"
  }
}
```

這個檔裡有 token：Linux 上請 `chmod 600`，也不要放進任何 git repo。B 也可以用同樣的方式設定。

---

## 更新、疑難排解與解除安裝

### 更新

```
cd claude-dotfiles
git pull
```

拉下來就生效，不用重新安裝。只有兩種情況要重跑 `install.py`（帶同樣的參數；重複執行沒關係）：新版加了共用資料夾，這時要先關掉所有 `claude-b` 視窗；或是你搬移了這個資料夾。

想改顏色、用量條長度、帳號名稱，都在 `usage_statusline.py` 開頭（`LOW`／`MID`／`HIGH`、`BAR_WIDTH`、`ACCOUNTS`）。改完可以用 `echo '{}' | python usage_statusline.py` 預覽。

### 疑難排解

| 問題 | 處理 |
|---|---|
| 找不到 `claude-a` | 確認是**新開的**視窗。PowerShell：`notepad $PROFILE` 裡要有 `# >>> claude-dotfiles >>>` 區塊。Linux：啟動檔裡要有同名區塊，或執行 `source ~/.bashrc` |
| PowerShell 說「已停用指令碼執行」 | `Set-ExecutionPolicy -Scope CurrentUser RemoteSigned` |
| cmd 找不到指令 | 確認 `~/.local/bin` 裡有 `claude-a.cmd`，而且這個資料夾在 PATH 上；沒有的話重跑 `install.py` |
| 狀態列沒出現 | 重開 Claude；確認 `~/.claude/settings.json` 有 `statusLine`，裡面的 Python 路徑也存在；用上面的預覽指令看看有沒有錯誤 |
| 出現方框或問號 | 換用 Windows Terminal 之類支援 Unicode 的終端機，字型例如 Cascadia Code |
| 安裝時「有同名的項目，未建立連結」 | A、B 有同名檔案，安裝程式不會覆蓋。決定好要留哪一份後，刪掉 `~/.claude-b` 裡的那個資料夾，再重跑 `install.py` |
| B 的設定跟 A 不一樣 | 設定只在用 `claude-b` 啟動時同步；直接設 `CLAUDE_CONFIG_DIR` 再執行 `claude` 不會同步 |

### 解除安裝

1. 刪掉啟動檔裡 `# >>> claude-dotfiles >>>` 到 `# <<< claude-dotfiles <<<` 之間的內容。Windows 是 `notepad $PROFILE`，Linux 是 `~/.bashrc` 或你用 `--rc` 指定的檔案。
2. 刪掉 `~/.claude/settings.json` 裡的 `statusLine` 區塊，或用 `settings.json.bak` 還原。
3. 刪除 B（C 也一樣）。**B 裡面是指向 A 的連結，刪錯方式會連 A 的資料一起刪掉：**

   ```powershell
   # Windows：只用這個指令，不要用檔案總管或 Remove-Item -Recurse
   cmd /c rmdir /s /q "$HOME\.claude-b"
   ```

   ```bash
   # Linux：先刪連結再刪目錄，不要用 rm -rf ~/.claude-b/*/
   find ~/.claude-b -maxdepth 1 -type l -delete
   rm -rf ~/.claude-b
   ```
4. 刪掉 `~/.claude` 裡的 `usage-cache.json`、`statusline-pet.json`、`statusline.json`、`api-cost.json`。Windows 也要刪 `~/.local/bin/claude-a.cmd`、`claude-b.cmd`、`claude-c.cmd`。

### 檔案與安全

| 檔案 | 用途 |
|---|---|
| `install.py` | 建立 B／C 的目錄與共用連結、設定狀態列、安裝 `claude-a/b/c` 指令 |
| `usage_statusline.py` | 狀態列與小寵物 |
| `claude-accounts.ps1`、`claude-accounts.sh`、`bin/*.cmd` | PowerShell、bash/zsh、cmd 版的 `claude-a/b/c` |

- repo 裡**沒有**任何憑證。憑證只存在各電腦的 `~/.claude*/.credentials.json`。
- 用量是呼叫 Claude Code 內部用的 `api/oauth/usage` 取得的。它不是公開 API，Claude Code 改版後可能會失效。
- 沒在用的帳號 token 過期時，狀態列會替它刷新；正在用的帳號由 Claude 自己刷新，兩邊不會衝突。
