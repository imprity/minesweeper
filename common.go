package main

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
