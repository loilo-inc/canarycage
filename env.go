package main

import (
	"github.com/pkg/errors"
	"fmt"
	"math"
	"github.com/aws/aws-sdk-go/aws"
)

type Envars struct {
	_                        struct{} `type:"struct"`
	Region                   *string  `locationName:"region" type:"string"`
	Cluster                  *string  `locationName:"cluster" type:"string" required:"true"`
	LoadBalancerArn          *string  `locationName:"loadBalancerArn" type:"string" required:"true"`
	ServiceName              *string  `locationName:"serviceName" type:"string" required:"true"`
	CurrentTaskDefinitionArn *string  `locationName:"currentTaskDefinitionArn" type:"string" required:"true"`
	NextTaskDefinitionArn    *string  `locationName:"nextTaskDefinitionArn" type:"string" required:"true"`
	AvailabilityThreshold    *float64 `locationName:"availabilityThreshold" type:"double"`
	ResponseTimeThreshold    *float64 `locationName:"responseTimeThreshold" type:"double"`
	RollOutPeriod            *int64   `locationName:"rollOutPeriod" type:"integer"`
}

const kRegionKey = "CAGE_AWS_REGION"
const kServiceKey = "CAGE_ECS_SERVICE"
const kClusterKey = "CAGE_ECS_CLUSTER"
const kCurrentTaskDefinitionArnKey = "CAGE_CURRENT_TASK_DEFINITION_ARN"
const kNextTaskDefinitionArnKey = "CAGE_NEXT_TASK_DEFINITION_ARN"
const kLoadBalancerArnKey = "CAGE_LB_ARN"
const kAvailabilityThresholdKey = "CAGE_AVAILABILITY_THRESHOLD"
const kResponseTimeThresholdKey = "CAGE_RESPONSE_TIME_THRESHOLD"
const kRollOutPeriodKey = "CAGE_ROLL_OUT_PERIOD"

const kAvailabilityThresholdDefaultValue = 0.9970
const kResponseTimeThresholdDefaultValue = 1.0
const kRollOutPeriodDefaultValue = 300
const kDefaultRegion = "us-west-2"

func EnsureEnvars(
	dest *Envars,
) (error) {
	// required
	if dest.Cluster == nil ||
		dest.LoadBalancerArn == nil ||
		dest.ServiceName == nil ||
		dest.CurrentTaskDefinitionArn == nil ||
		dest.NextTaskDefinitionArn == nil {
		return errors.New(fmt.Sprintf("some required envars are not defined: %#v", *dest))
	}
	if dest.Region == nil {
		dest.Region = aws.String(kDefaultRegion)
	}
	if dest.AvailabilityThreshold == nil {
		dest.AvailabilityThreshold = aws.Float64(kAvailabilityThresholdDefaultValue)
	}
	if avl := *dest.AvailabilityThreshold; !(0.0 <= avl && avl <= 1.0) {
		return errors.New(fmt.Sprintf("CAGE_AVAILABILITY_THRESHOLD must be between 0 and 1, but got '%f'", avl))
	}
	if dest.ResponseTimeThreshold == nil {
		dest.ResponseTimeThreshold = aws.Float64(kResponseTimeThresholdDefaultValue)
	}
	if rsp := *dest.ResponseTimeThreshold; !(0 < rsp && rsp <= 300) {
		return errors.New(fmt.Sprintf("CAGE_RESPONSE_TIME_THRESHOLD must be greater than 0, but got '%f'", rsp))
	}
	// sec
	if dest.RollOutPeriod == nil {
		dest.RollOutPeriod = aws.Int64(kRollOutPeriodDefaultValue)
	}
	if period := *dest.RollOutPeriod; !(60 <= period && float64(period) != math.NaN() && float64(period) != math.Inf(0)) {
		return errors.New(fmt.Sprintf("CAGE_ROLLOUT_PERIOD must be lesser than 60, but got '%d'", period))
	}
	return nil
}
