package main

import (
	"fmt"

	"github.com/MFpizza/claude-dotfiles/internal/config"
)

// menu is the numbered menu shown when claude-accounts runs without arguments.
func (a *app) menu() error {
	for !a.eof {
		fmt.Fprintln(a.out)
		a.list()
		a.say("menu", a.cfg.Layout, a.cfg.Lang)
		var err error
		switch a.ask("prompt") {
		case "1":
			err = a.add(a.ask("ask_type") == "2", "")
		case "2":
			err = a.remove(a.ask("ask_which"), false, false)
		case "3":
			name := a.ask("ask_which")
			if acc := a.cfg.Find(name); acc == nil {
				err = a.noAccount(name)
			} else {
				err = a.setShown(name, !acc.StatusLine)
			}
		case "4":
			switch a.ask("ask_layout") {
			case "1":
				err = a.setLayout(config.LayoutFull)
			case "2":
				err = a.setLayout(config.LayoutCompact)
			}
		case "5":
			switch a.ask("ask_lang") {
			case "1":
				err = a.setLang("auto")
			case "2":
				err = a.setLang("en")
			case "3":
				err = a.setLang("zh-TW")
			}
		case "6":
			return a.uninstall(false)
		case "0", "":
			return nil
		default:
			a.say("invalid_choice")
		}
		if err != nil {
			fmt.Fprintln(a.out, err)
		}
	}
	return nil
}
