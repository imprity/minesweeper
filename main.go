package main

import (
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

var ErrorLogger *log.Logger = log.New(os.Stderr, "ERROR : ", log.Lshortfile)
var InfoLogger *log.Logger = log.New(os.Stdout, "INFO : ", log.Lshortfile)

type App struct {
	Game *Game
}

func NewApp() *App {
	a := new(App)
	a.Game = NewGame()
	return a
}

func (a *App) Update() error {
	// ==========================
	// update global timer
	// ==========================
	UpdateGlobalTimer()

	// ==========================
	// update fps
	// ==========================
	eb.SetWindowTitle(fmt.Sprintf("FPS : %.2f", eb.ActualFPS()))

	if err := a.Game.Update(); err != nil {
		return err
	}

	return nil
}

func (a *App) Draw(dst *eb.Image) {
	a.Game.Draw(dst)
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	ScreenWidth = f64(outsideWidth)
	ScreenHeight = f64(outsideHeight)

	return a.Game.Layout(outsideWidth, outsideHeight)
}

func main() {
	LoadAssets()

	app := NewApp()

	eb.SetVsyncEnabled(true)
	eb.SetWindowSize(int(ScreenWidth), int(ScreenHeight))
	eb.SetWindowResizingMode(eb.WindowResizingModeEnabled)
	eb.SetWindowTitle("test")

	if err := eb.RunGame(app); err != nil {
		panic(err)
	}
}
