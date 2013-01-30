package scheduler

import (
	"fmt"
	//"html/template"
	//"io/ioutil"
	"net/http"
)

func (s *Scheduler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the home page")
}

func (s *Scheduler) EventsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the event[s] page")
}

func (s *Scheduler) EventHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the event page")
}

func (s *Scheduler) PlanHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the floor plan page")
}

func (s *Scheduler) EditHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the edit page")
}
