// golang.design/x/clipboard thinks
// crashing is the best solution despite it having a
// Init funciton that returns an error...

//go:build !(!js && !(!windows && !cgo) && minedev)

package main

var TheClipboardManager struct {
	Initialized bool
}

func InitClipboardManager() {
	InfoLogger.Print("initializing clipboard")
	ErrLogger.Printf("clipboard is disabled")
}

func ClipboardWriteText(str string) {
}

func ClipboardReadText() string {
	return ""
}
