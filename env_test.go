package cage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEnsureEnvars(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		e := &Envars{
			Region:              "us-west-2",
			Cluster:             "cluster",
			Service:             "service-next",
			TaskDefinitionInput: &ecs.RegisterTaskDefinitionInput{},
		}
		if err := EnsureEnvars(e); err != nil {
			t.Fatalf(err.Error())
		}
	})
	t.Run("with td arn", func(t *testing.T) {
		e := &Envars{
			Region:            "us-west-2",
			Cluster:           "cluster",
			Service:           "next",
			TaskDefinitionArn: aws.String("arn://aaa"),
		}
		if err := EnsureEnvars(e); err != nil {
			t.Fatalf(err.Error())
		}
	})
	t.Run("should return err if nor taskDefinitionArn neither TaskDefinitionInput is defined", func(t *testing.T) {
		e := &Envars{
			Region: "us-west-2",
			Cluster: "cluster",
			Service: "next",
		}
		if err := EnsureEnvars(e); err == nil {
			t.Fatalf(err.Error())
		}
	})
	t.Run("should return err if required props are not defined", func(t *testing.T) {
		dummy := "aaa"
		arr := []string{
			RegionKey,
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
	})
}

func TestMergeEnvars(t *testing.T) {
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
