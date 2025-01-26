package evtimer

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

type Evtimer struct {
	kd      int
	args    map[int32]*Event
	count   int32
	idcount uint32
}

func NewEvtimer() *Evtimer {
	var et = Evtimer{args: make(map[int32]*Event)}
	kd, err := syscall.Kqueue()
	if err != nil {
		panic("[kqueue failed]" + err.Error())
	}
	et.kd = kd
	return &et
}

const _dis = 1000000000

func (et *Evtimer) Add(ev Event, exp time.Duration) {
	var ke [1]syscall.Kevent_t
	fd := et.getFd()
	ke[0].Ident = uint64(fd)
	ke[0].Filter = syscall.EVFILT_TIMER
	ke[0].Flags = syscall.EV_ADD | syscall.EV_ENABLE
	ke[0].Fflags = syscall.NOTE_SECONDS
	ke[0].Data = exp.Nanoseconds() / _dis
	_, err := syscall.Kevent(et.kd, ke[:], nil, nil)
	et.count++
	if err != nil {
		panic("[kevent add timer event error]" + err.Error())
	}
	et.args[int32(fd)] = &ev
}

const int32edge = (1 << 31) - 1

func (et *Evtimer) getFd() uint32 {
	if et.idcount+1 >= int32edge {
		et.idcount = 0
	} else {
		et.idcount++
	}
	return et.idcount
}
func Run(et *Evtimer) {
	var event_list, del_event_list [10]syscall.Kevent_t
	var ts syscall.Timespec = syscall.Timespec{
		Sec: 1,
	}
	for et.count > 0 {
		n, err := syscall.Kevent(et.kd, nil, event_list[:], &ts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[kevent error]"+err.Error())
			return
		} else if n < 1 {
			continue
		}
		for i := 0; i < n; i++ {
			e := &event_list[i]
			de := &del_event_list[i]
			de.Ident = e.Ident
			de.Filter = e.Filter
			de.Flags = syscall.EV_DELETE
			ea := et.args[int32(e.Ident)]
			delete(et.args, int32(e.Ident))
			ea.Cb(et, ea)
			et.count--
		}
		_, err = syscall.Kevent(et.kd, del_event_list[:n], nil, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[kevent del error]"+err.Error())
		}
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
