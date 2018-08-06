package main

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/aws"
	"encoding/base64"
	"encoding/json"
	"github.com/apex/log"
)

func UnmarshalTaskDefinition(data string) (*ecs.RegisterTaskDefinitionInput, error) {
	taskDefinitionJson, _ := base64.StdEncoding.DecodeString(data)
	log.Debugf("%s", taskDefinitionJson)
	taskDefinition := new(ecs.RegisterTaskDefinitionInput)
	if err := json.Unmarshal(([]byte)(taskDefinitionJson), taskDefinition); err != nil {
		return nil, err
	}
	log.Debugf("%s", *taskDefinition.TaskRoleArn)
	return taskDefinition, nil
}

func UnmarshalServiceDefinition(data string) (*ecs.CreateServiceInput, error) {
	serviceDefinitionJson, _ := base64.StdEncoding.DecodeString(data)
	o := new(CreateServiceInput)
	if err := json.Unmarshal([]byte(serviceDefinitionJson), o); err != nil {
		return nil, err
	}
	var loadBalancers []*ecs.LoadBalancer
	for _, v := range o.LoadBalancers {
		loadBalancers = append(loadBalancers, &ecs.LoadBalancer{
			ContainerName:    &v.ContainerName,
			ContainerPort:    &v.ContainerPort,
			LoadBalancerName: &v.LoadBalancerName,
			TargetGroupArn:   &v.TargetGroupArn,
		})
	}
	var placementConstraints []*ecs.PlacementConstraint
	for _, v := range o.PlacementConstraints {
		placementConstraints = append(placementConstraints, &ecs.PlacementConstraint{
			Expression: &v.Expression,
			Type:       &v.Type,
		})
	}
	var placementStrategy []*ecs.PlacementStrategy
	for _, v := range o.PlacementStrategy {
		placementStrategy = append(placementStrategy, &ecs.PlacementStrategy{
			Field: &v.Field,
			Type:  &v.Type,
		})
	}
	var serviceRegistries []*ecs.ServiceRegistry
	for _, v := range o.ServiceRegistries {
		serviceRegistries = append(serviceRegistries, &ecs.ServiceRegistry{
			ContainerName: &v.ContainerName,
			ContainerPort: &v.ContainerPort,
			Port: &v.Port,
			RegistryArn: &v.RegistryArn,
		})
	}
	ret := &ecs.CreateServiceInput{
		ClientToken: &o.ClientToken,
		Cluster:     &o.Cluster,
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: &(o.DeploymentConfiguration.MaximumPercent),
			MinimumHealthyPercent: &(o.DeploymentConfiguration.MinimumHealthyPercent),
		},
		DesiredCount:                  &o.DesiredCount,
		HealthCheckGracePeriodSeconds: &o.HealthCheckGracePeriodSeconds,
		LaunchType:                    &o.LaunchType,
		LoadBalancers:                 loadBalancers,
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: &(o.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp),
				SecurityGroups: aws.StringSlice(o.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups),
				Subnets:        aws.StringSlice(o.NetworkConfiguration.AwsvpcConfiguration.Subnets),
			},
		},
		PlacementConstraints: placementConstraints,
		PlacementStrategy:    placementStrategy,
		PlatformVersion:      &o.PlatformVersion,
		Role:                 &o.Role,
		SchedulingStrategy:   &o.SchedulingStrategy,
		ServiceName:          &o.ServiceName,
		ServiceRegistries:    serviceRegistries,
		TaskDefinition:       &o.TaskDefinition,
	}
	log.Debugf("%s::", *ret.TaskDefinition)
	return ret, nil
}

