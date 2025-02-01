//go:build !js

package sound

import (
	"bytes"
	"sync"
	"time"

	eba "github.com/hajimehoshi/ebiten/v2/audio"
)

type Context struct {
	sampleRate int
	context    *eba.Context

	audioMap     map[string][]byte
	audioMapLock sync.Mutex
}

func NewContext(sampleRate int) (*Context, chan struct{}, error) {
	c := new(Context)
	c.sampleRate = sampleRate
	c.audioMap = make(map[string][]byte)

	var readyChan = make(chan struct{})

	// it'll be ready on start since it's not on browser
	close(readyChan)

	c.context = eba.NewContext(sampleRate)

	return c, readyChan, nil
}

func (c *Context) SampleRate() int {
	return c.sampleRate
}

func (c *Context) RegisterAudio(
	audioName string,
	audioFile []byte,
	audioFileType string,
) <-chan error {
	errChan := make(chan error)

	go func() {
		decoded, err := decodeAudioF32(audioFile, audioFileType, c.SampleRate())
		if err != nil {
			errChan <- err
			close(errChan)
			return
		}

		c.audioMapLock.Lock()
		c.audioMap[audioName] = decoded
		c.audioMapLock.Unlock()

		errChan <- nil
		close(errChan)
	}()

	return errChan
}

func (c *Context) NewPlayer(audioName string) *Player {
	c.audioMapLock.Lock()
	audioBytes := c.audioMap[audioName]
	c.audioMapLock.Unlock()

	p := new(Player)
	var err error
	p.player, err = c.context.NewPlayerF32(bytes.NewReader(audioBytes))

	if err != nil {
		// TODO: actually handle error instead of catching fire lol
		errLogger.Fatalf("NewPlayer failed for %s : %v", audioName, err)
	}

	p.sampleRate = c.SampleRate()

	return p
}

type Player struct {
	player     *eba.Player
	sampleRate int
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

func (p *Player) SetPosition(offset time.Duration) {
	p.player.SetPosition(offset)
}

func (p *Player) SetVolume(volume float64) {
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	p.player.SetVolume(volume)
}

func (p *Player) Volume() float64 {
	return p.player.Volume()
}
