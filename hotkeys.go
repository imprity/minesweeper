package main

import (
	eb "github.com/hajimehoshi/ebiten/v2"
)

const (
	ReloadAssetsKey   eb.Key = eb.KeyF5
	SaveColorTableKey eb.Key = eb.KeyF10

	ShowDebugConsoleKey = eb.KeyF1

	ShowMinesKey = eb.KeyF2

	SetToDecoBoardKey = eb.KeyF7
	InstantWinKey     = eb.KeyF8

	ShowColorPickerKey = eb.KeyF3
	ColorPickerUpKey   = eb.KeyW
	ColorPickerDownKey = eb.KeyS
)
