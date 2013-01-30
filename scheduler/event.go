package scheduler

import (
	"errors"
	//"fmt"
	"gpio"
	"time"
)

type Event struct {
	Pins        []int
	State       gpio.State
	NextTime    time.Time
	RepeatDays  []bool
	RepeatWeeks []bool
}

func (e *Event) UpdateNextTime() error {
	// e.nextTime is thisTime right now
	now := time.Now()
	today := e.NextTime.Weekday()
	_, thisWeek := e.NextTime.ISOWeek()

	firstTime := true
	for {
		day := 0
		if firstTime {
			day = int(today) + 1
			firstTime = false
		}
		for ; day < 7; day++ {
			// Add one more day to the wait time
			e.NextTime = e.NextTime.Add(24 * time.Hour)
			// If we're up to today and it is enabled then we're done
			if e.NextTime.After(now) && e.RepeatWeeks[(thisWeek-1)%52] && e.RepeatDays[day] {
				return nil
			}
		}
		thisWeek++
	}

	return errors.New("This point sould never be reached")
}

func (e *Event) Update(
	pins []int,
	state gpio.State,
	nextTime time.Time,
	repeatDays, repeatWeeks []bool) error {
	e.Pins = pins
	e.State = state
	e.NextTime = nextTime
	e.RepeatDays = repeatDays
	e.RepeatWeeks = repeatWeeks
	return nil
}
