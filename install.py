"""Install the multi-account setup and usage status line on this computer.

Safe to run again (e.g. after moving this folder): every step is idempotent.
    python install.py              accounts A and B
    python install.py --with-api   also account C (API billing)

Linux only:
    --base DIR    account A's config dir (default $CLAUDE_CONFIG_DIR, else ~/.claude);
                  B and C become DIR-b and DIR-c
    --rc FILE     shell rc file that loads claude-a/b/c (default ~/.bashrc)
    --no-rc       leave rc files alone; print the lines to add instead
"""
import json
import os
import shutil
import subprocess
import sys

REPO = os.path.dirname(os.path.abspath(__file__))
HOME = os.path.expanduser("~")
WINDOWS = sys.platform == "win32"
# sessions/ is the live-session registry that ListAgents / SendMessage read,
# so sharing it lets sessions of every account find and message each other.
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


def link_shared_dirs(base, account_dir):
    os.makedirs(account_dir, exist_ok=True)
    label = "~/" + os.path.basename(account_dir)
    shared = []
    for name in SHARED_DIRS:
        target = os.path.join(base, name)
        link = os.path.join(account_dir, name)
        os.makedirs(target, exist_ok=True)
        if os.path.lexists(link) and not os.path.samefile(link, target):
            # A real folder created before this dir was shared: fold it into A's.
            if not merge_into(link, target):
                print(f"  ! {label}/{name} 有與 ~/{os.path.basename(base)}/{name} 同名的項目，"
                      f"未建立連結；請手動處理後重跑")
                continue
        if not os.path.lexists(link):
            if WINDOWS:
                subprocess.run(["cmd", "/c", "mklink", "/J", link, target],
                               check=True, stdout=subprocess.DEVNULL)
            else:
                os.symlink(target, link)
        shared.append(name)
    step(f"{label} 共用 {', '.join(shared)}")


def configure_status_line(base):
    path = os.path.join(base, "settings.json")
    settings = {}
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            settings = json.load(f)
        shutil.copy2(path, path + ".bak")
    python = sys.executable.replace("\\", "/")
    script = os.path.join(REPO, "usage_statusline.py").replace("\\", "/")
    command = f'"{python}" "{script}"'
    if not WINDOWS:
        command += f' --base "{base}"'
    settings["statusLine"] = {
        "type": "command",
        "command": command,
        "padding": 0,
    }
    with open(path, "w", encoding="utf-8") as f:
        json.dump(settings, f, ensure_ascii=False, indent=2)
    step("settings.json 已設定 statusLine（原檔備份為 settings.json.bak）")


def replace_block(text, block):
    """Put block between the BEGIN/END markers in text, or append it."""
    if BEGIN in text and END in text:
        head, rest = text.split(BEGIN, 1)
        return head + block + rest.split(END, 1)[1].lstrip("\n")
    return text.rstrip("\n") + ("\n\n" if text.strip() else "") + block


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
        text = replace_block(text, block)
        os.makedirs(os.path.dirname(path), exist_ok=True)
        with open(path, "w", encoding="utf-8") as f:
            f.write(text)
        step(f"PowerShell profile：{path}")


def install_rc_block(base, rc):
    """bash/zsh: define claude-a/b/c by sourcing claude-accounts.sh from the rc file."""
    sh = os.path.join(REPO, "claude-accounts.sh")
    block = f"{BEGIN}\nCLAUDE_DOTFILES_BASE='{base}'\n. '{sh}'\n{END}\n"
    if rc is None:
        print("  · 未修改 rc 檔；請把下面幾行加進你的 shell 啟動檔：\n")
        print("    " + block.rstrip("\n").replace("\n", "\n    ") + "\n")
        return
    text = ""
    if os.path.exists(rc):
        with open(rc, encoding="utf-8") as f:
            text = f.read()
    with open(rc, "w", encoding="utf-8") as f:
        f.write(replace_block(text, block))
    step(f"shell 啟動檔：{rc}")


def install_cmd_shims(with_api):
    """cmd.exe can't see PowerShell functions; put forwarders in ~/.local/bin (on PATH)."""
    bin_dir = os.path.join(HOME, ".local", "bin")
    os.makedirs(bin_dir, exist_ok=True)
    names = ("claude-a.cmd", "claude-b.cmd") + (("claude-c.cmd",) if with_api else ())
    for name in names:
        target = os.path.join(REPO, "bin", name)
        with open(os.path.join(bin_dir, name), "w", newline="\r\n") as f:
            f.write(f'@call "{target}" %*\n')
    step(f"cmd.exe 指令：{bin_dir}\\" + "、".join(names))


def main():
    if sys.version_info < (3, 8):  # os.stat follows Windows junctions from 3.8
        sys.exit("需要 Python 3.8 以上")
    sys.stdout.reconfigure(encoding="utf-8")
    args = sys.argv[1:]
    with_api = "--with-api" in args  # account C is opt-in per computer

    def option(name, default):
        if name not in args:
            return default
        i = args.index(name)
        if i + 1 >= len(args):
            sys.exit(f"{name} 後面要接路徑")
        return os.path.abspath(os.path.expanduser(args[i + 1]))

    if WINDOWS:
        base = os.path.join(HOME, ".claude")
    else:
        base = option("--base", os.path.abspath(
            os.environ.get("CLAUDE_CONFIG_DIR") or os.path.join(HOME, ".claude")))
        base = base.rstrip("/")
        rc = None if "--no-rc" in args else option("--rc", os.path.join(HOME, ".bashrc"))

    print(f"安裝 claude-dotfiles …（帳號 A：{base}）")
    link_shared_dirs(base, base + "-b")
    if with_api:
        link_shared_dirs(base, base + "-c")
    configure_status_line(base)
    if WINDOWS:
        install_profile_block()
        install_cmd_shims(with_api)
    else:
        install_rc_block(base, rc)
    commands = "claude-a、claude-b" + ("、claude-c（API 計費）" if with_api else "")
    shell = "PowerShell 或 cmd 視窗" if WINDOWS else "終端機（或 source 啟動檔）"
    print(f"\n完成。開新的{shell}，執行 {commands}；"
          "第一次使用某個帳號時在 Claude 裡輸入 /login。")


if __name__ == "__main__":
    main()
