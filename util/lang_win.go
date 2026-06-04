//go:build windows

package util

import (
	"syscall"
)

func detectWindowsLanguage() string {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetUserDefaultUILanguage")

	ret, _, _ := proc.Call()
	langID := uint16(ret)

	// 한국어 (Korean) LANGID = 0x0412
	if langID == 0x0412 {
		return "ko"
	}

	return "en"
}
