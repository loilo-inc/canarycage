package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Envars struct {
	_                         struct{} `type:"struct"`
	Region                    string   `json:"region" type:"string"`
	Cluster                   string   `json:"cluster" type:"string" required:"true"`
	Service                   string   `json:"service" type:"string" required:"true"`
	CanaryInstanceArn         string
	TaskDefinitionArn         string `json:"nextTaskDefinitionArn" type:"string"`
	TaskDefinitionInput       *ecs.RegisterTaskDefinitionInput
	ServiceDefinitionInput    *ecs.CreateServiceInput
	CanaryTaskIdleDuration    int // sec
	CanaryTaskRunningWait     int // sec
	CanaryTaskHealthCheckWait int // sec
	CanaryTaskStoppedWait     int // sec
	ServiceStableWait         int // sec
}

// required
const ClusterKey = "CAGE_CLUSTER"
const ServiceKey = "CAGE_SERVICE"

// either required
const TaskDefinitionArnKey = "CAGE_TASK_DEFINITION_ARN"

// optional
const CanaryInstanceArnKey = "CAGE_CANARY_INSTANCE_ARN"
const RegionKey = "CAGE_REGION"
const CanaryTaskIdleDuration = "CAGE_CANARY_TASK_IDLE_DURATION"
const UpdateServiceKey = "CAGE_UPDATE_SERVIEC"
const TaskRunningTimeout = "CAGE_TASK_RUNNING_TIMEOUT"
const TaskHealthCheckTimeout = "CAGE_TASK_HEALTH_CHECK_TIMEOUT"
const TaskStoppedTimeout = "CAGE_TASK_STOPPED_TIMEOUT"
const ServiceStableTimeout = "CAGE_SERVICE_STABLE_TIMEOUT"

var (
	envarLiteralRegexp = regexp.MustCompile(`\$\{([^}\r\n]+)\}`)
)

func EnsureEnvars(
	dest *Envars,
) error {
	// required
	if dest.Region == "" {
		return fmt.Errorf("--region [%s] is required", RegionKey)
	}
	if dest.Cluster == "" {
		return fmt.Errorf("--cluster [%s] is required", ClusterKey)
	}
	if dest.Service == "" {
		return fmt.Errorf("--service [%s] is required", ServiceKey)
	}
	if dest.TaskDefinitionArn == "" && dest.TaskDefinitionInput == nil {
		return fmt.Errorf("--nextTaskDefinitionArn or deploy context must be provided")
	}
	return nil
}

func LoadServiceDefinition(dir string) (*ecs.CreateServiceInput, error) {
	svcPath := filepath.Join(dir, "service.json")
	_, noSvc := os.Stat(svcPath)
	var service ecs.CreateServiceInput
	if noSvc != nil {
		return nil, fmt.Errorf("no 'service.json' found in %s", dir)
	}
	if err := readAndUnmarshalJson(svcPath, &service); err != nil {
		return nil, fmt.Errorf("failed to read and unmarshal 'service.json': %s", err)
	}
	return &service, nil
}

func LoadTaskDefinition(dir string) (*ecs.RegisterTaskDefinitionInput, error) {
	tdPath := filepath.Join(dir, "task-definition.json")
	_, noTd := os.Stat(tdPath)
	var td ecs.RegisterTaskDefinitionInput
	if noTd != nil {
		return nil, fmt.Errorf("no 'task-definition.json' found in %s", dir)
	}
	if err := readAndUnmarshalJson(tdPath, &td); err != nil {
		return nil, fmt.Errorf("failed to read and unmarshal 'task-definition.json': %s", err)
	}
	return &td, nil
}

func MergeEnvars(dest *Envars, src *Envars) {
	if src.Region != "" {
		dest.Region = src.Region
	}
	if src.Cluster != "" {
		dest.Cluster = src.Cluster
	}
	if src.Service != "" {
		dest.Service = src.Service
	}
	if src.CanaryInstanceArn != "" {
		dest.CanaryInstanceArn = src.CanaryInstanceArn
	}
	if src.TaskDefinitionArn != "" {
		dest.TaskDefinitionArn = src.TaskDefinitionArn
	}
	if src.TaskDefinitionInput != nil {
		dest.TaskDefinitionInput = src.TaskDefinitionInput
	}
	if src.ServiceDefinitionInput != nil {
		dest.ServiceDefinitionInput = src.ServiceDefinitionInput
	}
}

func readAndUnmarshalJson(path string, dest any) error {
	if b, err := os.ReadFile(path); err != nil {
		return err
	} else if d, err := applyEnvarsToJSON(b, path); err != nil {
		return err
	} else if err := json.Unmarshal(d, dest); err != nil {
		return err
	}
	return nil
}

func applyEnvarsToJSON(d []byte, path string) ([]byte, error) {
	var js any
	if err := json.Unmarshal(d, &js); err != nil {
		return nil, err
	}
	applied, err := applyEnvarsToJSONValue(js, path)
	if err != nil {
		return nil, err
	}
	return json.Marshal(applied)
}

func applyEnvarsToJSONValue(value any, path string) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			if envarLiteralRegexp.MatchString(key) {
				return nil, fmt.Errorf("envar literal found in JSON object key '%s' in %s; envars can only be used in JSON string values", key, path)
			}
			applied, err := applyEnvarsToJSONValue(child, path)
			if err != nil {
				return nil, err
			}
			v[key] = applied
		}
		return v, nil
	case []any:
		for i, child := range v {
			applied, err := applyEnvarsToJSONValue(child, path)
			if err != nil {
				return nil, err
			}
			v[i] = applied
		}
		return v, nil
	case string:
		return applyEnvarsToString(v, path)
	default:
		return v, nil
	}
}

func applyEnvarsToString(str string, path string) (string, error) {
	var replaceErr error
	replaced := envarLiteralRegexp.ReplaceAllStringFunc(str, func(literal string) string {
		if replaceErr != nil {
			return literal
		}
		envar, err := lookupEnvar(literal, path)
		if err != nil {
			replaceErr = err
			return literal
		}
		return envar
	})
	return replaced, replaceErr
}

func lookupEnvar(literal string, path string) (string, error) {
	match := envarLiteralRegexp.FindStringSubmatch(literal)
	if len(match) != 2 {
		return "", fmt.Errorf("invalid envar literal '%s' found in %s", literal, path)
	}
	if envar, ok := os.LookupEnv(match[1]); ok {
		return envar, nil
	}
	return "", fmt.Errorf("envar literal '%s' found in %s but was not defined", literal, path)
}
