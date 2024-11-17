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
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

var FlagHotReload bool
var FlagPProf bool

func init() {
	flag.BoolVar(&FlagHotReload, "hot", false, "enable hot reloading")
	flag.BoolVar(&FlagPProf, "pprof", false, "enable pprof")
}

type App struct {
	ShowDebugConsole bool
	Scene            *GameUI
	//Scene           *LayoutTest
}

func NewApp() *App {
	a := new(App)
	a.Scene = NewGameUI()
	//a.Scene = NewLayoutTest()
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
	DebugPrint("FPS", fpsStr)
	DebugPrint("TPS", tpsStr)

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

	InitClipboardManager()

	LoadAssets()

	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("Minesweeper")

	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}
