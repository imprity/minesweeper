package main

import (
	"math/rand/v2"
)

//==============================================
// BOARD STUFFS
//==============================================

type Board struct {
	Width  int
	Height int

	Mines Array2D[bool]

	Revealed Array2D[bool]
	Flags    Array2D[bool]
}

func NewBoard(width int, height int) Board {
	var board Board

	board.Width = width
	board.Height = height

	board.Mines = New2DArray[bool](width, height)
	board.Revealed = New2DArray[bool](width, height)
	board.Flags = New2DArray[bool](width, height)

	return board
}

func (board *Board) PlaceMines(count, exceptX, exceptY int, seed [32]byte) {
	tilesTotal := board.Width * board.Height

	maxCount := tilesTotal - 1

	count = min(count, maxCount)

	minePlaces := make([][2]int, 0, count)

	rng := rand.New(rand.NewChaCha8(seed))

	for x := range board.Width {
		for y := range board.Height {
			if Abs(x-exceptX) <= 1 && Abs(y-exceptY) <= 1 {
				continue
			}

			minePlaces = append(minePlaces, [2]int{x, y})
		}
	}

	if len(minePlaces) < count {
		for x := exceptX - 1; x <= exceptX+1; x++ {
			for y := exceptY - 1; y <= exceptY+1; y++ {
				if !(x == exceptX && y == exceptY) && board.IsPosInBoard(x, y) {
					minePlaces = append(minePlaces, [2]int{x, y})
				}
			}
		}
	}

	for range 4 {
		rng.Shuffle(len(minePlaces), func(i, j int) {
			minePlaces[i], minePlaces[j] = minePlaces[j], minePlaces[i]
		})
	}

	for i := 0; i < count; i++ {
		//board.Mines[minePlaces[i][0]][minePlaces[i][1]] = true
		board.Mines.Set(minePlaces[i][0], minePlaces[i][1], true)
	}
}

func (board *Board) Copy() Board {
	copy := NewBoard(board.Width, board.Height)

	iterator := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iterator.HasNext() {
		x, y := iterator.GetNext()

		copy.Mines.Set(x, y, board.Mines.Get(x, y))
		copy.Revealed.Set(x, y, board.Revealed.Get(x, y))
		copy.Flags.Set(x, y, board.Flags.Get(x, y))
	}

	return copy
}

func (board *Board) SaveTo(targetBoard Board) {
	if !(board.Width == targetBoard.Width && board.Height == targetBoard.Height) {
		ErrLogger.Fatalf("targetBoard dimmensions is not equal to board")
	}

	iterator := NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iterator.HasNext() {
		x, y := iterator.GetNext()

		targetBoard.Mines.Set(x, y, board.Mines.Get(x, y))
		targetBoard.Revealed.Set(x, y, board.Revealed.Get(x, y))
		targetBoard.Flags.Set(x, y, board.Flags.Get(x, y))
	}
}

func (board *Board) IsPosInBoard(posX int, posY int) bool {
	return posX >= 0 && posX < board.Width && posY >= 0 && posY < board.Height
}

func (board *Board) SpreadSafeArea(posX int, posY int) {
	if !board.IsPosInBoard(posX, posY) {
		return
	}

	if board.Revealed.Get(posX, posY) {
		return
	}

	if board.Mines.Get(posX, posY) {
		return
	}

	board.Revealed.Set(posX, posY, true)

	if board.GetNeighborMineCount(posX, posY) > 0 {
		return
	}

	board.Flags.Set(posX, posY, false)

	iterator := NewBoardIterator(posX-1, posY-1, posX+1, posY+1)
	for iterator.HasNext() {
		x, y := iterator.GetNext()
		if board.IsPosInBoard(x, y) {
			board.SpreadSafeArea(x, y)
		}
	}
}

func (board *Board) GetNeighborMineCount(posX int, posY int) int {
	var mineCount int = 0
	for x := posX - 1; x <= posX+1; x++ {
		for y := posY - 1; y <= posY+1; y++ {
			if board.IsPosInBoard(x, y) && board.Mines.Get(x, y) {
				mineCount += 1
			}
		}
	}

	return mineCount
}

func (board *Board) GetNeighborFlagCount(posX int, posY int) int {
	var flagCount int = 0
	for x := max(posX-1, 0); x < min(posX+2, board.Width); x++ {
		for y := max(posY-1, 0); y < min(posY+2, board.Height); y++ {
			if board.Flags.Get(x, y) {
				flagCount += 1
			}
		}
	}

	return flagCount
}

func (board *Board) HasNoMines() bool {
	for x := range board.Width {
		for y := range board.Height {
			if board.Mines.Get(x, y) {
				return false
			}
		}
	}

	return true
}

