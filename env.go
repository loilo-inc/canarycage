package cage

import (
	"encoding/json"
	"fmt"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/service/ecs"
	"os"
	"path/filepath"
)

type Envars struct {
	_                      struct{} `type:"struct"`
	Region                 string   `json:"region" type:"string"`
	Cluster                string   `json:"cluster" type:"string" required:"true"`
	Service                string   `json:"service" type:"string" required:"true"`
	CanaryInstanceArn      *string
	TaskDefinitionArn      *string `json:"nextTaskDefinitionArn" type:"string"`
	TaskDefinitionInput    *ecs.RegisterTaskDefinitionInput
	ServiceDefinitionInput *ecs.CreateServiceInput
}

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
		return NewErrorf("--cluster [%s] is required", ClusterKey)
	} else if dest.Service == "" {
		return NewErrorf("--service [%s] is required", ServiceKey)
	}
	if dest.TaskDefinitionArn == nil && dest.TaskDefinitionInput == nil {
		return NewErrorf("--nextTaskDefinitionArn or deploy context must be provided")
	}
	if dest.Region == "" {
		log.Fatalf("region must be specified. set --region flag or see also https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html")
	}
	return nil
}

func LoadDefinitionsFromFiles(dir string) (
	*ecs.RegisterTaskDefinitionInput,
	*ecs.CreateServiceInput,
	error,
) {
	svcPath := filepath.Join(dir, "service.json")
	tdPath := filepath.Join(dir, "task-definition.json")
	_, noSvc := os.Stat(svcPath)
	_, noTd := os.Stat(tdPath)
	var service ecs.CreateServiceInput
	var td ecs.RegisterTaskDefinitionInput
	if noSvc != nil || noTd != nil {
		return nil, nil, fmt.Errorf("roll out context specified at '%s' but no 'service.json' or 'task-definition.json'", dir)
	}
	if _, err := ReadAndUnmarshalJson(svcPath, &service); err != nil {
		return nil, nil, fmt.Errorf("failed to read and unmarshal service.json: %s", err)
	}
	if _, err := ReadAndUnmarshalJson(tdPath, &td); err != nil {
		return nil, nil, fmt.Errorf("failed to read and unmarshal task-definition.json: %s", err)
	}
	return &td, &service, nil
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
	if src.CanaryInstanceArn != nil {
		dest.CanaryInstanceArn = src.CanaryInstanceArn
	}
	if src.TaskDefinitionArn != nil {
		dest.TaskDefinitionArn = src.TaskDefinitionArn
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
