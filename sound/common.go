package sound

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"

	eba "github.com/hajimehoshi/ebiten/v2/audio"
)

var errLogger = log.New(os.Stderr, "[ FAIL ]: ", log.Lshortfile)
var warnLogger = log.New(os.Stderr, "[ WARN ]: ", log.Lshortfile)

type decodeStream interface {
	io.ReadSeeker
	Length() int64
	SampleRate() int
}

func decodeAudioImpl(
	audioFile []byte,
	audioFileType string,
	sampleRate int,
	decodeToF32 bool,
) ([]byte, error) {
	var stream decodeStream
	var err error

	// NOTE: this is not a perfect way to determine the audio file type
	// since audio file can be in different container.
	//
	// But it is good enough for what we are trying to do
	switch strings.ToLower(audioFileType) {
	case ".ogg":
		if decodeToF32 {
			stream, err = vorbis.DecodeF32(bytes.NewReader(audioFile))
		} else {
			stream, err = vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(audioFile))
		}
	case ".wav":
		if decodeToF32 {
			stream, err = wav.DecodeF32(bytes.NewReader(audioFile))
		} else {
			stream, err = wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(audioFile))
		}
	case ".mp3":
		if decodeToF32 {
			stream, err = mp3.DecodeF32(bytes.NewReader(audioFile))
		} else {
			stream, err = mp3.DecodeWithSampleRate(sampleRate, bytes.NewReader(audioFile))
		}
	}
	if err != nil {
		return nil, err
	}

	var readSeeker io.ReadSeeker = stream

	if decodeToF32 {
		readSeeker = eba.ResampleF32(
			stream, stream.Length(), stream.SampleRate(), sampleRate)
	}

	decoded, err := io.ReadAll(readSeeker)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func decodeAudio(
	audioFile []byte,
	audioFileType string,
	sampleRate int,
) ([]byte, error) {
	return decodeAudioImpl(
		audioFile,
		audioFileType,
		sampleRate,
		false,
	)
}

func decodeAudioF32(
	audioFile []byte,
	audioFileType string,
	sampleRate int,
) ([]byte, error) {
	return decodeAudioImpl(
		audioFile,
		audioFileType,
		sampleRate,
		true,
	)
}
