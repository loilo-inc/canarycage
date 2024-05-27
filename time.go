package cage

import "time"

type timeImpl struct{}

func (t *timeImpl) Now() time.Time {
	return time.Now()
}
func (t *timeImpl) NewTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}
