// Package accounts adds and removes accounts: their folders, the links to the main
// account's shared data and their claude-<letter> commands.
package accounts

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/MFpizza/claude-dotfiles/internal/links"
)

// SharedDirs live in the main account; every other account links to them.
// sessions/ is the live-session registry ListAgents and SendMessage read, so sharing
// it lets sessions of every account find and message each other.
var SharedDirs = []string{"projects", "skills", "agents", "commands", "plugins", "file-history", "sessions"}

// Share links accDir's shared folders to mainDir's. A real folder already there is
// folded into the main account's first; names that clash are left alone and returned.
func Share(mainDir, accDir string) ([]string, error) {
	if err := os.MkdirAll(accDir, 0o755); err != nil {
		return nil, err
	}
	var conflicts []string
	for _, name := range SharedDirs {
		target, link := filepath.Join(mainDir, name), filepath.Join(accDir, name)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return conflicts, err
		}
		if _, err := os.Lstat(link); err == nil {
			if links.Points(link, target) {
				continue
			}
			if links.IsLink(link) {
				if err := os.Remove(link); err != nil {
					return conflicts, err
				}
			} else if merged, err := mergeInto(link, target); err != nil {
				return conflicts, err
			} else if !merged {
				conflicts = append(conflicts, name)
				continue
			}
		}
		if err := links.Dir(link, target); err != nil {
			return conflicts, err
		}
	}
	return conflicts, nil
}

// mergeInto moves src's entries into dst without overwriting, and removes src if that
// emptied it.
func mergeInto(src, dst string) (bool, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		to := filepath.Join(dst, e.Name())
		if _, err := os.Lstat(to); errors.Is(err, fs.ErrNotExist) {
			if err := os.Rename(filepath.Join(src, e.Name()), to); err != nil {
				return false, err
			}
		}
	}
	if left, err := os.ReadDir(src); err != nil || len(left) > 0 {
		return false, err
	}
	return true, os.Remove(src)
}

// RemoveDir deletes an account dir. Its links are removed first so nothing under
// them, which is the main account's data, is ever touched.
func RemoveDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		if p := filepath.Join(dir, e.Name()); links.IsLink(p) {
			if err := os.Remove(p); err != nil {
				return err
			}
		}
	}
	return os.RemoveAll(dir)
}
