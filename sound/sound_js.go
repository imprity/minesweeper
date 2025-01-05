//go:build js

package sound

import (
	"runtime"
	"syscall/js"
	"time"
	"fmt"
)

var jsFunctionMap map[string]js.Value = make(map[string]js.Value)

type Context struct {
	sampleRate int
}

func NewContext(sampleRate int) (*Context, chan struct{}, error) {
	c := new(Context)
	c.sampleRate = sampleRate

	jsFuncNames := []string {
		"initAudioContext",
		"newBufferFromAudioFile",
		"newPlayer",
		"playerIsPlaying",
		"playerPlay",
		"playerPause",
		"playerDuration",
		"playerPosition",
		"playerSetPosition",
		"playerVolume",
		"playerSetVolume",
	}

	for _, name := range jsFuncNames {
		jsFunctionMap[name] = js.Global().Get(name)
	}

	jsFunctionMap["initAudioContext"].Invoke(js.ValueOf(sampleRate))

	readyChan := make(chan struct{})
	var onReadyFunc js.Func
	onReadyFunc = js.FuncOf(func(this js.Value, args []js.Value) any{
		close(readyChan)
		onReadyFunc.Release()
		return nil
	})
	js.Global().Set("ON_AUDIO_RESUME", onReadyFunc)

	return c, readyChan, nil
}

func (c *Context) RegisterAudio(
	audioName string,
	audioFile []byte,
	audioFileType string,
) <-chan error {
	runtime.KeepAlive(audioFile)
	uint8Array := js.Global().Get("Uint8Array").New(len(audioFile))
	js.CopyBytesToJS(uint8Array, audioFile)
	arrayBuffer := uint8Array.Get("buffer")

	errChan := make(chan error, 1)

	var onDecoded js.Func
	onDecoded = js.FuncOf(func(this js.Value, args []js.Value) any {
		if args[0].Bool() {
			errChan <- nil
			close(errChan)
		} else {
			errChan <- fmt.Errorf("failed to register audio")
			close(errChan)
		}

		onDecoded.Release()
		return nil
	})

	jsFunctionMap["newBufferFromAudioFile"].Invoke(
		js.ValueOf(audioName),
		arrayBuffer,
		onDecoded,
	)

	return errChan
}

type Player struct {
	playerId js.Value
	playerIdInt int
}

func (c *Context) NewPlayer(audioName string) *Player {
	p := new(Player)
	p.playerId = jsFunctionMap["newPlayer"].Invoke(js.ValueOf(audioName))
	p.playerIdInt = p.playerId.Int()

	return p
}

func (p *Player) IsPlaying() bool {
	return jsFunctionMap["playerIsPlaying"].Invoke(p.playerId).Bool()
}

func (p *Player) Pause() {
	jsFunctionMap["playerPause"].Invoke(p.playerId)
}

func (p *Player) Play() {
	jsFunctionMap["playerPlay"].Invoke(p.playerId)
}

func (p *Player) SetPosition(offset time.Duration) {
	inSeconds := float64(offset) / float64(time.Second)
	jsFunctionMap["playerSetPosition"].Invoke(p.playerId, js.ValueOf(inSeconds))
}

func (p *Player) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	jsFunctionMap["playerSetVolume"].Invoke(p.playerId, js.ValueOf(volume))
}

func (p *Player) Volume() float64 {
	return jsFunctionMap["playerVolume"].Invoke(p.playerId).Float()
}
