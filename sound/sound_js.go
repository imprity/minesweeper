//go:build js

package sound

import (
	"runtime"
	"syscall/js"
	"time"
)

var jsFunctionMap map[string]js.Value = make(map[string]js.Value)

type Context struct {
	sampleRate int
}

func NewContext(sampleRate int) (*Context, chan struct{}, error) {
	c := new(Context)
	c.sampleRate = sampleRate

	jsFuncNames := []string{
		"initAudioContext",
		"newBufferFromAudioFile",
		"newBufferFromUndecodedAudioFile",
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
	onReadyFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		close(readyChan)
		onReadyFunc.Release()
		return nil
	})
	js.Global().Set("ON_AUDIO_RESUME", onReadyFunc)

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
	runtime.KeepAlive(audioFile)
	uint8Array := js.Global().Get("Uint8Array").New(len(audioFile))
	js.CopyBytesToJS(uint8Array, audioFile)
	arrayBuffer := uint8Array.Get("buffer")

	errChan := make(chan error, 1)

	var onDecoded js.Func
	onDecoded = js.FuncOf(func(this js.Value, args []js.Value) any {
		doRelease := false

		if args[0].Bool() {
			doRelease = true
			errChan <- nil
			close(errChan)
		} else {
			// errChan <- fmt.Errorf("failed to register audio")
			// close(errChan)

			// native javascript decoder failed
			// let's try golang one
			warnLogger.Print("native javascript decoder failed, trying native go decoder")
			go func() {
				defer func() {
					onDecoded.Release()
				}()
				decoded, err := decodeAudioF32(audioFile, audioFileType, c.SampleRate())
				if err != nil {
					errChan <- err
					close(errChan)
					return
				}

				// web audio expect samples for each channel
				// but our decoders combine and samples and send it as one array
				// so we need to separate it
				const channelCount = 2 // channel count is always 2
				const f32BitDepth = 4

				var channelDatas [][]byte

				for ci := range channelCount {
					channel := make([]byte, len(decoded)/channelCount)

					readOffset := 0
					readOffset += ci * f32BitDepth
					writeOffset := 0

					for readOffset+(f32BitDepth-1) < len(decoded) && writeOffset < len(channel) {
						channel[writeOffset] = decoded[readOffset]
						channel[writeOffset+1] = decoded[readOffset+1]
						channel[writeOffset+2] = decoded[readOffset+2]
						channel[writeOffset+3] = decoded[readOffset+3]

						readOffset += f32BitDepth * channelCount
						writeOffset += f32BitDepth
					}

					channelDatas = append(channelDatas, channel)
				}

				var channelDatasJs []any

				for ci := range channelCount {
					data := channelDatas[ci]
					runtime.KeepAlive(data)
					uint8Arr := js.Global().Get("Uint8Array").New(len(data))
					js.CopyBytesToJS(uint8Arr, data)
					uint8ArrBuf := uint8Arr.Get("buffer")
					jsArray := js.Global().Get("Float32Array").New(
						uint8ArrBuf,
						uint8Arr.Get("byteOffset"),
						uint8Arr.Get("byteLength").Int()/4,
					)

					channelDatasJs = append(channelDatasJs, jsArray)
				}

				jsFunctionMap["newBufferFromUndecodedAudioFile"].Invoke(
					js.ValueOf(audioName),
					js.ValueOf(channelDatasJs),
					js.ValueOf(c.SampleRate()),
				)

				errChan <- nil
				close(errChan)
			}()
		}

		if doRelease {
			onDecoded.Release()
		}
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
	playerId    js.Value
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
