//go:build ignore

package main

import (
	ms "minesweeper"

	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	"github.com/sqweek/dialog"

	eb "github.com/hajimehoshi/ebiten/v2"
)

const (
	ShowGridKey   = eb.KeyG
	ShowOverlyKey = eb.KeyO

	SaveConfigurationKey = eb.KeyS
	LoadConfigurationKey = eb.KeyL
)

type TileType int

//go:embed itch_stuff/run-game-overlay.png
var overlayImageData []byte

const (
	TileTypeRevealed TileType = iota

	TileTypeBg

	TileTypeFlagNoBg
	TileTypeFlag

	TileTypeBomb

	TileTypeSize
)

type MouseMode int

const (
	MouseModeNone MouseMode = iota
	MouseModeShow
	MouseModeHide
)

type MockupBoard struct {
	DrawTile    ms.Array2D[bool]
	TileTypes   ms.Array2D[TileType]
	TileNumbers ms.Array2D[int]

	BoardWidth  int
	BoardHeight int

	BoardScreenWidth float64
}

type App struct {
	MockupBoard

	TileStyles ms.Array2D[ms.TileStyle]

	MouseMode MouseMode

	DrawGrid bool

	DrawOverly   bool
	OverlayImage *eb.Image

	BoardRect ms.FRectangle

	SaveConfigurationQueued bool
}

func NewApp() ms.Scene {
	a := new(App)

	a.BoardWidth = 30
	a.BoardHeight = 30

	a.BoardScreenWidth = 100

	a.DrawGrid = true
	a.DrawOverly = true

	image, _, err := image.Decode(bytes.NewReader(overlayImageData))
	if err != nil {
		ms.ErrLogger.Fatal("failed to create overlay image", err)
	}
	a.OverlayImage = eb.NewImageFromImage(image)

	a.DrawTile = ms.NewArray2D[bool](a.BoardWidth, a.BoardHeight)
	a.TileNumbers = ms.NewArray2D[int](a.BoardWidth, a.BoardHeight)
	a.TileTypes = ms.NewArray2D[TileType](a.BoardWidth, a.BoardHeight)
	a.TileStyles = ms.NewArray2D[ms.TileStyle](a.BoardWidth, a.BoardHeight)

	for x := range a.BoardWidth {
		for y := range a.BoardHeight {
			tileStyle := ms.NewTileStyle()
			a.TileStyles.Set(x, y, tileStyle)
		}
	}

	return a
}

func (a *App) Update() {
	ms.SetRedraw()

	if ms.IsKeyJustPressed(ShowGridKey) {
		a.DrawGrid = !a.DrawGrid
	}

	if ms.IsKeyJustPressed(ShowOverlyKey) {
		a.DrawOverly = !a.DrawOverly
	}

	if ms.IsKeyJustPressed(SaveConfigurationKey) {
		a.SaveConfigurationQueued = true
	}

	if ms.IsKeyJustPressed(LoadConfigurationKey) {
		if err := a.LoadConfiguration(); err != nil {
			ms.ErrLogger.Printf("failed to load configuration: %v", err)
		}
	}

	_, wheelY := eb.Wheel()

	a.BoardScreenWidth += wheelY * 30
	a.BoardScreenWidth = max(a.BoardScreenWidth, 50)

	a.BoardRect = ms.FRectWH(
		a.BoardScreenWidth,
		a.BoardScreenWidth/float64(a.BoardWidth)*float64(a.BoardHeight),
	)

	a.BoardRect = ms.CenterFRectangle(
		a.BoardRect,
		ms.ScreenWidth*0.5, ms.ScreenHeight*0.5,
	)

	mousePos := ms.CursorFPt()
	boardX, boardY := ms.MousePosToBoardPos(
		a.BoardRect,
		a.BoardWidth, a.BoardHeight,
		mousePos,
	)

	if 0 <= boardX && boardX < a.BoardWidth && 0 <= boardY && boardY < a.BoardHeight {
		if ms.IsMouseButtonJustPressed(eb.MouseButtonLeft) && a.MouseMode == MouseModeNone {
			drawTile := a.DrawTile.Get(boardX, boardY)

			if drawTile {
				a.MouseMode = MouseModeHide
			} else {
				a.MouseMode = MouseModeShow
			}
		}

		if ms.IsMouseButtonPressed(eb.MouseButtonLeft) {
			if a.MouseMode == MouseModeShow {
				if !a.DrawTile.Get(boardX, boardY) {
					a.DrawTile.Set(boardX, boardY, true)
					a.TileTypes.Set(boardX, boardY, TileTypeRevealed)
					a.TileNumbers.Set(boardX, boardY, 0)
				}
			} else if a.MouseMode == MouseModeHide {
				a.DrawTile.Set(boardX, boardY, false)
			}
		} else {
			a.MouseMode = MouseModeNone
		}

		if a.DrawTile.Get(boardX, boardY) {
			if ms.IsMouseButtonJustPressed(eb.MouseButtonMiddle) {
				tileType := a.TileTypes.Get(boardX, boardY)
				tileType += 1
				if tileType >= TileTypeSize {
					tileType = 0
				}

				if tileType != TileTypeRevealed {
					a.TileNumbers.Set(boardX, boardY, 0)
				}
				a.TileTypes.Set(boardX, boardY, tileType)
			}

			if ms.IsMouseButtonJustPressed(eb.MouseButtonRight) {
				number := a.TileNumbers.Get(boardX, boardY)
				number += 1
				if number >= 9 {
					number = 0
				}
				a.TileNumbers.Set(boardX, boardY, number)
			}
		}
	}

	// set tile style to tile type
	for x := range a.BoardWidth {
		for y := range a.BoardHeight {
			tileType := a.TileTypes.Get(x, y)
			style := ms.NewTileStyle()

			if !a.DrawTile.Get(x, y) {
				a.TileStyles.Set(x, y, style)
				continue
			}

			drawBg := false

			if tileType == TileTypeBg {
				drawBg = true
			}

			if tileType == TileTypeRevealed {
				drawBg = true

				style.DrawTile = true
				style.TileFillColor = ms.GetTileFillColor(
					a.BoardWidth, a.BoardHeight, x, y,
				)
				style.TileStrokeColor = ms.ColorTileRevealedStroke

				number := a.TileNumbers.Get(x, y)

				if 1 <= number && number <= 8 {
					style.DrawFg = true
					style.FgType = ms.TileFgTypeNumber
					style.FgNumber = number
					style.FgColor = ms.ColorTableGetNumber(number)
				}
			}

			if tileType == TileTypeFlag || tileType == TileTypeFlagNoBg {
				if tileType != TileTypeFlagNoBg {
					drawBg = true
				}
				style.DrawFg = true
				style.FgType = ms.TileFgTypeFlag
				style.FgColor = ms.ColorFlag
				style.FgFlagAnim = 1
			}

			if tileType == TileTypeBomb {
				drawBg = true

				style.BgFillColor = ms.GetBgFillColor(
					a.BoardWidth, a.BoardHeight, x, y,
				)
				style.BgBombAnim = 1
			}

			if drawBg {
				style.DrawBg = true
				style.BgFillColor = ms.GetBgFillColor(
					a.BoardWidth, a.BoardHeight, x, y,
				)
			}

			a.TileStyles.Set(x, y, style)
		}
	}
}

