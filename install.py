"""Install the dual-account setup and usage status line on this computer.

Safe to run again (e.g. after moving this folder): every step is idempotent.
    python install.py
"""
import json
import os
import shutil
import subprocess
import sys

REPO = os.path.dirname(os.path.abspath(__file__))
HOME = os.path.expanduser("~")
CLAUDE_A = os.path.join(HOME, ".claude")
CLAUDE_B = os.path.join(HOME, ".claude-b")
# sessions/ is the live-session registry that ListAgents / SendMessage read,
# so sharing it lets A and B sessions find and message each other.
SHARED_DIRS = ["projects", "skills", "agents", "commands", "plugins", "file-history",
               "sessions"]
BEGIN, END = "# >>> claude-dotfiles >>>", "# <<< claude-dotfiles <<<"


def step(msg):
    print(f"  ✓ {msg}")


def merge_into(src, dst):
    """Move src's entries into dst (never overwriting) and remove src if emptied."""
    for entry in os.listdir(src):
        if not os.path.lexists(os.path.join(dst, entry)):
            shutil.move(os.path.join(src, entry), os.path.join(dst, entry))
    if os.listdir(src):
        return False
    os.rmdir(src)
    return True


def link_shared_dirs():
    os.makedirs(CLAUDE_B, exist_ok=True)
    shared = []
    for name in SHARED_DIRS:
        target = os.path.join(CLAUDE_A, name)
        link = os.path.join(CLAUDE_B, name)
        os.makedirs(target, exist_ok=True)
        if os.path.lexists(link) and not os.path.samefile(link, target):
            # A real folder B created before this dir was shared: fold it into A's.
            if not merge_into(link, target):
                print(f"  ! ~/.claude-b/{name} 有與 ~/.claude/{name} 同名的項目，"
                      f"未建立連結；請手動處理後重跑")
                continue
        if not os.path.lexists(link):
            if sys.platform == "win32":
                subprocess.run(["cmd", "/c", "mklink", "/J", link, target],
                               check=True, stdout=subprocess.DEVNULL)
            else:
                os.symlink(target, link)
        shared.append(name)
    step(f"~/.claude-b 共用 {', '.join(shared)}")


def configure_status_line():
    path = os.path.join(CLAUDE_A, "settings.json")
    settings = {}
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            settings = json.load(f)
        shutil.copy2(path, path + ".bak")
    python = sys.executable.replace("\\", "/")
    script = os.path.join(REPO, "usage_statusline.py").replace("\\", "/")
    settings["statusLine"] = {
        "type": "command",
        "command": f'"{python}" "{script}"',
        "padding": 0,
    }
    with open(path, "w", encoding="utf-8") as f:
        json.dump(settings, f, ensure_ascii=False, indent=2)
    step("settings.json 已設定 statusLine（原檔備份為 settings.json.bak）")


def profile_paths():
    """$PROFILE of Windows PowerShell and PowerShell 7, whichever is installed."""
    paths = []
    for shell in ("powershell", "pwsh"):
        if not shutil.which(shell):
            continue
        out = subprocess.run([shell, "-NoProfile", "-Command", "$PROFILE"],
                             capture_output=True, text=True)
        if out.returncode == 0 and out.stdout.strip():
            paths.append(out.stdout.strip())
    return paths


def install_profile_block():
    ps1 = os.path.join(REPO, "claude-accounts.ps1")
    block = f"{BEGIN}\n. '{ps1}'\n{END}\n"
    for path in profile_paths():
        text = ""
        if os.path.exists(path):
            with open(path, encoding="utf-8-sig") as f:
                text = f.read()
        if BEGIN in text and END in text:
            head, rest = text.split(BEGIN, 1)
            text = head + block + rest.split(END, 1)[1].lstrip("\n")
        else:
            text = text.rstrip("\n") + ("\n\n" if text.strip() else "") + block
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            f.write(text)
        step(f"PowerShell profile：{path}")


def install_cmd_shims():
    """cmd.exe can't see PowerShell functions; put forwarders in ~/.local/bin (on PATH)."""
    if sys.platform != "win32":
        return
    bin_dir = os.path.join(HOME, ".local", "bin")
    os.makedirs(bin_dir, exist_ok=True)
    for name in ("claude-a.cmd", "claude-b.cmd"):
        target = os.path.join(REPO, "bin", name)
        with open(os.path.join(bin_dir, name), "w", newline="\r\n") as f:
            f.write(f'@call "{target}" %*\n')
    step(f"cmd.exe 指令：{bin_dir}\\claude-a.cmd、claude-b.cmd")


def main():
    if sys.version_info < (3, 8):  # os.stat follows Windows junctions from 3.8
        sys.exit("需要 Python 3.8 以上")
    sys.stdout.reconfigure(encoding="utf-8")
    print("安裝 claude-dotfiles …")
    link_shared_dirs()
    configure_status_line()
    install_profile_block()
    install_cmd_shims()
    print("\n完成。開新的 PowerShell 或 cmd 視窗，執行 claude-a 或 claude-b；"
          "第一次使用某個帳號時在 Claude 裡輸入 /login。")


if __name__ == "__main__":
    main()
