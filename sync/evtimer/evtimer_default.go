//go:build !darwin

package evtimer

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type SyncEvtimer struct {
	flushchan chan struct{}
	linkslock sync.Mutex
	links     map[reflect.Value]Event
	isClose   atomic.Bool
	count     atomic.Int32
}

func NewSyncEvtimer() *SyncEvtimer {
	var se = SyncEvtimer{
		flushchan: make(chan struct{}, 5),
		links:     make(map[reflect.Value]Event),
	}
	return &se
}
func (et *SyncEvtimer) Add(ev Event, exp time.Duration) {
	if et.isClose.Load() == true {
		panic("evtimer is close!")
	}
	t := time.NewTimer(exp)
	et.linkslock.Lock()
	et.links[reflect.ValueOf(t)] = ev
	et.linkslock.Unlock()
	et.count.Add(1)
}
func (et *SyncEvtimer) Stop() {
	if !et.isClose.CompareAndSwap(false, true) {
		return
	}
	for et.count.Load() > 0 {
		time.Sleep(time.Millisecond * 100)
	}
}
func Run(et *SyncEvtimer) error {
	var (
		flushchan = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(et.flushchan)}
		vqueue    = make(chan Event, 5)
	)
	go func() {
		for task := range vqueue {
			task.Cb(et, &task)
		}
	}()
	for !et.isClose.Load() {
		chanlist := make([]reflect.SelectCase, 1, 1+len(et.links))
		chanlist[0] = flushchan
		for k := range et.links {
			chanlist = append(chanlist, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: k})
		}
		pos, _, ok := reflect.Select(chanlist)
		if !ok {
			continue
		} else if pos == 0 {
			continue
		}
		et.linkslock.Lock()
		e := et.links[chanlist[pos].Chan]
		delete(et.links, chanlist[pos].Chan)
		et.linkslock.Unlock()
		et.count.Add(-1)
		vqueue <- e
	}
	close(vqueue)
	return nil
}
