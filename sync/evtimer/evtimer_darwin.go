package evtimer

import (
	"container/list"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type SyncEvtimer struct {
	isClose   atomic.Bool
	count     atomic.Int32
	kq        int
	links     map[int32]Event
	toadd     *list.List
	toaddlock sync.Mutex
	currfd    int32
}
type element struct {
	ev  Event
	exp time.Duration
}

const _int32Edge = 1<<31 - 1

func NewSyncEvtimer() *SyncEvtimer {
	var se = SyncEvtimer{
		links: make(map[int32]Event),
		toadd: list.New(),
	}
	kq, err := syscall.Kqueue()
	if err != nil {
		panic("kqueue error " + err.Error())
	}
	se.kq = kq
	return &se
}
func (et *SyncEvtimer) getFd() int32 {
	et.currfd++
	_, ok := et.links[et.currfd]
	for ok {
		if et.currfd < _int32Edge {
			et.currfd++
		} else {
			et.currfd = 0
		}
		_, ok = et.links[et.currfd]
	}
	return et.currfd
}
func (et *SyncEvtimer) add(ev Event, exp time.Duration) {
	fd := et.getFd()
	var ke = [1]syscall.Kevent_t{
		{
			Ident: uint64(fd), Filter: syscall.EVFILT_TIMER, Flags: syscall.EV_ADD | syscall.EV_ENABLE,
			Fflags: syscall.NOTE_SECONDS, Data: exp.Nanoseconds() / 1000000000,
		},
	}
	syscall.Kevent(et.kq, ke[:], nil, nil)
}

func (et *SyncEvtimer) Add(ev Event, exp time.Duration) {
	et.toaddlock.Lock()
	et.toadd.PushBack(element{
		ev: ev, exp: exp,
	})
	et.toaddlock.Unlock()
}
func Run(et *SyncEvtimer) error {
	var (
		klist, kdlist [10]syscall.Kevent_t
		// eventList     [10]Event
		ts = syscall.Timespec{
			Sec: 1,
		}
		equeue chan []Event = make(chan []Event, 5)
	)

	go func() {
		for elist := range equeue {
			for i := range elist {
				elist[i].Cb(et, &elist[i])
			}
		}
	}()
	for {
		if et.isClose.Load() && et.count.Load() < 1 {
			return nil
		}
		n, err := syscall.Kevent(et.kq, nil, klist[:], &ts)
		var elelist []Event
		if err != nil {
			fmt.Fprintln(os.Stderr, "[kevent error]"+err.Error())
			return err
		} else if n < 1 {
			goto _end
		}
		elelist = make([]Event, n)
		for i := 0; i < n; i++ {
			ee := &klist[i]
			ke := &kdlist[i]
			ke.Ident = ee.Ident
			ke.Filter = ee.Filter
			ke.Flags = syscall.EV_DELETE
			elelist[i] = et.links[int32(ee.Ident)]
			delete(et.links, int32(ee.Ident))
		}
		_, err = syscall.Kevent(et.kq, kdlist[:n], nil, nil)
		if err != nil {
			panic("delete event error " + err.Error())
		}
	_end:
		// var elelist []element
		et.toaddlock.Lock()
		if et.toadd.Len() > 0 {
			var toaddlist []syscall.Kevent_t = make([]syscall.Kevent_t, et.toadd.Len())
			// elelist = make([]element, et.toadd.Len())
			for i := 0; i < et.toadd.Len(); i++ {
				e := &toaddlist[i]
				e.Ident = uint64(et.getFd())
				e.Filter = syscall.EVFILT_TIMER
				e.Flags = syscall.EV_ADD | syscall.EV_ENABLE
				e.Fflags = syscall.NOTE_SECONDS
				ele := et.toadd.Front()
				v := ele.Value.(element)
				e.Data = v.exp.Milliseconds() / 1000
				et.toadd.Remove(ele)
				et.links[int32(e.Ident)] = v.ev
			}
			_, err := syscall.Kevent(et.kq, toaddlist, nil, nil)
			if err != nil {
				panic("add event failed " + err.Error())
			}
		}
		et.toaddlock.Unlock()
		if len(elelist) > 0 {
			equeue <- elelist
		}
	}
}
func TryWait(et *SyncEvtimer) bool {
	return et.count.Load() < 1
}

func (et *SyncEvtimer) Stop() {
	if !et.isClose.CompareAndSwap(false, true) {
		return
	}
	for !TryWait(et) {
		time.Sleep(time.Millisecond * 100)
	}
}
