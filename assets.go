package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

//go:embed dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf
//go:embed tmps/converted
//go:embed assets
var EmbeddedAssets embed.FS

var TileSprite Sprite

var RetryButtonImage *eb.Image

var (
	ClearFace *ebt.GoTextFace

	RegularFace *ebt.GoTextFace
	BoldFace    *ebt.GoTextFace
)

var (
	WaterShader *eb.Shader

	// these two must have same dimmensions
	WaterShaderImage1 *eb.Image
	WaterShaderImage2 *eb.Image
)

var WhiteImage *eb.Image
var MissingImage *eb.Image

var MissingShader *eb.Shader

var SoundEffects [][]byte

func init() {
	// ===================
	// create WhiteImage
	// ===================
	whiteImg := image.NewNRGBA(RectWH(3, 3))
	for x := range 3 {
		for y := range 3 {
			whiteImg.Set(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	wholeWhiteImage := eb.NewImageFromImage(whiteImg)
	WhiteImage = wholeWhiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*eb.Image)

	// ===================
	// create MissingImage
	// ===================

	// create checker board
	checkerBoard := image.NewNRGBA(RectWH(6, 6))
	for x := range 6 {
		for y := range 6 {
			doPurple := false
			if x <= 2 {
				if y <= 2 {
					doPurple = true
				}
			} else {
				if y > 2 {
					doPurple = true
				}
			}

			if doPurple {
				checkerBoard.Set(x, y, color.NRGBA{255, 0, 255, 255})
			} else {
				checkerBoard.Set(x, y, color.NRGBA{0, 0, 0, 255})
			}
		}
	}
	MissingImage = eb.NewImageFromImage(checkerBoard)

	// ===================
	// create MissingShader
	// ===================
	const missingShaderCode string = `
		//kage:unit pixels

		package main

		func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
			return color
		}`

	if shader, err := eb.NewShader([]byte(missingShaderCode)); err == nil {
		MissingShader = shader
	} else {
		ErrLogger.Fatalf("failed to create missing shader: %v", err)
	}
}

func LoadAssets() {
	InfoLogger.Print("loading assets")

	var hotReloadPath string

	if FlagHotReload {
		var err error
		if hotReloadPath, err = RelativePath("./"); err != nil {
			ErrLogger.Fatalf("failed to get assets path: %v", err)
		}
	}

	loadData := func(filePath string) ([]byte, error) {
		var data []byte
		var err error
		if FlagHotReload {
			data, err = os.ReadFile(filepath.Join(hotReloadPath, filePath))
		} else {
			data, err = fs.ReadFile(EmbeddedAssets, filePath)
		}
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	mustLoadData := func(filepath string) []byte {
		data, err := loadData(filepath)
		if err != nil {
			ErrLogger.Fatalf("failed to load %s: %v", filepath, err)
		}
		return data
	}

	loadImage := func(filepath string) (*eb.Image, error) {
		imageFile, err := loadData(filepath)
		if err != nil {
			return nil, err
		}

		image, _, err := image.Decode(bytes.NewReader(imageFile))
		if err != nil {
			return nil, err
		}

		return eb.NewImageFromImage(image), nil
	}

	loadAudioBytes := func(path string) ([]byte, error) {
		audioFile, err := loadData(path)
		if err != nil {
			return nil, err
		}

		var decoder AudioDecoder

		if CheckFileExt(path, ".wav") {
			decoder, err = wav.DecodeWithSampleRate(SampleRate(), bytes.NewReader(audioFile))
			if err != nil {
				return nil, err
			}
		} else if CheckFileExt(path, ".mp3") {
			decoder, err = mp3.DecodeWithSampleRate(SampleRate(), bytes.NewReader(audioFile))
			if err != nil {
				return nil, err
			}
		} else if CheckFileExt(path, ".ogg") {
			decoder, err = vorbis.DecodeWithSampleRate(SampleRate(), bytes.NewReader(audioFile))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("usupported file format: %s", filepath.Ext(path))
		}

		audioBytes, err := io.ReadAll(decoder)
		if err != nil {
			return nil, err
		}

		return audioBytes, nil
	}

	mustLoadAudioBytes := func(filepath string) []byte {
		audio, err := loadAudioBytes(filepath)
		if err != nil {
			ErrLogger.Fatalf("failed to load %s: %v", filepath, err)
		}
		return audio
	}

	// load tile sprite
	{
		tileSpriteJson := mustLoadData("assets/spritesheet-100x100-5x5.json")
		tileSprite, err := ParseSpriteJsonMetadata(bytes.NewReader(tileSpriteJson))
		if err != nil {
			ErrLogger.Fatalf("failed to load sprite: %v", err)
		}

		image, err := loadImage("assets/spritesheet-100x100-5x5.png")
		if err != nil {
			ErrLogger.Fatalf("failed to load sprite: %v", err)
		}

		tileSprite.Image = image
		TileSprite = tileSprite
	}

	// load RetryButtonImage
	{
		image, err := loadImage("assets/retry-button.png")
		if err != nil {
			ErrLogger.Fatalf("failed to load retry button image: %v", err)
		}
		RetryButtonImage = image
	}

	// load fonts
	{
		fontFile := mustLoadData("assets/Sen-VariableFont_wght.ttf")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(fontFile))
		if err != nil {
			ErrLogger.Fatalf("failed to load font: %v", err)
		}

		BoldFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}

		BoldFace.SetVariation(ebt.MustParseTag("wght"), 700)

		RegularFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}

		RegularFace.SetVariation(ebt.MustParseTag("wght"), 400)
	}
	{
		clearFontFile := mustLoadData("dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(clearFontFile))
		if err != nil {
			ErrLogger.Fatalf("failed to load font: %v", err)
		}

		ClearFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}

	// load water shader
	{
		shaderFile := mustLoadData("assets/water_shader.go")
		shader, err := eb.NewShader(shaderFile)
		if err != nil {
			ErrLogger.Printf("failed to load WaterShader: %v", err)
			WaterShader = MissingShader
		} else {
			WaterShader = shader
		}

		const waterShaderImage1Path = "assets/noise1.png"
		const waterShaderImage2Path = "assets/noise2.png"

		if WaterShaderImage1, err = loadImage(waterShaderImage1Path); err != nil {
			ErrLogger.Fatalf("failed to load image %v: %v", waterShaderImage1Path, err)
		}

		if WaterShaderImage2, err = loadImage(waterShaderImage2Path); err != nil {
			ErrLogger.Fatalf("failed to load image %v: %v", waterShaderImage2Path, err)
		}

		img1Rect := WaterShaderImage1.Bounds()
		img2Rect := WaterShaderImage2.Bounds()

		if img1Rect.Dx() != img1Rect.Dx() || img2Rect.Dy() != img2Rect.Dy() {
			ErrLogger.Fatalf("WaterShaderImage1 and WaterShaderImage2 has different sizes")
		}
	}

	// load color table
	loadColorTable := func() error {
		jsonData, err := loadData("assets/color-table.json")
		if err != nil {
			return err
		}
		table, err := ColorTableFromJson(jsonData)
		if err != nil {
			return err
		}
		TheColorTable = table
		return nil
	}
	if err := loadColorTable(); err != nil {
		ErrLogger.Printf("failed to load color table: %v", err)
	}

	// load bezier table
	loadBezierTable := func() error {
		jsonData, err := loadData("assets/bezier-table.json")
		if err != nil {
			return err
		}
		table, err := BezierTableFromJson(jsonData)
		if err != nil {
			return err
		}
		TheBezierTable = table
		return nil
	}
	if err := loadBezierTable(); err != nil {
		ErrLogger.Printf("failed to load bezier table: %v", err)
	}

	// load audios
	{
		sfs := SoundEffects

		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch1.ogg"))                     // 0
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch2.ogg"))                     // 1
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch3.ogg"))                     // 2
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch4.ogg"))                     // 3
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch5.ogg"))                     // 4
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/GUI_Sound_Effects_by_Lokif/misc_menu_4.ogg")) // 5
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/GUI_Sound_Effects_by_Lokif/save.ogg"))        // 6
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/GUI_Sound_Effects_by_Lokif/misc_sound.ogg"))  // 7
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/GUI_Sound_Effects_by_Lokif/negative_2.ogg"))  // 8
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/GUI_Sound_Effects_by_Lokif/positive.ogg"))    // 9
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/click1.ogg"))                      // 10
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/interface/dustbin.ogg"))                      // 11
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/UI_SFX_Set/switch38.ogg"))                    // 12
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/krank_sounds/summer/unlink.ogg"))             // 13
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/interface/cut.ogg"))                          // 14
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/krank_sounds/summer/link.ogg"))               // 15
		sfs = append(sfs, mustLoadAudioBytes("tmps/converted/krank_sounds/summer/unlink.ogg"))             // 16

		SoundEffects = sfs
	}
}

func SaveColorTable() {
	InfoLogger.Print("saving color table")

	saveImp := func() error {
		jsonBytes, err := ColorTableToJson(TheColorTable)
		if err != nil {
			return err
		}
		path, err := RelativePath("assets/color-table.json")
		if err != nil {
			return err
		}
		err = os.WriteFile(path, jsonBytes, 0664)
		if err != nil {
			return err
		}

		return nil
	}

	if err := saveImp(); err != nil {
		ErrLogger.Printf("failed to save color table: %v", err)
	}
}

func SaveBezierTable() {
	InfoLogger.Print("saving bezier table")

	saveImp := func() error {
		jsonBytes, err := BezierTableToJson(TheBezierTable)
		if err != nil {
			return err
		}
		path, err := RelativePath("assets/bezier-table.json")
		if err != nil {
			return err
		}
		err = os.WriteFile(path, jsonBytes, 0664)
		if err != nil {
			return err
		}

		return nil
	}

	if err := saveImp(); err != nil {
		ErrLogger.Printf("failed to save color table: %v", err)
	}
}
