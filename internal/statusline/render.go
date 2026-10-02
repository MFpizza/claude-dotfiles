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

const (
	barWidth  = 10
	fiveHours = 5 * time.Hour
	week      = 7 * 24 * time.Hour
)

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

// timeLeftIcon is a moon showing the time left before a window resets: full when it
// has just reset, new when it is about to.
func timeLeftIcon(resetsAt string, period time.Duration, now time.Time) string {
	t, err := time.Parse(time.RFC3339, resetsAt)
	if err != nil {
		return ""
	}
	left := t.Sub(now)
	if left <= 0 {
		return ""
	}
	quarters := int(math.Round(float64(left) / float64(period) * 4))
	return []string{"🌑", "🌘", "🌗", "🌖", "🌕"}[max(0, min(4, quarters))]
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

func fmtWindow(label string, period time.Duration, w *Window, stale bool, now time.Time) string {
	head := style.Muted + label + style.Reset + " "
	pct, resets, ok := windowState(w, stale, now)
	if !ok {
		return head + style.Track + strings.Repeat("▱", barWidth) + style.Reset + " " + style.Muted + "   —" + style.Reset
	}
	s := head + bar(pct) + " " + levelColor(pct) + style.Bold + fmt.Sprintf("%3.0f%%", pct) + style.Reset
	if r := fmtRemaining(resets, now); r != "" {
		return s + " " + style.Muted + timeLeftIcon(resets, period, now) + " " + fmt.Sprintf("%-6s", r) + style.Reset
	}
	return s + strings.Repeat(" ", 9)
}

func compactWindow(label string, period time.Duration, w *Window, stale, withBar bool, now time.Time) string {
	pct, resets, ok := windowState(w, stale, now)
	if !ok {
		return style.Muted + label + " —" + style.Reset
	}
	s := style.Muted + label + style.Reset + " "
	if withBar {
		s += bar(pct) + " "
	}
	s += levelColor(pct) + style.Bold + fmt.Sprintf("%.0f%%", pct) + style.Reset
	if icon := timeLeftIcon(resets, period, now); icon != "" {
		s += " " + style.Muted + icon + style.Reset
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
	full := fmtTag(acc.Name, e.Plan, active) + style.Sep + fmtWindow("5h", fiveHours, e.FiveHour, stale, now) +
		style.Sep + fmtWindow(lang.T("wk"), week, e.SevenDay, stale, now)
	compact := compactTag(acc.Name, e.Plan, active) + " " + compactWindow("5h", fiveHours, e.FiveHour, stale, true, now) +
		" " + compactWindow(lang.T("wk"), week, e.SevenDay, stale, false, now)
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
