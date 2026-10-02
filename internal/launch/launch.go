// Package launch starts Claude Code as one of the accounts.
package launch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

var ErrNoClaude = errors.New("claude not found")

type DirMissingError struct{ Name, Dir string }

func (e *DirMissingError) Error() string {
	return fmt.Sprintf("account %s: %s is missing", e.Name, e.Dir)
}

// Plan is the program, argv and environment that Exec runs.
type Plan struct {
	Path string
	Argv []string
	Env  []string
}

func Build(cfg *config.Config, mainDir, name string, args []string, lookPath func(string) (string, error)) (*Plan, error) {
	acc := cfg.Find(name)
	if acc == nil {
		return nil, accounts.ErrNoAccount
	}
	dir := acc.Path()
	if _, err := os.Stat(dir); err != nil {
		return nil, &DirMissingError{Name: name, Dir: acc.Dir}
	}
	if !config.SamePath(dir, mainDir) {
		// Settings flow one way, from the main account to the others.
		if data, err := os.ReadFile(filepath.Join(mainDir, "settings.json")); err == nil {
			fsutil.WriteFileAtomic(filepath.Join(dir, "settings.json"), data, 0o644)
		}
	}
	if own := filepath.Join(dir, "account-settings.json"); fileExists(own) {
		args = append([]string{"--settings", own}, args...)
	}
	env := withoutVar(os.Environ(), "CLAUDE_CONFIG_DIR")
	// For ~/.claude leave the variable unset, so claude-a and plain claude share
	// ~/.claude.json (with it set, Claude reads <dir>/.claude.json instead).
	if !config.SamePath(dir, filepath.Join(config.Home(), ".claude")) {
		env = append(env, "CLAUDE_CONFIG_DIR="+dir)
	}
	if bin := os.Getenv("CLAUDE_BIN"); bin != "" {
		return &Plan{Path: bin, Argv: append([]string{bin}, args...), Env: env}, nil
	}
	if p, err := lookPath("claude"); err == nil {
		return &Plan{Path: p, Argv: append([]string{p}, args...), Env: env}, nil
	}
	// Claude installed per project and started with `npx claude`; --no never downloads.
	if p, err := lookPath("npx"); err == nil {
		return &Plan{Path: p, Argv: append([]string{p, "--no", "--", "claude"}, args...), Env: env}, nil
	}
	return nil, ErrNoClaude
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func withoutVar(env []string, name string) []string {
	var out []string
	for _, kv := range env {
		key, _, _ := strings.Cut(kv, "=")
		if key == name || (runtime.GOOS == "windows" && strings.EqualFold(key, name)) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
