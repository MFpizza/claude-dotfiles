// claude-accounts runs several Claude Code accounts side by side. Called as
// claude-<letter> it starts that account; otherwise it manages accounts or, as
// `claude-accounts statusline`, renders the status line.
package main

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/launch"
	"github.com/MFpizza/claude-dotfiles/internal/statusline"
)

var version = "dev"

var accountCmd = regexp.MustCompile(`^claude-([a-z])$`)

func main() { os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr)) }

// commandName is the name this program was started as, e.g. "claude-b".
func commandName(argv0 string) string {
	return strings.TrimSuffix(strings.ToLower(filepath.Base(argv0)), ".exe")
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if m := accountCmd.FindStringSubmatch(commandName(args[0])); m != nil {
		return launchAccount(m[1], args[1:], stderr)
	}
	sub, rest := "", args[1:]
	if len(rest) > 0 {
		sub, rest = rest[0], rest[1:]
	}
	switch sub {
	case "statusline":
		return statuslineCmd(rest, stdin, stdout)
	case "version", "--version":
		fmt.Fprintln(stdout, version)
		return 0
	}
	a, err := newApp(stdin, stdout)
	if err == nil {
		err = a.dispatch(sub, rest)
	}
	if err != nil {
		fmt.Fprintln(stderr, "claude-accounts:", err)
		return 1
	}
	return 0
}

func launchAccount(name string, args []string, stderr io.Writer) int {
	mainDir := config.ResolveMainDir("")
	cfg, err := config.Load(mainDir)
	if err != nil {
		fmt.Fprintln(stderr, i18n.Resolve("auto").T("err_not_set_up"))
		return 1
	}
	lang := i18n.Resolve(cfg.Lang)
	plan, err := launch.Build(cfg, mainDir, name, args, exec.LookPath)
	var missing *launch.DirMissingError
	switch {
	case errors.Is(err, accounts.ErrNoAccount):
		err = errors.New(lang.T("err_no_account", name, strings.Join(cfg.Names(), ", ")))
	case errors.As(err, &missing):
		err = errors.New(lang.T("err_dir_missing", name, missing.Dir, name))
	case errors.Is(err, launch.ErrNoClaude):
		err = errors.New(lang.T("err_no_claude"))
	case err == nil:
		err = launch.Exec(plan)
	}
	if err != nil {
		fmt.Fprintln(stderr, "claude-"+name+":", err)
		return 1
	}
	return 0
}

// statuslineCmd never fails: Claude Code would just show an empty status line.
func statuslineCmd(args []string, stdin io.Reader, stdout io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stdout, "claude-accounts: %v", r)
		}
	}()
	now := time.Now()
	statusline.Run(statusline.Options{
		MainDir: config.ResolveMainDir(flagValue(args, "--base")),
		In:      stdin, Out: stdout, Now: now,
		Fetcher: statusline.NewFetcher(),
		Rand:    rand.New(rand.NewSource(now.UnixNano())),
	})
	return 0
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// positional drops flags (and --dir's value) from args.
func positional(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--dir":
			i++
		case !strings.HasPrefix(args[i], "--"):
			out = append(out, args[i])
		}
	}
	return out
}
