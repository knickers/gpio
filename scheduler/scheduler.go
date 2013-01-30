package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gpio"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

type Scheduler struct {
	pins   []gpio.Pin
	events []chan Event // sorted by next closest event time
}

func New() *Scheduler {
	return new(Scheduler)
}

func (s *Scheduler) exists(pin int) int {
	for i, p := range s.pins {
		if p.GetNumber() == uint(pin) {
			return i
		}
	}
	return -1
}

func (s *Scheduler) SetPinState(pin int, state gpio.State) {
	i := s.exists(pin)

	// This pin doesn't exist yet, create a new one
	if i == -1 {
		p, err := gpio.NewPin(uint(pin), gpio.OUTPUT)
		if err != nil {
			fmt.Println(err)
			return
		}
		s.pins = append(s.pins, *p)
		i = len(s.pins) - 1
	}

	s.pins[i].SetState(state)
}

func (s *Scheduler) CloseGPIOPins() {
	for _, p := range s.pins {
		p.Close()
	}
}

func (s *Scheduler) Pop() (Event, error) {
	if len(s.events) < 1 {
		return Event{}, errors.New("The events list is empty")
	}
	e := <-s.events[0]
	if len(s.events) > 1 {
		s.events = s.events[1:]
	} else {
		s.events = []chan Event{}
	}
	return e, nil
}

func (s *Scheduler) Push(e Event) error {
	evnt := make(chan Event, 1)
	evnt <- e
	s.events = append(s.events, evnt)
	return nil
}

func (s *Scheduler) InsertInOrder(e Event) error {
	ch := make(chan Event, 1)
	ch <- e
	s.events = append(s.events, ch)
	for i := len(s.events) - 2; i >= 0; i-- {
		evnt := <-s.events[i]
		if e.NextTime.After(evnt.NextTime) {
			s.events[i] <- evnt
			break
		}
		s.events[i] <- evnt
		tmp := s.events[i]
		s.events[i] = s.events[i+1]
		s.events[i+1] = tmp
	}
	return nil
}

func (s *Scheduler) GetNextTime() (time.Time, error) {
	if len(s.events) == 0 {
		return time.Now(), errors.New("No events in the queue")
	}

	e := <-s.events[0]
	nextTime := e.NextTime
	s.events[0] <- e

	return nextTime, nil
}

func (s *Scheduler) ManageEventQueue() {
	//log("Entering main loop\n")
	for {
		//log("Popping off the next event\n")
		event, err := s.Pop()
		if err == nil {
			now := time.Now()

			fmt.Printf("Sleeping %v till next event at %v...", event.NextTime.Sub(now), event.NextTime)
			time.Sleep(event.NextTime.Sub(now))
			fmt.Println("done")

			//log("Setting the gpio pin states to %s.", event.State.String())
				/*
			for _, pin := range event.Pins {
				//log("%d.", pin)
				err = C.SetPinState(pin, event.state)
				if err != nil {
					break
				}
			}
				*/
			//log("done\n")

			//log("Updating the next time for this event...")
			err = event.UpdateNextTime()
			if err != nil {
				fmt.Println(err)
				break
			}
			//log("done\n")

			//log("Putting this event back on the queue...")
			err = s.InsertInOrder(event)
			if err != nil {
				fmt.Println(err)
				break
			}
			//log("done\n")
		} else {
			//log(err.Error() + "\n")
			time.Sleep(time.Second)
		}
	}
}

func (s *Scheduler) GenerateRandomEvents(num int) {
	for i := 0; i < num; i++ {
		// up to five pins per event
		n := rand.Int()%5 + 1
		var pins []int
		for j := 0; j < n; j++ {
			pins = append(pins, rand.Int()%8)
		}
		// on or off
		state := gpio.State(rand.Int() % 2)
		// up to twenty seconds in the future
		dur, err := time.ParseDuration(fmt.Sprintf("%ds", rand.Int()%20+1))
		if err != nil {
			fmt.Println(err)
			break
		}
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
		s.InsertInOrder(Event{pins, state, nextT, days, weeks})
	}
}

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
	//log("%v\n", events)
	for _, e := range events {
		err = s.InsertInOrder(e)
		if err != nil {
			fmt.Println("Insert:", err)
			return err
		}
	}
	return nil
}
