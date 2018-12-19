package cage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnsureEnvars(t *testing.T) {
	e := &Envars{
		Cluster:                 aws.String("cluster"),
		Service:                 aws.String("service-next"),
		TaskDefinitionBase64:    aws.String("hoge"),
		ServiceDefinitionBase64: aws.String("next"),
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars4(t *testing.T) {
	e := &Envars{
		Cluster:              aws.String("cluster"),
		TaskDefinitionBase64: aws.String("current"),
		Service:              aws.String("next"),
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars2(t *testing.T) {
	// 必須環境変数がなければエラー
	dummy := aws.String("aaa")
	arr := []string{
		ServiceKey,
		TaskDefinitionBase64Key,
		ClusterKey,
	}
	for i, v := range arr {
		m := make(map[string]*string)
		m[ServiceKey] = dummy
		m[TaskDefinitionArnKey] = dummy
		m[ClusterKey] = dummy
		for j, u := range arr {
			if i == j {
				m[u] = nil
			}
		}
		e := &Envars{
			Service:              m[ServiceKey],
			TaskDefinitionBase64: m[TaskDefinitionBase64Key],
			Cluster:              m[ClusterKey],
		}
		err := EnsureEnvars(e)
		if err == nil {
			t.Fatalf("should return error if %s is not defined", v)
		}
	}
}

func dummyEnvs() *Envars {
	dummy := aws.String("aaa")
	return &Envars{
		Service:              dummy,
		TaskDefinitionBase64: dummy,
		Cluster:              dummy,
	}
}

func TestEnvars_Merge(t *testing.T) {
	e1 := &Envars{
		Region: aws.String("us-west-2"),
		Cluster: aws.String("cluster"),
		CanaryService: aws.String("canary"),
	}
	e2 := &Envars {
		Cluster: aws.String("hoge"),
		Service: aws.String("fuga"),
		CanaryService: aws.String(""),
	}
	err := e1.Merge(e2)
	assert.Nil(t, err)
	assert.Equal(t, *e1.Region, "us-west-2")
	assert.Equal(t, *e1.Cluster, "hoge")
	assert.Equal(t, *e1.Service, "fuga")
	assert.Equal(t, *e1.CanaryService, "canary")
}