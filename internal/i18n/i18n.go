// Package i18n holds the user-facing text in English and Traditional Chinese.
package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Lang string

const (
	En   Lang = "en"
	ZhTW Lang = "zh-TW"
)

// Resolve turns the config's lang setting ("en", "zh-TW", "auto" or "") into a language.
func Resolve(setting string) Lang {
	switch setting {
	case string(En):
		return En
	case string(ZhTW):
		return ZhTW
	}
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if l, ok := FromLocale(os.Getenv(key)); ok {
			return l
		}
	}
	return systemLang()
}

// FromLocale maps a locale such as zh_TW.UTF-8 or zh-Hant to a language; ok is false
// for an empty locale.
func FromLocale(locale string) (Lang, bool) {
	l := strings.ToLower(strings.ReplaceAll(locale, "-", "_"))
	if l == "" {
		return "", false
	}
	for _, prefix := range []string{"zh_tw", "zh_hk", "zh_mo", "zh_hant"} {
		if strings.HasPrefix(l, prefix) {
			return ZhTW, true
		}
	}
	return En, true
}

// T returns the message for key in l, formatted with args; English is the fallback.
func (l Lang) T(key string, args ...any) string {
	m, ok := messages[key]
	if !ok {
		return key
	}
	s := m[0]
	if l == ZhTW {
		s = m[1]
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
