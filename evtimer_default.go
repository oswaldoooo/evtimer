//go:build !linux && !darwin

package evtimer

import (
	"reflect"
	"time"
)

type Evtimer struct {
	count int32
	links map[reflect.Value]Event
}

func NewEvtimer() *Evtimer {
	return &Evtimer{links: make(map[reflect.Value]Event)}
}
func (et *Evtimer) Add(ev Event, exp time.Duration) {
	t := time.NewTimer(exp)
	et.links[reflect.ValueOf(t.C)] = ev
	et.count++
}
func Run(et *Evtimer) {
	for {
		timerlist := make([]reflect.SelectCase, 0, len(et.links))
		for k := range et.links {
			timerlist = append(timerlist, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: k})
		}
		pos, _, ok := reflect.Select(timerlist)
		if !ok {
			delete(et.links, timerlist[pos].Chan)
			continue
		}
		index := timerlist[pos].Chan
		ev := et.links[index]
		delete(et.links, index)
		et.count--
		ev.Cb(et, &ev)
	}
}
func Wait(et *Evtimer) {
	for !TryWait(et) {
		time.Sleep(time.Millisecond * 100)
	}
}
func TryWait(et *Evtimer) bool {
	return et.count < 1
}
