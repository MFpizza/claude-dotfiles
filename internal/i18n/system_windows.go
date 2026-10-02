//go:build windows

package i18n

import "syscall"

var uiLanguage = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultUILanguage")

// systemLang reads the Windows display language; LANG is rarely set there.
func systemLang() Lang {
	if uiLanguage.Find() != nil {
		return En
	}
	id, _, _ := uiLanguage.Call()
	switch id & 0xffff {
	case 0x0404, 0x0c04, 0x1404: // zh-TW, zh-HK, zh-MO
		return ZhTW
	}
	return En
}
