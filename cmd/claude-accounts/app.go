package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/links"
	"github.com/MFpizza/claude-dotfiles/internal/migrate"
	"github.com/MFpizza/claude-dotfiles/internal/settings"
	"github.com/MFpizza/claude-dotfiles/internal/statusline"
)

type app struct {
	mainDir    string
	exe        string
	cfg        *config.Config // nil until set up
	lang       i18n.Lang
	in         *bufio.Reader
	out        io.Writer
	eof        bool
	shellFiles func() []string
}

func newApp(stdin io.Reader, stdout io.Writer) (*app, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	a := &app{mainDir: config.ResolveMainDir(""), exe: exe, in: bufio.NewReader(stdin), out: stdout,
		shellFiles: func() []string { return migrate.ShellFiles(config.Home()) }}
	return a, a.load()
}

// load reads the config; a missing one means "not set up yet", a broken one is an error.
func (a *app) load() error {
	cfg, err := config.Load(a.mainDir)
	switch {
	case err == nil:
		a.cfg = cfg
	case !errors.Is(err, fs.ErrNotExist):
		return errors.New(i18n.Resolve("auto").T("err_config", filepath.Join(a.mainDir, config.FileName), err))
	}
	setting := "auto"
	if a.cfg != nil {
		setting = a.cfg.Lang
	}
	a.lang = i18n.Resolve(setting)
	return nil
}

func (a *app) say(key string, args ...any) { fmt.Fprint(a.out, a.lang.T(key, args...)) }

func (a *app) ask(key string, args ...any) string {
	a.say(key, args...)
	line, err := a.in.ReadString('\n')
	if err != nil && line == "" {
		a.eof = true
	}
	return strings.TrimSpace(line)
}

func (a *app) confirm(defaultYes bool, key string, args ...any) bool {
	switch strings.ToLower(a.ask(key, args...)) {
	case "":
		return defaultYes
	case "y", "yes":
		return true
	}
	return false
}

func (a *app) fail(key string, args ...any) error { return errors.New(a.lang.T(key, args...)) }

func (a *app) save() error { return a.cfg.Save(a.mainDir) }

func (a *app) dispatch(sub string, args []string) error {
	switch sub {
	case "help", "-h", "--help":
		a.say("usage")
		return nil
	case "uninstall":
		if a.cfg == nil {
			a.cfg = migrate.Detect(a.mainDir)
		}
		return a.uninstall(hasFlag(args, "--yes"))
	}
	if err := a.ensureSetup(sub == ""); err != nil {
		return err
	}
	pos := positional(args)
	switch sub {
	case "":
		return a.menu()
	case "add":
		return a.add(hasFlag(args, "--api"), flagValue(args, "--dir"))
	case "list":
		a.list()
		return nil
	case "remove", "show", "hide":
		if len(pos) != 1 {
			return a.fail("err_usage_letter", sub)
		}
		if sub == "remove" {
			return a.remove(pos[0], hasFlag(args, "--yes"), hasFlag(args, "--delete-dir"))
		}
		return a.setShown(pos[0], sub == "show")
	case "layout":
		if len(pos) != 1 {
			return a.fail("err_layout")
		}
		return a.setLayout(pos[0])
	case "lang":
		if len(pos) != 1 {
			return a.fail("err_lang")
		}
		return a.setLang(pos[0])
	}
	return a.fail("err_unknown_cmd", sub)
}

func (a *app) ensureSetup(interactive bool) error {
	if a.cfg != nil {
		return a.repair()
	}
	return a.setup(interactive)
}

// setup is the first run: it imports accounts of the old version and removes what
// its installer added, then installs commands and the status line.
func (a *app) setup(interactive bool) error {
	a.say("setup_title", config.CollapseHome(a.mainDir))
	cfg := migrate.Detect(a.mainDir)
	for i := 1; i < len(cfg.Accounts); i++ {
		acc := &cfg.Accounts[i]
		a.say("found_account", strings.ToUpper(acc.Name), acc.Dir)
		if acc.Type == config.TypeAPI && interactive && !a.confirm(true, "confirm_api", strings.ToUpper(acc.Name)) {
			acc.Type = config.TypeSubscription
		}
	}
	if layout := migrate.LegacyLayout(a.mainDir); layout != "" {
		cfg.Layout = layout
	}
	for _, f := range a.shellFiles() {
		if ok, err := migrate.StripBlock(f); err == nil && ok {
			a.say("cleaned_block", f, f)
		}
	}
	for _, f := range migrate.OldShims(config.Home()) {
		if os.Remove(f) == nil {
			a.say("removed_file", f)
		}
	}
	if err := os.MkdirAll(a.mainDir, 0o755); err != nil {
		return err
	}
	a.cfg = cfg
	if err := a.save(); err != nil {
		return err
	}
	migrate.RemoveLegacyLayout(a.mainDir)
	if err := a.repair(); err != nil {
		return err
	}
	a.say("setup_done", a.commandList())
	return nil
}

