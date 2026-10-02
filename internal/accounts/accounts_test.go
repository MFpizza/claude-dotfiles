package accounts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

func setup(t *testing.T) (cfg *config.Config, main, exe string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	main = filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	exe = filepath.Join(home, "bin", "claude-accounts")
	write(t, exe, "binary")
	return config.New(main), main, exe
}

func TestAddAssignsLettersAndCommands(t *testing.T) {
	cfg, main, exe := setup(t)
	b, conflicts, err := Add(cfg, main, exe, false, "")
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("%v %v", conflicts, err)
	}
	if b.Name != "b" || b.Type != config.TypeSubscription || !b.StatusLine || b.Dir != "~/.claude-b" {
		t.Fatalf("got %+v", b)
	}
	if !links.Points(CommandPath(filepath.Dir(exe), "b"), exe) {
		t.Error("claude-b command is missing")
	}
	if !links.Points(filepath.Join(main+"-b", "projects"), filepath.Join(main, "projects")) {
		t.Error("shared folders are not linked")
	}
	c, _, err := Add(cfg, main, exe, true, "")
	if err != nil || c.Name != "c" || c.Type != config.TypeAPI {
		t.Fatalf("got %+v %v", c, err)
	}
}

func TestAddCustomDir(t *testing.T) {
	cfg, main, _ := setup(t)
	dir := filepath.Join(t.TempDir(), "work")
	acc, _, err := Add(cfg, main, "", false, dir)
	if err != nil || !config.SamePath(acc.Path(), dir) {
		t.Fatalf("got %+v %v", acc, err)
	}
}

func TestRemoveKeepsOrDeletesDir(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	Add(cfg, main, exe, false, "")
	if err := Remove(cfg, exe, "b", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(main + "-b"); err != nil {
		t.Error("b's dir should be kept")
	}
	if _, err := os.Lstat(CommandPath(filepath.Dir(exe), "b")); !os.IsNotExist(err) {
		t.Error("claude-b command should be gone")
	}
	if err := Remove(cfg, exe, "c", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(main + "-c"); !os.IsNotExist(err) {
		t.Error("c's dir should be deleted")
	}
	if _, err := os.Stat(filepath.Join(main, "projects")); err != nil {
		t.Error("main account data was touched")
	}
	if got := cfg.Names(); len(got) != 1 {
		t.Fatalf("accounts left: %v", got)
	}
}

func TestRemoveRefusals(t *testing.T) {
	cfg, _, exe := setup(t)
	if err := Remove(cfg, exe, "a", false); !errors.Is(err, ErrMain) {
		t.Errorf("main: %v", err)
	}
	if err := Remove(cfg, exe, "q", false); !errors.Is(err, ErrNoAccount) {
		t.Errorf("unknown: %v", err)
	}
}

// Review Focus 4: a dir deleted by hand still lets the account be removed.
func TestRemoveMissingDir(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	if err := os.RemoveAll(main + "-b"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(cfg, exe, "b", true); err != nil {
		t.Fatal(err)
	}
	if cfg.Find("b") != nil {
		t.Error("b is still listed")
	}
}

func TestSyncCommandsRecreatesMissing(t *testing.T) {
	cfg, main, exe := setup(t)
	Add(cfg, main, exe, false, "")
	os.Remove(CommandPath(filepath.Dir(exe), "b"))
	if err := SyncCommands(cfg, exe); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a", "b"} {
		if !links.Points(CommandPath(filepath.Dir(exe), n), exe) {
			t.Errorf("claude-%s missing", n)
		}
	}
}

func TestAddRefusesAnotherAccountsDir(t *testing.T) {
	cfg, main, exe := setup(t)
	if _, _, err := Add(cfg, main, exe, false, ""); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{main, "~/.claude", main + "-b"} {
		if _, _, err := Add(cfg, main, exe, false, dir); !errors.Is(err, ErrDirInUse) {
			t.Errorf("--dir %s: got %v", dir, err)
		}
	}
	if len(cfg.Accounts) != 2 {
		t.Fatalf("accounts %v", cfg.Names())
	}
}
