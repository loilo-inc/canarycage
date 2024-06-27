package env_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/loilo-inc/canarycage/env"
	"github.com/stretchr/testify/assert"
)

func TestEnsureEnvars(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		e := &env.Envars{
			Region:              "us-west-2",
			Cluster:             "cluster",
			Service:             "service-next",
			TaskDefinitionInput: &ecs.RegisterTaskDefinitionInput{},
		}
		if err := env.EnsureEnvars(e); err != nil {
			t.Fatalf(err.Error())
		}
	})
	t.Run("with td arn", func(t *testing.T) {
		e := &env.Envars{
			Region:            "us-west-2",
			Cluster:           "cluster",
			Service:           "next",
			TaskDefinitionArn: "arn://aaa",
		}
		if err := env.EnsureEnvars(e); err != nil {
			t.Fatalf(err.Error())
		}
	})
	t.Run("should return err if nor taskDefinitionArn neither TaskDefinitionInput is defined", func(t *testing.T) {
		e := &env.Envars{
			Region:  "us-west-2",
			Cluster: "cluster",
			Service: "next",
		}
		err := env.EnsureEnvars(e)
		assert.Errorf(t, err, "--nextTaskDefinitionArn or deploy context must be provided")
	})
	t.Run("should return err if required props are not defined", func(t *testing.T) {
		dummy := "aaa"
		arr := []string{
			env.RegionKey,
			env.ServiceKey,
			env.ClusterKey,
		}
		for i, v := range arr {
			m := make(map[string]string)
			m[env.ServiceKey] = dummy
			m[env.TaskDefinitionArnKey] = dummy
			m[env.ClusterKey] = dummy
			for j, u := range arr {
				if i == j {
					m[u] = ""
				}
			}
			e := &env.Envars{
				Service: m[env.ServiceKey],
				Cluster: m[env.ClusterKey],
			}
			err := env.EnsureEnvars(e)
			if err == nil {
				t.Fatalf("should return error if %s is not defined", v)
			}
		}
	})
}

func TestMergeEnvars(t *testing.T) {
	e1 := &env.Envars{
		Region:  "us-west-2",
		Cluster: "cluster",
	}
	e2 := &env.Envars{
		Cluster: "hoge",
		Service: "fuga",
	}
	env.MergeEnvars(e1, e2)
	assert.Equal(t, e1.Region, "us-west-2")
	assert.Equal(t, e1.Cluster, "hoge")
	assert.Equal(t, e1.Service, "fuga")
}
