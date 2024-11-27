package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/silbinarywolf/preferdiscretegpu"
	"net/http"
	_ "net/http/pprof"

	eb "github.com/hajimehoshi/ebiten/v2"
	eba "github.com/hajimehoshi/ebiten/v2/audio"
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var AudioContext *eba.Context

var ErrLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

var FlagHotReload bool
var FlagPProf bool

func init() {
	flag.BoolVar(&FlagHotReload, "hot", false, "enable hot reloading")
	flag.BoolVar(&FlagPProf, "pprof", false, "enable pprof")
}

type Scene interface {
	Update()
	Draw(dst *eb.Image)
	Layout(outsideWidth, outsideHeight int)
}

type App struct {
	ShowDebugConsole bool
	Scene            Scene
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
	a.Scene.Draw(dst)

	if a.ShowDebugConsole {
		DrawDebugMsgs(dst)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	ScreenWidth = f64(outsideWidth)
	ScreenHeight = f64(outsideHeight)

	a.Scene.Layout(outsideWidth, outsideHeight)

	return outsideWidth, outsideHeight
}

func main() {
	flag.Parse()

	if FlagPProf {
		go func() {
			InfoLogger.Print("initializing pprof")
			InfoLogger.Print(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	AudioContext = eba.NewContext(44100)

	InitClipboardManager()

	LoadAssets()

	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("Minesweeper")
	eb.SetScreenClearedEveryFrame(false)

	op := &eb.RunGameOptions{
		SingleThread: true,
	}

	if err := eb.RunGameWithOptions(app, op); err != nil {
		panic(err)
	}
}
