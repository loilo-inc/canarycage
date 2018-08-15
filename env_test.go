package main

import (
	"testing"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws"
	"math"
	"io/ioutil"
	"encoding/json"
	"github.com/stretchr/testify/assert"
)

func TestEnsureEnvars(t *testing.T) {
	e := &Envars{
		Cluster:                     aws.String("cluster"),
		NextServiceName:             aws.String("service-next"),
		CurrentServiceName:          aws.String("service-current"),
		NextTaskDefinitionBase64:    aws.String("hoge"),
		NextServiceDefinitionBase64: aws.String("next"),
		LoadBalancerArn:             aws.String("lb"),
		AvailabilityThreshold:       aws.Float64(0.9),
		ResponseTimeThreshold:       aws.Float64(0.5),
		RollOutPeriod:               aws.Int64(60),
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars4(t *testing.T) {
	e := &Envars{
		Cluster:                  aws.String("cluster"),
		CurrentServiceName:       aws.String("service"),
		NextTaskDefinitionBase64: aws.String("current"),
		NextServiceName:          aws.String("next"),
		LoadBalancerArn:          aws.String("lb"),
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars2(t *testing.T) {
	// 必須環境変数がなければエラー
	dummy := aws.String("aaa")
	arr := []string{
		kNextServiceNameKey,
		kCurrentServiceNameKey,
		kNextTaskDefinitionBase64Key,
		kClusterKey,
		kLoadBalancerArnKey,
	}
	for i, v := range arr {
		m := make(map[string]*string)
		m[kNextServiceNameKey] = dummy
		m[kCurrentServiceNameKey] = dummy
		m[kNextTaskDefinitionBase64Key] = dummy
		m[kClusterKey] = dummy
		m[kLoadBalancerArnKey] = dummy
		for j, u := range arr {
			if i == j {
				m[u] = nil
			}
		}
		e := &Envars{
			CurrentServiceName:       m[kCurrentServiceNameKey],
			NextServiceName:          m[kNextServiceNameKey],
			NextTaskDefinitionBase64: m[kNextTaskDefinitionBase64Key],
			Cluster:                  m[kClusterKey],
			LoadBalancerArn:          m[kLoadBalancerArnKey],
		}
		err := EnsureEnvars(e)
		if err == nil {
			t.Fatalf("should return error if %s is not defined: %s", v, m[v])
		}
	}
}

func dummyEnvs() *Envars {
	dummy := aws.String("aaa")
	return &Envars{
		CurrentServiceName:       dummy,
		NextServiceName:          dummy,
		NextTaskDefinitionBase64: dummy,
		Cluster:                  dummy,
		LoadBalancerArn:          dummy,
	}
}
func TestEnsureEnvars3(t *testing.T) {
	// availabilityがおかしい
	log.SetLevel(log.DebugLevel)
	arr := []float64{-1.0, 1.1, math.NaN(), math.Inf(0), math.Inf(-1)}
	for _, v := range arr {
		e := dummyEnvs()
		e.AvailabilityThreshold = aws.Float64(v)
		if err := EnsureEnvars(e); err == nil {
			t.Fatalf("should return error if availability is invalid: %f", v)
		}
	}
	for _, v := range []float64{0, math.NaN(), math.Inf(0), math.Inf(-1)} {
		e := dummyEnvs()
		e.ResponseTimeThreshold = aws.Float64(v)
		if err := EnsureEnvars(e); err == nil {
			t.Fatalf("should return error if responsen time is invalid: %f", v)
		}
	}
	for _, v := range []int64{0, 59, int64(math.NaN()), int64(math.Inf(0)), int64(math.Inf(-1))} {
		e := dummyEnvs()
		e.RollOutPeriod = aws.Int64(v)
		if err := EnsureEnvars(e); err == nil {
			t.Fatalf("should return error if roll out period is invalid: %d", v)
		}
	}
}

func TestUnmarshalEnvars(t *testing.T) {
	// jsonからも読み込める
	d, _ := ioutil.ReadFile("fixtures/envars.json")
	dest := Envars{}
	err := json.Unmarshal(d, &dest)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, "us-east-2", *dest.Region)
	assert.Equal(t, "cluster", *dest.Cluster)
	assert.Equal(t, "arn://lb", *dest.LoadBalancerArn)
	assert.Equal(t, "service-next", *dest.NextServiceName)
	assert.Equal(t, "service-current", *dest.CurrentServiceName)
	assert.Equal(t, "next-task", *dest.NextTaskDefinitionBase64)
	assert.Equal(t, "next-service", *dest.NextServiceDefinitionBase64)
	assert.Equal(t, 0.9999, *dest.AvailabilityThreshold)
	assert.Equal(t, 1.2, *dest.ResponseTimeThreshold)
	assert.Equal(t, int64(100), *dest.RollOutPeriod)
	assert.Equal(t, int64(61), *dest.UpdateServicePeriod)
	assert.Equal(t, int64(301), *dest.UpdateServiceTimeout)
}
