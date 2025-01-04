package sound

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

type Context struct {
	sampleRate int
	otoContext *oto.Context

	audioMap     map[string][]byte
	audioMapLock sync.Mutex
}

func NewContext(sampleRate int) (*Context, chan struct{}, error) {
	c := new(Context)
	c.sampleRate = sampleRate
	c.audioMap = make(map[string][]byte)

	op := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}

	var readyChan chan struct{}
	var err error
	c.otoContext, readyChan, err = oto.NewContext(op)

	return c, readyChan, err
}

func (c *Context) SampleRate() int {
	return c.sampleRate
}

type decodeStream interface {
	io.ReadSeeker
	Length() int64
}

func (c *Context) registerAudio(
	audioName string,
	audioFile []byte,
	audioFileType string,
) <-chan error {
	errChan := make(chan error)

	go func() {
		var stream decodeStream
		var err error

		// NOTE: this is not a perfect way to determine the audio file type
		// since audio file can be in different container.
		//
		// But it is good enough for what we are trying to do
		switch strings.ToLower(audioFileType) {
		case ".ogg":
			stream, err = vorbis.DecodeWithSampleRate(c.SampleRate(), bytes.NewReader(audioFile))
		case ".wav":
			stream, err = wav.DecodeWithSampleRate(c.SampleRate(), bytes.NewReader(audioFile))
		case ".mp3":
			stream, err = mp3.DecodeWithSampleRate(c.SampleRate(), bytes.NewReader(audioFile))
		}
		if err != nil {
			errChan <- err
			close(errChan)
			return
		}

		decoded, err := io.ReadAll(stream)
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
	p.player = c.otoContext.NewPlayer(bytes.NewReader(audioBytes))
	p.sampleRate = c.SampleRate()

	return p
}

type Player struct {
	player     *oto.Player
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
	const bytesPerSample = 4

	o := int64(offset) * bytesPerSample * int64(p.sampleRate) / int64(time.Second)
	o -= o % bytesPerSample

	p.player.Seek(o, io.SeekStart)
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
