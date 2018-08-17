package test_integration

import (
	"io/ioutil"
	"github.com/aws/aws-sdk-go/service/ecs"
	"encoding/json"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/aws"
	"testing"
	"github.com/loilo-inc/canarycage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/apex/log"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

const kCurrentServiceName = "itg-test-service-current"
const kNextServiceName = "itg-test-service-next"
const kHealthyTDArn = "cage-test-server-healthy:16"
const kUnhealthyTDArn = "cage-test-server-unhealthy:16"
const kUpButBuggyTDArn = "cage-test-server-up-but-buggy:16"
const kUpButSlowTDArn = "cage-test-server-up-but-slow:16"
const kUpAndExitTDArn = "cage-test-server-up-and-exit:16"
const kUpButExitTDArn = "cage-test-server-up-but-exit:16"

func SetupAws() (*ecs.ECS, *cloudwatch.CloudWatch) {
	ses, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})
	return ecs.New(ses), cloudwatch.New(ses)
}

func ensureCurrentService(awsEcs ecsiface.ECSAPI, envars *cage.Envars) (error) {
	d, err := ioutil.ReadFile("service-template.json")
	if err != nil {
		return err
	}
	log.Infof("checking if service %s exists", *envars.CurrentServiceName)
	if o, err := awsEcs.DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName},
	}); err != nil {
		return err
	} else if len(o.Services) == 0 {
		log.Infof("%s", o.Failures)
		log.Infof("service %s doesn't exist", *envars.CurrentServiceName)
		input := &ecs.CreateServiceInput{}
		if err := json.Unmarshal(d, input); err != nil {
			return err
		}
		input.ServiceName = aws.String(kCurrentServiceName)
		input.TaskDefinition = aws.String(kHealthyTDArn)
		log.Infof("creating service '%s'", *input.ServiceName)
		if _, err := awsEcs.CreateService(input); err != nil {
			return err
		}
	}
	log.Infof("waiting for service '%s' become STABLE", *envars.CurrentServiceName)
	if err := awsEcs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{envars.CurrentServiceName},
	}); err != nil {
		return err
	}
	log.Infof("service '%s' ensured. now %d tasks running", *envars.CurrentServiceName)
	return nil
}

func cleanupCurrentService(awsEcs ecsiface.ECSAPI, envars *cage.Envars) (error) {
	log.Infof("cleaning up service '%s'...", kCurrentServiceName)
	if _, err := awsEcs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: envars.Cluster,
		Service: aws.String(kCurrentServiceName),
	}); err != nil {
		return err
	}
	if err := awsEcs.WaitUntilServicesInactive(&ecs.DescribeServicesInput{
		Cluster:  envars.Cluster,
		Services: []*string{aws.String(kCurrentServiceName)},
	}); err != nil {
		return err
	}
	log.Infof("cleanup completed.")
	return nil
}

func setupEnvars(envars *cage.Envars) {
	if d, err := ioutil.ReadFile("cage.json"); err != nil {
		log.Fatalf(err.Error())
	} else if err := json.Unmarshal(d, envars); err != nil {
		log.Fatalf(err.Error())
	}
}

func TestHealthyToHealthy(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	envars := &cage.Envars{}
	setupEnvars(envars)
	envars.NextTaskDefinitionArn = aws.String(kHealthyTDArn)
	envars.CurrentServiceName = aws.String(kCurrentServiceName)
	envars.NextServiceName = aws.String(kNextServiceName)
	if err := cage.EnsureEnvars(envars); err != nil {
		t.Fatalf(err.Error())
	}
	ec, cw := SetupAws()
	if err := ensureCurrentService(ec, envars); err != nil {
		t.Fatalf(err.Error())
	}
	var err error
	err = envars.StartGradualRollOut(ec, cw)
	if err := cleanupCurrentService(ec, envars); err != nil {
		t.Fatalf(err.Error())
	}
	if err != nil {
		t.Fatalf(err.Error())
	}
}
