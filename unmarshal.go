package main

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"encoding/base64"
	"encoding/json"
		)

func UnmarshalTaskDefinition(data string) (*ecs.RegisterTaskDefinitionInput, error) {
	taskDefinitionJson, _ := base64.StdEncoding.DecodeString(data)
	taskDefinition := new(ecs.RegisterTaskDefinitionInput)
	if err := json.Unmarshal(taskDefinitionJson, taskDefinition); err != nil {
		return nil, err
	}
	return taskDefinition, nil
}

func UnmarshalServiceDefinition(data string) (*ecs.CreateServiceInput, error) {
	serviceDefinitionJson, _ := base64.StdEncoding.DecodeString(data)
	serviceDefinition := new(ecs.CreateServiceInput)
	if err := json.Unmarshal(serviceDefinitionJson, serviceDefinition); err != nil {
		return nil, err
	}
	return serviceDefinition, nil
}
