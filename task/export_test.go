package task

import "github.com/loilo-inc/logos/di"

type CommonExport = common

func NewCommonExport(di *di.D, input *Input) *common {
	return &common{di: di, Input: input}
}
