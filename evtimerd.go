package evtimer

type Callback func(*Evtimer, *Event)
type Event struct {
	Cb Callback
}
