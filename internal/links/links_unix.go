//go:build !windows

package links

import (
	"errors"
	"io/fs"
	"os"
)

func Dir(link, target string) error { return os.Symlink(target, link) }

func Exe(exe, link string) error {
	if IsLink(link) && Points(link, exe) {
		return nil
	}
	if err := RemoveExe(link); err != nil {
		return err
	}
	return os.Symlink(exe, link)
}

func RemoveExe(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
