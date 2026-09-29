"""Claude Code status line: show 5-hour / weekly usage for every account.

Each subscription account's usage comes from its own OAuth token (api/oauth/usage).
Results are cached in usage-cache.json; on failure (network, expired token,
lapsed subscription) the last successful value is shown, marked with its age.
The API-billed account has no quota, so its spend is logged in api-cost.json.

    python usage_statusline.py --layout compact|full    switch layout
    python usage_statusline.py --base DIR ...           account A is DIR, not ~/.claude
"""
import json
import os
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone

HOME = os.path.expanduser("~")
# Account A's config dir; B and C sit next to it with a -b / -c suffix.
# Default ~/.claude; install.py passes --base when A lives elsewhere (Linux).
_argv = sys.argv[1:]
BASE = os.path.join(HOME, ".claude")
if _argv[:1] == ["--base"] and len(_argv) > 1:
    BASE, _argv = os.path.expanduser(_argv[1]), _argv[2:]
ACCOUNTS = [  # (name, config dir, "sub" = Pro/Max quota | "api" = pay-as-you-go)
    ("A", BASE, "sub"),
    ("B", BASE + "-b", "sub"),
    ("C", BASE + "-c", "api"),
]
CACHE_PATH = os.path.join(BASE, "usage-cache.json")
PET_PATH = os.path.join(BASE, "statusline-pet.json")
CONFIG_PATH = os.path.join(BASE, "statusline.json")
LEDGER_PATH = os.path.join(BASE, "api-cost.json")
LEDGER_DAYS = 62  # keep enough history for "this month"
LAYOUTS = ("full", "compact")
CACHE_TTL = 120  # seconds between fetches per account
USAGE_URL = "https://api.anthropic.com/api/oauth/usage"
TOKEN_URL = "https://platform.claude.com/v1/oauth/token"
CLIENT_ID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
TIMEOUT = 4


def rgb(hexcode):
    r, g, b = (int(hexcode[i:i + 2], 16) for i in (1, 3, 5))
    return f"\033[38;2;{r};{g};{b}m"


# Colour-blind friendly scale (theme is dark-daltonized): blue -> amber -> orange.
LOW, MID, HIGH = rgb("#5fafff"), rgb("#e5c07b"), rgb("#ff8c42")
ACCENT = rgb("#d97757")  # Claude orange, marks the running account
TEXT = rgb("#c8ccd4")
MUTED = rgb("#7f8794")
TRACK = rgb("#3e4451")
BOLD, ITALIC, RESET = "\033[1m", "\033[3m", "\033[0m"
BAR_WIDTH = 10
SEP = f" {TRACK}│{RESET} "


def load_json(path, default):
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    except (OSError, ValueError):
        return default


def save_json(path, data):
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f)
    os.replace(tmp, path)


def post_json(url, body):
    req = urllib.request.Request(
        url, data=json.dumps(body).encode(), method="POST",
        headers={"Content-Type": "application/json"})
    return json.loads(urllib.request.urlopen(req, timeout=TIMEOUT).read())


def refresh_token(cred_path, creds):
    """Refresh an expired token. Only used for the account not running here."""
    oauth = creds["claudeAiOauth"]
    data = post_json(TOKEN_URL, {
        "grant_type": "refresh_token",
        "refresh_token": oauth["refreshToken"],
        "client_id": CLIENT_ID,
        "scope": " ".join(oauth.get("scopes") or []),
    })
    oauth["accessToken"] = data["access_token"]
    if data.get("refresh_token"):
        oauth["refreshToken"] = data["refresh_token"]
    oauth["expiresAt"] = int((time.time() + data.get("expires_in", 3600)) * 1000)
    save_json(cred_path, creds)
    return oauth["accessToken"]


def fetch_usage(config_dir, is_active):
    cred_path = os.path.join(config_dir, ".credentials.json")
    creds = load_json(cred_path, None)
    if not creds or "claudeAiOauth" not in creds:
        raise RuntimeError("未登入")
    oauth = creds["claudeAiOauth"]
    token = oauth["accessToken"]
    if oauth.get("expiresAt", 0) / 1000 < time.time() + 60:
        if is_active:
            raise RuntimeError("token 過期")  # the running Claude refreshes its own
        token = refresh_token(cred_path, creds)
    req = urllib.request.Request(USAGE_URL, headers={
        "Authorization": "Bearer " + token,
        "anthropic-beta": "oauth-2025-04-20",
        "User-Agent": "claude-code-usage-statusline",
    })
    data = json.loads(urllib.request.urlopen(req, timeout=TIMEOUT).read())
    return {
        "plan": oauth.get("subscriptionType"),
        "five_hour": data.get("five_hour"),
        "seven_day": data.get("seven_day"),
    }


def parse_time(iso):
    return datetime.fromisoformat(iso.replace("Z", "+00:00"))


