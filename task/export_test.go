package task

import (
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/loilo-inc/logos/di"
)

type CommonExport = common
type AlbTaskExport = albTask
type SimpleTaskExport = simpleTask

func NewCommonExport(di *di.D, input *Input) *common {
	return &common{di: di, Input: input}
}

func NewAlbTaskExport(di *di.D, input *Input, lb *ecstypes.LoadBalancer) *albTask {
	return &albTask{common: &common{di: di, Input: input}, Lb: lb}
}

func NewSimpleTaskExport(di *di.D, input *Input) *simpleTask {
	return &simpleTask{common: &common{di: di, Input: input}}
}
