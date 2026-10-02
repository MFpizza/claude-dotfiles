package migrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
)

func TestDetectFindsLinkedAccounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	main := filepath.Join(home, ".claude")
	for _, s := range []string{"-b", "-c"} {
		if _, err := accounts.Share(main, main+s); err != nil {
			t.Fatal(err)
		}
	}
	os.MkdirAll(main+"-x", 0o755)    // a real dir that isn't an account
	accounts.Share(main, main+"-bb") // not a single letter
	cfg := Detect(main)
	if got := cfg.Names(); len(got) != 3 || got[1] != "b" || got[2] != "c" {
		t.Fatalf("got %v", got)
	}
	if cfg.Find("b").Type != config.TypeSubscription || cfg.Find("c").Type != config.TypeAPI {
		t.Fatalf("types: %+v", cfg.Accounts)
	}
	if cfg.Find("b").Dir != "~/.claude-b" {
		t.Fatalf("dir: %q", cfg.Find("b").Dir)
	}
}

func TestStripBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".bashrc")
	orig := "export A=1\n\n" + Begin + "\nCLAUDE_DOTFILES_BASE='/x'\n. '/x/claude-accounts.sh'\n" + End + "\nexport B=2\n"
	os.WriteFile(path, []byte(orig), 0o644)
	ok, err := StripBlock(path)
	if err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "export A=1\n\nexport B=2\n" {
		t.Fatalf("got %q", data)
	}
	if bak, _ := os.ReadFile(path + ".bak"); string(bak) != orig {
		t.Fatal("backup missing")
	}
	if ok, err := StripBlock(path); ok || err != nil {
		t.Fatalf("second run: %v %v", ok, err)
	}
	if ok, err := StripBlock(filepath.Join(t.TempDir(), "missing")); ok || err != nil {
		t.Fatalf("missing file: %v %v", ok, err)
	}
}

func TestLegacyLayout(t *testing.T) {
	dir := t.TempDir()
	if got := LegacyLayout(dir); got != "" {
		t.Fatalf("none: %q", got)
	}
	os.WriteFile(filepath.Join(dir, "statusline.json"), []byte(`{"layout":"compact"}`), 0o644)
	if got := LegacyLayout(dir); got != "compact" {
		t.Fatalf("got %q", got)
	}
	RemoveLegacyLayout(dir)
	if _, err := os.Stat(filepath.Join(dir, "statusline.json")); !os.IsNotExist(err) {
		t.Fatal("not removed")
	}
}

func TestOldShims(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	os.MkdirAll(bin, 0o755)
	os.WriteFile(filepath.Join(bin, "claude-b.cmd"), []byte("@call x"), 0o644)
	if got := OldShims(home); len(got) != 1 || filepath.Base(got[0]) != "claude-b.cmd" {
		t.Fatalf("got %v", got)
	}
}