func (a *App) Draw(dst *eb.Image) {
	dst.Fill(ms.ColorBg)

	ms.DrawBoard(
		dst,
		a.BoardWidth, a.BoardHeight,
		a.BoardRect,
		a.TileStyles,

		false, 0, 0,
	)

	if a.SaveConfigurationQueued {
		a.SaveConfigurationQueued = false

		if err := a.SaveConfiguration(dst); err != nil {
			ms.ErrLogger.Printf("failed to save configuration: %v", err)
		} else {
			ms.InfoLogger.Printf("saved configuration")
		}
	}

	// draw overlay
	if a.DrawOverly {
		ms.BeginAntiAlias(false)
		ms.BeginFilter(eb.FilterLinear)
		ms.BeginMipMap(false)
		ms.DrawImage(dst, a.OverlayImage, nil)
		ms.EndAntiAlias()
		ms.EndFilter()
		ms.EndMipMap()
	}

	// draw grid
	if a.DrawGrid {
		tileW, tileH := ms.GetBoardTileSize(
			a.BoardRect, a.BoardWidth, a.BoardHeight)

		startX := a.BoardRect.Min.X
		startY := a.BoardRect.Min.Y

		for x := 0; x < a.BoardWidth+1; x++ {
			fx := startX + float64(x)*tileW

			ms.StrokeLine(
				dst,
				fx, a.BoardRect.Min.Y, fx, a.BoardRect.Max.Y,
				1,
				color.NRGBA{0, 100, 0, 100},
			)
		}

		for y := 0; y < a.BoardHeight+1; y++ {
			fy := startY + float64(y)*tileH

			ms.StrokeLine(
				dst,
				a.BoardRect.Min.X, fy, a.BoardRect.Max.X, fy,
				1,
				color.NRGBA{0, 255, 0, 50},
			)
		}
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) {
}

func (a *App) SaveConfiguration(dst *eb.Image) error {
	screenshotPath, err := ms.TakeScreenshot(dst, "./itch_stuff/")
	if err != nil {
		return err
	}

	mockup := a.MockupBoard

	jsonBytes, err := json.MarshalIndent(mockup, "", "    ")
	if err != nil {
		return err
	}

	dir, file := filepath.Split(screenshotPath)
	filename := file[:len(file)-len(filepath.Ext(file))]

	jsonFileName := filepath.Join(dir, filename+".json")

	err = os.WriteFile(jsonFileName, jsonBytes, 0644)
	if err != nil {
		return err
	}

	ms.InfoLogger.Printf("saved configuration %s", filename)

	return nil
}

func (a *App) LoadConfiguration() error {
	filename, err := dialog.File().
		SetStartDir(filepath.Clean("./itch_stuff/")).
		Filter("json files", "png", "json").
		Load()

	if err != nil {
		return err
	}

	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	err = json.Unmarshal(file, &a.MockupBoard)
	if err != nil {
		return err
	}

	ms.InfoLogger.Printf("loaded configuration %s", filepath.Base(filename))

	return nil
}

func DumpBoolBoard(board ms.Array2D[bool], trueChar, falseChar byte) {
	w, h := board.Width, board.Height

	textBuf := make([]byte, (w+1)*h)

	for y := range h {
		textBuf[(y+1)*(w+1)-1] = '\n'
	}

	setTextAt := func(x, y int, b byte) {
		textBuf[x+y*(w+1)] = b
	}

	for x := range w {
		for y := range h {
			if board.Get(x, y) {
				setTextAt(x, y, trueChar)
			} else {
				setTextAt(x, y, falseChar)
			}
		}
	}
	fmt.Println(string(textBuf))
}

func main() {
	if !ms.ScreenshotEnabled {
		ms.ErrLogger.Fatal("Screenshot is not enabled, rebuild with screen shot feature on")
	}

	ms.OverrideFirstScene(NewApp)
	ms.AppMain()
}
