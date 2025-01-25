//go:build !minedev

package minesweeper

import (
	eb "github.com/hajimehoshi/ebiten/v2"
)

type ResourceEditor struct {
	DoShow bool
}

func NewResourceEditor() *ResourceEditor {
	return new(ResourceEditor)
}

func (re *ResourceEditor) Update() {
}

func (re *ResourceEditor) Draw(dst *eb.Image) {
}

func DebugPrintf(key, fmtStr string, values ...any) {
}

func DebugPrint(key string, values ...any) {
}

func DebugPuts(key, value string) {
}

func DebugPrintfPersist(key, fmtStr string, values ...any) {
}

func DebugPrintPersist(key string, values ...any) {
}

func DebugPutsPersist(key, value string) {
}

func DrawDebugMsgs(dst *eb.Image) {
}

func ClearDebugMsgs() {
}
