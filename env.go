package cage

import (
	"math"
	"github.com/aws/aws-sdk-go/aws"
)

type Envars struct {
	_                           struct{} `type:"struct"`
	Region                      *string  `json:"region" type:"string"`
	Cluster                     *string  `json:"cluster" type:"string" required:"true"`
	NextServiceName             *string  `json:"nextServiceName" type:"string" required:"true"`
	CurrentServiceName          *string  `json:"currentServiceName" type:"string" required:"true"`
	NextServiceDefinitionBase64 *string  `json:"nextServiceDefinitionBase64" type:"string"`
	NextTaskDefinitionBase64    *string  `json:"nextTaskDefinitionBase64" type:"string"`
	NextTaskDefinitionArn       *string  `json:"nextTaskDefinitionArn" type:"string"`
	AvailabilityThreshold       *float64 `json:"availabilityThreshold" type:"double"`
	ResponseTimeThreshold       *float64 `json:"responseTimeThreshold" type:"double"`
	RollOutPeriod               *int64   `json:"rollOutPeriod" type:"integer"`
}

// required
const ClusterKey = "CAGE_ECS_CLUSTER"
const NextServiceNameKey = "CAGE_NEXT_SERVICE_NAME"
const CurrentServiceNameKey = "CAGE_CURRENT_SERVICE_NAME"

// either required
const NextTaskDefinitionBase64Key = "CAGE_NEXT_TASK_DEFINITION_BASE64"
const NextTaskDefinitionArnKey = "CAGE_NEXT_TASK_DEFINITION_ARN"

// optional
const ConfigKey = "CAGE_CONFIG"
const NextServiceDefinitionBase64Key = "CAGE_NEXT_SERVICE_DEFINITION_BASE64"
const RegionKey = "CAGE_AWS_REGION"
const AvailabilityThresholdKey = "CAGE_AVAILABILITY_THRESHOLD"
const ResponseTimeThresholdKey = "CAGE_RESPONSE_TIME_THRESHOLD"
const RollOutPeriodKey = "CAGE_ROLL_OUT_PERIOD"

const kAvailabilityThresholdDefaultValue = 0.9970
const kResponseTimeThresholdDefaultValue = 1.0
const kRollOutPeriodDefaultValue = 300
const kDefaultRegion = "us-west-2"

func isEmpty(o *string) bool {
	return o == nil || *o == ""
}

func EnsureEnvars(
	dest *Envars,
) (error) {
	// required
	if isEmpty(dest.Cluster) {
		return NewErrorf("--cluster [%s] is required", ClusterKey)
	} else if isEmpty(dest.CurrentServiceName) {
		return NewErrorf("--currentServiceName [%s] is required", CurrentServiceNameKey)
	} else if isEmpty(dest.NextServiceName) {
		return NewErrorf("--nextServiceName [%s] is required", NextServiceNameKey)
	}
	if isEmpty(dest.NextTaskDefinitionArn) && isEmpty(dest.NextTaskDefinitionBase64) {
		return NewErrorf("--nextTaskDefinitionArn or --nextTaskDefinitionBase64 must be provided")
	}
	if isEmpty(dest.Region) {
		dest.Region = aws.String(kDefaultRegion)
	}
	if dest.AvailabilityThreshold == nil {
		dest.AvailabilityThreshold = aws.Float64(kAvailabilityThresholdDefaultValue)
	}
	if avl := *dest.AvailabilityThreshold; !(0.0 <= avl && avl <= 1.0) {
		return NewErrorf("--availabilityThreshold [%s] must be between 0 and 1, but got '%f'", AvailabilityThresholdKey, avl)
	}
	if dest.ResponseTimeThreshold == nil {
		dest.ResponseTimeThreshold = aws.Float64(kResponseTimeThresholdDefaultValue)
	}
	if rsp := *dest.ResponseTimeThreshold; !(0 < rsp && rsp <= 300) {
		return NewErrorf("--responseTimeThreshold [%s] must be greater than 0, but got '%f'", ResponseTimeThresholdKey, rsp)
	}
	// sec
	if dest.RollOutPeriod == nil {
		dest.RollOutPeriod = aws.Int64(kRollOutPeriodDefaultValue)
	}
	if period := *dest.RollOutPeriod; !(60 <= period && float64(period) != math.NaN() && float64(period) != math.Inf(0)) {
		return NewErrorf("--rollOutPeriod [%s] must be lesser than 60, but got '%d'", RollOutPeriodKey, period)
	}
	return nil
}