// repair points every command and the status line at this executable, which moves
// after an update or reinstall.
func (a *app) repair() error {
	if err := accounts.SyncCommands(a.cfg, a.exe); err != nil {
		return err
	}
	changed, err := settings.SetStatusLine(a.mainDir, a.exe)
	if err != nil {
		return err
	}
	if changed {
		a.say("statusline_set", filepath.Join(a.mainDir, "settings.json"))
	}
	return nil
}

func (a *app) commandList() string {
	var names []string
	for _, n := range a.cfg.Names() {
		names = append(names, "claude-"+n)
	}
	return strings.Join(names, ", ")
}

func (a *app) typeLabel(t string) string {
	if t == config.TypeAPI {
		return a.lang.T("type_api")
	}
	return a.lang.T("type_subscription")
}

func (a *app) noAccount(name string) error {
	return a.fail("err_no_account", name, strings.Join(a.cfg.Names(), ", "))
}

func (a *app) add(api bool, dir string) error {
	acc, conflicts, err := accounts.Add(a.cfg, a.mainDir, a.exe, api, dir)
	if errors.Is(err, config.ErrFull) {
		return a.fail("err_full")
	}
	if err != nil {
		return err
	}
	for _, c := range conflicts {
		a.say("conflict", acc.Dir, c)
	}
	if err := a.save(); err != nil {
		return err
	}
	a.say("added", strings.ToUpper(acc.Name), a.typeLabel(acc.Type), acc.Dir, acc.Name)
	if api {
		a.say("added_api", filepath.Join(acc.Dir, "account-settings.json"))
	}
	return nil
}

func (a *app) remove(name string, yes, deleteDir bool) error {
	if name == config.MainName {
		return a.fail("err_main")
	}
	acc := a.cfg.Find(name)
	if acc == nil {
		return a.noAccount(name)
	}
	if !yes && !a.confirm(false, "confirm_remove", strings.ToUpper(name)) {
		return nil
	}
	if !yes && !deleteDir {
		deleteDir = a.confirm(false, "confirm_delete_dir", acc.Dir)
	}
	dir := acc.Dir
	if err := accounts.Remove(a.cfg, a.exe, name, deleteDir); err != nil {
		return err
	}
	statusline.Forget(a.mainDir, name)
	if err := a.save(); err != nil {
		return err
	}
	a.say("removed", strings.ToUpper(name))
	if deleteDir {
		a.say("deleted_dir", dir)
	} else {
		a.say("kept_dir", dir)
	}
	return nil
}

func (a *app) list() {
	a.say("accounts_header")
	for _, acc := range a.cfg.Accounts {
		shown := a.lang.T("hidden")
		if acc.StatusLine {
			shown = a.lang.T("shown")
		}
		a.say("account_row", acc.Name, a.typeLabel(acc.Type), acc.Dir, shown)
	}
}

func (a *app) setShown(name string, show bool) error {
	acc := a.cfg.Find(name)
	if acc == nil {
		return a.noAccount(name)
	}
	acc.StatusLine = show
	if err := a.save(); err != nil {
		return err
	}
	if show {
		a.say("set_shown", strings.ToUpper(name))
	} else {
		a.say("set_hidden", strings.ToUpper(name))
	}
	return nil
}

func (a *app) setLayout(layout string) error {
	if layout != config.LayoutFull && layout != config.LayoutCompact {
		return a.fail("err_layout")
	}
	a.cfg.Layout = layout
	if err := a.save(); err != nil {
		return err
	}
	a.say("layout_set", layout)
	return nil
}

func (a *app) setLang(lang string) error {
	if lang != "auto" && lang != string(i18n.En) && lang != string(i18n.ZhTW) {
		return a.fail("err_lang")
	}
	a.cfg.Lang = lang
	a.lang = i18n.Resolve(lang)
	if err := a.save(); err != nil {
		return err
	}
	a.say("lang_set", lang)
	return nil
}

// uninstall removes the commands, the status line and the config; account folders
// and their data stay.
func (a *app) uninstall(yes bool) error {
	if !yes && !a.confirm(false, "confirm_uninstall") {
		return nil
	}
	for _, acc := range a.cfg.Accounts {
		if err := links.RemoveExe(accounts.CommandPath(filepath.Dir(a.exe), acc.Name)); err != nil {
			return err
		}
	}
	if err := settings.RemoveStatusLine(a.mainDir); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(a.mainDir, config.FileName)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	a.say("uninstalled", a.exe)
	return nil
}
