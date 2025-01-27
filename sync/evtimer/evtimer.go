package evtimer

type Callback func(*SyncEvtimer, *Event)
type Event struct {
	Cb Callback
}
