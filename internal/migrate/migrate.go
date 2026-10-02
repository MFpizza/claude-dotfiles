// Package migrate turns an install of the old Python/shell version (accounts A, B, C)
// into a claude-accounts.json and removes what the old installer added.
package migrate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/accounts"
	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/links"
)

// Markers of the block the old installer put in $PROFILE and ~/.bashrc.
const (
	Begin = "# >>> claude-dotfiles >>>"
	End   = "# <<< claude-dotfiles <<<"
)

// Detect builds a config from <main>-<letter> dirs whose shared folders link to the
// main account. By the old convention c is the API-billed account.
func Detect(mainDir string) *config.Config {
	cfg := config.New(mainDir)
	for ch := 'b'; ch <= 'z'; ch++ {
		name := string(ch)
		dir := mainDir + "-" + name
		if !linksTo(dir, mainDir) {
			continue
		}
		typ := config.TypeSubscription
		if name == "c" {
			typ = config.TypeAPI
		}
		cfg.Add(config.Account{Name: name, Dir: config.CollapseHome(dir), Type: typ, StatusLine: true})
	}
	return cfg
}

func linksTo(dir, mainDir string) bool {
	for _, name := range accounts.SharedDirs {
		p := filepath.Join(dir, name)
		if links.IsLink(p) && links.Points(p, filepath.Join(mainDir, name)) {
			return true
		}
	}
	return false
}

// LegacyLayout is the layout saved by the old status line, or "".
func LegacyLayout(mainDir string) string {
	var v struct {
		Layout string `json:"layout"`
	}
	if fsutil.ReadJSON(filepath.Join(mainDir, "statusline.json"), &v) == nil &&
		(v.Layout == config.LayoutFull || v.Layout == config.LayoutCompact) {
		return v.Layout
	}
	return ""
}

func RemoveLegacyLayout(mainDir string) { os.Remove(filepath.Join(mainDir, "statusline.json")) }

// StripBlock removes the old installer's block from a shell or PowerShell startup
// file, keeping the original as <file>.bak. A missing file or block is not an error.
func StripBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, nil
	}
	text := string(data)
	i, j := strings.Index(text, Begin), strings.Index(text, End)
	if i < 0 || j < i {
		return false, nil
	}
	text = text[:i] + strings.TrimLeft(text[j+len(End):], "\r\n")
	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return false, err
	}
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	return true, fsutil.WriteFileAtomic(path, []byte(text), mode)
}

// OldShims are the cmd.exe forwarders the old installer put in ~/.local/bin.
func OldShims(home string) []string {
	var found []string
	for _, n := range []string{"a", "b", "c"} {
		p := filepath.Join(home, ".local", "bin", "claude-"+n+".cmd")
		if _, err := os.Stat(p); err == nil {
			found = append(found, p)
		}
	}
	return found
}
