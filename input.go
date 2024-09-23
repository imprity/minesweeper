package main

import (
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
)

func IsMouseButtonPressed(button eb.MouseButton) bool {
	return eb.IsMouseButtonPressed(button)
}

func IsMouseButtonJustPressed(button eb.MouseButton) bool {
	return ebi.IsMouseButtonJustPressed(button)
}

var mouseButtonRepeatMap = make(map[eb.MouseButton]time.Duration)

func HandleMouseButtonRepeat(
	firstRate, repeatRate time.Duration,
	button eb.MouseButton,
) bool {
	if !IsMouseButtonPressed(button) {
		mouseButtonRepeatMap[button] = 0
		return false
	}

	if IsMouseButtonJustPressed(button) {
		mouseButtonRepeatMap[button] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := mouseButtonRepeatMap[button]

	if !ok {
		mouseButtonRepeatMap[button] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			mouseButtonRepeatMap[button] = now
			return true
		}
	}

	return false
}

func IsKeyPressed(key eb.Key) bool {
	return eb.IsKeyPressed(key)
}

func IsKeyJustPressed(key eb.Key) bool {
	return ebi.IsKeyJustPressed(key)
}
