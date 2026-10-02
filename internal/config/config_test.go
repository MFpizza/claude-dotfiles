package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	return home
}

func TestNextLetterFillsGaps(t *testing.T) {
	c := New("/tmp/main")
	for _, n := range []string{"b", "c", "d"} {
		c.Add(Account{Name: n, Dir: "/x-" + n, Type: TypeSubscription})
	}
	c.Delete("c")
	if got, err := c.NextLetter(); err != nil || got != "c" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestNextLetterFull(t *testing.T) {
	c := New("/tmp/main")
	for ch := 'b'; ch <= 'z'; ch++ {
		c.Add(Account{Name: string(ch)})
	}
	if _, err := c.NextLetter(); !errors.Is(err, ErrFull) {
		t.Fatalf("got %v", err)
	}
}

func TestAddKeepsMainFirstAndSorts(t *testing.T) {
	c := New("/tmp/main")
	c.Add(Account{Name: "d"})
	c.Add(Account{Name: "b"})
	if got := c.Names(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "d" {
		t.Fatalf("got %v", got)
	}
}

func TestExpandCollapseHome(t *testing.T) {
	home := setHome(t)
	p := filepath.Join(home, ".claude-b")
	if got := CollapseHome(p); got != "~/.claude-b" {
		t.Fatalf("collapse: %q", got)
	}
	if got := ExpandHome("~/.claude-b"); !SamePath(got, p) {
		t.Fatalf("expand: %q", got)
	}
	other := filepath.Join(t.TempDir(), "x")
	if got := CollapseHome(other); got != other {
		t.Fatalf("outside home: %q", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := setHome(t)
	main := filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	c := New(main)
	c.Layout = LayoutCompact
	c.Add(Account{Name: "c", Dir: "~/.claude-c", Type: TypeAPI, StatusLine: false})
	if err := c.Save(main); err != nil {
		t.Fatal(err)
	}
	got, err := Load(main)
	if err != nil {
		t.Fatal(err)
	}
	if got.Layout != LayoutCompact || got.Lang != "auto" || got.Find("c").Type != TypeAPI || got.Find("a").Dir != "~/.claude" {
		t.Fatalf("got %+v", got)
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
}

// Review Focus 2: a broken file is an error, never silently replaced.
func TestLoadRejectsBadFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, FileName), []byte("{oops"), 0o644)
	if _, err := Load(dir); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v", err)
	}
	os.WriteFile(filepath.Join(dir, FileName), []byte(`{"accounts":[{"name":"b"}]}`), 0o644)
	if _, err := Load(dir); err == nil {
		t.Fatal("a config whose first account isn't a must be rejected")
	}
}

// Review Focus 1: inside account b, the main account is found through b's links.
func TestResolveMainDirFromLinkedAccount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a symlink; junctions are covered by the accounts tests")
	}
	home := setHome(t)
	main := filepath.Join(home, ".claude")
	b := main + "-b"
	os.MkdirAll(filepath.Join(main, "projects"), 0o755)
	os.MkdirAll(b, 0o755)
	os.WriteFile(filepath.Join(main, FileName), []byte(`{}`), 0o644)
	if err := os.Symlink(filepath.Join(main, "projects"), filepath.Join(b, "projects")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", b)
	if got := ResolveMainDir(""); !SamePath(got, main) {
		t.Fatalf("got %q, want %q", got, main)
	}
	if got := ResolveMainDir("~/elsewhere"); !SamePath(got, filepath.Join(home, "elsewhere")) {
		t.Fatalf("override: %q", got)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	if got := ResolveMainDir(""); !SamePath(got, main) {
		t.Fatalf("default: %q", got)
	}
}
