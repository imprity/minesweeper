package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/silbinarywolf/preferdiscretegpu"

	eb "github.com/hajimehoshi/ebiten/v2"
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var (
	alwaysDraw         bool = false
	redrawFrameCounter int  = 4
)

func SetRedraw() {
	redrawFrameCounter = 4
}

var ErrLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

var FlagHotReload bool

func init() {
	flag.BoolVar(&FlagHotReload, "hot", false, "enable hot reloading")
}

type Scene interface {
	Update()
	Draw(dst *eb.Image)
	Layout(outsideWidth, outsideHeight int)
}

type App struct {
	ShowDebugConsole bool

	Scene Scene
}

func NewApp() *App {
	a := new(App)
	a.Scene = NewGameUI()
	return a
}

func (a *App) Update() error {
	ClearDebugMsgs()

	// ==========================
	// update global timer
	// ==========================
	UpdateGlobalTimer()

	UpdateSound()

	fpsStr := fmt.Sprintf("%.2f", eb.ActualFPS())
	tpsStr := fmt.Sprintf("%.2f", eb.ActualTPS())

	// ==========================
	// update windows title
	// ==========================
	eb.SetWindowTitle("Minesweeper FPS: " + fpsStr + " TPS: " + tpsStr)

	// ==========================
	// DebugPrint
	// ==========================
	DebugPuts("FPS", fpsStr)
	DebugPuts("TPS", tpsStr)

	// ==========================
	// asset loading and saving
	// ==========================
	if IsKeyJustPressed(ReloadAssetsKey) {
		LoadAssets()
	}

	if IsKeyJustPressed(SaveAssetsKey) {
		SaveColorTable()
		SaveBezierTable()
		SaveHSVmodTable()
	}

	// ==========================
	// debug showing
	// ==========================
	if IsKeyJustPressed(ShowDebugConsoleKey) {
		a.ShowDebugConsole = !a.ShowDebugConsole
	}

	a.Scene.Update()

	return nil
}

func (a *App) Draw(dst *eb.Image) {
	if redrawFrameCounter > 0 || alwaysDraw {
		a.Scene.Draw(dst)
	}

	if redrawFrameCounter > 0 || alwaysDraw {
		DebugPuts("do redraw", "true ")
	} else {
		DebugPuts("do redraw", "false")
	}

	if a.ShowDebugConsole {
		DrawDebugMsgs(dst)
	}

	redrawFrameCounter--
	redrawFrameCounter = max(redrawFrameCounter, 0)
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	if int(ScreenWidth) != outsideWidth || int(ScreenHeight) != outsideHeight {
		SetRedraw()
	}

	ScreenWidth = f64(outsideWidth)
	ScreenHeight = f64(outsideHeight)

	a.Scene.Layout(outsideWidth, outsideHeight)

	return outsideWidth, outsideHeight
}

func main() {
	flag.Parse()

	InitClipboardManager()

	InitSound()

	LoadAssets()

	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("Minesweeper")
	eb.SetScreenClearedEveryFrame(false)

	op := &eb.RunGameOptions{
		// NOTE: I have no idea why, but I think there is a bug
		// that only happens when multithreaded...
		SingleThread: true,
	}

	if err := eb.RunGameWithOptions(app, op); err != nil {
		panic(err)
	}
}
