//go:build !windows

package launch

import "syscall"

// Exec replaces this process with Claude, so signals and the exit code are Claude's own.
func Exec(p *Plan) error { return syscall.Exec(p.Path, p.Argv, p.Env) }
