package evtimer_test

import (
	"fmt"

	"testing"
	"time"

	"github.com/oswaldoooo/evtimer"
)

var count = 10

func greet(et *evtimer.Evtimer, ev *evtimer.Event) {
	fmt.Println("hello ,greet active")
	count--
	if count < 1 {
		return
	}
	et.Add(evtimer.Event{Cb: greet}, time.Second)
}
func greetOnce(et *evtimer.Evtimer, ev *evtimer.Event) {
	fmt.Println("hello ,greet active 2")
}
func TestTimer(t *testing.T) {
	et := evtimer.NewEvtimer()
	et.Add(evtimer.Event{
		Cb: greet,
	}, time.Second)
	et.Add(evtimer.Event{Cb: greetOnce}, time.Second*3)
	go evtimer.Run(et)
	evtimer.Wait(et)
}
