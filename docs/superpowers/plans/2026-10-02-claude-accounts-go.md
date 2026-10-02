# claude-accounts Go 重寫 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 claude-dotfiles（Python + shell 腳本、固定 A/B/C）改寫成單一 Go 執行檔 `claude-accounts`：可自由增減字母命名的帳號、只裝狀態列也能用、介面預設英文並支援繁中，並自動遷移舊版安裝。

**Architecture:** 一個執行檔依 argv[0] 分派：以 `claude-<字母>` 被呼叫時啟動該帳號的 Claude，否則執行子指令（互動選單、add/remove…、`statusline`）。帳號清單存在主帳號目錄的 `claude-accounts.json`，指令連結、共用資料夾、狀態列全部由它產生。各功能拆成 `internal/` 下職責單一的套件。

**Tech Stack:** Go（`go.mod` 宣告 `go 1.22`），只用標準函式庫；GitHub Actions 做 CI 與 Release。

**Spec:** `docs/superpowers/specs/2026-10-02-claude-accounts-go-design.md`

## Global Constraints

- Module path：`github.com/MFpizza/claude-dotfiles`；`go.mod` 寫 `go 1.22`；**只用標準函式庫**，不加任何第三方套件。
- 本機 Go 安裝在 `$HOME/.local/go`；每個 Bash 指令都要以 `export PATH="$HOME/.local/go/bin:$PATH";` 開頭（shell 狀態不會保留）。
- 執行檔名稱 `claude-accounts`（Windows 為 `.exe`）；帳號指令 `claude-<字母>`；帳號名稱只有 `a`–`z`，主帳號固定 `a`，新增時分配 `b`–`z` 中最小的空字母。
- 設定檔：主帳號目錄下的 `claude-accounts.json`。其他資料檔名稱不變：`usage-cache.json`、`statusline-pet.json`、`api-cost.json`、`settings.json`、`account-settings.json`、`.credentials.json`。
- 共用資料夾（依序）：`projects`、`skills`、`agents`、`commands`、`plugins`、`file-history`、`sessions`。
- 數值：用量快取 TTL 120 秒、超過 3×TTL 視為過期快取；HTTP 逾時 4 秒；token 在 60 秒內過期就刷新；寵物活躍間隔 < 300 秒；🎉 顯示 600 秒；一世 150 小時、最高 Lv20；API 花費紀錄保留 62 天。
- 配色：低 `#5fafff`、中 `#e5c07b`、高 `#ff8c42`（< 50% / < 80% / 其餘）、強調 `#d97757`、文字 `#c8ccd4`、次要 `#7f8794`、軌道 `#3e4451`；用量條 10 格 `▰▱`。
- 介面預設英文，繁中字串放在 `internal/i18n` 的同一張表；每個鍵都必須有兩種語言、且格式化參數一致。
- 寵物 emoji 限 Unicode 12 以內，不用 ZWJ 組合字。
- `claude-accounts statusline` 永遠回傳 0，且一定輸出內容。
- 程式碼註解用英文，風格比照現有 Python：只在非顯而易見處加一行說明。
- 每個 commit 訊息結尾加上：

  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
  ```

## Spec 補充說明（實作時以此為準）

- **找主帳號目錄**：`$CLAUDE_CONFIG_DIR`（未設則 `~/.claude`）底下有 `claude-accounts.json` 就是主帳號；沒有、但它的 `projects` 是連結時，主帳號是連結目標的上一層。這讓在 `claude-b` 的視窗裡執行 `claude-accounts` 也能找到主帳號。狀態列指令另外帶 `--base <主帳號目錄>`。
- **API 花費紀錄分帳號**：`api-cost.json` 每筆多一個 `account` 欄位；舊資料沒有這個欄位，視為帳號 `c`。
- **網路錯誤**在狀態列顯示成 `network error`／`網路錯誤`，不顯示 Go 的完整錯誤訊息。
- **狀態列 panic** 時輸出 `claude-accounts: <錯誤>`，回傳 0。

## Review Focus

1. **在非主帳號的視窗裡執行**（`CLAUDE_CONFIG_DIR=~/.claude-b`，指令沒帶 `--base`）：必須找到真正的主帳號，而不是把 `~/.claude-b` 當成主帳號另起一份設定。→ Task 3 `TestResolveMainDirFromLinkedAccount`
2. **`claude-accounts.json` 損毀**：管理指令要明確報錯、不能覆寫該檔；狀態列照樣輸出。→ Task 3 `TestLoadRejectsBadFile`、Task 11 `TestRunSurvivesBadInput`
3. **狀態列 stdin 是空的或不是 JSON**：照樣輸出寵物，回傳 0。→ Task 11 `TestRunSurvivesBadInput`、Task 12 `TestStatuslineCmdNeverFails`
4. **使用者手動刪掉了某個帳號的資料夾**：`claude-x` 要給出清楚的錯誤，`remove` 仍能把它從清單拿掉。→ Task 6 `TestBuildDirMissing`、Task 5 `TestRemoveMissingDir`
5. **`settings.json` 裡有使用者自己的設定**：設定狀態列時要保留原本的鍵順序與內容，並留下 `.bak`。→ Task 5 `TestSetStatusLinePreservesOrder`

---

## File Structure

```
go.mod
cmd/claude-accounts/main.go        argv[0] 分派、statusline 子指令、啟動帳號
cmd/claude-accounts/app.go         管理子指令（setup/add/remove/list/show/hide/layout/lang/uninstall）
cmd/claude-accounts/menu.go        互動選單
cmd/claude-accounts/main_test.go
internal/fsutil/fsutil.go          原子寫入、JSON 讀寫
internal/i18n/i18n.go              Lang、Resolve、T
internal/i18n/messages.go          英文／繁中字串表
internal/i18n/system_other.go      非 Windows：系統語言 = 英文
internal/i18n/system_windows.go    Windows：GetUserDefaultUILanguage
internal/config/config.go          Account/Config、字母分配、路徑工具、主帳號目錄
internal/links/links.go            IsLink、Points
internal/links/links_unix.go       symlink 版 Dir/Exe/RemoveExe
internal/links/links_windows.go    junction／hard link 版
internal/accounts/share.go         共用資料夾連結、合併、安全刪除
internal/accounts/accounts.go      Add/Remove/SyncCommands/CommandPath
internal/settings/settings.go      settings.json 的 statusLine（保留鍵順序）
internal/launch/launch.go          組出啟動 claude 的參數與環境
internal/launch/exec_unix.go       syscall.Exec
internal/launch/exec_windows.go    子行程 + 回傳退出碼
internal/migrate/migrate.go        偵測舊帳號、移除舊安裝區塊
internal/migrate/shell_unix.go     ~/.bashrc、~/.zshrc
internal/migrate/shell_windows.go  PowerShell $PROFILE
internal/style/style.go            ANSI 顏色
internal/pet/lines.go              成長線定義
internal/pet/pet.go                寵物邏輯
internal/statusline/input.go       解析 Claude Code 傳入的 JSON
internal/statusline/usage.go       用量快取、OAuth 查詢與刷新、rate_limits
internal/statusline/cost.go        API 花費紀錄
internal/statusline/render.go      各區塊排版
internal/statusline/run.go         組合整條狀態列
install.sh、install.ps1            一行安裝
.github/workflows/ci.yml、release.yml
README.md、README.zh-TW.md
```

刪除：`install.py`、`usage_statusline.py`、`claude-accounts.ps1`、`claude-accounts.sh`、`bin/`。

---

### Task 1: Go 工具鏈、module 與 fsutil

**Files:**
- Create: `go.mod`, `internal/fsutil/fsutil.go`, `internal/fsutil/fsutil_test.go`
- Modify: `.gitignore`

**Interfaces:**
- Produces: `fsutil.ReadJSON(path string, v any) error`、`fsutil.Marshal(v any, indent bool) ([]byte, error)`、`fsutil.WriteJSON(path string, v any, indent bool) error`、`fsutil.WriteFileAtomic(path string, data []byte, perm os.FileMode) error`

- [ ] **Step 1: 安裝 Go 到 `$HOME/.local/go`**

```bash
V=$(curl -fsSL 'https://go.dev/dl/?mode=json' | python3 -c 'import json,sys; print(json.load(sys.stdin)[0]["version"])')
curl -fsSL "https://go.dev/dl/$V.linux-amd64.tar.gz" | tar -C "$HOME/.local" -xz
export PATH="$HOME/.local/go/bin:$PATH"; go version
```
Expected: 印出 `go version go1.x linux/amd64`。

- [ ] **Step 2: 建立 module 與 .gitignore**

`go.mod`：
```
module github.com/MFpizza/claude-dotfiles

go 1.22
```
`.gitignore` 改成：
```
__pycache__/
*.pyc
/dist/
```

- [ ] **Step 3: 寫失敗的測試** `internal/fsutil/fsutil_test.go`

```go
package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteJSONRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	in := map[string]any{"name": "🐔 & <b>"}
	if err := WriteJSON(path, in, false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "🐔 & <b>") {
		t.Fatalf("text was escaped: %s", data)
	}
	var out map[string]any
	if err := ReadJSON(path, &out); err != nil || out["name"] != in["name"] {
		t.Fatalf("got %v, %v", out, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestWriteFileAtomicKeepsPerm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no unix permissions")
	}
	path := filepath.Join(t.TempDir(), "secret.json")
	if err := WriteFileAtomic(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v", fi.Mode().Perm())
	}
}

