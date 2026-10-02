//go:build !windows

package migrate

import "path/filepath"

// ShellFiles are the startup files the old installer may have written to.
func ShellFiles(home string) []string {
	return []string{filepath.Join(home, ".bashrc"), filepath.Join(home, ".zshrc")}
}
