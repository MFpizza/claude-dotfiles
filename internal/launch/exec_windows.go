//go:build windows

package launch

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// Exec runs Claude and exits with its code. Ctrl+C goes to Claude; this process
// ignores it and just waits.
func Exec(p *Plan) error {
	cmd := exec.Command(p.Path, p.Argv[1:]...)
	cmd.Env = p.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	signal.Ignore(os.Interrupt)
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		os.Exit(exit.ExitCode())
	}
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