// win condition
func (board *Board) IsAllSafeTileRevealed() bool {
	var iter BoardIterator = NewBoardIterator(0, 0, board.Width-1, board.Height-1)

	for iter.HasNext() {
		x, y := iter.GetNext()
		if !board.Revealed.Get(x, y) && !board.Mines.Get(x, y) {
			return false
		}
	}

	return true
}

//==============================================
// board iterator
//==============================================

type BoardIterator struct {
	MinX int
	MinY int
	MaxX int
	MaxY int

	CurrentX int
	CurrentY int
}

// inclusive
func NewBoardIterator(x1 int, y1 int, x2 int, y2 int) BoardIterator {
	iterator := BoardIterator{
		MinX: min(x1, x2),
		MinY: min(y1, y2),

		MaxX: max(x1, x2),
		MaxY: max(y1, y2),
	}

	iterator.CurrentX = iterator.MinX
	iterator.CurrentY = iterator.MinY

	return iterator
}

func (bi *BoardIterator) HasNext() bool {
	return bi.CurrentY <= bi.MaxY
}

func (bi *BoardIterator) GetNext() (int, int) {
	x := bi.CurrentX
	y := bi.CurrentY

	bi.CurrentX++
	if bi.CurrentX > bi.MaxX {
		bi.CurrentX = bi.MinX
		bi.CurrentY++
	}

	return x, y
}

func (bi *BoardIterator) Reset() {
	bi.CurrentX = bi.MinX
	bi.CurrentY = bi.MinY
}

// ==============================================
// board iteraction
// ==============================================

type BoardInteractionType int

const (
	InteractionTypeNone BoardInteractionType = iota
	InteractionTypeStep
	InteractionTypeFlag
	InteractionTypeCheck
)

type GameState int

const (
	GameStatePlaying GameState = iota
	GameStateWon
	GameStateLost
)

func (board *Board) InteractAt(
	posX int, posY int,
	interaction BoardInteractionType,
	gameState GameState,

	// information needed to spawn mines
	minesToSpawn int, seed [32]byte,
) GameState {
	if gameState != GameStatePlaying {
		return gameState
	}

	if interaction == InteractionTypeNone {
		return GameStatePlaying
	}
	if !board.IsPosInBoard(posX, posY) {
		return GameStatePlaying
	}

	defer func() {
		// remove flags where it's revealed
		for x := 0; x < board.Width; x++ {
			for y := 0; y < board.Height; y++ {
				if board.Revealed.Get(x, y) {
					board.Flags.Set(x, y, false)
				}
			}
		}
	}()

	switch interaction {
	case InteractionTypeStep:
		{
			if board.HasNoMines() {
				board.PlaceMines(minesToSpawn, posX, posY, seed)
			}
			if !board.Revealed.Get(posX, posY) {
				if board.Flags.Get(posX, posY) { // if flag is up, ignore step
					return GameStatePlaying
				}
				if board.Mines.Get(posX, posY) {
					return GameStateLost // user stepped on a mine
				}
				//we have to spread out
				board.SpreadSafeArea(posX, posY)
			}
			if board.IsAllSafeTileRevealed() {
				return GameStateWon
			} else {
				return GameStatePlaying
			}
		}
	case InteractionTypeFlag:
		{
			if !board.Revealed.Get(posX, posY) {
				//board.Flags[posX][posY] = !board.Flags[posX][posY]
				board.Flags.Set(posX, posY, !board.Flags.Get(posX, posY))
			}
			return GameStatePlaying
		}
	case InteractionTypeCheck:
		{
			if board.Revealed.Get(posX, posY) && board.GetNeighborMineCount(posX, posY) > 0 {
				var flagCount int = board.GetNeighborFlagCount(posX, posY)
				if board.GetNeighborMineCount(posX, posY) == flagCount {
					//check if user flagged it correctly
					iterator := NewBoardIterator(posX-1, posY-1, posX+1, posY+1)

					for iterator.HasNext() {
						x, y := iterator.GetNext()
						if board.IsPosInBoard(x, y) {
							if board.Flags.Get(x, y) && !board.Mines.Get(x, y) {
								return GameStateLost
							}
						}
					}

					//reset iterator
					iterator = NewBoardIterator(posX-1, posY-1, posX+1, posY+1)

					for iterator.HasNext() {
						x, y := iterator.GetNext()
						if board.IsPosInBoard(x, y) {
							board.SpreadSafeArea(x, y)
						}
					}

				}
			}

			if board.IsAllSafeTileRevealed() {
				return GameStateWon
			} else {
				return GameStatePlaying
			}
		}

	}
	panic("UNREACHABLE")
}
