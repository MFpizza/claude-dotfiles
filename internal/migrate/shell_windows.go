//go:build windows

package migrate

import (
	"os/exec"
	"strings"
)

// ShellFiles are the $PROFILE of Windows PowerShell and PowerShell 7, whichever exist.
func ShellFiles(home string) []string {
	var paths []string
	for _, shell := range []string{"powershell", "pwsh"} {
		if _, err := exec.LookPath(shell); err != nil {
			continue
		}
		out, err := exec.Command(shell, "-NoProfile", "-Command", "$PROFILE").Output()
		if p := strings.TrimSpace(string(out)); err == nil && p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