func TestReadJSONMissing(t *testing.T) {
	var v map[string]any
	err := ReadJSON(filepath.Join(t.TempDir(), "nope.json"), &v)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 4: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/fsutil/`
Expected: FAIL，`undefined: WriteJSON`。

- [ ] **Step 5: 實作** `internal/fsutil/fsutil.go`

```go
// Package fsutil holds the small file helpers shared by every package.
package fsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// ReadJSON decodes the file at path into v.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Marshal encodes v without HTML escaping, indented with two spaces when indent is true.
func Marshal(v any, indent bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteJSON writes v to path atomically.
func WriteJSON(path string, v any, indent bool) error {
	data, err := Marshal(v, indent)
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data, 0o644)
}

// WriteFileAtomic writes a temp file named after the PID and renames it over path,
// so several status line processes refreshing at once never see a torn file.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
```

- [ ] **Step 6: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/fsutil/`
Expected: `ok`

- [ ] **Step 7: Commit**

```bash
git add go.mod .gitignore internal/fsutil
git commit -F - <<'EOF'
feat: Go module 與原子寫入的 fsutil

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 2: i18n（英文／繁中）

**Files:**
- Create: `internal/i18n/i18n.go`, `internal/i18n/messages.go`, `internal/i18n/system_other.go`, `internal/i18n/system_windows.go`, `internal/i18n/i18n_test.go`

**Interfaces:**
- Produces: `type Lang string`、`i18n.En`、`i18n.ZhTW`、`i18n.Resolve(setting string) Lang`、`i18n.FromLocale(locale string) (Lang, bool)`、`(Lang).T(key string, args ...any) string`
- 後續任務使用的鍵（全部在本任務定義）：`wk today month session cached ago_m ago_h ago_d not_logged_in login_hint token_expired no_data network_error usage accounts_header type_subscription type_api shown hidden account_row menu prompt ask_type ask_which ask_layout ask_lang confirm_remove confirm_delete_dir confirm_uninstall confirm_api added added_api conflict removed deleted_dir kept_dir set_shown set_hidden layout_set lang_set setup_title found_account cleaned_block removed_file statusline_set setup_done uninstalled invalid_choice err_main err_no_account err_full err_layout err_lang err_no_claude err_dir_missing err_unknown_cmd err_usage_letter err_config err_not_set_up`

- [ ] **Step 1: 寫失敗的測試** `internal/i18n/i18n_test.go`

```go
package i18n

import (
	"regexp"
	"strings"
	"testing"
)

func TestEveryMessageHasBothLanguagesWithSameVerbs(t *testing.T) {
	verbs := regexp.MustCompile(`%[-0-9.]*[a-zA-Z]`)
	for key, m := range messages {
		if m[0] == "" || m[1] == "" {
			t.Errorf("%s: missing a translation", key)
		}
		en := strings.Join(verbs.FindAllString(m[0], -1), " ")
		zh := strings.Join(verbs.FindAllString(m[1], -1), " ")
		if en != zh {
			t.Errorf("%s: verbs differ: %q vs %q", key, en, zh)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "zh_TW.UTF-8")
	if got := Resolve("auto"); got != ZhTW {
		t.Errorf("auto with zh_TW: %v", got)
	}
	if got := Resolve("en"); got != En {
		t.Errorf("explicit en: %v", got)
	}
	t.Setenv("LANG", "fr_FR.UTF-8")
	if got := Resolve(""); got != En {
		t.Errorf("fr_FR: %v", got)
	}
	t.Setenv("LANG", "en_US.UTF-8")
	if got := Resolve("zh-TW"); got != ZhTW {
		t.Errorf("explicit zh-TW: %v", got)
	}
}

func TestFromLocale(t *testing.T) {
	cases := map[string]Lang{
		"zh_TW.UTF-8": ZhTW, "zh-HK": ZhTW, "zh_Hant_TW": ZhTW, "zh_MO": ZhTW,
		"zh_CN.UTF-8": En, "C.UTF-8": En, "en_US": En,
	}
	for in, want := range cases {
		if got, ok := FromLocale(in); !ok || got != want {
			t.Errorf("%s: got %v %v", in, got, ok)
		}
	}
	if _, ok := FromLocale(""); ok {
		t.Error("empty locale should not resolve")
	}
}

func TestT(t *testing.T) {
	if got := ZhTW.T("ago_d", 3); got != "3 天前" {
		t.Errorf("zh: %q", got)
	}
	if got := En.T("ago_d", 3); got != "3d ago" {
		t.Errorf("en: %q", got)
	}
	if got := En.T("no-such-key"); got != "no-such-key" {
		t.Errorf("unknown key: %q", got)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/i18n/`
Expected: FAIL，`undefined: messages`。

- [ ] **Step 3: 實作** `internal/i18n/i18n.go`

```go
// Package i18n holds the user-facing text in English and Traditional Chinese.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Lang string

const (
	En   Lang = "en"
	ZhTW Lang = "zh-TW"
)

// Resolve turns the config's lang setting ("en", "zh-TW", "auto" or "") into a language.
func Resolve(setting string) Lang {
	switch setting {
	case string(En):
		return En
	case string(ZhTW):
		return ZhTW
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := FromLocale(os.Getenv(key)); ok {
			return l
		}
	}
	return systemLang()
}

// FromLocale maps a locale such as zh_TW.UTF-8 or zh-Hant to a language; ok is false
// for an empty locale.
func FromLocale(locale string) (Lang, bool) {
	l := strings.ToLower(strings.ReplaceAll(locale, "-", "_"))
	if l == "" {
		return "", false
	}
	for _, prefix := range []string{"zh_tw", "zh_hk", "zh_mo", "zh_hant"} {
		if strings.HasPrefix(l, prefix) {
			return ZhTW, true
		}
	}
	return En, true
}

// T returns the message for key in l, formatted with args; English is the fallback.
func (l Lang) T(key string, args ...any) string {
	m, ok := messages[key]
	if !ok {
		return key
	}
	s := m[0]
	if l == ZhTW {
		s = m[1]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
```

`internal/i18n/system_other.go`：
```go
//go:build !windows

package i18n

func systemLang() Lang { return En }
```

`internal/i18n/system_windows.go`：
```go
//go:build windows

package i18n

import "syscall"

var uiLanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// systemLang reads the Windows display language; LANG is rarely set there.
func systemLang() Lang {
	if uiLanguage.Find() != nil {
		return En
	}
	id, _, _ := uiLanguage.Call()
	switch id & 0xffff {
	case 0x0404, 0x0c04, 0x1404: // zh-TW, zh-HK, zh-MO
		return ZhTW
	}
	return En
}
```

`internal/i18n/messages.go`：
```go
package i18n

// messages maps a key to its {English, Traditional Chinese} text.
var messages = map[string][2]string{
	// Status line
	"wk":            {"wk", "週"},
	"today":         {"today", "今日"},
	"month":         {"month", "本月"},
	"session":       {"session", "本次"},
	"cached":        {"cached · %s", "快取 · %s"},
	"ago_m":         {"%dm ago", "%d 分鐘前"},
	"ago_h":         {"%dh ago", "%d 小時前"},
	"ago_d":         {"%dd ago", "%d 天前"},
	"not_logged_in": {"not logged in", "未登入"},
	"login_hint":    {"run claude-%s, then /login", "執行 claude-%s 後 /login"},
	"token_expired": {"token expired", "token 過期"},
	"no_data":       {"no data", "無資料"},
	"network_error": {"network error", "網路錯誤"},

	// Commands
	"usage": {`claude-accounts: run several Claude Code accounts side by side

Usage:
  claude-accounts                       interactive menu (the first run sets things up)
  claude-accounts add [--api] [--dir DIR]
                                        add an account; it gets the next free letter
  claude-accounts remove <letter> [--yes] [--delete-dir]
  claude-accounts list
  claude-accounts show <letter>         show the account in the status line
  claude-accounts hide <letter>         hide it (still shown while you use it)
  claude-accounts layout full|compact
  claude-accounts lang auto|en|zh-TW
  claude-accounts uninstall [--yes]
  claude-accounts version

Every account gets a command, claude-a, claude-b, ...; use it like claude.
`, `claude-accounts：同時使用多個 Claude Code 帳號

用法：
  claude-accounts                       互動選單（第一次執行會完成設定）
  claude-accounts add [--api] [--dir 目錄]
                                        新增帳號，自動分配下一個字母
  claude-accounts remove <字母> [--yes] [--delete-dir]
  claude-accounts list
  claude-accounts show <字母>           在狀態列顯示這個帳號
  claude-accounts hide <字母>           不在狀態列顯示（正在使用時仍會顯示）
  claude-accounts layout full|compact
  claude-accounts lang auto|en|zh-TW
  claude-accounts uninstall [--yes]
  claude-accounts version

每個帳號都有一個指令：claude-a、claude-b……，用法和 claude 一樣。
`},
	"accounts_header":   {"Accounts:\n", "帳號：\n"},
	"type_subscription": {"subscription", "訂閱"},
	"type_api":          {"API", "API 計費"},
	"shown":             {"shown", "顯示"},
	"hidden":            {"hidden", "隱藏"},
	"account_row":       {"  %s  %-14s %-22s status line: %s\n", "  %s  %-14s %-22s 狀態列：%s\n"},
	"menu": {`
  1) Add account
  2) Remove account
  3) Show / hide in the status line
  4) Status line layout (now: %s)
  5) Language (now: %s)
  6) Uninstall
  0) Exit
`, `
  1) 新增帳號
  2) 移除帳號
  3) 在狀態列顯示／隱藏
  4) 狀態列版型（目前：%s）
  5) 語言（目前：%s）
  6) 解除安裝
  0) 離開
`},
	"prompt":             {"> ", "> "},
	"ask_type":           {"Type: 1) Subscription (Pro/Max)  2) API billing\n> ", "類型：1) 訂閱（Pro/Max）  2) API 計費\n> "},
	"ask_which":          {"Which account (letter)? ", "哪一個帳號（字母）？"},
	"ask_layout":         {"Layout: 1) full  2) compact\n> ", "版型：1) 完整版  2) 緊湊版\n> "},
	"ask_lang":           {"Language: 1) auto  2) English  3) 繁體中文\n> ", "語言：1) 自動  2) English  3) 繁體中文\n> "},
	"confirm_remove":     {"Remove account %s? [y/N] ", "移除帳號 %s？[y/N] "},
	"confirm_delete_dir": {"Also delete %s, including its login? [y/N] ", "一併刪除 %s（含登入憑證）？[y/N] "},
	"confirm_uninstall":  {"Remove every claude-* command and the status line? Account folders are kept. [y/N] ", "移除所有 claude-* 指令與狀態列？帳號資料夾會保留。[y/N] "},
	"confirm_api":        {"Is account %s billed through the API (Anthropic Console)? [Y/n] ", "帳號 %s 是 API 計費（Anthropic Console）帳號嗎？[Y/n] "},
	"added":              {"✓ Added account %s (%s) at %s\n  Run claude-%s, then type /login.\n", "✓ 已新增帳號 %s（%s），目錄 %s\n  執行 claude-%s，再輸入 /login。\n"},
	"added_api":          {"  At /login choose Anthropic Console; for a gateway, put its URL and token in %s\n", "  /login 時選 Anthropic Console；走 gateway 的話，把網址和 token 寫進 %s\n"},
	"conflict":           {"! %s/%s has items that also exist in the main account, so it was not linked. Sort it out by hand and run again.\n", "! %s/%s 有與主帳號同名的項目，未建立連結；請手動處理後再執行一次。\n"},
	"removed":            {"✓ Removed account %s\n", "✓ 已移除帳號 %s\n"},
	"deleted_dir":        {"✓ Deleted %s\n", "✓ 已刪除 %s\n"},
	"kept_dir":           {"  Kept %s\n", "  保留 %s\n"},
	"set_shown":          {"✓ Account %s is shown in the status line\n", "✓ 帳號 %s 會顯示在狀態列\n"},
	"set_hidden":         {"✓ Account %s is hidden from the status line (still shown while you use it)\n", "✓ 帳號 %s 不顯示在狀態列（正在使用時仍會顯示）\n"},
	"layout_set":         {"✓ Status line layout: %s (applies on the next refresh)\n", "✓ 狀態列版型：%s（下次狀態列更新時生效）\n"},
	"lang_set":           {"✓ Language: %s\n", "✓ 語言：%s\n"},
	"setup_title":        {"Setting up claude-accounts (main account: %s)\n", "設定 claude-accounts（主帳號：%s）\n"},
	"found_account":      {"✓ Found account %s at %s\n", "✓ 找到帳號 %s：%s\n"},
	"cleaned_block":      {"✓ Removed the old claude-dotfiles block from %s (backup: %s.bak)\n", "✓ 已移除 %s 裡舊的 claude-dotfiles 區塊（備份：%s.bak）\n"},
	"removed_file":       {"✓ Removed %s\n", "✓ 已移除 %s\n"},
	"statusline_set":     {"✓ Status line set in %s\n", "✓ 已在 %s 設定狀態列\n"},
	"setup_done":         {"\nDone. Open a new terminal and run %s; type /login the first time you use an account.\n", "\n完成。開新的終端機執行 %s；第一次使用某個帳號時輸入 /login。\n"},
	"uninstalled":        {"✓ Removed the claude-* commands and the status line. Account folders were kept.\n  To finish, delete %s\n", "✓ 已移除 claude-* 指令與狀態列，帳號資料夾都保留著。\n  最後請自行刪除 %s\n"},
	"invalid_choice":     {"Invalid choice\n", "無效的選擇\n"},

	// Errors
	"err_main":         {"account a is the main account and can't be removed", "帳號 a 是主帳號，不能移除"},
	"err_no_account":   {"no account %s (accounts: %s)", "沒有帳號 %s（目前的帳號：%s）"},
	"err_full":         {"all 26 letters are in use", "26 個字母都用完了"},
	"err_layout":       {"layout must be full or compact", "版型只能是 full 或 compact"},
	"err_lang":         {"language must be auto, en or zh-TW", "語言只能是 auto、en 或 zh-TW"},
	"err_no_claude":    {"can't find claude: install Claude Code, or set CLAUDE_BIN", "找不到 claude：請先安裝 Claude Code，或設定 CLAUDE_BIN"},
	"err_dir_missing":  {"account %s's folder %s is missing; run claude-accounts remove %s and add it again", "帳號 %s 的資料夾 %s 不見了；請執行 claude-accounts remove %s 再重新新增"},
	"err_unknown_cmd":  {"unknown command %q; run claude-accounts help", "未知的指令 %q；請執行 claude-accounts help"},
	"err_usage_letter": {"usage: claude-accounts %s <letter>", "用法：claude-accounts %s <字母>"},
	"err_config":       {"can't read %s: %v", "無法讀取 %s：%v"},
	"err_not_set_up":   {"claude-accounts isn't set up yet; run claude-accounts first", "claude-accounts 還沒設定；請先執行 claude-accounts"},
}
```

- [ ] **Step 4: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/i18n/ && GOOS=windows go vet ./internal/i18n/`
Expected: `ok`，Windows 版 vet 無錯誤。

- [ ] **Step 5: Commit**

```bash
git add internal/i18n
git commit -F - <<'EOF'
feat: 英文／繁中字串表與語系判斷

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 3: config（帳號清單、字母分配、路徑）

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`

**Interfaces:**
- Consumes: `fsutil.ReadJSON`、`fsutil.WriteJSON`
- Produces:
  - 常數 `FileName = "claude-accounts.json"`、`TypeSubscription = "subscription"`、`TypeAPI = "api"`、`LayoutFull = "full"`、`LayoutCompact = "compact"`、`MainName = "a"`；`var ErrFull error`
  - `type Account struct { Name, Dir, Type string; StatusLine bool }`（JSON：`name`、`dir`、`type`、`statusline`）、`(Account).Path() string`
  - `type Config struct { Version int; Layout, Lang string; Accounts []Account }`
  - `New(mainDir string) *Config`、`Load(mainDir string) (*Config, error)`、`(*Config).Save(mainDir string) error`
  - `(*Config).Find(name string) *Account`、`Names() []string`、`NextLetter() (string, error)`、`Add(a Account)`、`Delete(name string)`
  - `Home() string`、`ExpandHome(p string) string`、`CollapseHome(p string) string`、`SamePath(a, b string) bool`、`ActiveDir() string`、`ResolveMainDir(override string) string`

- [ ] **Step 1: 寫失敗的測試** `internal/config/config_test.go`

```go
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	return home
}

func TestNextLetterFillsGaps(t *testing.T) {
	c := New("/tmp/main")
	for _, n := range []string{"b", "c", "d"} {
		c.Add(Account{Name: n, Dir: "/x-" + n, Type: TypeSubscription})
	}
	c.Delete("c")
	if got, err := c.NextLetter(); err != nil || got != "c" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestNextLetterFull(t *testing.T) {
	c := New("/tmp/main")
	for ch := 'b'; ch <= 'z'; ch++ {
		c.Add(Account{Name: string(ch)})
	}
	if _, err := c.NextLetter(); !errors.Is(err, ErrFull) {
		t.Fatalf("got %v", err)
	}
}

func TestAddKeepsMainFirstAndSorts(t *testing.T) {
	c := New("/tmp/main")
	c.Add(Account{Name: "d"})
	c.Add(Account{Name: "b"})
	if got := c.Names(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "d" {
		t.Fatalf("got %v", got)
	}
}

func TestExpandCollapseHome(t *testing.T) {
	home := setHome(t)
	p := filepath.Join(home, ".claude-b")
	if got := CollapseHome(p); got != "~/.claude-b" {
		t.Fatalf("collapse: %q", got)
	}
	if got := ExpandHome("~/.claude-b"); !SamePath(got, p) {
		t.Fatalf("expand: %q", got)
	}
	other := filepath.Join(t.TempDir(), "x")
	if got := CollapseHome(other); got != other {
		t.Fatalf("outside home: %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := setHome(t)
	main := filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	c := New(main)
	c.Layout = LayoutCompact
	c.Add(Account{Name: "c", Dir: "~/.claude-c", Type: TypeAPI, StatusLine: false})
	if err := c.Save(main); err != nil {
		t.Fatal(err)
	}
	got, err := Load(main)
	if err != nil {
		t.Fatal(err)
	}
	if got.Layout != LayoutCompact || got.Lang != "auto" || got.Find("c").Type != TypeAPI || got.Find("a").Dir != "~/.claude" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

// Review Focus 2: a broken file is an error, never silently replaced.
func TestLoadRejectsBadFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte("{oops"), 0o644)
	if _, err := Load(dir); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"accounts":[{"name":"b"}]}`), 0o644)
	if _, err := Load(dir); err == nil {
		t.Fatal("a config whose first account isn't a must be rejected")
	}
}

// Review Focus 1: inside account b, the main account is found through b's links.
func TestResolveMainDirFromLinkedAccount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a symlink; junctions are covered by the accounts tests")
	}
	home := setHome(t)
	main := filepath.Join(home, ".claude")
	b := main + "-b"
	os.MkdirAll(filepath.Join(main, "projects"), 0o755)
	os.MkdirAll(b, 0o755)
	os.WriteFile(filepath.Join(main, FileName), []byte(`{}`), 0o644)
	if err := os.Symlink(filepath.Join(main, "projects"), filepath.Join(b, "projects")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", b)
	if got := ResolveMainDir(""); !SamePath(got, main) {
		t.Fatalf("got %q, want %q", got, main)
	}
	if got := ResolveMainDir("~/elsewhere"); !SamePath(got, filepath.Join(home, "elsewhere")) {
		t.Fatalf("override: %q", got)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if got := ResolveMainDir(""); !SamePath(got, main) {
		t.Fatalf("default: %q", got)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/config/`
Expected: FAIL，`undefined: New`。

- [ ] **Step 3: 實作** `internal/config/config.go`

```go
// Package config reads and writes claude-accounts.json, the list of accounts that
// every command link, shared folder and the status line are built from.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

const (
	FileName         = "claude-accounts.json"
	TypeSubscription = "subscription"
	TypeAPI          = "api"
	LayoutFull       = "full"
	LayoutCompact    = "compact"
	MainName         = "a"
)

var ErrFull = errors.New("all 26 letters are in use")

type Account struct {
	Name       string `json:"name"`
	Dir        string `json:"dir"`
	Type       string `json:"type"`
	StatusLine bool   `json:"statusline"`
}

// Path is the account's config dir with ~ expanded.
func (a Account) Path() string { return ExpandHome(a.Dir) }

type Config struct {
	Version  int       `json:"version"`
	Layout   string    `json:"layout"`
	Lang     string    `json:"lang"`
	Accounts []Account `json:"accounts"` // Accounts[0] is the main account "a"
}

func New(mainDir string) *Config {
	return &Config{Version: 1, Layout: LayoutFull, Lang: "auto", Accounts: []Account{
		{Name: MainName, Dir: CollapseHome(mainDir), Type: TypeSubscription, StatusLine: true},
	}}
}

func Load(mainDir string) (*Config, error) {
	var c Config
	path := filepath.Join(mainDir, FileName)
	if err := fsutil.ReadJSON(path, &c); err != nil {
		return nil, err
	}
	if len(c.Accounts) == 0 || c.Accounts[0].Name != MainName {
		return nil, fmt.Errorf("%s: the first account must be %q", path, MainName)
	}
	if c.Layout != LayoutCompact {
		c.Layout = LayoutFull
	}
	if c.Lang == "" {
		c.Lang = "auto"
	}
	return &c, nil
}

func (c *Config) Save(mainDir string) error {
	c.Version = 1
	return fsutil.WriteJSON(filepath.Join(mainDir, FileName), c, true)
}

func (c *Config) Find(name string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].Name == name {
			return &c.Accounts[i]
		}
	}
	return nil
}

func (c *Config) Names() []string {
	names := make([]string, len(c.Accounts))
	for i, a := range c.Accounts {
		names[i] = a.Name
	}
	return names
}

// NextLetter is the first free letter from b to z, so removed letters get reused.
func (c *Config) NextLetter() (string, error) {
	for ch := 'b'; ch <= 'z'; ch++ {
		if c.Find(string(ch)) == nil {
			return string(ch), nil
		}
	}
	return "", ErrFull
}

// Add appends a and keeps the accounts after the main one in letter order.
func (c *Config) Add(a Account) {
	c.Accounts = append(c.Accounts, a)
	rest := c.Accounts[1:]
	sort.Slice(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
}

func (c *Config) Delete(name string) {
	for i, a := range c.Accounts {
		if a.Name == name {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return
		}
	}
}

func Home() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func ExpandHome(p string) string {
	if p == "~" {
		return Home()
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(Home(), p[2:])
	}
	return p
}

// CollapseHome writes a path under the home directory as ~/..., which keeps
// claude-accounts.json readable and portable between machines.
func CollapseHome(p string) string {
	abs := clean(p)
	rel, err := filepath.Rel(clean(Home()), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return abs
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}

func clean(p string) string {
	if abs, err := filepath.Abs(ExpandHome(p)); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

func SamePath(a, b string) bool {
	a, b = clean(a), clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// ActiveDir is the config dir of the Claude that is running now.
func ActiveDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return clean(dir)
	}
	return clean(filepath.Join(Home(), ".claude"))
}

// ResolveMainDir finds the main account's dir. Inside another account (claude-b sets
// CLAUDE_CONFIG_DIR to its own dir) the main one is where its shared folders point.
func ResolveMainDir(override string) string {
	if override != "" {
		return clean(override)
	}
	start := ActiveDir()
	if _, err := os.Stat(filepath.Join(start, FileName)); err == nil {
		return start
	}
	if target, err := os.Readlink(filepath.Join(start, "projects")); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(start, target)
		}
		return filepath.Dir(clean(target))
	}
	return start
}
```

- [ ] **Step 4: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/config/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -F - <<'EOF'
feat: 帳號清單 claude-accounts.json、字母分配與主帳號目錄判斷

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 4: links 與共用資料夾

**Files:**
- Create: `internal/links/links.go`, `internal/links/links_unix.go`, `internal/links/links_windows.go`, `internal/accounts/share.go`, `internal/accounts/share_test.go`

**Interfaces:**
- Produces:
  - `links.IsLink(path string) bool`、`links.Points(link, target string) bool`
  - `links.Dir(link, target string) error`（Windows junction／其他 symlink）
  - `links.Exe(exe, link string) error`（Windows hard link，失敗改複製；其他 symlink；已指向 exe 時不動）
  - `links.RemoveExe(path string) error`（不存在不算錯；Windows 使用中就改名成 `.old`）
  - `accounts.SharedDirs []string`、`accounts.Share(mainDir, accDir string) (conflicts []string, err error)`、`accounts.RemoveDir(dir string) error`

- [ ] **Step 1: 寫失敗的測試** `internal/accounts/share_test.go`

```go
package accounts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/links"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShareCreatesLinks(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	conflicts, err := Share(main, acc)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	for _, name := range SharedDirs {
		if !links.IsLink(filepath.Join(acc, name)) || !links.Points(filepath.Join(acc, name), filepath.Join(main, name)) {
			t.Errorf("%s is not linked", name)
		}
	}
	if _, err := Share(main, acc); err != nil {
		t.Fatalf("second run: %v", err)
	}
}

func TestShareMergesExistingFolder(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	write(t, filepath.Join(main, "projects", "p1.txt"), "main")
	write(t, filepath.Join(acc, "projects", "p2.txt"), "acc")
	if conflicts, err := Share(main, acc); err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	for _, f := range []string{"p1.txt", "p2.txt"} {
		if _, err := os.Stat(filepath.Join(main, "projects", f)); err != nil {
			t.Errorf("%s missing from main", f)
		}
	}
	if !links.IsLink(filepath.Join(acc, "projects")) {
		t.Error("projects should now be a link")
	}
}

func TestShareKeepsConflicts(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	write(t, filepath.Join(main, "projects", "x.txt"), "main")
	write(t, filepath.Join(acc, "projects", "x.txt"), "acc")
	conflicts, err := Share(main, acc)
	if err != nil || len(conflicts) != 1 || conflicts[0] != "projects" {
		t.Fatalf("%v %v", conflicts, err)
	}
	if data, _ := os.ReadFile(filepath.Join(main, "projects", "x.txt")); string(data) != "main" {
		t.Error("main's file was overwritten")
	}
	if links.IsLink(filepath.Join(acc, "projects")) {
		t.Error("a conflicting folder must stay as it is")
	}
}

func TestRemoveDirKeepsMainData(t *testing.T) {
	root := t.TempDir()
	main, acc := filepath.Join(root, "main"), filepath.Join(root, "acc")
	if _, err := Share(main, acc); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(main, "projects", "keep.txt"), "x")
	write(t, filepath.Join(acc, ".credentials.json"), "{}")
	if err := RemoveDir(acc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(acc); !os.IsNotExist(err) {
		t.Error("account dir still exists")
	}
	if _, err := os.Stat(filepath.Join(main, "projects", "keep.txt")); err != nil {
		t.Error("main account data was deleted")
	}
	if err := RemoveDir(acc); err != nil {
		t.Errorf("removing a missing dir: %v", err)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/accounts/`
Expected: FAIL，`undefined: Share`。

- [ ] **Step 3: 實作 links**

`internal/links/links.go`：
```go
// Package links makes the directory links that share data between accounts and the
// executable links that become the claude-<letter> commands.
package links

import "os"

// IsLink reports whether path is a symlink or, on Windows, a junction.
func IsLink(path string) bool {
	_, err := os.Readlink(path)
	return err == nil
}

// Points reports whether link resolves to the same file or directory as target.
func Points(link, target string) bool {
	a, err := os.Stat(link)
	if err != nil {
		return false
	}
	b, err := os.Stat(target)
	return err == nil && os.SameFile(a, b)
}
```

`internal/links/links_unix.go`：
```go
//go:build !windows

package links

import (
	"errors"
	"io/fs"
	"os"
)

func Dir(link, target string) error { return os.Symlink(target, link) }

func Exe(exe, link string) error {
	if IsLink(link) && Points(link, exe) {
		return nil
	}
	if err := RemoveExe(link); err != nil {
		return err
	}
	return os.Symlink(exe, link)
}

func RemoveExe(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
```

`internal/links/links_windows.go`：
```go
//go:build windows

package links

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

// Dir makes a junction, which unlike a symlink needs no admin rights or developer mode.
func Dir(link, target string) error {
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J %s: %v: %s", link, err, out)
	}
	return nil
}

// Exe hard-links exe as link. After an update the old links still point at the old
// file, so anything that isn't the current exe is replaced.
func Exe(exe, link string) error {
	if Points(link, exe) {
		return nil
	}
	if err := RemoveExe(link); err != nil {
		return err
	}
	if err := os.Link(exe, link); err == nil {
		return nil
	}
	return copyFile(exe, link)
}

// RemoveExe deletes path; a running claude-x.exe can't be deleted but can be renamed.
func RemoveExe(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	old := path + ".old"
	os.Remove(old)
	return os.Rename(path, old)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
```

- [ ] **Step 4: 實作** `internal/accounts/share.go`

```go
// Package accounts adds and removes accounts: their folders, the links to the main
// account's shared data and their claude-<letter> commands.
package accounts

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MFpizza/claude-dotfiles/internal/links"
)

// SharedDirs live in the main account; every other account links to them.
// sessions/ is the live-session registry ListAgents and SendMessage read, so sharing
// it lets sessions of every account find and message each other.
var SharedDirs = []string{"projects", "skills", "agents", "commands", "plugins", "file-history", "sessions"}

// Share links accDir's shared folders to mainDir's. A real folder already there is
// folded into the main account's first; names that clash are left alone and returned.
func Share(mainDir, accDir string) ([]string, error) {
	if err := os.MkdirAll(accDir, 0o755); err != nil {
		return nil, err
	}
	var conflicts []string
	for _, name := range SharedDirs {
		target, link := filepath.Join(mainDir, name), filepath.Join(accDir, name)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return conflicts, err
		}
		if _, err := os.Lstat(link); err == nil {
			if links.Points(link, target) {
				continue
			}
			if links.IsLink(link) {
				if err := os.Remove(link); err != nil {
					return conflicts, err
				}
			} else if merged, err := mergeInto(link, target); err != nil {
				return conflicts, err
			} else if !merged {
				conflicts = append(conflicts, name)
				continue
			}
		}
		if err := links.Dir(link, target); err != nil {
			return conflicts, err
		}
	}
	return conflicts, nil
}

// mergeInto moves src's entries into dst without overwriting, and removes src if that
// emptied it.
func mergeInto(src, dst string) (bool, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		to := filepath.Join(dst, e.Name())
		if _, err := os.Lstat(to); errors.Is(err, fs.ErrNotExist) {
			if err := os.Rename(filepath.Join(src, e.Name()), to); err != nil {
				return false, err
			}
		}
	}
	if left, err := os.ReadDir(src); err != nil || len(left) > 0 {
		return false, err
	}
	return true, os.Remove(src)
}

// RemoveDir deletes an account dir. Its links are removed first so nothing under
// them, which is the main account's data, is ever touched.
func RemoveDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if p := filepath.Join(dir, e.Name()); links.IsLink(p) {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return os.RemoveAll(dir)
}
```

- [ ] **Step 5: 確認測試通過（含 Windows 編譯檢查）**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/accounts/ ./internal/links/ && GOOS=windows go vet ./internal/links/ ./internal/accounts/`
Expected: `ok`，Windows vet 無錯誤。

- [ ] **Step 6: Commit**

```bash
git add internal/links internal/accounts
git commit -F - <<'EOF'
feat: 共用資料夾連結（junction／symlink）與安全刪除帳號目錄

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 5: 新增／移除帳號、指令連結與 settings.json

**Files:**
- Create: `internal/accounts/accounts.go`, `internal/accounts/accounts_test.go`, `internal/settings/settings.go`, `internal/settings/settings_test.go`

**Interfaces:**
- Consumes: Task 3 的 `config.*`；Task 4 的 `links.Exe`、`links.RemoveExe`、`accounts.Share`、`accounts.RemoveDir`；`fsutil.WriteFileAtomic`
- Produces:
  - `accounts.ErrMain`、`accounts.ErrNoAccount`
  - `accounts.CommandPath(dir, name string) string`（Windows 加 `.exe`）
  - `accounts.SyncCommands(cfg *config.Config, exe string) error`（`exe == ""` 時什麼都不做）
  - `accounts.Add(cfg *config.Config, mainDir, exe string, api bool, dir string) (config.Account, []string, error)`
  - `accounts.Remove(cfg *config.Config, exe, name string, deleteDir bool) error`
  - `settings.Command(exe, mainDir string) string`、`settings.SetStatusLine(mainDir, exe string) (changed bool, err error)`、`settings.RemoveStatusLine(mainDir string) error`

- [ ] **Step 1: 寫失敗的測試** `internal/accounts/accounts_test.go`

```go
package accounts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

func setup(t *testing.T) (cfg *config.Config, main, exe string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	main = filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	exe = filepath.Join(home, "bin", "claude-accounts")
	write(t, exe, "binary")
	return config.New(main), main, exe
}

func TestAddAssignsLettersAndCommands(t *testing.T) {
	cfg, main, exe := setup(t)
	b, conflicts, err := Add(cfg, main, exe, false, "")
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	if b.Name != "b" || b.Type != config.TypeSubscription || !b.StatusLine || b.Dir != "~/.claude-b" {
		t.Fatalf("got %+v", b)
	}
	if !links.Points(CommandPath(filepath.Dir(exe), "b"), exe) {
		t.Error("claude-b command is missing")
	}
	if !links.Points(filepath.Join(main+"-b", "projects"), filepath.Join(main, "projects")) {
		t.Error("shared folders are not linked")
	}
	c, _, err := Add(cfg, main, exe, true, "")
	if err != nil || c.Name != "c" || c.Type != config.TypeAPI {
		t.Fatalf("got %+v %v", c, err)
	}
}

func TestAddCustomDir(t *testing.T) {
	cfg, main, _ := setup(t)
	dir := filepath.Join(t.TempDir(), "work")
	acc, _, err := Add(cfg, main, "", false, dir)
	if err != nil || !config.SamePath(acc.Path(), dir) {
		t.Fatalf("got %+v %v", acc, err)
	}
}

func TestRemoveKeepsOrDeletesDir(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	Add(cfg, main, exe, false, "")
	if err := Remove(cfg, exe, "b", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(main + "-b"); err != nil {
		t.Error("b's dir should be kept")
	}
	if _, err := os.Lstat(CommandPath(filepath.Dir(exe), "b")); !os.IsNotExist(err) {
		t.Error("claude-b command should be gone")
	}
	if err := Remove(cfg, exe, "c", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(main + "-c"); !os.IsNotExist(err) {
		t.Error("c's dir should be deleted")
	}
	if _, err := os.Stat(filepath.Join(main, "projects")); err != nil {
		t.Error("main account data was touched")
	}
	if got := cfg.Names(); len(got) != 1 {
		t.Fatalf("accounts left: %v", got)
	}
}

func TestRemoveRefusals(t *testing.T) {
	cfg, _, exe := setup(t)
	if err := Remove(cfg, exe, "a", false); !errors.Is(err, ErrMain) {
		t.Errorf("main: %v", err)
	}
	if err := Remove(cfg, exe, "q", false); !errors.Is(err, ErrNoAccount) {
		t.Errorf("unknown: %v", err)
	}
}

// Review Focus 4: a dir deleted by hand still lets the account be removed.
func TestRemoveMissingDir(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	if err := os.RemoveAll(main + "-b"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(cfg, exe, "b", true); err != nil {
		t.Fatal(err)
	}
	if cfg.Find("b") != nil {
		t.Error("b is still listed")
	}
}

func TestSyncCommandsRecreatesMissing(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	os.Remove(CommandPath(filepath.Dir(exe), "b"))
	if err := SyncCommands(cfg, exe); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a", "b"} {
		if !links.Points(CommandPath(filepath.Dir(exe), n), exe) {
			t.Errorf("claude-%s missing", n)
		}
	}
}
```

`internal/settings/settings_test.go`：
```go
package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Review Focus 5: the user's own settings keep their order and content.
func TestSetStatusLinePreservesOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	orig := `{"theme": "dark", "model": "opus", "hooks": {"Stop": [1, 2]}}`
	os.WriteFile(path, []byte(orig), 0o644)
	changed, err := SetStatusLine(dir, "/opt/bin/claude-accounts")
	if err != nil || !changed {
		t.Fatalf("%v %v", changed, err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	order := []string{`"theme"`, `"model"`, `"hooks"`, `"statusLine"`}
	last := -1
	for _, k := range order {
		i := strings.Index(text, k)
		if i <= last {
			t.Fatalf("key order changed:\n%s", text)
		}
		last = i
	}
	if !strings.Contains(text, `"command": "\"/opt/bin/claude-accounts\" statusline --base`) {
		t.Fatalf("command missing:\n%s", text)
	}
	if bak, _ := os.ReadFile(path + ".bak"); string(bak) != orig {
		t.Fatalf("backup: %s", bak)
	}
	if changed, err := SetStatusLine(dir, "/opt/bin/claude-accounts"); err != nil || changed {
		t.Fatalf("second run should be a no-op: %v %v", changed, err)
	}
}

func TestSetStatusLineCreatesFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	if _, err := SetStatusLine(dir, "/x/claude-accounts"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json.bak")); !os.IsNotExist(err) {
		t.Error("nothing to back up")
	}
}

func TestSetStatusLineRejectsBadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("{nope"), 0o644)
	if _, err := SetStatusLine(dir, "/x"); err == nil {
		t.Fatal("expected an error")
	}
	if data, _ := os.ReadFile(path); string(data) != "{nope" {
		t.Fatal("a broken file must not be overwritten")
	}
}

func TestRemoveStatusLine(t *testing.T) {
	dir := t.TempDir()
	SetStatusLine(dir, "/x")
	if err := RemoveStatusLine(dir); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "settings.json"))
	if strings.Contains(string(data), "statusLine") {
		t.Fatalf("still there: %s", data)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/accounts/ ./internal/settings/`
Expected: FAIL，`undefined: Add`、`undefined: SetStatusLine`。

- [ ] **Step 3: 實作** `internal/accounts/accounts.go`

```go
package accounts

import (
	"errors"
	"path/filepath"
	"runtime"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

var (
	ErrMain      = errors.New("the main account can't be removed")
	ErrNoAccount = errors.New("no such account")
)

// CommandPath is where the claude-<name> command for an account lives: next to the
// executable, which is on PATH since it runs.
func CommandPath(dir, name string) string {
	file := "claude-" + name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	return filepath.Join(dir, file)
}

// SyncCommands makes sure every account's command points at the current executable.
func SyncCommands(cfg *config.Config, exe string) error {
	if exe == "" {
		return nil
	}
	for _, acc := range cfg.Accounts {
		if err := links.Exe(exe, CommandPath(filepath.Dir(exe), acc.Name)); err != nil {
			return err
		}
	}
	return nil
}

// Add creates the next lettered account. The caller saves cfg.
func Add(cfg *config.Config, mainDir, exe string, api bool, dir string) (config.Account, []string, error) {
	name, err := cfg.NextLetter()
	if err != nil {
		return config.Account{}, nil, err
	}
	if dir == "" {
		dir = mainDir + "-" + name
	}
	if dir, err = filepath.Abs(config.ExpandHome(dir)); err != nil {
		return config.Account{}, nil, err
	}
	conflicts, err := Share(mainDir, dir)
	if err != nil {
		return config.Account{}, conflicts, err
	}
	typ := config.TypeSubscription
	if api {
		typ = config.TypeAPI
	}
	acc := config.Account{Name: name, Dir: config.CollapseHome(dir), Type: typ, StatusLine: true}
	cfg.Add(acc)
	if exe != "" {
		if err := links.Exe(exe, CommandPath(filepath.Dir(exe), name)); err != nil {
			return acc, conflicts, err
		}
	}
	return acc, conflicts, nil
}

// Remove drops an account and its command; deleteDir also deletes its folder.
// The caller saves cfg.
func Remove(cfg *config.Config, exe, name string, deleteDir bool) error {
	if name == config.MainName {
		return ErrMain
	}
	acc := cfg.Find(name)
	if acc == nil {
		return ErrNoAccount
	}
	dir := acc.Path()
	if exe != "" {
		if err := links.RemoveExe(CommandPath(filepath.Dir(exe), name)); err != nil {
			return err
		}
	}
	cfg.Delete(name)
	if !deleteDir {
		return nil
	}
	return RemoveDir(dir)
}
```

- [ ] **Step 4: 實作** `internal/settings/settings.go`

```go
// Package settings edits the statusLine entry of Claude Code's settings.json while
// leaving the user's other settings, and their order, exactly as they were.
package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

// Command is the status line command. Forward slashes work in every shell Claude
// Code uses, including on Windows.
func Command(exe, mainDir string) string {
	return fmt.Sprintf(`"%s" statusline --base "%s"`, filepath.ToSlash(exe), filepath.ToSlash(mainDir))
}

func SetStatusLine(mainDir, exe string) (bool, error) {
	path, data, obj, err := load(mainDir)
	if err != nil {
		return false, err
	}
	want := Command(exe, mainDir)
	var cur struct {
		Command string `json:"command"`
	}
	if raw, ok := obj.vals["statusLine"]; ok && json.Unmarshal(raw, &cur) == nil && cur.Command == want {
		return false, nil
	}
	val, _ := json.Marshal(map[string]any{"type": "command", "command": want, "padding": 0})
	obj.set("statusLine", val)
	return true, save(path, data, obj)
}

func RemoveStatusLine(mainDir string) error {
	path, data, obj, err := load(mainDir)
	if err != nil {
		return err
	}
	if _, ok := obj.vals["statusLine"]; !ok {
		return nil
	}
	obj.del("statusLine")
	return save(path, data, obj)
}

func load(mainDir string) (string, []byte, *object, error) {
	path := filepath.Join(mainDir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return path, nil, nil, err
	}
	obj, err := parse(data)
	if err != nil {
		return path, nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	return path, data, obj, nil
}

// save keeps the previous file as settings.json.bak before writing.
func save(path string, old []byte, obj *object) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if old != nil {
		if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
			return err
		}
	}
	return fsutil.WriteFileAtomic(path, obj.marshal(), 0o644)
}

// object is a JSON object that remembers its key order.
type object struct {
	keys []string
	vals map[string]json.RawMessage
}

func parse(data []byte) (*object, error) {
	obj := &object{vals: map[string]json.RawMessage{}}
	if len(bytes.TrimSpace(data)) == 0 {
		return obj, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, errors.New("not a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		obj.set(tok.(string), raw)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return obj, nil
}

func (o *object) set(key string, val json.RawMessage) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = val
}

func (o *object) del(key string) {
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			return
		}
	}
}

