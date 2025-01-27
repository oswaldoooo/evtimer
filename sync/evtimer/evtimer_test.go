package evtimer_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/oswaldoooo/evtimer/sync/evtimer"
)

var (
	count = 10
)

func greet(et *evtimer.SyncEvtimer, ev *evtimer.Event) {
	count--
	fmt.Println("count", count)
	if count < 1 {
		return
	}
	et.Add(*ev, time.Second)
}
func TestEvtimer(t *testing.T) {
	et := evtimer.NewSyncEvtimer()
	et.Add(evtimer.Event{
		Cb: greet,
	}, time.Second)
	evtimer.Run(et)
}
