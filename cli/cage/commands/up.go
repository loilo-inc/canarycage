package commands

import (
	"encoding/json"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/loilo-inc/canarycage"
	"github.com/urfave/cli"
	"path/filepath"
)

func UpCommand(ses *session.Session) cli.Command {
	context := "."
	return cli.Command{
		Name: "up",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:        "context, c",
				Usage:       "dirname that contains `service.json` and `task-definition.json`. default value is `.`",
				Value:       ".",
				Destination: &context,
			},
		},
		Action: func(ctx *cli.Context) {
			Up(ecs.New(ses), context)
		},
	}
}

func Up(
	ecscli ecsiface.ECSAPI,
	dir string,
) {
	serviceDefPath := filepath.Join(dir, "service.json")
	taskDefPath := filepath.Join(dir, "task-definition.json")
	var tdArn *string
	if td, err := cage.ReadFileAndApplyEnvars(taskDefPath); err != nil {
		log.Fatalf("failed to read %s: %s", serviceDefPath, err)
	} else {
		input := &ecs.RegisterTaskDefinitionInput{}
		if err := json.Unmarshal([]byte(td), input); err != nil {
			log.Fatalf("failed to unmarshal ecs.RegisterTaskDefinitionInput: %s", err)
		}
		log.Infof("registering task definition...")
		if o, err := ecscli.RegisterTaskDefinition(input); err != nil {
			log.Fatalf("failed to register task definition: %s", err)
		} else {
			log.Infof("registered: %s", *o.TaskDefinition.TaskDefinitionArn)
			tdArn = o.TaskDefinition.TaskDefinitionArn
		}
	}
	if svc, err := cage.ReadFileAndApplyEnvars(serviceDefPath); err != nil {
		log.Fatalf("failed to read %s: %s", serviceDefPath, err)
	} else {
		input := &ecs.CreateServiceInput{}
		if err := json.Unmarshal([]byte(svc), input); err != nil {
			log.Fatalf("failed to unmarshal ecs.CreateServiceInput: %s", err)
		}
		input.TaskDefinition = tdArn
		log.Infof("creating service '%s' with task-definition '%s'...", *input.ServiceName, *tdArn)
		if o, err := ecscli.CreateService(input); err != nil {
			log.Fatalf("failed to create service '%s': %s", *input.ServiceName, err.Error())
		} else {
			log.Infof("created: %s", o.Service.ServiceArn)
		}
		log.Infof("waiting for service '%s' to be STABLE")
		if err := ecscli.WaitUntilServicesStable(&ecs.DescribeServicesInput{
			Cluster:  input.Cluster,
			Services: []*string{input.ServiceName},
		}); err != nil {
			log.Fatalf(err.Error())
		} else {
			log.Infof("become: STABLE")
		}
	}
}
