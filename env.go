package main

import (
	"github.com/pkg/errors"
	"fmt"
	"math"
	"github.com/aws/aws-sdk-go/aws"
)

type Envars struct {
	_                           struct{} `type:"struct"`
	Region                      *string  `locationName:"region" type:"string"`
	Cluster                     *string  `locationName:"cluster" type:"string" required:"true"`
	LoadBalancerArn             *string  `locationName:"loadBalancerArn" type:"string" required:"true"`
	NextServiceName             *string  `locationName:"nextServiceName" type:"string" required:"true"`
	CurrentServiceName          *string  `locationName:"currentServiceName" type:"string" required:"true"`
	NextServiceDefinitionBase64 *string  `locationName:"serviceName" type:"string" required:"true"`
	NextTaskDefinitionBase64    *string  `locationName:"serviceName" type:"string" required:"true"`
	AvailabilityThreshold       *float64 `locationName:"availabilityThreshold" type:"double"`
	ResponseTimeThreshold       *float64 `locationName:"responseTimeThreshold" type:"double"`
	RollOutPeriod               *int64   `locationName:"rollOutPeriod" type:"integer"`
	UpdateServicePeriod         *int64   `locationName:"updateServicePeriod" type:"integer"`
	UpdateServiceTimeout        *int64   `locationName:"updateServiceTimeout" type:"integer"`
}

// required
const kClusterKey = "CAGE_ECS_CLUSTER"
const kNextServiceNameKey = "CAGE_NEXT_SERVICE_NAME"
const kCurrentServiceNameKey = "CAGE_CURRENT_SERVICE_NAME"
const kLoadBalancerArnKey = "CAGE_LB_ARN"
const kNextTaskDefinitionBase64Key = "CAGE_NEXT_TASK_DEFINITION_BASE64"

// optional
const kConfigFileKey = "CAGE_CONFIG_FILE"
const kNextServiceDefinitionBase64Key = "CAGE_NEXT_SERVICE_DEFINITION_BASE64"
const kRegionKey = "CAGE_AWS_REGION"
const kAvailabilityThresholdKey = "CAGE_AVAILABILITY_THRESHOLD"
const kResponseTimeThresholdKey = "CAGE_RESPONSE_TIME_THRESHOLD"
const kRollOutPeriodKey = "CAGE_ROLL_OUT_PERIOD"
const kUpdateServicePeriodKey = "CAGE_UPDATE_SERVICE_PERIOD"
const kUpdateServiceTimeoutKey = "CAGE_UPDATE_SERVICE_TIMEOUT"

const kAvailabilityThresholdDefaultValue = 0.9970
const kResponseTimeThresholdDefaultValue = 1.0
const kRollOutPeriodDefaultValue = 300
const kUpdateServicePeriodDefaultValue = 60
const kUpdateServiceTimeoutDefaultValue = 300
const kDefaultRegion = "us-west-2"

func isEmpty(o *string) bool {
	return o == nil || *o == ""
}

func EnsureEnvars(
	dest *Envars,
) (error) {
	// required
	if isEmpty(dest.Cluster) ||
		isEmpty(dest.LoadBalancerArn) ||
		isEmpty(dest.CurrentServiceName) ||
		isEmpty(dest.NextServiceName) ||
		isEmpty(dest.NextTaskDefinitionBase64) {
		return errors.New(fmt.Sprintf("some required envars are not defined: %#v", *dest))
	}
	if isEmpty(dest.Region) {
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
	if dest.UpdateServicePeriod == nil {
		dest.UpdateServicePeriod = aws.Int64(kUpdateServicePeriodDefaultValue)
	}
	if *dest.UpdateServicePeriod < 60 {
		return errors.New(fmt.Sprintf("%s must be greater than or equal to 60", kUpdateServicePeriodKey))
	}
	if dest.UpdateServiceTimeout == nil {
		dest.UpdateServiceTimeout = aws.Int64(kUpdateServiceTimeoutDefaultValue)
	}
	if v := *dest.UpdateServiceTimeout; v < *dest.UpdateServicePeriod {
		return errors.New(fmt.Sprintf(
			"%s must be grater than %s: %d, %d",
			kUpdateServiceTimeoutKey, kUpdateServicePeriodKey, v, *dest.UpdateServicePeriod,
		))
	}
	return nil
}
