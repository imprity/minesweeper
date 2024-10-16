// golang.design/x/clipboard thinks
// crashing is the best solution despite it having a
// Init funciton that returns an error...

//go:build js || (!windows && !cgo)

package main

import (
)

var TheClipboardManager struct {
	Initialized bool
}

func InitClipboardManager() {
	InfoLogger.Print("initializing clipboard")
	ErrorLogger.Printf("clipboard is disabled")
}

func ClipboardWriteText(str string) {
}

func ClipboardReadText() string {
	return ""
}
