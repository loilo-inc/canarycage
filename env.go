package main

import (
	"github.com/pkg/errors"
	"fmt"
	"strconv"
	"time"
	"github.com/apex/log"
	"math"
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

type Lookupper struct {
	lookup func(string) (string, bool)
	get    func(string) (string)
}

func (l *Lookupper) LookupEnv(key string, defaultValue string) (string) {
	if v, ok := l.lookup(key); ok {
		return v
	} else {
		return defaultValue
	}
}
func (l *Lookupper) InvariantEnvars(keys ...string) error {
	for _, v := range keys {
		if _, ok := l.lookup(v); !ok {
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

func EnsureEnvars(
	getEnv func(string) string,
	lookupEnv func(string) (string, bool),
) (*Envars, error) {
	l := Lookupper{
		get:    getEnv,
		lookup: lookupEnv,
	}
	if err := l.InvariantEnvars(
		kCurrentServiceNameKey,
		kCurrentTaskDefinitionArnKey,
		kNextServiceDefinitionBase64Key,
		kNextTaskDefinitionBase64Key,
		kClusterKey,
		kServiceLoadBalancerArnKey,
	); err != nil {
		log.Errorf("some required envars are not defined: %s", err)
		return nil, err
	}
	avl, err := strconv.ParseFloat(l.LookupEnv(kAvailabilityThresholdKey, "0.9970"), 64)
	if err != nil {
		return nil, err
	} else if !(0.0 <= avl && avl <= 1.0) {
		return nil, errors.New(fmt.Sprintf("CAGE_AVAILABILITY_THRESHOLD must be between 0 and 1, but got '%f'", avl))
	}
	resp, err := strconv.ParseFloat(l.LookupEnv(kResponseTimeThresholdKey, "1.000"), 64)
	if err != nil {
		return nil, err
	} else if !(0 < resp && resp <= 300) {
		return nil, errors.New(fmt.Sprintf("CAGE_RESPONSE_TIME_THRESHOLD must be greater than 0, but got '%f'", resp))
	}
	// sec
	period, err := strconv.ParseFloat(l.LookupEnv(kRollOutPeriodKey, "300"), 64)
	if err != nil {
		return nil, err
	} else if !(60 <= period && period != math.NaN() && period != math.Inf(0)) {
		return nil, errors.New(fmt.Sprintf("CAGE_ROLLOUT_PERIOD must be lesser than 60, but got '%f'", period))
	}
	return &Envars{
		Region:                      l.LookupEnv("CAGE_AWS_REGION", "us-west-2"),
		ReleaseStage:                l.LookupEnv("CAGE_RELEASE_STAGE", "local"),
		RollOutPeriod:               time.Duration(period) * time.Second,
		LoadBalancerArn:             getEnv(kServiceLoadBalancerArnKey),
		Cluster:                     getEnv(kClusterKey),
		CurrentServiceName:          getEnv(kCurrentServiceNameKey),
		CurrentTaskDefinitionArn:    getEnv(kCurrentTaskDefinitionArnKey),
		NextServiceDefinitionBase64: getEnv(kNextServiceDefinitionBase64Key),
		NextTaskDefinitionBase64:    getEnv(kNextTaskDefinitionBase64Key),
		AvailabilityThreshold:       avl,
		ResponseTimeThreshold:       resp,
	}, nil
}
