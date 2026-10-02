package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
)

func fakeLook(found map[string]string) func(string) (string, error) {
	return func(name string) (string, error) {
		if p, ok := found[name]; ok {
			return p, nil
		}
		return "", errors.New("not found")
	}
}

func setup(t *testing.T) (*config.Config, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("CLAUDE_BIN", "")
	main := filepath.Join(home, ".claude")
	os.MkdirAll(main, 0o755)
	os.MkdirAll(main+"-b", 0o755)
	os.WriteFile(filepath.Join(main, "settings.json"), []byte(`{"theme":"dark"}`), 0o644)
	cfg := config.New(main)
	cfg.Add(config.Account{Name: "b", Dir: "~/.claude-b", Type: config.TypeSubscription, StatusLine: true})
	return cfg, main
}

func envValue(env []string, key string) (string, bool) {
	for _, kv := range env {
		if strings.HasPrefix(kv, key+"=") {
			return kv[len(key)+1:], true
		}
	}
	return "", false
}

func TestBuildSyncsSettingsAndSetsDir(t *testing.T) {
	cfg, main := setup(t)
	p, err := Build(cfg, main, "b", []string{"--continue"}, fakeLook(map[string]string{"claude": "/x/claude"}))
	if err != nil {
		t.Fatal(err)
	}
	if p.Path != "/x/claude" || strings.Join(p.Argv, " ") != "/x/claude --continue" {
		t.Fatalf("got %+v", p)
	}
	if v, _ := envValue(p.Env, "CLAUDE_CONFIG_DIR"); !config.SamePath(v, main+"-b") {
		t.Fatalf("CLAUDE_CONFIG_DIR=%q", v)
	}
	if data, _ := os.ReadFile(filepath.Join(main+"-b", "settings.json")); string(data) != `{"theme":"dark"}` {
		t.Fatalf("settings not synced: %s", data)
	}
}

func TestBuildDefaultMainDirUnsetsVar(t *testing.T) {
	cfg, main := setup(t)
	t.Setenv("CLAUDE_CONFIG_DIR", "/somewhere/else")
	p, err := Build(cfg, main, "a", nil, fakeLook(map[string]string{"claude": "/x/claude"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envValue(p.Env, "CLAUDE_CONFIG_DIR"); ok {
		t.Fatal("~/.claude must run with CLAUDE_CONFIG_DIR unset")
	}
}

func TestBuildAccountSettings(t *testing.T) {
	cfg, main := setup(t)
	own := filepath.Join(main+"-b", "account-settings.json")
	os.WriteFile(own, []byte(`{}`), 0o600)
	p, _ := Build(cfg, main, "b", []string{"-p", "hi"}, fakeLook(map[string]string{"claude": "/x/claude"}))
	if len(p.Argv) != 5 || p.Argv[1] != "--settings" || !config.SamePath(p.Argv[2], own) || p.Argv[3] != "-p" {
		t.Fatalf("got %v", p.Argv)
	}
}

func TestBuildFindsClaude(t *testing.T) {
	cfg, main := setup(t)
	t.Setenv("CLAUDE_BIN", "/opt/claude")
	if p, _ := Build(cfg, main, "b", nil, fakeLook(map[string]string{"claude": "/x/claude"})); p.Path != "/opt/claude" {
		t.Errorf("CLAUDE_BIN should win: %v", p.Path)
	}
	t.Setenv("CLAUDE_BIN", "")
	p, _ := Build(cfg, main, "b", []string{"-c"}, fakeLook(map[string]string{"npx": "/n/npx"}))
	if strings.Join(p.Argv, " ") != "/n/npx --no -- claude -c" {
		t.Errorf("npx fallback: %v", p.Argv)
	}
	if _, err := Build(cfg, main, "b", nil, fakeLook(nil)); !errors.Is(err, ErrNoClaude) {
		t.Errorf("nothing found: %v", err)
	}
}

func TestBuildUnknownAccount(t *testing.T) {
	cfg, main := setup(t)
	if _, err := Build(cfg, main, "q", nil, fakeLook(nil)); !errors.Is(err, accounts.ErrNoAccount) {
		t.Fatalf("got %v", err)
	}
}

// Review Focus 4: a folder deleted by hand gives a clear error.
func TestBuildDirMissing(t *testing.T) {
	cfg, main := setup(t)
	os.RemoveAll(main + "-b")
	_, err := Build(cfg, main, "b", nil, fakeLook(map[string]string{"claude": "/x/claude"}))
	var missing *DirMissingError
	if !errors.As(err, &missing) || missing.Name != "b" {
		t.Fatalf("got %v", err)
	}
}
