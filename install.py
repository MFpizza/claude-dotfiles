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
SHARED_DIRS = ["projects", "skills", "agents", "commands", "plugins", "file-history"]
BEGIN, END = "# >>> claude-dotfiles >>>", "# <<< claude-dotfiles <<<"


def step(msg):
    print(f"  ✓ {msg}")


def link_shared_dirs():
    os.makedirs(CLAUDE_B, exist_ok=True)
    for name in SHARED_DIRS:
        target = os.path.join(CLAUDE_A, name)
        link = os.path.join(CLAUDE_B, name)
        os.makedirs(target, exist_ok=True)
        if os.path.lexists(link):
            continue
        if sys.platform == "win32":
            subprocess.run(["cmd", "/c", "mklink", "/J", link, target],
                           check=True, stdout=subprocess.DEVNULL)
        else:
            os.symlink(target, link)
    step(f"~/.claude-b 共用 {', '.join(SHARED_DIRS)}")


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


def main():
    if sys.version_info < (3, 7):
        sys.exit("需要 Python 3.7 以上")
    sys.stdout.reconfigure(encoding="utf-8")
    print("安裝 claude-dotfiles …")
    link_shared_dirs()
    configure_status_line()
    install_profile_block()
    print("\n完成。開新的 PowerShell 視窗，執行 claude-a 或 claude-b；"
          "第一次使用某個帳號時在 Claude 裡輸入 /login。")


if __name__ == "__main__":
    main()
