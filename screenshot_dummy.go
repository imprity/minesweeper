//go:build !screenshot

package minesweeper

import (
	eb "github.com/hajimehoshi/ebiten/v2"
)

func TakeScreenshot(img *eb.Image) (string, error) {
	return "", nil
}
