package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/stretchr/testify/assert"
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
			t.Fatal(err.Error())
		}
	})
	t.Run("with td arn", func(t *testing.T) {
		e := &Envars{
			Region:            "us-west-2",
			Cluster:           "cluster",
			Service:           "next",
			TaskDefinitionArn: "arn://aaa",
		}
		if err := EnsureEnvars(e); err != nil {
			t.Fatal(err.Error())
		}
	})
	t.Run("should return err if nor taskDefinitionArn neither TaskDefinitionInput is defined", func(t *testing.T) {
		e := &Envars{
			Region:  "us-west-2",
			Cluster: "cluster",
			Service: "next",
		}
		err := EnsureEnvars(e)
		assert.EqualError(t, err, "--nextTaskDefinitionArn or deploy context must be provided")
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

func TestApplyEnvarsToString(t *testing.T) {
	const path = "task-definition.json"

	t.Run("replaces envar literals", func(t *testing.T) {
		t.Setenv("IMAGE_REPOSITORY", "repo/app")
		t.Setenv("IMAGE_TAG", "v1.2.3")

		got, err := applyEnvarsToString("${IMAGE_REPOSITORY}:${IMAGE_TAG}", path)

		assert.NoError(t, err)
		assert.Equal(t, "repo/app:v1.2.3", got)
	})

	t.Run("keeps string without envar literals", func(t *testing.T) {
		got, err := applyEnvarsToString("repo/app:latest", path)

		assert.NoError(t, err)
		assert.Equal(t, "repo/app:latest", got)
	})

	t.Run("returns error if envar is not defined", func(t *testing.T) {
		unsetenv(t, "UNDEFINED_IMAGE_TAG")

		got, err := applyEnvarsToString("repo/app:${UNDEFINED_IMAGE_TAG}", path)

		assert.Equal(t, "repo/app:${UNDEFINED_IMAGE_TAG}", got)
		assert.EqualError(t, err, "envar literal '${UNDEFINED_IMAGE_TAG}' found in task-definition.json but was not defined")
	})
}

func TestLookupEnvar(t *testing.T) {
	const path = "service.json"

	t.Run("returns defined envar", func(t *testing.T) {
		t.Setenv("SERVICE_NAME", "canary-service")

		got, err := lookupEnvar("${SERVICE_NAME}", path)

		assert.NoError(t, err)
		assert.Equal(t, "canary-service", got)
	})

	t.Run("returns empty string for defined empty envar", func(t *testing.T) {
		t.Setenv("EMPTY_VALUE", "")

		got, err := lookupEnvar("${EMPTY_VALUE}", path)

		assert.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("returns error if literal is invalid", func(t *testing.T) {
		got, err := lookupEnvar("SERVICE_NAME", path)

		assert.Empty(t, got)
		assert.EqualError(t, err, "invalid envar literal 'SERVICE_NAME' found in service.json")
	})

	t.Run("returns error if envar is not defined", func(t *testing.T) {
		unsetenv(t, "UNDEFINED_SERVICE_NAME")

		got, err := lookupEnvar("${UNDEFINED_SERVICE_NAME}", path)

		assert.Empty(t, got)
		assert.EqualError(t, err, "envar literal '${UNDEFINED_SERVICE_NAME}' found in service.json but was not defined")
	})
}

func unsetenv(t *testing.T, key string) {
	t.Helper()

	value, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err.Error())
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, value)
		}
	})
}

func TestLoadServiceDefinition(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d, err := LoadServiceDefinition("../fixtures")
		if err != nil {
			t.Fatal(err.Error())
		}
		assert.Equal(t, *d.ServiceName, "service")
	})
	t.Run("should error if service.json is not found", func(t *testing.T) {
		_, err := LoadServiceDefinition("./testdata")
		assert.EqualError(t, err, "no 'service.json' found in ./testdata")
	})
	t.Run("should error if service.json is invalid", func(t *testing.T) {
		_, err := LoadServiceDefinition("./testdata/invalid")
		assert.ErrorContains(t, err, "failed to read and unmarshal 'service.json':")
	})
}

func TestLoadTaskDefinition(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		d, err := LoadTaskDefinition("../fixtures")
		if err != nil {
			t.Fatal(err.Error())
		}
		assert.Equal(t, *d.Family, "test-task")
	})
	t.Run("should error if task-definition.json is not found", func(t *testing.T) {
		_, err := LoadTaskDefinition("./testdata")
		assert.EqualError(t, err, "no 'task-definition.json' found in ./testdata")
	})
	t.Run("should error if task-definition.json is invalid", func(t *testing.T) {
		_, err := LoadTaskDefinition("./testdata/invalid")
		assert.ErrorContains(t, err, "failed to read and unmarshal 'task-definition.json':")
	})
}

func TestReadAndUnmarshalJsonEscapesEnvars(t *testing.T) {
	t.Setenv("IMAGE_TAG", `latest","taskRoleArn":"arn:aws:iam::123456789012:role/pwn","image":"attacker`)
	path := filepath.Join(t.TempDir(), "task-definition.json")
	if err := os.WriteFile(path, []byte(`{"image":"repo:${IMAGE_TAG}","family":"app"}`), 0o644); err != nil {
		t.Fatal(err.Error())
	}

	var got struct {
		Image       string `json:"image"`
		Family      string `json:"family"`
		TaskRoleArn string `json:"taskRoleArn"`
	}
	if err := readAndUnmarshalJson(path, &got); err != nil {
		t.Fatal(err.Error())
	}

	assert.Equal(t, `repo:latest","taskRoleArn":"arn:aws:iam::123456789012:role/pwn","image":"attacker`, got.Image)
	assert.Equal(t, "app", got.Family)
	assert.Empty(t, got.TaskRoleArn)
}

func TestReadAndUnmarshalJsonRejectsEnvarsInObjectKeys(t *testing.T) {
	t.Setenv("KEY", "image")
	path := filepath.Join(t.TempDir(), "task-definition.json")
	if err := os.WriteFile(path, []byte(`{"${KEY}":"value"}`), 0o644); err != nil {
		t.Fatal(err.Error())
	}

	var got map[string]string
	err := readAndUnmarshalJson(path, &got)

	assert.ErrorContains(t, err, "envar literal found in JSON object key '${KEY}'")
}

func TestReadAndUnmarshalJsonRejectsEnvarsOutsideStringValues(t *testing.T) {
	t.Setenv("COUNT", "1")
	path := filepath.Join(t.TempDir(), "service.json")
	if err := os.WriteFile(path, []byte(`{"desiredCount":${COUNT}}`), 0o644); err != nil {
		t.Fatal(err.Error())
	}

	var got map[string]int
	err := readAndUnmarshalJson(path, &got)

	assert.Error(t, err)
}
