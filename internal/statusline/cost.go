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
