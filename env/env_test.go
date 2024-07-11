package env_test

import (
	"os"
	"testing"

	"github.com/apex/log"
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

func TestLoadServiceDefinition(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d, err := env.LoadServiceDefinition("../fixtures")
		if err != nil {
			t.Fatalf(err.Error())
		}
		assert.Equal(t, *d.ServiceName, "service")
	})
	t.Run("should error if service.json is not found", func(t *testing.T) {
		_, err := env.LoadServiceDefinition("./testdata")
		assert.EqualError(t, err, "no 'service.json' found in ./testdata")
	})
	t.Run("should error if service.json is invalid", func(t *testing.T) {
		_, err := env.LoadServiceDefinition("./testdata/invalid")
		assert.ErrorContains(t, err, "failed to read and unmarshal 'service.json':")
	})
}

func TestLoadTaskDefinition(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d, err := env.LoadTaskDefinition("../fixtures")
		if err != nil {
			t.Fatalf(err.Error())
		}
		assert.Equal(t, *d.Family, "test-task")
	})
	t.Run("should error if task-definition.json is not found", func(t *testing.T) {
		_, err := env.LoadTaskDefinition("./testdata")
		assert.EqualError(t, err, "no 'task-definition.json' found in ./testdata")
	})
	t.Run("should error if task-definition.json is invalid", func(t *testing.T) {
		_, err := env.LoadTaskDefinition("./testdata/invalid")
		assert.ErrorContains(t, err, "failed to read and unmarshal 'task-definition.json':")
	})
}

func TestReadFileAndApplyEnvars(t *testing.T) {
	os.Setenv("HOGE", "hogehoge")
	os.Setenv("FUGA", "fugafuga")
	d, err := env.ReadFileAndApplyEnvars("./testdata/template.txt")
	if err != nil {
		t.Fatalf(err.Error())
	}
	s := string(d)
	e := `HOGE=hogehoge
FUGA=fugafuga
fugafuga=hogehoge`
	if s != e {
		log.Fatalf("e: %s, a: %s", e, s)
	}
}
