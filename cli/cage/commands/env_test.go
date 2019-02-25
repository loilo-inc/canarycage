package commands

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnsureEnvars(t *testing.T) {
	e := &Envars{
		Cluster:        "cluster",
		Service:        "service-next",
		taskDefinition: &ecs.RegisterTaskDefinitionInput{},
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars4(t *testing.T) {
	e := &Envars{
		Cluster: "cluster",
		Service: "next",
	}
	if err := EnsureEnvars(e); err != nil {
		t.Fatalf(err.Error())
	}
}

func TestEnsureEnvars2(t *testing.T) {
	// 必須環境変数がなければエラー
	dummy := "aaa"
	arr := []string{
		ServiceKey,
		ClusterKey,
	}
	for i, v := range arr {
		m := make(map[string]string)
		m[ServiceKey] = dummy
		m[TaskDefinitionArnKey] = dummy
		m[ClusterKey] = dummy
		for j, u := range arr {
			if i == j {
				m[u] = ""
			}
		}
		e := &Envars{
			Service: m[ServiceKey],
			Cluster: m[ClusterKey],
		}
		err := EnsureEnvars(e)
		if err == nil {
			t.Fatalf("should return error if %s is not defined", v)
		}
	}
}

func dummyEnvs() *Envars {
	dummy := "aaa"
	return &Envars{
		Service: dummy,
		Cluster: dummy,
	}
}

func TestEnvars_Merge(t *testing.T) {
	e1 := &Envars{
		Region:  "us-west-2",
		Cluster: "cluster",
	}
	e2 := &Envars{
		Cluster: "hoge",
		Service: "fuga",
	}
	MergeEnvars(e1, e2)
	assert.Equal(t, e1.Region, "us-west-2")
	assert.Equal(t, e1.Cluster, "hoge")
	assert.Equal(t, e1.Service, "fuga")
}
