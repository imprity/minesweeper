package main

import (
	"io"

	eba "github.com/hajimehoshi/ebiten/v2/audio"
)

type AudioDecoder interface {
	io.ReadSeeker
	Length() int64
}

func PlaySoundBytes(sound []byte, volume float64) *eba.Player {
	player := AudioContext.NewPlayerFromBytes(sound)
	player.SetVolume(volume)
	player.Play()
	return player
}
