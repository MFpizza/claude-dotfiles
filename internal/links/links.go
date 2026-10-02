// Package links makes the directory links that share data between accounts and the
// executable links that become the claude-<letter> commands.
package links

import "os"

// IsLink reports whether path is a symlink or, on Windows, a junction.
func IsLink(path string) bool {
	_, err := os.Readlink(path)
	return err == nil
}

// Points reports whether link resolves to the same file or directory as target.
func Points(link, target string) bool {
	a, err := os.Stat(link)
	if err != nil {
		return false
	}
	b, err := os.Stat(target)
	return err == nil && os.SameFile(a, b)
}
