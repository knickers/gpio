package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gpio"
	"io/ioutil"
	"math/rand"
	"msg"
	"os"
	"time"
)

type Scheduler struct {
	pins        []gpio.Pin
	Stamp       string // string formatting for printing time
	events      []chan Event
	queueLock   chan []int // index into events list, sorted by next event time
	nextEvent   chan Event
	stopWaiting chan bool
}

func New() *Scheduler {
	s := new(Scheduler)
	s.Stamp = "Jan 2 2006 at 3:04:05PM"
	s.nextEvent = make(chan Event)
	s.queueLock = make(chan []int, 1)
	s.queueLock <- []int{}
	s.stopWaiting = make(chan bool)
	return s
}

func (s *Scheduler) exists(pin int) int {
	for i, p := range s.pins {
		if p.GetNumber() == uint(pin) {
			return i
		}
	}
	return -1
}

func (s *Scheduler) SetPinState(pin int, state gpio.State) error {
	i := s.exists(pin)

	// This pin doesn't exist yet, create a new one
	if i == -1 {
		p, err := gpio.NewPin(uint(pin), gpio.OUTPUT)
		if err != nil {
			fmt.Println(err)
			return err
		}
		s.pins = append(s.pins, *p)
		i = len(s.pins) - 1
	}

	s.pins[i].SetState(state)
	return nil
}

func (s *Scheduler) CloseGPIOPins() {
	for _, p := range s.pins {
		p.Close()
	}
}

func (s *Scheduler) Pop() (Event, error) {
	queue := <-s.queueLock
	if len(queue) == 0 {
		s.queueLock <- queue
		return Event{}, errors.New("The events queue is empty")
	}
	e := <-s.events[queue[0]]
	if len(queue) > 1 {
		queue = queue[1:]
	} else {
		queue = []int{}
	}
	s.queueLock <- queue
	return e, nil
}

func (s *Scheduler) Push(e Event) error {
	if e.index > 0 && e.index < len(s.events) {
		s.events[e.index] <- e
	} else {
		e.index = len(s.events)
		evnt := make(chan Event, 1)
		evnt <- e
		s.events = append(s.events, evnt)
		queue := <-s.queueLock
		queue = append(queue, e.index)
		s.queueLock <- queue
	}
	return nil
}

func (s *Scheduler) InsertInOrder(e Event) error {
	s.Push(e)
	queue := <-s.queueLock
	for i := len(queue) - 2; i >= 0; i-- {
		evnt := <-s.events[queue[i]]
		nextTime := evnt.NextTime
		s.events[queue[i]] <- evnt
		if e.NextTime.After(nextTime) {
			s.queueLock <- queue
			return nil
		}
		tmp := queue[i]
		queue[i] = queue[i+1]
		queue[i+1] = tmp
	}
	s.queueLock <- queue
	return nil
}

func (s *Scheduler) GetNextTime() (time.Time, error) {
	if len(s.events) == 0 {
		return time.Now(), errors.New("No events in the queue")
	}

	q := <-s.queueLock
	e := <-s.events[q[0]]
	nextTime := e.NextTime
	s.events[q[0]] <- e
	s.queueLock <- q

	return nextTime, nil
}

func (s *Scheduler) feedNextEvent() {
	for {
		nextTime, err := s.GetNextTime()
		if err != nil {
			msg.Log(" feeder) Didn't find an event. Sleeping.\n")
			time.Sleep(time.Second)
			continue
		}

		msg.Log(" feeder) Next event on %v\n", nextTime.Format(s.Stamp))
		timer := time.AfterFunc(nextTime.Sub(time.Now()), func() {
			e, err := s.Pop()
			if err == nil {
				s.nextEvent <- e
				s.stopWaiting <- true
			}
		})

		<-s.stopWaiting
		if timer.Stop() {
			msg.Log(" feeder) Interrupted while waiting for current event.\n")
		} else {
			msg.Log(" feeder) Current event completed.\n")
		}
	}
}

func (s *Scheduler) ManageEventQueue() {
	go s.feedNextEvent()

	// main loop
	for {
		// wait for the next event
		msg.Log("\nmanager) Waiting for an event...\n")
		event := <-s.nextEvent
		msg.Log("manager) Got one an event.\n")

		// Set the gpio pin states
			/*
		for _, pin := range event.Pins {
			err = C.SetPinState(pin, event.state)
			if err != nil {
				break
			}
		}
			*/

		// Update the next time for this event
		err := event.UpdateNextTime()
		if err != nil {
			fmt.Println(err)
			break
		}

		// Put this event back on the queue
		err = s.InsertInOrder(event)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
}

// Generate num random events. All events will fire at least once in num * 2
// seconds after starting the program, then they will have random days and weeks
// in the future where they will fire again. These can be used for anything from
// filling in a schedule as a template to testing the output system.
func (s *Scheduler) GenerateRandomEvents(num int) {
	for i := 0; i < num; i++ {
		// up to eight pins per event
		n := rand.Int()%8 + 1
		var pins []int
		for j := 0; j < n; j++ {
			pins = append(pins, rand.Int()%8)
		}
		// on or off
		state := gpio.State(rand.Int() % 2)
		// up to num*2 seconds in the future
		dur := time.Second * time.Duration(rand.Int()%(num*2)+1)
		nextT := time.Now().Add(dur)
		// choose the days of the week to be applied
		var days []bool
		for j := 0; j < 7; j++ {
			r := false
			if rand.Int()%2 == 0 {
				r = true
			}
			days = append(days, r)
		}
		// choose the weeks of the year to be applied
		var weeks []bool
		for j := 0; j < 52; j++ {
			r := false
			if rand.Int()%2 == 0 {
				r = true
			}
			weeks = append(weeks, r)
		}
		s.InsertInOrder(Event{pins, -1, state, nextT, days, weeks})
	}
}

// Save the current in memory schedule out to file as a json encoded object
// according to schema.json
func (s *Scheduler) SaveSchedule(file string) error {
	var events []Event
	for _, e := range s.events {
		event := <-e
		events = append(events, event)
		e <- event
	}
	//bytes, err := json.Marshal(events)
	bytes, err := json.MarshalIndent(events, "", "\t")
	if err != nil {
		fmt.Println("Marshal:", err)
		return err
	}
	err = ioutil.WriteFile(file, bytes, os.FileMode(0664))
	if err != nil {
		fmt.Println("WriteOut:", err)
		return err
	}
	return nil
}

// Load a schedule file into memory. file should be a json encoded file
// according to schema.json
func (s *Scheduler) LoadSchedule(file string) error {
	fp, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println("ReadFile:", err)
		return err
	}
	//log("%s\n", string(fp))
	var events []Event
	err = json.Unmarshal(fp, &events)
	if err != nil {
		fmt.Println("Unmarshal:", err)
		return err
	}
	// Put all the events in
	for _, e := range events {
		err = s.InsertInOrder(e)
		if err != nil {
			fmt.Println("Insert:", err)
			return err
		}
	}
	return nil
}
