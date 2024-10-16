package main

import (
	"time"
)

var globalTimer time.Duration

const Years150 = time.Hour * 24 * 365 * 150

func UpdateGlobalTimer() {
	globalTimer += UpdateDelta()
}

func GlobalTimerNow() time.Duration {
	return globalTimer
}

func TimeSinceNow(t time.Duration) time.Duration {
	return GlobalTimerNow() - t
}

type Timer struct {
	Duration time.Duration
	Current  time.Duration
}

func (t *Timer) TickUp() {
	t.Current += UpdateDelta()
}

func (t *Timer) TickDown() {
	t.Current -= UpdateDelta()
}

func (t *Timer) ClampCurrent() {
	t.Current = Clamp(t.Current, 0, t.Duration)
}

func (t *Timer) Normalize() float64 {
	return Clamp(f64(t.Current)/f64(t.Duration), 0, 1)
}

func (t *Timer) NormalizeUnclamped() float64 {
	return f64(t.Current) / f64(t.Duration)
}

// Timer for profiling.
// Usage :
//
//	{
//		timer := NewProfTimer("some function")
//		defer timer.Report()
//		// reports some function took 10ms
//	}
type ProfTimer struct {
	Start time.Time
	Name  string
}

func NewProfTimer(name string) ProfTimer {
	return ProfTimer{
		Start: time.Now(),
		Name:  name,
	}
}

func (p ProfTimer) Report() {
	now := time.Now()
	InfoLogger.Printf("\"%v\" took %v\n", p.Name, now.Sub(p.Start))
}
