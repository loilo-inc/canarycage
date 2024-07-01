package timeout

import "time"

type Time struct{}

func (t *Time) Now() time.Time {
	return time.Now()
}
func (t *Time) NewTimer(d time.Duration) *time.Timer {
	return time.NewTimer(d)
}
