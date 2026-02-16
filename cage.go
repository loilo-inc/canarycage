package cage

import (
	"github.com/loilo-inc/canarycage/v5/key"
	"github.com/loilo-inc/canarycage/v5/logger"
	"github.com/loilo-inc/canarycage/v5/types"
	"github.com/loilo-inc/logos/v2/di"
)

type cage struct {
	di *di.D
}

func NewCage(di *di.D) types.Cage {
	return &cage{di}
}

func (c *cage) logger() logger.Logger {
	return c.di.Get(key.Logger).(logger.Logger)
}
