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
