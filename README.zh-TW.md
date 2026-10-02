# claude-dotfiles

**同時使用多個 Claude Code 帳號，額度用完一鍵換手，對話不中斷。**

```
● A Pro  │ 5h ▰▰▰▰▰▰▰▰▰▱  92% 🌘 0h41m  │ 週 ▰▰▰▰▰▰▰▱▱▱  67% 🌘 1d16h  │ Opus 5.5 │ 🐔✨ Lv7
○ B Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱  13% 🌖 3h46m  │ 週 ▰▰▰▰▱▱▱▱▱▱  44% 🌗 3d21h
```

A 的 5 小時額度快用完了？離開後打 `claude-b --continue`，B 就從剛剛那句話接著做。

[English](README.md)

## 特色

- 🔄 **換帳號不斷線**：所有帳號共用對話紀錄、記憶、skills、agents。
- 📊 **所有帳號的用量一眼看完**：5 小時與每週額度、重置倒數，連沒在用的帳號也看得到；快用完時變成橘色。
- ➕ **帳號要幾個就加幾個**：訂閱（Pro/Max）或 API 計費都行。只有一個帳號也能單獨使用狀態列。
- 🐣 **會長大的小寵物**：用 Claude 越久等級越高，滿級後轉生成別的物種，共 17 種最終型態可以收集。
- 📏 **兩種版型**：每個帳號一行，或全部擠成一行：

  ```
  ● A Pro 5h ▰▰▰▰▰▰▰▰▰▱ 92% 🌘 週 67% 🌘 │ ○ B Pro 5h ▰▱▱▱▱▱▱▱▱▱ 13% 🌖 週 44% 🌗 │ Opus 5.5 │ 🐔✨ Lv7
  ```
- 🌐 英文與繁體中文，依系統語言自動切換。
- 📦 單一執行檔，不用另外安裝任何東西。支援 Windows、Linux、macOS。

## 安裝

```powershell
irm https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.ps1 | iex     # Windows
```
```bash
curl -fsSL https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.sh | sh  # Linux、macOS
```

會把 `claude-accounts` 放進 `~/.local/bin` 並打開選單。重跑同一行就是更新。

## 使用方式

| 想做的事 | 指令 |
|---|---|
| 打開選單 | `claude-accounts` |
| 新增帳號（自動分配 b、c…） | `claude-accounts add`（API 計費：`add --api`） |
| 用帳號 b 開 Claude | `claude-b`（可接任何 `claude` 參數） |
| **A 額度用完，換 B 接著做** | 離開 A，在同一個資料夾執行 `claude-b --continue` |
| 挑以前的對話接著做 | `claude-b --resume`（所有帳號的對話都會列出） |
| 不在狀態列顯示某個帳號 | `claude-accounts hide c`（`show c` 改回來） |
| 狀態列改成一行 | `claude-accounts layout compact`（`full` 改回來） |
| 語言 | `claude-accounts lang en`（`zh-TW`、`auto`） |
| 移除帳號 | `claude-accounts remove c` |

第一次使用某個帳號時，在 Claude 裡輸入 `/login`。

注意事項：
- 可以同時開不同帳號的視窗，但**不要兩個視窗開同一個對話**，紀錄會互相覆蓋。
- 設定只從 a 同步到其他帳號：請在 `claude-a` 裡改，其他帳號下次啟動時會自動套用。
- 帳號專屬的設定（例如 API 帳號走 gateway 的網址和 token）寫在該帳號資料夾的 `account-settings.json`：

  ```json
  { "env": { "ANTHROPIC_BASE_URL": "https://你的-gateway/", "ANTHROPIC_AUTH_TOKEN": "…" } }
  ```

## 看懂狀態列

| 顯示 | 意思 |
|---|---|
| `●` / `○` | 這個視窗用的帳號／其他帳號 |
| `5h ▰▰▰▱▱▱▱▱▱▱ 31% 🌗 2h05m` | 5 小時額度：已用 31%，2 小時 5 分後重置 |
| `🌕` `🌖` `🌗` `🌘` `🌑` | 距離額度重置還剩多少時間：剛重置時是滿月，新月代表快要重置了 |
| 藍／琥珀／橘 | 低於 50%／低於 80%／80% 以上 |
| `快取 · 3 小時前`（緊湊版：`*`） | 暫時抓不到最新資料，顯示最後一次的數值 |
| `今日 $3.10 │ 本月 $25.70` | API 帳號在這台電腦的花費（實際帳單以 Console 為準） |
| `未登入 · 執行 claude-b 後 /login` | 這個帳號在這台電腦還沒登入 |

### 小寵物

經驗值是活躍時間（間隔超過 5 分鐘不算）：Lv7 大約 5 小時，Lv20 要 150 小時。滿 Lv20 會轉生，前面多一顆 ⭐，並隨機變成新的物種，優先抽還沒養過的。心情跟著對話的 context 用量：✨ 精神好、💦 累了、💤 該 `/compact` 了。

| 成長線 | 會長成 |
|---|---|
| 🥚 鳥系 | 🐣 🐥 🐔 → 🐓 🦃 🦚 或 🐦 🦉 🦅 |
| 🥚 水鳥系 | 🐣 🐥 🦆 🐧 🦩 🦢 |
| 🥚 爬蟲系 | 🦎 🐢 🐍 🐊 → 🦕 🦖 或 🐲 🐉 |
| 🥚 深海系 | 🦐 🐟 🐠 🐡 → 🦑 🐙 或 🦀 🦞 |
| 🥚 昆蟲系 | 🐛 🐜 🐞 🦗 🐝 🦋 |
| 🌰 植物系 | 🌱 🌿 🍀 → 🌷 🌹 🌻、🌳 🌸 🍒、🌳 🍏 🍎 或 🍃 🍇 🍷 |
| 🍼 哺乳系 | 🐾 → 🐱 🐈 🐆 🐅 🦁、🐶 🐕 🐺、🦦 🐬 🐳 🐋、🐰 🐇 🌕 或 🐵 🙈 🐒 🦧 🦍 |

每一系到 Lv20 都會戴上皇冠（👑）。

## 運作方式

- 帳號 a 是 `~/.claude`（或 `$CLAUDE_CONFIG_DIR`），帳號 x 是 `~/.claude-x`。帳號清單存在帳號 a 目錄裡的 `claude-accounts.json`。
- 其他帳號的 `projects`、`skills`、`agents`、`commands`、`plugins`、`file-history`、`sessions` 都連結到帳號 a，資料只存一份；登入憑證與 `.claude.json` 各自獨立。
- `claude-b` 其實是同一個程式換了名字（Windows 用 hard link，其他平台用 symlink）。
- 目前帳號的用量直接來自 Claude Code；其他帳號透過 `api/oauth/usage` 查詢，這是 Claude Code 內部使用的端點，日後可能改變。
- repo 裡沒有任何憑證；憑證只存在各帳號的 `.credentials.json`。

## 解除安裝

`claude-accounts uninstall` 會移除指令與狀態列，保留所有帳號資料夾；之後再刪掉 `~/.local/bin/claude-accounts`。想連帳號資料夾一起刪，先用 `claude-accounts remove <字母>`，它會先拆掉連結再刪資料夾，不會動到帳號 a 的資料。

## 從 Python 版升級

照上面的方式安裝即可。第一次執行時會找出原本的帳號 B、C，保留你的寵物，並移除 `$PROFILE`／`~/.bashrc` 裡舊的區塊（備份成 `.bak`）。完成後請開新的終端機。
