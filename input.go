package minesweeper

import (
	"time"

	eb "github.com/hajimehoshi/ebiten/v2"
	ebi "github.com/hajimehoshi/ebiten/v2/inpututil"
)

type TouchInfo struct {
	TouchID eb.TouchID

	StartedTime time.Duration
	StartedPos  FPoint

	EndedTime time.Duration
	EndedPos  FPoint
	DidEnd    bool

	Dragged bool

	// max number of simultaneous touches during
	// this was touching
	MaxTouchCount int
}

func (ti *TouchInfo) IsTouching() bool {
	return IsTouchIdTouching(ti.TouchID)
}

func (ti *TouchInfo) IsJustPressed() bool {
	return IsTouchIdJustPressed(ti.TouchID)
}

func (ti *TouchInfo) IsJustReleased() bool {
	return IsTouchIdJustReleased(ti.TouchID)
}

var TheInputManager struct {
	// below fields are updated by TheInputManager
	// only public for convinience
	// don't write in to it

	TouchInfos map[eb.TouchID]TouchInfo

	TouchingMap     map[eb.TouchID]bool
	JustTouchedMap  map[eb.TouchID]bool
	JustReleasedMap map[eb.TouchID]bool

	TouchingBuf     []eb.TouchID
	JustTouchedBuf  []eb.TouchID
	JustReleasedBuf []eb.TouchID
}

func InitInputManager() {
	im := &TheInputManager

	im.TouchInfos = make(map[eb.TouchID]TouchInfo)
}

func UpdateInput() {
	im := &TheInputManager

	// =============================
	// update touch buffers
	// =============================
	im.TouchingBuf = eb.AppendTouchIDs(im.TouchingBuf[:0])
	im.JustTouchedBuf = ebi.AppendJustPressedTouchIDs(im.JustTouchedBuf[:0])
	im.JustReleasedBuf = ebi.AppendJustReleasedTouchIDs(im.JustReleasedBuf[:0])

	// =============================
	// update touch maps
	// =============================
	im.TouchingMap = nil
	im.JustTouchedMap = nil
	im.JustReleasedMap = nil

	if len(im.TouchingBuf) > 0 {
		im.TouchingMap = make(map[eb.TouchID]bool)
		for _, id := range im.TouchingBuf {
			im.TouchingMap[id] = true
		}
	}
	if len(im.JustTouchedBuf) > 0 {
		im.JustTouchedMap = make(map[eb.TouchID]bool)
		for _, id := range im.JustTouchedBuf {
			im.JustTouchedMap[id] = true
		}
	}
	if len(im.JustReleasedBuf) > 0 {
		im.JustReleasedMap = make(map[eb.TouchID]bool)
		for _, id := range im.JustReleasedBuf {
			im.JustReleasedMap[id] = true
		}
	}

	// =============================
	// update touch infos
	// =============================
	for _, touchId := range im.JustTouchedBuf {
		im.TouchInfos[touchId] = TouchInfo{
			StartedTime: GlobalTimerNow(),
			StartedPos:  TouchFPt(touchId),
			TouchID:     touchId,
		}
	}

	const dragDistance = 15

	for _, touchId := range im.TouchingBuf {
		if info, ok := im.TouchInfos[touchId]; ok {
			curPos := TouchFPt(touchId)
			if info.StartedPos.Sub(curPos).LengthSquared() > dragDistance*dragDistance {
				info.Dragged = true
			}

			info.MaxTouchCount = max(info.MaxTouchCount, len(im.TouchingBuf))

			im.TouchInfos[touchId] = info
		}
	}

	for _, touchId := range im.JustReleasedBuf {
		if info, ok := im.TouchInfos[touchId]; ok {
			info.DidEnd = true
			info.EndedTime = GlobalTimerNow()
			touchX, touchY := ebi.TouchPositionInPreviousTick(touchId)
			info.EndedPos = FPt(f64(touchX), f64(touchY))
			im.TouchInfos[touchId] = info
		}
	}

	// for safety
	// remove TouchInfo that are unpressed and too old
	for touchId, info := range im.TouchInfos {
		if !IsTouchIdTouching(touchId) && TimeSinceNow(info.StartedTime) > time.Minute*30 {
			delete(im.TouchInfos, touchId)
		}
	}
}

func GetTouchInfo(touchId eb.TouchID) (TouchInfo, bool) {
	im := &TheInputManager
	info, ok := im.TouchInfos[touchId]
	return info, ok
}

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

func IsTouchFree() bool {
	im := &TheInputManager

	return len(im.TouchingBuf) <= 0
}

func IsTouching(rect FRectangle, touchIdIn *eb.TouchID) bool {
	im := &TheInputManager

	for _, touchId := range im.TouchingBuf {
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

func IsTouchJustPressed(rect FRectangle, touchIdIn *eb.TouchID) bool {
	im := &TheInputManager

	for _, touchId := range im.JustTouchedBuf {
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
	im := &TheInputManager

	for _, touchId := range im.JustReleasedBuf {
		pos := PrevTouchFPt(touchId)

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

func IsTouchIdTouching(touchId eb.TouchID) bool {
	im := &TheInputManager
	return im.TouchingMap[touchId]
}

func IsTouchIdJustPressed(touchId eb.TouchID) bool {
	im := &TheInputManager
	return im.JustTouchedMap[touchId]
}

func IsTouchIdJustReleased(touchId eb.TouchID) bool {
	im := &TheInputManager
	return im.JustReleasedMap[touchId]
}
