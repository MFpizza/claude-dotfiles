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
		"old": {Day: today, Cost: 1.00}, // written before accounts were tracked: c
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