func (o *object) marshal() []byte {
	var b bytes.Buffer
	b.WriteString("{")
	for i, k := range o.keys {
		if i > 0 {
			b.WriteString(",")
		}
		key, _ := json.Marshal(k)
		b.WriteString("\n  ")
		b.Write(key)
		b.WriteString(": ")
		if err := json.Indent(&b, o.vals[k], "  ", "  "); err != nil {
			b.Write(o.vals[k])
		}
	}
	if len(o.keys) > 0 {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.Bytes()
}
```

- [ ] **Step 5: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/accounts/ ./internal/settings/ && GOOS=windows go vet ./internal/...`
Expected: `ok`，Windows vet 無錯誤。

- [ ] **Step 6: Commit**

```bash
git add internal/accounts internal/settings
git commit -F - <<'EOF'
feat: 新增／移除帳號、claude-<字母> 指令連結與 statusLine 設定

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 6: 啟動帳號（launch）

**Files:**
- Create: `internal/launch/launch.go`, `internal/launch/exec_unix.go`, `internal/launch/exec_windows.go`, `internal/launch/launch_test.go`

**Interfaces:**
- Consumes: `config.*`、`accounts.ErrNoAccount`、`fsutil.WriteFileAtomic`
- Produces: `launch.ErrNoClaude`、`type launch.DirMissingError struct{ Name, Dir string }`、`type launch.Plan struct{ Path string; Argv, Env []string }`、`launch.Build(cfg *config.Config, mainDir, name string, args []string, lookPath func(string) (string, error)) (*Plan, error)`、`launch.Exec(p *Plan) error`

- [ ] **Step 1: 寫失敗的測試** `internal/launch/launch_test.go`

```go
package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
)

func fakeLook(found map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if p, ok := found[name]; ok {
			return p, nil
		}
		return "", errors.New("not found")
	}
}

func setup(t *testing.T) (*config.Config, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_BIN", "")
	main := filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	os.MkdirAll(main+"-b", 0o755)
	os.WriteFile(filepath.Join(main, "settings.json"), []byte(`{"theme":"dark"}`), 0o644)
	cfg := config.New(main)
	cfg.Add(config.Account{Name: "b", Dir: "~/.claude-b", Type: config.TypeSubscription, StatusLine: true})
	return cfg, main
}

func envValue(env []string, key string) (string, bool) {
	for _, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			return kv[len(key)+1:], true
		}
	}
	return "", false
}