def fmt_remaining(iso):
    if not iso:
        return ""
    minutes = int((parse_time(iso) - datetime.now(timezone.utc)).total_seconds() // 60) + 1
    if minutes <= 0:
        return ""
    days, rem = divmod(minutes, 1440)
    hours, mins = divmod(rem, 60)
    if days:
        return f"{days}d{hours:02d}h"
    if hours:
        return f"{hours}h{mins:02d}m"
    return f"{mins}m"


def level_color(pct):
    return LOW if pct < 50 else MID if pct < 80 else HIGH


def bar(pct):
    filled = min(BAR_WIDTH, max(0, round(pct / 100 * BAR_WIDTH)))
    if pct > 0 and filled == 0:
        filled = 1
    return (f"{level_color(pct)}{'▰' * filled}"
            f"{TRACK}{'▱' * (BAR_WIDTH - filled)}{RESET}")


def window_state(win, stale):
    """(percent used or None, resets_at) for one usage window."""
    if not win or win.get("utilization") is None:
        return None, None
    pct, resets_at = win["utilization"], win.get("resets_at")
    # A cached window whose reset time has passed is back to 0%.
    if stale and resets_at and parse_time(resets_at) < datetime.now(timezone.utc):
        pct, resets_at = 0, None
    return pct, resets_at


def fmt_window(label, win, stale):
    head = f"{MUTED}{label}{RESET} "
    pct, resets_at = window_state(win, stale)
    if pct is None:
        return head + f"{TRACK}{'▱' * BAR_WIDTH}{RESET} {MUTED}   —{RESET}"
    remaining = fmt_remaining(resets_at)
    return (head + bar(pct)
            + f" {level_color(pct)}{BOLD}{pct:>3.0f}%{RESET}"
            + (f" {MUTED}↻ {remaining:<6}{RESET}" if remaining else " " * 9))


def fmt_age(seconds):
    if seconds < 3600:
        return f"{int(seconds // 60)} 分鐘前"
    if seconds < 86400:
        return f"{int(seconds // 3600)} 小時前"
    return f"{int(seconds // 86400)} 天前"


def fmt_tag(name, plan, is_active):
    color = f"{TEXT}{BOLD}" if is_active else TEXT
    badge = "API" if plan == "api" else (plan or "").capitalize()
    return f"{dot(is_active)} {color}{name}{RESET} {MUTED}{badge:<4}{RESET}"


# Pet grows with the number of distinct sessions seen (Lv = 1 + sqrt(sessions)).
PET_STAGES = [(10, "🦚"), (7, "🐔"), (5, "🐥"), (3, "🐣"), (1, "🥚")]
PET_SEEN_MAX = 200  # remembered session ids, enough to avoid double counting


def context_pct(session):
    ctx = session.get("context_window") or {}
    if ctx.get("used_percentage") is not None:
        return ctx["used_percentage"]
    usage, size = ctx.get("current_usage") or {}, ctx.get("context_window_size")
    if not size:
        return None
    used = sum(usage.get(k) or 0 for k in (
        "input_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"))
    return used / size * 100


def fmt_pet(session):
    pet = load_json(PET_PATH, {"sessions": 0, "seen": []})
    sid = session.get("session_id")
    if sid and sid not in pet["seen"]:
        pet["sessions"] += 1
        pet["seen"] = (pet["seen"] + [sid])[-PET_SEEN_MAX:]
        try:
            save_json(PET_PATH, pet)
        except OSError:
            pass
    level = 1 + int(pet["sessions"] ** 0.5)
    body = next(e for lv, e in PET_STAGES if level >= lv)
    # Mood follows context usage: fresh -> normal -> tired -> sleepy (time to /compact).
    pct = context_pct(session)
    mood = ("" if pct is None or 30 <= pct < 60 else
            "✨" if pct < 30 else "💦" if pct < 85 else "💤")
    return f"{body}{mood} {MUTED}Lv{level}{RESET}"


def dot(is_active):
    return f"{ACCENT}●{RESET}" if is_active else f"{MUTED}○{RESET}"


def compact_window(label, win, stale):
    pct, resets_at = window_state(win, stale)
    if pct is None:
        return f"{MUTED}{label} —{RESET}"
    text = f"{MUTED}{label}{RESET} {level_color(pct)}{BOLD}{pct:.0f}%{RESET}"
    remaining = fmt_remaining(resets_at) if pct >= 80 else ""
    return text + (f" {MUTED}↻{remaining}{RESET}" if remaining else "")


def subscription_account(name, config_dir, is_active, cache, now):
    """(full line, compact segment) for a Pro/Max account, from api/oauth/usage."""
    entry = cache.get(name)
    error = None
    if not entry or now - entry.get("fetched_at", 0) > CACHE_TTL:
        try:
            usage = fetch_usage(config_dir, is_active)
            entry = dict(usage, fetched_at=now)
            cache[name] = entry
        except urllib.error.HTTPError as e:
            error = f"HTTP {e.code}"
        except Exception as e:  # network, missing creds, bad JSON
            error = str(e) or type(e).__name__
        if error and entry:
            entry["last_error"] = error

    if not entry:
        hint = f"執行 claude-{name.lower()} 後 /login" if error == "未登入" else ""
        full = (fmt_tag(name, None, is_active) + SEP
                + f"{MUTED}{ITALIC}{error or '無資料'}{RESET}"
                + (f" {TRACK}·{RESET} {MUTED}{hint}{RESET}" if hint else ""))
        return full, f"{dot(is_active)} {TEXT}{name}{RESET} {MUTED}{ITALIC}{error or '無資料'}{RESET}"
    stale = error is not None or now - entry["fetched_at"] > CACHE_TTL * 3
    full = (fmt_tag(name, entry.get("plan"), is_active) + SEP
            + fmt_window("5h", entry.get("five_hour"), stale) + SEP
            + fmt_window("週", entry.get("seven_day"), stale))
    if stale:
        full += f"{SEP}{MUTED}{ITALIC}快取 · {fmt_age(now - entry['fetched_at'])}{RESET}"
    compact = (f"{dot(is_active)} {TEXT}{BOLD if is_active else ''}{name}{RESET} "
               + compact_window("5h", entry.get("five_hour"), stale) + " "
               + compact_window("週", entry.get("seven_day"), stale)
               + (f"{MUTED}*{RESET}" if stale else ""))
    return full, compact


def session_cost(session):
    return (session.get("cost") or {}).get("total_cost_usd")


def record_api_cost(session, today):
    """Log this session's spend. Resuming restarts total_cost_usd at 0, so earlier
    runs of the same session are carried in `base`."""
    ledger = load_json(LEDGER_PATH, {})
    sid, cost = session.get("session_id"), session_cost(session)
    if sid and cost is not None:
        entry = ledger.setdefault(sid, {"day": today, "base": 0, "cost": 0})
        if cost < entry["cost"]:
            entry["base"] += entry["cost"]
        entry["cost"] = cost
        cutoff = datetime.fromtimestamp(time.time() - LEDGER_DAYS * 86400).strftime("%Y-%m-%d")
        ledger = {k: v for k, v in ledger.items() if v["day"] >= cutoff}
        try:
            save_json(LEDGER_PATH, ledger)
        except OSError:
            pass
    return ledger


def api_account(name, is_active, session):
    """(full line, compact segment) for an API-billed account, or None if never used.

    The API has no quota to show, so this reports spend recorded by this status line."""
    today = datetime.now().strftime("%Y-%m-%d")
    ledger = record_api_cost(session, today) if is_active else load_json(LEDGER_PATH, {})
    if not ledger and not is_active:
        return None
    spent = lambda keep: sum(e["base"] + e["cost"] for e in ledger.values() if keep(e["day"]))
    day, month = spent(lambda d: d == today), spent(lambda d: d[:7] == today[:7])
    money = lambda label, x: f"{MUTED}{label}{RESET} {TEXT}{BOLD}${x:.2f}{RESET}"
    parts = [money("今日", day), money("本月", month)]
    if is_active and session_cost(session) is not None:
        parts.insert(0, money("本次", session_cost(session)))
    full = fmt_tag(name, "api", is_active) + SEP + SEP.join(parts)
    compact = (f"{dot(is_active)} {TEXT}{BOLD if is_active else ''}{name}{RESET} "
               + money("今日", day))
    return full, compact


def set_layout(layout):
    if layout not in LAYOUTS:
        sys.exit(f"版型只能是：{', '.join(LAYOUTS)}")
    config = load_json(CONFIG_PATH, {})
    config["layout"] = layout
    save_json(CONFIG_PATH, config)
    print(f"狀態列版型：{layout}（下次狀態列更新時生效）")


def main():
    if _argv[:1] == ["--layout"]:
        return set_layout(_argv[1] if len(_argv) > 1 else "")
    try:
        session = json.loads(sys.stdin.read() or "{}")
    except ValueError:
        session = {}
    active_dir = os.path.normcase(os.path.abspath(
        os.environ.get("CLAUDE_CONFIG_DIR") or os.path.join(HOME, ".claude")))
    layout = load_json(CONFIG_PATH, {}).get("layout", "full")

    cache = load_json(CACHE_PATH, {})
    now = time.time()
    rows = []
    for name, config_dir, kind in ACCOUNTS:
        is_active = os.path.normcase(os.path.abspath(config_dir)) == active_dir
        if kind == "api":
            row = api_account(name, is_active, session)
        else:
            row = subscription_account(name, config_dir, is_active, cache, now)
        if row:
            rows.append(row)

    try:
        save_json(CACHE_PATH, cache)
    except OSError:
        pass

    model = (session.get("model") or {}).get("display_name")
    tail = ([f"{MUTED}{model}{RESET}"] if model else []) + [fmt_pet(session)]
    if layout == "compact":
        lines = [SEP.join([compact for _, compact in rows] + tail)]
    else:
        lines = [full for full, _ in rows]
        lines[0] += SEP + SEP.join(tail)
    sys.stdout.buffer.write("\n".join(lines).encode("utf-8"))


if __name__ == "__main__":
    main()
