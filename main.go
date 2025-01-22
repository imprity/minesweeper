package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/silbinarywolf/preferdiscretegpu"

	eb "github.com/hajimehoshi/ebiten/v2"
)

var (
	ScreenWidth  float64 = 600
	ScreenHeight float64 = 600
)

var (
	AlwaysDraw   bool = false
	InDevMode    bool = false
	PprofEnabled bool = false
)

var redrawTimer time.Time

func SetRedraw() {
	redrawTimer = time.Now()
}

var ErrLogger *log.Logger = log.New(os.Stderr, "ERROR: ", log.Lshortfile)
var WarnLogger *log.Logger = log.New(os.Stderr, "WARN: ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO: ", log.Lshortfile)

var FlagHotReload bool

// Get version string using git.
// Version string format is :
//
//	branchName-tag-commitCount-hash
//
// For example:
//
//	main--148-c9b1d68
//
// If dirty:
//
//	main--148-c9b1d68-dirty
//
// If release:
//
//	main--148-c9b1d68-release
//
//go:embed git_version.txt
var GitVersionString string

func init() {
	GitVersionString = strings.TrimSpace(GitVersionString)
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
	if IsKeyJustPressed(ReloadAssetsKey) && InDevMode {
		LoadAssets()
	}

	if IsKeyJustPressed(SaveAssetsKey) && InDevMode {
		SaveColorTable()
		SaveBezierTable()
		SaveHSVmodTable()
	}

	// ==========================
	// debug showing
	// ==========================
	if IsKeyJustPressed(ShowDebugConsoleKey) && InDevMode {
		a.ShowDebugConsole = !a.ShowDebugConsole
	}

	a.Scene.Update()

	return nil
}

func (a *App) Draw(dst *eb.Image) {
	timeSinceRedraw := time.Now().Sub(redrawTimer)
	redraw := timeSinceRedraw < time.Millisecond*100

	if redraw || AlwaysDraw {
		a.Scene.Draw(dst)
	}

	if redraw || AlwaysDraw {
		DebugPuts("do redraw", "true ")
	} else {
		DebugPuts("do redraw", "false")
	}

	if a.ShowDebugConsole {
		DrawDebugMsgs(dst)
	}
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
	InfoLogger.Printf("git version: %s", GitVersionString)

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
	eb.SetTPS(120)

	op := &eb.RunGameOptions{
		// NOTE: I have no idea why, but I think there is a bug
		// that only happens when multithreaded...
		SingleThread: true,
	}

	if err := eb.RunGameWithOptions(app, op); err != nil {
		panic(err)
	}
}
