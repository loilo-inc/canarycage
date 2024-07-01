package cage

import "github.com/loilo-inc/logos/di"

type CageExport = cage

func NewCageExport(di *di.D) *cage {
	return &cage{di}
}
