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
