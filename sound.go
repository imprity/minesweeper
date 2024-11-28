package main

import (
	"io"

	eba "github.com/hajimehoshi/ebiten/v2/audio"
)

var TheSoundManager struct {
	Context *eba.Context

	Volume     float64
	prevVolume float64

	tmpPlayers       []*eba.Player
	tmpPlayerVolumes []float64
}

func InitSound() {
	sm := &TheSoundManager

	sm.Volume = 1
	sm.prevVolume = 1

	sm.Context = eba.NewContext(44100)
}

func UpdateSound() {
	sm := &TheSoundManager

	// remove stopped temp players
	for i, player := range sm.tmpPlayers {
		if player != nil && !player.IsPlaying() {
			sm.tmpPlayers[i] = nil
		}
	}

	// change volumes
	if sm.prevVolume != sm.Volume {
		for i, player := range sm.tmpPlayers {
			if player != nil {
				player.SetVolume(sm.tmpPlayerVolumes[i] * sm.Volume)
			}
		}
	}
	sm.prevVolume = sm.Volume
}

func SampleRate() int {
	sm := &TheSoundManager

	return sm.Context.SampleRate()
}

func GlobalVolume() float64 {
	sm := &TheSoundManager

	return sm.Volume
}

func SetGlobalVolume(volume float64) {
	sm := &TheSoundManager
	sm.Volume = Clamp(volume, 0, 1)
}

func PlaySoundBytes(sound []byte, volume float64) {
	sm := &TheSoundManager

	player := sm.Context.NewPlayerFromBytes(sound)
	needToAppend := true
	for i := range sm.tmpPlayers {
		if sm.tmpPlayers[i] == nil {
			sm.tmpPlayers[i] = player
			sm.tmpPlayerVolumes[i] = volume
			needToAppend = false
			break
		}
	}

	if needToAppend {
		sm.tmpPlayers = append(sm.tmpPlayers, player)
		sm.tmpPlayerVolumes = append(sm.tmpPlayerVolumes, volume)
	}

	player.SetVolume(volume * sm.Volume)
	player.Play()
}

type AudioDecoder interface {
	io.ReadSeeker
	Length() int64
}
