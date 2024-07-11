package cage

import (
	"github.com/loilo-inc/canarycage/types"
	"github.com/loilo-inc/logos/di"
)

type cage struct {
	di *di.D
}

func NewCage(di *di.D) types.Cage {
	return &cage{di}
}
