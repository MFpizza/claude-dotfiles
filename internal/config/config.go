// Package config reads and writes claude-accounts.json, the list of accounts that
// every command link, shared folder and the status line are built from.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
)

const (
	FileName         = "claude-accounts.json"
	TypeSubscription = "subscription"
	TypeAPI          = "api"
	LayoutFull       = "full"
	LayoutCompact    = "compact"
	MainName         = "a"
)

var ErrFull = errors.New("all 26 letters are in use")

type Account struct {
	Name       string `json:"name"`
	Dir        string `json:"dir"`
	Type       string `json:"type"`
	StatusLine bool   `json:"statusline"`
}

// Path is the account's config dir with ~ expanded.
func (a Account) Path() string { return ExpandHome(a.Dir) }

type Config struct {
	Version  int       `json:"version"`
	Layout   string    `json:"layout"`
	Lang     string    `json:"lang"`
	Accounts []Account `json:"accounts"` // Accounts[0] is the main account "a"
}

func New(mainDir string) *Config {
	return &Config{Version: 1, Layout: LayoutFull, Lang: "auto", Accounts: []Account{
		{Name: MainName, Dir: CollapseHome(mainDir), Type: TypeSubscription, StatusLine: true},
	}}
}

func Load(mainDir string) (*Config, error) {
	var c Config
	path := filepath.Join(mainDir, FileName)
	if err := fsutil.ReadJSON(path, &c); err != nil {
		return nil, err
	}
	if len(c.Accounts) == 0 || c.Accounts[0].Name != MainName {
		return nil, fmt.Errorf("%s: the first account must be %q", path, MainName)
	}
	if c.Layout != LayoutCompact {
		c.Layout = LayoutFull
	}
	if c.Lang == "" {
		c.Lang = "auto"
	}
	return &c, nil
}

func (c *Config) Save(mainDir string) error {
	c.Version = 1
	return fsutil.WriteJSON(filepath.Join(mainDir, FileName), c, true)
}

func (c *Config) Find(name string) *Account {
	for i := range c.Accounts {
		if c.Accounts[i].Name == name {
			return &c.Accounts[i]
		}
	}
	return nil
}

func (c *Config) Names() []string {
	names := make([]string, len(c.Accounts))
	for i, a := range c.Accounts {
		names[i] = a.Name
	}
	return names
}

// NextLetter is the first free letter from b to z, so removed letters get reused.
func (c *Config) NextLetter() (string, error) {
	for ch := 'b'; ch <= 'z'; ch++ {
		if c.Find(string(ch)) == nil {
			return string(ch), nil
		}
	}
	return "", ErrFull
}

// Add appends a and keeps the accounts after the main one in letter order.
func (c *Config) Add(a Account) {
	c.Accounts = append(c.Accounts, a)
	rest := c.Accounts[1:]
	sort.Slice(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
}

func (c *Config) Delete(name string) {
	for i, a := range c.Accounts {
		if a.Name == name {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			return
		}
	}
}

func Home() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

func ExpandHome(p string) string {
	if p == "~" {
		return Home()
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(Home(), p[2:])
	}
	return p
}

// CollapseHome writes a path under the home directory as ~/..., which keeps
// claude-accounts.json readable and portable between machines.
func CollapseHome(p string) string {
	abs := clean(p)
	rel, err := filepath.Rel(clean(Home()), abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return abs
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}

func clean(p string) string {
	if abs, err := filepath.Abs(ExpandHome(p)); err == nil {
		return abs
	}
	return filepath.Clean(p)
}

func SamePath(a, b string) bool {
	a, b = clean(a), clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// ActiveDir is the config dir of the Claude that is running now.
func ActiveDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return clean(dir)
	}
	return clean(filepath.Join(Home(), ".claude"))
}

// ResolveMainDir finds the main account's dir. Inside another account (claude-b sets
// CLAUDE_CONFIG_DIR to its own dir) the main one is where its shared folders point.
func ResolveMainDir(override string) string {
	if override != "" {
		return clean(override)
	}
	start := ActiveDir()
	if _, err := os.Stat(filepath.Join(start, FileName)); err == nil {
		return start
	}
	if target, err := os.Readlink(filepath.Join(start, "projects")); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(start, target)
		}
		return filepath.Dir(clean(target))
	}
	return start
}
