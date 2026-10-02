//go:build windows

package links

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
)

// Dir makes a junction, which unlike a symlink needs no admin rights or developer mode.
func Dir(link, target string) error {
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J %s: %v: %s", link, err, out)
	}
	return nil
}

// Exe hard-links exe as link. After an update the old links still point at the old
// file, so anything that isn't the current exe is replaced.
func Exe(exe, link string) error {
	if Points(link, exe) {
		return nil
	}
	if err := RemoveExe(link); err != nil {
		return err
	}
	if err := os.Link(exe, link); err == nil {
		return nil
	}
	return copyFile(exe, link)
}

// RemoveExe deletes path; a running claude-x.exe can't be deleted but can be renamed.
func RemoveExe(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	old := path + ".old"
	os.Remove(old)
	return os.Rename(path, old)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
