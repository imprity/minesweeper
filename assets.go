package main

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

//go:embed dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf
//go:embed assets/COOPBL.TTF
//go:embed assets
var EmbeddedAssets embed.FS

var TileSprite Sprite

var (
	ClearFace *ebt.GoTextFace
	DecoFace  *ebt.GoTextFace
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
		ErrorLogger.Fatalf("failed to create missing shader: %v", err)
	}
}

func LoadAssets() {
	InfoLogger.Print("loading assets")

	var hotReloadPath string

	if FlagHotReload {
		var err error
		if hotReloadPath, err = RelativePath("./"); err != nil {
			ErrorLogger.Fatalf("failed to get assets path: %v", err)
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
			ErrorLogger.Fatalf("failed to load %s: %v", filepath, err)
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

	// load tile sprite
	{
		tileSpriteJson := mustLoadData("assets/spritesheet-100x100-5x5.json")
		tileSprite, err := ParseSpriteJsonMetadata(bytes.NewReader(tileSpriteJson))
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite: %v", err)
		}

		image, err := loadImage("assets/spritesheet-100x100-5x5.png")
		if err != nil {
			ErrorLogger.Fatalf("failed to load sprite: %v", err)
		}

		tileSprite.Image = image
		TileSprite = tileSprite
	}

	// load fonts
	{
		decoFontFile := mustLoadData("assets/COOPBL.TTF")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(decoFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font: %v", err)
		}
		DecoFace = &ebt.GoTextFace{
			Source: faceSource,
			Size:   64,
		}
	}
	{
		clearFontFile := mustLoadData("dejavu-fonts-ttf-2.37/ttf/DejaVuSansMono.ttf")
		faceSource, err := ebt.NewGoTextFaceSource(bytes.NewReader(clearFontFile))
		if err != nil {
			ErrorLogger.Fatalf("failed to load font: %v", err)
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
			ErrorLogger.Printf("failed to load WaterShader: %v", err)
			WaterShader = MissingShader
		} else {
			WaterShader = shader
		}

		const waterShaderImage1Path = "assets/noise6.png"
		const waterShaderImage2Path = "assets/noise8.png"

		if WaterShaderImage1, err = loadImage(waterShaderImage1Path); err != nil {
			ErrorLogger.Fatalf("failed to load image %v: %v", waterShaderImage1Path, err)
		}

		if WaterShaderImage2, err = loadImage(waterShaderImage2Path); err != nil {
			ErrorLogger.Fatalf("failed to load image %v: %v", waterShaderImage2Path, err)
		}

		img1Rect := WaterShaderImage1.Bounds()
		img2Rect := WaterShaderImage2.Bounds()

		if img1Rect.Dx() != img1Rect.Dx() || img2Rect.Dy() != img2Rect.Dy() {
			ErrorLogger.Fatalf("WaterShaderImage1 and WaterShaderImage2 has different sizes")
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
		ColorTable = table
		return nil
	}
	if err := loadColorTable(); err != nil {
		ErrorLogger.Printf("failed to load color table: %v", err)
	}
}

func SaveColorTable() {
	InfoLogger.Print("saving color table")

	saveImp := func() error {
		jsonBytes, err := ColorTableToJson(ColorTable)
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
		ErrorLogger.Printf("failed to save color table: %v", err)
	}
}
