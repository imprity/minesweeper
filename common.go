package minesweeper

import (
	"fmt"
)

type CircularQueue[T any] struct {
	End    int
	Start  int
	Length int
	Data   []T
}

func NewCircularQueue[T any](size int) CircularQueue[T] {
	return CircularQueue[T]{
		Data: make([]T, size),
	}
}

func (q *CircularQueue[T]) IsFull() bool {
	return q.Length >= len(q.Data)
}

func (q *CircularQueue[T]) IsEmpty() bool {
	return q.Length <= 0
}

func (q *CircularQueue[T]) Enqueue(item T) {
	index := q.End

	isFull := q.IsFull()

	if isFull {
		q.Start += 1
		q.Start = q.Start % len(q.Data)
		q.End += 1
		q.End = q.End % len(q.Data)
	} else {
		q.End += 1
		q.End = q.End % len(q.Data)
		q.Length += 1
	}

	q.Data[index] = item
}

func (q *CircularQueue[T]) Dequeue() T {
	if q.Length <= 0 {
		panic("CircularQueue:Dequeue: Dequeue on empty queue")
	}

	q.Length -= 1

	q.Start %= len(q.Data)
	returnIndex := q.Start
	q.Start += 1

	return q.Data[returnIndex]
}

func (q *CircularQueue[T]) At(index int) T {
	return q.Data[(q.Start+index)%len(q.Data)]
}

func (q *CircularQueue[T]) PeekFirst() T {
	return q.Data[q.Start%len(q.Data)]
}

func (q *CircularQueue[T]) PeekLast() T {
	return q.Data[(q.End-1)%len(q.Data)]
}

func (q *CircularQueue[T]) Clear() {
	q.Length = 0
	q.Start = 0
	q.End = 0
}

type Queue[T any] struct {
	Data []T
}

func (q *Queue[T]) Length() int {
	return len(q.Data)
}

func (q *Queue[T]) IsEmpty() bool {
	return len(q.Data) <= 0
}

func (q *Queue[T]) Enqueue(item T) {
	q.Data = append(q.Data, item)
}

func (q *Queue[T]) Dequeue() T {
	toReturn := q.Data[0]

	for i := 0; i+1 < len(q.Data); i++ {
		q.Data[i] = q.Data[i+1]
	}

	q.Data = q.Data[:len(q.Data)-1]

	return toReturn
}

func (q *Queue[T]) At(index int) T {
	return q.Data[index]
}

func (q *Queue[T]) Set(index int, item T) {
	q.Data[index] = item
}

func (q *Queue[T]) PeekFirst() T {
	return q.Data[0]
}

func (q *Queue[T]) PeekLast() T {
	return q.Data[len(q.Data)-1]
}

func (q *Queue[T]) Clear() {
	q.Data = q.Data[:0]
}

type Array2D[T any] struct {
	Width  int
	Height int

	Data []T
}

func New2DArray[T any](width, height int) Array2D[T] {
	arr := Array2D[T]{
		Width:  width,
		Height: height,
		Data:   make([]T, width*height),
	}

	return arr
}

func (a *Array2D[T]) Get(x, y int) T {
	if x < 0 || y < 0 || x >= a.Width || y >= a.Height {
		msg := fmt.Sprintf(
			"%d, %d is out side of %d, %d",
			x, y, a.Width, a.Height,
		)
		panic(msg)
	}
	return a.Data[x+y*a.Width]
}

func (a *Array2D[T]) Set(x, y int, t T) {
	if x < 0 || y < 0 || x >= a.Width || y >= a.Height {
		msg := fmt.Sprintf(
			"%d, %d is out side of %d, %d",
			x, y, a.Width, a.Height,
		)
		panic(msg)
	}
	a.Data[x+y*a.Width] = t
}

func (a *Array2D[T]) Resize(newWidth, newHeight int) {
	dataCap := cap(a.Data)
	requiredDataLen := newWidth * newHeight

	if requiredDataLen > dataCap {
		a.Data = make([]T, requiredDataLen)
	} else {
		a.Data = a.Data[:requiredDataLen]
	}

	a.Width = newWidth
	a.Height = newHeight
}
