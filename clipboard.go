package main

import (
	"unicode/utf8"

	"golang.design/x/clipboard"
)

var TheClipboardManager struct {
	Initialized bool
}

func InitClipboardManager() {
	cm := &TheClipboardManager
	err := clipboard.Init()
	cm.Initialized = err == nil
}

func ClipboardWriteText(str string) {
	cm := &TheClipboardManager
	if cm.Initialized {
		clipboard.Write(clipboard.FmtText, []byte(str))
	}
}

func ClipboardReadText() string {
	cm := &TheClipboardManager
	if cm.Initialized {
		bytes := clipboard.Read(clipboard.FmtText)
		// basic sanity check
		if utf8.Valid(bytes) {
			return string(bytes)
		}
	}

	return ""
}