func TestBuildSyncsSettingsAndSetsDir(t *testing.T) {
	cfg, main := setup(t)
	p, err := Build(cfg, main, "b", []string{"--continue"}, fakeLook(map[string]string{"claude": "/x/claude"}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Path != "/x/claude" || strings.Join(p.Argv, " ") != "/x/claude --continue" {
		t.Fatalf("got %+v", p)
	}
	if v, _ := envValue(p.Env, "CLAUDE_CONFIG_DIR"); !config.SamePath(v, main+"-b") {
		t.Fatalf("CLAUDE_CONFIG_DIR=%q", v)
	}
	if data, _ := os.ReadFile(filepath.Join(main+"-b", "settings.json")); string(data) != `{"theme":"dark"}` {
		t.Fatalf("settings not synced: %s", data)
	}
}

func TestBuildDefaultMainDirUnsetsVar(t *testing.T) {
	cfg, main := setup(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "/somewhere/else")
	p, err := Build(cfg, main, "a", nil, fakeLook(map[string]string{"claude": "/x/claude"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envValue(p.Env, "CLAUDE_CONFIG_DIR"); ok {
		t.Fatal("~/.claude must run with CLAUDE_CONFIG_DIR unset")
	}
}

func TestBuildAccountSettings(t *testing.T) {
	cfg, main := setup(t)
	own := filepath.Join(main+"-b", "account-settings.json")
	os.WriteFile(own, []byte(`{}`), 0o600)
	p, _ := Build(cfg, main, "b", []string{"-p", "hi"}, fakeLook(map[string]string{"claude": "/x/claude"}))
	if len(p.Argv) != 5 || p.Argv[1] != "--settings" || !config.SamePath(p.Argv[2], own) || p.Argv[3] != "-p" {
		t.Fatalf("got %v", p.Argv)
	}
}

func TestBuildFindsClaude(t *testing.T) {
	cfg, main := setup(t)
	t.Setenv("CLAUDE_BIN", "/opt/claude")
	if p, _ := Build(cfg, main, "b", nil, fakeLook(map[string]string{"claude": "/x/claude"})); p.Path != "/opt/claude" {
		t.Errorf("CLAUDE_BIN should win: %v", p.Path)
	}
	t.Setenv("CLAUDE_BIN", "")
	p, _ := Build(cfg, main, "b", []string{"-c"}, fakeLook(map[string]string{"npx": "/n/npx"}))
	if strings.Join(p.Argv, " ") != "/n/npx --no -- claude -c" {
		t.Errorf("npx fallback: %v", p.Argv)
	}
	if _, err := Build(cfg, main, "b", nil, fakeLook(nil)); !errors.Is(err, ErrNoClaude) {
		t.Errorf("nothing found: %v", err)
	}
}

func TestBuildUnknownAccount(t *testing.T) {
	cfg, main := setup(t)
	if _, err := Build(cfg, main, "q", nil, fakeLook(nil)); !errors.Is(err, accounts.ErrNoAccount) {
		t.Fatalf("got %v", err)
	}
}

// Review Focus 4: a folder deleted by hand gives a clear error.
func TestBuildDirMissing(t *testing.T) {
	cfg, main := setup(t)
	os.RemoveAll(main + "-b")
	_, err := Build(cfg, main, "b", nil, fakeLook(map[string]string{"claude": "/x/claude"}))
	var missing *DirMissingError
	if !errors.As(err, &missing) || missing.Name != "b" {
		t.Fatalf("got %v", err)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/launch/`
Expected: FAIL，`undefined: Build`。

- [ ] **Step 3: 實作** `internal/launch/launch.go`

```go
// Package launch starts Claude Code as one of the accounts.
package launch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

var ErrNoClaude = errors.New("claude not found")

type DirMissingError struct{ Name, Dir string }

func (e *DirMissingError) Error() string {
	return fmt.Sprintf("account %s: %s is missing", e.Name, e.Dir)
}

// Plan is the program, argv and environment that Exec runs.
type Plan struct {
	Path string
	Argv []string
	Env  []string
}

func Build(cfg *config.Config, mainDir, name string, args []string, lookPath func(string) (string, error)) (*Plan, error) {
	acc := cfg.Find(name)
	if acc == nil {
		return nil, accounts.ErrNoAccount
	}
	dir := acc.Path()
	if _, err := os.Stat(dir); err != nil {
		return nil, &DirMissingError{Name: name, Dir: acc.Dir}
	}
	if !config.SamePath(dir, mainDir) {
		// Settings flow one way, from the main account to the others.
		if data, err := os.ReadFile(filepath.Join(mainDir, "settings.json")); err == nil {
			fsutil.WriteFileAtomic(filepath.Join(dir, "settings.json"), data, 0o644)
		}
	}
	if own := filepath.Join(dir, "account-settings.json"); fileExists(own) {
		args = append([]string{"--settings", own}, args...)
	}
	env := withoutVar(os.Environ(), "CLAUDE_CONFIG_DIR")
	// For ~/.claude leave the variable unset, so claude-a and plain claude share
	// ~/.claude.json (with it set, Claude reads <dir>/.claude.json instead).
	if !config.SamePath(dir, filepath.Join(config.Home(), ".claude")) {
		env = append(env, "CLAUDE_CONFIG_DIR="+dir)
	}
	if bin := os.Getenv("CLAUDE_BIN"); bin != "" {
		return &Plan{Path: bin, Argv: append([]string{bin}, args...), Env: env}, nil
	}
	if p, err := lookPath("claude"); err == nil {
		return &Plan{Path: p, Argv: append([]string{p}, args...), Env: env}, nil
	}
	// Claude installed per project and started with `npx claude`; --no never downloads.
	if p, err := lookPath("npx"); err == nil {
		return &Plan{Path: p, Argv: append([]string{p, "--no", "--", "claude"}, args...), Env: env}, nil
	}
	return nil, ErrNoClaude
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func withoutVar(env []string, name string) []string {
	var out []string
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if key == name || (runtime.GOOS == "windows" && strings.EqualFold(key, name)) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
```

`internal/launch/exec_unix.go`：
```go
//go:build !windows

package launch

import "syscall"

// Exec replaces this process with Claude, so signals and the exit code are Claude's own.
func Exec(p *Plan) error { return syscall.Exec(p.Path, p.Argv, p.Env) }
```

`internal/launch/exec_windows.go`：
```go
//go:build windows

package launch

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// Exec runs Claude and exits with its code. Ctrl+C goes to Claude; this process
// ignores it and just waits.
func Exec(p *Plan) error {
	cmd := exec.Command(p.Path, p.Argv[1:]...)
	cmd.Env = p.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
```

- [ ] **Step 4: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/launch/ && GOOS=windows go vet ./internal/launch/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/launch
git commit -F - <<'EOF'
feat: 以指定帳號啟動 claude（同步設定、account-settings、npx 備援）

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 7: 從舊版遷移（migrate）

**Files:**
- Create: `internal/migrate/migrate.go`, `internal/migrate/shell_unix.go`, `internal/migrate/shell_windows.go`, `internal/migrate/migrate_test.go`

**Interfaces:**
- Consumes: `config.*`、`accounts.SharedDirs`、`accounts.Share`（測試用）、`links.IsLink`、`links.Points`、`fsutil.*`
- Produces: `migrate.Begin`、`migrate.End`、`migrate.Detect(mainDir string) *config.Config`、`migrate.LegacyLayout(mainDir string) string`、`migrate.RemoveLegacyLayout(mainDir string)`、`migrate.StripBlock(path string) (bool, error)`、`migrate.OldShims(home string) []string`、`migrate.ShellFiles(home string) []string`

- [ ] **Step 1: 寫失敗的測試** `internal/migrate/migrate_test.go`

```go
package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
)

func TestDetectFindsLinkedAccounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	main := filepath.Join(home, ".claude")
	for _, s := range []string{"-b", "-c"} {
		if _, err := accounts.Share(main, main+s); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(main+"-x", 0o755)              // a real dir that isn't an account
	accounts.Share(main, main+"-bb")           // not a single letter
	cfg := Detect(main)
	if got := cfg.Names(); len(got) != 3 || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %v", got)
	}
	if cfg.Find("b").Type != config.TypeSubscription || cfg.Find("c").Type != config.TypeAPI {
		t.Fatalf("types: %+v", cfg.Accounts)
	}
	if cfg.Find("b").Dir != "~/.claude-b" {
		t.Fatalf("dir: %q", cfg.Find("b").Dir)
	}
}

func TestStripBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bashrc")
	orig := "export A=1\n\n" + Begin + "\nCLAUDE_DOTFILES_BASE='/x'\n. '/x/claude-accounts.sh'\n" + End + "\nexport B=2\n"
	os.WriteFile(path, []byte(orig), 0o644)
	ok, err := StripBlock(path)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "export A=1\n\nexport B=2\n" {
		t.Fatalf("got %q", data)
	}
	if bak, _ := os.ReadFile(path + ".bak"); string(bak) != orig {
		t.Fatal("backup missing")
	}
	if ok, err := StripBlock(path); ok || err != nil {
		t.Fatalf("second run: %v %v", ok, err)
	}
	if ok, err := StripBlock(filepath.Join(t.TempDir(), "missing")); ok || err != nil {
		t.Fatalf("missing file: %v %v", ok, err)
	}
}

func TestLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	if got := LegacyLayout(dir); got != "" {
		t.Fatalf("none: %q", got)
	}
	os.WriteFile(filepath.Join(dir, "statusline.json"), []byte(`{"layout":"compact"}`), 0o644)
	if got := LegacyLayout(dir); got != "compact" {
		t.Fatalf("got %q", got)
	}
	RemoveLegacyLayout(dir)
	if _, err := os.Stat(filepath.Join(dir, "statusline.json")); !os.IsNotExist(err) {
		t.Fatal("not removed")
	}
}

func TestOldShims(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "claude-b.cmd"), []byte("@call x"), 0o644)
	if got := OldShims(home); len(got) != 1 || filepath.Base(got[0]) != "claude-b.cmd" {
		t.Fatalf("got %v", got)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/migrate/`
Expected: FAIL，`undefined: Detect`。

- [ ] **Step 3: 實作** `internal/migrate/migrate.go`

```go
// Package migrate turns an install of the old Python/shell version (accounts A, B, C)
// into a claude-accounts.json and removes what the old installer added.
package migrate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

// Markers of the block the old installer put in $PROFILE and ~/.bashrc.
const (
	Begin = "# >>> claude-dotfiles >>>"
	End   = "# <<< claude-dotfiles <<<"
)

// Detect builds a config from <main>-<letter> dirs whose shared folders link to the
// main account. By the old convention c is the API-billed account.
func Detect(mainDir string) *config.Config {
	cfg := config.New(mainDir)
	for ch := 'b'; ch <= 'z'; ch++ {
		name := string(ch)
		dir := mainDir + "-" + name
		if !linksTo(dir, mainDir) {
			continue
		}
		typ := config.TypeSubscription
		if name == "c" {
			typ = config.TypeAPI
		}
		cfg.Add(config.Account{Name: name, Dir: config.CollapseHome(dir), Type: typ, StatusLine: true})
	}
	return cfg
}

func linksTo(dir, mainDir string) bool {
	for _, name := range accounts.SharedDirs {
		p := filepath.Join(dir, name)
		if links.IsLink(p) && links.Points(p, filepath.Join(mainDir, name)) {
			return true
		}
	}
	return false
}

// LegacyLayout is the layout saved by the old status line, or "".
func LegacyLayout(mainDir string) string {
	var v struct {
		Layout string `json:"layout"`
	}
	if fsutil.ReadJSON(filepath.Join(mainDir, "statusline.json"), &v) == nil &&
		(v.Layout == config.LayoutFull || v.Layout == config.LayoutCompact) {
		return v.Layout
	}
	return ""
}

func RemoveLegacyLayout(mainDir string) { os.Remove(filepath.Join(mainDir, "statusline.json")) }

// StripBlock removes the old installer's block from a shell or PowerShell startup
// file, keeping the original as <file>.bak. A missing file or block is not an error.
func StripBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil
	}
	text := string(data)
	i, j := strings.Index(text, Begin), strings.Index(text, End)
	if i < 0 || j < i {
		return false, nil
	}
	text = text[:i] + strings.TrimLeft(text[j+len(End):], "\r\n")
	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return false, err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return true, fsutil.WriteFileAtomic(path, []byte(text), mode)
}

// OldShims are the cmd.exe forwarders the old installer put in ~/.local/bin.
func OldShims(home string) []string {
	var found []string
	for _, n := range []string{"a", "b", "c"} {
		p := filepath.Join(home, ".local", "bin", "claude-"+n+".cmd")
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	return found
}
```

`internal/migrate/shell_unix.go`：
```go
//go:build !windows

package migrate

import "path/filepath"

// ShellFiles are the startup files the old installer may have written to.
func ShellFiles(home string) []string {
	return []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".zshrc")}
}
```

`internal/migrate/shell_windows.go`：
```go
//go:build windows

package migrate

import (
	"os/exec"
	"strings"
)

// ShellFiles are the $PROFILE of Windows PowerShell and PowerShell 7, whichever exist.
func ShellFiles(home string) []string {
	var paths []string
	for _, shell := range []string{"powershell", "pwsh"} {
		if _, err := exec.LookPath(shell); err != nil {
			continue
		}
		out, err := exec.Command(shell, "-NoProfile", "-Command", "$PROFILE").Output()
		if p := strings.TrimSpace(string(out)); err == nil && p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
```

- [ ] **Step 4: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; gofmt -w internal/migrate && go test ./internal/migrate/ && GOOS=windows go vet ./internal/migrate/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/migrate
git commit -F - <<'EOF'
feat: 偵測舊版 A/B/C 帳號並移除舊安裝區塊

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 8: 顏色與小寵物

**Files:**
- Create: `internal/style/style.go`, `internal/pet/lines.go`, `internal/pet/pet.go`, `internal/pet/pet_test.go`

**Interfaces:**
- Consumes: `fsutil.ReadJSON`、`fsutil.WriteJSON`
- Produces:
  - `style.RGB(hex string) string`；`style.Low, Mid, High, Accent, Text, Muted, Track, Sep`（string 變數）；常數 `style.Bold, Italic, Reset`；`style.Strip(s string) string`
  - `pet.Save`、`pet.Past`、`pet.Lines`、`pet.HoursFor(level int) float64`、`pet.LevelFor(hours float64) int`、`pet.Body(s *Save) string`、`pet.Mood(pct *float64) string`、`pet.Render(s *Save, ctxPct *float64, now float64) string`、`pet.Update(path string, ctxPct *float64, now time.Time, rnd *rand.Rand) string`

- [ ] **Step 1: 實作顏色** `internal/style/style.go`（沒有邏輯，不另寫測試）

```go
// Package style holds the status line's ANSI colours.
package style

import (
	"fmt"
	"regexp"
)

func RGB(hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// Colour-blind friendly scale (theme is dark-daltonized): blue -> amber -> orange.
var (
	Low    = RGB("#5fafff")
	Mid    = RGB("#e5c07b")
	High   = RGB("#ff8c42")
	Accent = RGB("#d97757") // Claude orange, marks the running account
	Text   = RGB("#c8ccd4")
	Muted  = RGB("#7f8794")
	Track  = RGB("#3e4451")
	Sep    = " " + Track + "│" + Reset + " "
)

const (
	Bold   = "\x1b[1m"
	Italic = "\x1b[3m"
	Reset  = "\x1b[0m"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Strip removes colour codes, for tests and plain-text output.
func Strip(s string) string { return ansi.ReplaceAllString(s, "") }
```

- [ ] **Step 2: 寫失敗的測試** `internal/pet/pet_test.go`

```go
package pet

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

var t0 = time.Unix(1_800_000_000, 0)

func secs(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

func petFile(t *testing.T, s any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "statusline-pet.json")
	if raw, ok := s.(string); ok {
		os.WriteFile(path, []byte(raw), 0o644)
	} else if err := fsutil.WriteJSON(path, s, false); err != nil {
		t.Fatal(err)
	}
	return path
}

func load(t *testing.T, path string) Save {
	t.Helper()
	var s Save
	if err := fsutil.ReadJSON(path, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func hours(h float64) *float64 { return &h }

func TestLevelBoundaries(t *testing.T) {
	for l := 1; l <= MaxLevel; l++ {
		if got := LevelFor(HoursFor(l)); got != l {
			t.Errorf("level %d: got %d", l, got)
		}
	}
	if LevelFor(HoursFor(8)-0.001) != 7 || LevelFor(1000) != MaxLevel {
		t.Error("edges")
	}
}

func TestMigratesSessionSave(t *testing.T) {
	path := petFile(t, `{"sessions":39,"seen":["x"],"level":7}`)
	out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1))))
	s := load(t, path)
	if s.Level != 7 || s.Life != 1 || s.Line != "bird" || math.Abs(*s.Hours-HoursFor(7)) > 1e-9 {
		t.Fatalf("got %+v", s)
	}
	if s.Branch != "fowl" && s.Branch != "raptor" {
		t.Fatalf("branch %q", s.Branch)
	}
	if out != "🐔 Lv7" {
		t.Fatalf("got %q", out)
	}
}

func TestOldDragonSave(t *testing.T) {
	path := petFile(t, `{"life":2,"line":"dragon","hours":50,"level":14,"past":[{"line":"bird","ended":"x"}]}`)
	out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1))))
	if s := load(t, path); s.Line != "reptile" {
		t.Fatalf("got %+v", s)
	}
	if out != "⭐🦕 Lv14" && out != "⭐🐲 Lv14" {
		t.Fatalf("got %q", out)
	}
}

func TestActiveTimeOnly(t *testing.T) {
	ts := secs(t0)
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "fowl", Hours: hours(5), Level: 7, Tick: ts - 60})
	Update(path, nil, t0, rand.New(rand.NewSource(1)))
	if h := *load(t, path).Hours; math.Abs(h-(5+60.0/3600)) > 1e-9 {
		t.Fatalf("60s gap: %v", h)
	}
	Update(path, nil, t0.Add(time.Hour), rand.New(rand.NewSource(1)))
	if h := *load(t, path).Hours; math.Abs(h-(5+60.0/3600)) > 1e-9 {
		t.Fatalf("an idle hour must not count: %v", h)
	}
}

func TestLevelUpParty(t *testing.T) {
	ts := secs(t0)
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "fowl", Hours: hours(HoursFor(8) - 0.001), Level: 7, Tick: ts - 60})
	if out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1)))); out != "🐔🎉 Lv8" {
		t.Fatalf("got %q", out)
	}
	if out := style.Strip(Update(path, nil, t0.Add(601*time.Second), rand.New(rand.NewSource(1)))); out != "🐔 Lv8" {
		t.Fatalf("after 10 minutes: %q", out)
	}
}

func TestRebirthCollectsEveryEnding(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	now := t0
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "raptor", Hours: hours(4.8), Level: 7})
	seen := map[ending]bool{{"bird", "raptor"}: true}
	rebirth := func() Save {
		s := load(t, path)
		s.Hours, s.Tick = hours(LifeHours-1e-4), secs(now)-60
		fsutil.WriteJSON(path, s, false)
		Update(path, nil, now, rnd)
		now = now.Add(time.Hour)
		return load(t, path)
	}
	for i := 0; i < 15; i++ {
		s := rebirth()
		seen[ending{s.Line, s.Branch}] = true
	}
	if len(seen) != 16 {
		t.Fatalf("16 lives raised %d different final forms", len(seen))
	}
	for i := 0; i < 30; i++ {
		prev := load(t, path).Line
		if s := rebirth(); s.Line == prev {
			t.Fatalf("same line twice in a row: %s", prev)
		}
	}
	s := load(t, path)
	if out := style.Strip(Render(&s, nil, secs(now))); !strings.HasPrefix(out, "⭐45") {
		t.Fatalf("stars: %q", out)
	}
}

func TestBodyFlexibleLevels(t *testing.T) {
	var b strings.Builder
	for l := 1; l <= MaxLevel; l++ {
		b.WriteString(Body(&Save{Line: "mammal", Branch: "canine", Level: l}))
	}
	if want := "🍼🍼🐾🐾🐶🐶🐶🐶🐶🐕🐕🐕🐕🐕🐕🐺🐺🐺🐺👑🐺"; b.String() != want {
		t.Fatalf("got %s", b.String())
	}
}

func TestMood(t *testing.T) {
	cases := map[float64]string{0: "✨", 29.9: "✨", 30: "", 59: "", 60: "💦", 84.9: "💦", 85: "💤", 100: "💤"}
	for pct, want := range cases {
		p := pct
		if got := Mood(&p); got != want {
			t.Errorf("%v: %q", pct, got)
		}
	}
	if Mood(nil) != "" {
		t.Error("nil")
	}
}

func TestEmojiWithinUnicode12(t *testing.T) {
	newer := map[rune]bool{0x1F90C: true, 0x1F972: true, 0x1F977: true, 0x1F978: true, 0x1F9A3: true,
		0x1F9A4: true, 0x1F9AB: true, 0x1F9AC: true, 0x1F9AD: true, 0x1F9CB: true}
	check := func(stages []Stage) {
		for _, st := range stages {
			for _, r := range st.Emoji {
				if r == 0x200D || r >= 0x1FA70 || newer[r] {
					t.Errorf("%s uses U+%X", st.Emoji, r)
				}
			}
		}
	}
	endings := 0
	for _, l := range Lines {
		check(l.Stages)
		for _, b := range l.Branches {
			check(b.Stages)
		}
		endings += len(l.endings())
	}
	if endings != 16 {
		t.Fatalf("%d final forms", endings)
	}
}
```

- [ ] **Step 3: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/pet/`
Expected: FAIL，`undefined: Save`。

- [ ] **Step 4: 實作** `internal/pet/lines.go`

```go
package pet

// Stage is the emoji a pet shows from Level on.
type Stage struct {
	Level int
	Emoji string
}

// Branch is one way a line can grow after it splits.
type Branch struct {
	Key    string
	Stages []Stage
}

// Line is a family of pets: shared early stages, then optionally a random branch.
type Line struct {
	Key      string
	Stages   []Stage
	Branches []Branch
}

// Lines hatch from 🥚 (egg layers), 🌰 (plants) or 🍼 (mammals). Emoji stay within
// Unicode 12: newer ones such as 🦭 render as boxes in many terminals.
var Lines = []Line{
	{"bird", []Stage{{1, "🥚"}, {3, "🐣"}, {5, "🐥"}, {7, "🐔"}}, []Branch{
		{"fowl", []Stage{{10, "🐓"}, {13, "🦃"}, {16, "🦚"}, {20, "👑🦚"}}},
		{"raptor", []Stage{{10, "🐦"}, {13, "🦉"}, {16, "🦅"}, {20, "👑🦅"}}},
	}},
	{"water", []Stage{{1, "🥚"}, {3, "🐣"}, {5, "🐥"}, {7, "🦆"}, {10, "🐧"}, {13, "🦩"}, {16, "🦢"}, {20, "👑🦢"}}, nil},
	{"reptile", []Stage{{1, "🥚"}, {3, "🦎"}, {5, "🐢"}, {7, "🐍"}, {10, "🐊"}}, []Branch{
		{"dino", []Stage{{13, "🦕"}, {16, "🦖"}, {20, "👑🦖"}}},
		{"dragon", []Stage{{13, "🐲"}, {16, "🐉"}, {20, "👑🐉"}}},
	}},
	{"sea", []Stage{{1, "🥚"}, {3, "🦐"}, {5, "🐟"}, {7, "🐠"}, {10, "🐡"}}, []Branch{
		{"cephalopod", []Stage{{13, "🦑"}, {16, "🐙"}, {20, "👑🐙"}}},
		{"crustacean", []Stage{{13, "🦀"}, {16, "🦞"}, {20, "👑🦞"}}},
	}},
	{"insect", []Stage{{1, "🥚"}, {3, "🐛"}, {5, "🐜"}, {7, "🐞"}, {10, "🦗"}, {13, "🐝"}, {16, "🦋"}, {20, "👑🦋"}}, nil},
	{"plant", []Stage{{1, "🌰"}, {3, "🌱"}, {5, "🌿"}, {7, "🍀"}}, []Branch{
		{"flower", []Stage{{10, "🌷"}, {13, "🌹"}, {16, "🌻"}, {20, "👑🌻"}}},
		{"sakura", []Stage{{10, "🌳"}, {13, "🌸"}, {16, "🍒"}, {20, "👑🌸"}}},
		{"apple", []Stage{{10, "🌳"}, {13, "🍏"}, {16, "🍎"}, {20, "👑🍎"}}},
		{"grape", []Stage{{10, "🍃"}, {13, "🍇"}, {16, "🍷"}, {20, "👑🍷"}}},
	}},
	{"mammal", []Stage{{1, "🍼"}, {3, "🐾"}}, []Branch{
		{"feline", []Stage{{5, "🐱"}, {7, "🐈"}, {10, "🐆"}, {13, "🐅"}, {16, "🦁"}, {20, "👑🦁"}}},
		{"canine", []Stage{{5, "🐶"}, {10, "🐕"}, {16, "🐺"}, {20, "👑🐺"}}},
		{"marine", []Stage{{5, "🦦"}, {10, "🐬"}, {13, "🐳"}, {16, "🐋"}, {20, "👑🐋"}}},
		{"primate", []Stage{{5, "🐵"}, {7, "🙈"}, {10, "🐒"}, {13, "🦧"}, {16, "🦍"}, {20, "👑🦍"}}},
	}},
}
```

