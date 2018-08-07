package main

import (
	"os"
	"github.com/pkg/errors"
	"fmt"
	"strconv"
	"time"
)

type Envars struct {
	Region                      string
	ReleaseStage                string
	RollOutPeriod               time.Duration
	LoadBalancerArn             string
	Cluster                     string
	CurrentServiceName          string
	CurrentTaskDefinitionArn    string
	NextTaskDefinitionBase64    string
	NextServiceDefinitionBase64 string
	NextServiceName             string
	AvailabilityThreshold       float64
	ResponseTimeThreshold       float64
}

func LookupEnv(key string, defaultValue string) (string) {
	if v, ok := os.LookupEnv(key); ok {
		return v
	} else {
		return defaultValue
	}
}
func InvariantEnvars(keys ...string) error {
	for _, v := range keys {
		if _, ok := os.LookupEnv(v); !ok {
			return errors.New(fmt.Sprintf("required envar %s is not defined", v))
		}
	}
	return nil
}

const kCurrentServiceNameKey = "CAGE_CURRENT_SERVICE_NAME"
const kCurrentTaskDefinitionArnKey = "CAGE_CURRENT_TASK_DEFINITION_ARN"
const kNextTaskDefinitionBase64Key = "CAGE_NEXT_TASK_DEFINITION_BASE64"
const kNextServiceDefinitionBase64Key = "CAGE_NEXT_SERVICE_DEFINITION_BASE64"
const kClusterKey = "CAGE_AWS_ECS_CLUSTER"
const kServiceLoadBalancerArnKey = "CAGE_LB_ARN"
const kAvailabilityThresholdKey = "CAGE_AVAILABILITY_THRESHOLD"
const kResponseTimeThresholdKey = "CAGE_RESPONSE_TIME_THRESHOLD"
const kRollOutPeriodKey = "CAGE_ROLL_OUT_PERIOD"

func EnsureEnvars() (*Envars, error) {
	InvariantEnvars(
		kCurrentServiceNameKey,
		kCurrentTaskDefinitionArnKey,
		kNextServiceDefinitionBase64Key,
		kNextTaskDefinitionBase64Key,
		kClusterKey,
		kServiceLoadBalancerArnKey,
	)
	avl, err := strconv.ParseFloat(LookupEnv(kAvailabilityThresholdKey, "0.9970"), 64)
	if err != nil {
		return nil, err
	} else if avl < 0 || 1 < avl {
		return nil, errors.New(fmt.Sprintf("CAGE_AVAILABILITY_THRESHOLD must be between 0 and 1, but got '%f'", avl))
	}
	resp, err := strconv.ParseFloat(LookupEnv(kResponseTimeThresholdKey, "1.000"), 64)
	if err != nil {
		return nil, err
	} else if resp < 0 {
		return nil, errors.New(fmt.Sprintf("CAGE_RESPONSE_TIME_THRESHOLD must be greater than 0, but got '%f'", resp))
	}
	// sec
	period, err := strconv.ParseFloat(LookupEnv(kRollOutPeriodKey, "300"), 64)
	if err != nil {
		return nil, err
	} else if period < 60 {
		return nil, errors.New(fmt.Sprintf("CAGE_ROLLOUT_PERIOD must be lesser than 60, but got '%f'", period))
	}
	return &Envars{
		Region:                      LookupEnv("CAGE_AWS_REGION", "us-west-2"),
		ReleaseStage:                LookupEnv("CAGE_RELEASE_STAGE", "local"),
		RollOutPeriod:               time.Duration(period) * time.Second,
		LoadBalancerArn:             os.Getenv(kServiceLoadBalancerArnKey),
		Cluster:                     os.Getenv(kClusterKey),
		CurrentServiceName:          os.Getenv(kCurrentServiceNameKey),
		CurrentTaskDefinitionArn:    os.Getenv(kCurrentTaskDefinitionArnKey),
		NextServiceDefinitionBase64: os.Getenv(kCurrentServiceNameKey),
		NextTaskDefinitionBase64:    os.Getenv(kNextTaskDefinitionBase64Key),
		AvailabilityThreshold:       avl,
		ResponseTimeThreshold:       resp,
	}, nil
}
