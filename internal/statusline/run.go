package statusline

import (
	"io"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/config"
	"github.com/MFpizza/claude-dotfiles/internal/i18n"
	"github.com/MFpizza/claude-dotfiles/internal/pet"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

type Options struct {
	MainDir string
	In      io.Reader
	Out     io.Writer
	Now     time.Time
	Fetcher *Fetcher
	Rand    *rand.Rand
}

// Run prints the status line. Without a readable config it shows the main account
// alone, so the status line works before setup and with a broken file.
func Run(o Options) {
	in := ParseInput(o.In)
	cfg, err := config.Load(o.MainDir)
	if err != nil {
		cfg = config.New(o.MainDir)
	}
	lang := i18n.Resolve(cfg.Lang)
	activeDir := config.ActiveDir()

	var shown []config.Account
	active := map[string]bool{}
	for _, acc := range cfg.Accounts {
		active[acc.Name] = config.SamePath(acc.Path(), activeDir)
		if acc.StatusLine || active[acc.Name] {
			shown = append(shown, acc)
		}
	}

	cache := LoadCache(o.MainDir)
	errs := map[string]error{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, acc := range shown {
		if acc.Type == config.TypeAPI {
			continue
		}
		if active[acc.Name] {
			plan := Plan(acc.Path())
			if old := cache[acc.Name]; plan == "" && old != nil {
				plan = old.Plan
			}
			if e := FromRateLimits(in.RateLimits, plan, o.Now); e != nil {
				cache[acc.Name] = e
				continue
			}
		}
		if e := cache[acc.Name]; e != nil && o.Now.Sub(fromUnix(e.FetchedAt)) <= CacheTTL {
			continue
		}
		wg.Add(1)
		go func(acc config.Account, isActive bool) {
			defer wg.Done()
			e, err := o.Fetcher.Fetch(acc.Path(), isActive, o.Now)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs[acc.Name] = err
				if old := cache[acc.Name]; old != nil {
					old.LastError = err.Error()
				}
				return
			}
			cache[acc.Name] = e
		}(acc, active[acc.Name])
	}
	wg.Wait()
	cache.Save(o.MainDir)

	ledger := LoadLedger(o.MainDir)
	var fulls, compacts []string
	for _, acc := range shown {
		if acc.Type == config.TypeAPI {
			if active[acc.Name] && in.SessionID != "" && in.Cost.TotalCostUSD != nil {
				ledger.Record(acc.Name, in.SessionID, *in.Cost.TotalCostUSD, o.Now)
				ledger.Save(o.MainDir)
			}
			if full, compact, ok := apiRow(acc, ledger, in, active[acc.Name], o.Now, lang); ok {
				fulls, compacts = append(fulls, full), append(compacts, compact)
			}
			continue
		}
		full, compact := subscriptionRow(acc, cache[acc.Name], errs[acc.Name], active[acc.Name], o.Now, lang)
		fulls, compacts = append(fulls, full), append(compacts, compact)
	}

	var tail []string
	if in.Model.DisplayName != "" {
		tail = append(tail, style.Muted+in.Model.DisplayName+style.Reset)
	}
	tail = append(tail, pet.Update(filepath.Join(o.MainDir, "statusline-pet.json"), in.ContextPct(), o.Now, o.Rand))

	var lines []string
	switch {
	case cfg.Layout == config.LayoutCompact:
		lines = []string{strings.Join(append(compacts, tail...), style.Sep)}
	case len(fulls) == 0:
		lines = []string{strings.Join(tail, style.Sep)}
	default:
		lines = fulls
		lines[0] += style.Sep + strings.Join(tail, style.Sep)
	}
	io.WriteString(o.Out, strings.Join(lines, "\n"))
}
