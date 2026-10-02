package i18n

import (
	"regexp"
	"strings"
	"testing"
)

func TestEveryMessageHasBothLanguagesWithSameVerbs(t *testing.T) {
	verbs := regexp.MustCompile(`%[-0-9.]*[a-zA-Z]`)
	for key, m := range messages {
		if m[0] == "" || m[1] == "" {
			t.Errorf("%s: missing a translation", key)
		}
		en := strings.Join(verbs.FindAllString(m[0], -1), " ")
		zh := strings.Join(verbs.FindAllString(m[1], -1), " ")
		if en != zh {
			t.Errorf("%s: verbs differ: %q vs %q", key, en, zh)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "zh_TW.UTF-8")
	if got := Resolve("auto"); got != ZhTW {
		t.Errorf("auto with zh_TW: %v", got)
	}
	if got := Resolve("en"); got != En {
		t.Errorf("explicit en: %v", got)
	}
	t.Setenv("LANG", "fr_FR.UTF-8")
	if got := Resolve(""); got != En {
		t.Errorf("fr_FR: %v", got)
	}
	t.Setenv("LANG", "en_US.UTF-8")
	if got := Resolve("zh-TW"); got != ZhTW {
		t.Errorf("explicit zh-TW: %v", got)
	}
}

func TestFromLocale(t *testing.T) {
	cases := map[string]Lang{
		"zh_TW.UTF-8": ZhTW, "zh-HK": ZhTW, "zh_Hant_TW": ZhTW, "zh_MO": ZhTW,
		"zh_CN.UTF-8": En, "C.UTF-8": En, "en_US": En,
	}
	for in, want := range cases {
		if got, ok := FromLocale(in); !ok || got != want {
			t.Errorf("%s: got %v %v", in, got, ok)
		}
	}
	if _, ok := FromLocale(""); ok {
		t.Error("empty locale should not resolve")
	}
}

func TestT(t *testing.T) {
	if got := ZhTW.T("ago_d", 3); got != "3 天前" {
		t.Errorf("zh: %q", got)
	}
	if got := En.T("ago_d", 3); got != "3d ago" {
		t.Errorf("en: %q", got)
	}
	if got := En.T("no-such-key"); got != "no-such-key" {
		t.Errorf("unknown key: %q", got)
	}
}
