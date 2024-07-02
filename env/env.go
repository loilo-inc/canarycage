package env

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"golang.org/x/xerrors"
)

type Envars struct {
	_                         struct{} `type:"struct"`
	CI                        bool     `json:"ci" type:"bool"`
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
const TargetHealthCheckTimeout = "CAGE_TARGET_HEALTH_CHECK_TIMEOUT"

func EnsureEnvars(
	dest *Envars,
) error {
	// required
	if dest.Region == "" {
		return xerrors.Errorf("--region [%s] is required", RegionKey)
	}
	if dest.Cluster == "" {
		return xerrors.Errorf("--cluster [%s] is required", ClusterKey)
	} else if dest.Service == "" {
		return xerrors.Errorf("--service [%s] is required", ServiceKey)
	}
	if dest.TaskDefinitionArn == "" && dest.TaskDefinitionInput == nil {
		return xerrors.Errorf("--nextTaskDefinitionArn or deploy context must be provided")
	}
	return nil
}

func LoadServiceDefiniton(dir string) (*ecs.CreateServiceInput, error) {
	svcPath := filepath.Join(dir, "service.json")
	_, noSvc := os.Stat(svcPath)
	var service ecs.CreateServiceInput
	if noSvc != nil {
		return nil, xerrors.Errorf("roll out context specified at '%s' but no 'service.json' or 'task-definition.json'", dir)
	}
	if _, err := ReadAndUnmarshalJson(svcPath, &service); err != nil {
		return nil, xerrors.Errorf("failed to read and unmarshal service.json: %s", err)
	}
	return &service, nil
}

func LoadTaskDefiniton(dir string) (*ecs.RegisterTaskDefinitionInput, error) {
	tdPath := filepath.Join(dir, "task-definition.json")
	_, noTd := os.Stat(tdPath)
	var td ecs.RegisterTaskDefinitionInput
	if noTd != nil {
		return nil, xerrors.Errorf("roll out context specified at '%s' but no 'service.json' or 'task-definition.json'", dir)
	}
	if _, err := ReadAndUnmarshalJson(tdPath, &td); err != nil {
		return nil, xerrors.Errorf("failed to read and unmarshal task-definition.json: %s", err)
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

func ReadAndUnmarshalJson(path string, dest interface{}) ([]byte, error) {
	if d, err := ReadFileAndApplyEnvars(path); err != nil {
		return d, err
	} else {
		if err := json.Unmarshal(d, dest); err != nil {
			return d, err
		}
		return d, nil
	}
}

func ReadFileAndApplyEnvars(path string) ([]byte, error) {
	d, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	str := string(d)
	reg := regexp.MustCompile(`\${(.+?)}`)
	submatches := reg.FindAllStringSubmatch(str, -1)
	for _, m := range submatches {
		if envar, ok := os.LookupEnv(m[1]); ok {
			str = strings.Replace(str, m[0], envar, -1)
		} else {
			log.Fatalf("envar literal '%s' found in %s but was not defined", m[0], path)
		}
	}
	return []byte(str), nil
}
