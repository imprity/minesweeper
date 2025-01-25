//go:build alwaysdraw

package minesweeper

func init() {
	AlwaysDraw = true

	DebugPutsPersist("always draw", "true")
}
