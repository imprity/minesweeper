package minesweeper

import (
	"fmt"
	"io"
	"time"

	"minesweeper/sound"
)

var _ = fmt.Printf

const SampleRate = 44100
const BytesPerSample = 4

var TheSoundManager struct {
	Context *sound.Context

	volume     float64
	prevVolume float64

	players []*Player

	tmpPlayers map[string][]*Player

	contextReadyChan chan struct{}
	contextReady     bool
}

func InitSound() {
	sm := &TheSoundManager

	sm.volume = 1
	sm.prevVolume = 1

	/*
		contextOp := oto.NewContextOptions{
			SampleRate:   SampleRate,
			ChannelCount: 2,
			Format:       oto.FormatSignedInt16LE,
			BufferSize:   time.Millisecond * 50,
		}
	*/

	var err error
	sm.Context, sm.contextReadyChan, err = sound.NewContext(SampleRate)

	if err != nil {
		ErrLogger.Fatalf("couldn't initialize sound %v", err)
	}

	sm.tmpPlayers = make(map[string][]*Player)
}

func UpdateSound() {
	sm := &TheSoundManager

	if !sm.contextReady {
		select {
		case <-sm.contextReadyChan:
			InfoLogger.Print("ready to make sound")
			sm.contextReady = true
		default:
			// pass
		}
	}

	// change volumes
	if sm.prevVolume != sm.volume {
		for _, player := range sm.players {
			player.player.SetVolume(player.volume * sm.volume)
		}

		for _, players := range sm.tmpPlayers {
			for _, player := range players {
				player.player.SetVolume(player.volume * sm.volume)
			}
		}
	}

	// TEST TEST TEST TEST TEST
	/*
		totalPlayers := 0
		for i, soundName := range SoundSrcs {
			DebugPrint(fmt.Sprintf("sound %02d", i), len(sm.tmpPlayers[soundName]))
			totalPlayers += len(sm.tmpPlayers[soundName])
		}
		totalPlayers += len(sm.players)
		DebugPrint("total players", totalPlayers)
	*/
	// TEST TEST TEST TEST TEST

	sm.prevVolume = sm.volume
}

func RegisterAudio(audioName string, audioFile []byte, audioFileType string) <-chan error {
	sm := &TheSoundManager
	return sm.Context.RegisterAudio(
		audioName,
		audioFile,
		audioFileType,
	)
}

func newPlayerInternal(audioName string) *Player {
	sm := &TheSoundManager

	player := new(Player)
	player.player = sm.Context.NewPlayer(audioName)
	player.volume = 1

	return player
}

func NewPlayer(audioName string) *Player {
	sm := &TheSoundManager

	player := newPlayerInternal(audioName)

	sm.players = append(sm.players, player)

	return player
}

func GlobalVolume() float64 {
	sm := &TheSoundManager

	return sm.volume
}

func SetGlobalVolume(volume float64) {
	sm := &TheSoundManager
	sm.volume = Clamp(volume, 0, 1)
}

func IsSoundReady() bool {
	sm := &TheSoundManager
	return sm.contextReady
}

func PlaySoundBytes(audioName string, volume float64) {
	if !IsSoundReady() {
		return
	}

	sm := &TheSoundManager

	for _, player := range sm.tmpPlayers[audioName] {
		if !player.IsPlaying() {
			player.Pause()
			player.SetPosition(0)
			player.SetVolume(volume)
			player.Play()
			return
		}
	}

	// all players are busy, create new one
	tmpP := newPlayerInternal(audioName)
	tmpP.SetVolume(volume)
	tmpP.Play()

	sm.tmpPlayers[audioName] = append(sm.tmpPlayers[audioName], tmpP)
}

type AudioDecoder interface {
	io.ReadSeeker
	Length() int64
}

type Player struct {
	player *sound.Player
	volume float64
}

func (p *Player) IsPlaying() bool {
	return p.player.IsPlaying()
}

func (p *Player) Pause() {
	p.player.Pause()
}

func (p *Player) Play() {
	p.player.Play()
}

func (p *Player) PlayIfReady() {
	if IsSoundReady() {
		p.player.Play()
	}
}

func (p *Player) SetPosition(offset time.Duration) {
	p.player.SetPosition(offset)
}

func (p *Player) SetVolume(volume float64) {
	volume = Clamp(volume, 0, 1)
	p.volume = volume
	p.player.SetVolume(p.volume * GlobalVolume())
}

func (p *Player) Volume() float64 {
	return p.volume
}
