//go:build alwaysdraw

package main

func init() {
	AlwaysDraw = true

	DebugPutsPersist("always draw", "true")
}
