package cageapp

import "time"

type App struct {
	CI      bool
	NoColor bool
}

func (a *App) SpinInterval() time.Duration {
	if a.CI {
		return time.Second * 10
	}
	return time.Millisecond * 100
}