- [ ] **Step 5: 實作** `internal/pet/pet.go`

```go
// Package pet is the status line pet. It grows with active Claude time
// (Lv = 1 + 19 * cbrt(hours / LifeHours)); at Lv20 it is reborn with a ⭐ as a random
// line, preferring final forms not raised yet.
package pet

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

const (
	LifeHours   = 150.0
	MaxLevel    = 20
	ActiveGap   = 300.0 // seconds; refreshes further apart count as idle
	LevelUpSecs = 600.0 // how long 🎉 stays after a level-up or rebirth
)

type Past struct {
	Line   string `json:"line"`
	Branch string `json:"branch,omitempty"`
	Ended  string `json:"ended"`
}

type Save struct {
	Life      int      `json:"life"`
	Line      string   `json:"line"`
	Branch    string   `json:"branch,omitempty"`
	Hours     *float64 `json:"hours,omitempty"`
	Level     int      `json:"level"`
	LeveledAt float64  `json:"leveled_at,omitempty"`
	Tick      float64  `json:"tick,omitempty"`
	Past      []Past   `json:"past,omitempty"`
	Sessions  int      `json:"sessions,omitempty"` // old saves counted sessions instead of hours
}

func HoursFor(level int) float64 {
	return LifeHours * math.Pow(float64(level-1)/float64(MaxLevel-1), 3)
}

func LevelFor(hours float64) int {
	frac := math.Max(0, math.Min(hours/LifeHours, 1))
	return 1 + int(float64(MaxLevel-1)*math.Cbrt(frac)+1e-9)
}

type ending struct{ line, branch string }

func findLine(key string) *Line {
	for i := range Lines {
		if Lines[i].Key == key {
			return &Lines[i]
		}
	}
	return nil
}

func (l *Line) branch(key string) *Branch {
	for i := range l.Branches {
		if l.Branches[i].Key == key {
			return &l.Branches[i]
		}
	}
	return nil
}

// endings are the line's final forms: one per branch, or one if it never splits.
func (l *Line) endings() []ending {
	if len(l.Branches) == 0 {
		return []ending{{l.Key, ""}}
	}
	out := make([]ending, len(l.Branches))
	for i, b := range l.Branches {
		out[i] = ending{l.Key, b.Key}
	}
	return out
}

func raised(s *Save) map[ending]bool {
	done := map[ending]bool{}
	for _, p := range s.Past {
		done[ending{p.Line, p.Branch}] = true
	}
	return done
}

func pickBranch(l *Line, done map[ending]bool, rnd *rand.Rand) string {
	all := l.endings()
	var fresh []ending
	for _, e := range all {
		if !done[e] {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) == 0 {
		fresh = all
	}
	return fresh[rnd.Intn(len(fresh))].branch
}

// nextLine picks a line that still has final forms not raised; once every one has
// been, any line but the current one.
func nextLine(s *Save, rnd *rand.Rand) (string, string) {
	done := raised(s)
	done[ending{s.Line, s.Branch}] = true
	var fresh, others []*Line
	for i := range Lines {
		l := &Lines[i]
		for _, e := range l.endings() {
			if !done[e] {
				fresh = append(fresh, l)
				break
			}
		}
		if l.Key != s.Line {
			others = append(others, l)
		}
	}
	pool := fresh
	if len(pool) == 0 {
		pool = others
	}
	l := pool[rnd.Intn(len(pool))]
	return l.Key, pickBranch(l, done, rnd)
}

func normalize(s *Save, rnd *rand.Rand) {
	if s.Hours == nil {
		// Older saves counted sessions (Lv = 1 + sqrt(sessions)); keep that level.
		level := s.Level
		if level == 0 {
			level = 1 + int(math.Sqrt(float64(s.Sessions)))
		}
		h := HoursFor(level)
		*s = Save{Life: 1, Line: "bird", Hours: &h, Level: level}
	}
	if s.Life == 0 {
		s.Life = 1
	}
	if findLine(s.Line) == nil {
		if s.Line == "dragon" { // merged into reptile
			s.Line = "reptile"
		} else {
			s.Line = "bird"
		}
		s.Branch = ""
	}
	l := findLine(s.Line)
	if len(l.Branches) == 0 {
		s.Branch = ""
	} else if l.branch(s.Branch) == nil {
		s.Branch = pickBranch(l, raised(s), rnd)
	}
}

// feed adds the time since the last refresh if it was short enough to count as active.
func feed(s *Save, now float64, rnd *rand.Rand) {
	gap := now - s.Tick
	s.Tick = now
	if gap <= 0 || gap >= ActiveGap {
		return
	}
	h := *s.Hours + gap/3600
	for h >= LifeHours {
		h -= LifeHours
		s.Past = append(s.Past, Past{Line: s.Line, Branch: s.Branch,
			Ended: time.Unix(int64(now), 0).Format("2006-01-02")})
		s.Line, s.Branch = nextLine(s, rnd)
		s.Life++
		s.LeveledAt = now
	}
	s.Hours = &h
	if level := LevelFor(h); level != s.Level {
		s.Level, s.LeveledAt = level, now
	}
}

func Body(s *Save) string {
	l := findLine(s.Line)
	stages := l.Stages
	if b := l.branch(s.Branch); b != nil {
		stages = append(append([]Stage{}, stages...), b.Stages...)
	}
	body := stages[0].Emoji
	for _, st := range stages {
		if s.Level >= st.Level {
			body = st.Emoji
		}
	}
	return body
}

// Mood follows context usage: fresh -> normal -> tired -> sleepy (time to /compact).
func Mood(pct *float64) string {
	if pct == nil {
		return ""
	}
	switch p := *pct; {
	case p < 30:
		return "✨"
	case p < 60:
		return ""
	case p < 85:
		return "💦"
	}
	return "💤"
}

func Render(s *Save, ctxPct *float64, now float64) string {
	stars := strings.Repeat("⭐", s.Life-1)
	if s.Life-1 > 3 {
		stars = fmt.Sprintf("⭐%d", s.Life-1)
	}
	head := stars + Body(s) + Mood(ctxPct)
	if now-s.LeveledAt < LevelUpSecs {
		return fmt.Sprintf("%s🎉 %s%sLv%d%s", head, style.Accent, style.Bold, s.Level, style.Reset)
	}
	return fmt.Sprintf("%s %sLv%d%s", head, style.Muted, s.Level, style.Reset)
}

// Update loads the pet at path, feeds it and saves it, and returns its status text.
func Update(path string, ctxPct *float64, now time.Time, rnd *rand.Rand) string {
	var s Save
	if fsutil.ReadJSON(path, &s) != nil {
		s = Save{}
	}
	t := float64(now.UnixNano()) / 1e9
	normalize(&s, rnd)
	feed(&s, t, rnd)
	fsutil.WriteJSON(path, &s, false)
	return Render(&s, ctxPct, t)
}
```

- [ ] **Step 6: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/pet/ ./internal/style/`
Expected: `ok`

- [ ] **Step 7: Commit**

```bash
git add internal/style internal/pet
git commit -F - <<'EOF'
feat: 移植小寵物（活躍時數、轉生、7 系 16 種最終型態）

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 9: 狀態列輸入與用量查詢

**Files:**
- Create: `internal/statusline/input.go`, `internal/statusline/usage.go`, `internal/statusline/usage_test.go`

**Interfaces:**
- Consumes: `fsutil.*`
- Produces:
  - `type Input`（欄位 `SessionID`、`Model.DisplayName`、`ContextWindow`、`Cost.TotalCostUSD *float64`、`RateLimits *RateLimits`）、`type RateLimits{ FiveHour, SevenDay *RateLimit }`、`type RateLimit{ UsedPercentage *float64; ResetsAt float64 }`、`ParseInput(r io.Reader) Input`、`(Input).ContextPct() *float64`
  - `type Window{ Utilization *float64; ResetsAt string }`、`type Entry{ Plan string; FiveHour, SevenDay *Window; FetchedAt float64; LastError string }`、`type Cache map[string]*Entry`
  - `LoadCache(mainDir string) Cache`、`(Cache).Save(mainDir string) error`、`Forget(mainDir, name string)`
  - `CacheTTL`、`ErrNotLoggedIn`、`ErrTokenExpired`、`type HTTPError{ Code int }`
  - `type Fetcher{ UsageURL, TokenURL string; Client *http.Client }`、`NewFetcher() *Fetcher`、`(*Fetcher).Fetch(dir string, active bool, now time.Time) (*Entry, error)`
  - `Plan(dir string) string`、`FromRateLimits(rl *RateLimits, plan string, now time.Time) *Entry`、`unixSecs(t time.Time) float64`、`fromUnix(f float64) time.Time`

- [ ] **Step 1: 寫失敗的測試** `internal/statusline/usage_test.go`

```go
package statusline

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

var t0 = time.Unix(1_800_000_000, 0)

func writeCreds(t *testing.T, dir, token string, expiresAtMs float64, plan string) {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	creds := map[string]any{"claudeAiOauth": map[string]any{
		"accessToken": token, "refreshToken": "refresh-" + token, "expiresAt": expiresAtMs,
		"scopes": []string{"user:inference", "user:profile"}, "subscriptionType": plan,
	}, "other": "kept"}
	if err := fsutil.WriteJSON(filepath.Join(dir, ".credentials.json"), creds, false); err != nil {
		t.Fatal(err)
	}
}

const usageBody = `{"five_hour":{"utilization":31.0,"resets_at":"2027-01-15T10:05:00.123+00:00"},"seven_day":{"utilization":8,"resets_at":"2027-01-18T12:00:00Z"},"extra":1}`

func fetcher(srv *httptest.Server) *Fetcher {
	return &Fetcher{UsageURL: srv.URL + "/usage", TokenURL: srv.URL + "/token", Client: srv.Client()}
}

func TestFetchSuccess(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "tok", 9e12, "max")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("anthropic-beta") != "oauth-2025-04-20" {
			t.Errorf("headers: %v", r.Header)
		}
		w.Write([]byte(usageBody))
	}))
	defer srv.Close()
	e, err := fetcher(srv).Fetch(dir, false, t0)
	if err != nil {
		t.Fatal(err)
	}
	if e.Plan != "max" || *e.FiveHour.Utilization != 31 || e.SevenDay.ResetsAt != "2027-01-18T12:00:00Z" || e.FetchedAt != 1_800_000_000 {
		t.Fatalf("got %+v", e)
	}
}

func TestFetchErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	dir := t.TempDir()
	if _, err := fetcher(srv).Fetch(dir, false, t0); !errors.Is(err, ErrNotLoggedIn) {
		t.Errorf("no creds: %v", err)
	}
	writeCreds(t, dir, "tok", 9e12, "pro")
	var he *HTTPError
	if _, err := fetcher(srv).Fetch(dir, false, t0); !errors.As(err, &he) || he.Code != 401 || err.Error() != "HTTP 401" {
		t.Errorf("401: %v", err)
	}
}

func TestFetchExpiredActiveTokenIsLeftToClaude(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "tok", float64(t0.Unix())*1000, "pro")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL)
	}))
	defer srv.Close()
	if _, err := fetcher(srv).Fetch(dir, true, t0); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("got %v", err)
	}
}

func TestFetchRefreshesInactiveToken(t *testing.T) {
	dir := t.TempDir()
	writeCreds(t, dir, "old", float64(t0.Unix()+30)*1000, "pro")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["grant_type"] != "refresh_token" || body["refresh_token"] != "refresh-old" ||
				body["scope"] != "user:inference user:profile" || body["client_id"] == "" {
				t.Errorf("token request: %v", body)
			}
			w.Write([]byte(`{"access_token":"new","refresh_token":"refresh-new","expires_in":3600}`))
		case "/usage":
			if r.Header.Get("Authorization") != "Bearer new" {
				t.Errorf("usage called with %q", r.Header.Get("Authorization"))
			}
			w.Write([]byte(usageBody))
		}
	}))
	defer srv.Close()
	if _, err := fetcher(srv).Fetch(dir, false, t0); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".credentials.json")
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, `"accessToken":"new"`) || !strings.Contains(text, `"refreshToken":"refresh-new"`) ||
		!strings.Contains(text, `"other":"kept"`) || !strings.Contains(text, `"expiresAt":1800003600000`) {
		t.Fatalf("credentials: %s", text)
	}
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("perm %v", fi.Mode().Perm())
	}
}

func TestFromRateLimits(t *testing.T) {
	var in Input
	json.Unmarshal([]byte(`{"rate_limits":{"five_hour":{"used_percentage":9,"resets_at":1800007470},"seven_day":{"used_percentage":43,"resets_at":1800273570}}}`), &in)
	e := FromRateLimits(in.RateLimits, "pro", t0)
	if e == nil || *e.FiveHour.Utilization != 9 || e.FiveHour.ResetsAt != "2027-01-15T10:04:30Z" || *e.SevenDay.Utilization != 43 || e.Plan != "pro" {
		t.Fatalf("got %+v", e)
	}
	if FromRateLimits(nil, "pro", t0) != nil {
		t.Fatal("no rate limits means no entry")
	}
}

func TestParseInputAndContextPct(t *testing.T) {
	in := ParseInput(strings.NewReader(`{"session_id":"s","context_window":{"context_window_size":200000,"current_usage":{"input_tokens":1000,"cache_read_input_tokens":19000,"cache_creation_input_tokens":0,"output_tokens":5}}}`))
	if in.SessionID != "s" || in.ContextPct() == nil || *in.ContextPct() != 10 {
		t.Fatalf("got %+v", in)
	}
	if bad := ParseInput(strings.NewReader("not json")); bad.SessionID != "" || bad.ContextPct() != nil {
		t.Fatal("garbage input should parse as empty")
	}
}

func TestCacheLegacyKeysAndForget(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "usage-cache.json"), []byte(`{"A":{"plan":"pro","fetched_at":1},"B":{"plan":"max","fetched_at":2}}`), 0o644)
	c := LoadCache(dir)
	if c["a"] == nil || c["b"].Plan != "max" || c["A"] != nil {
		t.Fatalf("got %v", c)
	}
	c.Save(dir)
	Forget(dir, "b")
	if c := LoadCache(dir); c["b"] != nil || c["a"] == nil {
		t.Fatalf("after forget: %v", c)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/statusline/`
Expected: FAIL，`undefined: Fetcher`。

- [ ] **Step 3: 實作** `internal/statusline/input.go`

```go
// Package statusline renders the Claude Code status line: every account's usage,
// the model and the pet.
package statusline

import (
	"encoding/json"
	"io"
)

// Input is the JSON Claude Code passes on stdin. A field with an unexpected type is
// skipped and the rest still decoded.
type Input struct {
	SessionID string `json:"session_id"`
	Model     struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage    *float64 `json:"used_percentage"`
		ContextWindowSize float64  `json:"context_window_size"`
		CurrentUsage      struct {
			InputTokens              float64 `json:"input_tokens"`
			CacheCreationInputTokens float64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     float64 `json:"cache_read_input_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Cost struct {
		TotalCostUSD *float64 `json:"total_cost_usd"`
	} `json:"cost"`
	RateLimits *RateLimits `json:"rate_limits"`
}

type RateLimits struct {
	FiveHour *RateLimit `json:"five_hour"`
	SevenDay *RateLimit `json:"seven_day"`
}

type RateLimit struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       float64  `json:"resets_at"` // Unix seconds
}

func ParseInput(r io.Reader) Input {
	var in Input
	data, _ := io.ReadAll(r)
	json.Unmarshal(data, &in)
	return in
}

// ContextPct is how full the context window is, or nil if unknown.
func (in Input) ContextPct() *float64 {
	cw := in.ContextWindow
	if cw.UsedPercentage != nil {
		return cw.UsedPercentage
	}
	if cw.ContextWindowSize == 0 {
		return nil
	}
	u := cw.CurrentUsage
	pct := (u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens) / cw.ContextWindowSize * 100
	return &pct
}
```

- [ ] **Step 4: 實作** `internal/statusline/usage.go`

```go
package statusline

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

const (
	DefaultUsageURL = "https://api.anthropic.com/api/oauth/usage"
	DefaultTokenURL = "https://platform.claude.com/v1/oauth/token"
	clientID        = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	CacheTTL        = 120 * time.Second // between fetches per account
	requestTimeout  = 4 * time.Second
)

var (
	ErrNotLoggedIn  = errors.New("not logged in")
	ErrTokenExpired = errors.New("token expired")
)

type HTTPError struct{ Code int }

func (e *HTTPError) Error() string { return fmt.Sprintf("HTTP %d", e.Code) }

// Window is one usage window in the api/oauth/usage format, which the cache keeps.
type Window struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at,omitempty"`
}

type Entry struct {
	Plan      string  `json:"plan,omitempty"`
	FiveHour  *Window `json:"five_hour"`
	SevenDay  *Window `json:"seven_day"`
	FetchedAt float64 `json:"fetched_at"`
	LastError string  `json:"last_error,omitempty"`
}

// Cache is usage-cache.json, keyed by account letter.
type Cache map[string]*Entry

func cachePath(mainDir string) string { return filepath.Join(mainDir, "usage-cache.json") }

func LoadCache(mainDir string) Cache {
	c := Cache{}
	fsutil.ReadJSON(cachePath(mainDir), &c)
	for k, v := range c {
		// The Python version keyed accounts as "A", "B".
		if low := strings.ToLower(k); low != k {
			if c[low] == nil {
				c[low] = v
			}
			delete(c, k)
		}
	}
	return c
}

func (c Cache) Save(mainDir string) error { return fsutil.WriteJSON(cachePath(mainDir), c, false) }

// Forget drops a removed account's cached usage.
func Forget(mainDir, name string) {
	c := LoadCache(mainDir)
	if _, ok := c[name]; ok {
		delete(c, name)
		c.Save(mainDir)
	}
}

