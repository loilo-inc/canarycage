package cage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"os"
	"path/filepath"
)

type Envars struct {
	_                       struct{} `type:"struct"`
	Region                  *string  `json:"region" type:"string"`
	Cluster                 *string  `json:"cluster" type:"string" required:"true"`
	Service                 *string  `json:"service" type:"string" required:"true"`
	CanaryService           *string
	TaskDefinitionBase64    *string `json:"nextTaskDefinitionBase64" type:"string"`
	TaskDefinitionArn       *string `json:"nextTaskDefinitionArn" type:"string"`
	ServiceDefinitionBase64 *string
}

// required
const ClusterKey = "CAGE_CLUSTER"
const ServiceKey = "CAGE_SERVICE"

// either required
const ServiceDefinitionBase64Key = "CAGE_SERVICE_DEFINITION_BASE64"
const TaskDefinitionBase64Key = "CAGE_TASK_DEFINITION_BASE64"
const TaskDefinitionArnKey = "CAGE_TASK_DEFINITION_ARN"
const kDefaultRegion = "us-west-2"

// optional
const CanaryServiceKey = "CAGE_CANARY_SERVICE"
const RegionKey = "CAGE_REGION"

func isEmpty(o *string) bool {
	return o == nil || *o == ""
}

func EnsureEnvars(
	dest *Envars,
) (error) {
	// required
	if isEmpty(dest.Cluster) {
		return NewErrorf("--cluster [%s] is required", ClusterKey)
	} else if isEmpty(dest.Service) {
		return NewErrorf("--service [%s] is required", ServiceKey)
	}
	if isEmpty(dest.TaskDefinitionArn) && isEmpty(dest.TaskDefinitionBase64) {
		return NewErrorf("--nextTaskDefinitionArn or --nextTaskDefinitionBase64 must be provided")
	}
	if isEmpty(dest.Region) {
		dest.Region = aws.String(kDefaultRegion)
	}
	if isEmpty(dest.CanaryService) {
		dest.CanaryService = aws.String(fmt.Sprintf("%s-canary", *dest.Service))
	}
	return nil
}

func (e *Envars) LoadFromFiles(dir string) error {
	svcPath := filepath.Join(dir, "service.json")
	tdPath := filepath.Join(dir, "task-definition.json")
	_, noSvc := os.Stat(svcPath)
	_, noTd := os.Stat(tdPath)
	if noSvc != nil || noTd != nil {
		return NewErrorf("roll out context specified at '%s' but no 'service.json' or 'task-definition.json'", dir)
	}
	var (
		svc       = &ecs.CreateServiceInput{}
		td        = &ecs.RegisterTaskDefinitionInput{}
		svcBase64 string
		tdBase64  string
	)
	if d, err := ReadAndUnmarshalJson(svcPath, svc); err != nil {
		return NewErrorf("failed to read and unmarshal service.json: %s", err)
	} else {
		svcBase64 = base64.StdEncoding.EncodeToString(d)
	}
	if d, err := ReadAndUnmarshalJson(tdPath, td); err != nil {
		return NewErrorf("failed to read and unmarshal task-definition.json: %s", err)
	} else {
		tdBase64 = base64.StdEncoding.EncodeToString(d)
	}
	e.Cluster = svc.Cluster
	e.Service = svc.ServiceName
	e.ServiceDefinitionBase64 = &svcBase64
	e.TaskDefinitionBase64 = &tdBase64
	return nil
}

func (e *Envars) Merge(o *Envars) error {
	if !isEmpty(o.Region) {
		e.Region = o.Region
	}
	if !isEmpty(o.Cluster) {
		e.Cluster = o.Cluster
	}
	if !isEmpty(o.Service) {
		e.Service = o.Service
	}
	if !isEmpty(o.CanaryService) {
		e.CanaryService = o.CanaryService
	}
	if !isEmpty(o.TaskDefinitionBase64) {
		e.TaskDefinitionBase64 = o.TaskDefinitionBase64
	}
	if !isEmpty(o.TaskDefinitionArn) {
		e.TaskDefinitionArn = o.TaskDefinitionArn
	}
	if !isEmpty(o.ServiceDefinitionBase64) {
		e.ServiceDefinitionBase64 = o.ServiceDefinitionBase64
	}
	return nil
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
