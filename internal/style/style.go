// Package style holds the status line's ANSI colours.
package style

import (
	"fmt"
	"regexp"
)

func RGB(hex string) string {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

// Colour-blind friendly scale (theme is dark-daltonized): blue -> amber -> orange.
var (
	Low    = RGB("#5fafff")
	Mid    = RGB("#e5c07b")
	High   = RGB("#ff8c42")
	Accent = RGB("#d97757") // Claude orange, marks the running account
	Text   = RGB("#c8ccd4")
	Muted  = RGB("#7f8794")
	Track  = RGB("#3e4451")
	Sep    = " " + Track + "│" + Reset + " "
)

const (
	Bold   = "\x1b[1m"
	Italic = "\x1b[3m"
	Reset  = "\x1b[0m"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Strip removes colour codes, for tests and plain-text output.
func Strip(s string) string { return ansi.ReplaceAllString(s, "") }
