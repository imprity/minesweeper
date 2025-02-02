package minesweeper

import (
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
)

type InputGroupId int64

var inputGroupIdMax InputGroupId

func NewInputGroupId() InputGroupId {
	inputGroupIdMax++
	return inputGroupIdMax
}

func IsMouseButtonPressed(button eb.MouseButton) bool {
	return eb.IsMouseButtonPressed(button)
}

func IsMouseButtonJustPressed(button eb.MouseButton) bool {
	return ebi.IsMouseButtonJustPressed(button)
}

func IsMouseButtonJustReleased(button eb.MouseButton) bool {
	return ebi.IsMouseButtonJustReleased(button)
}

var mouseButtonRepeatMap = make(
	map[struct {
		Id     InputGroupId
		Button eb.MouseButton
	}]time.Duration,
)

func HandleMouseButtonRepeat(
	inputId InputGroupId,
	rect FRectangle,
	firstRate, repeatRate time.Duration,
	button eb.MouseButton,
) bool {
	idAndButton := struct {
		Id     InputGroupId
		Button eb.MouseButton
	}{
		Id:     inputId,
		Button: button,
	}

	cursor := CursorFPt()

	if !IsMouseButtonPressed(button) || !cursor.In(rect) {
		mouseButtonRepeatMap[idAndButton] = 0
		return false
	}

	if IsMouseButtonJustPressed(button) {
		mouseButtonRepeatMap[idAndButton] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := mouseButtonRepeatMap[idAndButton]

	if !ok {
		mouseButtonRepeatMap[idAndButton] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			mouseButtonRepeatMap[idAndButton] = now
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

var keyRepeatMap = make(map[eb.Key]time.Duration)

func HandleKeyRepeat(
	firstRate, repeatRate time.Duration,
	key eb.Key,
) bool {
	if !IsKeyPressed(key) {
		keyRepeatMap[key] = 0
		return false
	}

	if IsKeyJustPressed(key) {
		keyRepeatMap[key] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := keyRepeatMap[key]

	if !ok {
		keyRepeatMap[key] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			keyRepeatMap[key] = now
			return true
		}
	}

	return false
}

var touchIdBuffer []eb.TouchID

func IsTouchJustPressed(rect FRectangle, touchIdIn *eb.TouchID) bool {
	touchIdBuffer = ebi.AppendJustPressedTouchIDs(touchIdBuffer[:0])

	for _, touchId := range touchIdBuffer {
		posX, posY := eb.TouchPosition(touchId)

		pos := FPt(f64(posX), f64(posY))

		if pos.In(rect) {
			if touchIdIn != nil {
				*touchIdIn = touchId
			}
			return true
		}
	}

	return false
}

func IsTouchJustReleased(rect FRectangle, touchIdIn *eb.TouchID) bool {
	touchIdBuffer = ebi.AppendJustReleasedTouchIDs(touchIdBuffer[:0])

	for _, touchId := range touchIdBuffer {
		posX, posY := ebi.TouchPositionInPreviousTick(touchId)

		pos := FPt(f64(posX), f64(posY))

		if pos.In(rect) {
			if touchIdIn != nil {
				*touchIdIn = touchId
			}
			return true
		}
	}

	return false
}

func IsTouchFree() bool {
	touchIdBuffer = eb.AppendTouchIDs(touchIdBuffer[:0])

	return len(touchIdBuffer) <= 0
}

func IsTouching(rect FRectangle, touchIdIn *eb.TouchID) bool {
	touchIdBuffer = eb.AppendTouchIDs(touchIdBuffer[:0])

	for _, touchId := range touchIdBuffer {
		posX, posY := eb.TouchPosition(touchId)

		pos := FPt(f64(posX), f64(posY))

		if pos.In(rect) {
			if touchIdIn != nil {
				*touchIdIn = touchId
			}
			return true
		}
	}

	return false
}

var touchRepeatMap = make(map[InputGroupId]time.Duration)

func HandleTouchRepeat(
	inputId InputGroupId,
	rect FRectangle,
	firstRate, repeatRate time.Duration,
) bool {
	// nothing is touching inside rectangle, safe to free the map
	if !IsTouching(rect, nil) {
		touchRepeatMap[inputId] = 0
		return false
	}

	if IsTouchJustPressed(rect, nil) {
		touchRepeatMap[inputId] = GlobalTimerNow() + firstRate
		return true
	}

	time, ok := touchRepeatMap[inputId]

	if !ok {
		touchRepeatMap[inputId] = GlobalTimerNow() + firstRate
		return true
	} else {
		now := GlobalTimerNow()
		if now-time > repeatRate {
			touchRepeatMap[inputId] = now
			return true
		}
	}

	return false
}
