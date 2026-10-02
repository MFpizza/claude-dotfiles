// Package pet is the status line pet. It grows with active Claude time
// (Lv = 1 + 19 * cbrt(hours / LifeHours)); after RebirthHours it is reborn with a ⭐ as
// a random line, preferring final forms not raised yet.
package pet

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

const (
	LifeHours    = 150.0
	RebirthHours = LifeHours + 15 // Lv20 wears its crown for 15 active hours first
	MaxLevel     = 20
	ActiveGap    = 300.0 // seconds; refreshes further apart count as idle
	LevelUpSecs  = 600.0 // how long 🎉 stays after a level-up or rebirth
)

type Past struct {
	Line   string `json:"line"`
	Branch string `json:"branch,omitempty"`
	Ended  string `json:"ended"`
}

type Save struct {
	Life      int      `json:"life"`
	Line      string   `json:"line"`
	Branch    string   `json:"branch,omitempty"`
	Hours     *float64 `json:"hours,omitempty"`
	Level     int      `json:"level"`
	LeveledAt float64  `json:"leveled_at,omitempty"`
	Tick      float64  `json:"tick,omitempty"`
	Past      []Past   `json:"past,omitempty"`
	Sessions  int      `json:"sessions,omitempty"` // old saves counted sessions instead of hours
}

func HoursFor(level int) float64 {
	return LifeHours * math.Pow(float64(level-1)/float64(MaxLevel-1), 3)
}

func LevelFor(hours float64) int {
	frac := math.Max(0, math.Min(hours/LifeHours, 1))
	return 1 + int(float64(MaxLevel-1)*math.Cbrt(frac)+1e-9)
}

type ending struct{ line, branch string }

func findLine(key string) *Line {
	for i := range Lines {
		if Lines[i].Key == key {
			return &Lines[i]
		}
	}
	return nil
}

func (l *Line) branch(key string) *Branch {
	for i := range l.Branches {
		if l.Branches[i].Key == key {
			return &l.Branches[i]
		}
	}
	return nil
}

// endings are the line's final forms: one per branch, or one if it never splits.
func (l *Line) endings() []ending {
	if len(l.Branches) == 0 {
		return []ending{{l.Key, ""}}
	}
	out := make([]ending, len(l.Branches))
	for i, b := range l.Branches {
		out[i] = ending{l.Key, b.Key}
	}
	return out
}

func raised(s *Save) map[ending]bool {
	done := map[ending]bool{}
	for _, p := range s.Past {
		done[ending{p.Line, p.Branch}] = true
	}
	return done
}

func pickBranch(l *Line, done map[ending]bool, rnd *rand.Rand) string {
	all := l.endings()
	var fresh []ending
	for _, e := range all {
		if !done[e] {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) == 0 {
		fresh = all
	}
	return fresh[rnd.Intn(len(fresh))].branch
}

// nextLine picks a line that still has final forms not raised; once every one has
// been, any line but the current one.
func nextLine(s *Save, rnd *rand.Rand) (string, string) {
	done := raised(s)
	done[ending{s.Line, s.Branch}] = true
	var fresh, others []*Line
	for i := range Lines {
		l := &Lines[i]
		for _, e := range l.endings() {
			if !done[e] {
				fresh = append(fresh, l)
				break
			}
		}
		if l.Key != s.Line {
			others = append(others, l)
		}
	}
	pool := fresh
	if len(pool) == 0 {
		pool = others
	}
	l := pool[rnd.Intn(len(pool))]
	return l.Key, pickBranch(l, done, rnd)
}

func normalize(s *Save, rnd *rand.Rand) {
	if s.Hours == nil {
		// Older saves counted sessions (Lv = 1 + sqrt(sessions)); keep that level.
		level := s.Level
		if level == 0 {
			level = 1 + int(math.Sqrt(float64(s.Sessions)))
		}
		h := HoursFor(level)
		*s = Save{Life: 1, Line: "bird", Hours: &h, Level: level}
	}
	if s.Life == 0 {
		s.Life = 1
	}
	if findLine(s.Line) == nil {
		if s.Line == "dragon" { // merged into reptile
			s.Line = "reptile"
		} else {
			s.Line = "bird"
		}
		s.Branch = ""
	}
	l := findLine(s.Line)
	if len(l.Branches) == 0 {
		s.Branch = ""
	} else if l.branch(s.Branch) == nil {
		s.Branch = pickBranch(l, raised(s), rnd)
	}
}

// feed adds the time since the last refresh if it was short enough to count as active.
func feed(s *Save, now float64, rnd *rand.Rand) {
	gap := now - s.Tick
	s.Tick = now
	if gap <= 0 || gap >= ActiveGap {
		return
	}
	h := *s.Hours + gap/3600
	for h >= RebirthHours {
		h -= RebirthHours
		s.Past = append(s.Past, Past{Line: s.Line, Branch: s.Branch,
			Ended: time.Unix(int64(now), 0).Format("2006-01-02")})
		s.Line, s.Branch = nextLine(s, rnd)
		s.Life++
		s.LeveledAt = now
	}
	s.Hours = &h
	if level := LevelFor(h); level != s.Level {
		s.Level, s.LeveledAt = level, now
	}
}

func Body(s *Save) string {
	l := findLine(s.Line)
	stages := l.Stages
	if b := l.branch(s.Branch); b != nil {
		stages = append(append([]Stage{}, stages...), b.Stages...)
	}
	body := stages[0].Emoji
	for _, st := range stages {
		if s.Level >= st.Level {
			body = st.Emoji
		}
	}
	return body
}

// Mood follows context usage: fresh -> normal -> tired -> sleepy (time to /compact).
func Mood(pct *float64) string {
	if pct == nil {
		return ""
	}
	switch p := *pct; {
	case p < 30:
		return "✨"
	case p < 60:
		return ""
	case p < 85:
		return "💦"
	}
	return "💤"
}

func Render(s *Save, ctxPct *float64, now float64) string {
	stars := strings.Repeat("⭐", s.Life-1)
	if s.Life-1 > 3 {
		stars = fmt.Sprintf("⭐%d", s.Life-1)
	}
	head := stars + Body(s) + Mood(ctxPct)
	if now-s.LeveledAt < LevelUpSecs {
		return fmt.Sprintf("%s🎉 %s%sLv%d%s", head, style.Accent, style.Bold, s.Level, style.Reset)
	}
	return fmt.Sprintf("%s %sLv%d%s", head, style.Muted, s.Level, style.Reset)
}

// Update loads the pet at path, feeds it and saves it, and returns its status text.
func Update(path string, ctxPct *float64, now time.Time, rnd *rand.Rand) string {
	var s Save
	if fsutil.ReadJSON(path, &s) != nil {
		s = Save{}
	}
	t := float64(now.UnixNano()) / 1e9
	normalize(&s, rnd)
	feed(&s, t, rnd)
	fsutil.WriteJSON(path, &s, false)
	return Render(&s, ctxPct, t)
}
