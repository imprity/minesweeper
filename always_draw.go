//go:build alwaysdraw

package main

func init() {
	alwaysDraw = true

	DebugPutsPersist("always draw", "true")
}
