package minesweeper

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebt "github.com/hajimehoshi/ebiten/v2/text/v2"
)

//go:embed fonts/dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf
//go:embed fonts/Sen/Sen-VariableFont_wght.ttf
//go:embed assets
var EmbeddedAssets embed.FS

var (
	TileSprite     Sprite
	UISprite       Sprite
	CursorSprite   Sprite
	DragSignSprite Sprite
)

var RetryButtonImage *eb.Image

var (
	ClearFace *ebt.GoTextFace

	FaceSource *ebt.GoTextFaceSource
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

	loadSprite := func(imgPath string, jsonPath string) Sprite {
		jsonBytes := mustLoadData(jsonPath)
		sprite, err := ParseSpriteJsonMetadata(bytes.NewReader(jsonBytes))
		if err != nil {
			ErrLogger.Fatalf("failed to load sprite json \"%s\": %v", jsonPath, err)
		}

		img, err := loadImage(imgPath)
		if err != nil {
			ErrLogger.Fatalf("failed to load sprite image \"%s\": %v", imgPath, err)
		}
		sprite.Image = img
		sprite.BoundsRect = img.Bounds()
		return sprite
	}

	// load sprites
	TileSprite = loadSprite(
		"assets/tile-spritesheet-50x50-5x5.png",
		"assets/tile-spritesheet-50x50-5x5.json",
	)

	UISprite = loadSprite(
		"assets/ui-spritesheet-100x100-5x5.png",
		"assets/ui-spritesheet-100x100-5x5.json",
	)

	CursorSprite = loadSprite(
		"assets/cursor.png",
		"assets/cursor.json",
	)

	DragSignSprite = loadSprite(
		"assets/drag-sign.png",
		"assets/drag-sign.json",
	)

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
		fontFile := mustLoadData("fonts/Sen/Sen-VariableFont_wght.ttf")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(fontFile))
		if err != nil {
			ErrLogger.Fatalf("failed to load font: %v", err)
		}
		FaceSource = faceSource
	}
	{
		clearFontFile := mustLoadData("fonts/dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf")
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

		const waterShaderImage1Path = "assets/noise1.jpg"
		const waterShaderImage2Path = "assets/noise2.jpg"

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

	// load HSVmod table
	loadHSVmodTable := func() error {
		jsonData, err := loadData("assets/HSVmod-table.json")
		if err != nil {
			return err
		}
		table, err := HSVmodTableFromJson(jsonData)
		if err != nil {
			return err
		}
		TheHSVmodTable = table
		return nil
	}
	if err := loadHSVmodTable(); err != nil {
		ErrLogger.Printf("failed to load HSVmod table: %v", err)
	}

	// load audios
	audioErrors := make(map[string]<-chan error)

	for _, src := range SoundSrcs {
		var file []byte
		var err error
		var convertedToMp3 bool = false

		if FlagHotReload {
			file, err = os.ReadFile(filepath.Join(hotReloadPath, src))
		} else {
			embedded := EmbeddedSounds[src]
			file = EmbeddedSoundsData[embedded.Offset : embedded.Offset+embedded.Len]
			convertedToMp3 = EmbeddedSounds[src].ConvertedToMp3WithFfmpeg
		}

		if err != nil {
			ErrLogger.Fatalf("failed to load %s: %v", src, err)
		}

		if convertedToMp3 {
			audioErrors[src] = RegisterAudio(src, file, ".mp3")
		} else {
			audioErrors[src] = RegisterAudio(src, file, filepath.Ext(src))
		}
	}

	for _, errChan := range audioErrors {
		if err := <-errChan; err != nil {
			ErrLogger.Fatalf("failed to register audio: %v", err)
		}
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

func SaveHSVmodTable() {
	InfoLogger.Print("saving HSVmod table")

	saveImp := func() error {
		jsonBytes, err := HSVmodTableToJson(TheHSVmodTable)
		if err != nil {
			return err
		}
		path, err := RelativePath("assets/HSVmod-table.json")
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
		ErrLogger.Printf("failed to HSVmod table: %v", err)
	}
}
