package accounts

import (
	"errors"
	"path/filepath"
	"runtime"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

var (
	ErrMain      = errors.New("the main account can't be removed")
	ErrNoAccount = errors.New("no such account")
)

// CommandPath is where the claude-<name> command for an account lives: next to the
// executable, which is on PATH since it runs.
func CommandPath(dir, name string) string {
	file := "claude-" + name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	return filepath.Join(dir, file)
}

// SyncCommands makes sure every account's command points at the current executable.
func SyncCommands(cfg *config.Config, exe string) error {
	if exe == "" {
		return nil
	}
	for _, acc := range cfg.Accounts {
		if err := links.Exe(exe, CommandPath(filepath.Dir(exe), acc.Name)); err != nil {
			return err
		}
	}
	return nil
}

// Add creates the next lettered account. The caller saves cfg.
func Add(cfg *config.Config, mainDir, exe string, api bool, dir string) (config.Account, []string, error) {
	name, err := cfg.NextLetter()
	if err != nil {
		return config.Account{}, nil, err
	}
	if dir == "" {
		dir = mainDir + "-" + name
	}
	if dir, err = filepath.Abs(config.ExpandHome(dir)); err != nil {
		return config.Account{}, nil, err
	}
	conflicts, err := Share(mainDir, dir)
	if err != nil {
		return config.Account{}, conflicts, err
	}
	typ := config.TypeSubscription
	if api {
		typ = config.TypeAPI
	}
	acc := config.Account{Name: name, Dir: config.CollapseHome(dir), Type: typ, StatusLine: true}
	cfg.Add(acc)
	if exe != "" {
		if err := links.Exe(exe, CommandPath(filepath.Dir(exe), name)); err != nil {
			return acc, conflicts, err
		}
	}
	return acc, conflicts, nil
}

// Remove drops an account and its command; deleteDir also deletes its folder.
// The caller saves cfg.
func Remove(cfg *config.Config, exe, name string, deleteDir bool) error {
	if name == config.MainName {
		return ErrMain
	}
	acc := cfg.Find(name)
	if acc == nil {
		return ErrNoAccount
	}
	dir := acc.Path()
	if exe != "" {
		if err := links.RemoveExe(CommandPath(filepath.Dir(exe), name)); err != nil {
			return err
		}
	}
	cfg.Delete(name)
	if !deleteDir {
		return nil
	}
	return RemoveDir(dir)
}
