package pet

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MFpizza/claude-dotfiles/internal/fsutil"
	"github.com/MFpizza/claude-dotfiles/internal/style"
)

var t0 = time.Unix(1_800_000_000, 0)

func secs(t time.Time) float64 { return float64(t.UnixNano()) / 1e9 }

func petFile(t *testing.T, s any) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "statusline-pet.json")
	if raw, ok := s.(string); ok {
		os.WriteFile(path, []byte(raw), 0o644)
	} else if err := fsutil.WriteJSON(path, s, false); err != nil {
		t.Fatal(err)
	}
	return path
}

func load(t *testing.T, path string) Save {
	t.Helper()
	var s Save
	if err := fsutil.ReadJSON(path, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func hours(h float64) *float64 { return &h }

func TestLevelBoundaries(t *testing.T) {
	for l := 1; l <= MaxLevel; l++ {
		if got := LevelFor(HoursFor(l)); got != l {
			t.Errorf("level %d: got %d", l, got)
		}
	}
	if LevelFor(HoursFor(8)-0.001) != 7 || LevelFor(1000) != MaxLevel {
		t.Error("edges")
	}
}

func TestMigratesSessionSave(t *testing.T) {
	path := petFile(t, `{"sessions":39,"seen":["x"],"level":7}`)
	out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1))))
	s := load(t, path)
	if s.Level != 7 || s.Life != 1 || s.Line != "bird" || math.Abs(*s.Hours-HoursFor(7)) > 1e-9 {
		t.Fatalf("got %+v", s)
	}
	if s.Branch != "fowl" && s.Branch != "raptor" {
		t.Fatalf("branch %q", s.Branch)
	}
	if out != "🐔 Lv7" {
		t.Fatalf("got %q", out)
	}
}

func TestOldDragonSave(t *testing.T) {
	path := petFile(t, `{"life":2,"line":"dragon","hours":50,"level":14,"past":[{"line":"bird","ended":"x"}]}`)
	out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1))))
	if s := load(t, path); s.Line != "reptile" {
		t.Fatalf("got %+v", s)
	}
	if out != "⭐🦕 Lv14" && out != "⭐🐲 Lv14" {
		t.Fatalf("got %q", out)
	}
}

func TestActiveTimeOnly(t *testing.T) {
	ts := secs(t0)
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "fowl", Hours: hours(5), Level: 7, Tick: ts - 60})
	Update(path, nil, t0, rand.New(rand.NewSource(1)))
	if h := *load(t, path).Hours; math.Abs(h-(5+60.0/3600)) > 1e-9 {
		t.Fatalf("60s gap: %v", h)
	}
	Update(path, nil, t0.Add(time.Hour), rand.New(rand.NewSource(1)))
	if h := *load(t, path).Hours; math.Abs(h-(5+60.0/3600)) > 1e-9 {
		t.Fatalf("an idle hour must not count: %v", h)
	}
}

func TestLevelUpParty(t *testing.T) {
	ts := secs(t0)
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "fowl", Hours: hours(HoursFor(8) - 0.001), Level: 7, Tick: ts - 60})
	if out := style.Strip(Update(path, nil, t0, rand.New(rand.NewSource(1)))); out != "🐔🎉 Lv8" {
		t.Fatalf("got %q", out)
	}
	if out := style.Strip(Update(path, nil, t0.Add(601*time.Second), rand.New(rand.NewSource(1)))); out != "🐔 Lv8" {
		t.Fatalf("after 10 minutes: %q", out)
	}
}

func TestRebirthCollectsEveryEnding(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))
	now := t0
	path := petFile(t, Save{Life: 1, Line: "bird", Branch: "raptor", Hours: hours(4.8), Level: 7})
	seen := map[ending]bool{{"bird", "raptor"}: true}
	rebirth := func() Save {
		s := load(t, path)
		s.Hours, s.Tick = hours(LifeHours-1e-4), secs(now)-60
		fsutil.WriteJSON(path, s, false)
		Update(path, nil, now, rnd)
		now = now.Add(time.Hour)
		return load(t, path)
	}
	for i := 0; i < 16; i++ {
		s := rebirth()
		seen[ending{s.Line, s.Branch}] = true
	}
	if len(seen) != 17 {
		t.Fatalf("17 lives raised %d different final forms", len(seen))
	}
	for i := 0; i < 30; i++ {
		prev := load(t, path).Line
		if s := rebirth(); s.Line == prev {
			t.Fatalf("same line twice in a row: %s", prev)
		}
	}
	s := load(t, path)
	if out := style.Strip(Render(&s, nil, secs(now))); !strings.HasPrefix(out, "⭐46") {
		t.Fatalf("stars: %q", out)
	}
}

func TestBodyFlexibleLevels(t *testing.T) {
	var b strings.Builder
	for l := 1; l <= MaxLevel; l++ {
		b.WriteString(Body(&Save{Line: "mammal", Branch: "canine", Level: l}))
	}
	if want := "🍼🍼🐾🐾🐶🐶🐶🐶🐶🐕🐕🐕🐕🐕🐕🐺🐺🐺🐺👑🐺"; b.String() != want {
		t.Fatalf("got %s", b.String())
	}
}

func TestMood(t *testing.T) {
	cases := map[float64]string{0: "✨", 29.9: "✨", 30: "", 59: "", 60: "💦", 84.9: "💦", 85: "💤", 100: "💤"}
	for pct, want := range cases {
		p := pct
		if got := Mood(&p); got != want {
			t.Errorf("%v: %q", pct, got)
		}
	}
	if Mood(nil) != "" {
		t.Error("nil")
	}
}

func TestEmojiWithinUnicode12(t *testing.T) {
	newer := map[rune]bool{0x1F90C: true, 0x1F972: true, 0x1F977: true, 0x1F978: true, 0x1F9A3: true,
		0x1F9A4: true, 0x1F9AB: true, 0x1F9AC: true, 0x1F9AD: true, 0x1F9CB: true}
	check := func(stages []Stage) {
		for _, st := range stages {
			for _, r := range st.Emoji {
				if r == 0x200D || r >= 0x1FA70 || newer[r] {
					t.Errorf("%s uses U+%X", st.Emoji, r)
				}
			}
		}
	}
	endings := 0
	for _, l := range Lines {
		check(l.Stages)
		for _, b := range l.Branches {
			check(b.Stages)
		}
		endings += len(l.endings())
	}
	if endings != 17 {
		t.Fatalf("%d final forms", endings)
	}
}
