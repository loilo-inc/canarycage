package test

import (
	"time"

	"github.com/loilo-inc/canarycage/types"
)

func newTimer(_ time.Duration) *time.Timer {
	ch := make(chan time.Time)
	go func() {
		ch <- time.Now()
	}()
	return &time.Timer{
		C: ch,
	}
}

type timeImpl struct{}

func (t *timeImpl) Now() time.Time {
	return time.Now()
}
func (t *timeImpl) NewTimer(d time.Duration) *time.Timer {
	return newTimer(d)
}
func NewFakeTime() types.Time {
	return &timeImpl{}
}
