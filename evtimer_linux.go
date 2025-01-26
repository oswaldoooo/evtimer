package evtimer

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

type _EventArgs struct {
	ev *Event
}
type Evtimer struct {
	count int32
	eid   int
	args  map[int32]_EventArgs
}
type Itimerspec struct {
	Interval syscall.Timespec
	Value    syscall.Timespec
}

const (
	_dis = 1000000000
)

func NewEvtimer() *Evtimer {
	var et Evtimer = Evtimer{
		args: make(map[int32]_EventArgs),
	}
	eid, err := syscall.EpollCreate1(0)
	if err != nil {
		panic("[epoll create error]" + err.Error())
	}
	et.eid = eid
	return &et
}
func (et *Evtimer) Add(ev Event, exp time.Duration) {
	_td, _, err := syscall.Syscall(syscall.SYS_TIMERFD_CREATE, _clock_real_time, 0, 0)
	td := int32(_td)
	if td < 0 {
		fmt.Fprintln(os.Stderr, "[add event error]"+err.Error())
		return
	}
	ns := exp.Nanoseconds()
	var its Itimerspec
	its.Value.Sec = ns / _dis
	its.Value.Nsec = ns % _dis
	ok, _, err := syscall.Syscall(syscall.SYS_TIMERFD_SETTIME, _td, 0, uintptr(unsafe.Pointer(&its)))
	if int(ok) < 0 {
		fmt.Println("[debug] timerfd settime error " + err.Error())
	}
	var e syscall.EpollEvent
	e.Events = syscall.EPOLLIN
	e.Fd = td
	syscall.EpollCtl(et.eid, syscall.EPOLL_CTL_ADD, int(td), &e)
	et.args[td] = _EventArgs{
		ev: &ev,
	}
	et.count++
}

const (
	_clock_real_time = 0
)

func (et *Evtimer) handle(td int32) {
	var exp [8]byte
	syscall.Read(int(td), exp[:])
	syscall.Close(int(td))
	ea := et.args[td]
	delete(et.args, td)
	ea.ev.Cb(et, ea.ev)
}
func Run(et *Evtimer) {
	var elist [10]syscall.EpollEvent
	for {
		n, err := syscall.EpollWait(et.eid, elist[:], 1000)
		fmt.Println("[debug] elist count", n)
		if err != nil {
			fmt.Fprintln(os.Stderr, "[epoll wait error]"+err.Error())
			continue
		} else if n < 1 {
			continue
		}
		for i := 0; i < n; i++ {
			syscall.EpollCtl(syscall.EPOLL_CTL_DEL, syscall.EPOLLIN, int(elist[i].Fd), &elist[i])
			et.count--
			et.handle(elist[i].Fd)
		}
	}

}
func Wait(et *Evtimer) {
	for et.count > 0 {
		time.Sleep(time.Millisecond * 100)
	}
}
func TryWait(et *Evtimer) bool {
	return et.count == 0
}
