package cage

import "time"

var newTimer = time.NewTimer
var now = time.Now

func fakeTimer(d time.Duration) *time.Timer {
	ch := make(chan time.Time)
	go func() {
		ch <- time.Now()
	}()
	return &time.Timer{
		C: ch,
	}
}

func recoverTimer() {
	newTimer = time.NewTimer
}