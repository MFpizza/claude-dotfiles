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
