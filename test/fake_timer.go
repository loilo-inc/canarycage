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
	return &time.Timer{C: ch}
}

type ImmediateTime struct {
	never bool
}

var _ types.Time = (*ImmediateTime)(nil)

func (t *ImmediateTime) Now() time.Time {
	return time.Now()
}
func (t *ImmediateTime) NewTimer(d time.Duration) *time.Timer {
	return newTimer(d)
}

func NewFakeTime() types.Time {
	return &ImmediateTime{}
}

type NeverTime struct{}

var _ types.Time = (*NeverTime)(nil)

func (t *NeverTime) NewTimer(d time.Duration) *time.Timer {
	ch := make(chan time.Time)
	return &time.Timer{C: ch}
}

func (t *NeverTime) Now() time.Time {
	return time.Time{}
}

func NewNeverTimer() types.Time {
	return &NeverTime{}
}
