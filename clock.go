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
