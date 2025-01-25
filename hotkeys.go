package minesweeper

import (
	eb "github.com/hajimehoshi/ebiten/v2"
)

const (
	ReloadAssetsKey eb.Key = eb.KeyF5
	SaveAssetsKey   eb.Key = eb.KeyF10

	ShowDebugConsoleKey = eb.KeyF1

	SetToDecoBoardKey = eb.KeyF7
	InstantWinKey     = eb.KeyF8

	ShowResourceEditorKey = eb.KeyF3
	ResourceEditorUpKey   = eb.KeyW
	ResourceEditorDownKey = eb.KeyS

	ResetBoardKey       eb.Key = eb.KeyR
	ResetToSameBoardKey eb.Key = eb.KeyT

	ScreenshotKey eb.Key = eb.KeyP
)
