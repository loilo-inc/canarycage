package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"time"
	"github.com/apex/log"
)

func newLookupper() Lookupper {
	return Lookupper{
		get: func(s string) string {
			if s == "PATH" {
				return "path"
			} else {
				return ""
			}
		},
		lookup: func(s string) (string, bool) {
			if s == "PATH" {
				return "path", true
			} else if s == "EMPTY" {
				return "", true
			} else {
				return "", false
			}
		},
	}
}
func newLookupperWithMap(m *map[string]string) Lookupper {
	return Lookupper{
		get: func(s string) string {
			if o, ok := (*m)[s]; ok {
				return o
			} else {
				return ""
			}
		},
		lookup: func(s string) (string, bool) {
			if o, ok := (*m)[s]; ok {
				return o, ok
			} else {
				return "", ok
			}

		},
	}
}
func TestLookupper_InvariantEnvars(t *testing.T) {
	l := newLookupper()
	if err := l.InvariantEnvars("PATH"); err != nil {
		t.Fatalf(err.Error())
	}
	if err := l.InvariantEnvars("HOGEEE"); err == nil {
		t.Fatalf("should return error if specified envar isn't defined")
	}
}

func TestLookupEnv(t *testing.T) {
	l := newLookupper()
	path := l.get("PATH")
	if o := l.LookupEnv("PATH", "way"); o != path {
		t.Fatalf("E: %s, A: %s", path, o)
	}
	if o := l.LookupEnv("WAY", "way"); o != "way" {
		t.Fatalf("E: %s, A: %s", "way", o)
	}
}

func TestEnsureEnvars(t *testing.T) {
	envs := make(map[string]string)
	envs[kCurrentServiceNameKey] = "service-current"
	envs[kClusterKey] = "cluster"
	envs[kCurrentTaskDefinitionArnKey] = "arn://task-current"
	envs[kNextServiceDefinitionBase64Key] = "abcde"
	envs[kNextTaskDefinitionBase64Key] = "fghij"
	envs[kServiceLoadBalancerArnKey] = "arn://lb"
	envs[kAvailabilityThresholdKey] = "0.9"
	envs[kResponseTimeThresholdKey] = "0.5"
	envs[kRollOutPeriodKey] = "200"
	l := newLookupperWithMap(&envs)
	e, err := EnsureEnvars(l.get, l.lookup)
	if err != nil {
		t.Fatalf(err.Error())
	}
	assert.Equal(t, e.Cluster, envs[kClusterKey])
	assert.Equal(t, e.CurrentServiceName, envs[kCurrentServiceNameKey])
	assert.Equal(t, e.CurrentTaskDefinitionArn, envs[kCurrentTaskDefinitionArnKey])
	assert.Equal(t, e.NextServiceDefinitionBase64, envs[kNextServiceDefinitionBase64Key])
	assert.Equal(t, e.NextTaskDefinitionBase64, envs[kNextTaskDefinitionBase64Key])
	assert.Equal(t, e.LoadBalancerArn, envs[kServiceLoadBalancerArnKey])
	assert.Equal(t, e.AvailabilityThreshold, 0.9)
	assert.Equal(t, e.ResponseTimeThreshold, 0.5)
	assert.Equal(t, e.RollOutPeriod, time.Duration(200)*time.Second)
}

func TestEnsureEnvars2(t *testing.T) {
	// 必須環境変数がなければエラー
	arr := []string{
		kCurrentServiceNameKey,
		kCurrentTaskDefinitionArnKey,
		kNextServiceDefinitionBase64Key,
		kNextTaskDefinitionBase64Key,
		kClusterKey,
		kServiceLoadBalancerArnKey,
	}
	m := make(map[string]string)
	l := newLookupperWithMap(&m)
	for i, v := range arr {
		for j, u := range arr {
			if i == j {
				delete(m, u)
			} else {
				m[u] = "ok"
			}
		}
		_, err := EnsureEnvars(l.get, l.lookup)
		if err == nil {
			t.Fatalf("should return error if %s is not defined: %s", v, m[v])
		}
	}
}

func dummyEnvs() map[string]string {
	m := make(map[string]string)
	m[kCurrentServiceNameKey] = "hoge"
	m[kCurrentTaskDefinitionArnKey] = "hoge"
	m[kNextServiceDefinitionBase64Key] = "hoge"
	m[kNextTaskDefinitionBase64Key] = "hoge"
	m[kClusterKey] = "hoge"
	m[kServiceLoadBalancerArnKey] = "hoge"
	return m
}
func TestEnsureEnvars3(t *testing.T) {
	// availabilityがおかしい
	log.SetLevel(log.DebugLevel)
	arr := []string{"-1.0", "1.1", "NaN", "Infinity", "way", ""}
	for _, v := range arr {
		m := dummyEnvs()
		m[kAvailabilityThresholdKey] = v
		l := newLookupperWithMap(&m)
		if _, err := EnsureEnvars(l.get, l.lookup); err == nil {
			t.Fatalf("should return error if availability is invalid: %s", v)
		}
	}
	arr = []string {"0", "NaN", "Infinity", "way", ""}
	for _, v := range arr {
		m := dummyEnvs()
		m[kResponseTimeThresholdKey] = v
		l := newLookupperWithMap(&m)
		if _, err := EnsureEnvars(l.get, l.lookup); err == nil {
			t.Fatalf("should return error if responsen time is invalid: %s", v)
		}
	}
	arr = []string { "0", "59", "NaN", "Infinity", "way", ""}
	for _ , v := range arr {
		m := dummyEnvs()
		m[kRollOutPeriodKey] = v
		l := newLookupperWithMap(&m)
		if _, err := EnsureEnvars(l.get, l.lookup); err == nil {
			t.Fatalf("should return error if roll out period is invalid: %s", v)
		}
	}
}
