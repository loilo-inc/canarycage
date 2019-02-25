package commands

import (
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/loilo-inc/canarycage"
	"os"
	"path/filepath"
)

type Envars struct {
	_                 struct{} `type:"struct"`
	Region            string   `json:"region" type:"string"`
	Cluster           string   `json:"cluster" type:"string" required:"true"`
	Service           string   `json:"service" type:"string" required:"true"`
	CanaryInstanceArn *string  `json:"canaryInstanceArn" type:"string"`
	TaskDefinitionArn *string  `json:"nextTaskDefinitionArn" type:"string"`
	DeployFilesDir    *string  `json:"deployFilesDir" type:"string"`
}

const kDefaultRegion = "us-west-2"

// required
const ClusterKey = "CAGE_CLUSTER"
const ServiceKey = "CAGE_SERVICE"

// either required
const TaskDefinitionArnKey = "CAGE_TASK_DEFINITION_ARN"

// optional
const CanaryInstanceArnKey = "CAGE_CANARY_INSTANCE_ARN"
const RegionKey = "CAGE_REGION"

func EnsureEnvars(
	dest *Envars,
) error {
	// required
	if dest.Cluster == "" {
		return fmt.Errorf("--cluster [%s] is required", ClusterKey)
	} else if dest.Service == "" {
		return fmt.Errorf("--service [%s] is required", ServiceKey)
	}
	if dest.DeployFilesDir == nil && dest.TaskDefinitionArn == nil {
		return fmt.Errorf("--nextTaskDefinitionArn or deploy files dir (arg1) must be provided")
	}
	if dest.Region == "" {
		log.Warnf("--region was not set. use default region: %s", kDefaultRegion)
		dest.Region = kDefaultRegion
	}
	return nil
}

func LoadEnvarsFromFiles(dir string) (*Envars, error) {
	svcPath := filepath.Join(dir, "service.json")
	tdPath := filepath.Join(dir, "task-definition.json")
	_, noSvc := os.Stat(svcPath)
	_, noTd := os.Stat(tdPath)
	if noSvc != nil || noTd != nil {
		return nil, fmt.Errorf("roll out context specified at '%s' but no 'service.json' or 'task-definition.json'", dir)
	}
	dest := Envars{}
	if _, err := ReadAndUnmarshalJson(svcPath, &dest.serviceDefinition); err != nil {
		return nil, fmt.Errorf("failed to read and unmarshal service.json: %s", err)
	}
	if _, err := ReadAndUnmarshalJson(tdPath, &dest.taskDefinition); err != nil {
		return nil, fmt.Errorf("failed to read and unmarshal task-definition.json: %s", err)
	}
	dest.Cluster = *dest.serviceDefinition.Cluster
	dest.Service = *dest.serviceDefinition.ServiceName
	return &dest, nil
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
}

func ReadAndUnmarshalJson(path string, dest interface{}) ([]byte, error) {
	if d, err := cage.ReadFileAndApplyEnvars(path); err != nil {
		return d, err
	} else {
		if err := json.Unmarshal(d, dest); err != nil {
			return d, err
		}
		return d, nil
	}
}
