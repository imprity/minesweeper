//go:build minepprof

package main

import (
	"net/http"
	_ "net/http/pprof"
)

func init() {
	PprofEnabled = true

	DebugPutsPersist("pprof", "true")
	go func() {
		InfoLogger.Print("initializing pprof")
		InfoLogger.Print(http.ListenAndServe("localhost:6060", nil))
	}()
}