func unixSecs(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

func fromUnix(f float64) time.Time {
	sec, frac := math.Modf(f)
	return time.Unix(int64(sec), int64(frac*1e9))
}

type Fetcher struct {
	UsageURL, TokenURL string
	Client             *http.Client
}

func NewFetcher() *Fetcher {
	return &Fetcher{UsageURL: DefaultUsageURL, TokenURL: DefaultTokenURL,
		Client: &http.Client{Timeout: requestTimeout}}
}

func readCreds(dir string) (string, map[string]any, map[string]any, error) {
	path := filepath.Join(dir, ".credentials.json")
	var creds map[string]any
	if err := fsutil.ReadJSON(path, &creds); err != nil {
		return path, nil, nil, ErrNotLoggedIn
	}
	oauth, ok := creds["claudeAiOauth"].(map[string]any)
	if !ok {
		return path, nil, nil, ErrNotLoggedIn
	}
	return path, creds, oauth, nil
}

// Plan is the subscription type stored with the account's login, e.g. "pro".
func Plan(dir string) string {
	_, _, oauth, err := readCreds(dir)
	if err != nil {
		return ""
	}
	plan, _ := oauth["subscriptionType"].(string)
	return plan
}

// Fetch asks api/oauth/usage with the account's own token. An expiring token is
// refreshed only for an account that isn't running; the running Claude refreshes its own.
func (f *Fetcher) Fetch(dir string, active bool, now time.Time) (*Entry, error) {
	path, creds, oauth, err := readCreds(dir)
	if err != nil {
		return nil, err
	}
	token, _ := oauth["accessToken"].(string)
	expiresAt, _ := oauth["expiresAt"].(float64)
	if expiresAt/1000 < float64(now.Unix()+60) {
		if active {
			return nil, ErrTokenExpired
		}
		if token, err = f.refresh(path, creds, oauth, now); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(http.MethodGet, f.UsageURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-accounts-statusline")
	resp, err := f.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{resp.StatusCode}
	}
	var body struct {
		FiveHour *Window `json:"five_hour"`
		SevenDay *Window `json:"seven_day"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	plan, _ := oauth["subscriptionType"].(string)
	return &Entry{Plan: plan, FiveHour: body.FiveHour, SevenDay: body.SevenDay, FetchedAt: unixSecs(now)}, nil
}

func (f *Fetcher) refresh(path string, creds, oauth map[string]any, now time.Time) (string, error) {
	var scopes []string
	if list, ok := oauth["scopes"].([]any); ok {
		for _, s := range list {
			if str, ok := s.(string); ok {
				scopes = append(scopes, str)
			}
		}
	}
	refreshToken, _ := oauth["refreshToken"].(string)
	body, _ := json.Marshal(map[string]string{"grant_type": "refresh_token",
		"refresh_token": refreshToken, "client_id": clientID, "scope": strings.Join(scopes, " ")})
	resp, err := f.Client.Post(f.TokenURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{resp.StatusCode}
	}
	var data struct {
		AccessToken  string  `json:"access_token"`
		RefreshToken string  `json:"refresh_token"`
		ExpiresIn    float64 `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if data.AccessToken == "" {
		return "", errors.New("token refresh returned no access token")
	}
	if data.ExpiresIn == 0 {
		data.ExpiresIn = 3600
	}
	oauth["accessToken"] = data.AccessToken
	if data.RefreshToken != "" {
		oauth["refreshToken"] = data.RefreshToken
	}
	oauth["expiresAt"] = float64(now.Unix()+int64(data.ExpiresIn)) * 1000
	out, err := fsutil.Marshal(creds, false)
	if err != nil {
		return "", err
	}
	return data.AccessToken, fsutil.WriteFileAtomic(path, out, 0o600)
}

// FromRateLimits turns the running account's rate_limits from the status line input
// into a cache entry, so no request is needed for it.
func FromRateLimits(rl *RateLimits, plan string, now time.Time) *Entry {
	if rl == nil || (rl.FiveHour == nil && rl.SevenDay == nil) {
		return nil
	}
	conv := func(r *RateLimit) *Window {
		if r == nil || r.UsedPercentage == nil {
			return nil
		}
		w := &Window{Utilization: r.UsedPercentage}
		if r.ResetsAt > 0 {
			w.ResetsAt = time.Unix(int64(r.ResetsAt), 0).UTC().Format(time.RFC3339)
		}
		return w
	}
	return &Entry{Plan: plan, FiveHour: conv(rl.FiveHour), SevenDay: conv(rl.SevenDay), FetchedAt: unixSecs(now)}
}
```

- [ ] **Step 5: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/statusline/`
Expected: `ok`

- [ ] **Step 6: Commit**

```bash
git add internal/statusline
git commit -F - <<'EOF'
feat: 狀態列輸入解析、用量快取、OAuth 查詢與 rate_limits

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 10: API 花費紀錄

**Files:**
- Create: `internal/statusline/cost.go`, `internal/statusline/cost_test.go`

**Interfaces:**
- Consumes: `fsutil.*`
- Produces: `type LedgerEntry{ Day string; Base, Cost float64; Account string }`、`type Ledger map[string]*LedgerEntry`、`LoadLedger(mainDir string) Ledger`、`(Ledger).Save(mainDir string) error`、`(Ledger).Record(account, sid string, cost float64, now time.Time)`、`(Ledger).Spent(account string, keep func(day string) bool) float64`、`(Ledger).Has(account string) bool`

- [ ] **Step 1: 寫失敗的測試** `internal/statusline/cost_test.go`

```go
package statusline

import (
	"math"
	"testing"
	"time"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRecordCarriesResumedSessions(t *testing.T) {
	l := Ledger{}
	l.Record("c", "s1", 0.30, t0)
	l.Record("c", "s1", 0.50, t0)
	l.Record("c", "s1", 0.10, t0) // resumed: Claude restarts the total at 0
	all := func(string) bool { return true }
	if got := l.Spent("c", all); !near(got, 0.60) {
		t.Fatalf("got %v", got)
	}
}

func TestSpentPerAccountAndLegacy(t *testing.T) {
	today := t0.Format("2006-01-02")
	l := Ledger{
		"old": {Day: today, Cost: 1.00},                // written before accounts were tracked: c
		"s2":  {Day: today, Cost: 2.00, Account: "c"},
		"s3":  {Day: today, Cost: 4.00, Account: "d"},
	}
	all := func(string) bool { return true }
	if !near(l.Spent("c", all), 3.00) || !near(l.Spent("d", all), 4.00) {
		t.Fatalf("c=%v d=%v", l.Spent("c", all), l.Spent("d", all))
	}
	if !l.Has("d") || l.Has("e") {
		t.Fatal("Has")
	}
}

func TestRecordPrunesOldDays(t *testing.T) {
	l := Ledger{"ancient": {Day: t0.AddDate(0, 0, -70).Format("2006-01-02"), Cost: 9, Account: "c"}}
	l.Record("c", "s1", 1, t0)
	if l["ancient"] != nil {
		t.Fatal("entries older than 62 days should be dropped")
	}
}

func TestLedgerSaveLoad(t *testing.T) {
	dir := t.TempDir()
	l := Ledger{}
	l.Record("c", "s1", 0.25, t0.Add(time.Minute))
	if err := l.Save(dir); err != nil {
		t.Fatal(err)
	}
	if got := LoadLedger(dir); got["s1"] == nil || got["s1"].Account != "c" || !near(got["s1"].Cost, 0.25) {
		t.Fatalf("got %v", got)
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/statusline/ -run 'Record|Spent|Ledger'`
Expected: FAIL，`undefined: Ledger`。

- [ ] **Step 3: 實作** `internal/statusline/cost.go`

```go
package statusline

import (
	"path/filepath"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

const ledgerDays = 62 // enough history for "this month"

// LedgerEntry is one session's spend. Resuming a session restarts Claude's total at
// 0, so earlier runs of the same session are carried in Base.
type LedgerEntry struct {
	Day     string  `json:"day"`
	Base    float64 `json:"base"`
	Cost    float64 `json:"cost"`
	Account string  `json:"account,omitempty"` // empty in files from before it existed: account c
}

// Ledger is api-cost.json, keyed by session id.
type Ledger map[string]*LedgerEntry

func ledgerPath(mainDir string) string { return filepath.Join(mainDir, "api-cost.json") }

func LoadLedger(mainDir string) Ledger {
	l := Ledger{}
	fsutil.ReadJSON(ledgerPath(mainDir), &l)
	return l
}

func (l Ledger) Save(mainDir string) error { return fsutil.WriteJSON(ledgerPath(mainDir), l, false) }

func (l Ledger) Record(account, sid string, cost float64, now time.Time) {
	e := l[sid]
	if e == nil {
		e = &LedgerEntry{Day: now.Format("2006-01-02"), Account: account}
		l[sid] = e
	}
	if cost < e.Cost {
		e.Base += e.Cost
	}
	e.Cost = cost
	cutoff := now.AddDate(0, 0, -ledgerDays).Format("2006-01-02")
	for k, v := range l {
		if v.Day < cutoff {
			delete(l, k)
		}
	}
}

func entryAccount(e *LedgerEntry) string {
	if e.Account == "" {
		return "c"
	}
	return e.Account
}

func (l Ledger) Spent(account string, keep func(day string) bool) float64 {
	total := 0.0
	for _, e := range l {
		if entryAccount(e) == account && keep(e.Day) {
			total += e.Base + e.Cost
		}
	}
	return total
}

func (l Ledger) Has(account string) bool {
	for _, e := range l {
		if entryAccount(e) == account {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; gofmt -l internal/ ; go test ./internal/statusline/`
Expected: gofmt 沒有列出檔案；`ok`

- [ ] **Step 5: Commit**

```bash
git add internal/statusline
git commit -F - <<'EOF'
feat: 分帳號的 API 花費紀錄

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 11: 狀態列排版與整體流程

**Files:**
- Create: `internal/statusline/render.go`, `internal/statusline/run.go`, `internal/statusline/run_test.go`

**Interfaces:**
- Consumes: Task 3 `config.*`、Task 2 `i18n.*`、Task 8 `style.*` 與 `pet.Update`、Task 9／10 的所有型別
- Produces: `type Options{ MainDir string; In io.Reader; Out io.Writer; Now time.Time; Fetcher *Fetcher; Rand *rand.Rand }`、`statusline.Run(o Options)`

- [ ] **Step 1: 寫失敗的測試** `internal/statusline/run_test.go`

預期字串依照 Python 版的格式推算：`fmt_remaining` 是「剩餘分鐘無條件捨去再 +1」，所以 7470 秒 → 125 分 → `2h05m`；273570 秒 → 4560 分 → `3d04h`；8910 秒 → `2h29m`；557970 秒 → `6d11h`。

```go
package statusline

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/pet"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

type fixture struct {
	main string
	opts Options
	out  *bytes.Buffer
}

func pct(v float64) *float64 { return &v }

// newFixture: accounts a (running, subscription), b (subscription) and c (API), all
// with fresh data, and a server that fails the test if anything is fetched.
func newFixture(t *testing.T, lang string) *fixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	main := filepath.Join(home, ".claude")
	t.Setenv("CLAUDE_CONFIG_DIR", main)
	for _, d := range []string{main, main + "-b", main + "-c"} {
		os.MkdirAll(d, 0o755)
	}
	cfg := config.New(main)
	cfg.Lang = lang
	cfg.Add(config.Account{Name: "b", Dir: "~/.claude-b", Type: config.TypeSubscription, StatusLine: true})
	cfg.Add(config.Account{Name: "c", Dir: "~/.claude-c", Type: config.TypeAPI, StatusLine: true})
	if err := cfg.Save(main); err != nil {
		t.Fatal(err)
	}
	writeCreds(t, main, "tok-a", 9e12, "pro")
	ts := unixSecs(t0)
	iso := func(sec int) string { return t0.Add(time.Duration(sec) * time.Second).UTC().Format(time.RFC3339) }
	Cache{
		"a": {Plan: "pro", FetchedAt: ts}, // fresh, so nothing is fetched when a isn't the running account
		"b": {Plan: "pro", FiveHour: &Window{pct(100), iso(8910)}, SevenDay: &Window{pct(14), iso(557970)}, FetchedAt: ts},
	}.Save(main)
	Ledger{
		"s1":  {Day: t0.Format("2006-01-02"), Cost: 0.52, Account: "c"},
		"old": {Day: t0.Format("2006-01") + "-02", Cost: 25.18},
	}.Save(main)
	h := 4.8
	fsutil.WriteJSON(filepath.Join(main, "statusline-pet.json"),
		pet.Save{Life: 1, Line: "bird", Branch: "fowl", Hours: &h, Level: 7, Tick: ts - 60}, false)
	input := fmt.Sprintf(`{"session_id":"s9","model":{"display_name":"Opus 5.5"},"context_window":{"used_percentage":10},`+
		`"rate_limits":{"five_hour":{"used_percentage":31,"resets_at":%d},"seven_day":{"used_percentage":8,"resets_at":%d}}}`,
		t0.Unix()+7470, t0.Unix()+273570)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	out := &bytes.Buffer{}
	return &fixture{main: main, out: out, opts: Options{MainDir: main, In: strings.NewReader(input), Out: out,
		Now: t0, Fetcher: fetcher(srv), Rand: rand.New(rand.NewSource(1))}}
}

func (f *fixture) setLayout(t *testing.T, layout string) {
	cfg, _ := config.Load(f.main)
	cfg.Layout = layout
	cfg.Save(f.main)
}

func (f *fixture) run() string {
	Run(f.opts)
	return style.Strip(f.out.String())
}

func TestRunFull(t *testing.T) {
	want := strings.Join([]string{
		"● A Pro  │ 5h ▰▰▰▱▱▱▱▱▱▱  31% ↻ 2h05m  │ wk ▰▱▱▱▱▱▱▱▱▱   8% ↻ 3d04h  │ Opus 5.5 │ 🐔✨ Lv7",
		"○ B Pro  │ 5h ▰▰▰▰▰▰▰▰▰▰ 100% ↻ 2h29m  │ wk ▰▱▱▱▱▱▱▱▱▱  14% ↻ 6d11h ",
		"○ C API  │ today $0.52 │ month $25.70",
	}, "\n")
	if got := newFixture(t, "en").run(); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRunCompact(t *testing.T) {
	f := newFixture(t, "en")
	f.setLayout(t, config.LayoutCompact)
	want := "● A Pro 5h ▰▰▰▱▱▱▱▱▱▱ 31% wk 8% │ ○ B Pro 5h ▰▰▰▰▰▰▰▰▰▰ 100% ↻2h29m wk 14% │ ○ C API today $0.52 │ Opus 5.5 │ 🐔✨ Lv7"
	if got := f.run(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestRunChinese(t *testing.T) {
	f := newFixture(t, "zh-TW")
	got := f.run()
	for _, s := range []string{"│ 週 ▰▱▱▱▱▱▱▱▱▱   8%", "今日 $0.52 │ 本月 $25.70"} {
		if !strings.Contains(got, s) {
			t.Fatalf("missing %q in\n%s", s, got)
		}
	}
}

func TestRunCachesRateLimitsOfRunningAccount(t *testing.T) {
	f := newFixture(t, "en")
	f.run()
	if e := LoadCache(f.main)["a"]; e == nil || *e.FiveHour.Utilization != 31 || e.Plan != "pro" {
		t.Fatalf("got %+v", e)
	}
}

func TestRunFetchesStaleAccount(t *testing.T) {
	f := newFixture(t, "en")
	writeCreds(t, f.main+"-b", "tok-b", 9e12, "max")
	Cache{"b": {Plan: "pro", FetchedAt: unixSecs(t0) - 1000}}.Save(f.main)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok-b" {
			t.Errorf("auth %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(usageBody))
	}))
	defer srv.Close()
	f.opts.Fetcher = fetcher(srv)
	if got := f.run(); !strings.Contains(got, "○ B Max  │ 5h ▰▰▰▱▱▱▱▱▱▱  31%") {
		t.Fatalf("got\n%s", got)
	}
	if e := LoadCache(f.main)["b"]; e.FetchedAt != unixSecs(t0) {
		t.Fatalf("cache not updated: %+v", e)
	}
}

func TestRunShowsStaleCacheOnError(t *testing.T) {
	f := newFixture(t, "en")
	Cache{"b": {Plan: "pro", FiveHour: &Window{Utilization: pct(50)}, FetchedAt: unixSecs(t0) - 3*3600}}.Save(f.main)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // every request fails
	f.opts.Fetcher = fetcher(srv)
	got := f.run()
	if !strings.Contains(got, "○ B Pro  │ 5h ▰▰▰▰▰▱▱▱▱▱  50%") || !strings.Contains(got, "cached · 3h ago") {
		t.Fatalf("got\n%s", got)
	}
}

func TestRunHiddenAccounts(t *testing.T) {
	f := newFixture(t, "en")
	cfg, _ := config.Load(f.main)
	cfg.Find("a").StatusLine = false
	cfg.Find("b").StatusLine = false
	cfg.Save(f.main)
	got := f.run()
	if strings.Contains(got, "B Pro") || !strings.Contains(got, "● A Pro") {
		t.Fatalf("hidden b should go, running a should stay:\n%s", got)
	}
}

func TestRunActiveAPIAccountRecordsCost(t *testing.T) {
	f := newFixture(t, "en")
	t.Setenv("CLAUDE_CONFIG_DIR", f.main+"-c")
	f.opts.In = strings.NewReader(`{"session_id":"s9","cost":{"total_cost_usd":0.40}}`)
	got := f.run()
	if !strings.Contains(got, "● C API  │ session $0.40 │ today $0.92 │ month $26.10") {
		t.Fatalf("got\n%s", got)
	}
	if e := LoadLedger(f.main)["s9"]; e == nil || e.Account != "c" {
		t.Fatalf("ledger: %+v", e)
	}
}

// Review Focus 2 and 3: a broken config and garbage input still print a status line.
func TestRunSurvivesBadInput(t *testing.T) {
	f := newFixture(t, "en")
	os.WriteFile(filepath.Join(f.main, config.FileName), []byte("{broken"), 0o644)
	f.opts.In = strings.NewReader("not json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	f.opts.Fetcher = fetcher(srv)
	got := f.run()
	if !strings.Contains(got, "● A") || !strings.Contains(got, "🐔 Lv7") {
		t.Fatalf("got\n%s", got)
	}
	if data, _ := os.ReadFile(filepath.Join(f.main, config.FileName)); string(data) != "{broken" {
		t.Fatal("the status line must not rewrite the config")
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/statusline/ -run Run`
Expected: FAIL，`undefined: Run`。

- [ ] **Step 3: 實作** `internal/statusline/render.go`

```go
package statusline

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

const barWidth = 10

// fmtRemaining is the time left until an ISO reset time, rounded up to the minute.
func fmtRemaining(resetsAt string, now time.Time) string {
	t, err := time.Parse(time.RFC3339, resetsAt)
	if err != nil {
		return ""
	}
	minutes := int(math.Floor(t.Sub(now).Seconds()/60)) + 1
	if minutes <= 0 {
		return ""
	}
	days, hours, mins := minutes/1440, minutes%1440/60, minutes%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd%02dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh%02dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func levelColor(pct float64) string {
	switch {
	case pct < 50:
		return style.Low
	case pct < 80:
		return style.Mid
	}
	return style.High
}

func bar(pct float64) string {
	filled := int(math.RoundToEven(pct / 100 * barWidth))
	filled = max(0, min(barWidth, filled))
	if pct > 0 && filled == 0 {
		filled = 1
	}
	return levelColor(pct) + strings.Repeat("▰", filled) + style.Track + strings.Repeat("▱", barWidth-filled) + style.Reset
}

// windowState is a window's percent used and reset time; a cached window whose reset
// time has passed is back to 0%.
func windowState(w *Window, stale bool, now time.Time) (float64, string, bool) {
	if w == nil || w.Utilization == nil {
		return 0, "", false
	}
	if stale && w.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, w.ResetsAt); err == nil && t.Before(now) {
			return 0, "", true
		}
	}
	return *w.Utilization, w.ResetsAt, true
}

func fmtWindow(label string, w *Window, stale bool, now time.Time) string {
	head := style.Muted + label + style.Reset + " "
	pct, resets, ok := windowState(w, stale, now)
	if !ok {
		return head + style.Track + strings.Repeat("▱", barWidth) + style.Reset + " " + style.Muted + "   —" + style.Reset
	}
	s := head + bar(pct) + " " + levelColor(pct) + style.Bold + fmt.Sprintf("%3.0f%%", pct) + style.Reset
	if r := fmtRemaining(resets, now); r != "" {
		return s + " " + style.Muted + "↻ " + fmt.Sprintf("%-6s", r) + style.Reset
	}
	return s + strings.Repeat(" ", 9)
}

func compactWindow(label string, w *Window, stale, withBar bool, now time.Time) string {
	pct, resets, ok := windowState(w, stale, now)
	if !ok {
		return style.Muted + label + " —" + style.Reset
	}
	s := style.Muted + label + style.Reset + " "
	if withBar {
		s += bar(pct) + " "
	}
	s += levelColor(pct) + style.Bold + fmt.Sprintf("%.0f%%", pct) + style.Reset
	if pct >= 80 {
		if r := fmtRemaining(resets, now); r != "" {
			s += " " + style.Muted + "↻" + r + style.Reset
		}
	}
	return s
}

func fmtAge(lang i18n.Lang, d time.Duration) string {
	switch secs := d.Seconds(); {
	case secs < 3600:
		return lang.T("ago_m", int(secs/60))
	case secs < 86400:
		return lang.T("ago_h", int(secs/3600))
	default:
		return lang.T("ago_d", int(secs/86400))
	}
}

func badge(plan string) string {
	if plan == "api" {
		return "API"
	}
	if plan == "" {
		return ""
	}
	return strings.ToUpper(plan[:1]) + strings.ToLower(plan[1:])
}

func dot(active bool) string {
	if active {
		return style.Accent + "●" + style.Reset
	}
	return style.Muted + "○" + style.Reset
}

func fmtTag(name, plan string, active bool) string {
	color := style.Text
	if active {
		color += style.Bold
	}
	return dot(active) + " " + color + strings.ToUpper(name) + style.Reset + " " + style.Muted + fmt.Sprintf("%-4s", badge(plan)) + style.Reset
}

func compactTag(name, plan string, active bool) string {
	s := dot(active) + " " + style.Text
	if active {
		s += style.Bold
	}
	s += strings.ToUpper(name) + style.Reset
	if b := badge(plan); b != "" {
		s += " " + style.Muted + b + style.Reset
	}
	return s
}

func money(label string, x float64) string {
	return style.Muted + label + style.Reset + " " + style.Text + style.Bold + fmt.Sprintf("$%.2f", x) + style.Reset
}

func errorText(err error, lang i18n.Lang) string {
	var he *HTTPError
	var ue *url.Error
	switch {
	case err == nil:
		return lang.T("no_data")
	case errors.Is(err, ErrNotLoggedIn):
		return lang.T("not_logged_in")
	case errors.Is(err, ErrTokenExpired):
		return lang.T("token_expired")
	case errors.As(err, &he):
		return he.Error()
	case errors.As(err, &ue):
		return lang.T("network_error")
	}
	return err.Error()
}

// subscriptionRow is the (full line, compact segment) of a Pro/Max account.
func subscriptionRow(acc config.Account, e *Entry, err error, active bool, now time.Time, lang i18n.Lang) (string, string) {
	if e == nil {
		msg := style.Muted + style.Italic + errorText(err, lang) + style.Reset
		full := fmtTag(acc.Name, "", active) + style.Sep + msg
		if errors.Is(err, ErrNotLoggedIn) {
			full += " " + style.Track + "·" + style.Reset + " " + style.Muted + lang.T("login_hint", acc.Name) + style.Reset
		}
		return full, compactTag(acc.Name, "", active) + " " + msg
	}
	age := now.Sub(fromUnix(e.FetchedAt))
	stale := err != nil || age > 3*CacheTTL
	full := fmtTag(acc.Name, e.Plan, active) + style.Sep + fmtWindow("5h", e.FiveHour, stale, now) +
		style.Sep + fmtWindow(lang.T("wk"), e.SevenDay, stale, now)
	compact := compactTag(acc.Name, e.Plan, active) + " " + compactWindow("5h", e.FiveHour, stale, true, now) +
		" " + compactWindow(lang.T("wk"), e.SevenDay, stale, false, now)
	if stale {
		full += style.Sep + style.Muted + style.Italic + lang.T("cached", fmtAge(lang, age)) + style.Reset
		compact += style.Muted + "*" + style.Reset
	}
	return full, compact
}

// apiRow is the (full line, compact segment) of an API-billed account; ok is false for
// an account with no folder and no spend on this computer.
func apiRow(acc config.Account, l Ledger, in Input, active bool, now time.Time, lang i18n.Lang) (string, string, bool) {
	if !active && !l.Has(acc.Name) {
		if _, err := os.Stat(acc.Path()); err != nil {
			return "", "", false
		}
	}
	today := now.Format("2006-01-02")
	day := l.Spent(acc.Name, func(d string) bool { return d == today })
	month := l.Spent(acc.Name, func(d string) bool { return strings.HasPrefix(d, today[:7]) })
	parts := []string{money(lang.T("today"), day), money(lang.T("month"), month)}
	if active && in.Cost.TotalCostUSD != nil {
		parts = append([]string{money(lang.T("session"), *in.Cost.TotalCostUSD)}, parts...)
	}
	full := fmtTag(acc.Name, "api", active) + style.Sep + strings.Join(parts, style.Sep)
	return full, compactTag(acc.Name, "api", active) + " " + money(lang.T("today"), day), true
}
```

- [ ] **Step 4: 實作** `internal/statusline/run.go`

```go
package statusline

import (
	"io"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/pet"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

type Options struct {
	MainDir string
	In      io.Reader
	Out     io.Writer
	Now     time.Time
	Fetcher *Fetcher
	Rand    *rand.Rand
}

// Run prints the status line. Without a readable config it shows the main account
// alone, so the status line works before setup and with a broken file.
func Run(o Options) {
	in := ParseInput(o.In)
	cfg, err := config.Load(o.MainDir)
	if err != nil {
		cfg = config.New(o.MainDir)
	}
	lang := i18n.Resolve(cfg.Lang)
	activeDir := config.ActiveDir()

	var shown []config.Account
	active := map[string]bool{}
	for _, acc := range cfg.Accounts {
		active[acc.Name] = config.SamePath(acc.Path(), activeDir)
		if acc.StatusLine || active[acc.Name] {
			shown = append(shown, acc)
		}
	}

	cache := LoadCache(o.MainDir)
	errs := map[string]error{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, acc := range shown {
		if acc.Type == config.TypeAPI {
			continue
		}
		if active[acc.Name] {
			plan := Plan(acc.Path())
			if old := cache[acc.Name]; plan == "" && old != nil {
				plan = old.Plan
			}
			if e := FromRateLimits(in.RateLimits, plan, o.Now); e != nil {
				cache[acc.Name] = e
				continue
			}
		}
		if e := cache[acc.Name]; e != nil && o.Now.Sub(fromUnix(e.FetchedAt)) <= CacheTTL {
			continue
		}
		wg.Add(1)
		go func(acc config.Account, isActive bool) {
			defer wg.Done()
			e, err := o.Fetcher.Fetch(acc.Path(), isActive, o.Now)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs[acc.Name] = err
				if old := cache[acc.Name]; old != nil {
					old.LastError = err.Error()
				}
				return
			}
			cache[acc.Name] = e
		}(acc, active[acc.Name])
	}
	wg.Wait()
	cache.Save(o.MainDir)

	ledger := LoadLedger(o.MainDir)
	var fulls, compacts []string
	for _, acc := range shown {
		if acc.Type == config.TypeAPI {
			if active[acc.Name] && in.SessionID != "" && in.Cost.TotalCostUSD != nil {
				ledger.Record(acc.Name, in.SessionID, *in.Cost.TotalCostUSD, o.Now)
				ledger.Save(o.MainDir)
			}
			if full, compact, ok := apiRow(acc, ledger, in, active[acc.Name], o.Now, lang); ok {
				fulls, compacts = append(fulls, full), append(compacts, compact)
			}
			continue
		}
		full, compact := subscriptionRow(acc, cache[acc.Name], errs[acc.Name], active[acc.Name], o.Now, lang)
		fulls, compacts = append(fulls, full), append(compacts, compact)
	}

	var tail []string
	if in.Model.DisplayName != "" {
		tail = append(tail, style.Muted+in.Model.DisplayName+style.Reset)
	}
	tail = append(tail, pet.Update(filepath.Join(o.MainDir, "statusline-pet.json"), in.ContextPct(), o.Now, o.Rand))

	var lines []string
	switch {
	case cfg.Layout == config.LayoutCompact:
		lines = []string{strings.Join(append(compacts, tail...), style.Sep)}
	case len(fulls) == 0:
		lines = []string{strings.Join(tail, style.Sep)}
	default:
		lines = fulls
		lines[0] += style.Sep + strings.Join(tail, style.Sep)
	}
	io.WriteString(o.Out, strings.Join(lines, "\n"))
}
```

- [ ] **Step 5: 確認測試通過**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./internal/statusline/ && go vet ./...`
Expected: `ok`。若完整版或緊湊版的字串不符，先對照 Python 版的 `fmt_window`／`compact_window` 找出差異，修正程式而不是改預期字串。

- [ ] **Step 6: Commit**

```bash
git add internal/statusline
git commit -F - <<'EOF'
feat: 狀態列完整版／緊湊版排版、並行查詢與隱藏帳號

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 12: 主程式、管理子指令與互動選單

**Files:**
- Create: `cmd/claude-accounts/main.go`, `cmd/claude-accounts/app.go`, `cmd/claude-accounts/menu.go`, `cmd/claude-accounts/main_test.go`

**Interfaces:**
- Consumes: 前面所有套件（`config`、`i18n`、`accounts`、`links`、`settings`、`launch`、`migrate`、`statusline`）
- Produces: 執行檔 `claude-accounts`；`main.version`（由 `-ldflags "-X main.version=..."` 設定）

- [ ] **Step 1: 寫失敗的測試** `cmd/claude-accounts/main_test.go`

```go
package main

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/links"
	"github.com/MFpizza/claude-dotfiles/internal/migrate"
)

type testEnv struct {
	home, main, exe string
	out             *bytes.Buffer
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()
	home := t.TempDir()
	for k, v := range map[string]string{"HOME": home, "USERPROFILE": home, "CLAUDE_CONFIG_DIR": "",
		"LC_ALL": "", "LC_MESSAGES": "", "LANG": "en_US.UTF-8"} {
		t.Setenv(k, v)
	}
	exe := filepath.Join(home, "bin", "claude-accounts")
	os.MkdirAll(filepath.Dir(exe), 0o755)
	os.WriteFile(exe, []byte("binary"), 0o755)
	return &testEnv{home: home, main: filepath.Join(home, ".claude"), exe: exe, out: &bytes.Buffer{}}
}

func (e *testEnv) app(t *testing.T, input string) *app {
	t.Helper()
	a := &app{mainDir: config.ResolveMainDir(""), exe: e.exe, in: bufio.NewReader(strings.NewReader(input)),
		out: e.out, shellFiles: func() []string { return []string{filepath.Join(e.home, ".bashrc")} }}
	if err := a.load(); err != nil {
		t.Fatal(err)
	}
	a.lang = i18n.En
	return a
}

func TestCommandName(t *testing.T) {
	cases := map[string]string{"/usr/local/bin/claude-b": "claude-b", "claude-accounts": "claude-accounts", "claude-C.exe": "claude-c"}
	for in, want := range cases {
		if got := commandName(in); got != want {
			t.Errorf("%s: %q", in, got)
		}
	}
	if accountCmd.MatchString("claude-accounts") || !accountCmd.MatchString("claude-z") {
		t.Error("account command pattern")
	}
}

func TestSetupMigratesOldInstall(t *testing.T) {
	e := newEnv(t)
	accounts.Share(e.main, e.main+"-b")
	os.WriteFile(filepath.Join(e.home, ".bashrc"),
		[]byte("alias x=y\n"+migrate.Begin+"\n. '/old/claude-accounts.sh'\n"+migrate.End+"\n"), 0o644)
	os.WriteFile(filepath.Join(e.main, "statusline.json"), []byte(`{"layout":"compact"}`), 0o644)
	if err := e.app(t, "").dispatch("list", nil); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(e.main)
	if err != nil || strings.Join(cfg.Names(), ",") != "a,b" || cfg.Layout != config.LayoutCompact {
		t.Fatalf("config %+v %v", cfg, err)
	}
	if rc, _ := os.ReadFile(filepath.Join(e.home, ".bashrc")); string(rc) != "alias x=y\n" {
		t.Fatalf("bashrc: %q", rc)
	}
	settingsJSON, _ := os.ReadFile(filepath.Join(e.main, "settings.json"))
	if !strings.Contains(string(settingsJSON), "statusline --base") {
		t.Fatalf("settings: %s", settingsJSON)
	}
	for _, n := range []string{"a", "b"} {
		if !links.Points(accounts.CommandPath(filepath.Dir(e.exe), n), e.exe) {
			t.Errorf("claude-%s missing", n)
		}
	}
	if !strings.Contains(e.out.String(), "Found account B") {
		t.Fatalf("output:\n%s", e.out)
	}
}

func TestAddRemoveFlow(t *testing.T) {
	e := newEnv(t)
	if err := e.app(t, "").dispatch("add", []string{"--api"}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(e.main)
	if b := cfg.Find("b"); b == nil || b.Type != config.TypeAPI {
		t.Fatalf("got %+v", cfg.Accounts)
	}
	if err := e.app(t, "").dispatch("hide", []string{"b"}); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := config.Load(e.main); cfg.Find("b").StatusLine {
		t.Fatal("hide did not stick")
	}
	if err := e.app(t, "").dispatch("remove", []string{"b", "--yes", "--delete-dir"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(e.main + "-b"); !os.IsNotExist(err) {
		t.Fatal("dir should be deleted")
	}
	if cfg, _ := config.Load(e.main); cfg.Find("b") != nil {
		t.Fatal("b still listed")
	}
}

func TestRemoveAsksBeforeDeleting(t *testing.T) {
	e := newEnv(t)
	e.app(t, "").dispatch("add", nil)
	if err := e.app(t, "y\n\n").dispatch("remove", []string{"b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(e.main + "-b"); err != nil {
		t.Fatal("the folder is kept unless the user says yes")
	}
}

func TestCommandErrors(t *testing.T) {
	e := newEnv(t)
	for _, args := range [][]string{{"remove", "a", "--yes"}, {"remove", "q", "--yes"}, {"layout", "tiny"}, {"lang", "fr"}, {"frobnicate"}} {
		if err := e.app(t, "").dispatch(args[0], args[1:]); err == nil {
			t.Errorf("%v should fail", args)
		}
	}
}

func TestMenuAddsAccountAndChangesLayout(t *testing.T) {
	e := newEnv(t)
	if err := e.app(t, "1\n1\n4\n2\n0\n").dispatch("", nil); err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(e.main)
	if cfg.Find("b") == nil || cfg.Layout != config.LayoutCompact {
		t.Fatalf("got %+v", cfg)
	}
}

func TestMenuStopsAtEOF(t *testing.T) {
	e := newEnv(t)
	if err := e.app(t, "").dispatch("", nil); err != nil {
		t.Fatal(err)
	}
}

func TestUninstall(t *testing.T) {
	e := newEnv(t)
	e.app(t, "").dispatch("add", nil)
	if err := e.app(t, "").dispatch("uninstall", []string{"--yes"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(accounts.CommandPath(filepath.Dir(e.exe), "b")); !os.IsNotExist(err) {
		t.Error("claude-b should be gone")
	}
	if _, err := os.Stat(filepath.Join(e.main, config.FileName)); !os.IsNotExist(err) {
		t.Error("config should be gone")
	}
	if _, err := os.Stat(e.main + "-b"); err != nil {
		t.Error("account folders are kept")
	}
	if data, _ := os.ReadFile(filepath.Join(e.main, "settings.json")); strings.Contains(string(data), "statusLine") {
		t.Error("statusLine should be removed")
	}
}

// Review Focus 3: the status line always prints something and exits 0.
func TestStatuslineCmdNeverFails(t *testing.T) {
	e := newEnv(t)
	var out, errb bytes.Buffer
	code := run([]string{"claude-accounts", "statusline", "--base", e.main}, strings.NewReader("garbage"), &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "Lv1") || !strings.Contains(out.String(), "not logged in") {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
}

func TestLaunchUnknownAccount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("argv[0] dispatch is the same; launching is covered by internal/launch")
	}
	e := newEnv(t)
	e.app(t, "").dispatch("list", nil)
	var out, errb bytes.Buffer
	if code := run([]string{"/x/claude-q"}, strings.NewReader(""), &out, &errb); code != 1 || !strings.Contains(errb.String(), "no account q") {
		t.Fatalf("code %d err %q", code, errb.String())
	}
}
```

- [ ] **Step 2: 確認測試失敗**

Run: `export PATH="$HOME/.local/go/bin:$PATH"; go test ./cmd/claude-accounts/`
Expected: FAIL，`undefined: app`。

- [ ] **Step 3: 實作** `cmd/claude-accounts/main.go`

```go
// claude-accounts runs several Claude Code accounts side by side. Called as
// claude-<letter> it starts that account; otherwise it manages accounts or, as
// `claude-accounts statusline`, renders the status line.
package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/launch"
	"github.com/MFpizza/claude-dotfiles/internal/statusline"
)

var version = "dev"

var accountCmd = regexp.MustCompile(`^claude-([a-z])$`)

func main() { os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr)) }

// commandName is the name this program was started as, e.g. "claude-b".
func commandName(argv0 string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(argv0)), ".exe")
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if m := accountCmd.FindStringSubmatch(commandName(args[0])); m != nil {
		return launchAccount(m[1], args[1:], stderr)
	}
	sub, rest := "", args[1:]
	if len(rest) > 0 {
		sub, rest = rest[0], rest[1:]
	}
	switch sub {
	case "statusline":
		return statuslineCmd(rest, stdin, stdout)
	case "version", "--version":
		fmt.Fprintln(stdout, version)
		return 0
	}
	a, err := newApp(stdin, stdout)
	if err == nil {
		err = a.dispatch(sub, rest)
	}
	if err != nil {
		fmt.Fprintln(stderr, "claude-accounts:", err)
		return 1
	}
	return 0
}

func launchAccount(name string, args []string, stderr io.Writer) int {
	mainDir := config.ResolveMainDir("")
	cfg, err := config.Load(mainDir)
	if err != nil {
		fmt.Fprintln(stderr, i18n.Resolve("auto").T("err_not_set_up"))
		return 1
	}
	lang := i18n.Resolve(cfg.Lang)
	plan, err := launch.Build(cfg, mainDir, name, args, exec.LookPath)
	var missing *launch.DirMissingError
	switch {
	case errors.Is(err, accounts.ErrNoAccount):
		err = errors.New(lang.T("err_no_account", name, strings.Join(cfg.Names(), ", ")))
	case errors.As(err, &missing):
		err = errors.New(lang.T("err_dir_missing", name, missing.Dir, name))
	case errors.Is(err, launch.ErrNoClaude):
		err = errors.New(lang.T("err_no_claude"))
	case err == nil:
		err = launch.Exec(plan)
	}
	if err != nil {
		fmt.Fprintln(stderr, "claude-"+name+":", err)
		return 1
	}
	return 0
}

// statuslineCmd never fails: Claude Code would just show an empty status line.
func statuslineCmd(args []string, stdin io.Reader, stdout io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stdout, "claude-accounts: %v", r)
		}
	}()
	now := time.Now()
	statusline.Run(statusline.Options{
		MainDir: config.ResolveMainDir(flagValue(args, "--base")),
		In:      stdin, Out: stdout, Now: now,
		Fetcher: statusline.NewFetcher(),
		Rand:    rand.New(rand.NewSource(now.UnixNano())),
	})
	return 0
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// positional drops flags (and --dir's value) from args.
func positional(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--dir":
			i++
		case !strings.HasPrefix(args[i], "--"):
			out = append(out, args[i])
		}
	}
	return out
}
```

- [ ] **Step 4: 實作** `cmd/claude-accounts/app.go`

```go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/links"
	"github.com/MFpizza/claude-dotfiles/internal/migrate"
	"github.com/MFpizza/claude-dotfiles/internal/settings"
	"github.com/MFpizza/claude-dotfiles/internal/statusline"
)

type app struct {
	mainDir    string
	exe        string
	cfg        *config.Config // nil until set up
	lang       i18n.Lang
	in         *bufio.Reader
	out        io.Writer
	eof        bool
	shellFiles func() []string
}

func newApp(stdin io.Reader, stdout io.Writer) (*app, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	a := &app{mainDir: config.ResolveMainDir(""), exe: exe, in: bufio.NewReader(stdin), out: stdout,
		shellFiles: func() []string { return migrate.ShellFiles(config.Home()) }}
	return a, a.load()
}

// load reads the config; a missing one means "not set up yet", a broken one is an error.
func (a *app) load() error {
	cfg, err := config.Load(a.mainDir)
	switch {
	case err == nil:
		a.cfg = cfg
	case !errors.Is(err, fs.ErrNotExist):
		return errors.New(i18n.Resolve("auto").T("err_config", filepath.Join(a.mainDir, config.FileName), err))
	}
	setting := "auto"
	if a.cfg != nil {
		setting = a.cfg.Lang
	}
	a.lang = i18n.Resolve(setting)
	return nil
}

func (a *app) say(key string, args ...any) { fmt.Fprint(a.out, a.lang.T(key, args...)) }

func (a *app) ask(key string, args ...any) string {
	a.say(key, args...)
	line, err := a.in.ReadString('\n')
	if err != nil && line == "" {
		a.eof = true
	}
	return strings.TrimSpace(line)
}

func (a *app) confirm(defaultYes bool, key string, args ...any) bool {
	switch strings.ToLower(a.ask(key, args...)) {
	case "":
		return defaultYes
	case "y", "yes":
		return true
	}
	return false
}

func (a *app) fail(key string, args ...any) error { return errors.New(a.lang.T(key, args...)) }

func (a *app) save() error { return a.cfg.Save(a.mainDir) }

func (a *app) dispatch(sub string, args []string) error {
	switch sub {
	case "help", "-h", "--help":
		a.say("usage")
		return nil
	case "uninstall":
		if a.cfg == nil {
			a.cfg = migrate.Detect(a.mainDir)
		}
		return a.uninstall(hasFlag(args, "--yes"))
	}
	if err := a.ensureSetup(sub == ""); err != nil {
		return err
	}
	pos := positional(args)
	switch sub {
	case "":
		return a.menu()
	case "add":
		return a.add(hasFlag(args, "--api"), flagValue(args, "--dir"))
	case "list":
		a.list()
		return nil
	case "remove", "show", "hide":
		if len(pos) != 1 {
			return a.fail("err_usage_letter", sub)
		}
		if sub == "remove" {
			return a.remove(pos[0], hasFlag(args, "--yes"), hasFlag(args, "--delete-dir"))
		}
		return a.setShown(pos[0], sub == "show")
	case "layout":
		if len(pos) != 1 {
			return a.fail("err_layout")
		}
		return a.setLayout(pos[0])
	case "lang":
		if len(pos) != 1 {
			return a.fail("err_lang")
		}
		return a.setLang(pos[0])
	}
	return a.fail("err_unknown_cmd", sub)
}

func (a *app) ensureSetup(interactive bool) error {
	if a.cfg != nil {
		return a.repair()
	}
	return a.setup(interactive)
}

// setup is the first run: it imports accounts of the old version and removes what
// its installer added, then installs commands and the status line.
func (a *app) setup(interactive bool) error {
	a.say("setup_title", config.CollapseHome(a.mainDir))
	cfg := migrate.Detect(a.mainDir)
	for i := 1; i < len(cfg.Accounts); i++ {
		acc := &cfg.Accounts[i]
		a.say("found_account", strings.ToUpper(acc.Name), acc.Dir)
		if acc.Type == config.TypeAPI && interactive && !a.confirm(true, "confirm_api", strings.ToUpper(acc.Name)) {
			acc.Type = config.TypeSubscription
		}
	}
	if layout := migrate.LegacyLayout(a.mainDir); layout != "" {
		cfg.Layout = layout
	}
	for _, f := range a.shellFiles() {
		if ok, err := migrate.StripBlock(f); err == nil && ok {
			a.say("cleaned_block", f, f)
		}
	}
	for _, f := range migrate.OldShims(config.Home()) {
		if os.Remove(f) == nil {
			a.say("removed_file", f)
		}
	}
	if err := os.MkdirAll(a.mainDir, 0o755); err != nil {
		return err
	}
	a.cfg = cfg
	if err := a.save(); err != nil {
		return err
	}
	migrate.RemoveLegacyLayout(a.mainDir)
	if err := a.repair(); err != nil {
		return err
	}
	a.say("setup_done", a.commandList())
	return nil
}

// repair points every command and the status line at this executable, which moves
// after an update or reinstall.
func (a *app) repair() error {
	if err := accounts.SyncCommands(a.cfg, a.exe); err != nil {
		return err
	}
	changed, err := settings.SetStatusLine(a.mainDir, a.exe)
	if err != nil {
		return err
	}
	if changed {
		a.say("statusline_set", filepath.Join(a.mainDir, "settings.json"))
	}
	return nil
}

func (a *app) commandList() string {
	var names []string
	for _, n := range a.cfg.Names() {
		names = append(names, "claude-"+n)
	}
	return strings.Join(names, ", ")
}

func (a *app) typeLabel(t string) string {
	if t == config.TypeAPI {
		return a.lang.T("type_api")
	}
	return a.lang.T("type_subscription")
}

func (a *app) noAccount(name string) error {
	return a.fail("err_no_account", name, strings.Join(a.cfg.Names(), ", "))
}

func (a *app) add(api bool, dir string) error {
	acc, conflicts, err := accounts.Add(a.cfg, a.mainDir, a.exe, api, dir)
	if errors.Is(err, config.ErrFull) {
		return a.fail("err_full")
	}
	if err != nil {
		return err
	}
	for _, c := range conflicts {
		a.say("conflict", acc.Dir, c)
	}
	if err := a.save(); err != nil {
		return err
	}
	a.say("added", strings.ToUpper(acc.Name), a.typeLabel(acc.Type), acc.Dir, acc.Name)
	if api {
		a.say("added_api", filepath.Join(acc.Dir, "account-settings.json"))
	}
	return nil
}

func (a *app) remove(name string, yes, deleteDir bool) error {
	if name == config.MainName {
		return a.fail("err_main")
	}
	acc := a.cfg.Find(name)
	if acc == nil {
		return a.noAccount(name)
	}
	if !yes && !a.confirm(false, "confirm_remove", strings.ToUpper(name)) {
		return nil
	}
	if !yes && !deleteDir {
		deleteDir = a.confirm(false, "confirm_delete_dir", acc.Dir)
	}
	dir := acc.Dir
	if err := accounts.Remove(a.cfg, a.exe, name, deleteDir); err != nil {
		return err
	}
	statusline.Forget(a.mainDir, name)
	if err := a.save(); err != nil {
		return err
	}
	a.say("removed", strings.ToUpper(name))
	if deleteDir {
		a.say("deleted_dir", dir)
	} else {
		a.say("kept_dir", dir)
	}
	return nil
}

func (a *app) list() {
	a.say("accounts_header")
	for _, acc := range a.cfg.Accounts {
		shown := a.lang.T("hidden")
		if acc.StatusLine {
			shown = a.lang.T("shown")
		}
		a.say("account_row", acc.Name, a.typeLabel(acc.Type), acc.Dir, shown)
	}
}

func (a *app) setShown(name string, show bool) error {
	acc := a.cfg.Find(name)
	if acc == nil {
		return a.noAccount(name)
	}
	acc.StatusLine = show
	if err := a.save(); err != nil {
		return err
	}
	if show {
		a.say("set_shown", strings.ToUpper(name))
	} else {
		a.say("set_hidden", strings.ToUpper(name))
	}
	return nil
}

func (a *app) setLayout(layout string) error {
	if layout != config.LayoutFull && layout != config.LayoutCompact {
		return a.fail("err_layout")
	}
	a.cfg.Layout = layout
	if err := a.save(); err != nil {
		return err
	}
	a.say("layout_set", layout)
	return nil
}

func (a *app) setLang(lang string) error {
	if lang != "auto" && lang != string(i18n.En) && lang != string(i18n.ZhTW) {
		return a.fail("err_lang")
	}
	a.cfg.Lang = lang
	a.lang = i18n.Resolve(lang)
	if err := a.save(); err != nil {
		return err
	}
	a.say("lang_set", lang)
	return nil
}

// uninstall removes the commands, the status line and the config; account folders
// and their data stay.
func (a *app) uninstall(yes bool) error {
	if !yes && !a.confirm(false, "confirm_uninstall") {
		return nil
	}
	for _, acc := range a.cfg.Accounts {
		if err := links.RemoveExe(accounts.CommandPath(filepath.Dir(a.exe), acc.Name)); err != nil {
			return err
		}
	}
	if err := settings.RemoveStatusLine(a.mainDir); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(a.mainDir, config.FileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	a.say("uninstalled", a.exe)
	return nil
}
```

- [ ] **Step 5: 實作** `cmd/claude-accounts/menu.go`

```go
package main

import (
	"fmt"

	"github.com/MFpizza/claude-dotfiles/internal/config"
)

// menu is the numbered menu shown when claude-accounts runs without arguments.
func (a *app) menu() error {
	for !a.eof {
		fmt.Fprintln(a.out)
		a.list()
		a.say("menu", a.cfg.Layout, a.cfg.Lang)
		var err error
		switch a.ask("prompt") {
		case "1":
			err = a.add(a.ask("ask_type") == "2", "")
		case "2":
			err = a.remove(a.ask("ask_which"), false, false)
		case "3":
			name := a.ask("ask_which")
			if acc := a.cfg.Find(name); acc == nil {
				err = a.noAccount(name)
			} else {
				err = a.setShown(name, !acc.StatusLine)
			}
		case "4":
			switch a.ask("ask_layout") {
			case "1":
				err = a.setLayout(config.LayoutFull)
			case "2":
				err = a.setLayout(config.LayoutCompact)
			}
		case "5":
			switch a.ask("ask_lang") {
			case "1":
				err = a.setLang("auto")
			case "2":
				err = a.setLang("en")
			case "3":
				err = a.setLang("zh-TW")
			}
		case "6":
			return a.uninstall(false)
		case "0", "":
			return nil
		default:
			a.say("invalid_choice")
		}
		if err != nil {
			fmt.Fprintln(a.out, err)
		}
	}
	return nil
}
```

- [ ] **Step 6: 確認測試通過，並以真正的執行檔試跑（在暫存 HOME 裡，不碰真實設定）**

Run:
```bash
export PATH="$HOME/.local/go/bin:$PATH"
go test ./... && GOOS=windows go vet ./... && GOOS=darwin go vet ./...
T=$(mktemp -d); go build -o "$T/bin/claude-accounts" ./cmd/claude-accounts
HOME=$T CLAUDE_CONFIG_DIR= LANG=en_US.UTF-8 "$T/bin/claude-accounts" add
HOME=$T CLAUDE_CONFIG_DIR= LANG=zh_TW.UTF-8 "$T/bin/claude-accounts" list
ls -l "$T/bin"
echo '{"model":{"display_name":"Opus 5.5"}}' | HOME=$T CLAUDE_CONFIG_DIR= "$T/bin/claude-accounts" statusline --base "$T/.claude"; echo
HOME=$T CLAUDE_CONFIG_DIR= CLAUDE_BIN=/bin/echo "$T/bin/claude-b" --continue
```
Expected: 測試全過；`add` 印出 `✓ Added account B`；`list` 用中文列出 a、b；`bin/` 裡有 `claude-a`、`claude-b` 兩個指向執行檔的 symlink；狀態列輸出 `○ A … not logged in` 等內容與 `Lv1`；最後一行印出 `--continue`（`CLAUDE_BIN=/bin/echo` 被當成 claude 執行）。

- [ ] **Step 7: Commit**

```bash
git add cmd
git commit -F - <<'EOF'
feat: claude-accounts 主程式、管理子指令與互動選單

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---
### Task 13: 一行安裝腳本、CI 與 Release

**Files:**
- Create: `install.sh`, `install.ps1`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`

**Interfaces:**
- Consumes: Task 12 的執行檔；Release 檔名 `claude-accounts-<os>-<arch>[.exe]`（`os` ∈ windows/linux/darwin，`arch` ∈ amd64/arm64）
- Produces: `install.sh`、`install.ps1`（環境變數 `CLAUDE_ACCOUNTS_URL` 可覆寫下載網址，`CLAUDE_ACCOUNTS_DIR` 可覆寫安裝目錄，供測試使用）

- [ ] **Step 1: 寫** `install.sh`

```sh
#!/bin/sh
# Installs or updates claude-accounts:
#   curl -fsSL https://raw.githubusercontent.com/MFpizza/claude-dotfiles/master/install.sh | sh
set -eu

case "$(uname -s)" in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
    x86_64 | amd64) arch=amd64 ;;
    aarch64 | arm64) arch=arm64 ;;
    *) echo "Unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac

url="${CLAUDE_ACCOUNTS_URL:-https://github.com/MFpizza/claude-dotfiles/releases/latest/download/claude-accounts-$os-$arch}"
dir="${CLAUDE_ACCOUNTS_DIR:-$HOME/.local/bin}"
mkdir -p "$dir"
tmp="$dir/.claude-accounts.download"
echo "Downloading $url"
curl -fsSL -o "$tmp" "$url"
chmod +x "$tmp"
mv -f "$tmp" "$dir/claude-accounts"

case ":$PATH:" in
    *":$dir:"*) ;;
    *) echo "Add $dir to your PATH, e.g. in ~/.bashrc:  export PATH=\"$dir:\$PATH\"" ;;
esac

# Piped into sh, stdin is this script; the menu needs the terminal instead.
if (exec </dev/tty) 2>/dev/null; then
    "$dir/claude-accounts" </dev/tty
else
    "$dir/claude-accounts" list
fi
```

- [ ] **Step 2: 寫** `install.ps1`

```powershell
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
```

- [ ] **Step 3: 寫** `.github/workflows/ci.yml`

```yaml
name: ci
on: [push, pull_request]
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go vet ./...
      - run: go test ./...
```

- [ ] **Step 4: 寫** `.github/workflows/release.yml`

```yaml
name: release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...
      - name: Build
        run: |
          mkdir dist
          for target in windows/amd64 windows/arm64 linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do
            os=${target%/*}; arch=${target#*/}; ext=""
            [ "$os" = windows ] && ext=.exe
            CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
              -ldflags "-s -w -X main.version=$GITHUB_REF_NAME" \
              -o "dist/claude-accounts-$os-$arch$ext" ./cmd/claude-accounts
          done
      - run: gh release create "$GITHUB_REF_NAME" dist/* --generate-notes
        env:
          GH_TOKEN: ${{ github.token }}
```

- [ ] **Step 5: 用本機編出的執行檔測試 install.sh（暫存 HOME，`file://` 網址）**

Run:
```bash
export PATH="$HOME/.local/go/bin:$PATH"
T=$(mktemp -d); go build -o "$T/dl/claude-accounts-linux-amd64" ./cmd/claude-accounts
HOME=$T CLAUDE_CONFIG_DIR= LANG=en_US.UTF-8 CLAUDE_ACCOUNTS_URL="file://$T/dl/claude-accounts-linux-amd64" sh install.sh </dev/null
ls -l "$T/.local/bin" && cat "$T/.claude/claude-accounts.json"
for target in windows/amd64 windows/arm64 linux/arm64 darwin/amd64 darwin/arm64; do
  GOOS=${target%/*} GOARCH=${target#*/} CGO_ENABLED=0 go build -o /dev/null ./cmd/claude-accounts || echo "FAIL $target"
done
```
Expected: 印出 `Downloading file://…`、PATH 提示與 `Accounts:` 清單；`$T/.local/bin` 有 `claude-accounts` 與 `claude-a`；設定檔存在；6 個平台都編譯成功（沒有 `FAIL`）。`install.ps1` 無法在這台機器上執行，交給 Windows 使用者實測（見 Task 14 最後一步）。

- [ ] **Step 6: Commit**

```bash
chmod +x install.sh
git add install.sh install.ps1 .github
git commit -F - <<'EOF'
feat: 一行安裝腳本與 GitHub Actions CI／Release

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

---

### Task 14: README、移除 Python 版、最終驗證

**Files:**
- Create: `README.zh-TW.md`
- Modify: `README.md`（改寫成英文）
- Delete: `install.py`, `usage_statusline.py`, `claude-accounts.ps1`, `claude-accounts.sh`, `bin/`

**Interfaces:**
- Consumes: 所有子指令的實際行為與輸出

- [ ] **Step 1: 移除 Python 版**

```bash
git rm -q install.py usage_statusline.py claude-accounts.ps1 claude-accounts.sh bin/claude-a.cmd bin/claude-b.cmd bin/claude-c.cmd
```

- [ ] **Step 2: 改寫** `README.md`（英文）

````markdown
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
````

- [ ] **Step 3: 寫** `README.zh-TW.md`

````markdown
# claude-dotfiles

**同時使用多個 Claude Code 帳號，額度用完一鍵換手，對話不中斷。**

```
● A Pro  │ 5h ▰▰▰▰▰▰▰▰▰▱  92% ↻ 0h41m  │ 週 ▰▰▰▰▰▰▰▱▱▱  67% ↻ 1d16h  │ Opus 5.5 │ 🐔✨ Lv7
○ B Pro  │ 5h ▰▱▱▱▱▱▱▱▱▱  13% ↻ 3h46m  │ 週 ▰▰▰▰▱▱▱▱▱▱  44% ↻ 3d21h
```

A 的 5 小時額度快用完了？離開後打 `claude-b --continue`，B 就從剛剛那句話接著做。

[English](README.md)

## 特色

- 🔄 **換帳號不斷線**：所有帳號共用對話紀錄、記憶、skills、agents。
- 📊 **所有帳號的用量一眼看完**：5 小時與每週額度、重置倒數，連沒在用的帳號也看得到；快用完時變成橘色。
- ➕ **帳號要幾個就加幾個**：訂閱（Pro/Max）或 API 計費都行。只有一個帳號也能單獨使用狀態列。
- 🐣 **會長大的小寵物**：用 Claude 越久等級越高，滿級後轉生成別的物種，共 16 種最終型態可以收集。
- 📏 **兩種版型**：每個帳號一行，或全部擠成一行：

  ```
  ● A Pro 5h ▰▰▰▰▰▰▰▰▰▱ 92% ↻0h41m 週 67% │ ○ B Pro 5h ▰▱▱▱▱▱▱▱▱▱ 13% 週 44% │ Opus 5.5 │ 🐔✨ Lv7
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
| `5h ▰▰▰▱▱▱▱▱▱▱ 31% ↻ 2h05m` | 5 小時額度：已用 31%，2 小時 5 分後重置 |
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
| 🍼 哺乳系 | 🐾 → 🐱 🐈 🐆 🐅 🦁、🐶 🐕 🐺、🦦 🐬 🐳 🐋 或 🐵 🙈 🐒 🦧 🦍 |

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
````

- [ ] **Step 4: 最終驗證**

Run:
```bash
export PATH="$HOME/.local/go/bin:$PATH"
gofmt -l . ; go vet ./... && go test ./... && GOOS=windows go vet ./... && GOOS=darwin go vet ./...
git status --short
```
Expected: gofmt 沒有輸出；全部測試通過；`git status` 只列出 README 兩個檔案與刪除的 Python 檔案。

- [ ] **Step 5: Commit**

```bash
git add -A README.md README.zh-TW.md
git commit -F - <<'EOF'
docs: 英文 README 與繁中版，移除 Python 版本

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_013esmpw9AMufcwkocoj5v2w
EOF
```

- [ ] **Step 6: 交給使用者（不要自行執行）**

以下動作會改到使用者真實的環境或對外發佈，必須先詢問使用者：
1. 在這台 Linux 上，用 `CLAUDE_ACCOUNTS_URL=file://…` 安裝並遷移真實的 `~/.claude-zhon`（會改 `~/.bashrc`、`settings.json`）。
2. `git push`，以及推送 `v0.1.0` tag 觸發 Release。
3. 請使用者在 Windows 上實測 `install.ps1`、`claude-b`（Ctrl+C、退出碼）與 junction。
